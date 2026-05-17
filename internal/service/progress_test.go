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
// Tests
// ============================================================================

const (
	testProgressUserID   = "01966b0a-aaaa-7abc-def0-000000000010"
	testProgressLessonID = "01966b0a-bbbb-7abc-def0-000000000020"
	testProgressCourseID = "01966b0a-cccc-7abc-def0-000000000030"
)

func newProgressService(ps *fakeLessonProgressStore, es *fakeEnrollmentStore) service.LessonProgressService {
	return service.NewLessonProgressService(&fakeTxRunner{}, ps, es)
}

func TestLessonProgressService_Get(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	svc := newProgressService(ps, newFakeEnrollmentStore())

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
	svc := newProgressService(ps, newFakeEnrollmentStore())

	records, err := svc.ListByCourse(ctx, testProgressUserID, testProgressCourseID)

	assert.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestLessonProgressService_Save(t *testing.T) {
	ctx := context.Background()
	ps := newFakeLessonProgressStore()
	ps.percent = 50.0
	svc := newProgressService(ps, newFakeEnrollmentStore())

	p := &model.LessonProgress{
		UserID:         testProgressUserID,
		LessonID:       testProgressLessonID,
		IsCompleted:    true,
		WatchedSeconds: 300,
	}

	assert.NoError(t, svc.Save(ctx, p, testProgressCourseID))
}
