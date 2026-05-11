// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware

import (
	"net/http"

	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
)

// RequireCSRF enforces a double-submit CSRF check on unsafe HTTP methods.
// It compares a CSRF cookie value with a request header value.
// Typical setup:
// - Cookie: "csrf_token"
// - Header: "X-CSRF-Token"
//
// Why this middleware can be disabled:
// CSRF risk mainly exists when authentication relies on browser cookies.
// In bearer-only mode, requests are usually authenticated via Authorization header,
// so CSRF protection can be turned off.
func RequireCSRF(enabled bool, cookieName, headerName string) func(http.Handler) http.Handler {
	if !enabled {
		// No-op middleware to keep router wiring simple across modes.
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods do not mutate state, so they are not CSRF-checked.
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Double-submit pattern: cookie and header must both exist and match.
			c, err := r.Cookie(cookieName)
			if err != nil || c.Value == "" {
				_ = httputil.Error(w, fault.Forbidden(nil))
				return
			}

			if r.Header.Get(headerName) != c.Value {
				_ = httputil.Error(w, fault.Forbidden(nil))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
