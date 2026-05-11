// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package httputil_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
)

// ============================================================================
// ValidateUUID
// ============================================================================

func TestValidateUUID_Valid(t *testing.T) {
	err := httputil.ValidateUUID("01966b0a-1234-7abc-def0-1234567890ab")
	assert.NoError(t, err)
}

func TestValidateUUID_Invalid(t *testing.T) {
	err := httputil.ValidateUUID("not-a-uuid")
	assert.Error(t, err)
}

func TestValidateUUID_Empty(t *testing.T) {
	err := httputil.ValidateUUID("")
	assert.Error(t, err)
}

// ============================================================================
// Handle
// ============================================================================

func TestHandle_NoError_DoesNotWriteErrorResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := httputil.Handle(func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	}, logger)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandle_FaultError_WritesFaultCode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := httputil.Handle(func(_ http.ResponseWriter, _ *http.Request) error {
		return fault.NotFound("user", nil)
	}, logger)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandle_InternalFault_Returns500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := httputil.Handle(func(_ http.ResponseWriter, _ *http.Request) error {
		return fault.Internal(errors.New("db failure"))
	}, logger)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandle_NonFaultError_Returns500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := httputil.Handle(func(_ http.ResponseWriter, _ *http.Request) error {
		return errors.New("unexpected error")
	}, logger)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
