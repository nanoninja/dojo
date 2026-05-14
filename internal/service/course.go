// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

// CourseService handles course management: creation, updates, categorization, and deletion.
type CourseService interface {
	// List returns courses matching the given filter.
	List(ctx context.Context, f store.CourseFilter) ([]model.Course, error)

	// GetByID returns a course by ID, or ErrCourseNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Course, error)

	// GetBySlug returns a course by slug, or ErrCourseNotFound if not found.
	GetBySlug(ctx context.Context, slug string) (*model.Course, error)

	// Create inserts the course and atomically assigns its categories and tags.
	// primaryCategoryID may be empty if no primary category is set.
	Create(ctx context.Context, c *model.Course, categoryIDs []string, primaryCategoryID string, tagIDs []string) error

	// Update saves changes to an existing course.
	Update(ctx context.Context, c *model.Course) error

	// SetCategories replaces all category assignments for a course atomically.
	SetCategories(ctx context.Context, courseID string, categoryIDs []string, primaryCategoryID string) error

	// SetTags replaces all tag assignments for a course atomically.
	SetTags(ctx context.Context, courseID string, tagIDs []string) error

	// Delete soft-deletes a course.
	Delete(ctx context.Context, id string) error
}

type courseService struct {
	db         database.TxRunner
	courses    store.CourseStore
	categories store.CoursesCategoriesStore
	tags       store.CoursesTagsStore
}

// NewCourseService creates a CourseService backed by the given stores.
func NewCourseService(
	db database.TxRunner,
	courses store.CourseStore,
	categories store.CoursesCategoriesStore,
	tags store.CoursesTagsStore,
) CourseService {
	return &courseService{
		db:         db,
		courses:    courses,
		categories: categories,
		tags:       tags,
	}
}

func (s *courseService) List(ctx context.Context, f store.CourseFilter) ([]model.Course, error) {
	return s.courses.List(ctx, f)
}

func (s *courseService) GetByID(ctx context.Context, id string) (*model.Course, error) {
	c, err := s.courses.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	return c, nil
}

func (s *courseService) GetBySlug(ctx context.Context, slug string) (*model.Course, error) {
	c, err := s.courses.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	return c, nil
}

// Create uses WithTx: inserting the course + assigning categories and tags must succeed or fail together.
func (s *courseService) Create(ctx context.Context, c *model.Course, categoryIDs []string, primaryCategoryID string, tagIDs []string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		cs := store.NewCourseStore(q)
		if err := cs.Create(ctx, c); err != nil {
			return err
		}
		cats := store.NewCoursesCategoriesStore(q)
		for _, catID := range categoryIDs {
			if err := cats.Assign(ctx, c.ID, catID, catID == primaryCategoryID); err != nil {
				return err
			}
		}
		tags := store.NewCoursesTagsStore(q)
		for _, tagID := range tagIDs {
			if err := tags.Assign(ctx, c.ID, tagID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *courseService) Update(ctx context.Context, c *model.Course) error {
	return s.courses.Update(ctx, c)
}

// SetCategories uses WithTx: unassigning all then reassigning must be atomic to avoid partial state.
func (s *courseService) SetCategories(ctx context.Context, courseID string, categoryIDs []string, primaryCategoryID string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		cats := store.NewCoursesCategoriesStore(q)
		existing, err := cats.List(ctx, courseID)
		if err != nil {
			return err
		}
		for _, a := range existing {
			if err := cats.Unassign(ctx, courseID, a.CategoryID); err != nil {
				return err
			}
		}
		for _, catID := range categoryIDs {
			if err := cats.Assign(ctx, courseID, catID, catID == primaryCategoryID); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetTags uses WithTx: unassigning all then reassigning must be atomic to avoid partial state.
func (s *courseService) SetTags(ctx context.Context, courseID string, tagIDs []string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		tags := store.NewCoursesTagsStore(q)
		existing, err := tags.List(ctx, courseID)
		if err != nil {
			return err
		}
		for _, a := range existing {
			if err := tags.Unassign(ctx, courseID, a.TagID); err != nil {
				return err
			}
		}
		for _, tagID := range tagIDs {
			if err := tags.Assign(ctx, courseID, tagID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *courseService) Delete(ctx context.Context, id string) error {
	return s.courses.Delete(ctx, id)
}
