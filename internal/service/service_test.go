// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"fmt"
	"time"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// fakeUserStore
// ============================================================================

type fakeUserStore struct {
	users   map[string]*model.User // by ID
	byEmail map[string]*model.User // by email
	seq     int
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users:   make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (f *fakeUserStore) nextID() string {
	f.seq++
	return fmt.Sprintf("user-%d", f.seq)
}

func (f *fakeUserStore) List(_ context.Context, _ store.UserFilter) ([]model.User, int, error) {
	var result []model.User
	for _, u := range f.users {
		if u.Status != model.UserStatusDeleted {
			result = append(result, *u)
		}
	}
	return result, len(result), nil
}

func (f *fakeUserStore) FindByID(_ context.Context, id string) (*model.User, error) {
	u, ok := f.users[id]
	if !ok || u.Status == model.UserStatusDeleted {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) FindByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := f.byEmail[email]
	if !ok || u.Status == model.UserStatusDeleted {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) FindCredentialsByID(_ context.Context, id string) (*model.User, error) {
	u, err := f.FindByID(context.TODO(), id)
	return u, err
}

func (f *fakeUserStore) Create(_ context.Context, u *model.User) error {
	u.ID = f.nextID()
	cp := *u
	f.users[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return nil
}

func (f *fakeUserStore) Update(_ context.Context, u *model.User) error {
	if _, ok := f.users[u.ID]; !ok {
		return fmt.Errorf("user not found")
	}
	cp := *u
	f.users[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return nil
}

func (f *fakeUserStore) UpdatePassword(_ context.Context, id, hash string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.PasswordHash = hash
	return nil
}

func (f *fakeUserStore) UpdateLastLogin(_ context.Context, id, _ string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	now := time.Now()
	u.LastLoginAt = &now
	u.LoginCount++
	return nil
}

func (f *fakeUserStore) UpdateVerified(_ context.Context, id string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.IsVerified = true
	return nil
}

func (f *fakeUserStore) Delete(_ context.Context, id string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.Status = model.UserStatusDeleted
	return nil
}

func (f *fakeUserStore) IncrementFailedLogin(_ context.Context, id string) error {
	if u, ok := f.users[id]; ok {
		u.FailedLoginAttempts++
	}
	return nil
}

func (f *fakeUserStore) LockAccount(_ context.Context, id string, until time.Time) error {
	if u, ok := f.users[id]; ok {
		u.LockedUntil = &until
	}
	return nil
}

func (f *fakeUserStore) ResetFailedLogin(_ context.Context, id string) error {
	if u, ok := f.users[id]; ok {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

// ============================================================================
// fakeAuthStore
// ============================================================================

type fakeAuthStore struct {
	tokens map[string]*model.VerificationToken // by ID
	seq    int
}

func newFakeAuthStore() *fakeAuthStore {
	return &fakeAuthStore{tokens: make(map[string]*model.VerificationToken)}
}

func (f *fakeAuthStore) nextID() string {
	f.seq++
	return fmt.Sprintf("token-%d", f.seq)
}

func (f *fakeAuthStore) Create(_ context.Context, t *model.VerificationToken) error {
	t.ID = f.nextID()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	cp := *t
	f.tokens[t.ID] = &cp
	return nil
}

func (f *fakeAuthStore) FindActiveByUserAndType(_ context.Context, userID string, tokenType model.TokenType) (*model.VerificationToken, error) {
	var latest *model.VerificationToken

	for _, t := range f.tokens {
		// Keep only active tokens for this user and token type.
		if t.UserID != userID || t.Type != tokenType {
			continue
		}
		if t.UsedAt != nil || !t.ExpiresAt.After(time.Now()) {
			continue
		}

		// Return the most recently created active token.
		if latest == nil || t.CreatedAt.After(latest.CreatedAt) {
			tt := *t
			latest = &tt
		}
	}

	if latest == nil {
		return nil, nil
	}
	return latest, nil
}

func (f *fakeAuthStore) FindOne(_ context.Context, filter store.TokenFilter) (*model.VerificationToken, error) {
	for _, t := range f.tokens {
		if t.UserID == filter.UserID &&
			t.Token == filter.Token &&
			t.Type == filter.Type &&
			t.UsedAt == nil &&
			t.ExpiresAt.After(time.Now()) {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeAuthStore) MarkUsed(_ context.Context, id string) error {
	t, ok := f.tokens[id]
	if !ok {
		return fmt.Errorf("token not found")
	}
	now := time.Now()
	t.UsedAt = &now
	return nil
}

func (f *fakeAuthStore) IncrementAttempts(_ context.Context, id string) error {
	t, ok := f.tokens[id]
	if !ok {
		return fmt.Errorf("token not found")
	}
	t.Attempts++
	return nil
}

func (f *fakeAuthStore) DeleteExpired(_ context.Context, userID string) error {
	for id, t := range f.tokens {
		if t.UserID == userID && t.ExpiresAt.Before(time.Now()) {
			delete(f.tokens, id)
		}
	}
	return nil
}

// ============================================================================
// fakeRefreshTokenStore
// ============================================================================

type fakeRefreshTokenStore struct {
	tokens     map[string]*model.RefreshToken // by hash
	seq        int
	failRotate bool // when true, RotateToken returns an error without modifying state
}

func newFakeRefreshTokenStore() *fakeRefreshTokenStore {
	return &fakeRefreshTokenStore{tokens: make(map[string]*model.RefreshToken)}
}

func (f *fakeRefreshTokenStore) nextID() string {
	f.seq++
	return fmt.Sprintf("rt-%d", f.seq)
}

func (f *fakeRefreshTokenStore) Create(_ context.Context, t *model.RefreshToken) error {
	t.ID = f.nextID()
	cp := *t
	f.tokens[t.TokenHash] = &cp
	return nil
}

func (f *fakeRefreshTokenStore) FindByHash(_ context.Context, hash string) (*model.RefreshToken, error) {
	t, ok := f.tokens[hash]
	if !ok || t.RevokedAt != nil || t.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (f *fakeRefreshTokenStore) Revoke(_ context.Context, id string) error {
	for _, t := range f.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
			return nil
		}
	}
	return fmt.Errorf("token not found")
}

func (f *fakeRefreshTokenStore) RevokeAllForUser(_ context.Context, userID string) error {
	now := time.Now()
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRefreshTokenStore) RotateToken(ctx context.Context, oldID string, newToken *model.RefreshToken) error {
	if f.failRotate {
		return fmt.Errorf("simulated RotateToken failure")
	}
	if err := f.Revoke(ctx, oldID); err != nil {
		return err
	}
	return f.Create(ctx, newToken)
}

func (f *fakeRefreshTokenStore) DeleteExpired(_ context.Context, userID string) error {
	for hash, t := range f.tokens {
		if t.UserID == userID && t.ExpiresAt.Before(time.Now()) {
			delete(f.tokens, hash)
		}
	}
	return nil
}

// ============================================================================
// fakeMailer
// ============================================================================

type fakeMailer struct {
	sentVerification []string
	sentReset        []string
	sentOTP          []string
}

func (f *fakeMailer) SendAccountVerification(_ context.Context, to, _ string) error {
	f.sentVerification = append(f.sentVerification, to)
	return nil
}

func (f *fakeMailer) SendPasswordReset(_ context.Context, to, _ string) error {
	f.sentReset = append(f.sentReset, to)
	return nil
}

func (f *fakeMailer) SendOTP(_ context.Context, to, _ string) error {
	f.sentOTP = append(f.sentOTP, to)
	return nil
}

// ============================================================================
// fakeLoginAuditStore
// ============================================================================

type fakeLoginAuditStore struct {
	logs []model.LoginAuditLog
}

func (f *fakeLoginAuditStore) Create(_ context.Context, log *model.LoginAuditLog) error {
	f.logs = append(f.logs, *log)
	return nil
}

func (f *fakeLoginAuditStore) FindByUser(_ context.Context, _ string, _ int) ([]model.LoginAuditLog, error) {
	return f.logs, nil
}

func (f *fakeLoginAuditStore) List(_ context.Context, _ store.AuditFilter) ([]model.LoginAuditLog, int, error) {
	return f.logs, len(f.logs), nil
}

func (f *fakeLoginAuditStore) Purge(_ context.Context, _ time.Duration, _ int) (int64, error) {
	return 0, nil
}
