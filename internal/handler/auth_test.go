// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/service"
)

func newAuthHandler(auth *mockAuthService, user *mockUserService) *handler.AuthHandler {
	return newAuthHandlerWithTransport(auth, user, config.AuthTransport{
		Mode:              "bearer",
		AccessCookieName:  "access_token",
		RefreshCookieName: "refresh_token",
		CookiePath:        "/",
		CookieSameSite:    "lax",
	}, config.JWT{}, slog.Default(), &sync.WaitGroup{})
}

func newAuthHandlerWithTransport(
	auth *mockAuthService,
	user *mockUserService,
	cfg config.AuthTransport,
	jwt config.JWT,
	_ *slog.Logger,
	_ *sync.WaitGroup,
) *handler.AuthHandler {
	return handler.NewAuthHandler(
		auth,
		user,
		config.AuthTransport{
			Mode:              cfg.Mode,
			AccessCookieName:  cfg.AccessCookieName,
			RefreshCookieName: cfg.RefreshCookieName,
			CookiePath:        cfg.CookiePath,
			CookieSameSite:    cfg.CookieSameSite,
			CookieSecure:      cfg.CookieSecure,
			CookieDomain:      cfg.CookieDomain,
		},
		jwt,
		slog.Default(),
		&sync.WaitGroup{},
	)
}

// ============================================================================
// Register
// ============================================================================

func TestAuthHandler_Register_EmailTaken(t *testing.T) {
	auth := &mockAuthService{}
	user := &mockUserService{registerErr: service.ErrEmailTaken}
	h := newAuthHandler(auth, user)
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/register", map[string]string{
		"email":      "john@example.com",
		"password":   "Password1",
		"first_name": "John",
		"last_name":  "Doe",
	})

	serve(h.Register, w, r)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/register", map[string]string{
		"email":      "john@example.com",
		"password":   "Password1",
		"first_name": "John",
		"last_name":  "Doe",
	})

	serve(h.Register, w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body map[string]string
	decodeJSON(t, w, &body)

	assert.NotEqual(t, "", body["id"], "Register() response should contain id")
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	// Empty body must return 400.
	r := httptest.NewRequest("POST", "/auth/register", strings.NewReader(""))

	serve(h.Register, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	// Missing email must return 400.
	r := newJSONRequest("POST", "/auth/register", map[string]string{
		"password":   "Password1",
		"first_name": "John",
		"last_name":  "Doe",
	})

	serve(h.Register, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Login
// ============================================================================

func TestAuthHandler_Login_Success(t *testing.T) {
	auth := &mockAuthService{
		loginResult: &service.LoginResult{
			Pair: &service.TokenPair{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			},
		},
	}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})

	serve(h.Login, w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	decodeJSON(t, w, &body)

	assert.NotEqual(t, "", body["access_token"], "Login() response should contain access_token")
}

func TestAuthHandler_Login_CookieMode_SetsCookiesAndDoesNotReturnTokenPair(t *testing.T) {
	auth := &mockAuthService{
		loginResult: &service.LoginResult{
			Pair: &service.TokenPair{
				AccessToken:  "access-token-cookie",
				RefreshToken: "refresh-token-cookie",
			},
		},
	}
	h := newAuthHandlerWithTransport(auth, &mockUserService{}, config.AuthTransport{
		Mode:              "cookie",
		AccessCookieName:  "access_token",
		RefreshCookieName: "refresh_token",
		CookiePath:        "/",
		CookieSameSite:    "lax",
	}, config.JWT{}, slog.Default(), &sync.WaitGroup{})

	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})
	serve(h.Login, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	require.True(t, len(cookies) >= 2, "expected at least 2 cookies")
	var foundCSRF bool
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.Value != "" {
			foundCSRF = true
			break
		}
	}
	require.True(t, foundCSRF, "expected csrf_token cookie to be set")

	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Nil(t, body["access_token"], "cookie mode must not return access_token in JSON body")
	assert.Equal(t, true, body["authenticated"])
}

func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	auth := &mockAuthService{loginErr: service.ErrInvalidCredentials}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})

	serve(h.Login, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_AccountLocked(t *testing.T) {
	auth := &mockAuthService{loginErr: service.ErrAccountLocked}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})

	serve(h.Login, w, r)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestAuthHandler_Login_AccountSuspended(t *testing.T) {
	auth := &mockAuthService{loginErr: service.ErrAccountSuspended}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})

	serve(h.Login, w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthHandler_Login_AccountNotVerified(t *testing.T) {
	auth := &mockAuthService{loginErr: service.ErrAccountNotVerified}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})

	serve(h.Login, w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthHandler_Login_OTPRequired(t *testing.T) {
	auth := &mockAuthService{
		loginResult: &service.LoginResult{
			OTPRequired: true,
			UserID:      "user-123",
		},
	}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/login", map[string]string{
		"email":    "john@example.com",
		"password": "Password1",
	})

	serve(h.Login, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Equal(t, true, body["otp_required"], "Login() response should contain otp_required: true")
	assert.Equal(t, "user-123", body["user_id"])
}

