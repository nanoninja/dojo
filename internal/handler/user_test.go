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

func newUserHandler(user *mockUserService) *handler.UserHandler {
	return handler.NewUserHandler(user)
}

// ============================================================================
// GetByID
// ============================================================================

func TestUserHandler_GetByID_Found(t *testing.T) {
	us := &mockUserService{
		user: &model.User{ID: testUserID, Email: "john@example.com"},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/users/"+testUserID, nil), "id", testUserID)

	serve(h.GetByID, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, testUserID, body["id"])
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	us := &mockUserService{getByIDErr: service.ErrUserNotFound}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	// Use a valid UUID that doesn't exist in the mock — the service returns ErrUserNotFound.
	const unknownID = "01966b0a-ffff-7abc-def0-ffffffffffff"
	r := withChiParam(httptest.NewRequest("GET", "/users/"+unknownID, nil), "id", unknownID)

	serve(h.GetByID, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// List
// ============================================================================

func TestUserHandler_List(t *testing.T) {
	us := &mockUserService{
		users: []model.User{
			{ID: testUser1ID, Email: "alice@example.com"},
			{ID: testUser2ID, Email: "bob@example.com"},
		},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users?page=1&limit=10", nil)

	serve(h.List, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.NotNil(t, body["data"], "List() response should contain data")
	assert.NotNil(t, body["meta"], "List() response should contain meta")
}

func TestUserHandler_List_Total(t *testing.T) {
	us := &mockUserService{
		users: []model.User{
			{ID: testUser1ID, Email: "alice@example.com"},
			{ID: testUser2ID, Email: "bob@example.com"},
			{ID: testUser3ID, Email: "carol@example.com"},
		},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users?page=1&limit=2", nil)

	serve(h.List, w, r)

	var body map[string]any
	decodeJSON(t, w, &body)
	meta := body["meta"].(map[string]any)

	// Total must reflect the full result count, not the current page size.
	assert.Equal(t, float64(3), meta["total"].(float64))
}

func TestUserHandler_List_DefaultPagination(t *testing.T) {
	h := newUserHandler(&mockUserService{users: []model.User{}})
	w := httptest.NewRecorder()
	// No query params: must default to page=1 and limit=20.
	r := httptest.NewRequest("GET", "/users", nil)

	serve(h.List, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	meta := body["meta"].(map[string]any)
	assert.Equal(t, float64(1), meta["page"].(float64))
	assert.Equal(t, float64(20), meta["limit"].(float64))
	assert.Equal(t, float64(0), meta["total"].(float64))
}

// ============================================================================
// Me
// ============================================================================

func TestUserHandler_Me_Success(t *testing.T) {
	us := &mockUserService{
		user: &model.User{
			ID:        testUserID,
			Email:     "john@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Language:  "en",
			Timezone:  "UTC",
		},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest("GET", "/users/me", nil), testUserID)

	serve(h.Me, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, testUserID, body["id"])
	assert.Equal(t, "john@example.com", body["email"])
}

func TestUserHandler_Me_NotFound(t *testing.T) {
	us := &mockUserService{getByIDErr: service.ErrUserNotFound}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest(http.MethodGet, "/users/me", nil), testUserID)

	serve(h.Me, w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// UpdateProfile
// ============================================================================

func TestUserHandler_UpdateProfile_Success(t *testing.T) {
	us := &mockUserService{
		user: &model.User{
			ID:        testUserID,
			Email:     "john@example.com",
			FirstName: "Old",
			LastName:  "Name",
			Language:  "en",
			Timezone:  "UTC",
		},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()

	r := newJSONRequest("PUT", "/users/"+testUserID+"/profile", map[string]any{
		"first_name": "John",
		"last_name":  "Doe",
		"language":   "fr-FR",
		"timezone":   "Europe/Paris",
	})
	r = withChiParam(r, "id", testUserID)
	r = withUserID(t, r, testUserID)

	serve(h.UpdateProfile, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	decodeJSON(t, w, &body)
	assert.Equal(t, "John", body["first_name"])
	assert.Equal(t, "fr-FR", body["language"])
}

func TestUserHandler_UpdateProfile_OtherAccount(t *testing.T) {
	us := &mockUserService{
		user: &model.User{ID: testOtherUserID},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()

	r := newJSONRequest("PUT", "/users/"+testOtherUserID+"/profile", map[string]any{
		"first_name": "John",
		"last_name":  "Doe",
		"language":   "fr-FR",
		"timezone":   "Europe/Paris",
	})
	r = withChiParam(r, "id", testOtherUserID)
	r = withUserID(t, r, testUserID)

	serve(h.UpdateProfile, w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserHandler_UpdateProfile_InvalidBody(t *testing.T) {
	h := newUserHandler(&mockUserService{user: &model.User{ID: testUserID}})
	w := httptest.NewRecorder()

	// Missing required fields: first_name, last_name, language, timezone.
	r := newJSONRequest("PUT", "/users/"+testUserID+"/profile", map[string]any{})
	r = withChiParam(r, "id", testUserID)
	r = withUserID(t, r, testUserID)

	serve(h.UpdateProfile, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ChangePassword
// ============================================================================

func TestUserHandler_ChangePassword_Success(t *testing.T) {
	h := newUserHandler(&mockUserService{})
	w := httptest.NewRecorder()

	r := newJSONRequest(http.MethodPut, "/users/"+testUserID+"/password", map[string]string{
		"current_password": "OldPassword1!",
		"new_password":     "NewPassword1!",
	})
	r = withChiParam(r, "id", testUserID)
	r = withUserID(t, r, testUserID)

	serve(h.ChangePassword, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUserHandler_ChangePassword_OtherAccount(t *testing.T) {
	h := newUserHandler(&mockUserService{})
	w := httptest.NewRecorder()

	r := newJSONRequest(http.MethodPut, "/users/"+testOtherUserID+"/password", map[string]string{
		"current_password": "OldPassword1!",
		"new_password":     "NewPassword1!",
	})
	r = withChiParam(r, "id", testOtherUserID)
	r = withUserID(t, r, testUserID)

	serve(h.ChangePassword, w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	us := &mockUserService{changePassErr: service.ErrWrongPassword}
	h := newUserHandler(us)
	w := httptest.NewRecorder()

	r := newJSONRequest(http.MethodPut, "/users/"+testUserID+"/password", map[string]string{
		"current_password": "OldPassword1!",
		"new_password":     "NewPassword1!",
	})
	r = withChiParam(r, "id", testUserID)
	r = withUserID(t, r, testUserID)

	serve(h.ChangePassword, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_ChangePassword_InvalidBody(t *testing.T) {
	h := newUserHandler(&mockUserService{})
	w := httptest.NewRecorder()

	// Too short current password (min=8).
	r := newJSONRequest(http.MethodPut, "/users/"+testUserID+"/password", map[string]string{
		"current_password": "short",
		"new_password":     "NewPassword1!",
	})
	r = withChiParam(r, "id", testUserID)
	r = withUserID(t, r, testUserID)

	serve(h.ChangePassword, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Delete
// ============================================================================

func TestUserHandler_GetByID_WithAdminRole(t *testing.T) {
	us := &mockUserService{
		user: &model.User{ID: testUserID, Email: "admin@example.com"},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := withChiParam(httptest.NewRequest("GET", "/users/"+testUserID, nil), "id", testUserID)
	r = withRole(t, r, testAdminID, model.RoleAdmin)

	serve(h.GetByID, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_Delete_OwnAccount(t *testing.T) {
	h := newUserHandler(&mockUserService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/users/"+testUserID, nil)
	r = withChiParam(r, "id", testUserID)
	r = withUserID(t, r, testUserID)

	serve(h.Delete, w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUserHandler_Delete_OtherAccount(t *testing.T) {
	h := newUserHandler(&mockUserService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/users/"+testOtherUserID, nil)
	r = withChiParam(r, "id", testOtherUserID)
	// The authenticated user is testUserID, not testOtherUserID -> 403.
	r = withUserID(t, r, testUserID)

	serve(h.Delete, w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// LoginHistory
// ============================================================================

func TestUserHandler_LoginHistory_Success(t *testing.T) {
	us := &mockUserService{
		loginHistory: []model.LoginAuditLog{
			{ID: "log-1", Status: model.LoginStatusSuccess, IPAddress: "1.2.3.4", UserAgent: "Go-http-client/1.1"},
			{ID: "log-2", Status: model.LoginStatusFailedPassword, IPAddress: "1.2.3.4", UserAgent: "Go-http-client/1.1"},
		},
	}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest(http.MethodGet, "/users/me/login-history", nil), testUserID)

	serve(h.LoginHistory, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Len(t, body, 2)
	assert.Equal(t, "log-1", body[0]["id"])
	assert.Equal(t, string(model.LoginStatusSuccess), body[0]["status"].(string))
}

func TestUserHandler_LoginHistory_Empty(t *testing.T) {
	h := newUserHandler(&mockUserService{loginHistory: []model.LoginAuditLog{}})
	w := httptest.NewRecorder()
	r := withUserID(t, httptest.NewRequest(http.MethodGet, "/users/me/login-history?limit=5", nil), testUserID)

	serve(h.LoginHistory, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	decodeJSON(t, w, &body)
	assert.Empty(t, body)
}

func TestUserHandler_LoginHistory_LimitCapped(t *testing.T) {
	us := &mockUserService{loginHistory: []model.LoginAuditLog{}}
	h := newUserHandler(us)
	w := httptest.NewRecorder()
	// limit=999 doit être ramené à maxPageLimit (100) par le handler.
	r := withUserID(t, httptest.NewRequest(http.MethodGet, "/users/me/login-history?limit=999", nil), testUserID)

	serve(h.LoginHistory, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}
