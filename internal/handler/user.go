// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

const maxPageLimit = 100

// UserHandler handles HTTP requests for user account endpoints.
type UserHandler struct {
	user service.UserService
}

// NewUserHandler creates a new UserHandler with the given user service.
func NewUserHandler(user service.UserService) *UserHandler {
	return &UserHandler{user: user}
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/users/{id}
//
// @Summary   Get a user by ID
// @Tags      users
// @Produce   json
// @Security  BearerAuth
// @Param     id   path     string true "User ID"
// @Success   200  {object} model.User
// @Failure   401  {object} fault.ErrorResponse "missing or invalid token"
// @Failure   403  {object} fault.ErrorResponse "insufficient permissions"
// @Failure   404  {object} fault.ErrorResponse "user not found"
// @Router    /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	u, err := h.user.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, u)
}

// ============================================================================
// Me (Profile)
// ============================================================================

// MeResponse is the JSON body returned for the authenticated user's own profile.
// It includes sensitive fields that are hidden from other endpoints.
type MeResponse struct {
	ID           string           `json:"id"`
	Email        string           `json:"email"`
	Status       model.UserStatus `json:"status"`
	IsVerified   bool             `json:"is_verified"`
	Is2FAEnabled bool             `json:"is_2fa_enabled"`
	Role         model.Role       `json:"role"`

	// Profile
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	CompanyName *string `json:"company_name,omitempty"`
	Headline    *string `json:"headline,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Website     *string `json:"website,omitempty"`

	// Sensitive — visible only to the owner
	VATNumber *string    `json:"vat_number,omitempty"`
	BirthDate *time.Time `json:"birth_date,omitempty"`

	// Locale
	Language string `json:"language"`
	Timezone string `json:"timezone"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// Me handles GET /api/v1/users/me
//
// @Summary   Get the authenticated user's profile
// @Tags      users
// @Produce   json
// @Security  BearerAuth
// @Success   200 {object} MeResponse
// @Failure   401 {object} fault.ErrorResponse "missing or invalid token"
// @Failure   404 {object} fault.ErrorResponse "user not found"
// @Router    /api/v1/users/me [get]
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	u, err := h.user.GetByID(r.Context(), userID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, MeResponse{
		ID:           u.ID,
		Email:        u.Email,
		Status:       u.Status,
		IsVerified:   u.IsVerified,
		Is2FAEnabled: u.Is2FAEnabled,
		Role:         u.Role,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		CompanyName:  u.CompanyName,
		Headline:     u.Headline,
		Bio:          u.Bio,
		AvatarURL:    u.AvatarURL,
		Website:      u.Website,
		VATNumber:    u.VATNumber,
		BirthDate:    u.BirthDate,
		Language:     u.Language,
		Timezone:     u.Timezone,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	})
}

// ============================================================================
// List
// ============================================================================

// ListMeta holds pagination metadata returned alongside the user list.
type ListMeta struct {
	Page  int `json:"page"  example:"1"`
	Limit int `json:"limit" example:"20"`
	Total int `json:"total" example:"42"`
}

// ListUsersResponse is the JSON body returned by the list users endpoint.
type ListUsersResponse struct {
	Data []*model.User `json:"data"`
	Meta ListMeta      `json:"meta"`
}

// List handles GET /api/v1/users
//
// @Summary   List users with optional filters and pagination
// @Tags      users
// @Produce   json
// @Security  BearerAuth
// @Param     page    query    int    false "Page number"       default(1)
// @Param     limit   query    int    false "Items per page"    default(20)
// @Param     status  query    string false "Filter by status"  Enums(pending,active,suspended,banned)
// @Param     search  query    string false "Search by name or email"
// @Param     sort    query    string false "Sort order"        Enums(asc,desc)
// @Success   200  {object} ListUsersResponse
// @Failure   401  {object} fault.ErrorResponse "missing or invalid token"
// @Failure   403  {object} fault.ErrorResponse "insufficient permissions"
// @Router    /api/v1/users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()

	page := parseIntQuery(q.Get("page"), 1)
	limit := min(parseIntQuery(q.Get("limit"), 20), maxPageLimit)

	sortOrder := q.Get("sort")
	if sortOrder != "asc" && sortOrder != "desc" && sortOrder != "" {
		sortOrder = "asc"
	}

	filter := store.UserFilter{
		Status:    q.Get("status"),
		Search:    q.Get("search"),
		SortOrder: sortOrder,
		Limit:     limit,
		Offset:    (page - 1) * limit,
	}
	users, total, err := h.user.List(r.Context(), filter)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, map[string]any{
		"data": users,
		"meta": map[string]any{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// ============================================================================
// UpdateProfile
// ============================================================================

// UpdateProfileRequest holds the fields required to update a user profile.
type UpdateProfileRequest struct {
	FirstName   string  `json:"first_name"   validate:"required,min=3,max=50,alpha" example:"John"`
	LastName    string  `json:"last_name"    validate:"required,min=3,max=50,alpha" example:"Doe"`
	CompanyName *string `json:"company_name" validate:"omitempty,max=100"           example:"Acme Corp"`
	Headline    *string `json:"headline"     validate:"omitempty,max=150"           example:"Software Engineer"`
	Bio         *string `json:"bio"          validate:"omitempty,max=1000"          example:"I build things."`
	AvatarURL   *string `json:"avatar_url"   validate:"omitempty,max=512"           example:"https://example.com/avatar.jpg"`
	Website     *string `json:"website"      validate:"omitempty,max=255"           example:"https://example.com"`
	City        *string `json:"city"         validate:"omitempty,max=100"           example:"Paris"`
	PostalCode  *string `json:"postal_code"  validate:"omitempty,max=20"            example:"75001"`
	CountryCode *string `json:"country_code" validate:"omitempty,len=2"             example:"FR"`
	Language    string  `json:"language"     validate:"required,max=10"             example:"fr-FR"`
	Timezone    string  `json:"timezone"     validate:"required,max=50"             example:"Europe/Paris"`
}

// UpdateProfile handles PUT /api/v1/users/{id}/profile
//
// @Summary   Update a user profile
// @Tags      users
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path     string               true "User ID"
// @Param     body  body     UpdateProfileRequest true "Profile payload"
// @Param     X-CSRF-Token  header   string       false "Required in cookie/dual mode"
// @Success   200   {object} model.User
// @Failure   400   {object} fault.ErrorResponse "invalid request body"
// @Failure   401   {object} fault.ErrorResponse "missing or invalid token"
// @Failure   403   {object} fault.ErrorResponse "cannot update another user's profile"
// @Failure   404   {object} fault.ErrorResponse "user not found"
// @Router    /api/v1/users/{id}/profile [put]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) error {
	var req UpdateProfileRequest

	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}

	// Users may only update their own profile.
	callerID := middleware.UserIDFromContext(r.Context())
	if callerID != id {
		return fault.Forbidden(nil)
	}

	u, err := h.user.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}

	u.FirstName = req.FirstName
	u.LastName = req.LastName
	u.CompanyName = req.CompanyName
	u.Headline = req.Headline
	u.Bio = req.Bio
	u.AvatarURL = req.AvatarURL
	u.Website = req.Website
	u.City = req.City
	u.PostalCode = req.PostalCode
	u.CountryCode = req.CountryCode
	u.Language = req.Language
	u.Timezone = req.Timezone

	if err := h.user.UpdateProfile(r.Context(), u); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, u)
}

