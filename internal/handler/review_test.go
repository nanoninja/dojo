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
// mockReviewService
// ============================================================================

type mockReviewService struct {
	review  *model.Review
	reviews []model.Review
	getErr  error
	saveErr error
}

func (m *mockReviewService) List(_ context.Context, _ store.ReviewFilter) ([]model.Review, int, error) {
	return m.reviews, len(m.reviews), nil
}

func (m *mockReviewService) GetByID(_ context.Context, _ string) (*model.Review, error) {
	return m.review, m.getErr
}

func (m *mockReviewService) GetByUserAndCourse(_ context.Context, _, _ string) (*model.Review, error) {
	return m.review, m.getErr
}

func (m *mockReviewService) Create(_ context.Context, _ *model.Review) error {
	return m.saveErr
}

func (m *mockReviewService) Update(_ context.Context, _ *model.Review) error {
	return m.saveErr
}

func (m *mockReviewService) Delete(_ context.Context, _ string) error {
	if m.getErr != nil {
		return m.getErr
	}
	return m.saveErr
}

// ============================================================================
// helpers
// ============================================================================
const testReviewID = "01966b0a-aaaa-7abc-def0-000000000099"

func newReviewHandler(s *mockReviewService) *handler.ReviewHandler {
	return handler.NewReviewHandler(s)
}
func testReview() *model.Review {
	return &model.Review{
		ID:       testReviewID,
		UserID:   testUserID,
		CourseID: testCourseID,
		Rating:   4,
		Comment:  "Great course!",
	}
}

// ============================================================================
// List
// ============================================================================

func TestReviewHandler_List(t *testing.T) {
	s := &mockReviewService{reviews: []model.Review{*testReview()}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/courses/"+testCourseID+"/reviews", nil)

	serve(newReviewHandler(s).List, w, withChiParam(r, "course_id", testCourseID))

	require.Equal(t, http.StatusOK, w.Code)

	var body httputil.PageResponse[model.Review]
	decodeJSON(t, w, &body)

	assert.Len(t, body.Data, 1)
	assert.Equal(t, 1, body.Meta.Total)
}

func TestReviewHandler_List_InvalidCourseID(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/courses/bad/reviews", nil)

	serve(newReviewHandler(s).List, w, withChiParam(r, "course_id", "bad"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetByID
// ============================================================================

func TestReviewHandler_GetByID(t *testing.T) {
	s := &mockReviewService{review: testReview()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/reviews/"+testReviewID, nil)

	serve(newReviewHandler(s).GetByID, w, withChiParam(r, "id", testReviewID))

	require.Equal(t, http.StatusOK, w.Code)

	var body model.Review
	decodeJSON(t, w, &body)
	assert.Equal(t, testReviewID, body.ID)
}

func TestReviewHandler_GetByID_NotFound(t *testing.T) {
	s := &mockReviewService{getErr: service.ErrReviewNotFound}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/reviews/"+testReviewID, nil)

	serve(newReviewHandler(s).GetByID, w, withChiParam(r, "id", testReviewID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReviewHandler_GetByID_InvalidID(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/reviews/bad", nil), "id", "bad")

	serve(newReviewHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Create
// ============================================================================

func TestReviewHandler_Create(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(
		withRole(t, newJSONRequest("POST", "/courses/"+testCourseID+"/reviews", map[string]any{
			"rating":  4,
			"comment": "Great course!",
		}), testUserID, model.RoleUser),
		"course_id", testCourseID,
	)

	serve(newReviewHandler(s).Create, w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestReviewHandler_Create_InvalidCourseID(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(
		withRole(t, newJSONRequest("POST", "/courses/bad/reviews", map[string]any{
			"rating": 4,
		}), testUserID, model.RoleUser),
		"course_id", "bad",
	)

	serve(newReviewHandler(s).Create, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReviewHandler_Create_Unauthorized(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(
		newJSONRequest("POST", "/courses/"+testCourseID+"/reviews", map[string]any{"rating": 4}),
		"course_id", testCourseID,
	)

	serve(newReviewHandler(s).Create, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// Update
// ============================================================================

func TestReviewHandler_Update(t *testing.T) {
	s := &mockReviewService{review: testReview()}
	w := httptest.NewRecorder()
	r := withChiParam(withChiParam(
		withRole(t, newJSONRequest("PUT", "/courses/"+testCourseID+"/reviews/"+testReviewID, map[string]any{
			"rating":  5,
			"comment": "Even better!",
		}), testUserID, model.RoleUser),
		"course_id", testCourseID),
		"id", testReviewID,
	)

	serve(newReviewHandler(s).Update, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReviewHandler_Update_NotFound(t *testing.T) {
	s := &mockReviewService{getErr: service.ErrReviewNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(
		newJSONRequest("PUT", "/reviews/"+testReviewID, map[string]any{"rating": 5, "comment": "Great course!"}),
		"id", testReviewID,
	)

	serve(newReviewHandler(s).Update, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReviewHandler_Update_InvalidID(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(
		newJSONRequest("PUT", "/reviews/bad", map[string]any{"rating": 5, "comment": "x"}),
		"id", "bad",
	)

	serve(newReviewHandler(s).Update, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Delete
// ============================================================================

func TestReviewHandler_Delete(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/reviews/"+testReviewID, nil), "id", testReviewID)

	serve(newReviewHandler(s).Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestReviewHandler_Delete_NotFound(t *testing.T) {
	s := &mockReviewService{getErr: service.ErrReviewNotFound}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/reviews/"+testReviewID, nil), "id", testReviewID)

	serve(newReviewHandler(s).Delete, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReviewHandler_Delete_InvalidID(t *testing.T) {
	s := &mockReviewService{}
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("DELETE", "/reviews/bad", nil), "id", "bad")

	serve(newReviewHandler(s).Delete, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
