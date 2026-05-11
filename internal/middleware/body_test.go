// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/middleware"
)

func TestMaxBodySize_UnderLimit_PassesThrough(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"value"}`))

	var reached bool
	middleware.MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.True(t, reached, "MaxBodySize() handler was not called for a body under the limit")
	require.Equalf(t, http.StatusOK, w.Code, "MaxBodySize() status = %d, want %d", w.Code, http.StatusOK)
}

func TestMaxBodySize_OverLimit_BodyReadReturnsError(t *testing.T) {
	const limit = 10
	body := strings.NewReader("this body is definitely longer than ten bytes")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", body)

	var readErr error
	middleware.MaxBodySize(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, limit+1)
		_, readErr = r.Body.Read(buf)
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.Error(t, readErr, "MaxBodySize() expected a read error when body exceeds limit, got nil")
}

func TestMaxBodySize_GetRequest_NoBody_PassesThrough(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	var reached bool
	middleware.MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.True(t, reached, "MaxBodySize() handler was not called for a GET with no body")
}

func TestMaxBodySize_ExactLimit_PassesThrough(t *testing.T) {
	const limit = 5
	body := strings.NewReader("hello") // exactly 5 bytes

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", body)

	var reached bool
	middleware.MaxBodySize(limit)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	require.True(t, reached, "MaxBodySize() handler was not called for a body at exactly the limit")
}
