// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"context"
	"log/slog"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	maxPageLimit = 100
	asyncTimeout = 30 * time.Second
)

// parsePage extracts page and limit from query parameters, applying defaults and a maximum limit.
func parsePage(q url.Values) (page, limit int) {
	page = parseIntQuery(q.Get("page"), 1)
	limit = min(parseIntQuery(q.Get("limit"), 20), maxPageLimit)
	return
}

// parseIntQuery parses a query parameter as int, returning def if absent or invalid.
func parseIntQuery(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}

// parseBoolPtr parses a query parameter as *bool, returning nil if absent or invalid.
func parseBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return &b
}

// sendAsync runs fn in a background goroutine with a fresh context and logs any error.
// Use this for best-effort operations (email, notifications) that must not block
// the HTTP response and must not use r.Context() — which is cancelled on response send.
func sendAsync(
	wg *sync.WaitGroup,
	logger *slog.Logger,
	fn func(context.Context) error,
	logKey string, attrs ...any,
) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), asyncTimeout)
		defer cancel()

		if err := fn(ctx); err != nil {
			logger.Warn(logKey, append(attrs, "error", err)...)
		}
	}()
}
