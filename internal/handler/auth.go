// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/nanoninja/dojo/internal/config"
	_ "github.com/nanoninja/dojo/internal/fault" // swagger error response type
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

// AuthHandler handles HTTP requests for authentication endpoints.
type AuthHandler struct {
	auth   service.AuthService
	user   service.UserService
	cfg    config.AuthTransport
	jwt    config.JWT
	logger *slog.Logger
	wg     *sync.WaitGroup
}

// NewAuthHandler creates a new AuthHandler with the given auth and user services.
func NewAuthHandler(
	auth service.AuthService,
	user service.UserService,
	cfg config.AuthTransport,
	jwt config.JWT,
	logger *slog.Logger,
	wg *sync.WaitGroup,
) *AuthHandler {
	return &AuthHandler{
		auth:   auth,
		user:   user,
		cfg:    cfg,
		jwt:    jwt,
		logger: logger,
		wg:     wg,
	}
}

// ============================================================================
// Register
// ============================================================================

// RegisterRequest holds the fields required to create a new user account.
type RegisterRequest struct {
	Email     string `json:"email"      validate:"required,email,max=160"               example:"user@example.com"`
	Password  string `json:"password"   validate:"required,min=8,max=72,strongpassword" example:"Password1!"`
	FirstName string `json:"first_name" validate:"required,min=3,max=50,alpha"          example:"John"`
	LastName  string `json:"last_name"  validate:"required,min=3,max=50,alpha"          example:"Doe"`
	Confirm   string `json:"_confirm"   validate:"max=0"`
}

// RegisterResponse is the JSON body returned on successful registration.
type RegisterResponse struct {
	ID string `json:"id" example:"01966b0a-1234-7abc-def0-1234567890ab"`
}

// Register handles POST /auth/register
//
// @Summary  Register a new user
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body     RegisterRequest  true "Registration payload"
// @Success  201  {object} RegisterResponse
// @Failure  400  {object} fault.ErrorResponse "invalid request body"
// @Failure  409  {object} fault.ErrorResponse "email already in use"
// @Router   /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	u := &model.User{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Language:  "en",
		Timezone:  "UTC",
	}
	if err := h.user.Register(r.Context(), u, req.Password); err != nil {
		return toFault(err)
	}

	// Send verification email — registration succeeds even if email fails
	sendAsync(h.wg, h.logger, func(ctx context.Context) error {
		return h.auth.SendAccountVerification(ctx, u.ID)
	}, "email.verification.failed", "user_id", u.ID)

	return httputil.Created(w, RegisterResponse{ID: u.ID})
}

// ============================================================================
// Login
// ============================================================================

// LoginRequest holds the credentials required to authenticate.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email,max=160" example:"user@example.com"`
	Password string `json:"password" validate:"required,min=8,max=72"  example:"Password1!"`
}

// OTPRequiredResponse is returned when the user has 2FA enabled.
// The client must complete the OTP flow before receiving a token pair.
type OTPRequiredResponse struct {
	OTPRequired bool   `json:"otp_required" example:"true"`
	UserID      string `json:"user_id"      example:"01966b0a-1234-7abc-def0-1234567890ab"`
}

// AuthCookieModeResponse is returned when auth transport mode is cookie-only.
// In this mode, tokens are sent via HttpOnly cookies instead of JSON fields.
type AuthCookieModeResponse struct {
	Authenticated bool `json:"authenticated" example:"true"`
}

// Login handles POST /auth/login
//
// @Summary  Authenticate a user
// @Description In bearer/dual mode, returns a token pair in JSON. In cookie-only mode, sets HttpOnly auth cookies and returns {"authenticated":true}.
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body     LoginRequest        true "Login credentials"
// @Success  200  {object} service.TokenPair   "Token pair"
// @Success  200  {object} AuthCookieModeResponse "Cookie mode: tokens are set in HttpOnly cookies"
// @Success  200  {object} OTPRequiredResponse "2FA required — complete OTP flow"
// @Failure  401  {object} fault.ErrorResponse "invalid credentials"
// @Failure  403  {object} fault.ErrorResponse "account suspended or not verified"
// @Router   /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	ua := r.UserAgent()
	if len(ua) > 512 {
		ua = ua[:512]
	}
	result, err := h.auth.Login(r.Context(), req.Email, req.Password, r.RemoteAddr, ua)
	if err != nil {
		return toFault(err)
	}
	if result.OTPRequired {
		sendAsync(h.wg, h.logger, func(ctx context.Context) error {
			return h.auth.SendOTP(ctx, result.UserID)
		}, "email.otp.failed", "user_id", result.UserID)

		return httputil.OK(w, map[string]any{
			"otp_required": true,
			"user_id":      result.UserID,
		})
	}
	if h.shouldSetCookies() {
		// In cookie/dual mode we set both access + refresh cookies.
		h.setAuthCookies(w, result.Pair)
		if err := h.setCSRFCookie(w); err != nil {
			return toFault(err)
		}
	}
	if h.shouldReturnJSONPair() {
		return httputil.OK(w, result.Pair)
	}
	// Cookie-only mode intentionally avoids returning raw tokens in JSON.
	return httputil.OK(w, map[string]any{"authenticated": true})
}

