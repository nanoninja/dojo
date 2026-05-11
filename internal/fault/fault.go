// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package fault provides typed HTTP error constructors.
package fault

import "net/http"

// ErrorResponse is the JSON body returned by the API on all error responses.
// Every non-2xx response has exactly one "error" key with a human-readable message.
// This type exists solely to document the contract in the Swagger UI — the actual
// serialization is handled by httputil.Error which writes map[string]string{"error": ...}.
type ErrorResponse struct {
	Error string `json:"error" example:"a descriptive error message"`
}

// Fault represents an application error with an HTTP status code,
// a public message safe to expose to the client, and an optional internal cause.
type Fault struct {
	Code    int
	Message string
	Cause   error
}

func (f *Fault) Error() string {
	return f.Message
}

func (f *Fault) Unwrap() error {
	return f.Cause
}

// New creates a Fault with the given HTTP status code, client-facing message, and optional cause.
func New(code int, message string, cause error) *Fault {
	return &Fault{Code: code, Message: message, Cause: cause}
}

// BadRequest returns a 400 fault with a descriptive message.
func BadRequest(message string, cause error) *Fault {
	return New(http.StatusBadRequest, message, cause)
}

// Unauthorized returns a 401 fault with a generic message.
func Unauthorized(cause error) *Fault {
	return New(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), cause)
}

// Forbidden returns a 403 fault with a generic message.
func Forbidden(cause error) *Fault {
	return New(http.StatusForbidden, http.StatusText(http.StatusForbidden), cause)
}

// NotFound returns a 404 fault with a message identifying the missing resource.
func NotFound(resource string, cause error) *Fault {
	return New(http.StatusNotFound, resource+" not found", cause)
}

// Conflict returns a 409 fault with a descriptive message.
func Conflict(message string, cause error) *Fault {
	return New(http.StatusConflict, message, cause)
}

// TooManyRequests returns a 429 fault with a generic message.
func TooManyRequests(cause error) *Fault {
	return New(
		http.StatusTooManyRequests,
		http.StatusText(http.StatusTooManyRequests),
		cause,
	)
}

// Internal returns a 500 fault. The cause is kept server-side and never exposed to the client.
func Internal(cause error) *Fault {
	return New(
		http.StatusInternalServerError,
		http.StatusText(http.StatusInternalServerError),
		cause,
	)
}
