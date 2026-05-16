// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

// BundleHandler handles HTTP requests for bundles.
type BundleHandler struct {
	bundle service.BundleService
}

// NewBundleHandler returns a new BundleHandler.
func NewBundleHandler(bundle service.BundleService) *BundleHandler {
	return &BundleHandler{bundle: bundle}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/bundles
//
// @Summary  List bundles with optional filters
// @Tags     bundles
// @Produce  json
// @Security BearerAuth
// @Param    instructor_id  query    string  false  "Filter by instructor"
// @Param    is_published   query    bool    false  "Filter by published state"
// @Param    page           query    int     false  "Page number"    default(1)
// @Param    limit          query    int     false  "Items per page" default(20)
// @Success  200  {array}   model.Bundle
// @Failure  400  {object}  fault.ErrorResponse
// @Failure  500  {object}  fault.ErrorResponse
// @Router   /api/v1/bundles [get]
func (h *BundleHandler) List(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()

	page := parseIntQuery(q.Get("page"), 1)
	limit := min(parseIntQuery(q.Get("limit"), 20), maxPageLimit)

	f := store.BundleFilter{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	if v := q.Get("instructor_id"); v != "" {
		if !httputil.ValidateUUID(v) {
			return fault.BadRequest("invalid instructor_id", nil)
		}
		f.InstructorID = v
	}
	if v := q.Get("is_published"); v != "" {
		published := v == "true"
		f.IsPublished = &published
	}

	bundles, err := h.bundle.List(r.Context(), f)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, bundles)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/bundles/{id}
//
// @Summary  Get a bundle by ID
// @Tags     bundles
// @Produce  json
// @Security BearerAuth
// @Param    id   path      string  true  "Bundle ID"
// @Success  200  {object}  model.Bundle
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "bundle not found"
// @Router   /api/v1/bundles/{id} [get]
func (h *BundleHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid bundle id", nil)
	}
	b, err := h.bundle.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, b)
}

// ============================================================================
// Create
// ============================================================================

// CreateBundleRequest holds the fields required to create a bundle.
type CreateBundleRequest struct {
	InstructorID string   `json:"instructor_id" validate:"required,uuid"`
	Slug         string   `json:"slug"          validate:"required,min=1,max=255"`
	Title        string   `json:"title"         validate:"required,min=1,max=255"`
	Subtitle     *string  `json:"subtitle"`
	Description  *string  `json:"description"`
	ThumbnailURL *string  `json:"thumbnail_url"`
	IsFree       bool     `json:"is_free"`
	PriceCents   int      `json:"price_cents"`
	Currency     string   `json:"currency"      validate:"required,len=3"`
	IsPublished  bool     `json:"is_published"`
	SortOrder    int      `json:"sort_order"`
	CourseIDs    []string `json:"course_ids"    validate:"dive,uuid"`
}

// Create handles POST /api/v1/bundles
//
// @Summary  Create a new bundle
// @Tags     bundles
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      CreateBundleRequest  true  "Bundle payload"
// @Success  201   {object}  model.Bundle
// @Failure  400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure  409   {object}  fault.ErrorResponse  "slug already taken"
// @Router   /api/v1/bundles [post]
func (h *BundleHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateBundleRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	b := &model.Bundle{
		InstructorID: req.InstructorID,
		Slug:         req.Slug,
		Title:        req.Title,
		Subtitle:     req.Subtitle,
		Description:  req.Description,
		ThumbnailURL: req.ThumbnailURL,
		IsFree:       req.IsFree,
		PriceCents:   req.PriceCents,
		Currency:     req.Currency,
		IsPublished:  req.IsPublished,
		SortOrder:    req.SortOrder,
	}
	if err := h.bundle.Create(r.Context(), b, req.CourseIDs); err != nil {
		return toFault(err)
	}
	return httputil.Created(w, b)
}

// ============================================================================
// Update
// ============================================================================

// UpdateBundleRequest holds the fields that can be updated on a bundle.
type UpdateBundleRequest struct {
	Slug         string  `json:"slug"          validate:"required,min=1,max=255"`
	Title        string  `json:"title"         validate:"required,min=1,max=255"`
	Subtitle     *string `json:"subtitle"`
	Description  *string `json:"description"`
	ThumbnailURL *string `json:"thumbnail_url"`
	IsFree       bool    `json:"is_free"`
	PriceCents   int     `json:"price_cents"`
	Currency     string  `json:"currency"      validate:"required,len=3"`
	IsPublished  bool    `json:"is_published"`
	SortOrder    int     `json:"sort_order"`
}

// Update handles PUT /api/v1/bundles/{id}
//
// @Summary  Update an existing bundle
// @Tags     bundles
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      string               true  "Bundle ID"
// @Param    body  body      UpdateBundleRequest  true  "Bundle payload"
// @Success  200   {object}  model.Bundle
// @Failure  400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure  404   {object}  fault.ErrorResponse  "bundle not found"
// @Router   /api/v1/bundles/{id} [put]
func (h *BundleHandler) Update(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid bundle id", nil)
	}
	b, err := h.bundle.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	var req UpdateBundleRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	b.Slug = req.Slug
	b.Title = req.Title
	b.Subtitle = req.Subtitle
	b.Description = req.Description
	b.ThumbnailURL = req.ThumbnailURL
	b.IsFree = req.IsFree
	b.PriceCents = req.PriceCents
	b.Currency = req.Currency
	b.IsPublished = req.IsPublished
	b.SortOrder = req.SortOrder

	if err := h.bundle.Update(r.Context(), b); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, b)
}

// ============================================================================
// SetCourses
// ============================================================================

// SetCoursesRequest holds the ordered list of course IDs to assign to a bundle.
type SetCoursesRequest struct {
	CourseIDs []string `json:"course_ids" validate:"required,dive,uuid"`
}

// SetCourses handles PUT /api/v1/bundles/{id}/courses
//
// @Summary  Replace all course assignments for a bundle
// @Tags     bundles
// @Accept   json
// @Security BearerAuth
// @Param    id    path  string            true  "Bundle ID"
// @Param    body  body  SetCoursesRequest true  "Course IDs payload"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid request body"
// @Failure  404  {object}  fault.ErrorResponse  "bundle not found"
// @Router   /api/v1/bundles/{id}/courses [put]
func (h *BundleHandler) SetCourses(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid bundle id", nil)
	}
	var req SetCoursesRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	if err := h.bundle.SetCourses(r.Context(), id, req.CourseIDs); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/bundles/{id}
//
// @Summary  Delete a bundle
// @Tags     bundles
// @Security BearerAuth
// @Param    id  path  string  true  "Bundle ID"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "bundle not found"
// @Router   /api/v1/bundles/{id} [delete]
func (h *BundleHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid bundle id", nil)
	}
	if err := h.bundle.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
