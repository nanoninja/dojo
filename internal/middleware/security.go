// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware

import (
	"net"
	"net/http"
)

// SecureHeaders sets HTTP security headers on every response.
// HSTS is only applied in production to avoid issues with local HTTP dev servers.
func SecureHeaders(env string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent the response from being embedded in an <iframe>.
			// Protects against clickjacking attacks.
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent browsers from guessing (sniffing) the content type.
			// Without this, a JSON response could be interpreted as executable JavaScript.
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Control how much referrer information is sent with requests.
			// "strict-origin-when-cross-origin" sends the origin only for cross-origin
			// requests, preventing internal URLs from leaking to third-party sites.
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Restrictive CSP for API responses:
			// this API does not need to execute scripts or embed third-party resources.
			// It reduces browser-side XSS impact if a response is interpreted as HTML.
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")

			// Restrictive CSP for API responses:
			// this API does not need to execute scripts or embed third-party resources.
			// It reduces browser-side XSS impact if a response is interpreted as HTML.
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Force browsers to use HTTPS for the next 365 days (production only).
			// Not set in development because local servers typically run over HTTP.
			if env == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPAllowList restricts access to requests coming from the given IP addresses.
func IPAllowList(allowed []string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, ip := range allowed {
		set[ip] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if _, ok := set[ip]; !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