// ============================================================================
// Logout
// ============================================================================

func TestAuthHandler_Logout(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/auth/logout", nil), "user-123")

	serve(h.Logout, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthHandler_Logout_CookieMode_ClearsCSRFCookie(t *testing.T) {
	h := newAuthHandlerWithTransport(&mockAuthService{}, &mockUserService{}, config.AuthTransport{
		Mode:              "cookie",
		AccessCookieName:  "access_token",
		RefreshCookieName: "refresh_token",
		CookiePath:        "/",
		CookieSameSite:    "lax",
	}, config.JWT{}, slog.Default(), &sync.WaitGroup{})
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/auth/logout", nil), "user-123")

	serve(h.Logout, w, r)

	require.Equal(t, http.StatusNoContent, w.Code)

	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "csrf_token" && c.MaxAge < 0 {
			cleared = true
			break
		}
	}
	require.True(t, cleared, "expected csrf_token cookie to be cleared")
}

// ============================================================================
// VerifyAccount
// ============================================================================

func TestAuthHandler_VerifyAccount_Success(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/verify", map[string]string{
		"user_id": "user-123",
		// Token must be at least 64 characters long (validate:"min=64").
		"token": strings.Repeat("a", 64),
	})

	serve(h.VerifyAccount, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthHandler_VerifyAccount_MaxAttempts(t *testing.T) {
	auth := &mockAuthService{verifyAccErr: service.ErrTokenMaxAttempts}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/verify", map[string]string{
		"user_id": "user-123",
		"token":   strings.Repeat("c", 64),
	})

	serve(h.VerifyAccount, w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthHandler_VerifyAccount_InvalidToken(t *testing.T) {
	auth := &mockAuthService{verifyAccErr: service.ErrTokenInvalid}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/verify", map[string]string{
		"user_id": "user-123",
		"token":   strings.Repeat("b", 64),
	})

	serve(h.VerifyAccount, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// SendPasswordReset
// ============================================================================

func TestAuthHandler_SendPasswordReset(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/password/reset", map[string]string{
		"email": "john@example.com",
	})

	serve(h.SendPasswordReset, w, r)

	// Always 204: do not reveal whether the email exists.
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ============================================================================
// ResetPassword
// ============================================================================

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	h := newAuthHandler(&mockAuthService{}, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest(http.MethodPost, "/auth/password/new", map[string]string{
		"user_id":      "user-123",
		"token":        strings.Repeat("a", 64), // Token must satisfy min=64
		"new_password": "NewPassword1!",
	})

	serve(h.ResetPassword, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthHandler_ResetPassword_InvalidToken(t *testing.T) {
	auth := &mockAuthService{resetPassErr: service.ErrTokenInvalid}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest(http.MethodPost, "/auth/password/new", map[string]string{
		"user_id":      "user-123",
		"token":        strings.Repeat("b", 64),
		"new_password": "NewPassword1!",
	})

	serve(h.ResetPassword, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// VerifyOTP
// ============================================================================

func TestAuthHandler_VerifyOTP_Success(t *testing.T) {
	auth := &mockAuthService{
		verifyOTPPair: &service.TokenPair{
			AccessToken:  "otp-access-token",
			RefreshToken: "otp-refresh-token",
		},
	}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest(http.MethodPost, "/auth/otp/verify", map[string]string{
		"user_id": "user-123",
		"code":    "123456",
	})

	serve(h.VerifyOTP, w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	decodeJSON(t, w, &body)

	assert.NotEqual(t, "", body["access_token"], "VerifyOTP() response should contain access_token")
}

// ============================================================================
// RefreshToken
// ============================================================================

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	auth := &mockAuthService{
		refreshPair: &service.TokenPair{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
		},
	}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/token/refresh", map[string]string{
		"refresh_token": "some-refresh-token",
	})

	serve(h.RefreshToken, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RefreshToken_Invalid(t *testing.T) {
	auth := &mockAuthService{refreshErr: service.ErrInvalidRefreshToken}
	h := newAuthHandler(auth, &mockUserService{})
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/auth/token/refresh", map[string]string{
		"refresh_token": "bad-token",
	})

	serve(h.RefreshToken, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_RefreshToken_CookieFallback_UsesCookieValue(t *testing.T) {
	auth := &mockAuthService{
		refreshPair: &service.TokenPair{
			AccessToken:  "new-access-cookie",
			RefreshToken: "new-refresh-cookie",
		},
	}
	h := newAuthHandlerWithTransport(auth, &mockUserService{}, config.AuthTransport{
		Mode:              "cookie",
		AccessCookieName:  "access_token",
		RefreshCookieName: "refresh_token",
		CookiePath:        "/",
		CookieSameSite:    "lax",
	}, config.JWT{}, slog.Default(), &sync.WaitGroup{})

	r := httptest.NewRequest("POST", "/auth/token/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "cookie-refresh-token"})
	w := httptest.NewRecorder()

	serve(h.RefreshToken, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "cookie-refresh-token", auth.lastRefreshToken)
}
