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

// ChapterStore defines persistence operations for course chapters.
type ChapterStore interface {
	List(ctx context.Context, courseID string) ([]model.Chapter, error)
	FindByID(ctx context.Context, id string) (*model.Chapter, error)
	FindBySlug(ctx context.Context, courseID, slug string) (*model.Chapter, error)
	Create(ctx context.Context, c *model.Chapter) error
	Update(ctx context.Context, c *model.Chapter) error
	Delete(ctx context.Context, id string) error
}

type chapterStore struct {
	db database.Querier
}

// NewChapterStore returns a CourseChapterStore backed by the given database connection.
func NewChapterStore(db database.Querier) ChapterStore {
	return &chapterStore{db: db}
}

func (s *chapterStore) List(ctx context.Context, courseID string) ([]model.Chapter, error) {
	var chapters []model.Chapter
	err := s.db.SelectContext(ctx, &chapters, `
		SELECT * FROM chapters
		WHERE course_id = $1 ORDER BY sort_order ASC`,
		courseID,
	)
	return chapters, err
}

func (s *chapterStore) FindByID(ctx context.Context, id string) (*model.Chapter, error) {
	var c model.Chapter
	err := s.db.GetContext(ctx, &c, `SELECT * FROM chapters WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (s *chapterStore) FindBySlug(ctx context.Context, courseID, slug string) (*model.Chapter, error) {
	var c model.Chapter
	err := s.db.GetContext(ctx, &c, `
		SELECT * FROM chapters
		WHERE course_id = $1 AND slug = $2`,
		courseID,
		slug,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (s *chapterStore) Create(ctx context.Context, c *model.Chapter) error {
	return s.db.GetContext(ctx, &c.ID, `
		INSERT INTO chapters (
			course_id,
			title,
			slug,
			description,
			sort_order,
			is_free,
			is_published,
			duration_minutes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		c.CourseID,
		c.Title,
		c.Slug,
		c.Description,
		c.SortOrder,
		c.IsFree,
		c.IsPublished,
		c.DurationMinutes,
	)
}

func (s *chapterStore) Update(ctx context.Context, c *model.Chapter) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE chapters
		SET
			title            = $1,
			slug             = $2,
			description      = $3,
			sort_order       = $4,
			is_free          = $5,
			is_published     = $6,
			duration_minutes = $7
		WHERE id = $8`,
		c.Title,
		c.Slug,
		c.Description,
		c.SortOrder,
		c.IsFree,
		c.IsPublished,
		c.DurationMinutes,
		c.ID,
	)
	return err
}

func (s *chapterStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chapters WHERE id = $1`, id)
	return err
}
