// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

func newTestPurchase(userID, itemID string) *model.Purchase {
	return &model.Purchase{
		UserID:      userID,
		Type:        model.PurchaseTypeCourse,
		ItemID:      itemID,
		Status:      model.PurchaseStatusCompleted,
		AmountCents: 1999,
		Currency:    "EUR",
	}
}

func setupPurchase(t testing.TB, db *database.DB) (userID, courseID string) {
	t.Helper()
	courseID = setupCourse(t, db)
	userID = setupInstructor(t, db)
	return userID, courseID
}

func TestPurchaseStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "purchases", "courses", "users")

	ctx := context.Background()
	userID, courseID := setupPurchase(t, db)
	s := store.NewPurchaseStore(db)

	p := newTestPurchase(userID, courseID)
	assert.NoError(t, s.Create(ctx, p))
	assert.NotEqual(t, "", p.ID, "Create() did not set ID")
	assert.False(t, p.CreatedAt.IsZero(), "Create() did not set CreatedAt")
}

func TestPurchaseStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "purchases", "courses", "users")

	ctx := context.Background()
	userID, courseID := setupPurchase(t, db)
	s := store.NewPurchaseStore(db)

	p := newTestPurchase(userID, courseID)
	require.NoError(t, s.Create(ctx, p), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByID(ctx, p.ID)
		assert.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, p.ID, got.ID)
		assert.Equal(t, int64(1999), got.AmountCents)
		assert.Equal(t, "EUR", got.Currency)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Equal(t, (*model.Purchase)(nil), got)
	})
}

func TestPurchaseStore_ListByUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "purchases", "courses", "users")

	ctx := context.Background()
	userID, courseID := setupPurchase(t, db)
	s := store.NewPurchaseStore(db)

	p := newTestPurchase(userID, courseID)
	require.NoError(t, s.Create(ctx, p), "setup: Create()")

	t.Run("returns purchases for user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, p.ID, got[0].ID)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})
}

func TestPurchaseStore_Refund(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "purchases", "courses", "users")

	ctx := context.Background()
	userID, courseID := setupPurchase(t, db)
	s := store.NewPurchaseStore(db)

	p := newTestPurchase(userID, courseID)
	require.NoError(t, s.Create(ctx, p), "setup: Create()")

	assert.NoError(t, s.Refund(ctx, p.ID))

	got, err := s.FindByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.PurchaseStatusRefunded, got.Status)
	assert.False(t, got.RefundedAt == nil, "Refund() did not set RefundedAt")
}
