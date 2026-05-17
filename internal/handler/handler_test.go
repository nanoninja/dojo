// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// Helpers
// ============================================================================

// testJWTSecret is the signing key used for JWT tokens in handler tests.
const testJWTSecret = "test-handler-jwt-secret-key-32b"

// Test UUIDs used across handler tests.
// Using UUIDv7-shaped values (time-ordered prefix) for realism.
const (
	testUserID      = "01966b0a-1234-7abc-def0-1234567890ab"
	testOtherUserID = "01966b0a-5678-7abc-def0-1234567890cd"
	testAdminID     = "01966b0a-9012-7abc-def0-1234567890ef"
	testUser1ID     = "01966b0a-1111-7abc-def0-1234567890aa"
	testUser2ID     = "01966b0a-2222-7abc-def0-1234567890bb"
	testUser3ID     = "01966b0a-3333-7abc-def0-1234567890cc"
	testNewUserID   = "01966b0a-4444-7abc-def0-1234567890dd"
)

// serve wraps h with httputil.Handle (discarding logs) and writes to w.
// This mirrors the production handler wiring without cluttering test output.
func serve(h httputil.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	httputil.Handle(h, logger).ServeHTTP(w, r)
}

// newJSONRequest creates an HTTP request with a JSON-encoded body.
func newJSONRequest(method, path string, body any) *http.Request {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// withChiParam injects a chi URL parameter into the request context.
// If a chi route context already exists on the request, it is reused so that
// multiple calls can be chained without overwriting previous parameters.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withUserID signs a JWT and runs the request through the Authenticate middleware,
// injecting the userID into the request context exactly as production does.
func withUserID(t *testing.T, r *http.Request, userID string) *http.Request {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": "user",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testJWTSecret))
	require.NoError(t, err, "signing JWT")
	r.Header.Set("Authorization", "Bearer "+tok)

	var outCtx context.Context
	middleware.Authenticate(testJWTSecret)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		outCtx = req.Context()
	})).ServeHTTP(httptest.NewRecorder(), r)

	return r.WithContext(outCtx)
}

// withRole signs a JWT with the given role and runs the request through Authenticate,
// injecting both the userID and role into the request context.
func withRole(t *testing.T, r *http.Request, userID string, role model.Role) *http.Request {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": role.String(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testJWTSecret))
	require.NoError(t, err, "signing JWT")
	r.Header.Set("Authorization", "Bearer "+tok)

	var outCtx context.Context
	middleware.Authenticate(testJWTSecret)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		outCtx = req.Context()
	})).ServeHTTP(httptest.NewRecorder(), r)

	return r.WithContext(outCtx)
}

// decodeJSON decodes the JSON body from the recorder into v.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(w.Body).Decode(v), "decodeJSON")
}

// ============================================================================
// mockUserService — in-memory stub that implements service.UserService.
// Set the relevant fields to control what each method returns in a test.
// ============================================================================

type mockUserService struct {
	user            *model.User
	users           []model.User
	getByIDErr      error
	registerErr     error
	updateErr       error
	changePassErr   error
	deleteErr       error
	loginHistory    []model.LoginAuditLog
	loginHistoryErr error
}

func (m *mockUserService) List(_ context.Context, _ store.UserFilter) ([]model.User, int, error) {
	return m.users, len(m.users), nil
}

func (m *mockUserService) GetByID(_ context.Context, _ string) (*model.User, error) {
	return m.user, m.getByIDErr
}

func (m *mockUserService) Register(_ context.Context, u *model.User, _ string) error {
	u.ID = testNewUserID
	return m.registerErr
}

func (m *mockUserService) UpdateProfile(_ context.Context, _ *model.User) error {
	return m.updateErr
}

func (m *mockUserService) ChangePassword(_ context.Context, _, _, _ string) error {
	return m.changePassErr
}

func (m *mockUserService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockUserService) LoginHistory(_ context.Context, _ string, _ int) ([]model.LoginAuditLog, error) {
	return m.loginHistory, m.loginHistoryErr
}
