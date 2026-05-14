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

func setupCourse(t testing.TB, db *database.DB) string {
	t.Helper()
	ctx := context.Background()

	u := newTestUser()
	u.Email = "insrtuctor@example.com"
	us := store.NewUserStore(db, testutil.NewTestCipher(t))
	assert.NoError(t, us.Create(ctx, u), "setup: Create() user")

	c := &model.Course{
		InstructorID: u.ID,
		Slug:         "go-fundamentals",
		Title:        "Go Fundamentals",
		Level:        model.CourseLevelBeginner,
		ContentType:  model.ContentTypeVideo,
		Language:     "en",
		Currency:     "USD",
	}

	cs := store.NewCourseStore(db)
	assert.NoError(t, cs.Create(ctx, c), "setup: Create() course")
	return c.ID
}

func newTestChapter(courseID string) *model.Chapter {
	return &model.Chapter{
		CourseID:  courseID,
		Title:     "Introduction",
		Slug:      "introduction",
		SortOrder: 1,
	}
}

func TestChapterStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "chapters", "courses", "users")

	courseID := setupCourse(t, db)
	s := store.NewChapterStore(db)
	c := newTestChapter(courseID)

	assert.NoError(t, s.Create(context.Background(), c))
	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
}

func TestChapterStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "chapters", "courses", "users")

	s := store.NewChapterStore(db)
	ctx := context.Background()

	c := newTestChapter(setupCourse(t, db))
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, c.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, c.Title, found.Title)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestChapterStore_FindBySlug(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "chapters", "courses", "users")

	s := store.NewChapterStore(db)
	ctx := context.Background()

	courseID := setupCourse(t, db)
	c := newTestChapter(courseID)
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, courseID, "introduction")

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, c.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, courseID, "unknown")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestChapterStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "chapters", "courses", "users")

	s := store.NewChapterStore(db)
	ctx := context.Background()
	courseID := setupCourse(t, db)

	for i, slug := range []string{"intro", "basics", "advanced"} {
		assert.NoError(t, s.Create(ctx, &model.Chapter{
			CourseID:  courseID,
			Title:     slug,
			Slug:      slug,
			SortOrder: int16(i + 1),
		}), "setup: Create()")
	}

	t.Run("returns chapters for course", func(t *testing.T) {
		chapters, err := s.List(ctx, courseID)

		assert.NoError(t, err)
		assert.Len(t, chapters, 3)
	})

	t.Run("empty for unknown course", func(t *testing.T) {
		chapters, err := s.List(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Len(t, chapters, 0)
	})
}

func TestChapterStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "chapters", "courses", "users")

	s := store.NewChapterStore(db)
	ctx := context.Background()

	c := newTestChapter(setupCourse(t, db))
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	c.Title = "Introduction Updated"
	assert.NoError(t, s.Update(ctx, c))

	found, err := s.FindByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Introduction Updated", found.Title)
}

func TestChapterStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "chapters", "courses", "users")

	s := store.NewChapterStore(db)
	ctx := context.Background()

	c := newTestChapter(setupCourse(t, db))
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, c.ID))

	found, err := s.FindByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Nil(t, found, "chapter should not be findable after Delete()")
}
