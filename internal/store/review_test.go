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

// setupCourseForReview inserts an instructor and a course, returning the course ID.
func setupCourseForReview(t testing.TB, db *database.DB, instructorID string) string {
	t.Helper()

	c := newTestCourse(instructorID)
	c.Slug = "review-test-course"
	cs := store.NewCourseStore(db)

	assert.NoError(t, cs.Create(context.Background(), c), "setup: Create() course")
	return c.ID
}

func newTestReview(userID, courseID string) *model.Review {
	return &model.Review{
		UserID:   userID,
		CourseID: courseID,
		Rating:   4,
		Comment:  "Great course!",
	}
}

func TestReviewStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)

	r := newTestReview(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, r))
	assert.NotEqual(t, "", r.ID, "Create() did not set ID")
}

func TestReviewStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)

	r := newTestReview(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, r))

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByID(ctx, r.ID)
		assert.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
		assert.Equal(t, 4, got.Rating)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.NoError(t, err)
		assert.Equal(t, (*model.Review)(nil), got)
	})
}

func TestReviewStore_FindByUserAndCourse(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)

	r := newTestReview(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, r))

	t.Run("found", func(t *testing.T) {
		got, err := s.FindByUserAndCourse(ctx, instructorID, courseID)
		assert.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := s.FindByUserAndCourse(ctx, "00000000-0000-0000-0000-000000000000", courseID)
		assert.NoError(t, err)
		assert.Equal(t, (*model.Review)(nil), got)
	})
}

func TestReviewStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)

	r1 := newTestReview(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, r1))

	t.Run("no filter", func(t *testing.T) {
		reviews, total, err := s.List(ctx, store.ReviewFilter{})
		assert.NoError(t, err)
		assert.Len(t, reviews, 1)
		assert.Equal(t, 1, total)
	})

	t.Run("filter by course", func(t *testing.T) {
		reviews, total, err := s.List(ctx, store.ReviewFilter{CourseID: courseID})
		assert.NoError(t, err)
		assert.Len(t, reviews, 1)
		assert.Equal(t, 1, total)
	})

	t.Run("filter by user", func(t *testing.T) {
		reviews, total, err := s.List(ctx, store.ReviewFilter{UserID: instructorID})
		assert.NoError(t, err)
		assert.Len(t, reviews, 1)
		assert.Equal(t, 1, total)
	})

	t.Run("filter no match", func(t *testing.T) {
		reviews, total, err := s.List(ctx, store.ReviewFilter{CourseID: "00000000-0000-0000-0000-000000000000"})
		assert.NoError(t, err)
		assert.Len(t, reviews, 0)
		assert.Equal(t, 0, total)
	})
}

func TestReviewStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)

	r := newTestReview(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, r))

	r.Rating = 5
	r.Comment = "Even better!"
	assert.NoError(t, s.Update(ctx, r))

	got, err := s.FindByID(ctx, r.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, got.Rating)
	assert.Equal(t, "Even better!", got.Comment)
}

func TestReviewStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)

	r := newTestReview(instructorID, courseID)
	assert.NoError(t, s.Create(ctx, r))
	assert.NoError(t, s.Delete(ctx, r.ID))

	got, err := s.FindByID(ctx, r.ID)
	assert.NoError(t, err)
	assert.Equal(t, (*model.Review)(nil), got)
}

func TestReviewStore_RecalcRating(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_reviews", "courses", "users")

	ctx := context.Background()
	instructorID := setupInstructor(t, db)
	courseID := setupCourseForReview(t, db, instructorID)
	s := store.NewReviewStore(db)
	cs := store.NewCourseStore(db)

	r1 := newTestReview(instructorID, courseID)
	r1.Rating = 4
	assert.NoError(t, s.Create(ctx, r1))
	assert.NoError(t, s.RecalcRating(ctx, courseID))

	course, err := cs.FindByID(ctx, courseID)
	assert.NoError(t, err)
	assert.Equal(t, 1, course.RatingCount)
	assert.Equal(t, float64(4), course.RatingAverage)

	assert.NoError(t, s.Delete(ctx, r1.ID))
	assert.NoError(t, s.RecalcRating(ctx, courseID))

	course, err = cs.FindByID(ctx, courseID)
	assert.NoError(t, err)
	assert.Equal(t, 0, course.RatingCount)
	assert.Equal(t, float64(0), course.RatingAverage)
}
