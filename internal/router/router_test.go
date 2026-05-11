// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package router_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/router"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

func newTestConfig(mode string) *config.Config {
	return &config.Config{
		App: config.App{
			Env:     "test",
			Version: "v-test",
		},
		CORS: config.CORS{
			AllowedOrigins: []string{"http://localhost:3000"},
			MaxAge:         300,
		},
		JWT: config.JWT{
			Secret: "test-secret-key-32-chars-minimum",
		},
		AuthTransport: config.AuthTransport{
			Mode:              mode,
			AccessCookieName:  "access_token",
			RefreshCookieName: "refresh_token",
			CookiePath:        "/",
			CookieSameSite:    "lax",
		},
	}
}

func newTestRouter(mode string) http.Handler {
	auth := handler.NewAuthHandler(
		noopAuthService{},
		noopUserService{},
		config.AuthTransport{
			Mode:              mode,
			AccessCookieName:  "access_token",
			RefreshCookieName: "refresh_token",
			CookiePath:        "/",
			CookieSameSite:    "lax",
		},
		config.JWT{},
		slog.Default(),
		&sync.WaitGroup{},
	)
	user := handler.NewUserHandler(noopUserService{})

	return router.New(
		&router.Handlers{
			Auth: auth,
			User: user,
		},
		newTestConfig(mode),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		nil,
	)
}

// noopAuthService is a minimal AuthService implementation used for router wiring tests.
// Methods return zero values because route-registry tests do not assert business logic.
type noopAuthService struct{}

func (noopAuthService) Login(context.Context, string, string, string, string) (*service.LoginResult, error) {
	// Return a non-nil result to avoid nil dereference in AuthHandler.Login
	return &service.LoginResult{
		Pair: &service.TokenPair{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
		},
	}, nil
}

func (noopAuthService) Logout(context.Context, string) error { return nil }
func (noopAuthService) SendAccountVerification(context.Context, string) error {
	return nil
}

func (noopAuthService) VerifyAccount(context.Context, string, string) error { return nil }
func (noopAuthService) SendPasswordReset(context.Context, string) error     { return nil }
func (noopAuthService) ResetPassword(context.Context, string, string, string) error {
	return nil
}

func (noopAuthService) SendOTP(context.Context, string) error { return nil }
func (noopAuthService) VerifyOTP(context.Context, string, string) (*service.TokenPair, error) {
	return nil, nil
}

func (noopAuthService) RefreshToken(context.Context, string) (*service.TokenPair, error) {
	return nil, nil
}

// noopUserService is a minimal UserService implementation used for router wiring tests.
type noopUserService struct{}

func (noopUserService) List(context.Context, store.UserFilter) ([]model.User, int, error) {
	return nil, 0, nil
}

func (noopUserService) GetByID(context.Context, string) (*model.User, error) { return nil, nil }
func (noopUserService) Register(context.Context, *model.User, string) error  { return nil }
func (noopUserService) UpdateProfile(context.Context, *model.User) error     { return nil }
func (noopUserService) ChangePassword(context.Context, string, string, string) error {
	return nil
}

func (noopUserService) Delete(context.Context, string) error { return nil }

func (noopUserService) LoginHistory(context.Context, string, int) ([]model.LoginAuditLog, error) {
	return nil, nil
}

func TestRouter_RegistersExpectedRoutes(t *testing.T) {
	h := newTestRouter("bearer")
	routes, ok := h.(chi.Routes)

	require.True(t, ok, "router does not implement chi.Routes")

	// Keep this list in sync with internal/router/router.go.
	expected := map[string]bool{
		"GET /swagger/*":                     false,
		"GET /health":                        false,
		"GET /livez":                         false,
		"GET /readyz":                        false,
		"GET /metrics":                       false,
		"POST /auth/register":                false,
		"POST /auth/password/reset":          false,
		"POST /auth/password/new":            false,
		"POST /auth/login":                   false,
		"POST /auth/otp/verify":              false,
		"POST /auth/verify":                  false,
		"POST /auth/token/refresh":           false,
		"POST /auth/logout":                  false,
		"GET /api/v1/users/me":               false,
		"GET /api/v1/users/me/login-history": false,
		"GET /api/v1/users":                  false,
		"GET /api/v1/users/{id}":             false,
		"DELETE /api/v1/users/{id}":          false,
		"PUT /api/v1/users/{id}/profile":     false,
		"PUT /api/v1/users/{id}/password":    false,
	}

	err := chi.Walk(
		routes,
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			key := method + " " + route
			if _, exists := expected[key]; exists {
				expected[key] = true
			}
			return nil
		},
	)

	require.NoError(t, err, "chi.Walk")

	for route, seen := range expected {
		assert.Truef(t, seen, "missing route; %s", route)
	}
}

