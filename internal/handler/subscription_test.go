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
// mockSubscriptionService
// ============================================================================

type mockSubscriptionService struct {
	sub  *model.Subscription
	subs []model.Subscription
	err  error
}

func (m *mockSubscriptionService) GetActive(_ context.Context, _ string) (*model.Subscription, error) {
	return m.sub, m.err
}

func (m *mockSubscriptionService) ListByUser(_ context.Context, _ string) ([]model.Subscription, error) {
	return m.subs, m.err
}

func (m *mockSubscriptionService) Subscribe(_ context.Context, _ string, _ model.SubscriptionPlan) (*model.Subscription, error) {
	return m.sub, m.err
}

func (m *mockSubscriptionService) Cancel(_ context.Context, _ string) error {
	return m.err
}

func (m *mockSubscriptionService) IsActive(_ context.Context, _ string) (bool, error) {
	return m.sub != nil, m.err
}

// ============================================================================
// helpers
// ============================================================================

const testSubscriptionID = "01966b0a-bbbb-7abc-def0-000000000044"

func newSubscriptionHandler(s *mockSubscriptionService) *handler.SubscriptionHandler {
	return handler.NewSubscriptionHandler(s)
}

func testSubscription() *model.Subscription {
	now := time.Now()
	expires := now.Add(30 * 24 * time.Hour)
	return &model.Subscription{
		ID:        testSubscriptionID,
		UserID:    testUserID,
		Plan:      model.SubscriptionPlanMonthly,
		Status:    model.SubscriptionStatusActive,
		StartedAt: now,
		ExpiresAt: expires,
	}
}

// ============================================================================
// GetActive
// ============================================================================

func TestSubscriptionHandler_GetActive(t *testing.T) {
	s := &mockSubscriptionService{sub: testSubscription()}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/subscriptions/active", nil), testUserID)

	serve(newSubscriptionHandler(s).GetActive, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body model.Subscription
	decodeJSON(t, w, &body)
	assert.Equal(t, testSubscriptionID, body.ID)
}

func TestSubscriptionHandler_GetActive_NotFound(t *testing.T) {
	s := &mockSubscriptionService{err: service.ErrSubscriptionNotFound}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/subscriptions/active", nil), testUserID)

	serve(newSubscriptionHandler(s).GetActive, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubscriptionHandler_GetActive_MissingUserID(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/subscriptions/active", nil)

	serve(newSubscriptionHandler(s).GetActive, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// List
// ============================================================================

func TestSubscriptionHandler_List(t *testing.T) {
	s := &mockSubscriptionService{subs: []model.Subscription{*testSubscription()}}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/subscriptions", nil), testUserID)

	serve(newSubscriptionHandler(s).List, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body []model.Subscription
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestSubscriptionHandler_List_MissingUserID(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/subscriptions", nil)

	serve(newSubscriptionHandler(s).List, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// Subscribe
// ============================================================================

func TestSubscriptionHandler_Subscribe(t *testing.T) {
	s := &mockSubscriptionService{sub: testSubscription()}
	w := httptest.NewRecorder()
	r := withUserID(t, newJSONRequest("POST", "/subscriptions", map[string]any{
		"plan": "monthly",
	}), testUserID)

	serve(newSubscriptionHandler(s).Subscribe, w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var body model.Subscription
	decodeJSON(t, w, &body)
	assert.Equal(t, testSubscriptionID, body.ID)
}

func TestSubscriptionHandler_Subscribe_InvalidPlan(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := withUserID(t, newJSONRequest("POST", "/subscriptions", map[string]any{
		"plan": "weekly",
	}), testUserID)

	serve(newSubscriptionHandler(s).Subscribe, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubscriptionHandler_Subscribe_InvalidJSON(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/subscriptions", nil), testUserID)

	serve(newSubscriptionHandler(s).Subscribe, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubscriptionHandler_Subscribe_MissingUserID(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/subscriptions", map[string]any{"plan": "monthly"})

	serve(newSubscriptionHandler(s).Subscribe, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// Cancel
// ============================================================================

func TestSubscriptionHandler_Cancel(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("DELETE", "/subscriptions/"+testSubscriptionID, nil), testUserID)
	r = withChiParam(r, "id", testSubscriptionID)

	serve(newSubscriptionHandler(s).Cancel, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSubscriptionHandler_Cancel_NotFound(t *testing.T) {
	s := &mockSubscriptionService{err: service.ErrSubscriptionNotFound}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("DELETE", "/subscriptions/"+testSubscriptionID, nil), testUserID)
	r = withChiParam(r, "id", testSubscriptionID)

	serve(newSubscriptionHandler(s).Cancel, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubscriptionHandler_Cancel_InvalidUUID(t *testing.T) {
	s := &mockSubscriptionService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("DELETE", "/subscriptions/bad-id", nil), testUserID)
	r = withChiParam(r, "id", "bad-id")

	serve(newSubscriptionHandler(s).Cancel, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
