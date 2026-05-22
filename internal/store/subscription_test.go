// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

func newTestSubscription(userID string) *model.Subscription {
	return &model.Subscription{
		UserID:    userID,
		Plan:      model.SubscriptionPlanMonthly,
		Status:    model.SubscriptionStatusActive,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
}

func TestSubscriptionStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "subscriptions", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewSubscriptionStore(db)

	sub := newTestSubscription(userID)
	assert.NoError(t, s.Create(ctx, sub))
	assert.NotEqual(t, "", sub.ID, "Create() did not set ID")
	assert.False(t, sub.StartedAt.IsZero(), "Create() did not set StartedAt")
}

func TestSubscriptionStore_FindActiveByUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "subscriptions", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewSubscriptionStore(db)

	sub := newTestSubscription(userID)
	require.NoError(t, s.Create(ctx, sub), "setup: Create()")

	t.Run("found active", func(t *testing.T) {
		got, err := s.FindActiveByUser(ctx, userID)
		assert.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, sub.ID, got.ID)
		assert.Equal(t, model.SubscriptionStatusActive, got.Status)
	})

	t.Run("not found for unknown user", func(t *testing.T) {
		got, err := s.FindActiveByUser(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Equal(t, (*model.Subscription)(nil), got)
	})
}

func TestSubscriptionStore_ListByUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "subscriptions", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewSubscriptionStore(db)

	sub := newTestSubscription(userID)
	require.NoError(t, s.Create(ctx, sub), "setup: Create()")

	t.Run("returns subscriptions for user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, sub.ID, got[0].ID)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})
}

func TestSubscriptionStore_Cancel(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "subscriptions", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewSubscriptionStore(db)

	sub := newTestSubscription(userID)
	require.NoError(t, s.Create(ctx, sub), "setup: Create()")

	assert.NoError(t, s.Cancel(ctx, sub.ID))

	got, err := s.FindActiveByUser(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, (*model.Subscription)(nil), got, "cancelled subscription should not be returned as active")
}
