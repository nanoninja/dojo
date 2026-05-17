// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// fakeCourseStore
// ============================================================================

type fakeCourseStore struct {
	courses map[string]*model.Course
	seq     int
}

func newFakeCourseStore() *fakeCourseStore {
	return &fakeCourseStore{courses: make(map[string]*model.Course)}
}

func (f *fakeCourseStore) nextID() string {
	f.seq++
	return fmt.Sprintf("course-%d", f.seq)
}

func (f *fakeCourseStore) List(_ context.Context, _ store.CourseFilter) ([]model.Course, int, error) {
	result := make([]model.Course, 0, len(f.courses))
	for _, c := range f.courses {
		if c.DeletedAt == nil {
			result = append(result, *c)
		}
	}
	return result, len(result), nil
}

func (f *fakeCourseStore) FindByID(_ context.Context, id string) (*model.Course, error) {
	c, ok := f.courses[id]
	if !ok || c.DeletedAt != nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (f *fakeCourseStore) FindBySlug(_ context.Context, slug string) (*model.Course, error) {
	for _, c := range f.courses {
		if c.Slug == slug && c.DeletedAt == nil {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeCourseStore) Create(_ context.Context, c *model.Course) error {
	c.ID = f.nextID()
	cp := *c
	f.courses[c.ID] = &cp
	return nil
}

func (f *fakeCourseStore) Update(_ context.Context, c *model.Course) error {
	if _, ok := f.courses[c.ID]; !ok {
		return fmt.Errorf("course not found")
	}
	cp := *c
	f.courses[c.ID] = &cp
	return nil
}

func (f *fakeCourseStore) Delete(_ context.Context, id string) error {
	c, ok := f.courses[id]
	if !ok {
		return fmt.Errorf("course not found")
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

// ============================================================================
// fakeCoursesCategoriesStore
// ============================================================================

type fakeCoursesCategoriesStore struct {
	assignments []model.CategoryAssignment
}

func (f *fakeCoursesCategoriesStore) List(_ context.Context, courseID string) ([]model.CategoryAssignment, error) {
	result := make([]model.CategoryAssignment, 0)
	for _, a := range f.assignments {
		if a.CourseID == courseID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *fakeCoursesCategoriesStore) Assign(_ context.Context, courseID, categoryID string, isPrimary bool) error {
	f.assignments = append(f.assignments, model.CategoryAssignment{
		CourseID:   courseID,
		CategoryID: categoryID,
		IsPrimary:  isPrimary,
	})
	return nil
}

func (f *fakeCoursesCategoriesStore) Unassign(_ context.Context, courseID, categoryID string) error {
	result := f.assignments[:0]
	for _, a := range f.assignments {
		if a.CourseID != courseID || a.CategoryID != categoryID {
			result = append(result, a)
		}
	}
	f.assignments = result
	return nil
}

func (f *fakeCoursesCategoriesStore) SetPrimary(_ context.Context, courseID, categoryID string) error {
	for i := range f.assignments {
		if f.assignments[i].CourseID == courseID {
			f.assignments[i].IsPrimary = f.assignments[i].CategoryID == categoryID
		}
	}
	return nil
}

// ============================================================================
// fakeCoursesTagsStore
// ============================================================================

type fakeCoursesTagsStore struct {
	assignments []model.CourseTagAssignment
}

func (f *fakeCoursesTagsStore) List(_ context.Context, courseID string) ([]model.CourseTagAssignment, error) {
	result := make([]model.CourseTagAssignment, 0)
	for _, a := range f.assignments {
		if a.CourseID == courseID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *fakeCoursesTagsStore) Assign(_ context.Context, courseID, tagID string) error {
	f.assignments = append(f.assignments, model.CourseTagAssignment{CourseID: courseID, TagID: tagID})
	return nil
}

func (f *fakeCoursesTagsStore) Unassign(_ context.Context, courseID, tagID string) error {
	result := f.assignments[:0]
	for _, a := range f.assignments {
		if a.CourseID != courseID || a.TagID != tagID {
			result = append(result, a)
		}
	}
	f.assignments = result
	return nil
}

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

	courses, total, err := svc.List(ctx, store.CourseFilter{Limit: 10})
	assert.NoError(t, err)
	assert.Len(t, courses, 3)
	assert.Equal(t, 3, total)
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
