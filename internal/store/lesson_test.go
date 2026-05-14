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

func setupChapter(t testing.TB, db *database.DB, courseID string) string {
	t.Helper()

	s := store.NewChapterStore(db)
	c := &model.Chapter{
		CourseID:  courseID,
		Title:     "Introduction",
		Slug:      "introduction",
		SortOrder: 1,
	}
	assert.NoError(t, s.Create(context.Background(), c), "setup: Create() chapter")
	return c.ID
}

func newTestLesson(chapterID string) *model.Lesson {
	return &model.Lesson{
		ChapterID:   chapterID,
		Title:       "Variables",
		Slug:        "variables",
		ContentType: model.ContentTypeVideo,
		SortOrder:   1,
	}
}

func TestLessonStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lessons", "chapters", "courses", "users")

	chapterID := setupChapter(t, db, setupCourse(t, db))
	s := store.NewLessonStore(db)
	l := newTestLesson(chapterID)

	assert.NoError(t, s.Create(context.Background(), l))
	assert.NotEqual(t, "", l.ID, "Create() did not set ID")
}

func TestLessonStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lessons", "chapters", "courses", "users")

	s := store.NewLessonStore(db)
	ctx := context.Background()

	l := newTestLesson(setupChapter(t, db, setupCourse(t, db)))
	assert.NoError(t, s.Create(ctx, l), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, l.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, l.Title, found.Title)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestLessonStore_FindBySlug(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lessons", "chapters", "courses", "users")

	s := store.NewLessonStore(db)
	ctx := context.Background()

	chapterID := setupChapter(t, db, setupCourse(t, db))
	l := newTestLesson(chapterID)
	assert.NoError(t, s.Create(ctx, l), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, chapterID, "variables")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, l.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, chapterID, "unknown")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestLessonStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lessons", "chapters", "courses", "users")

	s := store.NewLessonStore(db)
	ctx := context.Background()
	chapterID := setupChapter(t, db, setupCourse(t, db))

	for i, slug := range []string{"variables", "functions", "structs"} {
		assert.NoError(t, s.Create(ctx, &model.Lesson{
			ChapterID:   chapterID,
			Title:       slug,
			Slug:        slug,
			ContentType: model.ContentTypeVideo,
			SortOrder:   int16(i + 1),
		}), "setup: Create()")
	}

	t.Run("returns lessons for chapter", func(t *testing.T) {
		lessons, err := s.List(ctx, chapterID)
		assert.NoError(t, err)
		assert.Len(t, lessons, 3)
	})

	t.Run("empty for unknown chapter", func(t *testing.T) {
		lessons, err := s.List(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Len(t, lessons, 0)
	})
}

func TestLessonStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lessons", "chapters", "courses", "users")

	s := store.NewLessonStore(db)
	ctx := context.Background()

	l := newTestLesson(setupChapter(t, db, setupCourse(t, db)))
	assert.NoError(t, s.Create(ctx, l), "setup: Create()")

	l.Title = "Variables & Types"
	assert.NoError(t, s.Update(ctx, l))

	found, err := s.FindByID(ctx, l.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Variables & Types", found.Title)
}

func TestLessonStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lessons", "chapters", "courses", "users")

	s := store.NewLessonStore(db)
	ctx := context.Background()

	l := newTestLesson(setupChapter(t, db, setupCourse(t, db)))
	assert.NoError(t, s.Create(ctx, l), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, l.ID))

	found, err := s.FindByID(ctx, l.ID)
	assert.NoError(t, err)
	assert.Nil(t, found, "lesson should not be findable after Delete()")
}

func newTestLessonResource(lessonID string) *model.LessonResource {
	return &model.LessonResource{
		LessonID:  lessonID,
		Title:     "Slides",
		FileURL:   "https://example.com/slides.pdf",
		FileName:  "slides.pdf",
		SortOrder: 1,
	}
}

func setupLesson(t testing.TB, db *database.DB, chapterID string) string {
	t.Helper()
	s := store.NewLessonStore(db)
	l := newTestLesson(chapterID)
	assert.NoError(t, s.Create(context.Background(), l), "setup: Create() lesson")
	return l.ID
}

func TestLessonResourceStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lesson_resources", "lessons", "chapters", "courses", "users")

	lessonID := setupLesson(t, db, setupChapter(t, db, setupCourse(t, db)))
	s := store.NewLessonResourceStore(db)
	r := newTestLessonResource(lessonID)

	assert.NoError(t, s.Create(context.Background(), r))
	assert.NotEqual(t, "", r.ID, "Create() did not set ID")
}

func TestLessonResourceStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lesson_resources", "lessons", "chapters", "courses", "users")

	ctx := context.Background()
	lessonID := setupLesson(t, db, setupChapter(t, db, setupCourse(t, db)))
	s := store.NewLessonResourceStore(db)

	r := newTestLessonResource(lessonID)
	assert.NoError(t, s.Create(ctx, r))

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByID(ctx, r.ID)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, r.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestLessonResourceStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lesson_resources", "lessons", "chapters", "courses", "users")

	ctx := context.Background()
	lessonID := setupLesson(t, db, setupChapter(t, db, setupCourse(t, db)))
	s := store.NewLessonResourceStore(db)

	for i, name := range []string{"Slides", "Exercises", "Cheatsheet"} {
		r := newTestLessonResource(lessonID)
		r.Title = name
		r.SortOrder = int16(i + 1)
		assert.NoError(t, s.Create(ctx, r))
	}

	resources, err := s.List(ctx, lessonID)
	assert.NoError(t, err)
	assert.Len(t, resources, 3)
}

func TestLessonResourceStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lesson_resources", "lessons", "chapters", "courses", "users")

	ctx := context.Background()
	lessonID := setupLesson(t, db, setupChapter(t, db, setupCourse(t, db)))
	s := store.NewLessonResourceStore(db)

	r := newTestLessonResource(lessonID)
	assert.NoError(t, s.Create(ctx, r))

	r.Title = "Updated Slides"
	assert.NoError(t, s.Update(ctx, r))

	got, err := s.FindByID(ctx, r.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Slides", got.Title)
}

func TestLessonResourceStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "lesson_resources", "lessons", "chapters", "courses", "users")

	ctx := context.Background()
	lessonID := setupLesson(t, db, setupChapter(t, db, setupCourse(t, db)))
	s := store.NewLessonResourceStore(db)

	r := newTestLessonResource(lessonID)
	assert.NoError(t, s.Create(ctx, r))
	assert.NoError(t, s.Delete(ctx, r.ID))

	got, err := s.FindByID(ctx, r.ID)
	assert.NoError(t, err)
	assert.Nil(t, got, "Delete() should remove the resource")
}
