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
// mockConsentService
// ============================================================================

type mockConsentService struct {
	consent  *model.Consent
	consents []model.Consent
	err      error
}

func (m *mockConsentService) ListByUser(_ context.Context, _ string) ([]model.Consent, error) {
	return m.consents, m.err
}

func (m *mockConsentService) GetByID(_ context.Context, _ string) (*model.Consent, error) {
	return m.consent, m.err
}

func (m *mockConsentService) Create(_ context.Context, _ *model.Consent) error {
	return m.err
}

// ============================================================================
// helpers
// ============================================================================

const testConsentID = "01966b0a-aaaa-7abc-def0-000000000033"

func newConsentHandler(s *mockConsentService) *handler.ConsentHandler {
	return handler.NewConsentHandler(s)
}

func testConsent() *model.Consent {
	ip := "192.168.1.1"
	v := "1.0"
	return &model.Consent{
		ID:         testConsentID,
		UserID:     testUserID,
		Type:       model.ConsentTypeTermsOfService,
		Version:    &v,
		IsAccepted: true,
		IPAddress:  &ip,
		UserAgent:  "Mozilla/5.0",
		Source:     model.ConsentSourceRegistration,
	}
}

// ============================================================================
// ListByUser
// ============================================================================

func TestConsentHandler_ListByUser(t *testing.T) {
	s := &mockConsentService{consents: []model.Consent{*testConsent()}}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/consents", nil), testUserID)

	serve(newConsentHandler(s).ListByUser, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body []model.Consent
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestConsentHandler_ListByUser_MissingUserID(t *testing.T) {
	s := &mockConsentService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/consents", nil)

	serve(newConsentHandler(s).ListByUser, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// GetByID
// ============================================================================

func TestConsentHandler_GetByID(t *testing.T) {
	s := &mockConsentService{consent: testConsent()}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/consents/"+testConsentID, nil), testUserID)
	r = withChiParam(r, "id", testConsentID)

	serve(newConsentHandler(s).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body model.Consent
	decodeJSON(t, w, &body)
	assert.Equal(t, testConsentID, body.ID)
}

func TestConsentHandler_GetByID_NotFound(t *testing.T) {
	s := &mockConsentService{err: service.ErrConsentNotFound}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/consents/"+testConsentID, nil), testUserID)
	r = withChiParam(r, "id", testConsentID)

	serve(newConsentHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestConsentHandler_GetByID_IDOR(t *testing.T) {
	c := testConsent()
	c.UserID = testOtherUserID
	s := &mockConsentService{consent: c}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/consents/"+testConsentID, nil), testUserID)
	r = withChiParam(r, "id", testConsentID)

	serve(newConsentHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestConsentHandler_GetByID_MissingUserID(t *testing.T) {
	s := &mockConsentService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/consents/"+testConsentID, nil)
	r = withChiParam(r, "id", testConsentID)

	serve(newConsentHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// Create
// ============================================================================

func TestConsentHandler_Create(t *testing.T) {
	s := &mockConsentService{}
	w := httptest.NewRecorder()
	r := withUserID(t, newJSONRequest("POST", "/consents", map[string]any{
		"type":        "terms_of_service",
		"version":     "1.0",
		"is_accepted": true,
		"user_agent":  "Mozilla/5.0",
		"source":      "registration",
	}), testUserID)

	serve(newConsentHandler(s).Create, w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestConsentHandler_Create_InvalidJSON(t *testing.T) {
	s := &mockConsentService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("POST", "/consents", nil), testUserID)

	serve(newConsentHandler(s).Create, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsentHandler_Create_MissingUserID(t *testing.T) {
	s := &mockConsentService{}
	w := httptest.NewRecorder()
	r := newJSONRequest("POST", "/consents", map[string]any{
		"type":        "terms_of_service",
		"is_accepted": true,
	})

	serve(newConsentHandler(s).Create, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
