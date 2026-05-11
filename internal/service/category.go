// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// CategoryService handles course category management.
type CategoryService interface {
	// List returns all visible categories.
	List(ctx context.Context) ([]model.Category, error)

	// GetByID returns a category by ID, or ErrCategoryNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Category, error)

	// GetBySlug returns a category by slug, or ErrCategoryNotFound if not found.
	GetBySlug(ctx context.Context, slug string) (*model.Category, error)

	// Create inserts a new category.
	Create(ctx context.Context, c *model.Category) error

	// Update saves changes to an existing category.
	Update(ctx context.Context, c *model.Category) error

	// Delete soft-deletes a category.
	Delete(ctx context.Context, id string) error
}

type categoryService struct {
	store store.CategoryStore
}

// NewCategoryService creates a CategoryService backed by the given store.
func NewCategoryService(s store.CategoryStore) CategoryService {
	return &categoryService{store: s}
}

func (s categoryService) List(ctx context.Context) ([]model.Category, error) {
	return s.store.List(ctx)
}

func (s categoryService) GetByID(ctx context.Context, id string) (*model.Category, error) {
	c, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCategoryNotFound
	}
	return c, nil
}

func (s categoryService) GetBySlug(ctx context.Context, slug string) (*model.Category, error) {
	c, err := s.store.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCategoryNotFound
	}
	return c, nil
}

func (s categoryService) Create(ctx context.Context, c *model.Category) error {
	return s.store.Create(ctx, c)
}

func (s categoryService) Update(ctx context.Context, c *model.Category) error {
	return s.store.Update(ctx, c)
}

func (s categoryService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
