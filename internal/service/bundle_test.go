// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// fakeBundleStore
// ============================================================================

type fakeBundleStore struct {
	bundles map[string]*model.Bundle
	seq     int
}

func newFakeBundleStore() *fakeBundleStore {
	return &fakeBundleStore{bundles: make(map[string]*model.Bundle)}
}

func (s *fakeBundleStore) nextID() string {
	s.seq++
	return fmt.Sprintf("bundle-%d", s.seq)
}

func (s *fakeBundleStore) List(_ context.Context, filter store.BundleFilter) ([]model.Bundle, error) {
	result := make([]model.Bundle, 0)

	for _, b := range s.bundles {
		if b.DeletedAt != nil {
			continue
		}
		if filter.InstructorID != "" && b.InstructorID != filter.InstructorID {
			continue
		}
		if filter.IsPublished != nil && b.IsPublished != *filter.IsPublished {
			continue
		}
		result = append(result, *b)
	}
	return result, nil
}

func (s *fakeBundleStore) FindByID(_ context.Context, id string) (*model.Bundle, error) {
	b, ok := s.bundles[id]
	if !ok || b.DeletedAt != nil {
		return nil, nil
	}
	cp := *b
	return &cp, nil
}

func (s *fakeBundleStore) FindBySlug(_ context.Context, slug string) (*model.Bundle, error) {
	for _, b := range s.bundles {
		if b.Slug == slug && b.DeletedAt == nil {
			cp := *b
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *fakeBundleStore) Create(_ context.Context, b *model.Bundle) error {
	b.ID = s.nextID()
	cp := *b
	s.bundles[b.ID] = &cp
	return nil
}

func (s fakeBundleStore) Update(_ context.Context, b *model.Bundle) error {
	if _, ok := s.bundles[b.ID]; !ok {
		return fmt.Errorf("bundle not found")
	}
	cp := *b
	s.bundles[b.ID] = &cp
	return nil
}

func (s *fakeBundleStore) Delete(_ context.Context, id string) error {
	b, ok := s.bundles[id]
	if !ok {
		return fmt.Errorf("bundle not found")
	}
	now := time.Now()
	b.DeletedAt = &now
	return nil
}

// ============================================================================
// fakeBundleCourseStore
// ============================================================================

type fakeBundleCourseStore struct {
	assignments []model.BundleCourseAssignment
}

func (f *fakeBundleCourseStore) List(_ context.Context, bundleID string) ([]model.BundleCourseAssignment, error) {
	result := make([]model.BundleCourseAssignment, 0)
	for _, a := range f.assignments {
		if a.BundleID == bundleID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *fakeBundleCourseStore) Assign(_ context.Context, bundleID, courseID string, sortOrder int) error {
	for i, a := range f.assignments {
		if a.BundleID == bundleID && a.CourseID == courseID {
			f.assignments[i].SortOrder = sortOrder
			return nil
		}
	}
	f.assignments = append(f.assignments, model.BundleCourseAssignment{
		BundleID:  bundleID,
		CourseID:  courseID,
		SortOrder: sortOrder,
	})
	return nil
}

func (f *fakeBundleCourseStore) Unassign(_ context.Context, bundleID, courseID string) error {
	result := f.assignments[:0]
	for _, a := range f.assignments {
		if a.BundleID != bundleID || a.CourseID != courseID {
			result = append(result, a)
		}
	}
	f.assignments = result
	return nil
}

func newBundleService(bs *fakeBundleStore, bcs *fakeBundleCourseStore) service.BundleService {
	return service.NewBundleService(&fakeTxRunner{}, bs, bcs)
}

func newTestBundle() *model.Bundle {
	return &model.Bundle{
		InstructorID: "instructor-1",
		Slug:         "go-bundle",
		Title:        "Go Bundle",
		Currency:     "EUR",
	}
}

func TestBundleService_Create(t *testing.T) {
	ctx := context.Background()
	svc := newBundleService(newFakeBundleStore(), &fakeBundleCourseStore{})

	b := newTestBundle()
	assert.NoError(t, svc.Create(ctx, b, nil))
}

func TestBundleService_GetByID(t *testing.T) {
	ctx := context.Background()
	bs := newFakeBundleStore()
	svc := newBundleService(bs, &fakeBundleCourseStore{})

	b := newTestBundle()
	assert.NoError(t, bs.Create(ctx, b))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, b.ID)

		assert.NoError(t, err)
		assert.Equal(t, b.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrBundleNotFound)
	})
}

func TestBundleService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	bs := newFakeBundleStore()
	svc := newBundleService(bs, &fakeBundleCourseStore{})

	b := newTestBundle()
	assert.NoError(t, bs.Create(ctx, b))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetBySlug(ctx, "go-bundle")
		assert.NoError(t, err)
		assert.Equal(t, b.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrBundleNotFound)
	})
}

func TestBundleService_List(t *testing.T) {
	ctx := context.Background()
	bs := newFakeBundleStore()
	svc := newBundleService(bs, &fakeBundleCourseStore{})

	b1 := newTestBundle()
	b2 := &model.Bundle{
		InstructorID: "instructor-2",
		Slug:         "python-bundle",
		Title:        "Python Bundle",
		Currency:     "EUR",
	}

	assert.NoError(t, bs.Create(ctx, b1))
	assert.NoError(t, bs.Create(ctx, b2))

	t.Run("no filter", func(t *testing.T) {
		result, err := svc.List(ctx, store.BundleFilter{})

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("filter by instructor", func(t *testing.T) {
		result, err := svc.List(ctx, store.BundleFilter{InstructorID: "instructor-1"})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestBundleService_Update(t *testing.T) {
	ctx := context.Background()
	bs := newFakeBundleStore()
	svc := newBundleService(bs, &fakeBundleCourseStore{})

	b := newTestBundle()
	assert.NoError(t, bs.Create(ctx, b))

	b.Title = "Go Bundle Updated"
	assert.NoError(t, svc.Update(ctx, b))

	got, err := svc.GetByID(ctx, b.ID)

	assert.NoError(t, err)
	assert.Equal(t, "Go Bundle Updated", got.Title)
}

func TestBundleService_SetCourses(t *testing.T) {
	ctx := context.Background()
	bs := newFakeBundleStore()
	bcs := &fakeBundleCourseStore{}
	svc := newBundleService(bs, bcs)

	b := newTestBundle()
	assert.NoError(t, bs.Create(ctx, b))

	assert.NoError(t, svc.SetCourses(ctx, b.ID, []string{"course-1", "course-2"}))
}

func TestBundleService_Delete(t *testing.T) {
	ctx := context.Background()
	bs := newFakeBundleStore()
	svc := newBundleService(bs, &fakeBundleCourseStore{})

	b := newTestBundle()
	assert.NoError(t, bs.Create(ctx, b))
	assert.NoError(t, bs.Delete(ctx, b.ID))

	_, err := svc.GetByID(ctx, b.ID)
	assert.ErrorIs(t, err, service.ErrBundleNotFound)
}
