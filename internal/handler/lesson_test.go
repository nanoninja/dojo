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
// mockLessonService
// ============================================================================

type mockLessonService struct {
	lesson    *model.Lesson
	lessons   []model.Lesson
	resource  *model.LessonResource
	resources []model.LessonResource
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockLessonService) List(_ context.Context, _ string) ([]model.Lesson, error) {
	return m.lessons, m.getErr
}

func (m *mockLessonService) GetByID(_ context.Context, _ string) (*model.Lesson, error) {
	return m.lesson, m.getErr
}

func (m *mockLessonService) GetBySlug(_ context.Context, _, _ string) (*model.Lesson, error) {
	return m.lesson, m.getErr
}

func (m *mockLessonService) Create(_ context.Context, l *model.Lesson) error {
	l.ID = "01966b0a-dddd-7abc-def0-000000000004"
	return m.createErr
}

func (m *mockLessonService) Update(_ context.Context, _ *model.Lesson) error {
	return m.updateErr
}

func (m *mockLessonService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockLessonService) ListResources(_ context.Context, _ string) ([]model.LessonResource, error) {
	return m.resources, m.getErr
}

func (m *mockLessonService) GetResourceByID(_ context.Context, _ string) (*model.LessonResource, error) {
	return m.resource, m.getErr
}

func (m *mockLessonService) AddResource(_ context.Context, r *model.LessonResource) error {
	r.ID = "01966b0a-eeee-7abc-def0-000000000005"
	return m.createErr
}

func (m *mockLessonService) UpdateResource(_ context.Context, _ *model.LessonResource) error {
	return m.updateErr
}

func (m *mockLessonService) RemoveResource(_ context.Context, _ string) error {
	return m.deleteErr
}

const (
	testLessonID   = "01966b0a-dddd-7abc-def0-000000000004"
	testResourceID = "01966b0a-eeee-7abc-def0-000000000005"
)

func newLessonHandler(ls *mockLessonService) *handler.LessonHandler {
	return handler.NewLessonHandler(ls)
}

// ============================================================================
// Lesson CRUD
// ============================================================================

func TestLessonHandler_List(t *testing.T) {
	ms := &mockLessonService{lessons: []model.Lesson{
		{ID: testLessonID, ChapterID: testChapterID, Title: "Variables", Slug: "variables"},
	}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/chapters/"+testChapterID+"/lessons", nil), "chapter_id", testChapterID)
	serve(newLessonHandler(ms).List, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestLessonHandler_List_InvalidChapterUUID(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/chapters/bad/lessons", nil), "chapter_id", "bad")
	serve(newLessonHandler(ms).List, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLessonHandler_GetByID_Found(t *testing.T) {
	ms := &mockLessonService{lesson: &model.Lesson{ID: testLessonID, Title: "Variables", Slug: "variables"}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/lessons/"+testLessonID, nil), "id", testLessonID)
	serve(newLessonHandler(ms).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestLessonHandler_GetByID_NotFound(t *testing.T) {
	ms := &mockLessonService{getErr: service.ErrLessonNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/lessons/"+testLessonID, nil), "id", testLessonID)
	serve(newLessonHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLessonHandler_GetByID_InvalidUUID(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/lessons/bad", nil), "id", "bad")
	serve(newLessonHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLessonHandler_Create(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/lessons", map[string]any{
		"chapter_id":   testChapterID,
		"title":        "Variables",
		"slug":         "variables",
		"content_type": "video",
	})
	serve(newLessonHandler(ms).Create, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "Variables", body["title"])
}

func TestLessonHandler_Create_InvalidBody(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/lessons", map[string]any{"title": ""})
	serve(newLessonHandler(ms).Create, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLessonHandler_Update(t *testing.T) {
	ms := &mockLessonService{lesson: &model.Lesson{ID: testLessonID, Title: "Variables", Slug: "variables"}}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/lessons/"+testLessonID, map[string]any{
		"title":        "Variables & Types",
		"slug":         "variables-types",
		"content_type": "video",
	}), "id", testLessonID)
	serve(newLessonHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLessonHandler_Update_NotFound(t *testing.T) {
	ms := &mockLessonService{getErr: service.ErrLessonNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/lessons/"+testLessonID, map[string]any{
		"title":        "Variables",
		"slug":         "variables",
		"content_type": "video",
	}), "id", testLessonID)
	serve(newLessonHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLessonHandler_Delete(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/lessons/"+testLessonID, nil), "id", testLessonID)
	serve(newLessonHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ============================================================================
// Resources
// ============================================================================

func TestLessonHandler_ListResources(t *testing.T) {
	ms := &mockLessonService{resources: []model.LessonResource{
		{ID: testResourceID, LessonID: testLessonID, Title: "Slides", FileURL: "https://example.com/s.pdf", FileName: "s.pdf"},
	}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/lessons/"+testLessonID+"/resources", nil), "id", testLessonID)
	serve(newLessonHandler(ms).ListResources, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestLessonHandler_AddResource(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("POST", "/lessons/"+testLessonID+"/resources", map[string]any{
		"title":     "Slides",
		"file_url":  "https://example.com/slides.pdf",
		"file_name": "slides.pdf",
	}), "id", testLessonID)
	serve(newLessonHandler(ms).AddResource, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "Slides", body["title"])
}

func TestLessonHandler_AddResource_InvalidBody(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("POST", "/lessons/"+testLessonID+"/resources", map[string]any{}), "id", testLessonID)
	serve(newLessonHandler(ms).AddResource, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLessonHandler_UpdateResource(t *testing.T) {
	ms := &mockLessonService{resource: &model.LessonResource{
		ID: testResourceID, LessonID: testLessonID, Title: "Slides", FileURL: "https://example.com/s.pdf", FileName: "s.pdf",
	}}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/lessons/resources/"+testResourceID, map[string]any{
		"title":     "Updated Slides",
		"file_url":  "https://example.com/slides-v2.pdf",
		"file_name": "slides-v2.pdf",
	}), "id", testResourceID)
	serve(newLessonHandler(ms).UpdateResource, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLessonHandler_UpdateResource_NotFound(t *testing.T) {
	ms := &mockLessonService{getErr: service.ErrLessonResourceNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/lessons/resources/"+testResourceID, map[string]any{
		"title":     "Slides",
		"file_url":  "https://example.com/s.pdf",
		"file_name": "s.pdf",
	}), "id", testResourceID)
	serve(newLessonHandler(ms).UpdateResource, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLessonHandler_RemoveResource(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/lessons/resources/"+testResourceID, nil), "id", testResourceID)
	serve(newLessonHandler(ms).RemoveResource, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestLessonHandler_RemoveResource_InvalidUUID(t *testing.T) {
	ms := &mockLessonService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/lessons/resources/bad", nil), "id", "bad")
	serve(newLessonHandler(ms).RemoveResource, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
