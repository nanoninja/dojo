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
// mockTagService
// ============================================================================

type mockTagService struct {
	tag       *model.Tag
	tags      []model.Tag
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockTagService) List(_ context.Context) ([]model.Tag, error) {
	return m.tags, m.getErr
}

func (m *mockTagService) GetByID(_ context.Context, _ string) (*model.Tag, error) {
	return m.tag, m.getErr
}

func (m *mockTagService) GetBySlug(_ context.Context, _ string) (*model.Tag, error) {
	return m.tag, m.getErr
}

func (m *mockTagService) Create(_ context.Context, t *model.Tag) error {
	t.ID = "01966b0a-aaaa-7abc-def0-000000000001"
	return m.createErr
}

func (m *mockTagService) Update(_ context.Context, _ *model.Tag) error {
	return m.updateErr
}

func (m *mockTagService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

const testTagID = "01966b0a-aaaa-7abc-def0-000000000001"

func newTagHandler(ts *mockTagService) *handler.TagHandler {
	return handler.NewTagHandler(ts)
}

func TestTagHandler_List(t *testing.T) {
	ms := &mockTagService{tags: []model.Tag{
		{ID: testTagID, Name: "Go", Slug: "go"},
	}}
	w := httptest.NewRecorder()
	serve(newTagHandler(ms).List, w, httptest.NewRequest("GET", "/tags", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestTagHandler_GetByID_Found(t *testing.T) {
	ms := &mockTagService{tag: &model.Tag{ID: testTagID, Name: "Go", Slug: "go"}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/tags/"+testTagID, nil), "id", testTagID)
	serve(newTagHandler(ms).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_GetByID_NotFound(t *testing.T) {
	ms := &mockTagService{getErr: service.ErrTagNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/tags/"+testTagID, nil), "id", testTagID)
	serve(newTagHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTagHandler_GetByID_InvalidUUID(t *testing.T) {
	ms := &mockTagService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/tags/not-a-uuid", nil), "id", "not-a-uuid")
	serve(newTagHandler(ms).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTagHandler_GetBySlug_Found(t *testing.T) {
	ms := &mockTagService{tag: &model.Tag{ID: testTagID, Name: "Go", Slug: "go"}}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/tags/slug/go", nil), "slug", "go")
	serve(newTagHandler(ms).GetBySlug, w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_GetBySlug_NotFound(t *testing.T) {
	ms := &mockTagService{getErr: service.ErrTagNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/tags/slug/nope", nil), "slug", "nope")
	serve(newTagHandler(ms).GetBySlug, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTagHandler_Create(t *testing.T) {
	ms := &mockTagService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/tags", map[string]any{"name": "Go", "slug": "go"})
	serve(newTagHandler(ms).Create, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "Go", body["name"])
}

func TestTagHandler_Update(t *testing.T) {
	ms := &mockTagService{tag: &model.Tag{ID: testTagID, Name: "Go", Slug: "go"}}
	w := httptest.NewRecorder()
	r := withChiParam(newJSONRequest("PUT", "/tags/"+testTagID, map[string]any{"name": "Golang", "slug": "golang"}), "id", testTagID)
	serve(newTagHandler(ms).Update, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_Delete(t *testing.T) {
	ms := &mockTagService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/tags/"+testTagID, nil), "id", testTagID)
	serve(newTagHandler(ms).Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
