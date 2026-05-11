// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// logCapture — slog handler that records log entries for assertions.
// ============================================================================

type logEntry struct {
	level   slog.Level
	message string
	attrs   map[string]string
}

type logCapture struct {
	entries []logEntry
}

func (c *logCapture) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (c *logCapture) Handle(_ context.Context, r slog.Record) error {
	entry := logEntry{
		level:   r.Level,
		message: r.Message,
		attrs:   make(map[string]string),
	}
	r.Attrs(func(a slog.Attr) bool {
		entry.attrs[a.Key] = a.Value.String()
		return true
	})
	c.entries = append(c.entries, entry)
	return nil
}

func (c *logCapture) WithAttrs(_ []slog.Attr) slog.Handler { return c }
func (c *logCapture) WithGroup(_ string) slog.Handler      { return c }

// hasLog returns true if any captured entry matches the given level, message, and key=value pairs.
func (c *logCapture) hasLog(level slog.Level, msg string, kvs ...string) bool {
	for _, e := range c.entries {
		if e.level != level || e.message != msg {
			continue
		}
		matched := true
		for i := 0; i+1 < len(kvs); i += 2 {
			if e.attrs[kvs[i]] != kvs[i+1] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// newAuthServiceWithCapture creates an AuthService wired to a log capture handler.
func newAuthServiceWithCapture(
	us *fakeUserStore,
	as *fakeAuthStore,
	rs *fakeRefreshTokenStore,
	m *fakeMailer,
) (service.AuthService, *logCapture) {
	logs := &logCapture{}
	logger := slog.New(logs)
	svc := service.NewAuthService(us, as, rs, &fakeLoginAuditStore{}, m, config.JWT{
		Secret:            "test-secret-key-for-unit-tests",
		ExpiryHours:       1,
		RefreshExpiryDays: 1,
	}, logger)
	return svc, logs
}

// ============================================================================
// Login audit logs
// ============================================================================

func TestAudit_Login_Success_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, err := svc.Login(context.Background(), u.Email, "secret", "1.2.3.4", "agent")
	assert.NoError(t, err)
	assert.True(t, logs.hasLog(slog.LevelInfo, "auth.login.success", "user_id", u.ID, "ip", "1.2.3.4"), "expected auth.login.success log entry")
}

func TestAudit_Login_WrongPassword_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "correct")

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.Login(context.Background(), u.Email, "wrong", "1.2.3.4", "agent")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.login.failed", "ip", "1.2.3.4", "reason", "invalid_credentials"), "expected auth.login.failed log entry")
}

func TestAudit_Login_AccountLocked_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	until := time.Now().Add(15 * time.Minute)
	u := &model.User{
		Email:        "locked@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusActive,
		IsVerified:   true,
		LockedUntil:  &until,
		Language:     "en",
		Timezone:     "UTC",
	}
	_ = us.Create(context.Background(), u)

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.Login(context.Background(), u.Email, "secret", "1.2.3.4", "agent")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.login.blocked", "reason", "account_locked"), "expected auth.login.blocked log entry for locked account")
}

func TestAudit_Login_AccountSuspended_IsLogged(t *testing.T) {
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

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.Login(context.Background(), u.Email, "secret", "", "agent")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.login.blocked", "reason", "account_suspended"), "expected auth.login.blocked log entry for suspended account")
}

func TestAudit_Login_NotVerified_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	u := &model.User{
		Email:        "unverified@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusActive,
		IsVerified:   false,
		Language:     "en",
		Timezone:     "UTC",
	}
	_ = us.Create(context.Background(), u)

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.Login(context.Background(), u.Email, "secret", "", "agent")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.login.blocked", "reason", "not_verified"), "expected auth.login.blocked log entry for unverified account")
}

func TestAudit_Login_AccountLockedAfterMaxAttempts_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	u := &model.User{
		Email:               "brute@example.com",
		PasswordHash:        string(hash),
		Status:              model.UserStatusActive,
		IsVerified:          true,
		FailedLoginAttempts: 4, // one more wrong attempt will trigger lock
		Language:            "en",
		Timezone:            "UTC",
	}
	_ = us.Create(context.Background(), u)

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.Login(context.Background(), u.Email, "wrong", "", "agent")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.account.locked", "user_id", u.ID), "expected auth.account.locked log entry after max attempts")
}

