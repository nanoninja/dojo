// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nanoninja/dojo/internal/fault"
)

// Send writes a JSON response with the given HTTP status code.
// It sets the Content-Type header before writing the status and body.
func Send(w http.ResponseWriter, code int, data any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(data)
}

// OK writes a 200 OK JSON response.
func OK(w http.ResponseWriter, data any) error {
	return Send(w, http.StatusOK, data)
}

// PageMeta holds pagination metadata returned with every paginated response.
type PageMeta struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
	Total int `json:"total"`
}

// PageResponse wraps a paginated list with its metadata.
type PageResponse[T any] struct {
	Data []T      `json:"data"`
	Meta PageMeta `json:"meta"`
}

// OKPaginated writes a 200 JSON response with a paginated envelope.
func OKPaginated[T any](w http.ResponseWriter, data []T, page, limit, total int) error {
	if data == nil {
		data = []T{}
	}
	return Send(w, http.StatusOK, PageResponse[T]{
		Data: data,
		Meta: PageMeta{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// Created writes a 201 Created JSON response.
func Created(w http.ResponseWriter, data any) error {
	return Send(w, http.StatusCreated, data)
}

// NoContent writes a 204 No Content response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes a JSON error response derived from err.
// If err is a *fault.Fault, its code and message are used directly.
// Otherwise a 500 Internal Server Error is returned with a generic message,
// keeping the internal cause hidden from the client.
func Error(w http.ResponseWriter, err error) error {
	if f, ok := errors.AsType[*fault.Fault](err); ok {
		return Send(w, f.Code, map[string]string{"error": f.Message})
	}
	return Send(w, http.StatusInternalServerError, map[string]string{
		"error": http.StatusText(http.StatusInternalServerError),
	})
}
