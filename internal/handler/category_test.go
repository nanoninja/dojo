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

const testCategoryID = "01966b0a-bbbb-7abc-def0-000000000002"

func newCategoryHandler(cs *mockCategoryService) *handler.CategoryHandler {
	return handler.NewCategoryHandler(cs)
}

func TestCategoryHandler_List(t *testing.T) {
	ms := &mockCategoryService{categories: []model.Category{
		{ID: testCategoryID, Name: "Backend", Slug: "backend"},
	}}
	w := httptest.NewRecorder()
	serve(newCategoryHandler(ms).List, w, httptest.NewRequest("GET", "/categories", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestCategoryHandler_GetByID_Found(t *testing.T) {
	ms := &mockCategoryService{category: &model.Category{ID: testCategoryID, Name: "Backend", Slug: "backend"}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/categories/"+testCategoryID, nil), "id", testCategoryID)
	serve(newCategoryHandler(ms).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_GetByID_NotFound(t *testing.T) {
	ms := &mockCategoryService{getErr: service.ErrCategoryNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/categories/"+testCategoryID, nil), "id", testCategoryID)
	serve(newCategoryHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCategoryHandler_GetByID_InvalidUUID(t *testing.T) {
	ms := &mockCategoryService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/categories/not-a-uuid", nil), "id", "not-a-uuid")
	serve(newCategoryHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_Create(t *testing.T) {
	ms := &mockCategoryService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/categories", map[string]any{
		"name": "Backend",
		"slug": "backend",
	})
	serve(newCategoryHandler(ms).Create, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "Backend", body["name"])
}

func TestCategoryHandler_Create_InvalidBody(t *testing.T) {
	ms := &mockCategoryService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/categories", map[string]any{"name": ""})
	serve(newCategoryHandler(ms).Create, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_Update(t *testing.T) {
	ms := &mockCategoryService{category: &model.Category{ID: testCategoryID, Name: "Backend", Slug: "backend"}}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/categories/"+testCategoryID, map[string]any{
		"name": "Backend Development",
		"slug": "backend-development",
	}), "id", testCategoryID)
	serve(newCategoryHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Update_NotFound(t *testing.T) {
	ms := &mockCategoryService{getErr: service.ErrCategoryNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/categories/"+testCategoryID, map[string]any{
		"name": "Backend",
		"slug": "backend",
	}), "id", testCategoryID)
	serve(newCategoryHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCategoryHandler_Delete(t *testing.T) {
	ms := &mockCategoryService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/categories/"+testCategoryID, nil), "id", testCategoryID)
	serve(newCategoryHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCategoryHandler_Delete_InvalidUUID(t *testing.T) {
	ms := &mockCategoryService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/categories/bad", nil), "id", "bad")
	serve(newCategoryHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
