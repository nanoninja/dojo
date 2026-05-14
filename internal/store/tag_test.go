// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

func newTestTag() *model.Tag {
	return &model.Tag{Name: "Go", Slug: "go"}
}

func TestTagStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)

	s := store.NewTagStore(db)
	tag := newTestTag()

	assert.NoError(t, s.Create(context.Background(), tag))
	assert.NotEqual(t, "", tag.ID, "Create() did not set ID")
}

func TestTagStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "tags")

	s := store.NewTagStore(db)
	ctx := context.Background()

	tag := newTestTag()
	assert.NoError(t, s.Create(ctx, tag), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, tag.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tag.Name, found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestTagStore_FindBySlug(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "tags")

	s := store.NewTagStore(db)
	ctx := context.Background()

	tag := newTestTag()
	assert.NoError(t, s.Create(ctx, tag), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, "go")

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tag.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, "unknown")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestTagsStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "tags")

	s := store.NewTagStore(db)
	ctx := context.Background()

	for _, name := range []string{"Go", "Python", "Rust"} {
		assert.NoError(t, s.Create(ctx, &model.Tag{Name: name, Slug: name}), "setup: Create()")
	}

	tags, err := s.List(ctx)

	assert.NoError(t, err)
	assert.Len(t, tags, 3)
}

func TestTagStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "tags")

	s := store.NewTagStore(db)
	ctx := context.Background()

	tag := newTestTag()
	assert.NoError(t, s.Create(ctx, tag), "setup: Create()")

	tag.Name = "Golang"
	tag.Slug = "golang"

	assert.NoError(t, s.Update(ctx, tag))

	found, err := s.FindByID(ctx, tag.ID)

	assert.NoError(t, err)
	assert.Equal(t, "Golang", found.Name)
}

func TestTagStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "tags")

	s := store.NewTagStore(db)
	ctx := context.Background()

	tag := newTestTag()
	assert.NoError(t, s.Create(ctx, tag), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, tag.ID))

	found, err := s.FindByID(ctx, tag.ID)

	assert.NoError(t, err, "setup: Create()")
	assert.Nil(t, found, "tag should not be findable after Delete()")
}