func TestRouter_ProtectedRoute_WithoutToken_Returns401(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, "GET /api/v1/users/me without token")
}

func TestRouter_LoginHistory_WithoutToken_Returns401(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/login-history", nil)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, "GET /api/v1/users/me/login-history without token")
}

func TestRouter_LoginHistory_WithUserToken_Returns200(t *testing.T) {
	r := newTestRouter("bearer")
	token := signTestJWT(t, testRouterUserID, "user")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/login-history", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "GET /api/v1/users/me/login-history with user token")
}

func TestRouter_WrongMethod_Returns405(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil) // login expects POST

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code, "GET /auth/login (expects POST)")
}

func TestRouter_Unknown_Returns404(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code, "GET /does-not-exist")
}

func TestRouter_CORS_Preflight_AllowedOrigin(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "OPTIONS /auth/login")

	got := w.Header().Get("Access-Control-Allow-Origin")
	require.Equal(t, "http://localhost:3000", got, "Access-Control-Allow-Origin")

	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	require.StringContains(t, allowMethods, http.MethodPost, "Access-Control-Allow-Methods should contain POST")
}

func TestRouter_CORS_Preflight_DisallowedOrigin(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	require.Equal(t, "", got, "Access-Control-Allow-Origin should be empty for disallowed origin")
}

func TestRouter_CORS_Preflight_CookieMode_AllowsCredentials(t *testing.T) {
	r := newTestRouter("cookie")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Credentials")
	require.Equal(t, "true", got, "Access-Control-Allow-Credentials in cookie mode")
}

func TestRouter_CORS_Preflight_BearerMode_DoesNotAllowCredentials(t *testing.T) {
	r := newTestRouter("bearer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Credentials")
	require.NotEqual(t, "true", got, "Access-Control-Allow-Credentials should not be set in bearer mode")
}

func TestRouter_CookieMode_RefreshWithoutCSRF_Returns403(t *testing.T) {
	r := newTestRouter("cookie")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-refresh-token"})
	req.Header.Set("Origin", "http://localhost:3000")

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, "POST /auth/token/refresh without CSRF")
}

func TestRouter_RateLimit_Login_Returns429(t *testing.T) {
	r := newTestRouter("bearer")

	var got429 bool

	// /auth/login is limited to 5 requests/min by IP in router.go
	for range 7 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"a@b.com","password":"Password1!"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "1.2.3.4:12345" // keep same IP for limiter

		r.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}

	require.True(t, got429, "expected at least one 429 Too Many Requests on /auth/login")
}

func signTestJWT(t *testing.T, sub, role string) string {
	t.Helper()

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(newTestConfig("bearer").JWT.Secret))

	require.NoError(t, err, "sign jwt")

	return tok
}

const (
	testRouterUserID  = "01966b0a-1234-7abc-def0-1234567890ab"
	testRouterAdminID = "01966b0a-9012-7abc-def0-1234567890ef"
)

func TestRouter_AdminRoutes_WithUserRole_Returns403(t *testing.T) {
	r := newTestRouter("bearer")
	token := signTestJWT(t, testRouterUserID, "user") // authenticated but not admin

	cases := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/users"},
		{method: http.MethodGet, path: "/api/v1/users/" + testRouterUserID},
		{method: http.MethodDelete, path: "/api/v1/users/" + testRouterUserID},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		r.ServeHTTP(w, req)

		require.Equalf(t, http.StatusForbidden, w.Code,
			"%s %s status = %d, want %d", tc.method, tc.path, w.Code, http.StatusForbidden)
	}
}

func TestRouter_AdminRoutes_WithAdminRole_Returns200Or204(t *testing.T) {
	r := newTestRouter("bearer")
	token := signTestJWT(t, testRouterAdminID, "admin")

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodGet, path: "/api/v1/users", want: http.StatusOK},
		{method: http.MethodDelete, path: "/api/v1/users/" + testRouterAdminID, want: http.StatusNoContent},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		r.ServeHTTP(w, req)

		require.Equalf(t, tc.want, w.Code, "%s %s status = %d, want %d", tc.method, tc.path, w.Code, tc.want)
	}
}
