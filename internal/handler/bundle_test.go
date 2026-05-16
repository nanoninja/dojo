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
// mockBundleService
// ============================================================================

type mockBundleService struct {
	bundle  *model.Bundle
	bundles []model.Bundle
	getErr  error
	saveErr error
}

func (m *mockBundleService) List(_ context.Context, _ store.BundleFilter) ([]model.Bundle, error) {
	return m.bundles, m.getErr
}

func (m *mockBundleService) GetByID(_ context.Context, _ string) (*model.Bundle, error) {
	return m.bundle, m.getErr
}

func (m *mockBundleService) GetBySlug(_ context.Context, _ string) (*model.Bundle, error) {
	return m.bundle, m.getErr
}

func (m *mockBundleService) Create(_ context.Context, b *model.Bundle, _ []string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	b.ID = testBundleID
	return nil
}

func (m *mockBundleService) Update(_ context.Context, _ *model.Bundle) error {
	return m.saveErr
}

func (m *mockBundleService) SetCourses(_ context.Context, _ string, _ []string) error {
	return m.saveErr
}

func (m *mockBundleService) Delete(_ context.Context, _ string) error {
	return m.saveErr
}

const (
	testBundleID           = "01966b0a-aaaa-7abc-def0-000000000001"
	testCourseIDForBundle  = "01966b0a-bbbb-7abc-def0-000000000002"
	testCourseIDForBundle2 = "01966b0a-cccc-7abc-def0-000000000003"
)

func newBundleHandler(s *mockBundleService) *handler.BundleHandler {
	return handler.NewBundleHandler(s)
}

func testBundle() *model.Bundle {
	return &model.Bundle{
		ID:           testBundleID,
		InstructorID: testUser1ID,
		Slug:         "go-bundle",
		Title:        "Go Bundle",
		Currency:     "EUR",
		IsPublished:  false,
	}
}

// ============================================================================
// List
// ============================================================================

func TestBundleHandler_List(t *testing.T) {
	s := &mockBundleService{bundles: []model.Bundle{*testBundle()}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bundles", nil)

	serve(newBundleHandler(s).List, rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	decodeJSON(t, rec, &body)
	assert.Len(t, body, 1)
}

func TestBundleHandler_List_InvalidInstructorID(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bundles?instructor_id=bad-uuid", nil)

	serve(newBundleHandler(s).List, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// GetByID
// ============================================================================

func TestBundleHandler_GetByID_Found(t *testing.T) {
	s := &mockBundleService{bundle: testBundle()}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bundles/"+testBundleID, nil)

	serve(newBundleHandler(s).GetByID, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBundleHandler_GetByID_NotFound(t *testing.T) {
	s := &mockBundleService{getErr: service.ErrBundleNotFound}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bundles/"+testBundleID, nil)

	serve(newBundleHandler(s).GetByID, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBundleHandler_GetByID_InvalidUUID(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bundles/bad", nil)

	serve(newBundleHandler(s).GetByID, rec, withChiParam(req, "id", "bad"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// Create
// ============================================================================

func TestBundleHandler_Create(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/bundles", map[string]any{
		"instructor_id": testUser1ID,
		"slug":          "go-bundle",
		"title":         "Go Bundle",
		"currency":      "EUR",
		"course_ids":    []string{testCourseIDForBundle},
	})

	serve(newBundleHandler(s).Create, rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var body map[string]any
	decodeJSON(t, rec, &body)
	assert.Equal(t, testBundleID, body["id"].(string))
}

func TestBundleHandler_Create_InvalidBody(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/bundles", map[string]any{
		"instructor_id": "bad-uuid",
	})

	serve(newBundleHandler(s).Create, rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBundleHandler_Create_SlugTaken(t *testing.T) {
	s := &mockBundleService{saveErr: service.ErrBundleSlugTaken}
	rec := httptest.NewRecorder()
	req := newJSONRequest("POST", "/bundles", map[string]any{
		"instructor_id": testUser1ID,
		"slug":          "go-bundle",
		"title":         "Go Bundle",
		"currency":      "EUR",
	})

	serve(newBundleHandler(s).Create, rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ============================================================================
// Update
// ============================================================================

func TestBundleHandler_Update(t *testing.T) {
	s := &mockBundleService{bundle: testBundle()}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PUT", "/bundles/"+testBundleID, map[string]any{
		"slug":         "go-bundle-updated",
		"title":        "Go Bundle Updated",
		"currency":     "EUR",
		"is_published": true,
	})

	serve(newBundleHandler(s).Update, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBundleHandler_Update_NotFound(t *testing.T) {
	s := &mockBundleService{getErr: service.ErrBundleNotFound}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PUT", "/bundles/"+testBundleID, map[string]any{
		"slug":     "go-bundle",
		"title":    "Go Bundle",
		"currency": "EUR",
	})

	serve(newBundleHandler(s).Update, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBundleHandler_Update_InvalidUUID(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PUT", "/bundles/bad", map[string]any{})

	serve(newBundleHandler(s).Update, rec, withChiParam(req, "id", "bad"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// SetCourses
// ============================================================================

func TestBundleHandler_SetCourses(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PUT", "/bundles/"+testBundleID+"/courses", map[string]any{
		"course_ids": []string{testCourseIDForBundle, testCourseIDForBundle2},
	})

	serve(newBundleHandler(s).SetCourses, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestBundleHandler_SetCourses_InvalidBody(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PUT", "/bundles/"+testBundleID+"/courses", map[string]any{
		"course_ids": []string{"bad-uuid"},
	})

	serve(newBundleHandler(s).SetCourses, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBundleHandler_SetCourses_InvalidBundleUUID(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := newJSONRequest("PUT", "/bundles/bad/courses", map[string]any{
		"course_ids": []string{testCourseIDForBundle},
	})

	serve(newBundleHandler(s).SetCourses, rec, withChiParam(req, "id", "bad"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// Delete
// ============================================================================

func TestBundleHandler_Delete(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/bundles/"+testBundleID, nil)

	serve(newBundleHandler(s).Delete, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestBundleHandler_Delete_InvalidUUID(t *testing.T) {
	s := &mockBundleService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/bundles/bad", nil)

	serve(newBundleHandler(s).Delete, rec, withChiParam(req, "id", "bad"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBundleHandler_Delete_NotFound(t *testing.T) {
	s := &mockBundleService{saveErr: service.ErrBundleNotFound}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/bundles/"+testBundleID, nil)

	serve(newBundleHandler(s).Delete, rec, withChiParam(req, "id", testBundleID))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
