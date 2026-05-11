// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// LessonService handles lesson management.
type LessonService interface {
	// List returns all visible lessons.
	List(ctx context.Context, chapterID string) ([]model.Lesson, error)

	// GetByID returns a lesson by ID, or ErrLessonNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Lesson, error)

	// GetBySlug returns a lesson by slug, or ErrLessonNotFound if not found.
	GetBySlug(ctx context.Context, chapterID, slug string) (*model.Lesson, error)

	// Create inserts a new lesson.
	Create(ctx context.Context, l *model.Lesson) error

	// Update saves changes to an existing lesson.
	Update(ctx context.Context, l *model.Lesson) error

	// Delete removes a lesson.
	Delete(ctx context.Context, id string) error

	// ListResources returns all resources attached to the given lesson.
	ListResources(ctx context.Context, lessonID string) ([]model.LessonResource, error)

	// GetResourceByID returns a resource by ID, or ErrLessonResourceNotFound if not found.
	GetResourceByID(ctx context.Context, id string) (*model.LessonResource, error)

	// AddResource attaches a new resource to a lesson.
	AddResource(ctx context.Context, r *model.LessonResource) error

	// UpdateResource saves changes to an existing resource.
	UpdateResource(ctx context.Context, r *model.LessonResource) error

	// RemoveResource deletes a resource.
	RemoveResource(ctx context.Context, id string) error
}

type lessonService struct {
	store     store.LessonStore
	resources store.LessonResourceStore
}

// NewLessonService creates a LessonService backed by the given stores.
func NewLessonService(s store.LessonStore, r store.LessonResourceStore) LessonService {
	return lessonService{store: s, resources: r}
}

func (s lessonService) List(ctx context.Context, chapterID string) ([]model.Lesson, error) {
	return s.store.List(ctx, chapterID)
}

func (s lessonService) GetByID(ctx context.Context, id string) (*model.Lesson, error) {
	l, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLessonNotFound
	}
	return l, nil
}

func (s lessonService) GetBySlug(ctx context.Context, chapterID, slug string) (*model.Lesson, error) {
	l, err := s.store.FindBySlug(ctx, chapterID, slug)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLessonNotFound
	}
	return l, nil
}

func (s lessonService) Create(ctx context.Context, l *model.Lesson) error {
	return s.store.Create(ctx, l)
}

func (s lessonService) Update(ctx context.Context, l *model.Lesson) error {
	return s.store.Update(ctx, l)
}

func (s lessonService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s lessonService) ListResources(ctx context.Context, lessonID string) ([]model.LessonResource, error) {
	return s.resources.List(ctx, lessonID)
}

func (s lessonService) GetResourceByID(ctx context.Context, id string) (*model.LessonResource, error) {
	r, err := s.resources.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrLessonResourceNotFound
	}
	return r, nil
}

func (s lessonService) AddResource(ctx context.Context, r *model.LessonResource) error {
	return s.resources.Create(ctx, r)
}

func (s lessonService) UpdateResource(ctx context.Context, r *model.LessonResource) error {
	return s.resources.Update(ctx, r)
}

func (s lessonService) RemoveResource(ctx context.Context, id string) error {
	return s.resources.Delete(ctx, id)
}