// ============================================================================
// ChangePassword
// ============================================================================

// ChangePasswordRequest holds the fields required to change a user password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=8,max=72"                example:"OldPassword1!"`
	NewPassword     string `json:"new_password"     validate:"required,min=8,max=72,strongpassword" example:"NewPassword1!"`
}

// ChangePassword handles PUT /api/v1/users/{id}/password
//
// @Summary   Change the password of a user
// @Tags      users
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path  string                true "User ID"
// @Param     body  body  ChangePasswordRequest true "Password payload"
// @Param     X-CSRF-Token  header  string      false "Required in cookie/dual mode"
// @Success   204
// @Failure   400  {object} fault.ErrorResponse "invalid request body"
// @Failure   401  {object} fault.ErrorResponse "current password is incorrect"
// @Failure   403  {object} fault.ErrorResponse "cannot change another user's password"
// @Failure   404  {object} fault.ErrorResponse "user not found"
// @Router    /api/v1/users/{id}/password [put]
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) error {
	var req ChangePasswordRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}

	// Users may only change their own password.
	callerID := middleware.UserIDFromContext(r.Context())
	if callerID != id {
		return fault.Forbidden(nil)
	}

	if err := h.user.ChangePassword(r.Context(), id, req.CurrentPassword, req.NewPassword); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/users/{id}
//
// @Summary   Soft-delete a user account
// @Tags      users
// @Produce   json
// @Security  BearerAuth
// @Param     id   path string true "User ID"
// @Param     X-CSRF-Token  header  string false "Required in cookie/dual mode"
// @Success   204
// @Failure   401  {object} fault.ErrorResponse "missing or invalid token"
// @Failure   403  {object} fault.ErrorResponse "cannot delete another user's account"
// @Failure   404  {object} fault.ErrorResponse "user not found"
// @Router    /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}

	// A user can only delete their own account
	callerID := middleware.UserIDFromContext(r.Context())
	if callerID != id {
		return fault.Forbidden(nil)
	}

	if err := h.user.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// LoginHistory
// ============================================================================

// LoginHistoryEntry is a single entry in the login history response.
type LoginHistoryEntry struct {
	ID        string            `json:"id"`
	Status    model.LoginStatus `json:"status"`
	IPAddress string            `json:"ip_address"`
	UserAgent string            `json:"user_agent"`
	CreatedAt time.Time         `json:"created_at"`
}

// LoginHistory handles GET /api/v1/users/me/login-history
//
// @Summary   Get the login history of the authenticated user
// @Tags      users
// @Produce   json
// @Security  BearerAuth
// @Param     limit  query    int  false  "Number of entries to return" default(20)
// @Success   200  {array}  LoginHistoryEntry
// @Failure   401  {object} fault.ErrorResponse "missing or invalid token"
// @Router    /api/v1/users/me/login-history [get]
func (h *UserHandler) LoginHistory(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	limit := min(parseIntQuery(r.URL.Query().Get("limit"), 20), maxPageLimit)

	logs, err := h.user.LoginHistory(r.Context(), userID, limit)
	if err != nil {
		return toFault(err)
	}

	entries := make([]LoginHistoryEntry, len(logs))
	for i, l := range logs {
		entries[i] = LoginHistoryEntry{
			ID:        l.ID,
			Status:    l.Status,
			IPAddress: l.IPAddress,
			UserAgent: l.UserAgent,
			CreatedAt: l.CreatedAt,
		}
	}
	return httputil.OK(w, entries)
}

// parseIntQuery parses a query parameter as int, returning def if absent or invalid.
func parseIntQuery(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}

// parseBoolPtr parses a query parameter as *bool, returning nil if absent or invalid.
func parseBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return &b
}
