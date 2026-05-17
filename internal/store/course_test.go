// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

func newTestCourse(instructorID string) *model.Course {
	return &model.Course{
		InstructorID: instructorID,
		Slug:         "go-fundamentals",
		Title:        "Go Fundamentals",
		Level:        model.CourseLevelBeginner,
		ContentType:  model.ContentTypeVideo,
		Language:     "en",
		Currency:     "USD",
	}
}

func setupInstructor(t testing.TB, db *database.DB) string {
	t.Helper()
	u := newTestUser()
	u.Email = "course-instructor@example.com"
	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	assert.NoError(t, us.Create(context.Background(), u), "setup: Create() instructor")
	return u.ID
}

func TestCourseStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses", "users")

	s := store.NewCourseStore(db)
	c := newTestCourse(setupInstructor(t, db))

	assert.NoError(t, s.Create(context.Background(), c))
	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
}

func TestCourseStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses", "users")

	ctx := context.Background()
	s := store.NewCourseStore(db)

	c := newTestCourse(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByID(ctx, c.ID)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestCourseStore_FindBySlug(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses", "users")

	ctx := context.Background()
	s := store.NewCourseStore(db)

	c := newTestCourse(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := s.FindBySlug(ctx, "go-fundamentals")
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindBySlug(ctx, "non-existent")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestCourseStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses", "users")

	ctx := context.Background()
	s := store.NewCourseStore(db)
	instructorID := setupInstructor(t, db)

	for _, slug := range []string{"course-a", "course-b", "course-c"} {
		c := newTestCourse(instructorID)
		c.Slug = slug
		assert.NoError(t, s.Create(ctx, c))
	}

	courses, total, err := s.List(ctx, store.CourseFilter{Limit: 10})
	assert.NoError(t, err)
	assert.Len(t, courses, 3)
	assert.Equal(t, 3, total)
}

func TestCourseStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses", "users")

	ctx := context.Background()
	s := store.NewCourseStore(db)

	c := newTestCourse(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, c))

	c.Title = "Go Fundamentals v2"
	assert.NoError(t, s.Update(ctx, c))

	got, err := s.FindByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Go Fundamentals v2", got.Title)
}

func TestCourseStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses", "users")

	ctx := context.Background()
	s := store.NewCourseStore(db)

	c := newTestCourse(setupInstructor(t, db))
	assert.NoError(t, s.Create(ctx, c))
	assert.NoError(t, s.Delete(ctx, c.ID))

	got, err := s.FindByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Nil(t, got, "Delete() should soft-delete the course")
}

func TestCoursesCategoriesStore_AssignAndList(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses_categories", "courses", "categories", "users")

	ctx := context.Background()
	courseID := setupCourse(t, db)

	catStore := store.NewCategoryStore(db)
	cat := &model.Category{Slug: "go", Name: "Go"}
	assert.NoError(t, catStore.Create(ctx, cat))

	ccs := store.NewCoursesCategoriesStore(db)
	assert.NoError(t, ccs.Assign(ctx, courseID, cat.ID, true))

	assignments, err := ccs.List(ctx, courseID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, cat.ID, assignments[0].CategoryID)
	assert.True(t, assignments[0].IsPrimary)
}

func TestCoursesCategoriesStore_Unassign(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses_categories", "courses", "categories", "users")

	ctx := context.Background()
	courseID := setupCourse(t, db)

	catStore := store.NewCategoryStore(db)
	cat := &model.Category{Slug: "go", Name: "Go"}
	assert.NoError(t, catStore.Create(ctx, cat))

	ccs := store.NewCoursesCategoriesStore(db)
	assert.NoError(t, ccs.Assign(ctx, courseID, cat.ID, false))
	assert.NoError(t, ccs.Unassign(ctx, courseID, cat.ID))

	assignments, err := ccs.List(ctx, courseID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 0)
}

func TestCoursesCategoriesStore_SetPrimary(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses_categories", "courses", "categories", "users")

	ctx := context.Background()
	courseID := setupCourse(t, db)

	catStore := store.NewCategoryStore(db)
	cat1 := &model.Category{Slug: "go", Name: "Go"}
	cat2 := &model.Category{Slug: "backend", Name: "Backend"}
	assert.NoError(t, catStore.Create(ctx, cat1))
	assert.NoError(t, catStore.Create(ctx, cat2))

	ccs := store.NewCoursesCategoriesStore(db)
	assert.NoError(t, ccs.Assign(ctx, courseID, cat1.ID, true))
	assert.NoError(t, ccs.Assign(ctx, courseID, cat2.ID, false))
	assert.NoError(t, ccs.SetPrimary(ctx, courseID, cat2.ID))

	assignments, err := ccs.List(ctx, courseID)
	assert.NoError(t, err)
	for _, a := range assignments {
		if a.CategoryID == cat2.ID {
			assert.True(t, a.IsPrimary)
		} else {
			assert.False(t, a.IsPrimary)
		}
	}
}

func TestCoursesTagsStore_AssignAndList(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses_tags", "courses", "tags", "users")

	ctx := context.Background()
	courseID := setupCourse(t, db)

	tagStore := store.NewTagStore(db)
	tag := &model.Tag{Slug: "golang", Name: "Golang"}
	assert.NoError(t, tagStore.Create(ctx, tag))

	cts := store.NewCoursesTagsStore(db)
	assert.NoError(t, cts.Assign(ctx, courseID, tag.ID))

	assignments, err := cts.List(ctx, courseID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, tag.ID, assignments[0].TagID)
}

func TestCoursesTagsStore_Unassign(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "courses_tags", "courses", "tags", "users")

	ctx := context.Background()
	courseID := setupCourse(t, db)

	tagStore := store.NewTagStore(db)
	tag := &model.Tag{Slug: "golang", Name: "Golang"}
	assert.NoError(t, tagStore.Create(ctx, tag))

	cts := store.NewCoursesTagsStore(db)
	assert.NoError(t, cts.Assign(ctx, courseID, tag.ID))
	assert.NoError(t, cts.Unassign(ctx, courseID, tag.ID))

	assignments, err := cts.List(ctx, courseID)
	assert.NoError(t, err)
	assert.Len(t, assignments, 0)
}
