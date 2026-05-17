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

// LessonProgressStore defines persistence operations for user lesson progress.
type LessonProgressStore interface {
	// FindByUserAndLesson returns the progress for a user on a lesson, or nil if none.
	FindByUserAndLesson(ctx context.Context, userID, lessonID string) (*model.LessonProgress, error)

	// ListByUserAndCourse returns all progress records for a user within a course's lessons.
	ListByUserAndCourse(ctx context.Context, userID, courseID string) ([]model.LessonProgress, error)

	// Save inserts or updates progress for a user on a lesson.
	Save(ctx context.Context, p *model.LessonProgress) error

	// CalcProgressPercent returns the completion percentage for a user in a course,
	// based on published lessons only. Returns 0 if no lessons exist.
	CalcProgressPercent(ctx context.Context, userID, courseID string) (float64, error)
}

type lessonProgressStore struct {
	db database.Querier
}

// NewLessonProgressStore returns a LessonProgressStore backed by the given querier.
func NewLessonProgressStore(db database.Querier) LessonProgressStore {
	return &lessonProgressStore{db: db}
}

func (s *lessonProgressStore) FindByUserAndLesson(ctx context.Context, userID, lessonID string) (*model.LessonProgress, error) {
	var p model.LessonProgress
	err := s.db.GetContext(ctx, &p, `
		SELECT * FROM user_lesson_progress
		WHERE user_id = $1 AND lesson_id = $2`,
		userID, lessonID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *lessonProgressStore) ListByUserAndCourse(ctx context.Context, userID, courseID string) ([]model.LessonProgress, error) {
	result := make([]model.LessonProgress, 0)
	err := s.db.SelectContext(ctx, &result, `
		SELECT ulp.*
		FROM user_lesson_progress ulp
			JOIN lessons AS l ON l.id = ulp.lesson_id
			JOIN chapters AS c ON c.id = l.chapter_id
		WHERE ulp.user_id = $1 AND c.course_id = $2`,
		userID, courseID,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *lessonProgressStore) Save(ctx context.Context, p *model.LessonProgress) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_lesson_progress (
			user_id, lesson_id, is_completed, watched_seconds, last_watched_at
		) VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (user_id, lesson_id) DO UPDATE SET
			is_completed    = EXCLUDED.is_completed,
			watched_seconds = EXCLUDED.watched_seconds,
			last_watched_at = now()`,
		p.UserID, p.LessonID, p.IsCompleted, p.WatchedSeconds,
	)
	return err
}

func (s *lessonProgressStore) CalcProgressPercent(ctx context.Context, userID, courseID string) (float64, error) {
	var percent float64
	err := s.db.GetContext(ctx, &percent, `
		SELECT COALESCE(
			ROUND(
				COUNT(ulp.lesson_id) FILTER (WHERE ulp.is_completed = TRUE) * 100.0
				/ NULLIF(COUNT(l.id), 0),
			2),
		0)
		FROM lessons l
		JOIN chapters c ON c.id = l.chapter_id
		LEFT JOIN user_lesson_progress ulp
			ON ulp.lesson_id = l.id AND ulp.user_id = $1
		WHERE c.course_id = $2
		  AND l.is_published = TRUE`,
		userID, courseID,
	)
	return percent, err
}
