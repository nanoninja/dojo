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

func newTestCategory() *model.Category {
	return &model.Category{
		Name:      "Backend",
		Slug:      "backend",
		IsVisible: true,
	}
}

func TestCategoryStore_Create(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	c := newTestCategory()

	assert.NoError(t, s.Create(context.Background(), c))
	assert.NotEqual(t, "", c.ID, "Create() did not set ID")
}

func TestCategoryStore_FindByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	ctx := context.Background()

	c := newTestCategory()
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindByID(ctx, c.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, c.Name, found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindByID(ctx, "00000000-0000-0000-0000-000000000000")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestCategoryStore_FindBySlug(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	ctx := context.Background()

	c := newTestCategory()
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	t.Run("found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, "backend")

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, c.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := s.FindBySlug(ctx, "unknow")

		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestCategoryStore_List(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	ctx := context.Background()

	for _, name := range []string{"Backend", "Frontend", "DevOps"} {
		assert.NoError(t, s.Create(ctx, &model.Category{Name: name, Slug: name, IsVisible: true}))
	}

	categories, err := s.List(ctx)

	assert.NoError(t, err)
	assert.Len(t, categories, 3)
}

func TestCategoryStore_Update(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	ctx := context.Background()

	c := newTestCategory()
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")

	c.Name = "Backend Development"
	c.Slug = "backend-development"
	assert.NoError(t, s.Update(ctx, c))

	found, err := s.FindByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Backend Development", found.Name)
}

func TestCategoryStore_Delete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	ctx := context.Background()

	c := newTestCategory()
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, c.ID))

	// Soft delete — FindByID should return nil
	found, err := s.FindByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Nil(t, found, "category should not be findable after soft Delete()")
}

func TestCategoryStore_Delete_HiddenFromList(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "categories")

	s := store.NewCategoryStore(db)
	ctx := context.Background()

	c := newTestCategory()
	assert.NoError(t, s.Create(ctx, c), "setup: Create()")
	assert.NoError(t, s.Delete(ctx, c.ID))

	categories, err := s.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, categories, 0, "deleted category should not appear in List()")
}
