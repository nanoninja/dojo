// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"errors"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

var (
	// ErrProgressNotFound is returned when no progress record exists for a user/lesson pair.
	ErrProgressNotFound = errors.New("lesson progress not found")
)

// LessonProgressService handles lesson progress tracking and course completion updates.
type LessonProgressService interface {
	// Get returns the progress for a user on a lesson, or ErrProgressNotFound if none.
	Get(ctx context.Context, userID, lessonID string) (*model.LessonProgress, error)

	// ListByCourse returns all progress records for a user within a course.
	ListByCourse(ctx context.Context, userID, courseID string) ([]model.LessonProgress, error)

	// Save records progress for a lesson and atomically recalculates the
	// enrollment progress_percent for the given course.
	Save(ctx context.Context, p *model.LessonProgress, courseID string) error
}

type lessonProgressService struct {
	db          database.TxRunner
	progress    store.LessonProgressStore
	enrollments store.EnrollmentStore
}

// NewLessonProgressService creates a LessonProgressService backed by the given stores.
func NewLessonProgressService(
	db database.TxRunner,
	progress store.LessonProgressStore,
	enrollments store.EnrollmentStore,
) LessonProgressService {
	return &lessonProgressService{
		db:          db,
		progress:    progress,
		enrollments: enrollments,
	}
}

func (s *lessonProgressService) Get(ctx context.Context, userID, lessonID string) (*model.LessonProgress, error) {
	p, err := s.progress.FindByUserAndLesson(ctx, userID, lessonID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProgressNotFound
	}
	return p, nil
}

func (s *lessonProgressService) ListByCourse(ctx context.Context, userID, courseID string) ([]model.LessonProgress, error) {
	return s.progress.ListByUserAndCourse(ctx, userID, courseID)
}

func (s *lessonProgressService) Save(ctx context.Context, p *model.LessonProgress, courseID string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		ps := store.NewLessonProgressStore(q)
		if err := ps.Save(ctx, p); err != nil {
			return err
		}
		percent, err := ps.CalcProgressPercent(ctx, p.UserID, courseID)
		if err != nil {
			return err
		}
		es := store.NewEnrollmentStore(q)
		return es.UpdateProgress(ctx, p.UserID, courseID, percent)
	})
}
