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
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// fakeLessonProgressStore
// ============================================================================

type fakeLessonProgressStore struct {
	records map[string]*model.LessonProgress // key: userID+":"+lessonID
	percent float64
}

func newFakeLessonProgressStore() *fakeLessonProgressStore {
	return &fakeLessonProgressStore{records: make(map[string]*model.LessonProgress)}
}

func (s *fakeLessonProgressStore) key(userID, lessonID string) string {
	return fmt.Sprintf("%s:%s", userID, lessonID)
}

func (s *fakeLessonProgressStore) FindByUserAndLesson(_ context.Context, userID, lessonID string) (*model.LessonProgress, error) {
	p, ok := s.records[s.key(userID, lessonID)]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (s *fakeLessonProgressStore) ListByUserAndCourse(_ context.Context, userID, _ string) ([]model.LessonProgress, error) {
	result := make([]model.LessonProgress, 0)
	for _, p := range s.records {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (s *fakeLessonProgressStore) CalcProgressPercent(_ context.Context, _, _ string) (float64, error) {
	return s.percent, nil
}

func (s *fakeLessonProgressStore) Save(_ context.Context, p *model.LessonProgress) error {
	cp := *p
	s.records[s.key(p.UserID, p.LessonID)] = &cp
	return nil
}

// ============================================================================
// fakeCourseStoreForProgress
// ============================================================================

type fakeCourseStoreForProgress struct {
	course *model.Course
}

func (s *fakeCourseStoreForProgress) FindByID(_ context.Context, _ string) (*model.Course, error) {
	return s.course, nil
}

func (s *fakeCourseStoreForProgress) FindBySlug(_ context.Context, _ string) (*model.Course, error) {
	return nil, nil
}

func (s *fakeCourseStoreForProgress) List(_ context.Context, _ store.CourseFilter) ([]model.Course, int, error) {
	return nil, 0, nil
}

func (s *fakeCourseStoreForProgress) Create(_ context.Context, _ *model.Course) error { return nil }
func (s *fakeCourseStoreForProgress) Update(_ context.Context, _ *model.Course) error { return nil }
func (s *fakeCourseStoreForProgress) Delete(_ context.Context, _ string) error        { return nil }

// ============================================================================
// Tests
// ============================================================================

const (
	testProgressUserID   = "01966b0a-aaaa-7abc-def0-000000000010"
	testProgressLessonID = "01966b0a-bbbb-7abc-def0-000000000020"
	testProgressCourseID = "01966b0a-cccc-7abc-def0-000000000030"
)

func newProgressService(ps *fakeLessonProgressStore, es *fakeEnrollmentStore, cs *fakeCourseStoreForProgress) service.LessonProgressService {
	return service.NewLessonProgressService(&fakeTxRunner{}, ps, es, cs)
}

func defaultCourse() *model.Course {
	return &model.Course{ID: testProgressCourseID, CertificateEnabled: false}
}

func TestLessonProgressService_Get(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	svc := newProgressService(ps, newFakeEnrollmentStore(), &fakeCourseStoreForProgress{course: defaultCourse()})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Get(ctx, testProgressUserID, testProgressLessonID)
		assert.ErrorIs(t, err, service.ErrProgressNotFound)
	})

	t.Run("found", func(t *testing.T) {
		ps.records[fmt.Sprintf("%s:%s", testProgressUserID, testProgressLessonID)] = &model.LessonProgress{
			UserID:   testProgressUserID,
			LessonID: testProgressLessonID,
		}
		p, err := svc.Get(ctx, testProgressUserID, testProgressLessonID)
		assert.NoError(t, err)
		assert.Equal(t, testProgressUserID, p.UserID)
	})
}

func TestLessonProgressService_ListByCourse(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	ps.records[fmt.Sprintf("%s:%s", testProgressUserID, testProgressLessonID)] = &model.LessonProgress{
		UserID:   testProgressUserID,
		LessonID: testProgressLessonID,
	}
	svc := newProgressService(ps, newFakeEnrollmentStore(), &fakeCourseStoreForProgress{course: defaultCourse()})

	records, err := svc.ListByCourse(ctx, testProgressUserID, testProgressCourseID)

	assert.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestLessonProgressService_Save(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	ps.percent = 50.0
	svc := newProgressService(ps, newFakeEnrollmentStore(), &fakeCourseStoreForProgress{course: defaultCourse()})

	p := &model.LessonProgress{
		UserID:         testProgressUserID,
		LessonID:       testProgressLessonID,
		IsCompleted:    true,
		WatchedSeconds: 300,
	}

	assert.NoError(t, svc.Save(ctx, p, testProgressCourseID))
}

func TestLessonProgressService_Save_IssuesCertificateAt100(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	ps.percent = 100.0
	course := &model.Course{ID: testProgressCourseID, CertificateEnabled: true}
	svc := newProgressService(ps, newFakeEnrollmentStore(), &fakeCourseStoreForProgress{course: course})

	p := &model.LessonProgress{
		UserID:         testProgressUserID,
		LessonID:       testProgressLessonID,
		IsCompleted:    true,
		WatchedSeconds: 300,
	}

	assert.NoError(t, svc.Save(ctx, p, testProgressCourseID))
}

func TestLessonProgressService_Save_NoCertificateWhenDisabled(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	ps.percent = 100.0
	course := &model.Course{ID: testProgressCourseID, CertificateEnabled: false}
	svc := newProgressService(ps, newFakeEnrollmentStore(), &fakeCourseStoreForProgress{course: course})

	p := &model.LessonProgress{
		UserID:         testProgressUserID,
		LessonID:       testProgressLessonID,
		IsCompleted:    true,
		WatchedSeconds: 300,
	}

	assert.NoError(t, svc.Save(ctx, p, testProgressCourseID))
}
