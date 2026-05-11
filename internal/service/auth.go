// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxTokenAttempts  = 5
	maxLoginAttempts  = 5
	loginLockDuration = 15 * time.Minute
)

// TokenPair holds a JWT access token and a refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"a1b2c3d4e5f6..."`
}

// AuthService handles authentication flows: login, logout, token management, and account verification.
type AuthService interface {
	// Login authenticates a user by email and password.
	// Returns OTPRequired=true if 2FA is enabled, TokenPair otherwise.
	Login(ctx context.Context, email, password, ip, userAgent string) (*LoginResult, error)

	// Logout revokes all active refresh tokens for the user.
	Logout(ctx context.Context, userID string) error

	// SendAccountVerification generates a token and sends a verification email.
	SendAccountVerification(ctx context.Context, userID string) error

	// VerifyAccount validates the token and marks the account as verified.
	VerifyAccount(ctx context.Context, userID, token string) error

	// SendPasswordReset generates a reset token and sends it by email.
	SendPasswordReset(ctx context.Context, email string) error

	// ResetPassword validates the reset token and sets a new password.
	ResetPassword(ctx context.Context, userID, token, newPassword string) error

	// SendOTP generates a 6-digit code and sends it by email.
	SendOTP(ctx context.Context, userID string) error

	// VerifyOTP validates the OTP code and returns a JWT token pair.
	VerifyOTP(ctx context.Context, userID, code string) (*TokenPair, error)

	// RefreshToken validates a refresh token and returns a new token pair.
	RefreshToken(ctx context.Context, rawToken string) (*TokenPair, error)
}

// LoginResult is returned by Login. When OTPRequired is true, Pair is nil
// and the caller must complete the 2FA flow before receiving a token pair.
type LoginResult struct {
	Pair        *TokenPair
	OTPRequired bool
	UserID      string
}

type authService struct {
	users  store.UserStore
	auth   store.AuthStore
	tokens store.RefreshTokenStore
	audit  store.LoginAuditStore
	mailer AuthMailer
	jwt    config.JWT
	logger *slog.Logger
}

// NewAuthService creates an AuthService wired with the given stores, mailer, and JWT config.
func NewAuthService(
	users store.UserStore,
	auth store.AuthStore,
	tokens store.RefreshTokenStore,
	audit store.LoginAuditStore,
	mailer AuthMailer,
	jwt config.JWT,
	logger *slog.Logger,
) AuthService {
	return &authService{
		users:  users,
		auth:   auth,
		tokens: tokens,
		audit:  audit,
		mailer: mailer,
		jwt:    jwt,
		logger: logger,
	}
}

func (s *authService) Login(ctx context.Context, email, password, ip, userAgent string) (*LoginResult, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		s.writeAuditLog(ctx, nil, email, ip, userAgent, model.LoginStatusFailedNotFound)
		return nil, ErrInvalidCredentials
	}

	// Reject suspended, banned or deleted accounts
	switch u.Status {
	case model.UserStatusSuspended, model.UserStatusBanned:
		s.logger.Warn("auth.login.blocked", "user_id", u.ID, "reason", "account_suspended")
		s.writeAuditLog(ctx, &u.ID, email, ip, userAgent, model.LoginStatusFailedLocked)
		return nil, ErrAccountSuspended

	case model.UserStatusDeleted:
		s.writeAuditLog(ctx, nil, email, ip, userAgent, model.LoginStatusFailedNotFound)
		return nil, ErrInvalidCredentials
	}

	// Reject if the account is temporarily locked after too many failed attempts.
	if u.LockedUntil != nil && time.Now().Before(*u.LockedUntil) {
		s.logger.Warn("auth.login.blocked", "user_id", u.ID, "reason", "account_locked")
		s.writeAuditLog(ctx, &u.ID, email, ip, userAgent, model.LoginStatusFailedLocked)
		return nil, ErrAccountLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		// Track the failed attempts; lock the account on reaching the thresold.
		if incErr := s.users.IncrementFailedLogin(ctx, u.ID); incErr != nil {
			s.logger.Warn("failed to increment login attempts", "error", incErr, "user_id", u.ID)
		}
		if u.FailedLoginAttempts+1 >= maxLoginAttempts {
			until := time.Now().Add(loginLockDuration)
			if lockErr := s.users.LockAccount(ctx, u.ID, until); lockErr != nil {
				s.logger.Warn("failed to lock account", "error", lockErr, "user_id", u.ID)
			}
			s.logger.Warn("auth.account.locked", "user_id", u.ID, "locked_until", until)
		}
		s.logger.Warn("auth.login.failed", "ip", ip, "reason", "invalid_credentials")
		s.writeAuditLog(ctx, &u.ID, email, ip, userAgent, model.LoginStatusFailedPassword)
		return nil, ErrInvalidCredentials
	}

	if !u.IsVerified {
		s.logger.Warn("auth.login.blocked", "user_id", u.ID, "reason", "not_verified")
		s.writeAuditLog(ctx, &u.ID, email, ip, userAgent, model.LoginStatusFailedUnverified)
		return nil, ErrAccountNotVerified
	}

	// Clear any previous failed attempts on successful password check.
	if err := s.users.ResetFailedLogin(ctx, u.ID); err != nil {
		s.logger.Warn("failed to reset failed login attempts", "error", err, "user_id", u.ID)
	}

	if u.Is2FAEnabled {
		s.logger.Info("auth.login.otp_required", "user_id", u.ID, "ip", ip)
		return &LoginResult{
			OTPRequired: true,
			UserID:      u.ID,
		}, nil
	}

	pair, err := s.generateTokenPair(ctx, u.ID, u.Role)
	if err != nil {
		return nil, err
	}

	if err := s.users.UpdateLastLogin(ctx, u.ID, ip); err != nil {
		s.logger.Warn("failed to update last login", "error", err, "user_id", u.ID)
	}

	s.logger.Info("auth.login.success", "user_id", u.ID, "ip", ip, "role", u.Role.String())
	s.writeAuditLog(ctx, &u.ID, email, ip, userAgent, model.LoginStatusSuccess)
	return &LoginResult{Pair: pair}, nil
}

