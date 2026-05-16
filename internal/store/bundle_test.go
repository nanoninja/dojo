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

func newTestBundle(instructorID string) *model.Bundle {
	return &model.Bundle{
		InstructorID: instructorID,
		Slug:         "go-bundle",
		Title:        "Go Bundle",
		IsFree:       false,
		PriceCents:   4900,
		Currency:     "EUR",
		IsPublished:  false,
		SortOrder:    10,
	}
}

// ============================================================================
// BundleStore
// ============================================================================

func TestBundleStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundles", "users")

	s := store.NewBundleStore(db)
	b := newTestBundle(setupInstructor(t, db))

	assert.NoError(t, s.Create(context.Background(), b))
	assert.NotEqual(t, "", b.ID, "Create() did not set ID")
}

func TestBundleStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundles", "users")

	s := store.NewBundleStore(db)
	ctx := context.Background()

	b := newTestBundle(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, b), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, b.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, b.Slug, found.Slug)
		assert.Equal(t, b.Title, found.Title)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestBundleStore_FindBySlug(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundles", "users")

	s := store.NewBundleStore(db)
	ctx := context.Background()

	b := newTestBundle(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, b), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, b.Slug)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, b.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, "unkown-slug")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestBundleStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundles", "users")

	instructorID := setupInstructor(t, db)
	s := store.NewBundleStore(db)
	ctx := context.Background()

	b1 := newTestBundle(instructorID)
	b2 := newTestBundle(instructorID)
	b2.Slug = "go-bundle-advanced"
	b2.IsPublished = true

	assert.NoError(t, s.Create(ctx, b1), "setup: Create() b1")
	assert.NoError(t, s.Create(ctx, b2), "setup: Create() b2")

	t.Run("no filter", func(t *testing.T) {
		bundles, err := s.List(ctx, store.BundleFilter{})

		assert.NoError(t, err)
		assert.Len(t, bundles, 2)
	})

	t.Run("filter by instructor", func(t *testing.T) {
		bundles, err := s.List(ctx, store.BundleFilter{InstructorID: instructorID})

		assert.NoError(t, err)
		assert.Len(t, bundles, 2)
	})

	t.Run("filter by published", func(t *testing.T) {
		published := true
		bundles, err := s.List(ctx, store.BundleFilter{IsPublished: &published})

		assert.NoError(t, err)
		assert.Len(t, bundles, 1)
		assert.Equal(t, b2.Slug, bundles[0].Slug)
	})
}

func TestBundleStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundles", "users")

	s := store.NewBundleStore(db)
	ctx := context.Background()

	b := newTestBundle(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, b), "setup: Create()")

	b.Title = "Go Bundle Updated"
	b.IsPublished = true
	assert.NoError(t, s.Update(ctx, b))

	found, err := s.FindByID(ctx, b.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Go Bundle Updated", found.Title)
	assert.Equal(t, true, found.IsPublished)
}

func TestBundleStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundles", "users")

	s := store.NewBundleStore(db)
	ctx := context.Background()

	b := newTestBundle(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, b), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, b.ID))

	found, err := s.FindByID(ctx, b.ID)
	assert.NoError(t, err)
	assert.Nil(t, found, "bundle should not be findable after Delete()")
}

// ============================================================================
// BundleCourseStore
// ============================================================================

func TestBundleCourseStore(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundle_courses", "bundles", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)

	b := newTestBundle(instructorID)
	assert.NoError(t, store.NewBundleStore(db).Create(ctx, b), "setup: Create() bundle")

	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() course")

	s := store.NewBundleCourseStore(db)
	assert.NoError(t, s.Assign(ctx, b.ID, course.ID, 10))

	assignments, err := s.List(ctx, b.ID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, course.ID, assignments[0].CourseID)
	assert.Equal(t, 10, assignments[0].SortOrder)
}

func TestBundleCourseStore_Assign_UpdatesSortOrder(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundle_courses", "bundles", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)

	b := newTestBundle(instructorID)
	assert.NoError(t, store.NewBundleStore(db).Create(ctx, b), "setup: Create() bundle")

	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() course")

	s := store.NewBundleCourseStore(db)
	assert.NoError(t, s.Assign(ctx, b.ID, course.ID, 10), "setup: Assign()")
	assert.NoError(t, s.Assign(ctx, b.ID, course.ID, 20))

	assignments, err := s.List(ctx, b.ID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, 20, assignments[0].SortOrder)
}

func TestBundleCourseStore_Unassign(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundle_courses", "bundles", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)

	b := newTestBundle(instructorID)
	assert.NoError(t, store.NewBundleStore(db).Create(ctx, b), "setup: Create() bundle")

	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() course")

	s := store.NewBundleCourseStore(db)
	assert.NoError(t, s.Assign(ctx, b.ID, course.ID, 10), "setup: Assign()")
	assert.NoError(t, s.Unassign(ctx, b.ID, course.ID))

	assignments, err := s.List(ctx, b.ID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 0)
}

func TestBundleCourseStore_List_OrderedBySortOrder(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "bundle_courses", "bundles", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)

	b := newTestBundle(instructorID)
	assert.NoError(t, store.NewBundleStore(db).Create(ctx, b), "setup: Create() bundle")

	course1 := newTestCourse(instructorID)
	course2 := newTestCourse(instructorID)
	course2.Slug = "go-advanced"

	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course1), "setup: Create() course1")
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course2), "setup: Create() course2")

	s := store.NewBundleCourseStore(db)
	assert.NoError(t, s.Assign(ctx, b.ID, course2.ID, 10))
	assert.NoError(t, s.Assign(ctx, b.ID, course1.ID, 20))

	assignments, err := s.List(ctx, b.ID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 2)
	assert.Equal(t, course2.ID, assignments[0].CourseID, "first should be sort_order=10")
	assert.Equal(t, course1.ID, assignments[1].CourseID, "second should be sort_order=20")
}
