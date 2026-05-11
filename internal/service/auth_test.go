// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// testJWT is the JWT configuration used across all tests in this file.
var testJWT = config.JWT{
	Secret:            "test-secret-key-for-unit-tests",
	ExpiryHours:       1,
	RefreshExpiryDays: 1,
}

// hashToken mirrors the SHA-256 hashing used internally by authService.
func hashToken(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// newVerifiedUser creates a verified user with a hashed password
// and inserts it into fakeUserStore.
func newVerifiedUser(t *testing.T, us *fakeUserStore, password string) *model.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	assert.NoError(t, err, "setup: bcrypt")
	u := &model.User{
		ID:           "", // populated by Create
		Email:        "john@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusActive,
		IsVerified:   true,
		Language:     "en",
		Timezone:     "UTC",
	}
	assert.NoError(t, us.Create(context.Background(), u), "setup: Create()")
	return u
}

func newAuthService(
	us *fakeUserStore,
	as *fakeAuthStore,
	rs *fakeRefreshTokenStore,
	m *fakeMailer,
) service.AuthService {
	return service.NewAuthService(
		us,
		as,
		rs,
		&fakeLoginAuditStore{},
		m,
		testJWT,
		slog.Default(),
	)
}

// ============================================================================
// Login
// ============================================================================

func TestAuthService_Login_Success(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	result, err := svc.Login(context.Background(), u.Email, "secret", "127.0.0.1", "agent")
	assert.NoError(t, err)
	assert.False(t, result.OTPRequired, "OTPRequired should be false")
	assert.NotNil(t, result.Pair, "Pair should not be nil")
	assert.NotEqual(t, "", result.Pair.AccessToken, "AccessToken should not be empty")
	assert.NotEqual(t, "", result.Pair.RefreshToken, "RefreshToken should not be empty")
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	svc := newAuthService(newFakeUserStore(), newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.Login(context.Background(), "nobody@example.com", "secret", "", "agent")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestAuthService_Login_AccountLocked(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	// Lock the account by setting LockedUntil to a future time.
	until := time.Now().Add(15 * time.Minute)
	us.users[u.ID].LockedUntil = &until

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.Login(context.Background(), u.Email, "secret", "", "agent")
	assert.ErrorIs(t, err, service.ErrAccountLocked)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "correct")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.Login(context.Background(), u.Email, "wrong", "", "agent")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestAuthService_Login_AccountNotVerified(t *testing.T) {
	us := newFakeUserStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	u := &model.User{
		Email:        "unverified@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusPending,
		IsVerified:   false,
		Language:     "en",
		Timezone:     "UTC",
	}
	_ = us.Create(context.Background(), u)

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.Login(context.Background(), u.Email, "secret", "", "agent")
	assert.ErrorIs(t, err, service.ErrAccountNotVerified)
}

func TestAuthService_Login_AccountSuspended(t *testing.T) {
	us := newFakeUserStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	u := &model.User{
		Email:        "suspended@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusSuspended,
		IsVerified:   true,
		Language:     "en",
		Timezone:     "UTC",
	}
	_ = us.Create(context.Background(), u)

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.Login(context.Background(), u.Email, "secret", "", "agent")
	assert.ErrorIs(t, err, service.ErrAccountSuspended)
}

func TestAuthService_Login_OTPRequired(t *testing.T) {
	us := newFakeUserStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	u := &model.User{
		Email:        "otp@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusActive,
		IsVerified:   true,
		Is2FAEnabled: true,
		Language:     "en",
		Timezone:     "UTC",
	}
	_ = us.Create(context.Background(), u)

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	result, err := svc.Login(context.Background(), u.Email, "secret", "", "agent")
	assert.NoError(t, err)
	assert.True(t, result.OTPRequired, "OTPRequired should be true for 2FA user")
	assert.Nil(t, result.Pair, "Pair should be nil when OTP is required")
}

// ============================================================================
// Logout
// ============================================================================

func TestAuthService_Logout(t *testing.T) {
	us := newFakeUserStore()
	rs := newFakeRefreshTokenStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), rs, &fakeMailer{})
	ctx := context.Background()

	// Get a token pair via Login to obtain an active refresh token.
	result, err := svc.Login(ctx, u.Email, "secret", "", "agent")
	assert.NoError(t, err, "setup: Login()")
	assert.NoError(t, svc.Logout(ctx, u.ID))

	// The refresh token must become invalid after Logout.
	hash := hashToken(result.Pair.RefreshToken)
	found, err := rs.FindByHash(ctx, hash)
	assert.NoError(t, err)
	assert.Nil(t, found, "refresh token should be revoked after Logout")
}

