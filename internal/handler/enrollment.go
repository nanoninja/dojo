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

// EnrollmentHandler handles HTTP requests for course enrollments.
type EnrollmentHandler struct {
	enrollment service.EnrollmentService
}

// NewEnrollmentHandler returns a new EnrollmentHandler.
func NewEnrollmentHandler(enrollment service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{enrollment: enrollment}
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/enrollments
//
// @Summary  List enrollments with optional filters
// @Tags     enrollments
// @Produce  json
// @Security BearerAuth
// @Param    user_id    query    string  false  "Filter by user"
// @Param    course_id  query    string  false  "Filter by course"
// @Param    status     query    string  false  "Filter by status"  Enums(active,completed,expired,refunded)
// @Param    page       query    int     false  "Page number"    default(1)
// @Param    limit      query    int     false  "Items per page" default(20)
// @Success  200  {array}   model.CourseEnrollment
// @Failure  400  {object}  fault.ErrorResponse
// @Failure  500  {object}  fault.ErrorResponse
// @Router   /api/v1/enrollments [get]
func (h *EnrollmentHandler) List(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()

	page := parseIntQuery(q.Get("page"), 1)
	limit := min(parseIntQuery(q.Get("limit"), 20), maxPageLimit)

	f := store.EnrollmentFilter{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	if v := q.Get("user_id"); v != "" {
		if !httputil.ValidateUUID(v) {
			return fault.BadRequest("invalid user_id", nil)
		}
		f.UserID = v
	}
	if v := q.Get("course_id"); v != "" {
		if !httputil.ValidateUUID(v) {
			return fault.BadRequest("invalid course_id", nil)
		}
		f.CourseID = v
	}
	if v := q.Get("status"); v != "" {
		status := model.EnrollmentStatus(v)
		switch status {
		case model.EnrollmentStatusActive,
			model.EnrollmentStatusCompleted,
			model.EnrollmentStatusExpired,
			model.EnrollmentStatusRefunded:
		default:
			return fault.BadRequest("invalid status", nil)
		}
		f.Status = status
	}

	enrollments, err := h.enrollment.List(r.Context(), f)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, enrollments)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/enrollments/{id}
//
// @Summary  Get an enrollment by ID
// @Tags     enrollments
// @Produce  json
// @Security BearerAuth
// @Param    id   path      string  true  "Enrollment ID"
// @Success  200  {object}  model.CourseEnrollment
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "enrollment not found"
// @Router   /api/v1/enrollments/{id} [get]
func (h *EnrollmentHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid enrollment id", nil)
	}
	e, err := h.enrollment.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, e)
}

// ============================================================================
// Enroll
// ============================================================================

// EnrollRequest holds the fields required to enroll a user in a course.
type EnrollRequest struct {
	UserID   string `json:"user_id"   validate:"required,uuid"`
	CourseID string `json:"course_id" validate:"required,uuid"`
}

// Enroll handles POST /api/v1/enrollments
//
// @Summary  Enroll a user in a course
// @Tags     enrollments
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      EnrollRequest  true  "Enrollment payload"
// @Success  201   {object}  model.CourseEnrollment
// @Failure  400   {object}  fault.ErrorResponse  "invalid request body"
// @Failure  409   {object}  fault.ErrorResponse  "user already enrolled"
// @Router   /api/v1/enrollments [post]
func (h *EnrollmentHandler) Enroll(w http.ResponseWriter, r *http.Request) error {
	var req EnrollRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	e, err := h.enrollment.Enroll(r.Context(), req.UserID, req.CourseID)
	if err != nil {
		return toFault(err)
	}
	return httputil.Created(w, e)
}

// ============================================================================
// UpdateStatus
// ============================================================================

// UpdateStatusRequest holds the new status for an enrollment.
type UpdateStatusRequest struct {
	Status model.EnrollmentStatus `json:"status" validate:"required,oneof=active completed expired refunded"`
}

// UpdateStatus handles PATCH /api/v1/enrollments/{id}/status
//
// @Summary  Update the status of an enrollment
// @Tags     enrollments
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path  string               true  "Enrollment ID"
// @Param    body  body  UpdateStatusRequest  true  "Status payload"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid request body"
// @Failure  404  {object}  fault.ErrorResponse  "enrollment not found"
// @Router   /api/v1/enrollments/{id}/status [patch]
func (h *EnrollmentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid enrollment id", nil)
	}
	var req UpdateStatusRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	if err := h.enrollment.UpdateStatus(r.Context(), id, req.Status); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}

// ============================================================================
// Delete
// ============================================================================

// Delete handles DELETE /api/v1/enrollments/{id}
//
// @Summary  Delete an enrollment
// @Tags     enrollments
// @Security BearerAuth
// @Param    id  path  string  true  "Enrollment ID"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "enrollment not found"
// @Router   /api/v1/enrollments/{id} [delete]
func (h *EnrollmentHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid enrollment id", nil)
	}
	if err := h.enrollment.Delete(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
