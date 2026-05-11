// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func newUserService(us *fakeUserStore) service.UserService {
	return service.NewUserService(us, &fakeLoginAuditStore{})
}

func TestUserService_Register(t *testing.T) {
	ctx := context.Background()
	svc := newUserService(newFakeUserStore())

	u := &model.User{
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Language:  "en",
		Timezone:  "UTC",
	}

	assert.NoError(t, svc.Register(ctx, u, "secret123"))
	assert.NotEqual(t, "", u.ID, "Register() did not set ID")
	assert.Equal(t, model.UserStatusPending, u.Status)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("secret123")), "password not hashed correctly")
}

func TestUserService_GetByID(t *testing.T) {
	ctx := context.Background()
	us := newFakeUserStore()
	svc := newUserService(us)

	u := &model.User{Email: "john@example.com", Language: "en", Timezone: "UTC"}
	assert.NoError(t, svc.Register(ctx, u, "secret"), "setup: Register()")

	t.Run("found", func(t *testing.T) {
		found, err := svc.GetByID(ctx, u.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, u.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})
}

func TestUserService_List(t *testing.T) {
	ctx := context.Background()
	us := newFakeUserStore()
	svc := newUserService(us)

	// Create 3 users.
	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		u := &model.User{Email: email, Language: "en", Timezone: "UTC"}
		assert.NoError(t, svc.Register(ctx, u, "secret"), "setup: Register()")
	}

	t.Run("returns all users and correct total", func(t *testing.T) {
		users, total, err := svc.List(ctx, store.UserFilter{Limit: 10})

		assert.NoError(t, err)
		assert.Len(t, users, 3)
		assert.Equal(t, 3, total)
	})

	t.Run("total is independent of limit", func(t *testing.T) {
		// Even with limit=1, total must reflect the full matching set.
		_, total, err := svc.List(ctx, store.UserFilter{Limit: 1})

		assert.NoError(t, err)
		assert.Equal(t, 3, total, "total should be 3 regardless of limit")
	})
}

func TestUserService_ChangePassword(t *testing.T) {
	ctx := context.Background()
	us := newFakeUserStore()
	svc := newUserService(us)

	u := &model.User{Email: "john@example.com", Language: "en", Timezone: "UTC"}
	assert.NoError(t, svc.Register(ctx, u, "old-password"), "setup: Register()")

	t.Run("correct password", func(t *testing.T) {
		assert.NoError(t, svc.ChangePassword(ctx, u.ID, "old-password", "new-password"))
	})

	t.Run("wrong current password", func(t *testing.T) {
		err := svc.ChangePassword(ctx, u.ID, "wrong-password", "whatever")
		assert.ErrorIs(t, err, service.ErrWrongPassword)
	})

	t.Run("user not found", func(t *testing.T) {
		err := svc.ChangePassword(ctx, "non-existent-id", "old-password", "new-password")
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})
}

func TestUserService_UpdateProfile(t *testing.T) {
	ctx := context.Background()
	us := newFakeUserStore()
	svc := newUserService(us)

	u := &model.User{
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Language:  "en",
		Timezone:  "UTC",
	}
	assert.NoError(t, svc.Register(ctx, u, "secret123"), "setup: Register()")

	u.FirstName = "Jane"
	u.LastName = "Roe"
	u.Timezone = "Europe/Paris"

	assert.NoError(t, svc.UpdateProfile(ctx, u))

	found, err := svc.GetByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Jane", found.FirstName)
	assert.Equal(t, "Roe", found.LastName)
	assert.Equal(t, "Europe/Paris", found.Timezone)
}

func TestUserService_Delete(t *testing.T) {
	ctx := context.Background()
	us := newFakeUserStore()
	svc := newUserService(us)

	u := &model.User{
		Email:    "john@example.com",
		Language: "en",
		Timezone: "UTC",
	}
	assert.NoError(t, svc.Register(ctx, u, "secret123"), "setup: Register()")
	assert.NoError(t, svc.Delete(ctx, u.ID))

	_, err := svc.GetByID(ctx, u.ID)
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestUserService_LoginHistory(t *testing.T) {
	ctx := context.Background()
	us := newFakeUserStore()
	svc := newUserService(us)

	u := &model.User{Email: "john@example.com", Language: "en", Timezone: "UTC"}
	assert.NoError(t, svc.Register(ctx, u, "secret"), "setup: Register()")

	logs, err := svc.LoginHistory(ctx, u.ID, 10)
	assert.NoError(t, err)
	assert.Empty(t, logs)
}
