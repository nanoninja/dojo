// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

var errTx = errors.New("tx error")

func newCourseService(
	tx *fakeTxRunner,
	cs *fakeCourseStore,
	cats *fakeCoursesCategoriesStore,
	tags *fakeCoursesTagsStore,
) service.CourseService {
	return service.NewCourseService(tx, cs, cats, tags)
}

func baseCourse() *model.Course {
	return &model.Course{
		InstructorID: "inst-1",
		Slug:         "go-fundamentals",
		Title:        "Go Fundamentals",
		Level:        model.CourseLevelBeginner,
		ContentType:  model.ContentTypeVideo,
		Language:     "en",
		Currency:     "USD",
	}
}

func TestCourseService_GetByID(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCourseStore()
	svc := newCourseService(&fakeTxRunner{}, cs, &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	c := baseCourse()
	assert.NoError(t, cs.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, c.ID)
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrCourseNotFound)
	})
}

func TestCourseService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCourseStore()
	svc := newCourseService(&fakeTxRunner{}, cs, &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	c := baseCourse()
	assert.NoError(t, cs.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetBySlug(ctx, "go-fundamentals")
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrCourseNotFound)
	})
}

func TestCourseService_List(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCourseStore()
	svc := newCourseService(&fakeTxRunner{}, cs, &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	for _, slug := range []string{"course-a", "course-b", "course-c"} {
		c := baseCourse()
		c.Slug = slug
		assert.NoError(t, cs.Create(ctx, c))
	}

	courses, err := svc.List(ctx, store.CourseFilter{Limit: 10})
	assert.NoError(t, err)
	assert.Len(t, courses, 3)
}

func TestCourseService_Update(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCourseStore()
	svc := newCourseService(&fakeTxRunner{}, cs, &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	c := baseCourse()
	assert.NoError(t, cs.Create(ctx, c))

	c.Title = "Go Fundamentals v2"
	assert.NoError(t, svc.Update(ctx, c))

	got, err := svc.GetByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Go Fundamentals v2", got.Title)
}

func TestCourseService_Delete(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCourseStore()
	svc := newCourseService(&fakeTxRunner{}, cs, &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	c := baseCourse()
	assert.NoError(t, cs.Create(ctx, c))
	assert.NoError(t, svc.Delete(ctx, c.ID))

	_, err := svc.GetByID(ctx, c.ID)
	assert.ErrorIs(t, err, service.ErrCourseNotFound)
}

func TestCourseService_Create_TxError(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTxRunner{err: errTx}
	svc := newCourseService(tx, newFakeCourseStore(), &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	err := svc.Create(ctx, baseCourse(), []string{"cat-1"}, "cat-1", []string{"tag-1"})
	assert.Error(t, err)
}

func TestCourseService_SetCategories_TxError(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTxRunner{err: errTx}
	svc := newCourseService(tx, newFakeCourseStore(), &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	err := svc.SetCategories(ctx, "course-1", []string{"cat-1"}, "cat-1")
	assert.Error(t, err)
}

func TestCourseService_SetTags_TxError(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTxRunner{err: errTx}
	svc := newCourseService(tx, newFakeCourseStore(), &fakeCoursesCategoriesStore{}, &fakeCoursesTagsStore{})

	err := svc.SetTags(ctx, "course-1", []string{"tag-1"})
	assert.Error(t, err)
}
