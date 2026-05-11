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

// newTestUser returns a minimal valid user for use in store tests.
func newTestUser() *model.User {
	return &model.User{
		Email:     "john.doe@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Language:  "en",
		Timezone:  "UTC",
		Status:    model.UserStatusPending,
	}
}

func TestUserStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	u := newTestUser()

	assert.NoError(t, s.Create(context.Background(), u))
	assert.NotEqual(t, "", u.ID, "Create() did not set ID")
}

func TestUserStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create() user")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, u.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, u.Email, found.Email)
	})

	t.Run("not found", func(t *testing.T) {
		// Use a correctly formatted UUID (36 chars, hexadecimal)
		// that does not exist in the table.
		found, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserStore_FindByEmail(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create() user")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByEmail(ctx, u.Email)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, u.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByEmail(ctx, "unknown@example.com")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserStore_FindCredentialsByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create() user")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindCredentialsByID(ctx, u.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, u.ID, found.ID)
		assert.Equal(t, u.Email, found.Email)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindCredentialsByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserStore_UpdateVerified(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create()")
	assert.NoError(t, s.UpdateVerified(ctx, u.ID))

	found, err := s.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.True(t, found.IsVerified, "UpdateVerified() is_verified should be true")
}

func TestUserStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, u.ID))

	// Soft delete - FindByID
	found, err := s.FindByID(ctx, u.ID)

	assert.NoError(t, err)
	assert.Nil(t, found, "user should not be findable after soft delete")
}

func TestUserStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u1 := newTestUser()
	u1.Email = "alice@example.com"
	u1.FirstName = "Alice"
	u1.Status = model.UserStatusActive
	assert.NoError(t, s.Create(ctx, u1), "setup: Create() u1")

	u2 := newTestUser()
	u2.Email = "bob@example.com"
	u2.FirstName = "Bob"
	u2.Status = model.UserStatusSuspended
	assert.NoError(t, s.Create(ctx, u2), "setup: Create() u2")

	u3 := newTestUser()
	u3.Email = "alina@example.com"
	u3.FirstName = "Alina"
	u3.Status = model.UserStatusActive
	assert.NoError(t, s.Create(ctx, u3), "setup: Create() u3")

	t.Run("status filter and total", func(t *testing.T) {
		users, total, err := s.List(ctx, store.UserFilter{
			Status: string(model.UserStatusActive),
			Limit:  10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, int64(total))
		assert.Len(t, users, 2)
	})

	t.Run("search + pagination", func(t *testing.T) {
		_, total, err := s.List(ctx, store.UserFilter{
			Search: "ali",
			Limit:  1,
		})
		assert.NoError(t, err)

		// "Alice" and "Alina" must both match despite limit=1.
		assert.Equal(t, 2, int64(total))
	})
}

func TestUserStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create()")

	company := "Acme"
	vat := "FR123456789"
	u.FirstName = "Jane"
	u.LastName = "Roe"
	u.CompanyName = &company
	u.VATNumber = &vat
	u.Language = "fr"
	u.Timezone = "Europe/Paris"
	assert.NoError(t, s.Update(ctx, u))

	found, err := s.FindByID(ctx, u.ID)

	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Jane", found.FirstName)
	assert.Equal(t, "Roe", found.LastName)
	assert.NotNil(t, found.CompanyName)
	assert.Equal(t, "Acme", *found.CompanyName)
	assert.Equal(t, "FR123456789", *found.VATNumber)
}

func TestUserStore_UpdatePassword(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	u.PasswordHash = "old-hash"

	assert.NoError(t, s.Create(ctx, u), "setup; Create()")
	assert.NoError(t, s.UpdatePassword(ctx, u.ID, "new-hash"))

	found, err := s.FindByEmail(ctx, u.Email)

	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "new-hash", found.PasswordHash)
}

func TestUserStore_UpdateLastLogin(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create() user")
	assert.NoError(t, s.UpdateLastLogin(ctx, u.ID, "203.0.113.42"))

	found, err := s.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.NotNil(t, found.LastLoginAt, "UpdateLastLogin() did not set last_login_at")
	assert.True(t, time.Since(*found.LastLoginAt) < time.Minute, "last_login_at looks stale")
	assert.NotNil(t, found.LastLoginIP)
	assert.Equal(t, "203.0.113.42", *found.LastLoginIP)
	assert.Equal(t, 1, found.LoginCount)
}

func TestUserStore_LockoutMethods(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "users")

	s := store.NewUserStore(db, testutil.NewTestCipher(t))
	ctx := context.Background()

	u := newTestUser()
	assert.NoError(t, s.Create(ctx, u), "setup: Create()")

	// IncrementFailedLogin
	assert.NoError(t, s.IncrementFailedLogin(ctx, u.ID))
	assert.NoError(t, s.IncrementFailedLogin(ctx, u.ID))

	found, err := s.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, found.FailedLoginAttempts)

	// LockAccount
	until := time.Now().Add(15 * time.Minute)
	assert.NoError(t, s.LockAccount(ctx, u.ID, until))

	found, err = s.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found.LockedUntil)
	assert.True(t, found.LockedUntil.After(time.Now()))

	// ResetFailedLogin
	assert.NoError(t, s.ResetFailedLogin(ctx, u.ID))

	found, err = s.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, found.FailedLoginAttempts)
	assert.Nil(t, found.LockedUntil)
}
