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

// CategoryStore defines persistence operations for course categories.
type CategoryStore interface {
	List(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, id string) (*model.Category, error)
	FindBySlug(ctx context.Context, slug string) (*model.Category, error)
	Create(ctx context.Context, c *model.Category) error
	Update(ctx context.Context, c *model.Category) error
	Delete(ctx context.Context, id string) error
}

// CategoryAssignmentStore defines persistence operations for course-category assignments.
type CategoryAssignmentStore interface {
	List(ctx context.Context, courseID string) ([]model.CategoryAssignment, error)
	Assign(ctx context.Context, courseID, categoryID string, isPrimary bool) error
	Unassign(ctx context.Context, courseID, categoryID string) error
	SetPrimary(ctx context.Context, courseID, categoryID string) error
}

type categoryStore struct {
	db database.Querier
}

// NewCategoryStore returns a CourseCategoryStore backed by the given database connection.
func NewCategoryStore(db database.Querier) CategoryStore {
	return &categoryStore{db: db}
}

func (s *categoryStore) List(ctx context.Context) ([]model.Category, error) {
	var categories []model.Category
	err := s.db.SelectContext(ctx, &categories, `
		SELECT * FROM course_categories
		WHERE deleted_at IS NULL
		ORDER BY sort_order ASC`,
	)
	return categories, err
}

func (s *categoryStore) FindByID(ctx context.Context, id string) (*model.Category, error) {
	var c model.Category
	err := s.db.GetContext(ctx, &c, `
		SELECT * FROM course_categories
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (s *categoryStore) FindBySlug(ctx context.Context, slug string) (*model.Category, error) {
	var c model.Category
	err := s.db.GetContext(ctx, &c, `
		SELECT * FROM course_categories
		WHERE slug = $1 AND deleted_at IS NULL`,
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

func (s *categoryStore) Create(ctx context.Context, c *model.Category) error {
	return s.db.GetContext(ctx, &c.ID, `
		INSERT INTO course_categories (
			parent_id, slug, name, description,
			color_hex, icon_url, sort_order, is_visible
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		c.ParentID,
		c.Slug,
		c.Name,
		c.Description,
		c.ColorHex,
		c.IconURL,
		c.SortOrder,
		c.IsVisible,
	)
}

func (s *categoryStore) Update(ctx context.Context, c *model.Category) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE course_categories
		SET
			parent_id   = $1,
			slug        = $2,
			name        = $3,
			description = $4,
			color_hex   = $5,
			icon_url    = $6,
			sort_order  = $7,
			is_visible  = $8
		WHERE id = $9 AND deleted_at IS NULL`,
		c.ParentID,
		c.Slug,
		c.Name,
		c.Description,
		c.ColorHex,
		c.IconURL,
		c.SortOrder,
		c.IsVisible,
		c.ID,
	)
	return err
}

func (s *categoryStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE course_categories SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

type categoryAssignmentStore struct {
	db database.Querier
}

// NewCategoryAssignmentStore returns a CourseCategoryAssignmentStore backed by the given querier.
func NewCategoryAssignmentStore(db database.Querier) CategoryAssignmentStore {
	return &categoryAssignmentStore{db: db}
}

func (s *categoryAssignmentStore) List(ctx context.Context, courseID string) ([]model.CategoryAssignment, error) {
	var assignments []model.CategoryAssignment
	err := s.db.SelectContext(ctx, &assignments, `
		SELECT * FROM course_category_assignments
		WHERE course_id = $1
		ORDER BY is_primary DESC, assigned_at ASC`,
		courseID,
	)
	return assignments, err
}

func (s *categoryAssignmentStore) Assign(ctx context.Context, courseID, categoryID string, isPrimary bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO course_category_assignments (course_id, category_id, is_primary)
		VALUES ($1, $2, $3)
		ON CONFLICT (course_id, category_id) DO UPDATE SET is_primary = EXCLUDED.is_primary`,
		courseID,
		categoryID,
		isPrimary,
	)
	return err
}

func (s *categoryAssignmentStore) Unassign(ctx context.Context, courseID, categoryID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM course_category_assignments
		WHERE course_id = $1 AND category_id = $2`,
		courseID,
		categoryID,
	)
	return err
}

func (s *categoryAssignmentStore) SetPrimary(ctx context.Context, courseID, categoryID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE course_category_assignments
		SET is_primary = (category_id = $2)
		WHERE course_id = $1`,
		courseID,
		categoryID,
	)
	return err
}
