// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/model"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const (
	contextKeyUserID contextKey = "userID"
	contextKeyRole   contextKey = "role"
)

// Authenticate validates the JWT token and injects the userID into the request context.
func Authenticate(secret string) func(http.Handler) http.Handler {
	// Backward-compatible default: bearer header only.
	return AuthenticateWithTransport(secret, "bearer", "access_token")
}

// AuthenticateWithTransport returns a middleware that validates JWT tokens using
// the configured transport mode (bearer, cookie, or dual).
func AuthenticateWithTransport(secret, mode, accessCookieName string) func(http.Handler) http.Handler {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if accessCookieName == "" {
		accessCookieName = "access_token"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r, mode, accessCookieName)
			if tokenStr == "" {
				_ = httputil.Error(w, fault.Unauthorized(nil))
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				_ = httputil.Error(w, fault.Unauthorized(nil))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				_ = httputil.Error(w, fault.Unauthorized(nil))
				return
			}
			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				_ = httputil.Error(w, fault.Unauthorized(nil))
				return
			}
			roleStr, ok := claims["role"].(string)
			if !ok || roleStr == "" {
				_ = httputil.Error(w, fault.Unauthorized(nil))
				return
			}
			role := model.ParseRole(roleStr)

			ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
			ctx = context.WithValue(ctx, contextKeyRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request, mode, accessCookieName string) string {
	// 1) Prefer Authorization header when present
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		if tok := strings.TrimSpace(after); tok != "" {
			return tok
		}
	}

	// 2) Cookie fallback only for cookie/dual modes.
	if mode == "cookie" || mode == "dual" {
		c, err := r.Cookie(accessCookieName)
		if err == nil {
			return strings.TrimSpace(c.Value)
		}
	}

	return ""
}

// RequireRole rejects requests where the authenticated user's role
// is lower than the required minimum. Must be used after Authenticate.
func RequireRole(minimum model.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if role < minimum {
				_ = httputil.Error(w, fault.Forbidden(nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext retrieves the authenticated userID from the context.
// Returns empty string if not found.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKeyUserID).(string)
	return id
}

// RoleFromContext retrieves the authenticated user's role from the context.
func RoleFromContext(ctx context.Context) model.Role {
	role, _ := ctx.Value(contextKeyRole).(model.Role)
	return role
}
