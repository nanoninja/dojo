// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nanoninja/assert"
)

type fakeAuthMailerForDispatch struct {
	calls int
	errs  []error
}

func (f *fakeAuthMailerForDispatch) SendAccountVerification(_ context.Context, _, _ string) error {
	f.calls++
	if len(f.errs) == 0 {
		return nil
	}
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}

func (f *fakeAuthMailerForDispatch) SendPasswordReset(ctx context.Context, to, token string) error {
	return f.SendAccountVerification(ctx, to, token)
}

func (f *fakeAuthMailerForDispatch) SendOTP(ctx context.Context, to, code string) error {
	return f.SendAccountVerification(ctx, to, code)
}

func TestResilientAuthMailer_SuccessFirstTry(t *testing.T) {
	next := &fakeAuthMailerForDispatch{}
	m := NewResilientAuthMailer(next, MailDispatchConfig{
		Enabled:        true,
		Timeout:        500 * time.Millisecond,
		RetryAttempts:  3,
		RetryBaseDelay: 30 * time.Millisecond,
	})

	assert.NoError(t, m.SendOTP(context.Background(), "a@b.com", "123456"))
	assert.Equal(t, 1, next.calls)
}

func TestResilientAuthMailer_RetryThenSuccess(t *testing.T) {
	next := &fakeAuthMailerForDispatch{
		errs: []error{errors.New("temp"), nil},
	}
	m := NewResilientAuthMailer(next, MailDispatchConfig{
		Enabled:        true,
		Timeout:        500 * time.Millisecond,
		RetryAttempts:  3,
		RetryBaseDelay: 5 * time.Millisecond,
	})

	assert.NoError(t, m.SendPasswordReset(context.Background(), "a@b.com", "tok"))
	assert.Equal(t, 2, next.calls)
}

func TestResilientAuthMailer_AllAttemptsFail(t *testing.T) {
	next := &fakeAuthMailerForDispatch{
		errs: []error{errors.New("e1"), errors.New("e2"), errors.New("e3")},
	}
	m := NewResilientAuthMailer(next, MailDispatchConfig{
		Enabled:        true,
		Timeout:        500 * time.Millisecond,
		RetryAttempts:  3,
		RetryBaseDelay: 5 * time.Millisecond,
	})

	assert.NotNil(t, m.SendAccountVerification(context.Background(), "a@b.com", "tok"))
	assert.Equal(t, 3, next.calls)
}
