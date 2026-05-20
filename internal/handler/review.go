// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

//nolint:unused
type reviewPageResponse = httputil.PageResponse[model.Review]

// ReviewHandler handles HTTP requests for course review endpoints.
type ReviewHandler struct {
	review service.ReviewService
}

// NewReviewHandler creates a ReviewHandler backed by the given service.
func NewReviewHandler(review service.ReviewService) *ReviewHandler {
	return &ReviewHandler{review: review}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/courses/{course_id}/reviews
//
// @Summary   List reviews for a course
// @Tags      reviews
// @Produce   json
// @Security  BearerAuth
// @Param     course_id  path   string  true   "Course ID"
// @Param     sort       query  string  false  "Sort order"    Enums(asc,desc)
// @Param     page       query  int     false  "Page number"   default(1)
// @Param     limit      query  int     false  "Items per page" default(20)
// @Success   200  {object}  reviewPageResponse
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/courses/{course_id}/reviews [get]
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) error {
	courseID := chi.URLParam(r, "course_id")
	if !httputil.ValidateUUID(courseID) {
		return fault.BadRequest("invalid course id", nil)
	}

	q := r.URL.Query()
	page, limit := parsePage(q)

	sortDir := store.SortDirDesc
	if s := q.Get("sort"); s == "asc" {
		sortDir = store.SortDirAsc
	}

	filter := store.ReviewFilter{
		CourseID: courseID,
		SortDir:  sortDir,
		Limit:    limit,
		Offset:   (page - 1) * limit,
	}

	reviews, total, err := h.review.List(r.Context(), filter)
	if err != nil {
		return toFault(err)
	}

	return httputil.OKPaginated(w, reviews, page, limit, total)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/courses/{course_id}/reviews/{id}
//
// @Summary   Get a review by ID
// @Tags      reviews
// @Produce   json
// @Security  BearerAuth
// @Param     course_id  path  string  true  "Course ID"
// @Param     id         path  string  true  "Review ID"
// @Success   200  {object}  model.Review
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   404  {object}  fault.ErrorResponse "review not found"
// @Router    /api/v1/courses/{course_id}/reviews/{id} [get]
func (h *ReviewHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid review id", nil)
	}
	review, err := h.review.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, review)
}

// ============================================================================
// Create
// ============================================================================

// CreateReviewRequest holds the payload for creating a course review.
type CreateReviewRequest struct {
	Rating  int    `json:"rating"  validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"omitempty,min=5,max=500"`
}

// Create handles POST /api/v1/courses/{course_id}/reviews
//
// @Summary   Create a review for a course
// @Tags      reviews
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     course_id  path      string               true  "Course ID"
// @Param     body       body      CreateReviewRequest  true  "Review payload"
// @Success   201  {object}  model.Review
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   401  {object}  fault.ErrorResponse
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/courses/{course_id}/reviews [post]
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) error {
	courseID := chi.URLParam(r, "course_id")
	if !httputil.ValidateUUID(courseID) {
		return fault.BadRequest("invalid course id", nil)
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user i"))
	}

	var req CreateReviewRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	review := &model.Review{
		UserID:   userID,
		CourseID: courseID,
		Rating:   req.Rating,
		Comment:  req.Comment,
	}

	if err := h.review.Create(r.Context(), review); err != nil {
		return toFault(err)
	}

	return httputil.Created(w, review)
}

// ============================================================================
// Update
// ============================================================================

// UpdateReviewRequest holds the payload for updating a course review.
type UpdateReviewRequest struct {
	Rating  int    `json:"rating"  validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"omitempty,min=5,max=500"`
}

// Update handles PUT /api/v1/courses/{course_id}/reviews/{id}
//
// @Summary   Update a review
// @Tags      reviews
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     course_id  path      string               true  "Course ID"
// @Param     id         path      string               true  "Review ID"
// @Param     body       body      UpdateReviewRequest  true  "Review payload"
// @Success   200  {object}  model.Review
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   404  {object}  fault.ErrorResponse "review not found"
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/courses/{course_id}/reviews/{id} [put]
func (h *ReviewHandler) Update(w http.ResponseWriter, r *http.Request) error {
	var req UpdateReviewRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}

	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid review id", nil)
	}

	review, err := h.review.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}

	review.Comment = req.Comment

	if err := h.review.Update(r.Context(), review); err != nil {
		return toFault(err)
	}

	return httputil.OK(w, review)
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/courses/{course_id}/reviews/{id}
//
// @Summary   Delete a review
// @Tags      reviews
// @Security  BearerAuth
// @Param     course_id  path  string  true  "Course ID"
// @Param     id         path  string  true  "Review ID"
// @Success   204
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   404  {object}  fault.ErrorResponse "review not found"
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/courses/{course_id}/reviews/{id} [delete]
func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid review id", nil)
	}
	if err := h.review.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
