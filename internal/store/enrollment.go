// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
)

// EnrollmentFilter holds filtering options for listing enrollments.
type EnrollmentFilter struct {
	UserID   string
	CourseID string
	Status   model.EnrollmentStatus
	Limit    int
	Offset   int
}

// EnrollmentStore defines persistence operations for course enrollments.
type EnrollmentStore interface {
	// List returns enrollments matching the given filter.
	List(ctx context.Context, f EnrollmentFilter) ([]model.CourseEnrollment, error)

	// FindByID returns an enrollment by its ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.CourseEnrollment, error)

	// FindByUserAndCourse returns an enrollment for a specific user/course pair, or nil if not found.
	FindByUserAndCourse(ctx context.Context, userID, courseID string) (*model.CourseEnrollment, error)

	// Create inserts a new enrollment and sets its ID.
	Create(ctx context.Context, e *model.CourseEnrollment) error

	// Update persists status and progress changes to an existing enrollment.
	Update(ctx context.Context, e *model.CourseEnrollment) error

	// Delete removes an enrollment permanently.
	Delete(ctx context.Context, id string) error
}

type enrollmentStore struct {
	db database.Querier
}

// NewEnrollmentStore returns an EnrollmentStore backed by the given querier.
func NewEnrollmentStore(db database.Querier) EnrollmentStore {
	return &enrollmentStore{db: db}
}

func (s *enrollmentStore) List(ctx context.Context, f EnrollmentFilter) ([]model.CourseEnrollment, error) {
	query := `SELECT * FROM course_enrollments WHERE true`
	args := make([]any, 0, 4)

	if f.UserID != "" {
		query += ` AND user_id = ?`
		args = append(args, f.UserID)
	}
	if f.CourseID != "" {
		query += ` AND course_id = ?`
		args = append(args, f.CourseID)
	}
	if f.Status != "" {
		query += ` AND status = ?`
		args = append(args, f.Status)
	}

	query += ` ORDER BY enrolled_at DESC`

	if f.Limit <= 0 {
		f.Limit = 100
	}

	query += ` LIMIT ?`
	args = append(args, f.Limit)

	if f.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, f.Offset)
	}

	query = s.db.Rebind(query)
	enrollments := make([]model.CourseEnrollment, 0, f.Limit)

	if err := s.db.SelectContext(ctx, &enrollments, query, args...); err != nil {
		return nil, err
	}

	return enrollments, nil
}

func (s *enrollmentStore) FindByID(ctx context.Context, id string) (*model.CourseEnrollment, error) {
	var e model.CourseEnrollment
	err := s.db.GetContext(ctx, &e, `
		SELECT * FROM course_enrollments WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (s *enrollmentStore) FindByUserAndCourse(ctx context.Context, userID, courseID string) (*model.CourseEnrollment, error) {
	var e model.CourseEnrollment
	err := s.db.GetContext(ctx, &e, `
		SELECT * FROM course_enrollments 
		WHERE user_id = $1 AND course_id = $2`,
		userID,
		courseID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (s *enrollmentStore) Create(ctx context.Context, e *model.CourseEnrollment) error {
	return s.db.GetContext(ctx, &e.ID, `
		INSERT INTO course_enrollments (
			user_id,
			course_id,
			status,
			expires_at
		) VALUES ($1, $2, $3, $4)
		RETURNING id`,
		e.UserID,
		e.CourseID,
		e.Status,
		e.ExpiresAt,
	)
}

func (s *enrollmentStore) Update(ctx context.Context, e *model.CourseEnrollment) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE course_enrollments
		SET
			status           = $1,
			progress_percent = $2,
			last_accessed_at = $3,
			completed_at     = $4,
			expires_at       = $5
		WHERE id = $6`,
		e.Status,
		e.ProgressPercent,
		e.LastAccessedAt,
		e.CompletedAt,
		e.ExpiresAt,
		e.ID,
	)
	return err
}

func (s *enrollmentStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM course_enrollments WHERE id = $1`, id)
	return err
}
