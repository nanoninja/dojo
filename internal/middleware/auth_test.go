// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
)

const testSecret = "test-middleware-jwt-secret-key-32b"

// newAuthRequest creates a request with a signed JWT containing the given role.
func newAuthRequest(t *testing.T, role string) *http.Request {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))

	require.NoError(t, err, "signing JWT")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	return r
}

// runMiddleware runs the Authenticate + RequireRole chain and returns the response code.
func runMiddleware(t *testing.T, role string, minimum model.Role) int {
	t.Helper()
	w := httptest.NewRecorder()
	r := newAuthRequest(t, role)

	handler := middleware.Authenticate(testSecret)(
		middleware.RequireRole(minimum)(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		),
	)
	handler.ServeHTTP(w, r)
	return w.Code
}

// ============================================================================
// RequireRole
// ============================================================================

func TestRequireRole_AllowsExactRole(t *testing.T) {
	code := runMiddleware(t, "admin", model.RoleAdmin)

	assert.Equal(t, http.StatusOK, code, "exact role")
}

func TestRequireRole_AllowsHigherRole(t *testing.T) {
	// superadmin satisfies a requirement of admin.
	code := runMiddleware(t, "superadmin", model.RoleAdmin)

	assert.Equal(t, http.StatusOK, code, "higher role")
}

func TestRequireRole_BlocksLowerRole(t *testing.T) {
	// user cannot access an admin route.
	code := runMiddleware(t, "user", model.RoleAdmin)

	assert.Equal(t, http.StatusForbidden, code, "lower role")
}

func TestRequireRole_Hierarchy(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		minimum model.Role
		want    int
	}{
		{"user meets user", "user", model.RoleUser, http.StatusOK},
		{"moderator meets user", "moderator", model.RoleUser, http.StatusOK},
		{"moderator meets moderator", "moderator", model.RoleModerator, http.StatusOK},
		{"user blocked from moderator", "user", model.RoleModerator, http.StatusForbidden},
		{"manager meets admin? no", "manager", model.RoleAdmin, http.StatusForbidden},
		{"admin meets admin", "admin", model.RoleAdmin, http.StatusOK},
		{"system meets all", "system", model.RoleSuperAdmin, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runMiddleware(t, tt.role, tt.minimum)

			assert.Equalf(t, tt.want, code, "role=%q minimum=%v", tt.role, tt.minimum)
		})
	}
}

// ============================================================================
// Authenticate
// ============================================================================

func TestAuthenticate_MissingHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	middleware.Authenticate(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "missing header")
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer invalid.token.here")

	middleware.Authenticate(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "invalid token")
}

func TestAuthenticate_InjectsRoleIntoContext(t *testing.T) {
	r := newAuthRequest(t, "admin")
	var gotRole model.Role

	middleware.Authenticate(testSecret)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		gotRole = middleware.RoleFromContext(req.Context())
	})).ServeHTTP(httptest.NewRecorder(), r)

	assert.Equal(t, model.RoleAdmin, gotRole, "context role")
}

func TestAuthenticateWithTransport_CookieMode_UsesCookie(t *testing.T) {
	tok := signJWT(t, "user-123", "admin")
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "access_token", Value: tok})

	w := httptest.NewRecorder()
	middleware.AuthenticateWithTransport(testSecret, "cookie", "access_token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "cookie mode")
}

func TestAuthenticateWithTransport_DualMode_FallsBackToCookie(t *testing.T) {
	tok := signJWT(t, "user-123", "user")
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "access_token", Value: tok})

	w := httptest.NewRecorder()
	middleware.AuthenticateWithTransport(testSecret, "dual", "access_token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "dual mode fallback to cookie")
}

func TestAuthenticateWithTransport_BearerMode_IgnoresCookie(t *testing.T) {
	tok := signJWT(t, "user-123", "user")
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "access_token", Value: tok})

	w := httptest.NewRecorder()
	middleware.AuthenticateWithTransport(testSecret, "bearer", "access_token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code, "bearer mode ignores cookie")
}

func signJWT(t *testing.T, sub, role string) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))

	require.NoError(t, err, "signing JWT")

	return tok
}
