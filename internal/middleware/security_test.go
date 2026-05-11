// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/middleware"
)

func runSecureHeaders(env string) http.Header {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	middleware.SecureHeaders(env)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	return w.Header()
}

func TestSecureHeaders_SetsBaselineHeaders(t *testing.T) {
	h := runSecureHeaders("development")

	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
}

func TestSecureHeaders_SetsContentSecurityPolicy(t *testing.T) {
	h := runSecureHeaders("development")
	want := "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

	assert.Equal(t, want, h.Get("Content-Security-Policy"))
}

func TestSecureHeaders_ProductionSetsHSTS(t *testing.T) {
	h := runSecureHeaders("production")

	assert.Equal(t, "max-age=31536000; includeSubDomains", h.Get("Strict-Transport-Security"))
}

func TestSecureHeaders_NonProductionDoesNotSetHSTS(t *testing.T) {
	h := runSecureHeaders("test")

	assert.Equal(t, "", h.Get("Strict-Transport-Security"))
}

func TestSecureHeaders_SetsPermissionsPolicy(t *testing.T) {
	h := runSecureHeaders("development")

	assert.Equal(t, "geolocation=(), microphone=(), camera=()", h.Get("Permissions-Policy"))
}

func runIPAllowList(remoteAddr string, allowed []string) int {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.RemoteAddr = remoteAddr

	middleware.IPAllowList(allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	return w.Code
}

func TestIPAllowList_AllowsListedIP(t *testing.T) {
	code := runIPAllowList("192.168.1.1:1234", []string{"192.168.1.1"})

	assert.Equal(t, http.StatusOK, code)
}

func TestIPAllowList_BlocksUnlistedIP(t *testing.T) {
	code := runIPAllowList("10.0.0.1:1234", []string{"192.168.1.1"})

	assert.Equal(t, http.StatusForbidden, code)
}

func TestIPAllowList_AllowsMultipleIPs(t *testing.T) {
	allowed := []string{"192.168.1.1", "10.0.0.1"}

	assert.Equal(t, http.StatusOK, runIPAllowList("192.168.1.1:1234", allowed))
	assert.Equal(t, http.StatusOK, runIPAllowList("10.0.0.1:1234", allowed))
	assert.Equal(t, http.StatusForbidden, runIPAllowList("172.16.0.1:1234", allowed))
}
