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

// BundleFilter holds filtering options for listing bundles.
type BundleFilter struct {
	InstructorID string
	IsPublished  *bool
	Limit        int
	Offset       int
}

// BundleStore defines persistence operations for bundles.
type BundleStore interface {
	// List returns bundles matching the given filter.
	List(ctx context.Context, f BundleFilter) ([]model.Bundle, error)

	// FindByID returns a bundle by its ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Bundle, error)

	// FindBySlug returns a bundle by its slug, or nil if not found.
	FindBySlug(ctx context.Context, slug string) (*model.Bundle, error)

	// Create inserts a new bundle and sets its ID.
	Create(ctx context.Context, b *model.Bundle) error

	// Update persists changes to an existing bundle.
	Update(ctx context.Context, b *model.Bundle) error

	// Delete soft-deletes a bundle by setting deleted_at.
	Delete(ctx context.Context, id string) error
}

// BundleCourseStore defines persistence operations for bundle-course assignments.
type BundleCourseStore interface {
	// List returns all course assignments for a given bundle, ordered by sort_order.
	List(ctx context.Context, bundleID string) ([]model.BundleCourseAssignment, error)

	// Assign adds a course to a bundle. If already assigned, updates sort_order.
	Assign(ctx context.Context, bundleID, courseID string, sortOrder int) error

	// Unassign removes a course from a bundle.
	Unassign(ctx context.Context, bundleID, courseID string) error
}

type bundleStore struct {
	db database.Querier
}

// NewBundleStore returns a bundleStore backed by the given querier.
func NewBundleStore(db database.Querier) BundleStore {
	return &bundleStore{db: db}
}

func (s *bundleStore) List(ctx context.Context, f BundleFilter) ([]model.Bundle, error) {
	query := `SELECT * FROM bundles WHERE deleted_at IS NULL`
	args := make([]any, 0, 4)

	if f.InstructorID != "" {
		query += ` AND instructor_id = ?`
		args = append(args, f.InstructorID)
	}
	if f.IsPublished != nil {
		query += ` AND is_published = ?`
		args = append(args, *f.IsPublished)
	}

	query += ` ORDER BY sort_order ASC, created_at DESC`

	if f.Limit <= 0 {
		f.Limit = 100
	}

	query += ` LIMIT ?`
	args = append(args, f.Limit)

	if f.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, f.Offset)
	}

	query = s.db.Rebind(query)
	bundles := make([]model.Bundle, 0, f.Limit)

	if err := s.db.SelectContext(ctx, &bundles, query, args...); err != nil {
		return nil, err
	}
	return bundles, nil
}

func (s *bundleStore) FindByID(ctx context.Context, id string) (*model.Bundle, error) {
	var b model.Bundle
	err := s.db.GetContext(ctx, &b, `
		SELECT * FROM bundles
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (s *bundleStore) FindBySlug(ctx context.Context, slug string) (*model.Bundle, error) {
	var b model.Bundle
	err := s.db.GetContext(ctx, &b, `
		SELECT * FROM bundles
		WHERE slug = $1 AND deleted_at IS NULL`,
		slug,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (s *bundleStore) Create(ctx context.Context, b *model.Bundle) error {
	return s.db.GetContext(ctx, &b.ID, `
		INSERT INTO bundles (
			instructor_id, slug, title, subtitle, description,
			thumbnail_url, is_free, price_cents, currency,
			is_published, sort_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		b.InstructorID, b.Slug, b.Title, b.Subtitle, b.Description,
		b.ThumbnailURL, b.IsFree, b.PriceCents, b.Currency,
		b.IsPublished, b.SortOrder,
	)
}

func (s *bundleStore) Update(ctx context.Context, b *model.Bundle) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE bundles
		SET
			slug          = $1,
			title         = $2,
			subtitle      = $3,
			description   = $4,
			thumbnail_url = $5,
			is_free       = $6,
			price_cents   = $7,
			currency      = $8,
			is_published  = $9,
			sort_order    = $10
		WHERE id = $11 AND deleted_at IS NULL`,
		b.Slug, b.Title, b.Subtitle, b.Description,
		b.ThumbnailURL, b.IsFree, b.PriceCents, b.Currency,
		b.IsPublished, b.SortOrder, b.ID,
	)
	return err
}

func (s *bundleStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE bundles SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

type bundleCourseStore struct {
	db database.Querier
}

// NewBundleCourseStore returns a BundleCourseStore backed by the giver querier.
func NewBundleCourseStore(db database.Querier) BundleCourseStore {
	return &bundleCourseStore{db: db}
}

func (s *bundleCourseStore) List(ctx context.Context, bundleID string) ([]model.BundleCourseAssignment, error) {
	assignments := make([]model.BundleCourseAssignment, 0)
	err := s.db.SelectContext(ctx, &assignments, `
		SELECT * FROM bundle_courses
		WHERE bundle_id = $1
		ORDER BY sort_order ASC`,
		bundleID,
	)
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *bundleCourseStore) Assign(ctx context.Context, bundleID, courseID string, sortOrder int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bundle_courses (bundle_id, course_id, sort_order)
		VALUES ($1, $2, $3)
		ON CONFLICT (bundle_id, course_id) DO UPDATE SET sort_order = EXCLUDED.sort_order`,
		bundleID, courseID, sortOrder,
	)
	return err
}

func (s *bundleCourseStore) Unassign(ctx context.Context, bundleID, courseID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM bundle_courses WHERE bundle_id = $1 AND course_id = $2`,
		bundleID, courseID,
	)
	return err
}
