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

// CourseFilter holds filtering options for listing courses.
type CourseFilter struct {
	InstructorID string
	CategoryID   string
	Search       string
	Level        model.CourseLevel
	Language     string
	IsFree       *bool
	IsPublished  *bool
	SortDir      SortDir
	Limit        int
	Offset       int
}

// CourseStore defines persistence operations for courses.
type CourseStore interface {
	// List returns courses matching the given filter and the total count without pagination.
	List(ctx context.Context, f CourseFilter) ([]model.Course, int, error)

	// FindByID returns a course by its ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Course, error)

	// FindBySlug returns a course by its slug, or nil if not found.
	FindBySlug(ctx context.Context, slug string) (*model.Course, error)

	// Create inserts a new course and sets its ID.
	Create(ctx context.Context, c *model.Course) error

	// Update persists changes to an existing course.
	Update(ctx context.Context, c *model.Course) error

	// Delete soft-deletes a course by setting deleted_at.
	Delete(ctx context.Context, id string) error
}

type courseStore struct {
	db database.Querier
}

// NewCourseStore returns a CourseStore backed by the given database connection.
func NewCourseStore(db database.Querier) CourseStore {
	return &courseStore{db: db}
}

func (s *courseStore) List(ctx context.Context, f CourseFilter) ([]model.Course, int, error) {
	where := ` WHERE deleted_at IS NULL`
	args := make([]any, 0, 10)

	if f.InstructorID != "" {
		where += ` AND instructor_id = ?`
		args = append(args, f.InstructorID)
	}
	if f.CategoryID != "" {
		where += ` AND id IN (
			SELECT course_id FROM courses_categories WHERE category_id = ?
		)`
		args = append(args, f.CategoryID)
	}
	if f.Search != "" {
		where += ` AND (title LIKE ? OR subtitle LIKE ?)`
		s := "%" + f.Search + "%"
		args = append(args, s, s)
	}
	if f.Level != "" {
		where += ` AND level = ?`
		args = append(args, f.Level)
	}
	if f.Language != "" {
		where += ` AND language = ?`
		args = append(args, f.Language)
	}
	if f.IsFree != nil {
		where += ` AND is_free = ?`
		args = append(args, *f.IsFree)
	}
	if f.IsPublished != nil {
		where += ` AND is_published = ?`
		args = append(args, *f.IsPublished)
	}

	var total int
	countQuery := s.db.Rebind(`SELECT COUNT(*) FROM courses` + where)
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	order := SortDirDesc
	if f.SortDir == SortDirAsc {
		order = SortDirAsc
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}

	query := s.db.Rebind(`SELECT * FROM courses` + where + ` ORDER BY created_at ` + string(order) + ` LIMIT ?`)
	pageArgs := append(args, f.Limit)

	if f.Offset > 0 {
		query = s.db.Rebind(`SELECT * FROM courses` + where + ` ORDER BY created_at ` + string(order) + ` LIMIT ? OFFSET ?`)
		pageArgs = append(pageArgs, f.Offset)
	}

	courses := make([]model.Course, 0, f.Limit)
	if err := s.db.SelectContext(ctx, &courses, query, pageArgs...); err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

func (s *courseStore) FindByID(ctx context.Context, id string) (*model.Course, error) {
	var c model.Course
	err := s.db.GetContext(ctx, &c, `
		SELECT * FROM courses
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

func (s *courseStore) FindBySlug(ctx context.Context, slug string) (*model.Course, error) {
	var c model.Course
	err := s.db.GetContext(ctx, &c, `
		SELECT * FROM courses
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

func (s *courseStore) Create(ctx context.Context, c *model.Course) error {
	return s.db.GetContext(ctx, &c.ID, `
		INSERT INTO courses (
			instructor_id,
			slug,
			title,
			subtitle,
			description,
			prerequisites,
			objectives,
			meta_title,
			meta_description,
			meta_keywords,
			thumbnail_url,
			trailer_url, 
			level,
			content_type,
			language,
			duration_minutes,
			is_free,
			subscription_only, 
			price_cents,
			currency,
			is_published,
			is_featured,
			certificate_enabled,
			sort_order,
			published_at
		) VALUES (
		 	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		) RETURNING id`,
		c.InstructorID,
		c.Slug,
		c.Title,
		c.Subtitle,
		c.Description,
		c.Prerequisites,
		c.Objectives,
		c.MetaTitle,
		c.MetaDescription,
		c.MetaKeywords,
		c.ThumbnailURL,
		c.TrailerURL,
		c.Level,
		c.ContentType,
		c.Language,
		c.DurationMinutes,
		c.IsFree,
		c.SubscriptionOnly,
		c.PriceCents,
		c.Currency,
		c.IsPublished,
		c.IsFeatured,
		c.CertificateEnabled,
		c.SortOrder,
		c.PublishedAt,
	)
}

func (s *courseStore) Update(ctx context.Context, c *model.Course) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE courses
		SET
			slug                = $1,
			title               = $2,
			subtitle            = $3,
			description         = $4,
			prerequisites       = $5,
			objectives          = $6,
			meta_title          = $7,
			meta_description    = $8,
			meta_keywords       = $9,
			thumbnail_url       = $10,
			trailer_url         = $11,
			level               = $12,
			content_type        = $13,
			language            = $14,
			duration_minutes    = $15,
			is_free             = $16,
			subscription_only   = $17,
			price_cents         = $18,
			currency            = $19,
			is_published        = $20,
			is_featured         = $21,
			certificate_enabled = $22,
			sort_order          = $23,
			published_at        = $24
		WHERE id = $25 AND deleted_at IS NULL`,
		c.Slug,
		c.Title,
		c.Subtitle,
		c.Description,
		c.Prerequisites,
		c.Objectives,
		c.MetaTitle,
		c.MetaDescription,
		c.MetaKeywords,
		c.ThumbnailURL,
		c.TrailerURL,
		c.Level,
		c.ContentType,
		c.Language,
		c.DurationMinutes,
		c.IsFree,
		c.SubscriptionOnly,
		c.PriceCents,
		c.Currency,
		c.IsPublished,
		c.IsFeatured,
		c.CertificateEnabled,
		c.SortOrder,
		c.PublishedAt,
		c.ID,
	)
	return err
}

func (s *courseStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE courses SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
