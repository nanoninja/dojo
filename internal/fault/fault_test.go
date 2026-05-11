// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package fault

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
)

func TestFault_Error(t *testing.T) {
	f := &Fault{
		Code:    http.StatusBadRequest,
		Message: "invalid payload",
	}
	require.Equal(t, "invalid payload", f.Error())
}

func TestFault_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	f := &Fault{
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
		Cause:   cause,
	}
	require.ErrorIs(t, f, cause)
}

func TestConflict(t *testing.T) {
	cause := errors.New("duplicate")
	f := Conflict("email already in use", cause)

	require.Equal(t, http.StatusConflict, f.Code)
	require.Equal(t, "email already in use", f.Message)
	require.ErrorIs(t, f, cause)
}

func TestInternal(t *testing.T) {
	cause := errors.New("db timeout")
	f := Internal(cause)

	require.Equal(t, http.StatusInternalServerError, f.Code)
	require.Equal(t, http.StatusText(http.StatusInternalServerError), f.Message)
	require.ErrorIs(t, f, cause)
}

func TestFault_TooManyRequests(t *testing.T) {
	f := TooManyRequests(nil)

	assert.Equal(t, http.StatusTooManyRequests, f.Code)
}
