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
)

// CategoryHandler handles HTTP requests for course category endpoints.
type CategoryHandler struct {
	category service.CategoryService
}

// NewCategoryHandler creates a new CategoryHandler with the given category service.
func NewCategoryHandler(category service.CategoryService) *CategoryHandler {
	return &CategoryHandler{category: category}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/categories
//
// @Summary   List all course categories
// @Tags      categories
// @Produce   json
// @Security  BearerAuth
// @Success   200  {array}   model.Category
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/categories [get]
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) error {
	categories, err := h.category.List(r.Context())
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, categories)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/categories/{id}
//
// @Summary   Get a category by ID
// @Tags      categories
// @Produce   json
// @Security  BearerAuth
// @Param     id   path      string  true  "Category ID"
// @Success   200  {object}  model.Category
// @Failure   400  {object}  fault.ErrorResponse  "invalid id"
// @Failure   404  {object}  fault.ErrorResponse  "category not found"
// @Router    /api/v1/categories/{id} [get]
func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid category id", nil)
	}
	c, err := h.category.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, c)
}

// ============================================================================
// Create
// ============================================================================

// CreateCategoryRequest holds the fields required to create a category.
type CreateCategoryRequest struct {
	ParentID    *string `json:"parent_id"   validate:"omitempty,uuid"`
	Slug        string  `json:"slug"        validate:"required,max=255"`
	Name        string  `json:"name"        validate:"required,max=255"`
	Description *string `json:"description" validate:"omitempty"`
	ColorHex    *string `json:"color_hex"   validate:"omitempty,max=7"`
	IconURL     *string `json:"icon_url"    validate:"omitempty,max=512"`
	SortOrder   int16   `json:"sort_order"`
	IsVisible   bool    `json:"is_visible"`
}

// Create handles POST /api/v1/categories
//
// @Summary   Create a new category
// @Tags      categories
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     body  body      CreateCategoryRequest  true  "Category payload"
// @Success   201   {object}  model.Category
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   401   {object}  fault.ErrorResponse  "missing or invalid token"
// @Router    /api/v1/categories [post]
func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateCategoryRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	c := &model.Category{
		ParentID:    req.ParentID,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		ColorHex:    req.ColorHex,
		IconURL:     req.IconURL,
		SortOrder:   req.SortOrder,
		IsVisible:   req.IsVisible,
	}
	if err := h.category.Create(r.Context(), c); err != nil {
		return toFault(err)
	}
	return httputil.Created(w, c)
}

// ============================================================================
// Update
// ============================================================================

// UpdateCategoryRequest holds the fields required to update a category.
type UpdateCategoryRequest struct {
	ParentID    *string `json:"parent_id"   validate:"omitempty,uuid"`
	Slug        string  `json:"slug"        validate:"required,max=255"`
	Name        string  `json:"name"        validate:"required,max=255"`
	Description *string `json:"description" validate:"omitempty"`
	ColorHex    *string `json:"color_hex"   validate:"omitempty,max=7"`
	IconURL     *string `json:"icon_url"    validate:"omitempty,max=512"`
	SortOrder   int16   `json:"sort_order"`
	IsVisible   bool    `json:"is_visible"`
}

// Update handles PUT /api/v1/categories/{id}
//
// @Summary   Update a category
// @Tags      categories
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string                 true  "Category ID"
// @Param     body  body      UpdateCategoryRequest  true  "Category payload"
// @Success   200   {object}  model.Category
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404   {object}  fault.ErrorResponse  "category not found"
// @Router    /api/v1/categories/{id} [put]
func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) error {
	var req UpdateCategoryRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid category id", nil)
	}
	c, err := h.category.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	c.ParentID = req.ParentID
	c.Slug = req.Slug
	c.Name = req.Name
	c.Description = req.Description
	c.ColorHex = req.ColorHex
	c.IconURL = req.IconURL
	c.SortOrder = req.SortOrder
	c.IsVisible = req.IsVisible

	if err := h.category.Update(r.Context(), c); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, c)
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/categories/{id}
//
// @Summary   Soft-delete a category
// @Tags      categories
// @Security  BearerAuth
// @Param     id  path  string  true  "Category ID"
// @Success   204
// @Failure   401  {object}  fault.ErrorResponse  "missing or invalid token"
// @Failure   404  {object}  fault.ErrorResponse  "category not found"
func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid category id", nil)
	}
	if err := h.category.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
