// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const asyncTimeout = 30 * time.Second

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
