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

// LessonStore defines persistence operations for lessons.
// It covers the full lifecycle of a lesson within a chapter.
type LessonStore interface {
	// List returns all lessons for the given chapter, ordered by sort_order.
	List(ctx context.Context, chapterID string) ([]model.Lesson, error)

	// FindByID returns a lesson by ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Lesson, error)

	// FindBySlug returns a lesson by slug within a chapter, or nil if not found.
	FindBySlug(ctx context.Context, chapterID, slug string) (*model.Lesson, error)

	// Create inserts a new lesson and sets its ID.
	Create(ctx context.Context, l *model.Lesson) error

	// Update saves changes to an existing lesson.
	Update(ctx context.Context, l *model.Lesson) error

	// Delete removes a lesson.
	Delete(ctx context.Context, id string) error
}

// LessonResourceStore defines persistence operations for lesson resources.
// Resources are files or links attached to a lesson (PDFs, slides, etc.).
type LessonResourceStore interface {
	// List returns all resources for the given lesson.
	List(ctx context.Context, lessonID string) ([]model.LessonResource, error)

	// FindByID returns a resource by ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.LessonResource, error)

	// Create inserts a new resource and sets its ID.
	Create(ctx context.Context, r *model.LessonResource) error

	// Update saves changes to an existing resource.
	Update(ctx context.Context, r *model.LessonResource) error

	// Delete removes a resource.
	Delete(ctx context.Context, id string) error
}

type lessonStore struct {
	db database.Querier
}

// NewLessonStore returns a LessonStore backed by the given database connection.
func NewLessonStore(db database.Querier) LessonStore {
	return &lessonStore{db: db}
}

func (s *lessonStore) List(ctx context.Context, chapterID string) ([]model.Lesson, error) {
	var lessons []model.Lesson
	err := s.db.SelectContext(ctx, &lessons, `
		SELECT * FROM lessons
		WHERE chapter_id = $1
		ORDER BY sort_order ASC`,
		chapterID,
	)
	return lessons, err
}

func (s *lessonStore) FindByID(ctx context.Context, id string) (*model.Lesson, error) {
	var l model.Lesson
	err := s.db.GetContext(ctx, &l, `
		SELECT * FROM lessons
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (s *lessonStore) FindBySlug(ctx context.Context, chapterID, slug string) (*model.Lesson, error) {
	var l model.Lesson
	err := s.db.GetContext(ctx, &l, `
		SELECT * FROM lessons
		WHERE chapter_id = $1 AND slug = $2`,
		chapterID,
		slug,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (s *lessonStore) Create(ctx context.Context, l *model.Lesson) error {
	return s.db.GetContext(ctx, &l.ID, `
		INSERT INTO lessons (
			chapter_id,
			title,
			slug,
			description,
			sort_order,
			content_type,
			media_url,
			is_free,
			is_published,
			duration_minutes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		RETURNING id`,
		l.ChapterID,
		l.Title,
		l.Slug,
		l.Description,
		l.SortOrder,
		l.ContentType,
		l.MediaURL,
		l.IsFree,
		l.IsPublished,
		l.DurationMinutes,
	)
}

func (s *lessonStore) Update(ctx context.Context, l *model.Lesson) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE lessons
		SET
			title            = $1,
			slug             = $2,
			description      = $3,
			sort_order       = $4,
			content_type     = $5,
			media_url        = $6,
			is_free          = $7,
			is_published     = $8,
			duration_minutes = $9
		WHERE id = $10`,
		l.Title,
		l.Slug,
		l.Description,
		l.SortOrder,
		l.ContentType,
		l.MediaURL,
		l.IsFree,
		l.IsPublished,
		l.DurationMinutes,
		l.ID,
	)
	return err
}

func (s *lessonStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM lessons WHERE id = $1`, id)
	return err
}

type lessonResourceStore struct {
	db database.Querier
}

// NewLessonResourceStore returns a LessonResourceStore backed by the given querier.
func NewLessonResourceStore(db database.Querier) LessonResourceStore {
	return &lessonResourceStore{db: db}
}

func (s *lessonResourceStore) List(ctx context.Context, lessonID string) ([]model.LessonResource, error) {
	var resources []model.LessonResource
	err := s.db.SelectContext(ctx, &resources, `
		SELECT * FROM lesson_resources
		WHERE lesson_id = $1`,
		lessonID,
	)
	return resources, err
}

func (s *lessonResourceStore) FindByID(ctx context.Context, id string) (*model.LessonResource, error) {
	var resource model.LessonResource
	err := s.db.GetContext(ctx, &resource, `
		SELECT * FROM lesson_resources
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &resource, nil
}

func (s *lessonResourceStore) Create(ctx context.Context, r *model.LessonResource) error {
	return s.db.GetContext(ctx, &r.ID, `
		INSERT INTO lesson_resources (
			lesson_id,
			title,
			description,
			file_url,
			file_name,
			file_size_bytes,
			mime_type,
			sort_order,
			is_public
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id`,
		r.LessonID,
		r.Title,
		r.Description,
		r.FileURL,
		r.FileName,
		r.FileSizeBytes,
		r.MimeType,
		r.SortOrder,
		r.IsPublic,
	)
}

func (s *lessonResourceStore) Update(ctx context.Context, r *model.LessonResource) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE lesson_resources
		SET
            title           = $1,
            description     = $2,
            file_url        = $3,
            file_name       = $4,
            file_size_bytes = $5,
            mime_type       = $6,
            sort_order      = $7,
            is_public       = $8
		WHERE id = $9`,
		r.Title,
		r.Description,
		r.FileURL,
		r.FileName,
		r.FileSizeBytes,
		r.MimeType,
		r.SortOrder,
		r.IsPublic,
		r.ID,
	)
	return err
}

func (s *lessonResourceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM lesson_resources WHERE id = $1`, id)
	return err
}
