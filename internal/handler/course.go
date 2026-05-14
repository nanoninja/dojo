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

// CourseHandler handles HTTP requests for course endpoints.
type CourseHandler struct {
	course service.CourseService
}

// NewCourseHandler creates a new CourseHandler with the given course service.
func NewCourseHandler(course service.CourseService) *CourseHandler {
	return &CourseHandler{course: course}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/courses
//
// @Summary   List courses with optional filters
// @Tags      courses
// @Produce   json
// @Security  BearerAuth
// @Param     instructor_id  query    string  false  "Filter by instructor"
// @Param     category_id    query    string  false  "Filter by category"
// @Param     search         query    string  false  "Search by title or subtitle"
// @Param     level          query    string  false  "Filter by level"  Enums(beginner,intermediate,advanced,expert)
// @Param     language       query    string  false  "Filter by language"
// @Param     is_free        query    bool    false  "Filter free courses"
// @Param     is_published   query    bool    false  "Filter published courses"
// @Param     sort           query    string  false  "Sort order"  Enums(asc,desc)
// @Param     page           query    int     false  "Page number"    default(1)
// @Param     limit          query    int     false  "Items per page" default(20)
// @Success   200  {array}   model.Course
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/courses [get]
func (h *CourseHandler) List(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()

	page := parseIntQuery(q.Get("page"), 1)
	limit := min(parseIntQuery(q.Get("limit"), 20), maxPageLimit)

	search := q.Get("search")
	if len(search) > 100 {
		return fault.BadRequest("search must not exceed 100 characters", nil)
	}

	level := model.CourseLevel(q.Get("level"))
	if level != "" {
		switch level {
		case model.CourseLevelBeginner,
			model.CourseLevelIntermediate,
			model.CourseLevelAdvanced,
			model.CourseLevelExpert:
		default:
			return fault.BadRequest("invalid level", nil)
		}
	}

	language := q.Get("language")
	if len(language) > 10 {
		return fault.BadRequest("language must not exceed 10 characters", nil)
	}

	sortDir := store.SortDirDesc
	if s := q.Get("sort"); s == "asc" {
		sortDir = store.SortDirAsc
	}

	filter := store.CourseFilter{
		Search:      search,
		Level:       level,
		Language:    language,
		IsFree:      parseBoolPtr(q.Get("is_free")),
		IsPublished: parseBoolPtr(q.Get("is_published")),
		SortDir:     sortDir,
		Limit:       limit,
		Offset:      (page - 1) * limit,
	}

	if id := q.Get("instructor_id"); id != "" {
		if !httputil.ValidateUUID(id) {
			return fault.BadRequest("invalid instructor_id", nil)
		}
		filter.InstructorID = id
	}

	if id := q.Get("category_id"); id != "" {
		if !httputil.ValidateUUID(id) {
			return fault.BadRequest("invalid category_id", nil)
		}
		filter.CategoryID = id
	}

	courses, err := h.course.List(r.Context(), filter)
	if err != nil {
		return toFault(err)
	}

	return httputil.OK(w, courses)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/courses/{id}
//
// @Summary   Get a course by ID
// @Tags      courses
// @Produce   json
// @Security  BearerAuth
// @Param     id   path      string  true  "Course ID"
// @Success   200  {object}  model.Course
// @Failure   404  {object}  fault.ErrorResponse "course not found"
// @Router    /api/v1/courses/{id} [get]
func (h *CourseHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid course id", nil)
	}
	c, err := h.course.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, c)
}

// ============================================================================
// Create
// ============================================================================

