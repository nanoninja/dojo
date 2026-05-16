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
	// ErrBundleNotFound is returned when a bundle lookup yields no result.
	ErrBundleNotFound = errors.New("bundle not found")

	// ErrBundleSlugTaken is returned when creating or updating a bundle with an already-used slug.
	ErrBundleSlugTaken = errors.New("bundle slug already taken")
)

// BundleService handles bundle management and course assignments.
type BundleService interface {
	// List returns bundles matching the given filter.
	List(ctx context.Context, f store.BundleFilter) ([]model.Bundle, error)

	// GetByID returns a bundle by ID, or ErrBundleNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Bundle, error)

	// GetBySlug returns a bundle by slug, or ErrBundleNotFound if not found.
	GetBySlug(ctx context.Context, slug string) (*model.Bundle, error)

	// Create inserts a bundle and atomically assigns its courses.
	Create(ctx context.Context, b *model.Bundle, courseIDs []string) error

	// Update saves changes to an existing bundle.
	Update(ctx context.Context, b *model.Bundle) error

	// SetCourses replaces all course assignments for a bundle atomically.
	SetCourses(ctx context.Context, bundleID string, courseIDs []string) error

	// Delete soft-deletes a bundles.
	Delete(ctx context.Context, id string) error
}

type bundleService struct {
	db      database.TxRunner
	bundles store.BundleStore
	courses store.BundleCourseStore
}

// NewBundleService creates a BundleService backed by the given stores.
func NewBundleService(
	db database.TxRunner,
	bundles store.BundleStore,
	courses store.BundleCourseStore,
) BundleService {
	return &bundleService{
		db:      db,
		bundles: bundles,
		courses: courses,
	}
}
func (s *bundleService) List(ctx context.Context, f store.BundleFilter) ([]model.Bundle, error) {
	return s.bundles.List(ctx, f)
}

func (s *bundleService) GetByID(ctx context.Context, id string) (*model.Bundle, error) {
	b, err := s.bundles.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrBundleNotFound
	}
	return b, nil
}

func (s *bundleService) GetBySlug(ctx context.Context, slug string) (*model.Bundle, error) {
	b, err := s.bundles.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrBundleNotFound
	}
	return b, nil
}

func (s *bundleService) Create(ctx context.Context, b *model.Bundle, courseIDs []string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		bs := store.NewBundleStore(q)
		if err := bs.Create(ctx, b); err != nil {
			return err
		}
		bcs := store.NewBundleCourseStore(q)
		for i, courseID := range courseIDs {
			if err := bcs.Assign(ctx, b.ID, courseID, (i+1)*10); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *bundleService) Update(ctx context.Context, b *model.Bundle) error {
	return s.bundles.Update(ctx, b)
}

func (s *bundleService) SetCourses(ctx context.Context, bundleID string, courseIDs []string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		bcs := store.NewBundleCourseStore(q)
		existing, err := bcs.List(ctx, bundleID)
		if err != nil {
			return err
		}
		for _, a := range existing {
			if err := bcs.Unassign(ctx, bundleID, a.CourseID); err != nil {
				return err
			}
		}
		for i, courseID := range courseIDs {
			if err := bcs.Assign(ctx, bundleID, courseID, (i+1)*10); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *bundleService) Delete(ctx context.Context, id string) error {
	return s.bundles.Delete(ctx, id)
}