// ============================================================================
// Logout
// ============================================================================

// Logout handles POST /auth/logout
//
// @Summary   Revoke all active refresh tokens for the current user
// @Description Revokes server-side refresh sessions. In cookie/dual mode, also clears auth cookies on the client.
// @Tags      auth
// @Produce   json
// @Security  BearerAuth
// @Param     X-CSRF-Token  header  string  false  "Required in cookie/dual mode"
// @Success   204
// @Failure   403  {object} fault.ErrorResponse "csrf validation failed"
// @Failure   401  {object} fault.ErrorResponse "missing or invalid token"
// @Router    /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if err := h.auth.Logout(r.Context(), userID); err != nil {
		return toFault(err)
	}
	if h.shouldSetCookies() {
		// Always clear client cookies in addition to server-side token revocation.
		h.clearAuthCookies(w)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// VerifyAccount
// ============================================================================

// VerifyAccountRequest holds the fields required to verify an account.
type VerifyAccountRequest struct {
	UserID string `json:"user_id" validate:"required"        example:"01966b0a-1234-7abc-def0-1234567890ab"`
	Token  string `json:"token"   validate:"required,min=64" example:"a1b2c3d4..."`
}

// VerifyAccount handles POST /auth/verify
//
// @Summary  Verify an email address using the token sent by email
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body VerifyAccountRequest true "Verification payload"
// @Success  204
// @Failure  400  {object} fault.ErrorResponse "invalid or expired token"
// @Router   /auth/verify [post]
func (h *AuthHandler) VerifyAccount(w http.ResponseWriter, r *http.Request) error {
	var req VerifyAccountRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	if err := h.auth.VerifyAccount(r.Context(), req.UserID, req.Token); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ResendRequest holds the user ID for email resend endpoints.
type ResendRequest struct {
	UserID string `json:"user_id" validate:"required,uuid" example:"01966b0a-1234-7abc-def0-1234567890ab"`
}

// ResendVerification handles POST /auth/verify/resend
//
// @Summary  Resend the account verification email
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body ResendRequest true "Resend payload"
// @Success  204
// @Failure  400  {object} fault.ErrorResponse "invalid request body"
// @Router   /auth/verify/resend [post]
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) error {
	var req ResendRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	sendAsync(h.wg, h.logger, func(ctx context.Context) error {
		return h.auth.SendAccountVerification(ctx, req.UserID)
	}, "email.verification.resend.failed", "user_id", req.UserID)

	httputil.NoContent(w)
	return nil
}

// ============================================================================
// OTP
// ============================================================================

// VerifyOTPRequest holds the fields required to validate a one-time password.
type VerifyOTPRequest struct {
	UserID string `json:"user_id" validate:"required"   example:"01966b0a-1234-7abc-def0-1234567890ab"`
	Code   string `json:"code"    validate:"required,len=6" example:"123456"`
}

// VerifyOTP handles POST /auth/otp/verify
//
// @Summary  Validate a 2FA OTP code and return a token pair
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body     VerifyOTPRequest true "OTP payload"
// @Success  200  {object} service.TokenPair
// @Failure  400  {object} fault.ErrorResponse "invalid or expired code"
// @Router   /auth/otp/verify [post]
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) error {
	var req VerifyOTPRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	pair, err := h.auth.VerifyOTP(r.Context(), req.UserID, req.Code)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, pair)
}

// ResendOTP handles POST /auth/otp/resend
//
// @Summary  Resend the OTP code by email
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body ResendRequest true "Resend payload"
// @Success  204
// @Failure  400  {object} fault.ErrorResponse "invalid request body"
// @Router   /auth/otp/resend [post]
func (h *AuthHandler) ResendOTP(w http.ResponseWriter, r *http.Request) error {
	var req ResendRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	sendAsync(h.wg, h.logger, func(ctx context.Context) error {
		return h.auth.SendOTP(ctx, req.UserID)
	}, "email.otp.resend.failed", "user_id", req.UserID)

	httputil.NoContent(w)
	return nil
}

// ============================================================================
// Password reset
// ============================================================================

// SendPasswordResetRequest holds the email address to send the reset link to.
type SendPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

// SendPasswordReset handles POST /auth/password/reset
//
// @Summary  Send a password reset email
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body SendPasswordResetRequest true "Email payload"
// @Success  204
// @Failure  400  {object} fault.ErrorResponse "invalid email"
// @Router   /auth/password/reset [post]
func (h *AuthHandler) SendPasswordReset(w http.ResponseWriter, r *http.Request) error {
	var req SendPasswordResetRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	// Always return 204 — do not reveal whether the email exists
	sendAsync(h.wg, h.logger, func(ctx context.Context) error {
		return h.auth.SendPasswordReset(ctx, req.Email)
	}, "email.password_reset.failed")
	httputil.NoContent(w)
	return nil
}

