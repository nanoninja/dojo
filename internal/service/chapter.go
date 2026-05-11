// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// ChapterService handles course chapter management.
type ChapterService interface {
	// List returns all chapters for the given course, ordered by sort_order.
	List(ctx context.Context, courseID string) ([]model.Chapter, error)

	// GetByID returns a chapter by ID, or ErrCourseChapterNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Chapter, error)

	// GetBySlug returns a chapter by slug within a course, or ErrCourseChapterNotFound if not found.
	GetBySlug(ctx context.Context, courseID, slug string) (*model.Chapter, error)

	// Create inserts a new chapter.
	Create(ctx context.Context, c *model.Chapter) error

	// Update saves changes to an existing chapter.
	Update(ctx context.Context, c *model.Chapter) error

	// Delete removes a chapter.
	Delete(ctx context.Context, id string) error
}

type chapterService struct {
	store store.ChapterStore
}

// NewChapterService creates a CourseChapterService backed by the given store.
func NewChapterService(s store.ChapterStore) ChapterService {
	return chapterService{store: s}
}

func (s chapterService) List(ctx context.Context, courseID string) ([]model.Chapter, error) {
	return s.store.List(ctx, courseID)
}

func (s chapterService) GetByID(ctx context.Context, id string) (*model.Chapter, error) {
	c, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrChapterNotFound
	}
	return c, nil
}

func (s chapterService) GetBySlug(ctx context.Context, courseID, slug string) (*model.Chapter, error) {
	c, err := s.store.FindBySlug(ctx, courseID, slug)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrChapterNotFound
	}
	return c, nil
}

func (s chapterService) Create(ctx context.Context, c *model.Chapter) error {
	return s.store.Create(ctx, c)
}

func (s chapterService) Update(ctx context.Context, c *model.Chapter) error {
	return s.store.Update(ctx, c)
}

func (s chapterService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
