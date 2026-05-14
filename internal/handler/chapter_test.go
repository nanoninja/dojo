// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

const testChapterID = "01966b0a-cccc-7abc-def0-000000000003"

func newChapterHandler(cs *mockChapterService) *handler.ChapterHandler {
	return handler.NewChapterHandler(cs)
}

func TestChapterHandler_List(t *testing.T) {
	ms := &mockChapterService{chapters: []model.Chapter{
		{ID: testChapterID, CourseID: testUserID, Title: "Introduction", Slug: "introduction"},
	}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/courses/"+testUserID+"/chapters", nil), "course_id", testUserID)
	serve(newChapterHandler(ms).List, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestChapterHandler_List_InvalidCourseUUID(t *testing.T) {
	ms := &mockChapterService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/courses/bad/chapters", nil), "course_id", "bad")
	serve(newChapterHandler(ms).List, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChapterHandler_GetByID_Found(t *testing.T) {
	ms := &mockChapterService{chapter: &model.Chapter{ID: testChapterID, Title: "Introduction", Slug: "introduction"}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/chapters/"+testChapterID, nil), "id", testChapterID)
	serve(newChapterHandler(ms).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestChapterHandler_GetByID_NotFound(t *testing.T) {
	ms := &mockChapterService{getErr: service.ErrChapterNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/chapters/"+testChapterID, nil), "id", testChapterID)
	serve(newChapterHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChapterHandler_GetByID_InvalidUUID(t *testing.T) {
	ms := &mockChapterService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/chapters/bad", nil), "id", "bad")
	serve(newChapterHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChapterHandler_Create(t *testing.T) {
	ms := &mockChapterService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/chapters", map[string]any{
		"course_id": testUserID,
		"title":     "Introduction",
		"slug":      "introduction",
	})
	serve(newChapterHandler(ms).Create, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "Introduction", body["title"])
}

func TestChapterHandler_Create_InvalidBody(t *testing.T) {
	ms := &mockChapterService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/chapters", map[string]any{"title": ""})
	serve(newChapterHandler(ms).Create, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChapterHandler_Update(t *testing.T) {
	ms := &mockChapterService{chapter: &model.Chapter{ID: testChapterID, Title: "Introduction", Slug: "introduction"}}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/chapters/"+testChapterID, map[string]any{
		"title": "Introduction Updated",
		"slug":  "introduction-updated",
	}), "id", testChapterID)
	serve(newChapterHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChapterHandler_Update_NotFound(t *testing.T) {
	ms := &mockChapterService{getErr: service.ErrChapterNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/chapters/"+testChapterID, map[string]any{
		"title": "Introduction",
		"slug":  "introduction",
	}), "id", testChapterID)
	serve(newChapterHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChapterHandler_Delete(t *testing.T) {
	ms := &mockChapterService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/chapters/"+testChapterID, nil), "id", testChapterID)
	serve(newChapterHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestChapterHandler_Delete_InvalidUUID(t *testing.T) {
	ms := &mockChapterService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/chapters/bad", nil), "id", "bad")
	serve(newChapterHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
