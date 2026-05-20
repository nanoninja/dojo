// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

// ReviewService handles course review management.
type ReviewService interface {
	// List returns reviews matching the given filter and their total count without pagination.
	List(ctx context.Context, f store.ReviewFilter) ([]model.Review, int, error)

	// GetByID returns a review by ID, or ErrReviewNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Review, error)

	// GetByUserAndCourse returns a user's review for a course, or ErrReviewNotFound if not found.
	GetByUserAndCourse(ctx context.Context, userID, courseID string) (*model.Review, error)

	// Create inserts a new review and recalculates the course rating atomically.
	Create(ctx context.Context, r *model.Review) error

	// Update saves changes to an existing review and recalculates the course rating atomically.
	Update(ctx context.Context, r *model.Review) error

	// Delete removes a review and recalculates the course rating atomically.
	Delete(ctx context.Context, id string) error
}

type reviewService struct {
	db      database.TxRunner
	reviews store.ReviewStore
}

// NewReviewService creates a ReviewService backed by the given store.
func NewReviewService(db database.TxRunner, reviews store.ReviewStore) ReviewService {
	return &reviewService{db: db, reviews: reviews}
}

func (s *reviewService) List(ctx context.Context, f store.ReviewFilter) ([]model.Review, int, error) {
	return s.reviews.List(ctx, f)
}

func (s *reviewService) GetByID(ctx context.Context, id string) (*model.Review, error) {
	r, err := s.reviews.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReviewNotFound
	}
	return r, nil
}

func (s *reviewService) GetByUserAndCourse(ctx context.Context, userID, courseID string) (*model.Review, error) {
	r, err := s.reviews.FindByUserAndCourse(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReviewNotFound
	}
	return r, nil
}

func (s *reviewService) Create(ctx context.Context, r *model.Review) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		rs := store.NewReviewStore(q)
		if err := rs.Create(ctx, r); err != nil {
			return err
		}
		return rs.RecalcRating(ctx, r.CourseID)
	})
}

func (s *reviewService) Update(ctx context.Context, r *model.Review) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		rs := store.NewReviewStore(q)
		if err := rs.Update(ctx, r); err != nil {
			return err
		}
		return rs.RecalcRating(ctx, r.CourseID)
	})
}

func (s *reviewService) Delete(ctx context.Context, id string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		rs := store.NewReviewStore(q)
		r, err := rs.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if r == nil {
			return ErrReviewNotFound
		}
		if err := rs.Delete(ctx, id); err != nil {
			return err
		}
		return rs.RecalcRating(ctx, r.CourseID)
	})
}
