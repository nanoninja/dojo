// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package httputil

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/nanoninja/dojo/internal/fault"
)

// HandlerFunc is an HTTP handler that returns an error.
// Use Handle to adapt it to the standard http.HandlerFunc.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts a HandlerFunc to the standard http.HandlerFunc.
// If the handler returns an error, it is logged and the appropriate
// HTTP response is sent to the client via Error.
func Handle(h HandlerFunc, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			if f, ok := errors.AsType[*fault.Fault](err); ok {
				if f.Code == http.StatusInternalServerError {
					logger.Error("internal error", "error", f.Cause, "path", r.URL.Path)
				}
			} else {
				logger.Error("unhandled error", "error", err, "path", r.URL.Path)
			}
			_ = Error(w, err)
		}
	}
}
