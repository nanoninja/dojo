// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

func newEnrollmentService(enrollments store.EnrollmentStore) service.EnrollmentService {
	return service.NewEnrollmentService(enrollments)
}

func TestEnrollmentService_Enroll(t *testing.T) {
	ctx := context.Background()
	svc := newEnrollmentService(newFakeEnrollmentStore())

	t.Run("success", func(t *testing.T) {
		e, err := svc.Enroll(ctx, "user-1", "course-1")

		assert.NoError(t, err)
		assert.NotEqual(t, "", e.ID)
		assert.Equal(t, model.EnrollmentStatusActive, e.Status)
	})

	t.Run("already enrolled", func(t *testing.T) {
		_, err := svc.Enroll(ctx, "user-1", "course-1")

		assert.ErrorIs(t, err, service.ErrAlreadyEnrolled)
	})
}

func TestEnrollmentService_GetByID(t *testing.T) {
	ctx := context.Background()
	svc := newEnrollmentService(newFakeEnrollmentStore())

	e, err := svc.Enroll(ctx, "user-1", "course-1")
	assert.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, e.ID)
		assert.NoError(t, err)
		assert.Equal(t, e.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existant")
		assert.ErrorIs(t, err, service.ErrEnrollmentNotFound)
	})
}

func TestEnrollmentService_List(t *testing.T) {
	ctx := context.Background()
	svc := newEnrollmentService(newFakeEnrollmentStore())

	assert.NoError(t, func() error { _, err := svc.Enroll(ctx, "user-1", "course-1"); return err }())
	assert.NoError(t, func() error { _, err := svc.Enroll(ctx, "user-1", "course-2"); return err }())
	assert.NoError(t, func() error { _, err := svc.Enroll(ctx, "user-2", "course-1"); return err }())

	t.Run("no filter", func(t *testing.T) {
		result, err := svc.List(ctx, store.EnrollmentFilter{})

		assert.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("filter by user", func(t *testing.T) {
		result, err := svc.List(ctx, store.EnrollmentFilter{UserID: "user-1"})

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("filter by course", func(t *testing.T) {
		result, err := svc.List(ctx, store.EnrollmentFilter{CourseID: "course-1"})

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestEnrollmentService_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	svc := newEnrollmentService(newFakeEnrollmentStore())

	e, err := svc.Enroll(ctx, "user-1", "course-1")
	assert.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		assert.NoError(t, svc.UpdateStatus(ctx, e.ID, model.EnrollmentStatusCompleted))

		got, err := svc.GetByID(ctx, e.ID)
		assert.NoError(t, err)
		assert.Equal(t, model.EnrollmentStatusCompleted, got.Status)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.UpdateStatus(ctx, "non-existent", model.EnrollmentStatusCompleted)
		assert.ErrorIs(t, err, service.ErrEnrollmentNotFound)
	})
}

func TestEnrollmentService_Delete(t *testing.T) {
	ctx := context.Background()
	svc := newEnrollmentService(newFakeEnrollmentStore())

	e, err := svc.Enroll(ctx, "user-1", "course-1")
	assert.NoError(t, err)

	assert.NoError(t, svc.Delete(ctx, e.ID))

	_, err = svc.GetByID(ctx, e.ID)
	assert.ErrorIs(t, err, service.ErrEnrollmentNotFound)
}
