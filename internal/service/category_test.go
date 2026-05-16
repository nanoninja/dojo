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
)

// ============================================================================
// fakeCategoryStore
// ============================================================================

type fakeCategoryStore struct {
	categories map[string]*model.Category
	seq        int
}

func newFakeCategoryStore() *fakeCategoryStore {
	return &fakeCategoryStore{categories: make(map[string]*model.Category)}
}

func (f *fakeCategoryStore) nextID() string {
	f.seq++
	return fmt.Sprintf("cat-%d", f.seq)
}

func (f *fakeCategoryStore) List(_ context.Context) ([]model.Category, error) {
	result := make([]model.Category, 0, len(f.categories))
	for _, c := range f.categories {
		if c.DeletedAt == nil {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (f *fakeCategoryStore) FindByID(_ context.Context, id string) (*model.Category, error) {
	c, ok := f.categories[id]
	if !ok || c.DeletedAt != nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (f *fakeCategoryStore) FindBySlug(_ context.Context, slug string) (*model.Category, error) {
	for _, c := range f.categories {
		if c.Slug == slug && c.DeletedAt == nil {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeCategoryStore) Create(_ context.Context, c *model.Category) error {
	c.ID = f.nextID()
	cp := *c
	f.categories[c.ID] = &cp
	return nil
}

func (f *fakeCategoryStore) Update(_ context.Context, c *model.Category) error {
	if _, ok := f.categories[c.ID]; !ok {
		return fmt.Errorf("category not found")
	}
	cp := *c
	f.categories[c.ID] = &cp
	return nil
}

func (f *fakeCategoryStore) Delete(_ context.Context, id string) error {
	c, ok := f.categories[id]
	if !ok {
		return fmt.Errorf("category not found")
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

func newCategoryService(cs *fakeCategoryStore) service.CategoryService {
	return service.NewCategoryService(cs)
}

func TestCategoryService_Create(t *testing.T) {
	ctx := context.Background()
	svc := newCategoryService(newFakeCategoryStore())

	c := &model.Category{Name: "Backend", Slug: "backend"}
	assert.NoError(t, svc.Create(ctx, c))
	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
}

func TestCategoryService_GetByID(t *testing.T) {
	ctx := context.Background()
	svc := newCategoryService(newFakeCategoryStore())

	c := &model.Category{Name: "Backend", Slug: "backend"}
	assert.NoError(t, svc.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, c.ID)
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrCategoryNotFound)
	})
}

func TestCategoryService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	svc := newCategoryService(newFakeCategoryStore())

	c := &model.Category{Name: "Backend", Slug: "backend"}
	assert.NoError(t, svc.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetBySlug(ctx, "backend")
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrCategoryNotFound)
	})
}

func TestCategoryService_List(t *testing.T) {
	ctx := context.Background()
	svc := newCategoryService(newFakeCategoryStore())

	for _, name := range []string{"Backend", "Frontend", "DevOps"} {
		assert.NoError(t, svc.Create(ctx, &model.Category{Name: name, Slug: name}))
	}

	categories, err := svc.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, categories, 3)
}

func TestCategoryService_Update(t *testing.T) {
	ctx := context.Background()
	svc := newCategoryService(newFakeCategoryStore())

	c := &model.Category{Name: "Backend", Slug: "backend"}
	assert.NoError(t, svc.Create(ctx, c))

	c.Name = "Backend Development"
	assert.NoError(t, svc.Update(ctx, c))

	got, err := svc.GetByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Backend Development", got.Name)
}

func TestCategoryService_Delete(t *testing.T) {
	ctx := context.Background()
	svc := newCategoryService(newFakeCategoryStore())

	c := &model.Category{Name: "Backend", Slug: "backend"}
	assert.NoError(t, svc.Create(ctx, c))
	assert.NoError(t, svc.Delete(ctx, c.ID))

	_, err := svc.GetByID(ctx, c.ID)
	assert.ErrorIs(t, err, service.ErrCategoryNotFound)
}
