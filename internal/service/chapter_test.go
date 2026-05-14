// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

const testCourseID = "01966b0a-0000-7abc-def0-000000000000"

func newChapterService(cs *fakeChapterStore) service.ChapterService {
	return service.NewChapterService(cs)
}

func TestChapterService_Create(t *testing.T) {
	ctx := context.Background()
	svc := newChapterService(newFakeChapterStore())

	c := &model.Chapter{CourseID: testCourseID, Title: "Introduction", Slug: "introduction"}
	assert.NoError(t, svc.Create(ctx, c))
	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
}

func TestChapterService_GetByID(t *testing.T) {
	ctx := context.Background()
	svc := newChapterService(newFakeChapterStore())

	c := &model.Chapter{CourseID: testCourseID, Title: "Introduction", Slug: "introduction"}
	assert.NoError(t, svc.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, c.ID)
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrChapterNotFound)
	})
}

func TestChapterService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	svc := newChapterService(newFakeChapterStore())

	c := &model.Chapter{CourseID: testCourseID, Title: "Introduction", Slug: "introduction"}
	assert.NoError(t, svc.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetBySlug(ctx, testCourseID, "introduction")
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, testCourseID, "non-existent")
		assert.ErrorIs(t, err, service.ErrChapterNotFound)
	})

	t.Run("wrong course", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "other-course-id", "introduction")
		assert.ErrorIs(t, err, service.ErrChapterNotFound)
	})
}

func TestChapterService_List(t *testing.T) {
	ctx := context.Background()
	svc := newChapterService(newFakeChapterStore())

	for i, slug := range []string{"intro", "basics", "advanced"} {
		assert.NoError(t, svc.Create(ctx, &model.Chapter{
			CourseID:  testCourseID,
			Title:     slug,
			Slug:      slug,
			SortOrder: int16(i + 1),
		}))
	}

	t.Run("returns chapters for course", func(t *testing.T) {
		chapters, err := svc.List(ctx, testCourseID)
		assert.NoError(t, err)
		assert.Len(t, chapters, 3)
	})

	t.Run("returns empty for unknown course", func(t *testing.T) {
		chapters, err := svc.List(ctx, "unknown-course")
		assert.NoError(t, err)
		assert.Len(t, chapters, 0)
	})
}

func TestChapterService_Update(t *testing.T) {
	ctx := context.Background()
	svc := newChapterService(newFakeChapterStore())

	c := &model.Chapter{CourseID: testCourseID, Title: "Introduction", Slug: "introduction"}
	assert.NoError(t, svc.Create(ctx, c))

	c.Title = "Introduction Updated"
	assert.NoError(t, svc.Update(ctx, c))

	got, err := svc.GetByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Introduction Updated", got.Title)
}

func TestChapterService_Delete(t *testing.T) {
	ctx := context.Background()
	svc := newChapterService(newFakeChapterStore())

	c := &model.Chapter{CourseID: testCourseID, Title: "Introduction", Slug: "introduction"}
	assert.NoError(t, svc.Create(ctx, c))
	assert.NoError(t, svc.Delete(ctx, c.ID))

	_, err := svc.GetByID(ctx, c.ID)
	assert.ErrorIs(t, err, service.ErrChapterNotFound)
}
