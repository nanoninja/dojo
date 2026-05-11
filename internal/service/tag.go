// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// TagService handles tag management.
type TagService interface {
	// List returns all tags.
	List(ctx context.Context) ([]model.Tag, error)

	// GetByID returns a tag by ID, or ErrTagNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Tag, error)

	// GetBySlug returns a tag by slug, or ErrTagNotFound if not found.
	GetBySlug(ctx context.Context, slug string) (*model.Tag, error)

	// Create inserts a new tag.
	Create(ctx context.Context, t *model.Tag) error

	// Updates saves changes to an existing tag.
	Update(ctx context.Context, t *model.Tag) error

	// Delete removes a tag.
	Delete(ctx context.Context, id string) error
}

type tagService struct {
	store store.TagStore
}

// NewTagService creates a TagService backed by the given store.
func NewTagService(s store.TagStore) TagService {
	return tagService{store: s}
}

func (s tagService) List(ctx context.Context) ([]model.Tag, error) {
	return s.store.List(ctx)
}

func (s tagService) GetByID(ctx context.Context, id string) (*model.Tag, error) {
	t, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTagNotFound
	}
	return t, nil
}

func (s tagService) GetBySlug(ctx context.Context, slug string) (*model.Tag, error) {
	t, err := s.store.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTagNotFound
	}
	return t, nil
}

func (s tagService) Create(ctx context.Context, t *model.Tag) error {
	return s.store.Create(ctx, t)
}

func (s tagService) Update(ctx context.Context, t *model.Tag) error {
	return s.store.Update(ctx, t)
}

func (s tagService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
