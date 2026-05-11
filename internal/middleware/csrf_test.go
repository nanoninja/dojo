// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/middleware"
)

func TestRequireCSRF_UnsafeMethod_WithMatchingToken_Allows(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-123"})
	r.Header.Set("X-CSRF-Token", "token-123")

	middleware.RequireCSRF(true, "csrf_token", "X-CSRF-Token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equalf(t, http.StatusOK, w.Code, "status = %d, want %d", w.Code, http.StatusOK)
}

func TestRequireCSRF_UnsafeMethod_MissingCookie_Returns403(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-CSRF-Token", "token-123")

	middleware.RequireCSRF(true, "csrf_token", "X-CSRF-Token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equalf(t, http.StatusForbidden, w.Code, "status = %d, want %d", w.Code, http.StatusForbidden)
}

func TestRequireCSRF_UnsafeMethod_Mismatch_Returns403(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: "csrf_token", Value: "cookie-token"})
	r.Header.Set("X-CSRF-Token", "header-token")

	middleware.RequireCSRF(true, "csrf_token", "X-CSRF-Token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equalf(t, http.StatusForbidden, w.Code, "status = %d, want %d", w.Code, http.StatusForbidden)
}

func TestRequireCSRF_SafeMethod_BypassesCheck(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	middleware.RequireCSRF(true, "csrf_token", "X-CSRF-Token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equalf(t, http.StatusOK, w.Code, "status = %d, want %d", w.Code, http.StatusOK)
}

func TestRequireCSRF_Disabled_BypassesCheck(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)

	middleware.RequireCSRF(false, "csrf_token", "X-CSRF-Token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Equalf(t, http.StatusOK, w.Code, "status = %d, want %d", w.Code, http.StatusOK)
}
