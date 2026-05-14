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

const testChapterID = "01966b0a-1111-7abc-def0-aaaaaaaaaaaa"

func newLessonService(ls *fakeLessonStore, rs *fakeLessonResourceStore) service.LessonService {
	return service.NewLessonService(ls, rs)
}

func TestLessonService_Create(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))
	assert.NotEqual(t, "", l.ID, "Create() did not set ID")
}

func TestLessonService_GetByID(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, l.ID)
		assert.NoError(t, err)
		assert.Equal(t, l.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrLessonNotFound)
	})
}

func TestLessonService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetBySlug(ctx, testChapterID, "variables")
		assert.NoError(t, err)
		assert.Equal(t, l.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, testChapterID, "non-existent")
		assert.ErrorIs(t, err, service.ErrLessonNotFound)
	})

	t.Run("wrong chapter", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "other-chapter-id", "variables")
		assert.ErrorIs(t, err, service.ErrLessonNotFound)
	})
}

func TestLessonService_List(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	for i, slug := range []string{"intro", "variables", "functions"} {
		assert.NoError(t, svc.Create(ctx, &model.Lesson{
			ChapterID:   testChapterID,
			Title:       slug,
			Slug:        slug,
			ContentType: model.ContentTypeVideo,
			SortOrder:   int16(i + 1),
		}))
	}

	t.Run("returns lessons for chapter", func(t *testing.T) {
		lessons, err := svc.List(ctx, testChapterID)
		assert.NoError(t, err)
		assert.Len(t, lessons, 3)
	})

	t.Run("returns empty for unknown chapter", func(t *testing.T) {
		lessons, err := svc.List(ctx, "unknown-chapter")
		assert.NoError(t, err)
		assert.Len(t, lessons, 0)
	})
}

func TestLessonService_Update(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	l.Title = "Variables & Types"
	assert.NoError(t, svc.Update(ctx, l))

	got, err := svc.GetByID(ctx, l.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Variables & Types", got.Title)
}

func TestLessonService_Delete(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))
	assert.NoError(t, svc.Delete(ctx, l.ID))

	_, err := svc.GetByID(ctx, l.ID)
	assert.ErrorIs(t, err, service.ErrLessonNotFound)
}

func TestLessonService_AddResource(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	res := &model.LessonResource{LessonID: l.ID, Title: "Slides", FileURL: "https://example.com/slides.pdf", FileName: "slides.pdf"}
	assert.NoError(t, svc.AddResource(ctx, res))
	assert.NotEqual(t, "", res.ID, "AddResource() did not set ID")
}

func TestLessonService_ListResources(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	for _, name := range []string{"Slides", "Code", "Cheatsheet"} {
		assert.NoError(t, svc.AddResource(ctx, &model.LessonResource{
			LessonID: l.ID,
			Title:    name,
			FileURL:  "https://example.com/" + name,
			FileName: name + ".pdf",
		}))
	}

	resources, err := svc.ListResources(ctx, l.ID)
	assert.NoError(t, err)
	assert.Len(t, resources, 3)
}

func TestLessonService_GetResourceByID(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	res := &model.LessonResource{LessonID: l.ID, Title: "Slides", FileURL: "https://example.com/slides.pdf", FileName: "slides.pdf"}
	assert.NoError(t, svc.AddResource(ctx, res))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetResourceByID(ctx, res.ID)
		assert.NoError(t, err)
		assert.Equal(t, res.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetResourceByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrLessonResourceNotFound)
	})
}

func TestLessonService_UpdateResource(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	res := &model.LessonResource{LessonID: l.ID, Title: "Slides", FileURL: "https://example.com/slides.pdf", FileName: "slides.pdf"}
	assert.NoError(t, svc.AddResource(ctx, res))

	res.Title = "Updated Slides"
	assert.NoError(t, svc.UpdateResource(ctx, res))

	got, err := svc.GetResourceByID(ctx, res.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Slides", got.Title)
}

func TestLessonService_RemoveResource(t *testing.T) {
	ctx := context.Background()
	svc := newLessonService(newFakeLessonStore(), newFakeLessonResourceStore())

	l := &model.Lesson{ChapterID: testChapterID, Title: "Variables", Slug: "variables", ContentType: model.ContentTypeVideo}
	assert.NoError(t, svc.Create(ctx, l))

	res := &model.LessonResource{LessonID: l.ID, Title: "Slides", FileURL: "https://example.com/slides.pdf", FileName: "slides.pdf"}
	assert.NoError(t, svc.AddResource(ctx, res))
	assert.NoError(t, svc.RemoveResource(ctx, res.ID))

	_, err := svc.GetResourceByID(ctx, res.ID)
	assert.ErrorIs(t, err, service.ErrLessonResourceNotFound)
}
