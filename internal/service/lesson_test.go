// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

// ============================================================================
// fakeLessonStore
// ============================================================================

type fakeLessonStore struct {
	lessons map[string]*model.Lesson
	seq     int
}

func newFakeLessonStore() *fakeLessonStore {
	return &fakeLessonStore{lessons: make(map[string]*model.Lesson)}
}

func (f *fakeLessonStore) nextID() string {
	f.seq++
	return fmt.Sprintf("lesson-%d", f.seq)
}

func (f *fakeLessonStore) List(_ context.Context, chapterID string) ([]model.Lesson, error) {
	result := make([]model.Lesson, 0)
	for _, l := range f.lessons {
		if l.ChapterID == chapterID {
			result = append(result, *l)
		}
	}
	return result, nil
}

func (f *fakeLessonStore) FindByID(_ context.Context, id string) (*model.Lesson, error) {
	l, ok := f.lessons[id]
	if !ok {
		return nil, nil
	}
	cp := *l
	return &cp, nil
}

func (f *fakeLessonStore) FindBySlug(_ context.Context, chapterID, slug string) (*model.Lesson, error) {
	for _, l := range f.lessons {
		if l.ChapterID == chapterID && l.Slug == slug {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeLessonStore) Create(_ context.Context, l *model.Lesson) error {
	l.ID = f.nextID()
	cp := *l
	f.lessons[l.ID] = &cp
	return nil
}

func (f *fakeLessonStore) Update(_ context.Context, l *model.Lesson) error {
	if _, ok := f.lessons[l.ID]; !ok {
		return fmt.Errorf("lesson not found")
	}
	cp := *l
	f.lessons[l.ID] = &cp
	return nil
}

func (f *fakeLessonStore) Delete(_ context.Context, id string) error {
	if _, ok := f.lessons[id]; !ok {
		return fmt.Errorf("lesson not found")
	}
	delete(f.lessons, id)
	return nil
}

// ============================================================================
// fakeLessonResourceStore
// ============================================================================

type fakeLessonResourceStore struct {
	resources map[string]*model.LessonResource
	seq       int
}

func newFakeLessonResourceStore() *fakeLessonResourceStore {
	return &fakeLessonResourceStore{resources: make(map[string]*model.LessonResource)}
}

func (f *fakeLessonResourceStore) nextID() string {
	f.seq++
	return fmt.Sprintf("res-%d", f.seq)
}

func (f *fakeLessonResourceStore) List(_ context.Context, lessonID string) ([]model.LessonResource, error) {
	result := make([]model.LessonResource, 0)
	for _, r := range f.resources {
		if r.LessonID == lessonID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (f *fakeLessonResourceStore) FindByID(_ context.Context, id string) (*model.LessonResource, error) {
	r, ok := f.resources[id]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (f *fakeLessonResourceStore) Create(_ context.Context, r *model.LessonResource) error {
	r.ID = f.nextID()
	cp := *r
	f.resources[r.ID] = &cp
	return nil
}

func (f *fakeLessonResourceStore) Update(_ context.Context, r *model.LessonResource) error {
	if _, ok := f.resources[r.ID]; !ok {
		return fmt.Errorf("resource not found")
	}
	cp := *r
	f.resources[r.ID] = &cp
	return nil
}

func (f *fakeLessonResourceStore) Delete(_ context.Context, id string) error {
	if _, ok := f.resources[id]; !ok {
		return fmt.Errorf("resource not found")
	}
	delete(f.resources, id)
	return nil
}

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
