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
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// mockCourseService
// ============================================================================

type mockCourseService struct {
	course           *model.Course
	courses          []model.Course
	getErr           error
	createErr        error
	updateErr        error
	deleteErr        error
	setCategoriesErr error
	setTagsErr       error
}

func (m *mockCourseService) List(_ context.Context, _ store.CourseFilter) ([]model.Course, int, error) {
	return m.courses, len(m.courses), m.getErr
}

func (m *mockCourseService) GetByID(_ context.Context, _ string) (*model.Course, error) {
	return m.course, m.getErr
}

func (m *mockCourseService) GetBySlug(_ context.Context, _ string) (*model.Course, error) {
	return m.course, m.getErr
}

func (m *mockCourseService) Create(_ context.Context, c *model.Course, _ []string, _ string, _ []string) error {
	c.ID = "01966b0a-ffff-7abc-def0-000000000006"
	return m.createErr
}

func (m *mockCourseService) Update(_ context.Context, _ *model.Course) error {
	return m.updateErr
}

func (m *mockCourseService) SetCategories(_ context.Context, _ string, _ []string, _ string) error {
	return m.setCategoriesErr
}

func (m *mockCourseService) SetTags(_ context.Context, _ string, _ []string) error {
	return m.setTagsErr
}

func (m *mockCourseService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

const testCourseID = "01966b0a-ffff-7abc-def0-000000000006"

func newCourseHandler(cs *mockCourseService) *handler.CourseHandler {
	return handler.NewCourseHandler(cs)
}

func TestCourseHandler_List(t *testing.T) {
	ms := &mockCourseService{courses: []model.Course{
		{
			ID:    testCourseID,
			Title: "Go Fundamentals",
			Slug:  "go-fundamentals",
		},
	}}
	w := httptest.NewRecorder()
	serve(newCourseHandler(ms).List, w, httptest.NewRequest("GET", "/courses", nil))

	require.Equal(t, http.StatusOK, w.Code)

	var body httputil.PageResponse[model.Course]
	decodeJSON(t, w, &body)
	assert.Len(t, body.Data, 1)
	assert.Equal(t, 1, body.Meta.Total)
}

func TestCourseHandler_List_InvalidLevel(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	serve(newCourseHandler(ms).List, w, httptest.NewRequest("GET", "/courses?level=invalid", nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCourseHandler_List_InvalidInstructorUUID(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	serve(newCourseHandler(ms).List, w, httptest.NewRequest("GET", "/courses?instructor_id=bad", nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCourseHandler_GetByID_Found(t *testing.T) {
	ms := &mockCourseService{course: &model.Course{ID: testCourseID, Title: "Go Fundamentals"}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/courses/"+testCourseID, nil), "id", testCourseID)
	serve(newCourseHandler(ms).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCourseHandler_GetByID_NotFound(t *testing.T) {
	ms := &mockCourseService{getErr: service.ErrCourseNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/courses/"+testCourseID, nil), "id", testCourseID)
	serve(newCourseHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCourseHandler_GetByID_InvalidUUID(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/courses/bad", nil), "id", "bad")
	serve(newCourseHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCourseHandler_Create(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/courses", map[string]any{
		"instructor_id": testUserID,
		"slug":          "go-fundamentals",
		"title":         "Go Fundamentals",
		"level":         "beginner",
		"content_type":  "video",
		"language":      "en",
		"currency":      "USD",
	})
	serve(newCourseHandler(ms).Create, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "Go Fundamentals", body["title"])
}

func TestCourseHandler_Create_InvalidBody(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/courses", map[string]any{"title": ""})
	serve(newCourseHandler(ms).Create, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCourseHandler_Update(t *testing.T) {
	ms := &mockCourseService{course: &model.Course{ID: testCourseID, Title: "Go Fundamentals"}}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/courses/"+testCourseID, map[string]any{
		"slug":         "go-fundamentals-v2",
		"title":        "Go Fundamentals v2",
		"level":        "beginner",
		"content_type": "video",
		"language":     "en",
		"currency":     "USD",
	}), "id", testCourseID)
	serve(newCourseHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCourseHandler_Update_NotFound(t *testing.T) {
	ms := &mockCourseService{getErr: service.ErrCourseNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/courses/"+testCourseID, map[string]any{
		"slug":         "go-fundamentals",
		"title":        "Go Fundamentals",
		"level":        "beginner",
		"content_type": "video",
		"language":     "en",
		"currency":     "USD",
	}), "id", testCourseID)
	serve(newCourseHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCourseHandler_SetCategories(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/courses/"+testCourseID+"/categories", map[string]any{
		"category_ids":        []string{testCategoryID},
		"primary_category_id": testCategoryID,
	}), "id", testCourseID)
	serve(newCourseHandler(ms).SetCategories, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCourseHandler_SetCategories_InvalidUUID(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/courses/bad/categories", map[string]any{
		"category_ids": []string{testCategoryID},
	}), "id", "bad")
	serve(newCourseHandler(ms).SetCategories, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCourseHandler_SetTags(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/courses/"+testCourseID+"/tags", map[string]any{
		"tag_ids": []string{testTagID},
	}), "id", testCourseID)
	serve(newCourseHandler(ms).SetTags, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCourseHandler_Delete(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/courses/"+testCourseID, nil), "id", testCourseID)
	serve(newCourseHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCourseHandler_Delete_InvalidUUID(t *testing.T) {
	ms := &mockCourseService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/courses/bad", nil), "id", "bad")
	serve(newCourseHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
