// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

// ============================================================================
// mockPurchaseService
// ============================================================================

type mockPurchaseService struct {
	purchase  *model.Purchase
	purchases []model.Purchase
	err       error
}

func (m *mockPurchaseService) GetByID(_ context.Context, _ string) (*model.Purchase, error) {
	return m.purchase, m.err
}

func (m *mockPurchaseService) ListByUser(_ context.Context, _ string) ([]model.Purchase, error) {
	return m.purchases, m.err
}

func (m *mockPurchaseService) BuyCourse(_ context.Context, _, _ string, _ int64, _ string) (*model.Purchase, error) {
	return m.purchase, m.err
}

func (m *mockPurchaseService) BuyBundle(_ context.Context, _, _ string, _ int64, _ string) (*model.Purchase, error) {
	return m.purchase, m.err
}

func (m *mockPurchaseService) Refund(_ context.Context, _ string) error {
	return m.err
}

func (m *mockPurchaseService) ConfirmPayment(_ context.Context, _, _ string) error {
	return m.err
}

func (m *mockPurchaseService) CancelPending(_ context.Context, _ string) error {
	return m.err
}

// ============================================================================
// helpers
// ============================================================================

const testPurchaseID = "01966b0a-cccc-7abc-def0-000000000055"

func newPurchaseHandler(s *mockPurchaseService) *handler.PurchaseHandler {
	return handler.NewPurchaseHandler(s)
}

func testPurchase() *model.Purchase {
	return &model.Purchase{
		ID:          testPurchaseID,
		UserID:      testUserID,
		Type:        model.PurchaseTypeCourse,
		ItemID:      "01966b0a-dddd-7abc-def0-000000000066",
		Status:      model.PurchaseStatusCompleted,
		AmountCents: 1999,
		Currency:    "EUR",
		CreatedAt:   time.Now(),
	}
}

// ============================================================================
// GetByID
// ============================================================================

func TestPurchaseHandler_GetByID(t *testing.T) {
	s := &mockPurchaseService{purchase: testPurchase()}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/purchases/"+testPurchaseID, nil), testUserID)
	r = withChiParam(r, "id", testPurchaseID)

	serve(newPurchaseHandler(s).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body model.Purchase
	decodeJSON(t, w, &body)
	assert.Equal(t, testPurchaseID, body.ID)
}

func TestPurchaseHandler_GetByID_NotFound(t *testing.T) {
	s := &mockPurchaseService{err: service.ErrPurchaseNotFound}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/purchases/"+testPurchaseID, nil), testUserID)
	r = withChiParam(r, "id", testPurchaseID)

	serve(newPurchaseHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPurchaseHandler_GetByID_InvalidUUID(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/purchases/bad-id", nil), testUserID)
	r = withChiParam(r, "id", "bad-id")

	serve(newPurchaseHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// List
// ============================================================================

func TestPurchaseHandler_List(t *testing.T) {
	s := &mockPurchaseService{purchases: []model.Purchase{*testPurchase()}}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/purchases", nil), testUserID)

	serve(newPurchaseHandler(s).List, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body []model.Purchase
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestPurchaseHandler_List_MissingUserID(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/purchases", nil)

	serve(newPurchaseHandler(s).List, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// BuyCourse
// ============================================================================

func TestPurchaseHandler_BuyCourse(t *testing.T) {
	s := &mockPurchaseService{purchase: testPurchase()}
	w := httptest.NewRecorder()
	r := withUserID(t, newJSONRequest("POST", "/purchases/courses", map[string]any{
		"course_id":    "01966b0a-dddd-7abc-def0-000000000066",
		"amount_cents": 1999,
		"currency":     "EUR",
	}), testUserID)

	serve(newPurchaseHandler(s).BuyCourse, w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var body model.Purchase
	decodeJSON(t, w, &body)
	assert.Equal(t, testPurchaseID, body.ID)
}

func TestPurchaseHandler_BuyCourse_InvalidJSON(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/purchases/courses", nil), testUserID)

	serve(newPurchaseHandler(s).BuyCourse, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPurchaseHandler_BuyCourse_MissingUserID(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/purchases/courses", map[string]any{
		"course_id":    "01966b0a-dddd-7abc-def0-000000000066",
		"amount_cents": 1999,
		"currency":     "EUR",
	})

	serve(newPurchaseHandler(s).BuyCourse, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// BuyBundle
// ============================================================================

func TestPurchaseHandler_BuyBundle(t *testing.T) {
	s := &mockPurchaseService{purchase: testPurchase()}
	w := httptest.NewRecorder()
	r := withUserID(t, newJSONRequest("POST", "/purchases/bundles", map[string]any{
		"bundle_id":    "01966b0a-eeee-7abc-def0-000000000077",
		"amount_cents": 4999,
		"currency":     "EUR",
	}), testUserID)

	serve(newPurchaseHandler(s).BuyBundle, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestPurchaseHandler_BuyBundle_InvalidJSON(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/purchases/bundles", nil), testUserID)

	serve(newPurchaseHandler(s).BuyBundle, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPurchaseHandler_BuyBundle_MissingUserID(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/purchases/bundles", map[string]any{
		"bundle_id":    "01966b0a-eeee-7abc-def0-000000000077",
		"amount_cents": 4999,
		"currency":     "EUR",
	})

	serve(newPurchaseHandler(s).BuyBundle, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// Refund
// ============================================================================

func TestPurchaseHandler_Refund(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/purchases/"+testPurchaseID+"/refund", nil), testUserID)
	r = withChiParam(r, "id", testPurchaseID)

	serve(newPurchaseHandler(s).Refund, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestPurchaseHandler_Refund_NotFound(t *testing.T) {
	s := &mockPurchaseService{err: service.ErrPurchaseNotFound}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/purchases/"+testPurchaseID+"/refund", nil), testUserID)
	r = withChiParam(r, "id", testPurchaseID)

	serve(newPurchaseHandler(s).Refund, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPurchaseHandler_Refund_InvalidUUID(t *testing.T) {
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/purchases/bad-id/refund", nil), testUserID)
	r = withChiParam(r, "id", "bad-id")

	serve(newPurchaseHandler(s).Refund, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
