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

// CoursesCategoriesStore defines persistence operations for course-category assignments.
type CoursesCategoriesStore interface {
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
		SELECT * FROM categories
		WHERE deleted_at IS NULL
		ORDER BY sort_order ASC`,
	)
	return categories, err
}

func (s *categoryStore) FindByID(ctx context.Context, id string) (*model.Category, error) {
	var c model.Category
	err := s.db.GetContext(ctx, &c, `
		SELECT * FROM categories
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
		SELECT * FROM categories
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
		INSERT INTO categories (
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
		UPDATE categories
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
		UPDATE categories SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

type coursesCategoriesStore struct {
	db database.Querier
}

// NewCoursesCategoriesStore returns a CourseCoursesCategoriesStore backed by the given querier.
func NewCoursesCategoriesStore(db database.Querier) CoursesCategoriesStore {
	return &coursesCategoriesStore{db: db}
}

func (s *coursesCategoriesStore) List(ctx context.Context, courseID string) ([]model.CategoryAssignment, error) {
	var assignments []model.CategoryAssignment
	err := s.db.SelectContext(ctx, &assignments, `
		SELECT * FROM courses_categories
		WHERE course_id = $1
		ORDER BY is_primary DESC, assigned_at ASC`,
		courseID,
	)
	return assignments, err
}

func (s *coursesCategoriesStore) Assign(ctx context.Context, courseID, categoryID string, isPrimary bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO courses_categories (course_id, category_id, is_primary)
		VALUES ($1, $2, $3)
		ON CONFLICT (course_id, category_id) DO UPDATE SET is_primary = EXCLUDED.is_primary`,
		courseID,
		categoryID,
		isPrimary,
	)
	return err
}

func (s *coursesCategoriesStore) Unassign(ctx context.Context, courseID, categoryID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM courses_categories
		WHERE course_id = $1 AND category_id = $2`,
		courseID,
		categoryID,
	)
	return err
}

func (s *coursesCategoriesStore) SetPrimary(ctx context.Context, courseID, categoryID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE courses_categories
		SET is_primary = (category_id = $2)
		WHERE course_id = $1`,
		courseID,
		categoryID,
	)
	return err
}
