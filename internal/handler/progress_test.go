// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

// ============================================================================
// mockProgressService
// ============================================================================

type mockProgressService struct {
	progress *model.LessonProgress
	records  []model.LessonProgress
	getErr   error
	saveErr  error
}

func (m *mockProgressService) Get(_ context.Context, _, _ string) (*model.LessonProgress, error) {
	return m.progress, m.getErr
}

func (m *mockProgressService) ListByCourse(_ context.Context, _, _ string) ([]model.LessonProgress, error) {
	return m.records, m.getErr
}

func (m *mockProgressService) Save(_ context.Context, _ *model.LessonProgress, _ string) error {
	return m.saveErr
}

const (
	testProgressUserID   = "01966b0a-aaaa-7abc-def0-000000000010"
	testProgressLessonID = "01966b0a-bbbb-7abc-def0-000000000020"
	testProgressCourseID = "01966b0a-cccc-7abc-def0-000000000030"
)

func newProgressHandler(s *mockProgressService) *handler.ProgressHandler {
	return handler.NewProgressHandler(s)
}

// ============================================================================
// Get
// ============================================================================

func TestProgressHandler_Get(t *testing.T) {
	s := &mockProgressService{progress: &model.LessonProgress{
		UserID:   testProgressUserID,
		LessonID: testProgressLessonID,
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/"+testProgressUserID+"/lessons/"+testProgressLessonID, nil)
	req = withChiParam(req, "user_id", testProgressUserID)
	req = withChiParam(req, "lesson_id", testProgressLessonID)

	serve(newProgressHandler(s).Get, rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestProgressHandler_Get_Found(t *testing.T) {
	s := &mockProgressService{progress: &model.LessonProgress{
		UserID:   testProgressUserID,
		LessonID: testProgressLessonID,
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/"+testProgressUserID+"/lessons/"+testProgressLessonID, nil)
	req = withChiParam(req, "user_id", testProgressUserID)
	req = withChiParam(req, "lesson_id", testProgressLessonID)

	serve(newProgressHandler(s).Get, rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestProgressHandler_Get_NotFound(t *testing.T) {
	s := &mockProgressService{getErr: service.ErrProgressNotFound}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/"+testProgressUserID+"/lessons/"+testProgressLessonID, nil)
	req = withChiParam(req, "user_id", testProgressUserID)
	req = withChiParam(req, "lesson_id", testProgressLessonID)

	serve(newProgressHandler(s).Get, rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestProgressHandler_Get_InvalidUserID(t *testing.T) {
	s := &mockProgressService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/bad/lessons/"+testProgressLessonID, nil)
	req = withChiParam(req, "user_id", "bad")
	req = withChiParam(req, "lesson_id", testProgressLessonID)

	serve(newProgressHandler(s).Get, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProgressHandler_Get_InvalidLessonID(t *testing.T) {
	s := &mockProgressService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/"+testProgressUserID+"/lessons/bad", nil)
	req = withChiParam(req, "user_id", testProgressUserID)
	req = withChiParam(req, "lesson_id", "bad")

	serve(newProgressHandler(s).Get, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// ListByCourse
// ============================================================================

func TestProgressHandler_ListByCourse(t *testing.T) {
	s := &mockProgressService{records: []model.LessonProgress{
		{UserID: testProgressUserID, LessonID: testProgressLessonID},
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/"+testProgressUserID+"/courses/"+testProgressCourseID, nil)
	req = withChiParam(req, "user_id", testProgressUserID)
	req = withChiParam(req, "course_id", testProgressCourseID)

	serve(newProgressHandler(s).ListByCourse, rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	decodeJSON(t, rec, &body)
	assert.Len(t, body, 1)
}

func TestProgressHandler_ListByCourse_InvalidCourseID(t *testing.T) {
	s := &mockProgressService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/progress/"+testProgressUserID+"/courses/bad", nil)
	req = withChiParam(req, "user_id", testProgressUserID)
	req = withChiParam(req, "course_id", "bad")

	serve(newProgressHandler(s).ListByCourse, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// Save
// ============================================================================

func TestProgressHandler_Save(t *testing.T) {
	s := &mockProgressService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/progress", map[string]any{
		"user_id":         testProgressUserID,
		"lesson_id":       testProgressLessonID,
		"course_id":       testProgressCourseID,
		"is_completed":    true,
		"watched_seconds": 300,
	})

	serve(newProgressHandler(s).Save, rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestProgressHandler_Save_InvalidBody(t *testing.T) {
	s := &mockProgressService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/progress", map[string]any{
		"user_id": "bad-uuid",
	})

	serve(newProgressHandler(s).Save, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
