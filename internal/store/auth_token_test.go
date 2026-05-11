// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

// newTestVerificationToken returns a valid verification token for tests.
func newTestVerificationToken(userID string, tokenType model.TokenType) *model.VerificationToken {
	return &model.VerificationToken{
		UserID:    userID,
		Token:     "test-token-value",
		Type:      tokenType,
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestAuthStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "verification_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	as := store.NewAuthStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestVerificationToken(u.ID, model.TokenTypeEmailVerification)
	assert.NoError(t, as.Create(ctx, tk))
	assert.NotEqual(t, "", tk.ID, "Create() did not set ID")
}

func TestAuthStore_FindActiveByUserAndType(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "verification_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	as := store.NewAuthStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	used := &model.VerificationToken{
		UserID:    u.ID,
		Token:     "used-token",
		Type:      model.TokenTypeOTP,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.NoError(t, as.Create(ctx, used), "setup: Create() used token")
	assert.NoError(t, as.MarkUsed(ctx, used.ID), "setup: MarkUsed()")

	active := &model.VerificationToken{
		UserID:    u.ID,
		Token:     "active-token",
		Type:      model.TokenTypeOTP,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.NoError(t, as.Create(ctx, active), "setup: Create() active token")

	found, err := as.FindActiveByUserAndType(ctx, u.ID, model.TokenTypeOTP)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, active.ID, found.ID)
}

func TestAuthStore_FindOne(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "verification_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	as := store.NewAuthStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestVerificationToken(u.ID, model.TokenTypeEmailVerification)
	assert.NoError(t, as.Create(ctx, tk), "setup: Create() token")

	t.Run("found", func(t *testing.T) {
		found, err := as.FindOne(ctx, store.TokenFilter{
			UserID: u.ID,
			Token:  tk.Token,
			Type:   tk.Type,
		})
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tk.ID, found.ID)
	})

	t.Run("not found — wrong token", func(t *testing.T) {
		found, err := as.FindOne(ctx, store.TokenFilter{
			UserID: u.ID,
			Token:  "wrong-token",
			Type:   tk.Type,
		})
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found — wrong type", func(t *testing.T) {
		found, err := as.FindOne(ctx, store.TokenFilter{
			UserID: u.ID,
			Token:  tk.Token,
			Type:   model.TokenTypePasswordReset,
		})
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestAuthStore_MarkUsed(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "verification_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	as := store.NewAuthStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestVerificationToken(u.ID, model.TokenTypeEmailVerification)
	assert.NoError(t, as.Create(ctx, tk), "setup: Create() token")
	assert.NoError(t, as.MarkUsed(ctx, tk.ID))

	// A token marked as used must no longer be retrievable.
	found, err := as.FindOne(ctx, store.TokenFilter{
		UserID: u.ID,
		Token:  tk.Token,
		Type:   tk.Type,
	})
	assert.NoError(t, err)
	assert.Nil(t, found, "token should not be findable after being marked used")
}

func TestAuthStore_IncrementAttempts(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "verification_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	as := store.NewAuthStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestVerificationToken(u.ID, model.TokenTypeOTP)
	assert.NoError(t, as.Create(ctx, tk), "setup: Create() token")
	assert.NoError(t, as.IncrementAttempts(ctx, tk.ID))
	assert.NoError(t, as.IncrementAttempts(ctx, tk.ID))

	found, err := as.FindOne(ctx, store.TokenFilter{
		UserID: u.ID,
		Token:  tk.Token,
		Type:   tk.Type,
	})
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, 2, found.Attempts)
}

func TestAuthStore_DeleteExpired(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "verification_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	as := store.NewAuthStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	// Token already expired.
	expired := &model.VerificationToken{
		UserID:    u.ID,
		Token:     "expired-token",
		Type:      model.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	assert.NoError(t, as.Create(ctx, expired), "setup: Create() expired token")

	// Still-valid token.
	valid := &model.VerificationToken{
		UserID:    u.ID,
		Token:     "valid-token",
		Type:      model.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.NoError(t, as.Create(ctx, valid), "setup: Create() valid token")
	assert.NoError(t, as.DeleteExpired(ctx, u.ID))

	// The valid token should still be retrievable.
	found, err := as.FindOne(ctx, store.TokenFilter{
		UserID: u.ID,
		Token:  valid.Token,
		Type:   valid.Type,
	})
	assert.NoError(t, err)
	assert.NotNil(t, found, "DeleteExpired() should not have deleted the valid token")
}
