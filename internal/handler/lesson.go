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

// LessonHandler handles HTTP requests for lesson and lesson resource endpoints.
type LessonHandler struct {
	lesson service.LessonService
}

// NewLessonHandler creates a new LessonHandler with the given lesson service.
func NewLessonHandler(lesson service.LessonService) *LessonHandler {
	return &LessonHandler{lesson: lesson}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/chapters/{chapter_id}/lessons
//
// @Summary   List lessons for a chapter
// @Tags      lessons
// @Produce   json
// @Security  BearerAuth
// @Param     chapter_id  path      string  true  "Chapter ID"
// @Success   200         {array}   model.Lesson
// @Failure   400         {object}  fault.ErrorResponse  "invalid chapter id"
// @Failure   500         {object}  fault.ErrorResponse
// @Router    /api/v1/chapters/{chapter_id}/lessons [get]
func (h *LessonHandler) List(w http.ResponseWriter, r *http.Request) error {
	chapterID := chi.URLParam(r, "chapter_id")
	if err := httputil.ValidateUUID(chapterID); err != nil {
		return err
	}
	lessons, err := h.lesson.List(r.Context(), chapterID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, lessons)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/lessons/{id}
//
// @Summary   Get a lesson by ID
// @Tags      lessons
// @Produce   json
// @Security  BearerAuth
// @Param     id   path      string  true  "Lesson ID"
// @Success   200  {object}  model.Lesson
// @Failure   400  {object}  fault.ErrorResponse  "invalid id"
// @Failure   404  {object}  fault.ErrorResponse  "lesson not found"
// @Router    /api/v1/lessons/{id} [get]
func (h *LessonHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	lesson, err := h.lesson.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, lesson)
}

// ============================================================================
// Create
// ============================================================================

// CreateLessonRequest holds the fields required to create a lesson.
type CreateLessonRequest struct {
	ChapterID       string            `json:"chapter_id"       validate:"required,uuid"`
	Title           string            `json:"title"            validate:"required,min=2,max=255"`
	Slug            string            `json:"slug"             validate:"required,min=2,max=255"`
	Description     *string           `json:"description"      validate:"omitempty"`
	SortOrder       int16             `json:"sort_order"`
	ContentType     model.ContentType `json:"content_type"     validate:"required,oneof=video article audio live document mixed"`
	MediaURL        *string           `json:"media_url"        validate:"omitempty,max=512"`
	IsFree          bool              `json:"is_free"`
	IsPublished     bool              `json:"is_published"`
	DurationMinutes int               `json:"duration_minutes" validate:"min=0"`
}

// Create handles POST /api/v1/lessons
//
// @Summary   Create a new lesson
// @Tags      lessons
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     body  body      CreateLessonRequest  true  "Lesson payload"
// @Success   201   {object}  model.Lesson
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   401   {object}  fault.ErrorResponse  "missing or invalid token"
// @Router    /api/v1/lessons [post]
func (h *LessonHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateLessonRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	l := &model.Lesson{
		ChapterID:       req.ChapterID,
		Title:           req.Title,
		Slug:            req.Slug,
		Description:     req.Description,
		SortOrder:       req.SortOrder,
		ContentType:     req.ContentType,
		MediaURL:        req.MediaURL,
		IsFree:          req.IsFree,
		IsPublished:     req.IsPublished,
		DurationMinutes: req.DurationMinutes,
	}
	if err := h.lesson.Create(r.Context(), l); err != nil {
		return toFault(err)
	}
	return httputil.Created(w, l)
}

// ============================================================================
// Update
// ============================================================================

// UpdateLessonRequest holds the fields required to update a lesson.
type UpdateLessonRequest struct {
	Title           string            `json:"title"            validate:"required,min=2,max=255"`
	Slug            string            `json:"slug"             validate:"required,min=2,max=255"`
	Description     *string           `json:"description"      validate:"omitempty"`
	SortOrder       int16             `json:"sort_order"`
	ContentType     model.ContentType `json:"content_type"     validate:"required,oneof=video article audio live document mixed"`
	MediaURL        *string           `json:"media_url"        validate:"omitempty,max=512"`
	IsFree          bool              `json:"is_free"`
	IsPublished     bool              `json:"is_published"`
	DurationMinutes int               `json:"duration_minutes" validate:"min=0"`
}

// Update handles PUT /api/v1/lessons/{id}
//
// @Summary   Update a lesson
// @Tags      lessons
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string               true  "Lesson ID"
// @Param     body  body      UpdateLessonRequest  true  "Lesson payload"
// @Success   200   {object}  model.Lesson
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404   {object}  fault.ErrorResponse  "lesson not found"
// @Router    /api/v1/lessons/{id} [put]
func (h *LessonHandler) Update(w http.ResponseWriter, r *http.Request) error {
	var req UpdateLessonRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	l, err := h.lesson.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	l.Title = req.Title
	l.Slug = req.Slug
	l.Description = req.Description
	l.SortOrder = req.SortOrder
	l.ContentType = req.ContentType
	l.MediaURL = req.MediaURL
	l.IsFree = req.IsFree
	l.IsPublished = req.IsPublished
	l.DurationMinutes = req.DurationMinutes

	if err := h.lesson.Update(r.Context(), l); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, l)
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/lessons/{id}
//
// @Summary   Delete a lesson
// @Tags      lessons
// @Security  BearerAuth
// @Param     id  path  string  true  "Lesson ID"
// @Success   204
// @Failure   401  {object}  fault.ErrorResponse  "missing or invalid token"
// @Failure   404  {object}  fault.ErrorResponse  "lesson not found"
// @Router    /api/v1/lessons/{id} [delete]
func (h *LessonHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	if err := h.lesson.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// ListResources
// ============================================================================

// ListResources handles GET /api/v1/lessons/{id}/resources
//
// @Summary   List resources for a lesson
// @Tags      lessons
// @Produce   json
// @Security  BearerAuth
// @Param     id   path      string  true  "Lesson ID"
// @Success   200  {array}   model.LessonResource
// @Failure   400  {object}  fault.ErrorResponse  "invalid id"
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/lessons/{id}/resources [get]
func (h *LessonHandler) ListResources(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	resources, err := h.lesson.ListResources(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, resources)
}

// ============================================================================
// AddResource
// ============================================================================

// AddResourceRequest holds the fields required to attach a resource to a lesson.
type AddResourceRequest struct {
	Title         string  `json:"title"           validate:"required,min=2,max=255"`
	Description   *string `json:"description"     validate:"omitempty"`
	FileURL       string  `json:"file_url"        validate:"required,max=512"`
	FileName      string  `json:"file_name"       validate:"required,max=255"`
	FileSizeBytes *int64  `json:"file_size_bytes" validate:"omitempty,min=0"`
	MimeType      *string `json:"mime_type"       validate:"omitempty,max=100"`
	SortOrder     int16   `json:"sort_order"`
	IsPublic      bool    `json:"is_public"`
}

// AddResource handles POST /api/v1/lessons/{id}/resources
//
// @Summary   Add a resource to a lesson
// @Tags      lessons
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string              true  "Lesson ID"
// @Param     body  body      AddResourceRequest  true  "Resource payload"
// @Success   201   {object}  model.LessonResource
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   401   {object}  fault.ErrorResponse  "missing or invalid token"
// @Router    /api/v1/lessons/{id}/resources [post]
func (h *LessonHandler) AddResource(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	var req AddResourceRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	res := &model.LessonResource{
		LessonID:      id,
		Title:         req.Title,
		Description:   req.Description,
		FileURL:       req.FileURL,
		FileName:      req.FileName,
		FileSizeBytes: req.FileSizeBytes,
		MimeType:      req.MimeType,
		SortOrder:     req.SortOrder,
		IsPublic:      req.IsPublic,
	}
	if err := h.lesson.AddResource(r.Context(), res); err != nil {
		return toFault(err)
	}
	return httputil.Created(w, res)
}

// ============================================================================
// UpdateResource
// ============================================================================

// UpdateResourceRequest holds the fields required to update a lesson resource.
type UpdateResourceRequest struct {
	Title         string  `json:"title"           validate:"required,min=2,max=255"`
	Description   *string `json:"description"     validate:"omitempty"`
	FileURL       string  `json:"file_url"        validate:"required,max=512"`
	FileName      string  `json:"file_name"       validate:"required,max=255"`
	FileSizeBytes *int64  `json:"file_size_bytes" validate:"omitempty,min=0"`
	MimeType      *string `json:"mime_type"       validate:"omitempty,max=100"`
	SortOrder     int16   `json:"sort_order"`
	IsPublic      bool    `json:"is_public"`
}

// UpdateResource handles PUT /api/v1/lessons/resources/{id}
//
// @Summary   Update a lesson resource
// @Tags      lessons
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string                 true  "Resource ID"
// @Param     body  body      UpdateResourceRequest  true  "Resource payload"
// @Success   200   {object}  model.LessonResource
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404   {object}  fault.ErrorResponse  "lesson resource not found"
// @Router    /api/v1/lessons/resources/{id} [put]
func (h *LessonHandler) UpdateResource(w http.ResponseWriter, r *http.Request) error {
	var req UpdateResourceRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	res, err := h.lesson.GetResourceByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	res.Title = req.Title
	res.Description = req.Description
	res.FileURL = req.FileURL
	res.FileName = req.FileName
	res.FileSizeBytes = req.FileSizeBytes
	res.MimeType = req.MimeType
	res.SortOrder = req.SortOrder
	res.IsPublic = req.IsPublic

	if err := h.lesson.UpdateResource(r.Context(), res); err != nil {
		return toFault(err)
	}
	return httputil.OK(w, res)
}

// ============================================================================
// RemoveResource
// ============================================================================

// RemoveResource handles DELETE /api/v1/lessons/resources/{id}
//
// @Summary   Remove a resource from a lesson
// @Tags      lessons
// @Security  BearerAuth
// @Param     id  path  string  true  "Resource ID"
// @Success   204
// @Failure   401  {object}  fault.ErrorResponse  "missing or invalid token"
// @Failure   404  {object}  fault.ErrorResponse  "lesson resource not found"
// @Router    /api/v1/lessons/resources/{id} [delete]
func (h *LessonHandler) RemoveResource(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := httputil.ValidateUUID(id); err != nil {
		return err
	}
	if err := h.lesson.RemoveResource(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