// CreateCourseRequest holds the fields required to create a course.
type CreateCourseRequest struct {
	InstructorID       string            `json:"instructor_id"       validate:"required,uuid"`
	Slug               string            `json:"slug"                validate:"required,max=255"`
	Title              string            `json:"title"               validate:"required,max=255"`
	Subtitle           *string           `json:"subtitle"            validate:"omitempty,max=500"`
	Description        *string           `json:"description"         validate:"omitempty"`
	Prerequisites      *string           `json:"prerequisites"       validate:"omitempty"`
	Objectives         *string           `json:"objectives"          validate:"omitempty"`
	MetaTitle          *string           `json:"meta_title"          validate:"omitempty,max=255"`
	MetaDescription    *string           `json:"meta_description"    validate:"omitempty,max=500"`
	MetaKeywords       *string           `json:"meta_keywords"       validate:"omitempty,max=255"`
	ThumbnailURL       *string           `json:"thumbnail_url"       validate:"omitempty,max=512"`
	TrailerURL         *string           `json:"trailer_url"         validate:"omitempty,max=512"`
	Level              model.CourseLevel `json:"level"               validate:"required,oneof=beginner intermediate advanced expert"`
	ContentType        model.ContentType `json:"content_type"        validate:"required,oneof=video article audio live document mixed"`
	Language           string            `json:"language"            validate:"required,max=10"`
	IsFree             bool              `json:"is_free"`
	SubscriptionOnly   bool              `json:"subscription_only"`
	PriceCents         int               `json:"price_cents"         validate:"min=0"`
	Currency           string            `json:"currency"            validate:"required,len=3"`
	IsPublished        bool              `json:"is_published"`
	IsFeatured         bool              `json:"is_featured"`
	CertificateEnabled bool              `json:"certificate_enabled"`
	SortOrder          int16             `json:"sort_order"`
	PrimaryCategoryID  string            `json:"primary_category_id" validate:"omitempty"`
	CategoryIDs        []string          `json:"category_ids"        validate:"omitempty"`
	TagIDs             []string          `json:"tag_ids"             validate:"omitempty"`
}

// Create handles POST /api/v1/courses
//
// @Summary   Create a new course
// @Tags      courses
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     body  body      CreateCourseRequest  true  "Course payload"
// @Success   201   {object}  model.Course
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   401   {object}  fault.ErrorResponse  "missing or invalid token"
// @Router    /api/v1/courses [post]
func (h *CourseHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateCourseRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	c := &model.Course{
		InstructorID:       req.InstructorID,
		Slug:               req.Slug,
		Title:              req.Title,
		Subtitle:           req.Subtitle,
		Description:        req.Description,
		Prerequisites:      req.Prerequisites,
		Objectives:         req.Objectives,
		MetaTitle:          req.MetaTitle,
		MetaDescription:    req.MetaDescription,
		MetaKeywords:       req.MetaKeywords,
		ThumbnailURL:       req.ThumbnailURL,
		TrailerURL:         req.TrailerURL,
		Level:              req.Level,
		ContentType:        req.ContentType,
		Language:           req.Language,
		IsFree:             req.IsFree,
		SubscriptionOnly:   req.SubscriptionOnly,
		PriceCents:         req.PriceCents,
		Currency:           req.Currency,
		IsPublished:        req.IsPublished,
		CertificateEnabled: req.CertificateEnabled,
		SortOrder:          req.SortOrder,
	}

	if err := h.course.Create(r.Context(), c, req.CategoryIDs, req.PrimaryCategoryID, req.TagIDs); err != nil {
		return toFault(err)
	}

	return httputil.Created(w, c)
}

// ============================================================================
// Update
// ============================================================================

// UpdateCourseRequest holds the fields required to update a course.
type UpdateCourseRequest struct {
	Slug               string            `json:"slug"                 validate:"required,max=255"`
	Title              string            `json:"title"                validate:"required,max=255"`
	Subtitle           *string           `json:"subtitle"             validate:"omitempty,max=500"`
	Description        *string           `json:"description"          validate:"omitempty"`
	Prerequisites      *string           `json:"prerequisites"        validate:"omitempty"`
	Objectives         *string           `json:"objectives"           validate:"omitempty"`
	MetaTitle          *string           `json:"meta_title"           validate:"omitempty,max=255"`
	MetaDescription    *string           `json:"meta_description"     validate:"omitempty,max=500"`
	MetaKeywords       *string           `json:"meta_keywords"        validate:"omitempty,max=255"`
	ThumbnailURL       *string           `json:"thumbnail_url"        validate:"omitempty,max=512"`
	TrailerURL         *string           `json:"trailer_url"          validate:"omitempty,max=512"`
	Level              model.CourseLevel `json:"level"                validate:"required,oneof=beginner intermediate advanced expert"`
	ContentType        model.ContentType `json:"content_type"         validate:"required,oneof=video article audio live document mixed"`
	Language           string            `json:"language"             validate:"required,max=10"`
	IsFree             bool              `json:"is_free"`
	SubscriptionOnly   bool              `json:"subscription_only"`
	PriceCents         int               `json:"price_cents"          validate:"min=0"`
	Currency           string            `json:"currency"             validate:"required,len=3"`
	IsPublished        bool              `json:"is_published"`
	IsFeatured         bool              `json:"is_featured"`
	CertificateEnabled bool              `json:"certificate_enabled"`
	SortOrder          int16             `json:"sort_order"`
}

