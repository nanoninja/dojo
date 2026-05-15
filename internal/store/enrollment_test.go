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

func newTestEnrollment(userID, courseID string) *model.CourseEnrollment {
	return &model.CourseEnrollment{
		UserID:   userID,
		CourseID: courseID,
		Status:   model.EnrollmentStatusActive,
	}
}

func TestEnrollmentStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_enrollments", "courses", "users")

	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(context.Background(), course), "setup: Create() course")

	s := store.NewEnrollmentStore(db)
	e := newTestEnrollment(instructorID, course.ID)

	assert.NoError(t, s.Create(context.Background(), e))
	assert.NotEqual(t, "", e.ID, "Create() did not set ID")
}

func TestEnrollmentStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_enrollments", "courses", "users")

	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(context.Background(), course), "setup: Create() course")

	s := store.NewEnrollmentStore(db)
	ctx := context.Background()

	e := newTestEnrollment(instructorID, course.ID)
	assert.NoError(t, s.Create(ctx, e), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, e.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, e.UserID, found.UserID)
		assert.Equal(t, e.CourseID, found.CourseID)
		assert.Equal(t, model.EnrollmentStatusActive, found.Status)
	})
}

func TestEnrollmentStore_FindByUserAndCourse(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_enrollments", "courses", "users")

	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(context.Background(), course), "setup: Create() course")

	s := store.NewEnrollmentStore(db)
	ctx := context.Background()

	e := newTestEnrollment(instructorID, course.ID)
	assert.NoError(t, s.Create(ctx, e), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByUserAndCourse(ctx, instructorID, course.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, e.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByUserAndCourse(ctx, "00000000-0000-0000-0000-000000000000", course.ID)

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestEnrollmentStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_enrollments", "courses", "users")

	instructorID := setupInstructor(t, db)
	courseStore := store.NewCourseStore(db)
	ctx := context.Background()

	course1 := newTestCourse(instructorID)
	course2 := newTestCourse(instructorID)
	course2.Slug = "go-advanced"

	assert.NoError(t, courseStore.Create(ctx, course1), "setup: Create() course1")
	assert.NoError(t, courseStore.Create(ctx, course2), "setup: Create() course2")

	s := store.NewEnrollmentStore(db)

	assert.NoError(t, s.Create(ctx, newTestEnrollment(instructorID, course1.ID)))
	assert.NoError(t, s.Create(ctx, newTestEnrollment(instructorID, course2.ID)))

	t.Run("no filter", func(t *testing.T) {
		enrollments, err := s.List(ctx, store.EnrollmentFilter{})

		assert.NoError(t, err)
		assert.Len(t, enrollments, 2)
	})

	t.Run("filter by user", func(t *testing.T) {
		enrollemnts, err := s.List(ctx, store.EnrollmentFilter{
			UserID: instructorID,
		})

		assert.NoError(t, err)
		assert.Len(t, enrollemnts, 2)
	})

	t.Run("filter by course", func(t *testing.T) {
		enrollments, err := s.List(ctx, store.EnrollmentFilter{
			CourseID: course1.ID,
		})

		assert.NoError(t, err)
		assert.Len(t, enrollments, 1)
	})

	t.Run("filter by status", func(t *testing.T) {
		enrollments, err := s.List(ctx, store.EnrollmentFilter{
			Status: model.EnrollmentStatusActive,
		})

		assert.NoError(t, err)
		assert.Len(t, enrollments, 2)
	})
}

func TestEnrollmentStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_enrollments", "courses", "users")

	instructorID := setupInstructor(t, db)
	course := newTestCourse(instructorID)
	assert.NoError(t, store.NewCourseStore(db).Create(context.Background(), course), "setup: Create() course")

	s := store.NewEnrollmentStore(db)
	ctx := context.Background()

	e := newTestEnrollment(instructorID, course.ID)
	assert.NoError(t, s.Create(ctx, e), "setup: Create()")

	e.Status = model.EnrollmentStatusCompleted
	e.ProgressPercent = 100.0
	assert.NoError(t, s.Update(ctx, e))

	found, err := s.FindByID(ctx, e.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.EnrollmentStatusCompleted, found.Status)
	assert.Equal(t, 100.0, found.ProgressPercent)
}

func TestEnrollmentStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "course_enrollments", "courses", "users")

	instrutorID := setupInstructor(t, db)
	course := newTestCourse(instrutorID)
	assert.NoError(t, store.NewCourseStore(db).Create(context.Background(), course), "setup: Create() course")

	s := store.NewEnrollmentStore(db)
	ctx := context.Background()

	e := newTestEnrollment(instrutorID, course.ID)
	assert.NoError(t, s.Create(ctx, e), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, e.ID))

	found, err := s.FindByID(ctx, e.ID)
	assert.NoError(t, err)
	assert.Nil(t, found, "enrollment should not be findable after Delete()")
}