// ============================================================================
// VerifyAccount
// ============================================================================

func TestAuthService_VerifyAccount_Valid(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")
	// Simulate a user that is not yet verified.
	us.users[u.ID].IsVerified = false

	rawToken := "raw-verification-token"
	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken(rawToken),
		Type:      model.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc := newAuthService(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	assert.NoError(t, svc.VerifyAccount(context.Background(), u.ID, rawToken))
	assert.True(t, us.users[u.ID].IsVerified, "user should be marked as verified")
}

func TestAuthService_VerifyAccount_InvalidToken(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	err := svc.VerifyAccount(context.Background(), u.ID, "bad-token")
	assert.ErrorIs(t, err, service.ErrTokenInvalid)
}

func TestAuthService_VerifyAccount_MaxAttempts(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")
	us.users[u.ID].IsVerified = false

	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken("raw-verification-token"),
		Type:      model.TokenTypeEmailVerification,
		Attempts:  5,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc := newAuthService(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	err := svc.VerifyAccount(context.Background(), u.ID, "raw-verification-token")
	assert.ErrorIs(t, err, service.ErrTokenMaxAttempts)
}

// ============================================================================
// RefreshToken
// ============================================================================

func TestAuthService_RefreshToken_Valid(t *testing.T) {
	us := newFakeUserStore()
	rs := newFakeRefreshTokenStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), rs, &fakeMailer{})
	ctx := context.Background()

	first, err := svc.Login(ctx, u.Email, "secret", "", "agent")
	assert.NoError(t, err, "setup: Login()")

	second, err := svc.RefreshToken(ctx, first.Pair.RefreshToken)
	assert.NoError(t, err)
	assert.NotEqual(t, "", second.AccessToken, "AccessToken should not be empty")
	assert.NotEqual(t, "", second.RefreshToken, "RefreshToken should not be empty")
	// The previous token must be revoked after rotation.
	assert.NotEqual(t, first.Pair.RefreshToken, second.RefreshToken, "should rotate the token, not reuse it")
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	svc := newAuthService(newFakeUserStore(), newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.RefreshToken(context.Background(), "invalid-token")
	assert.ErrorIs(t, err, service.ErrInvalidRefreshToken)
}

func TestAuthService_RefreshToken_OldTokenRevoked(t *testing.T) {
	us := newFakeUserStore()
	rs := newFakeRefreshTokenStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), rs, &fakeMailer{})
	ctx := context.Background()

	first, err := svc.Login(ctx, u.Email, "secret", "", "agent")
	assert.NoError(t, err, "setup: Login()")

	_, err = svc.RefreshToken(ctx, first.Pair.RefreshToken)
	assert.NoError(t, err)

	// The old token must no longer be accepted after rotation.
	_, err = svc.RefreshToken(ctx, first.Pair.RefreshToken)
	assert.ErrorIs(t, err, service.ErrInvalidRefreshToken, "old token should be rejected after rotation")
}

func TestAuthService_RefreshToken_AtomicRotation(t *testing.T) {
	// If RotateToken fails, the old token must remain valid so the user is not locked out.
	us := newFakeUserStore()
	rs := newFakeRefreshTokenStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), rs, &fakeMailer{})
	ctx := context.Background()

	first, err := svc.Login(ctx, u.Email, "secret", "", "agent")
	assert.NoError(t, err, "setup: Login()")

	// Simulate a failure during rotation.
	rs.failRotate = true
	_, err = svc.RefreshToken(ctx, first.Pair.RefreshToken)
	assert.Error(t, err, "expected an error when RotateToken fails")
	rs.failRotate = false

	// The old token must still work after the failed rotation.
	second, err := svc.RefreshToken(ctx, first.Pair.RefreshToken)
	assert.NoError(t, err, "old token should still be valid after failed rotation")
	assert.NotEqual(t, "", second.AccessToken, "expected a valid token pair after recovery")
	assert.NotEqual(t, "", second.RefreshToken, "expected a valid token pair after recovery")
}

