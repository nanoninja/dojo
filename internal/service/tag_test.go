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
// fakeTagStore
// ============================================================================

type fakeTagStore struct {
	tags map[string]*model.Tag
	seq  int
}

func newFakeTagStore() *fakeTagStore {
	return &fakeTagStore{tags: make(map[string]*model.Tag)}
}

func (f *fakeTagStore) nextID() string {
	f.seq++
	return fmt.Sprintf("tag-%d", f.seq)
}

func (f *fakeTagStore) List(_ context.Context) ([]model.Tag, error) {
	result := make([]model.Tag, 0, len(f.tags))
	for _, t := range f.tags {
		result = append(result, *t)
	}
	return result, nil
}

func (f *fakeTagStore) FindByID(_ context.Context, id string) (*model.Tag, error) {
	t, ok := f.tags[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTagStore) FindBySlug(_ context.Context, slug string) (*model.Tag, error) {
	for _, t := range f.tags {
		if t.Slug == slug {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeTagStore) Create(_ context.Context, t *model.Tag) error {
	t.ID = f.nextID()
	cp := *t
	f.tags[t.ID] = &cp
	return nil
}

func (f *fakeTagStore) Update(_ context.Context, t *model.Tag) error {
	if _, ok := f.tags[t.ID]; !ok {
		return fmt.Errorf("tag not found")
	}
	cp := *t
	f.tags[t.ID] = &cp
	return nil
}

func (f *fakeTagStore) Delete(_ context.Context, id string) error {
	if _, ok := f.tags[id]; !ok {
		return fmt.Errorf("tag not found")
	}
	delete(f.tags, id)
	return nil
}

func newTagService(ts *fakeTagStore) service.TagService {
	return service.NewTagService(ts)
}

func TestTagService_Create(t *testing.T) {
	ctx := context.Background()
	svc := newTagService(newFakeTagStore())

	tag := &model.Tag{Name: "Go", Slug: "go"}

	assert.NoError(t, svc.Create(ctx, tag))
	assert.NotEqual(t, "", tag.ID, "Create() did not set ID")
}

func TestTagService_GetByID(t *testing.T) {
	ctx := context.Background()
	ts := newFakeTagStore()
	svc := newTagService(ts)

	tag := &model.Tag{Name: "Go", Slug: "go"}
	assert.NoError(t, svc.Create(ctx, tag))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, tag.ID)

		assert.NoError(t, err)
		assert.Equal(t, tag.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")

		assert.ErrorIs(t, err, service.ErrTagNotFound)
	})
}

func TestTagService_List(t *testing.T) {
	ctx := context.Background()
	svc := newTagService(newFakeTagStore())

	for _, name := range []string{"Go", "Python", "Rust"} {
		assert.NoError(t, svc.Create(ctx, &model.Tag{Name: name, Slug: name}))
	}

	tags, err := svc.List(ctx)

	assert.NoError(t, err)
	assert.Len(t, tags, 3)
}

func TestTagService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	svc := newTagService(newFakeTagStore())

	tag := &model.Tag{Name: "Go", Slug: "go"}
	assert.NoError(t, svc.Create(ctx, tag))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetBySlug(ctx, "go")
		assert.NoError(t, err)
		assert.Equal(t, tag.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrTagNotFound)
	})
}

func TestTagService_Update(t *testing.T) {
	ctx := context.Background()
	svc := newTagService(newFakeTagStore())

	tag := &model.Tag{Name: "Go", Slug: "go"}
	assert.NoError(t, svc.Create(ctx, tag))

	tag.Name = "Golang"
	assert.NoError(t, svc.Update(ctx, tag))

	got, err := svc.GetByID(ctx, tag.ID)

	assert.NoError(t, err)
	assert.Equal(t, "Golang", got.Name)
}

func TestTagService_Delete(t *testing.T) {
	ctx := context.Background()
	svc := newTagService(newFakeTagStore())

	tag := &model.Tag{Name: "Go", Slug: "go"}
	assert.NoError(t, svc.Create(ctx, tag))
	assert.NoError(t, svc.Delete(ctx, tag.ID))

	_, err := svc.GetByID(ctx, tag.ID)
	assert.ErrorIs(t, err, service.ErrTagNotFound)
}
