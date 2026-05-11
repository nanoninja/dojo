// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

// TagHandler handles HTTP requests for tag endpoints.
type TagHandler struct {
	tag service.TagService
}

// NewTagHandler creates a new TagHandler with the given tag service.
func NewTagHandler(tag service.TagService) *TagHandler {
	return &TagHandler{tag: tag}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/tags
//
// @Summary   List all course tags
// @Tags      tags
// @Produce   json
// @Success   200   {array}    model.Tag
// @Failure   500   {object}   fault.ErrorResponse
// @Router    /api/v1/tags [get]
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) error {
	tags, err := h.tag.List(r.Context())
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, tags)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/tags/{id}
//
// @Summary   Get a tag by ID
// @Tags      tags
// @Produce   json
// @Param     id   path      string  true  "Tag ID"
// @Success   200  {object}  model.Tag
// @Failure   400  {object}  fault.ErrorResponse  "invalid id"
// @Failure   404  {object}  fault.ErrorResponse  "tag not found"
// @Router    /api/v1/tags/{id} [get]
func (h *TagHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	t, err := h.tag.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, t)
}

// ============================================================================
// Create
// ============================================================================

// CreateTagRequest holds the fields required to create a tag.
type CreateTagRequest struct {
	Name string `json:"name" validate:"required,min=2,max=80"`
	Slug string `json:"slug" validate:"required,min=2,max=80"`
}

// Create handles POST /api/v1/tags
//
// @Summary   Create a new tag
// @Tags      tags
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     body  body      CreateTagRequest  true  "Tag payload"
// @Success   201   {object}  model.Tag
// @Failure   400   {object}  fault.ErrorResponse    "invalid request body"
// @Failure   401   {object}  fault.ErrorResponse    "missing or invalid token"
// @Router    /api/v1/tags [post]
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateTagRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	t := &model.Tag{
		Name: req.Name,
		Slug: req.Slug,
	}
	if err := h.tag.Create(r.Context(), t); err != nil {
		return toFault(err)
	}
	return httputil.Created(w, t)
}

// ============================================================================
// Update
// ============================================================================

// UpdateTagRequest holds the fields required to update a tag.
type UpdateTagRequest struct {
	Name string `json:"name" validate:"required,min=2,max=80"`
	Slug string `json:"slug" validate:"required,min=2,max=80"`
}

// Update handles PUT /api/v1/tags/{id}
//
// @Summary   Update a tag
// @Tags      tags
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string               true  "Tag ID"
// @Param     body  body      UpdateTagRequest     true  "Tag payload"
// @Success   200   {object}  model.Tag
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404   {object}  fault.ErrorResponse  "tag not found"
// @Router    /api/v1/tags/{id} [put]
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) error {
	var req UpdateTagRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	t, err := h.tag.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	t.Name = req.Name
	t.Slug = req.Slug

	if err := h.tag.Update(r.Context(), t); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, t)
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/tags/{id}
//
// @Summary   Delete a tag
// @Tags      tags
// @Security  BearerAuth
// @Param     id   path  string  true  "Tag ID"
// @Success   204
// @Failure   401  {object}  fault.ErrorResponse  "missing or invalid token"
// @Failure   404  {object}  fault.ErrorResponse  "tag not found"
// @Router    /api/v1/tags/{id} [delete]
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	if err := h.tag.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
