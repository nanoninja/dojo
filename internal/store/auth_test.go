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

// newTestRefreshToken returns a valid token for tests.
// A user must be created first and its ID passed as argument.
func newTestRefreshToken(userID string) *model.RefreshToken {
	return &model.RefreshToken{
		UserID:    userID,
		TokenHash: "testhash-unique-value",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
}

func TestRefreshTokenStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "refresh_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	rs := store.NewRefreshTokenStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestRefreshToken(u.ID)
	assert.NoError(t, rs.Create(ctx, tk))
	assert.NotEqual(t, "", tk.ID, "Create() did not set ID")
}

func TestRefreshTokenStore_FindByHash(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "refresh_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	rs := store.NewRefreshTokenStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestRefreshToken(u.ID)
	assert.NoError(t, rs.Create(ctx, tk), "setup: Create() token")

	t.Run("found", func(t *testing.T) {
		found, err := rs.FindByHash(ctx, tk.TokenHash)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, u.ID, found.UserID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := rs.FindByHash(ctx, "non-existent-hash")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestRefreshTokenStore_Revoke(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "refresh_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	rs := store.NewRefreshTokenStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	tk := newTestRefreshToken(u.ID)
	assert.NoError(t, rs.Create(ctx, tk), "setup: Create() token")
	assert.NoError(t, rs.Revoke(ctx, tk.ID))

	// A revoked token must no longer be retrievable.
	found, err := rs.FindByHash(ctx, tk.TokenHash)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "refresh_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	rs := store.NewRefreshTokenStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	// Create two tokens for the same user.
	tk1 := &model.RefreshToken{UserID: u.ID, TokenHash: "hash-one", ExpiresAt: time.Now().Add(time.Hour)}
	tk2 := &model.RefreshToken{UserID: u.ID, TokenHash: "hash-two", ExpiresAt: time.Now().Add(time.Hour)}

	assert.NoError(t, rs.Create(ctx, tk1), "setup: Create() tk1")
	assert.NoError(t, rs.Create(ctx, tk2), "setup: Create() tk2")
	assert.NoError(t, rs.RevokeAllForUser(ctx, u.ID))

	// Both tokens must no longer be retrievable.
	for _, hash := range []string{"hash-one", "hash-two"} {
		found, err := rs.FindByHash(ctx, hash)

		assert.NoError(t, err)
		assert.Nilf(t, found, "token %q should not be findable after RevokeAllForUser", hash)
	}
}

func TestRefreshTokenStore_DeleteExpired(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "refresh_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	rs := store.NewRefreshTokenStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	// Already-expired token (in the past).
	expired := &model.RefreshToken{
		UserID:    u.ID,
		TokenHash: "expired-hash",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	assert.NoError(t, rs.Create(ctx, expired), "setup: Create() expired token")

	// Still-valid token.
	valid := &model.RefreshToken{
		UserID:    u.ID,
		TokenHash: "valid-hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.NoError(t, rs.Create(ctx, valid), "setup: Create() valid token")
	assert.NoError(t, rs.DeleteExpired(ctx, u.ID))

	// The valid token should still be retrievable.
	found, err := rs.FindByHash(ctx, valid.TokenHash)
	assert.NoError(t, err)
	assert.NotNil(t, found, "DeleteExpired() should not have deleted the valid token")
}

func TestRefreshTokenStore_RotateToken(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "refresh_tokens", "users")

	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	rs := store.NewRefreshTokenStore(db)
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	oldToken := &model.RefreshToken{
		UserID:    u.ID,
		TokenHash: "rotate-old-hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.NoError(t, rs.Create(ctx, oldToken), "setup: Create() old token")

	newToken := &model.RefreshToken{
		UserID:    u.ID,
		TokenHash: "rotate-new-hash",
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	assert.NoError(t, rs.RotateToken(ctx, oldToken.ID, newToken))
	assert.NotEqual(t, "", newToken.ID)

	// Old token must no longer be active.
	oldFound, err := rs.FindByHash(ctx, oldToken.TokenHash)
	assert.NoError(t, err)
	assert.Nil(t, oldFound)

	// New token must be active.
	newFound, err := rs.FindByHash(ctx, newToken.TokenHash)
	assert.NoError(t, err)
	assert.NotNil(t, newFound)
}
