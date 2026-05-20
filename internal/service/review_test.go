// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// fakeReviewStore
// ============================================================================

type fakeReviewStore struct {
	reviews map[string]*model.Review
	seq     int
}

func newFakeReviewStore() *fakeReviewStore {
	return &fakeReviewStore{reviews: make(map[string]*model.Review)}
}

func (s *fakeReviewStore) nextID() string {
	s.seq++
	return fmt.Sprintf("review-%d", s.seq)
}

func (s *fakeReviewStore) List(_ context.Context, _ store.ReviewFilter) ([]model.Review, int, error) {
	result := make([]model.Review, 0, len(s.reviews))
	for _, r := range s.reviews {
		result = append(result, *r)
	}
	return result, len(result), nil
}

func (s *fakeReviewStore) FindByID(_ context.Context, id string) (*model.Review, error) {
	r, ok := s.reviews[id]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (s *fakeReviewStore) FindByUserAndCourse(_ context.Context, userID, courseID string) (*model.Review, error) {
	for _, r := range s.reviews {
		if r.UserID == userID && r.CourseID == courseID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *fakeReviewStore) Create(_ context.Context, r *model.Review) error {
	r.ID = s.nextID()
	cp := *r
	s.reviews[r.ID] = &cp
	return nil
}

func (s *fakeReviewStore) Update(_ context.Context, r *model.Review) error {
	if _, ok := s.reviews[r.ID]; !ok {
		return fmt.Errorf("review not found")
	}
	cp := *r
	s.reviews[r.ID] = &cp
	return nil
}

func (s *fakeReviewStore) Delete(_ context.Context, id string) error {
	if _, ok := s.reviews[id]; !ok {
		return fmt.Errorf("review not found")
	}
	delete(s.reviews, id)
	return nil
}

func (s *fakeReviewStore) RecalcRating(_ context.Context, _ string) error {
	return nil
}

// ============================================================================
// helpers
// ============================================================================

func newReviewService(rs *fakeReviewStore) service.ReviewService {
	return service.NewReviewService(&fakeTxRunner{}, rs)
}

func baseReview() *model.Review {
	return &model.Review{
		UserID:   "user-1",
		CourseID: "course-1",
		Rating:   4,
		Comment:  "Great course!",
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestReviewService_GetByID(t *testing.T) {
	ctx := context.Background()
	rs := newFakeReviewStore()
	svc := newReviewService(rs)

	r := baseReview()
	assert.NoError(t, rs.Create(ctx, r))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, r.ID)

		assert.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
	})
}

func TestReviewService_GetByUserAndCourse(t *testing.T) {
	ctx := context.Background()
	rs := newFakeReviewStore()
	svc := newReviewService(rs)

	r := baseReview()
	assert.NoError(t, rs.Create(ctx, r))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByUserAndCourse(ctx, "user-1", "course-1")

		assert.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByUserAndCourse(ctx, "user-2", "course-1")
		assert.ErrorIs(t, err, service.ErrReviewNotFound)
	})
}

func TestReviewService_List(t *testing.T) {
	ctx := context.Background()
	rs := newFakeReviewStore()
	svc := newReviewService(rs)

	assert.NoError(t, rs.Create(ctx, baseReview()))
	r2 := baseReview()
	r2.UserID = "user-2"
	assert.NoError(t, rs.Create(ctx, r2))

	reviews, total, err := svc.List(ctx, store.ReviewFilter{})

	assert.NoError(t, err)
	assert.Len(t, reviews, 2)
	assert.Equal(t, 2, total)
}

func TestReviewService_Create(t *testing.T) {
	ctx := context.Background()
	svc := newReviewService(newFakeReviewStore())

	assert.NoError(t, svc.Create(ctx, baseReview()))
}

func TestReviewService_Create_TxError(t *testing.T) {
	ctx := context.Background()
	svc := service.NewReviewService(&fakeTxRunner{err: errTx}, newFakeReviewStore())

	assert.Error(t, svc.Create(ctx, baseReview()))
}

func TestReviewService_Update(t *testing.T) {
	ctx := context.Background()
	rs := newFakeReviewStore()
	svc := newReviewService(rs)

	r := baseReview()
	assert.NoError(t, rs.Create(ctx, r))

	r.Rating = 5
	r.Comment = "Even better!"
	assert.NoError(t, svc.Update(ctx, r))
}

func TestReviewService_Update_TxError(t *testing.T) {
	ctx := context.Background()
	svc := service.NewReviewService(&fakeTxRunner{err: errTx}, newFakeReviewStore())

	assert.Error(t, svc.Update(ctx, baseReview()))
}

func TestReviewService_Delete(t *testing.T) {
	ctx := context.Background()
	rs := newFakeReviewStore()
	svc := newReviewService(rs)

	r := baseReview()
	assert.NoError(t, rs.Create(ctx, r))
	assert.NoError(t, svc.Delete(ctx, r.ID))
}

func TestReviewService_Delete_TxError(t *testing.T) {
	ctx := context.Background()
	svc := service.NewReviewService(&fakeTxRunner{err: errTx}, newFakeReviewStore())

	assert.Error(t, svc.Delete(ctx, "any-id"))
}
