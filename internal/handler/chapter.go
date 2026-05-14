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

// ChapterHandler handles HTTP requests for course chapter endpoints.
type ChapterHandler struct {
	chapter service.ChapterService
}

// NewChapterHandler creates a new CourseChapterHandler with the given chapter service.
func NewChapterHandler(chapter service.ChapterService) *ChapterHandler {
	return &ChapterHandler{chapter: chapter}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/courses/{course_id}/chapters
//
// @Summary   List chapters for a course
// @Tags      chapters
// @Produce   json
// @Security  BearerAuth
// @Param     course_id  path      string  true  "Course ID"
// @Success   200        {array}   model.Chapter
// @Failure   400        {object}  fault.ErrorResponse  "invalid course id"
// @Failure   500        {object}  fault.ErrorResponse
// @Router    /api/v1/courses/{course_id}/chapters [get]
func (h *ChapterHandler) List(w http.ResponseWriter, r *http.Request) error {
	courseID := chi.URLParam(r, "course_id")
	if !httputil.ValidateUUID(courseID) {
		return fault.BadRequest("invalid course id", nil)
	}
	chapters, err := h.chapter.List(r.Context(), courseID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, chapters)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/chapters/{id}
//
// @Summary   Get a chapter by ID
// @Tags      chapters
// @Produce   json
// @Security  BearerAuth
// @Param     id   path      string  true  "Chapter ID"
// @Success   200  {object}  model.Chapter
// @Failure   400  {object}  fault.ErrorResponse  "invalid id"
// @Failure   404  {object}  fault.ErrorResponse  "chapter not found"
// @Router    /api/v1/chapters/{id} [get]
func (h *ChapterHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid chapter id", nil)
	}
	chapter, err := h.chapter.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, chapter)
}

// ============================================================================
// Create
// ============================================================================

// CreateChapterRequest holds the fields required to create a chapter.
type CreateChapterRequest struct {
	CourseID        string  `json:"course_id"        validate:"required,uuid"`
	Title           string  `json:"title"            validate:"required,min=2,max=255"`
	Slug            string  `json:"slug"             validate:"required,min=2,max=255"`
	Description     *string `json:"description"      validate:"omitempty"`
	SortOrder       int16   `json:"sort_order"`
	IsFree          bool    `json:"is_free"`
	IsPublished     bool    `json:"is_published"`
	DurationMinutes int     `json:"duration_minutes" validate:"min=0"`
}

// Create handles POST /api/v1/chapters
//
// @Summary   Create a new chapter
// @Tags      chapters
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     body  body      CreateChapterRequest  true  "Chapter payload"
// @Success   201   {object}  model.Chapter
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   401   {object}  fault.ErrorResponse  "missing or invalid token"
// @Router    /api/v1/chapters [post]
func (h *ChapterHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateChapterRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	c := &model.Chapter{
		CourseID:        req.CourseID,
		Title:           req.Title,
		Slug:            req.Slug,
		Description:     req.Description,
		SortOrder:       req.SortOrder,
		IsFree:          req.IsFree,
		IsPublished:     req.IsPublished,
		DurationMinutes: req.DurationMinutes,
	}
	if err := h.chapter.Create(r.Context(), c); err != nil {
		return toFault(err)
	}
	return httputil.Created(w, c)
}

// ============================================================================
// Update
// ============================================================================

// UpdateChapterRequest holds the fields required to update a chapter.
type UpdateChapterRequest struct {
	Title           string  `json:"title"            validate:"required,min=2,max=255"`
	Slug            string  `json:"slug"             validate:"required,min=2,max=255"`
	Description     *string `json:"description"      validate:"omitempty"`
	SortOrder       int16   `json:"sort_order"`
	IsFree          bool    `json:"is_free"`
	IsPublished     bool    `json:"is_published"`
	DurationMinutes int     `json:"duration_minutes" validate:"min=0"`
}

// Update handles PUT /api/v1/chapters/{id}
//
// @Summary   Update a chapter
// @Tags      chapters
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string                true  "Chapter ID"
// @Param     body  body      UpdateChapterRequest  true  "Chapter payload"
// @Success   200   {object}  model.Chapter
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404   {object}  fault.ErrorResponse  "chapter not found"
// @Router    /api/v1/chapters/{id} [put]
func (h *ChapterHandler) Update(w http.ResponseWriter, r *http.Request) error {
	var req UpdateChapterRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid chapter id", nil)
	}
	c, err := h.chapter.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	c.Title = req.Title
	c.Slug = req.Slug
	c.Description = req.Description
	c.SortOrder = req.SortOrder
	c.IsFree = req.IsFree
	c.IsPublished = req.IsPublished
	c.DurationMinutes = req.DurationMinutes

	if err := h.chapter.Update(r.Context(), c); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, c)
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/chapters/{id}
//
// @Summary   Delete a chapter
// @Tags      chapters
// @Security  BearerAuth
// @Param     id  path  string  true  "Chapter ID"
// @Success   204
// @Failure   401  {object}  fault.ErrorResponse  "missing or invalid token"
// @Failure   404  {object}  fault.ErrorResponse  "chapter not found"
// @Router    /api/v1/chapters/{id} [delete]
func (h *ChapterHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid chapter id", nil)
	}
	if err := h.chapter.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