func TestAudit_Login_OTPRequired_IsLogged(t *testing.T) {
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

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.Login(context.Background(), u.Email, "secret", "1.2.3.4", "agent")
	assert.True(t, logs.hasLog(slog.LevelInfo, "auth.login.otp_required", "user_id", u.ID, "ip", "1.2.3.4"), "expected auth.login.otp_required log entry")
}

// ============================================================================
// Logout audit logs
// ============================================================================

func TestAudit_Logout_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	u := newVerifiedUser(t, us, "secret")

	svc, logs := newAuthServiceWithCapture(us, newFakeAuthStore(), newFakeRefreshTokenStore(), &fakeMailer{})
	assert.NoError(t, svc.Logout(context.Background(), u.ID))
	assert.True(t, logs.hasLog(slog.LevelInfo, "auth.logout", "user_id", u.ID), "expected auth.logout log entry")
}

// ============================================================================
// VerifyAccount audit logs
// ============================================================================

func TestAudit_VerifyAccount_Success_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")
	us.users[u.ID].IsVerified = false

	rawToken := "raw-verification-token"
	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken(rawToken),
		Type:      model.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_ = svc.VerifyAccount(context.Background(), u.ID, rawToken)
	assert.True(t, logs.hasLog(slog.LevelInfo, "auth.verify_account.success", "user_id", u.ID), "expected auth.verify_account.success log entry")
}

func TestAudit_VerifyAccount_Failed_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")
	us.users[u.ID].IsVerified = false

	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken("correct-token"),
		Type:      model.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_ = svc.VerifyAccount(context.Background(), u.ID, "wrong-token")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.verify_account.failed", "user_id", u.ID), "expected auth.verify_account.failed log entry")
}

// ============================================================================
// ResetPassword audit logs
// ============================================================================

func TestAudit_ResetPassword_Success_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "old")

	rawToken := "raw-reset-token"
	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken(rawToken),
		Type:      model.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_ = svc.ResetPassword(context.Background(), u.ID, rawToken, "NewPassword1!")
	assert.True(t, logs.hasLog(slog.LevelInfo, "auth.password_reset.success", "user_id", u.ID), "expected auth.password_reset.success log entry")
}

func TestAudit_ResetPassword_MaxAttempts_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "old")

	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken("token"),
		Type:      model.TokenTypePasswordReset,
		Attempts:  5,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_ = svc.ResetPassword(context.Background(), u.ID, "token", "NewPassword1!")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.password_reset.blocked", "user_id", u.ID, "reason", "max_attempts"), "expected auth.password_reset.blocked log entry")
}

// ============================================================================
// VerifyOTP audit logs
// ============================================================================

func TestAudit_VerifyOTP_Success_IsLogged(t *testing.T) {
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

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.VerifyOTP(context.Background(), u.ID, code)
	assert.True(t, logs.hasLog(slog.LevelInfo, "auth.otp.success", "user_id", u.ID), "expected auth.otp.success log entry")
}

func TestAudit_VerifyOTP_Failed_IsLogged(t *testing.T) {
	us := newFakeUserStore()
	as := newFakeAuthStore()
	u := newVerifiedUser(t, us, "secret")

	_ = as.Create(context.Background(), &model.VerificationToken{
		UserID:    u.ID,
		Token:     hashToken("123456"),
		Type:      model.TokenTypeOTP,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.VerifyOTP(context.Background(), u.ID, "000000")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.otp.failed", "user_id", u.ID), "expected auth.otp.failed log entry")
}

func TestAudit_VerifyOTP_MaxAttempts_IsLogged(t *testing.T) {
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

	svc, logs := newAuthServiceWithCapture(us, as, newFakeRefreshTokenStore(), &fakeMailer{})
	_, _ = svc.VerifyOTP(context.Background(), u.ID, "123456")
	assert.True(t, logs.hasLog(slog.LevelWarn, "auth.otp.blocked", "user_id", u.ID, "reason", "max_attempts"), "expected auth.otp.blocked log entry")
}
