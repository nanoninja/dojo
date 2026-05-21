// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

func newTestConsent(userID string) *model.Consent {
	ip := "192.168.1.1"
	v := "1.0"
	return &model.Consent{
		UserID:     userID,
		Type:       model.ConsentTypeTermsOfService,
		Version:    &v,
		IsAccepted: true,
		IPAddress:  &ip,
		UserAgent:  "Mozilla/5.0",
		Source:     model.ConsentSourceRegistration,
	}
}

func TestConsentStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "user_consents", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewConsentStore(db, testutil.NewTestCipher(t))

	c := newTestConsent(userID)
	assert.NoError(t, s.Create(ctx, c))
	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
	assert.False(t, c.CreatedAt.IsZero(), "Create() did not set CreatedAt")
}

func TestConsentStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "user_consents", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewConsentStore(db, testutil.NewTestCipher(t))

	c := newTestConsent(userID)
	require.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByID(ctx, c.ID)
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
		assert.Equal(t, model.ConsentTypeTermsOfService, got.Type)
		assert.Equal(t, "192.168.1.1", *got.IPAddress)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Equal(t, (*model.Consent)(nil), got)
	})
}

func TestConsentStore_ListByUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "user_consents", "users")

	ctx := context.Background()
	userID := setupInstructor(t, db)
	s := store.NewConsentStore(db, testutil.NewTestCipher(t))

	c := newTestConsent(userID)
	require.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("returns consents for user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, c.ID, got[0].ID)
		assert.Equal(t, "192.168.1.1", *got[0].IPAddress)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})
}
