// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"errors"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// ErrEnrollmentNotFound is returned when a course enrollment cannot be found.
var ErrEnrollmentNotFound = errors.New("enrollment not found")

// ErrAlreadyEnrolled is returned when a user is already enrolled in a course.
var ErrAlreadyEnrolled = errors.New("user already enrolled in this course")

// EnrollmentService defines business operations for course enrollments.
type EnrollmentService interface {
	// List returns enrollments matching the given filter.
	List(ctx context.Context, f store.EnrollmentFilter) ([]model.CourseEnrollment, error)

	// GetByID returns an enrollment by its ID, or ErrEnrollmentNotFound.
	GetByID(ctx context.Context, id string) (*model.CourseEnrollment, error)

	// Enroll registers a user to a course, or returns ErrAlreadyEnrolled.
	Enroll(ctx context.Context, userID, courseID string) (*model.CourseEnrollment, error)

	// UpdateStatus changes the status of an existing enrollment.
	UpdateStatus(ctx context.Context, id string, status model.EnrollmentStatus) error

	// Delete removes an enrollment permanently.
	Delete(ctx context.Context, id string) error
}

type enrollmentService struct {
	enrollments store.EnrollmentStore
}

// NewEnrollmentService returns an EnrollmentService backed by the given store.
func NewEnrollmentService(enrollments store.EnrollmentStore) EnrollmentService {
	return &enrollmentService{enrollments: enrollments}
}

func (s *enrollmentService) List(ctx context.Context, f store.EnrollmentFilter) ([]model.CourseEnrollment, error) {
	return s.enrollments.List(ctx, f)
}

func (s *enrollmentService) GetByID(ctx context.Context, id string) (*model.CourseEnrollment, error) {
	e, err := s.enrollments.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, ErrEnrollmentNotFound
	}
	return e, nil
}

func (s *enrollmentService) Enroll(ctx context.Context, userID, courseID string) (*model.CourseEnrollment, error) {
	existing, err := s.enrollments.FindByUserAndCourse(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyEnrolled
	}
	e := &model.CourseEnrollment{
		UserID:   userID,
		CourseID: courseID,
		Status:   model.EnrollmentStatusActive,
	}
	if err := s.enrollments.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *enrollmentService) UpdateStatus(ctx context.Context, id string, status model.EnrollmentStatus) error {
	e, err := s.enrollments.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e == nil {
		return ErrEnrollmentNotFound
	}
	e.Status = status
	return s.enrollments.Update(ctx, e)
}

func (s *enrollmentService) Delete(ctx context.Context, id string) error {
	return s.enrollments.Delete(ctx, id)
}