// Update handles PUT /api/v1/courses/{id}
//
// @Summary   Update a course
// @Tags      courses
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path      string               true  "Course ID"
// @Param     body  body      UpdateCourseRequest  true  "Course payload"
// @Success   200   {object}  model.Course
// @Failure   400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404   {object}  fault.ErrorResponse  "course not found"
// @Router    /api/v1/courses/{id} [put]
func (h *CourseHandler) Update(w http.ResponseWriter, r *http.Request) error {
	var req UpdateCourseRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid course id", nil)
	}

	c, err := h.course.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}

	c.Slug = req.Slug
	c.Title = req.Title
	c.Subtitle = req.Subtitle
	c.Description = req.Description
	c.Prerequisites = req.Prerequisites
	c.Objectives = req.Objectives
	c.MetaTitle = req.MetaTitle
	c.MetaDescription = req.MetaDescription
	c.MetaKeywords = req.MetaKeywords
	c.ThumbnailURL = req.ThumbnailURL
	c.TrailerURL = req.TrailerURL
	c.Level = req.Level
	c.ContentType = req.ContentType
	c.Language = req.Language
	c.IsFree = req.IsFree
	c.SubscriptionOnly = req.SubscriptionOnly
	c.PriceCents = req.PriceCents
	c.Currency = req.Currency
	c.IsPublished = req.IsPublished
	c.IsFeatured = req.IsFeatured
	c.CertificateEnabled = req.CertificateEnabled
	c.SortOrder = req.SortOrder

	if err := h.course.Update(r.Context(), c); err != nil {
		return toFault(err)
	}

	return httputil.OK(w, c)
}

// ============================================================================
// SetCategories
// ============================================================================

// SetCategoriesRequest holds the category assignments for a course.
type SetCategoriesRequest struct {
	CategoryIDs       []string `json:"category_ids"        validate:"required,dive,uuid"`
	PrimaryCategoryID string   `json:"primary_category_id" validate:"omitempty,uuid"`
}

// SetCategories handles PUT /api/v1/courses/{id}/categories
//
// @Summary   Replace all category assignments for a course
// @Tags      courses
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path  string                true  "Course ID"
// @Param     body  body  SetCategoriesRequest  true  "Categories payload"
// @Success   204
// @Failure   400  {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404  {object}  fault.ErrorResponse  "course not found"
// @Router    /api/v1/courses/{id}/categories [put]
func (h *CourseHandler) SetCategories(w http.ResponseWriter, r *http.Request) error {
	var req SetCategoriesRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid course id", nil)
	}
	if err := h.course.SetCategories(r.Context(), id, req.CategoryIDs, req.PrimaryCategoryID); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// SetTags
// ============================================================================

// SetTagsRequest holds the tag assignments for a course.
type SetTagsRequest struct {
	TagIDs []string `json:"tag_ids" validate:"required,dive,uuid"`
}

// SetTags handles PUT /api/v1/courses/{id}/tags
//
// @Summary   Replace all tag assignments for a course
// @Tags      courses
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     id    path  string          true  "Course ID"
// @Param     body  body  SetTagsRequest  true  "Tags payload"
// @Success   204
// @Failure   400  {object}  fault.ErrorResponse  "invalid request body"
// @Failure   404  {object}  fault.ErrorResponse  "course not found"
// @Router    /api/v1/courses/{id}/tags [put]
func (h *CourseHandler) SetTags(w http.ResponseWriter, r *http.Request) error {
	var req SetTagsRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid course id", nil)
	}
	if err := h.course.SetTags(r.Context(), id, req.TagIDs); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/courses/{id}
//
// @Summary   Soft-delete a course
// @Tags      courses
// @Produce   json
// @Security  BearerAuth
// @Param     id  path  string  true  "Course ID"
// @Success   204
// @Failure   401  {object}  fault.ErrorResponse  "missing or invalid token"
// @Failure   404  {object}  fault.ErrorResponse  "course not found"
// @Router    /api/v1/courses/{id} [delete]
func (h *CourseHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid course id", nil)
	}
	if err := h.course.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