// ResetPasswordRequest holds the fields required to set a new password.
type ResetPasswordRequest struct {
	UserID      string `json:"user_id"      validate:"required"                          example:"01966b0a-1234-7abc-def0-1234567890ab"`
	Token       string `json:"token"        validate:"required,min=64"                   example:"a1b2c3d4..."`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72,strongpassword" example:"NewPassword1!"`
}

// ResetPassword handles POST /auth/password/new
//
// @Summary  Set a new password using a reset token
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body ResetPasswordRequest true "Reset payload"
// @Success  204
// @Failure  400  {object} fault.ErrorResponse "invalid or expired token"
// @Router   /auth/password/new [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var req ResetPasswordRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	if err := h.auth.ResetPassword(r.Context(), req.UserID, req.Token, req.NewPassword); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// Token refresh
// ============================================================================

// RefreshTokenRequest holds the refresh token used to obtain a new token pair.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"omitempty" example:"a1b2c3d4e5f6..."`
}

// RefreshCookieModeResponse is returned when refresh transport mode is cookie-only.
// In this mode, tokens are rotated through HttpOnly cookies.
type RefreshCookieModeResponse struct {
	Refreshed bool `json:"refreshed" example:"true"`
}

// RefreshToken handles POST /auth/token/refresh
//
// @Summary  Rotate a refresh token and return a new token pair
// @Description Accepts refresh token from JSON body or, in cookie/dual mode, from the refresh cookie. In cookie-only mode, rotated tokens are returned via HttpOnly cookies.
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body     RefreshTokenRequest true "Refresh token payload"
// @Param    X-CSRF-Token  header  string false "Required in cookie/dual mode"
// @Success  200  {object} service.TokenPair
// @Success  200  {object} RefreshCookieModeResponse "Cookie mode: tokens are set in HttpOnly cookies"
// @Failure  403  {object} fault.ErrorResponse "csrf validation failed"
// @Failure  401  {object} fault.ErrorResponse "invalid or expired refresh token"
// @Router   /auth/token/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) error {
	var req RefreshTokenRequest

	// JSON body is optional for cookie/dual mode (cookie fallback).
	// We only decode when the request actually contains a body.
	if r.ContentLength > 0 {
		if err := httputil.Bind(r, &req); err != nil {
			return err
		}
	}
	refreshToken := req.RefreshToken
	if refreshToken == "" && h.shouldSetCookies() {
		if c, err := r.Cookie(h.cfg.RefreshCookieName); err == nil {
			refreshToken = c.Value
		}
	}
	if refreshToken == "" {
		return toFault(service.ErrInvalidRefreshToken)
	}
	// Use the resolved token source (body first, then cookie fallback).
	pair, err := h.auth.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		return toFault(err)
	}
	if h.shouldSetCookies() {
		h.setAuthCookies(w, pair)
		if err := h.setCSRFCookie(w); err != nil {
			return toFault(err)
		}
	}
	if h.shouldReturnJSONPair() {
		return httputil.OK(w, pair)
	}
	return httputil.OK(w, map[string]any{"refreshed": true})
}

func (h *AuthHandler) shouldSetCookies() bool {
	return h.cfg.Mode == "cookie" || h.cfg.Mode == "dual"
}

func (h *AuthHandler) shouldReturnJSONPair() bool {
	return h.cfg.Mode == "bearer" || h.cfg.Mode == "dual"
}

func sameSiteFromString(v string) http.SameSite {
	switch v {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, pair *service.TokenPair) {
	ss := sameSiteFromString(h.cfg.CookieSameSite)

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.AccessCookieName,
		Value:    pair.AccessToken,
		Path:     h.cfg.CookiePath,
		Domain:   h.cfg.CookieDomain,
		MaxAge:   h.jwt.ExpiryHours * 3600,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: ss,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.RefreshCookieName,
		Value:    pair.RefreshToken,
		Path:     h.cfg.CookiePath,
		Domain:   h.cfg.CookieDomain,
		MaxAge:   h.jwt.RefreshExpiryDays * 86400,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: ss,
	})
}

func (h *AuthHandler) setCSRFCookie(w http.ResponseWriter) error {
	token, err := generateCSRFToken()
	if err != nil {
		return err
	}

	// CSRF cookie must be readable by frontend JS to be mirrored in X-CSRF-Token header.
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     h.cfg.CookiePath,
		Domain:   h.cfg.CookieDomain,
		HttpOnly: false,
		Secure:   h.cfg.CookieSecure,
		SameSite: sameSiteFromString(h.cfg.CookieSameSite),
	})

	return nil
}

func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{
		h.cfg.AccessCookieName,
		h.cfg.RefreshCookieName,
		"csrf_token",
	} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     h.cfg.CookiePath,
			Domain:   h.cfg.CookieDomain,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.cfg.CookieSecure,
			SameSite: sameSiteFromString(h.cfg.CookieSameSite),
		})
	}
}

func generateCSRFToken() (string, error) {
	// 32 random bytes is enough entropy for CSRF token usage.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
