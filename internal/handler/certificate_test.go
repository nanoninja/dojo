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
// mockCertificateService
// ============================================================================

type mockCertificateService struct {
	cert   *model.Certificate
	certs  []model.Certificate
	getErr error
}

func (m *mockCertificateService) GetByID(_ context.Context, _ string) (*model.Certificate, error) {
	return m.cert, m.getErr
}

func (m *mockCertificateService) GetByUUID(_ context.Context, _ string) (*model.Certificate, error) {
	return m.cert, m.getErr
}

func (m *mockCertificateService) ListByUser(_ context.Context, _ string) ([]model.Certificate, error) {
	return m.certs, m.getErr
}

// ============================================================================
// helpers
// ============================================================================

const testCertificateID = "01966b0a-cccc-7abc-def0-000000000011"
const testCertificateUUID = "01966b0a-dddd-7abc-def0-000000000022"

func newCertificateHandler(s *mockCertificateService) *handler.CertificateHandler {
	return handler.NewCertificateHandler(s)
}

func testCertificate() *model.Certificate {
	return &model.Certificate{
		ID:       testCertificateID,
		UserID:   testUserID,
		CourseID: testCourseID,
		UUID:     testCertificateUUID,
		IssuedAt: time.Now(),
	}
}

// ============================================================================
// ListByUser
// ============================================================================

func TestCertificateHandler_ListByUser(t *testing.T) {
	s := &mockCertificateService{certs: []model.Certificate{*testCertificate()}}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/certificates", nil), testUserID)

	serve(newCertificateHandler(s).ListByUser, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body []model.Certificate
	decodeJSON(t, w, &body)
	assert.Len(t, body, 1)
}

func TestCertificateHandler_ListByUser_MissingUserID(t *testing.T) {
	s := &mockCertificateService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/certificates", nil)

	serve(newCertificateHandler(s).ListByUser, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// GetByID
// ============================================================================

func TestCertificateHandler_GetByID(t *testing.T) {
	s := &mockCertificateService{cert: testCertificate()}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/certificates/"+testCertificateID, nil), testUserID)
	r = withChiParam(r, "id", testCertificateID)

	serve(newCertificateHandler(s).GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body model.Certificate
	decodeJSON(t, w, &body)
	assert.Equal(t, testCertificateID, body.ID)
}

func TestCertificateHandler_GetByID_InvalidID(t *testing.T) {
	s := &mockCertificateService{}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/certificates/bad", nil), testUserID)
	r = withChiParam(r, "id", "bad")

	serve(newCertificateHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCertificateHandler_GetByID_NotFound(t *testing.T) {
	s := &mockCertificateService{getErr: service.ErrCertificateNotFound}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/certificates/"+testCertificateID, nil), testUserID)
	r = withChiParam(r, "id", testCertificateID)

	serve(newCertificateHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCertificateHandler_GetByID_IDOR(t *testing.T) {
	cert := testCertificate()
	cert.UserID = testOtherUserID
	s := &mockCertificateService{cert: cert}
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/certificates/"+testCertificateID, nil), testUserID)
	r = withChiParam(r, "id", testCertificateID)

	serve(newCertificateHandler(s).GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCertificateHandler_GetByID_MissingUserID(t *testing.T) {
	s := &mockCertificateService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/certificates/"+testCertificateID, nil)

	serve(newCertificateHandler(s).GetByID, w, withChiParam(r, "id", testCertificateID))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// Verify
// ============================================================================

func TestCertificateHandler_Verify(t *testing.T) {
	s := &mockCertificateService{cert: testCertificate()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/certificates/verify/"+testCertificateUUID, nil)

	serve(newCertificateHandler(s).Verify, w, withChiParam(r, "uuid", testCertificateUUID))

	require.Equal(t, http.StatusOK, w.Code)

	var body model.Certificate
	decodeJSON(t, w, &body)
	assert.Equal(t, testCertificateUUID, body.UUID)
}

func TestCertificateHandler_Verify_InvalidUUID(t *testing.T) {
	s := &mockCertificateService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/certificates/verify/bad", nil)

	serve(newCertificateHandler(s).Verify, w, withChiParam(r, "uuid", "bad"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCertificateHandler_Verify_NotFound(t *testing.T) {
	s := &mockCertificateService{getErr: service.ErrCertificateNotFound}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/certificates/verify/"+testCertificateUUID, nil)

	serve(newCertificateHandler(s).Verify, w, withChiParam(r, "uuid", testCertificateUUID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}