func (s *authService) Logout(ctx context.Context, userID string) error {
	if err := s.tokens.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	s.logger.Info("auth.logout", "user_id", userID)
	return nil
}

func (s *authService) SendAccountVerification(ctx context.Context, userID string) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}

	// Generate random token and store its hash
	rawToken, hash, err := generateToken()
	if err != nil {
		return err
	}

	if err := s.auth.DeleteExpired(ctx, userID); err != nil {
		s.logger.Warn("failed to delete expired verification tokens", "error", err, "user_id", userID)
	}
	t := &model.VerificationToken{
		UserID:    userID,
		Token:     hash,
		Type:      model.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := s.auth.Create(ctx, t); err != nil {
		return err
	}

	return s.mailer.SendAccountVerification(ctx, u.Email, rawToken)
}

func (s *authService) VerifyAccount(ctx context.Context, userID, token string) error {
	// Load the latest active token for this user and flow.
	// Token comparison is intentionally done in service code so we can increment
	// attempts on mismatch and enforce brute-force thresholds.
	t, err := s.auth.FindActiveByUserAndType(ctx, userID, model.TokenTypeEmailVerification)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTokenInvalid
	}

	// Stop processing if brute-force threshold is reached.
	if t.Attempts >= maxTokenAttempts {
		return ErrTokenMaxAttempts
	}

	// Always return the same public error for mismatches to avoid leaking token validity.
	if subtle.ConstantTimeCompare([]byte(hashToken(token)), []byte(t.Token)) != 1 {
		_ = s.auth.IncrementAttempts(ctx, t.ID)
		s.logger.Warn("auth.verify_account.failed", "user_id", userID)
		return ErrTokenInvalid
	}

	if err := s.auth.MarkUsed(ctx, t.ID); err != nil {
		return err
	}

	s.logger.Info("auth.verify_account.success", "user_id", userID)
	return s.users.UpdateVerified(ctx, userID)
}

func (s *authService) SendPasswordReset(ctx context.Context, email string) error {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
	}

	rawToken, hash, err := generateToken()
	if err != nil {
		return err
	}

	if err := s.auth.DeleteExpired(ctx, u.ID); err != nil {
		s.logger.Warn("failed to delete expired password reset tokens", "error", err, "user_id", u.ID)
	}
	t := &model.VerificationToken{
		UserID:    u.ID,
		Token:     hash,
		Type:      model.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := s.auth.Create(ctx, t); err != nil {
		return err
	}

	return s.mailer.SendPasswordReset(ctx, email, rawToken)
}

func (s *authService) ResetPassword(ctx context.Context, userID, token, newPassword string) error {
	// Load the latest active password-reset token and validate it locally
	// so failed attempts can be tracked.
	t, err := s.auth.FindActiveByUserAndType(ctx, userID, model.TokenTypePasswordReset)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTokenInvalid
	}

	// Reject verification after too many failed attempts.
	if t.Attempts >= maxTokenAttempts {
		s.logger.Warn("auth.password_reset.blocked", "user_id", userID, "reason", "max_attempts")
		return ErrTokenMaxAttempts
	}

	if hashToken(token) != t.Token {
		_ = s.auth.IncrementAttempts(ctx, t.ID)
		s.logger.Warn("auth.password_reset.failed", "user_id", userID)
		return ErrTokenInvalid
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, userID, string(hashed)); err != nil {
		return err
	}

	s.logger.Info("auth.password_reset.success", "user_id", userID)
	return s.auth.MarkUsed(ctx, t.ID)
}

