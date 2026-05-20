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

// ReviewFilter holds filtering options for listing course reviews.
type ReviewFilter struct {
	UserID   string
	CourseID string
	Limit    int
	SortDir  SortDir
	Offset   int
}

// ReviewStore defines persistence operations for course reviews.
type ReviewStore interface {
	// List returns reviews matching the given filter and their total count without pagination.
	List(ctx context.Context, f ReviewFilter) ([]model.Review, int, error)

	// FindByID returns a review by its ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Review, error)

	// FindByUserAndCourse returns the review left by a user on a course, or nil if not found.
	FindByUserAndCourse(ctx context.Context, userID, courseID string) (*model.Review, error)

	// Create inserts a new review and sets its ID.
	Create(ctx context.Context, r *model.Review) error

	// Update persists changes to an existing review.
	Update(ctx context.Context, r *model.Review) error

	// Delete removes a review by ID.
	Delete(ctx context.Context, id string) error

	// RecalcRating recomputes rating_average and rating_count on the course from its reviews.
	RecalcRating(ctx context.Context, courseID string) error
}

type reviewStore struct {
	db database.Querier
}

// NewReviewStore returns a ReviewStore backed by the given querier.
func NewReviewStore(db database.Querier) ReviewStore {
	return &reviewStore{db: db}
}

func (s *reviewStore) List(ctx context.Context, f ReviewFilter) ([]model.Review, int, error) {
	where := ` WHERE true`
	args := make([]any, 0, 3)

	if f.UserID != "" {
		where += ` AND user_id = ?`
		args = append(args, f.UserID)
	}
	if f.CourseID != "" {
		where += ` AND course_id = ?`
		args = append(args, f.CourseID)
	}

	var total int
	countQuery := s.db.Rebind(`SELECT COUNT(*) FROM course_reviews` + where)
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	order := SortDirDesc
	if f.SortDir == SortDirAsc {
		order = SortDirAsc
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}

	sel := `SELECT * FROM course_reviews` + where + ` ORDER BY created_at ` + string(order) + ` LIMIT ?`
	pageArgs := append(args[:len(args):len(args)], f.Limit)

	if f.Offset > 0 {
		sel += ` OFFSET ?`
		pageArgs = append(pageArgs, f.Offset)
	}

	reviews := make([]model.Review, 0, f.Limit)
	if err := s.db.SelectContext(ctx, &reviews, s.db.Rebind(sel), pageArgs...); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (s *reviewStore) FindByID(ctx context.Context, id string) (*model.Review, error) {
	var r model.Review
	err := s.db.GetContext(ctx, &r, `
		SELECT
			id, user_id, course_id, rating,
			comment, created_at, updated_at
		FROM course_reviews
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *reviewStore) FindByUserAndCourse(ctx context.Context, userID, courseID string) (*model.Review, error) {
	var r model.Review
	err := s.db.GetContext(ctx, &r, `
		SELECT
			id, user_id, course_id, rating,
			comment, created_at, updated_at
		FROM course_reviews
		WHERE user_id = $1 AND course_id = $2`,
		userID, courseID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *reviewStore) Create(ctx context.Context, r *model.Review) error {
	return s.db.GetContext(ctx, &r.ID, `
		INSERT INTO course_reviews (
			user_id,
			course_id,
			rating,
			comment
		) VALUES ($1, $2, $3, $4)
		RETURNING id`,
		r.UserID,
		r.CourseID,
		r.Rating,
		r.Comment,
	)
}

func (s *reviewStore) Update(ctx context.Context, r *model.Review) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE course_reviews
		SET
			rating  = $1,
			comment = $2
		WHERE id = $3`,
		r.Rating,
		r.Comment,
		r.ID,
	)
	return err
}

func (s *reviewStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM course_reviews WHERE id = $1`, id)
	return err
}

func (s *reviewStore) RecalcRating(ctx context.Context, courseID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE courses
		SET
			rating_average = (
				SELECT COALESCE(AVG(rating), 0) FROM course_reviews WHERE course_id = $1
			),
			rating_count = (
				SELECT COUNT(*) FROM course_reviews WHERE course_id = $1
			)
		WHERE id = $1`,
		courseID,
	)
	return err
}
