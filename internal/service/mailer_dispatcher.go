// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"fmt"
	"time"
)

// MailDispatchConfig controls sync mail sending reliability.
type MailDispatchConfig struct {
	Enabled        bool          // Keep true for now; allows easy future bypass.
	Timeout        time.Duration // Per send timeout.
	RetryAttempts  int           // Total attempts, including first try.
	RetryBaseDelay time.Duration // Initial backoff delay.
}

// resilientAuthMailer wraps AuthMailer with timeout + retry.
type resilientAuthMailer struct {
	next AuthMailer
	cfg  MailDispatchConfig
}

// NewResilientAuthMailer wraps an AuthMailer with timeout and retry logic.
func NewResilientAuthMailer(next AuthMailer, cfg MailDispatchConfig) AuthMailer {
	return &resilientAuthMailer{next: next, cfg: cfg}
}

func (m *resilientAuthMailer) SendAccountVerification(ctx context.Context, to, token string) error {
	return m.withRetry(ctx, "send_account_verification", func(c context.Context) error {
		return m.next.SendAccountVerification(c, to, token)
	})
}

func (m *resilientAuthMailer) SendPasswordReset(ctx context.Context, to, token string) error {
	return m.withRetry(ctx, "send_password_reset", func(c context.Context) error {
		return m.next.SendPasswordReset(c, to, token)
	})
}

func (m *resilientAuthMailer) SendOTP(ctx context.Context, to, code string) error {
	return m.withRetry(ctx, "send_otp", func(c context.Context) error {
		return m.next.SendOTP(c, to, code)
	})
}

func (m *resilientAuthMailer) withRetry(ctx context.Context, op string, fn func(context.Context) error) error {
	if !m.cfg.Enabled {
		return fn(ctx)
	}

	attempts := m.cfg.RetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	delay := m.cfg.RetryBaseDelay
	if delay <= 0 {
		delay = 200 * time.Millisecond
	}
	timeout := m.cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	var lastErr error

	for i := 0; i < attempts; i++ {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		err := fn(callCtx)
		cancel()

		if err == nil {
			return nil
		}
		lastErr = err
		if i == attempts-1 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}

	return fmt.Errorf("%s failed after %d attempts: %w", op, attempts, lastErr)
}
