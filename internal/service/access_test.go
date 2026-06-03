// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

type fakeAccessSubscriptionStore struct {
	sub *model.Subscription
	err error
}

func (s *fakeAccessSubscriptionStore) FindActiveByUser(_ context.Context, _ string) (*model.Subscription, error) {
	return s.sub, s.err
}

func (*fakeAccessSubscriptionStore) ListByUser(_ context.Context, _ string) ([]model.Subscription, error) {
	return nil, nil
}

func (*fakeAccessSubscriptionStore) Create(_ context.Context, _ *model.Subscription) error {
	return nil
}

func (*fakeAccessSubscriptionStore) Cancel(_ context.Context, _ string) error {
	return nil
}

type fakeAccessEnrollmentStore struct {
	enroll *model.CourseEnrollment
	err    error
}

func (s *fakeAccessEnrollmentStore) FindByUserAndCourse(_ context.Context, _, _ string) (*model.CourseEnrollment, error) {
	return s.enroll, s.err
}

func (*fakeAccessEnrollmentStore) List(_ context.Context, _ store.EnrollmentFilter) ([]model.CourseEnrollment, int, error) {
	return nil, 0, nil
}

func (*fakeAccessEnrollmentStore) FindByID(_ context.Context, _ string) (*model.CourseEnrollment, error) {
	return nil, nil
}

func (*fakeAccessEnrollmentStore) Create(_ context.Context, _ *model.CourseEnrollment) error {
	return nil
}

func (*fakeAccessEnrollmentStore) Update(_ context.Context, _ *model.CourseEnrollment) error {
	return nil
}

func (*fakeAccessEnrollmentStore) UpdateProgress(_ context.Context, _, _ string, _ float64) error {
	return nil
}

func (*fakeAccessEnrollmentStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (*fakeAccessEnrollmentStore) CancelByPurchase(_ context.Context, _ string) error {
	return nil
}

func TestAccessService_CanAccess(t *testing.T) {
	ctx := context.Background()
	const userID = "user-1"
	const courseID = "course-1"

	t.Run("active subscription grants access", func(t *testing.T) {
		svc := service.NewAccessService(
			&fakeAccessSubscriptionStore{sub: &model.Subscription{Status: model.SubscriptionStatusActive}},
			&fakeAccessEnrollmentStore{},
		)
		assert.NoError(t, svc.CanAccess(ctx, userID, courseID))
	})

	t.Run("active enrollment grants access", func(t *testing.T) {
		svc := service.NewAccessService(
			&fakeAccessSubscriptionStore{},
			&fakeAccessEnrollmentStore{enroll: &model.CourseEnrollment{Status: model.EnrollmentStatusActive}},
		)
		assert.NoError(t, svc.CanAccess(ctx, userID, courseID))
	})

	t.Run("no subscription no enrollment returns forbidden", func(t *testing.T) {
		svc := service.NewAccessService(
			&fakeAccessSubscriptionStore{},
			&fakeAccessEnrollmentStore{},
		)
		assert.Error(t, svc.CanAccess(ctx, userID, courseID))
	})

	t.Run("cancelled enrollment returns forbidden", func(t *testing.T) {
		svc := service.NewAccessService(
			&fakeAccessSubscriptionStore{},
			&fakeAccessEnrollmentStore{enroll: &model.CourseEnrollment{Status: model.EnrollmentStatusCancelled}},
		)
		assert.Error(t, svc.CanAccess(ctx, userID, courseID))
	})

	t.Run("subcription store error returns error", func(t *testing.T) {
		svc := service.NewAccessService(
			&fakeAccessSubscriptionStore{err: errors.New("db error")},
			&fakeAccessEnrollmentStore{},
		)
		assert.Error(t, svc.CanAccess(ctx, userID, courseID))
	})

	t.Run("enrollment store error returns error", func(t *testing.T) {
		svc := service.NewAccessService(
			&fakeAccessSubscriptionStore{},
			&fakeAccessEnrollmentStore{err: errors.New("db error")},
		)
		assert.Error(t, svc.CanAccess(ctx, userID, courseID))
	})
}
