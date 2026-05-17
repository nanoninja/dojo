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

func newTestProgress(userID, lessonID string) *model.LessonProgress {
	return &model.LessonProgress{
		UserID:         userID,
		LessonID:       lessonID,
		IsCompleted:    false,
		WatchedSeconds: 0,
	}
}

func TestLessonProgressStore_Save(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db,
		"user_lesson_progress",
		"course_enrollments",
		"lessons",
		"chapters",
	)

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() course")

	chapter := &model.Chapter{CourseID: course.ID, Title: "Ch1", Slug: "ch1"}
	assert.NoError(t, store.NewChapterStore(db).Create(ctx, chapter), "setup: Create() chapter")

	lesson := &model.Lesson{ChapterID: chapter.ID, Title: "L1", Slug: "l1", ContentType: model.ContentTypeVideo}
	assert.NoError(t, store.NewLessonStore(db).Create(ctx, lesson), "setup: Create() lesson")

	s := store.NewLessonProgressStore(db)
	p := newTestProgress(instructorID, lesson.ID)

	assert.NoError(t, s.Save(ctx, p))
}

func TestLessonProgressStore_Save_Idempotent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db,
		"user_lesson_progress",
		"course_enrollments",
		"lessons",
		"chapters",
		"courses",
		"users",
	)

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() course")

	chapter := &model.Chapter{CourseID: course.ID, Title: "Ch1", Slug: "ch1"}
	assert.NoError(t, store.NewChapterStore(db).Create(ctx, chapter), "setup: Create() chapter")

	lesson := &model.Lesson{ChapterID: chapter.ID, Title: "L1", Slug: "l1", ContentType: model.ContentTypeVideo}
	assert.NoError(t, store.NewLessonStore(db).Create(ctx, lesson), "setup: Create() lesson")

	s := store.NewLessonProgressStore(db)
	p := newTestProgress(instructorID, lesson.ID)

	assert.NoError(t, s.Save(ctx, p), "first save")

	p.IsCompleted = true
	p.WatchedSeconds = 120
	assert.NoError(t, s.Save(ctx, p), "second save (upsert)")

	found, err := s.FindByUserAndLesson(ctx, instructorID, lesson.ID)
	assert.NoError(t, err)
	assert.Equal(t, true, found.IsCompleted)
	assert.Equal(t, 120, found.WatchedSeconds)
}

func TestLessonProgressStore_FindByUserAndLesson(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db,
		"user_lesson_progress",
		"course_enrollments",
		"lessons",
		"chapters",
		"courses",
		"users",
	)

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() course")

	chapter := &model.Chapter{CourseID: course.ID, Title: "Ch1", Slug: "ch1"}
	assert.NoError(t, store.NewChapterStore(db).Create(ctx, chapter), "setup: Create() chapter")

	lesson := &model.Lesson{ChapterID: chapter.ID, Title: "L1", Slug: "l1", ContentType: model.ContentTypeVideo}
	assert.NoError(t, store.NewLessonStore(db).Create(ctx, lesson), "setup: Create() lesson")

	s := store.NewLessonProgressStore(db)
	assert.NoError(t, s.Save(ctx, newTestProgress(instructorID, lesson.ID)), "setup: Save()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByUserAndLesson(ctx, instructorID, lesson.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, instructorID, found.UserID)
		assert.Equal(t, lesson.ID, found.LessonID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByUserAndLesson(ctx, "00000000-0000-0000-0000-000000000000", lesson.ID)

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestLessonProgressStore_CalcProgressPercent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db,
		"user_lesson_progress",
		"course_enrollments",
		"lessons",
		"chapters",
		"courses",
		"users",
	)

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(ctx, course), "setup: Create() chapter")

	chapter := &model.Chapter{CourseID: course.ID, Title: "Ch1", Slug: "ch1"}
	assert.NoError(t, store.NewChapterStore(db).Create(ctx, chapter), "setup: Create() chapter")

	lessonStore := store.NewLessonStore(db)
	l1 := &model.Lesson{ChapterID: chapter.ID, Title: "L1", Slug: "l1", ContentType: model.ContentTypeVideo, IsPublished: true}
	l2 := &model.Lesson{ChapterID: chapter.ID, Title: "L2", Slug: "l2", ContentType: model.ContentTypeVideo, IsPublished: true}

	assert.NoError(t, lessonStore.Create(ctx, l1), "setup: Create() l1")
	assert.NoError(t, lessonStore.Create(ctx, l2), "setup: Create() l2")

	s := store.NewLessonProgressStore(db)

	t.Run("no progress", func(t *testing.T) {
		pct, err := s.CalcProgressPercent(ctx, instructorID, course.ID)

		assert.NoError(t, err)
		assert.Equal(t, 0.0, pct)
	})

	t.Run("one of two completed", func(t *testing.T) {
		assert.NoError(t, s.Save(ctx, &model.LessonProgress{
			UserID:      instructorID,
			LessonID:    l1.ID,
			IsCompleted: true,
		}))
		pct, err := s.CalcProgressPercent(ctx, instructorID, course.ID)

		assert.NoError(t, err)
		assert.Equal(t, 50.0, pct)
	})

	t.Run("all completed", func(t *testing.T) {
		assert.NoError(t, s.Save(ctx, &model.LessonProgress{
			UserID:      instructorID,
			LessonID:    l2.ID,
			IsCompleted: true,
		}))
		pct, err := s.CalcProgressPercent(ctx, instructorID, course.ID)

		assert.NoError(t, err)
		assert.Equal(t, 100.0, pct)
	})
}
