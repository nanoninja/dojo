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
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// mockEnrollmentService
// ============================================================================

type mockEnrollmentService struct {
	enrollment  *model.CourseEnrollment
	enrollments []model.CourseEnrollment
	getErr      error
	enrollErr   error
	updateErr   error
	deleteErr   error
}

func (m *mockEnrollmentService) List(_ context.Context, _ store.EnrollmentFilter) ([]model.CourseEnrollment, error) {
	return m.enrollments, m.getErr
}

func (m *mockEnrollmentService) GetByID(_ context.Context, _ string) (*model.CourseEnrollment, error) {
	return m.enrollment, m.getErr
}

func (m *mockEnrollmentService) Enroll(_ context.Context, userID, courseID string) (*model.CourseEnrollment, error) {
	if m.enrollErr != nil {
		return nil, m.enrollErr
	}
	return &model.CourseEnrollment{
		ID:       "01966b0a-eeee-7abc-def0-000000000099",
		UserID:   userID,
		CourseID: courseID,
		Status:   model.EnrollmentStatusActive,
	}, nil
}

func (m *mockEnrollmentService) UpdateStatus(_ context.Context, _ string, _ model.EnrollmentStatus) error {
	return m.updateErr
}

func (m *mockEnrollmentService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

const (
	testEnrollmentID          = "01966b0a-eeee-7abc-def0-000000000099"
	testCourseIDForEnrollment = "01966b0a-ffff-7abc-def0-000000000006"
)

func newEnrollmentHandler(s *mockEnrollmentService) *handler.EnrollmentHandler {
	return handler.NewEnrollmentHandler(s)
}

func TestEnrollmentHandler_List(t *testing.T) {
	s := &mockEnrollmentService{enrollments: []model.CourseEnrollment{
		{
			ID:       testEnrollmentID,
			UserID:   testUser1ID,
			CourseID: testCourseIDForEnrollment,
			Status:   model.EnrollmentStatusActive,
		},
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollments", nil)
	serve(newEnrollmentHandler(s).List, rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	decodeJSON(t, rec, &body)
	assert.Len(t, body, 1)
}

func TestEnrollmentHandler_List_InvalidUserID(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollment?user_id=bad-uuid", nil)

	serve(newEnrollmentHandler(s).List, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnrollmentHandler_List_InvalidCourseID(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollments?course_id=bad-uuid", nil)

	serve(newEnrollmentHandler(s).List, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnrollmentHandler_List_InvalidStatus(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollments?status=unknown", nil)

	serve(newEnrollmentHandler(s).List, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnrollmentHandler_GetByID_Found(t *testing.T) {
	s := &mockEnrollmentService{enrollment: &model.CourseEnrollment{
		ID:     testEnrollmentID,
		Status: model.EnrollmentStatusActive,
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollments/"+testEnrollmentID, nil)

	serve(newEnrollmentHandler(s).GetByID, rec, withChiParam(req, "id", testEnrollmentID))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestEnrollmentHandler_GetByID_NotFound(t *testing.T) {
	s := &mockEnrollmentService{getErr: service.ErrEnrollmentNotFound}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollments/"+testEnrollmentID, nil)

	serve(newEnrollmentHandler(s).GetByID, rec, withChiParam(req, "id", testEnrollmentID))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestEnrollmentHandler_GetByID_InvalidUUID(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/enrollments/bad", nil)

	serve(newEnrollmentHandler(s).GetByID, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnrollmentHandler_Enroll(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/enrollments", map[string]any{
		"user_id":   testUserID,
		"course_id": testCourseIDForEnrollment,
	})

	serve(newEnrollmentHandler(s).Enroll, rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var body map[string]any
	decodeJSON(t, rec, &body)
	assert.Equal(t, string(model.EnrollmentStatusActive), body["status"].(string))
}

func TestEnrollmentHandler_Enroll_InvalidBody(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/enrollments", map[string]any{"user_id": "bad"})

	serve(newEnrollmentHandler(s).Enroll, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnrollmentHandler_Enroll_AlreadyEnrolled(t *testing.T) {
	s := &mockEnrollmentService{enrollErr: service.ErrAlreadyEnrolled}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/enrollments", map[string]any{
		"user_id":   testUserID,
		"course_id": testCourseIDForEnrollment,
	})

	serve(newEnrollmentHandler(s).Enroll, rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestEnrollmentHandler_UpdateStatus(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PATCH", "/enrollments/"+testEnrollmentID+"/status", map[string]any{
		"status": "completed",
	})

	serve(newEnrollmentHandler(s).UpdateStatus, rec, withChiParam(req, "id", testEnrollmentID))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestEnrollmentHandler_UpdateStatus_InvalidBody(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PATCH", "/enrollments/"+testEnrollmentID+"/status", map[string]any{
		"status": "invalid",
	})

	serve(newEnrollmentHandler(s).UpdateStatus, rec, withChiParam(req, "id", testEnrollmentID))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnrollmentHandler_UpdateStatus_NotFound(t *testing.T) {
	s := &mockEnrollmentService{updateErr: service.ErrEnrollmentNotFound}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PATCH", "/enrollments/"+testEnrollmentID+"/status", map[string]any{
		"status": "completed",
	})

	serve(newEnrollmentHandler(s).UpdateStatus, rec, withChiParam(req, "id", testEnrollmentID))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestEnrollmentHandler_Delete(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/enrollments/"+testEnrollmentID, nil)

	serve(newEnrollmentHandler(s).Delete, rec, withChiParam(req, "id", testEnrollmentID))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestEnrollmentHandler_Delete_InvalidUUID(t *testing.T) {
	s := &mockEnrollmentService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/enrollments/bad", nil)

	serve(newEnrollmentHandler(s).Delete, rec, withChiParam(req, "id", "bad"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
