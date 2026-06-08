// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"errors"

	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// AccessService checks whether a user is allowed to consume course content.
type AccessService interface {
	// CanAccess returns nil if the user has access to the given course, or
	// fault.Forbidden if they do not. Access is granted when the user has an
	// active subscription or an active enrollment on that specific course.
	CanAccess(ctx context.Context, userID, courseID string) error
}

type accessService struct {
	subscriptions store.SubscriptionStore
	enrollments   store.EnrollmentStore
}

// NewAccessService returns an AccessService backed by the given stores.
func NewAccessService(subscriptions store.SubscriptionStore, enrollments store.EnrollmentStore) AccessService {
	return &accessService{subscriptions: subscriptions, enrollments: enrollments}
}

func (s *accessService) CanAccess(ctx context.Context, userID, courseID string) error {
	sub, err := s.subscriptions.FindActiveByUser(ctx, userID)
	if err != nil {
		return fault.Internal(err)
	}
	if sub != nil {
		return nil
	}
	enroll, err := s.enrollments.FindByUserAndCourse(ctx, userID, courseID)
	if err != nil {
		return fault.Internal(err)
	}
	if enroll != nil && enroll.Status == model.EnrollmentStatusActive {
		return nil
	}
	return fault.Forbidden(errors.New("no active subscription or enrollment"))
}
