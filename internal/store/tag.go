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

// TagStore defines persistence operations for tags.
type TagStore interface {
	List(ctx context.Context) ([]model.Tag, error)
	FindByID(ctx context.Context, id string) (*model.Tag, error)
	FindBySlug(ctx context.Context, slug string) (*model.Tag, error)
	Create(ctx context.Context, t *model.Tag) error
	Update(ctx context.Context, t *model.Tag) error
	Delete(ctx context.Context, id string) error
}

// TagAssignmentStore defines persistence operations for course-tag assignments.
type TagAssignmentStore interface {
	List(ctx context.Context, courseID string) ([]model.CourseTagAssignment, error)
	Assign(ctx context.Context, courseID, tagID string) error
	Unassign(ctx context.Context, courseID, tagID string) error
}

type tagStore struct {
	db database.Querier
}

// NewTagStore returns a TagStore backed by the given querier.
func NewTagStore(db database.Querier) TagStore {
	return &tagStore{db: db}
}

func (s *tagStore) List(ctx context.Context) ([]model.Tag, error) {
	var tags []model.Tag
	err := s.db.SelectContext(ctx, &tags, `
		SELECT * FROM tags
		ORDER BY created_at ASC`,
	)
	return tags, err
}

func (s *tagStore) FindByID(ctx context.Context, id string) (*model.Tag, error) {
	var tag model.Tag
	err := s.db.GetContext(ctx, &tag, `SELECT * FROM tags WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

func (s *tagStore) FindBySlug(ctx context.Context, slug string) (*model.Tag, error) {
	var tag model.Tag
	err := s.db.GetContext(ctx, &tag, `SELECT * FROM tags WHERE slug = $1`, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

func (s *tagStore) Create(ctx context.Context, t *model.Tag) error {
	return s.db.GetContext(ctx, &t.ID, `
		INSERT INTO tags (slug, name) VALUES ($1, $2) RETURNING id`,
		t.Slug,
		t.Name,
	)
}

func (s *tagStore) Update(ctx context.Context, t *model.Tag) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tags SET slug = $1, name = $2 WHERE id = $3`,
		t.Slug,
		t.Name,
		t.ID,
	)
	return err
}

func (s *tagStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id = $1`, id)
	return err
}

type tagAssignmentStore struct {
	db database.Querier
}

// NewTagAssignmentStore returns a CourseTagAssignmentStore backed by the given querier.
func NewTagAssignmentStore(db database.Querier) TagAssignmentStore {
	return &tagAssignmentStore{db: db}
}

func (s *tagAssignmentStore) List(ctx context.Context, courseID string) ([]model.CourseTagAssignment, error) {
	var assignments []model.CourseTagAssignment
	err := s.db.SelectContext(ctx, &assignments, `
		SELECT * FROM course_tag_assignments
		WHERE course_id = $1
		ORDER BY assigned_at ASC`,
		courseID,
	)
	return assignments, err
}

func (s *tagAssignmentStore) Assign(ctx context.Context, courseID, tagID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO course_tag_assignments (course_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT (course_id, tag_id) DO NOTHING`,
		courseID,
		tagID,
	)
	return err
}

func (s *tagAssignmentStore) Unassign(ctx context.Context, courseID, tagID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM course_tag_assignments
		WHERE course_id = $1 AND tag_id = $2`,
		courseID,
		tagID,
	)
	return err
}
