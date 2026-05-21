// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

func newTestCertificate(userID, courseID string) *model.Certificate {
	return &model.Certificate{
		UserID:   userID,
		CourseID: courseID,
	}
}
func TestCertificateStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "certificates", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)

	s := store.NewCertificateStore(db)
	c := newTestCertificate(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
	assert.NotEqual(t, "", c.UUID, "Create() did not set UUID")
	assert.False(t, c.IssuedAt.IsZero(), "Create() did not set IssuedAt")
}

func TestCertificateStore_Create_Idempotent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "certificates", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)

	s := store.NewCertificateStore(db)
	c := newTestCertificate(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, c))

	c2 := newTestCertificate(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, c2), "second insert on same use/course should not error")
}

func TestCertificateStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "certificates", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)

	s := store.NewCertificateStore(db)
	c := newTestCertificate(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByID(ctx, c.ID)

		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
		assert.Equal(t, c.UUID, got.UUID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Equal(t, (*model.Certificate)(nil), got)
	})
}

func TestCertificateStore_FindByUUID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "certificates", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)

	s := store.NewCertificateStore(db)
	c := newTestCertificate(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByUUID(ctx, c.UUID)

		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
		assert.Equal(t, c.UUID, got.UUID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByUUID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Equal(t, (*model.Certificate)(nil), got)
	})
}

func TestCertificateStore_ListByUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "certificates", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)

	s := store.NewCertificateStore(db)
	c := newTestCertificate(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("returns certificates for user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, instructorID)

		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, c.ID, got[0].ID)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		got, err := s.ListByUser(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})
}