func (s *authService) SendOTP(ctx context.Context, userID string) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}

	// Generate 6-digit code
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return fmt.Errorf("generating otp: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())
	hash := hashToken(code)

	if err := s.auth.DeleteExpired(ctx, userID); err != nil {
		s.logger.Warn("failed to delete expired OTP tokens", "error", err, "user_id", userID)
	}
	t := &model.VerificationToken{
		UserID:    userID,
		Token:     hash,
		Type:      model.TokenTypeOTP,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.auth.Create(ctx, t); err != nil {
		return err
	}

	return s.mailer.SendOTP(ctx, u.Email, code)
}

func (s *authService) VerifyOTP(ctx context.Context, userID, code string) (*TokenPair, error) {
	// Load the latest active OTP token and validate it locally
	// so failed attempts can be tracked.
	t, err := s.auth.FindActiveByUserAndType(ctx, userID, model.TokenTypeOTP)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTokenInvalid
	}

	// Block verification when too many failed attempts are already recorded.
	if t.Attempts >= maxTokenAttempts {
		s.logger.Warn("auth.otp.blocked", "user_id", userID, "reason", "max_attempts")
		return nil, ErrTokenMaxAttempts
	}

	// Compare hashed input code with stored hash.
	if hashToken(code) != t.Token {
		_ = s.auth.IncrementAttempts(ctx, t.ID)
		s.logger.Warn("auth.otp.failed", "user_id", userID)
		return nil, ErrTokenInvalid
	}

	if err := s.auth.MarkUsed(ctx, t.ID); err != nil {
		return nil, err
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	s.logger.Info("auth.otp.success", "user_id", userID)

	// Generate token pair now that OTP is validated
	return s.generateTokenPair(ctx, userID, u.Role)
}

func (s *authService) RefreshToken(ctx context.Context, rawToken string) (*TokenPair, error) {
	hash := hashToken(rawToken)
	rt, err := s.tokens.FindByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, ErrInvalidRefreshToken
	}

	u, err := s.users.FindByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	// Sign the JWT access token
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  u.ID,
		"role": u.Role.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(s.jwt.ExpiryHours) * time.Hour).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwt.Secret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Build the new refresh token (not yet stored)
	rawNew, newRT, err := s.buildRefreshToken(u.ID, now.Add(time.Duration(s.jwt.RefreshExpiryDays)*24*time.Hour))
	if err != nil {
		return nil, err
	}

	// Atomically revoke the old token and insert the new one
	if err := s.tokens.DeleteExpired(ctx, u.ID); err != nil {
		s.logger.Warn("failed to delete expired refresh tokens", "error", err, "user_id", u.ID)
	}
	if err := s.tokens.RotateToken(ctx, rt.ID, newRT); err != nil {
		return nil, fmt.Errorf("rotating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawNew,
	}, nil
}

// generateTokenPair creates a JWT access token and a refresh token for the given user.
func (s *authService) generateTokenPair(ctx context.Context, userID string, role model.Role) (*TokenPair, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(s.jwt.ExpiryHours) * time.Hour).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwt.Secret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	rawToken, rt, err := s.buildRefreshToken(userID, now.Add(time.Duration(s.jwt.RefreshExpiryDays)*24*time.Hour))
	if err != nil {
		return nil, err
	}

	if err := s.tokens.DeleteExpired(ctx, userID); err != nil {
		s.logger.Warn("failed to delete expired refresh tokens", "error", err, "user_id", userID)
	}
	if err := s.tokens.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawToken,
	}, nil
}

// buildRefreshToken generates a raw refresh token and its matching model.RefreshToken (not yet stored).
func (s *authService) buildRefreshToken(userID string, expiresAt time.Time) (rawToken string, rt *model.RefreshToken, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generating refresh token: %w", err)
	}
	rawToken = hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(rawToken))
	rt = &model.RefreshToken{
		UserID:    userID,
		TokenHash: hex.EncodeToString(hash[:]),
		ExpiresAt: expiresAt,
	}
	return rawToken, rt, nil
}

// writeAuditLog records a login attempt in a best-effort manner.
// Errors are logged but never propagated — audit failure must not block the auth flow.
func (s authService) writeAuditLog(ctx context.Context, userID *string, email, ip, userAgent string, status model.LoginStatus) {
	log := &model.LoginAuditLog{
		UserID:    userID,
		Email:     email,
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    status,
	}
	if err := s.audit.Create(ctx, log); err != nil {
		s.logger.Warn("audit.login.write_failed", "error", err, "email", email, "status", status)
	}
}

// generateToken creates a random token and returns the raw value and its SHA-256 hash.
func generateToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating token: %w", err)
	}
	raw = hex.EncodeToString(b)
	hash = hashToken(raw)
	return raw, hash, nil
}

// hashToken returns the SHA-256 hex hash of a token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