// ============================================================================
// SendAccountVerification
// ============================================================================

func TestAuthService_SendAccountVerification_UserNotFound(t *testing.T) {
	svc := newAuthService(newFakeUserStore(), newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	err := svc.SendAccountVerification(context.Background(), "non-existent-id")
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestAuthService_SendAccountVerification(t *testing.T) {
	us := newFakeUserStore()
	mailer := &fakeMailer{}
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), mailer)
	assert.NoError(t, svc.SendAccountVerification(context.Background(), u.ID))
	assert.Len(t, mailer.sentVerification, 1, "mailer should have been called once")
	assert.Equal(t, u.Email, mailer.sentVerification[0])
}

// ============================================================================
// SendPasswordReset
// ============================================================================

func TestAuthService_SendPasswordReset(t *testing.T) {
	us := newFakeUserStore()
	mailer := &fakeMailer{}
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), mailer)
	// SendPasswordReset must not return an error when the email is unknown
	// to avoid revealing which accounts exist.
	assert.NoError(t, svc.SendPasswordReset(context.Background(), u.Email))
	assert.Len(t, mailer.sentReset, 1, "should have sent an email")

	assert.NoError(t, svc.SendPasswordReset(context.Background(), "unknown@example.com"), "unknown email should not return error")
}

// ============================================================================
// ResetPassword
// ============================================================================

func TestAuthService_ResetPassword(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "old-password")

	rawToken := "raw-reset-token"
	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken(rawToken),
		Type:      model.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc := newAuthService(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	assert.NoError(t, svc.ResetPassword(context.Background(), u.ID, rawToken, "new-password"))

	// The new password must authenticate successfully.
	stored := us.users[u.ID].PasswordHash
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored), []byte("new-password")), "new password not saved correctly")
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	err := svc.ResetPassword(context.Background(), u.ID, "bad-token", "new-password")
	assert.ErrorIs(t, err, service.ErrTokenInvalid)
}

func TestAuthService_ResetPassword_MaxAttempts(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "old-password")

	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken("raw-reset-token"),
		Type:      model.TokenTypePasswordReset,
		Attempts:  5,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc := newAuthService(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	err := svc.ResetPassword(context.Background(), u.ID, "raw-reset-token", "new-password")
	assert.ErrorIs(t, err, service.ErrTokenMaxAttempts)
}

// ============================================================================
// SendOTP / VerifyOTP
// ============================================================================

func TestAuthService_SendOTP(t *testing.T) {
	us := newFakeUserStore()
	mailer := &fakeMailer{}
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), mailer)
	assert.NoError(t, svc.SendOTP(context.Background(), u.ID))
	assert.Len(t, mailer.sentOTP, 1, "should have sent an OTP email")
}

func TestAuthService_VerifyOTP_InvalidCode(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	svc := newAuthService(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.VerifyOTP(context.Background(), u.ID, "000000")
	assert.ErrorIs(t, err, service.ErrTokenInvalid)
}

func TestAuthService_VerifyOTP_MaxAttempts(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")

	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken("123456"),
		Type:      model.TokenTypeOTP,
		Attempts:  5,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	svc := newAuthService(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.VerifyOTP(context.Background(), u.ID, "123456")
	assert.ErrorIs(t, err, service.ErrTokenMaxAttempts)
}

func TestAuthService_VerifyOTP_Valid(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")

	code := "123456"
	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken(code),
		Type:      model.TokenTypeOTP,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	svc := newAuthService(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	pair, err := svc.VerifyOTP(context.Background(), u.ID, code)
	assert.NoError(t, err)
	assert.NotNil(t, pair, "should return a token pair")
	assert.NotEqual(t, "", pair.AccessToken, "AccessToken should not be empty")
}
