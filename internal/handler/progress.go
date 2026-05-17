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

// ProgressHandler handles HTTP requests for lesson progress tracking.
type ProgressHandler struct {
	progress service.LessonProgressService
}

// NewProgressHandler returns a new ProgressHandler.
func NewProgressHandler(progress service.LessonProgressService) *ProgressHandler {
	return &ProgressHandler{progress: progress}
}

// ============================================================================
// Get
// ============================================================================

// Get handles GET /api/v1/progress/{user_id}/lessons/{lesson_id}
//
// @Summary  Get a user's progress on a specific lesson
// @Tags     progress
// @Produce  json
// @Security BearerAuth
// @Param    user_id    path      string  true  "User ID"
// @Param    lesson_id  path      string  true  "Lesson ID"
// @Success  200  {object}  model.LessonProgress
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "progress not found"
// @Router   /api/v1/progress/{user_id}/lessons/{lesson_id} [get]
func (h *ProgressHandler) Get(w http.ResponseWriter, r *http.Request) error {
	userID := chi.URLParam(r, "user_id")
	if !httputil.ValidateUUID(userID) {
		return fault.BadRequest("invalid user_id", nil)
	}
	lessonID := chi.URLParam(r, "lesson_id")
	if !httputil.ValidateUUID(lessonID) {
		return fault.BadRequest("invalid lesson_id", nil)
	}
	p, err := h.progress.Get(r.Context(), userID, lessonID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, p)
}

// ============================================================================
// ListByCourse
// ============================================================================

// ListByCourse handles GET /api/v1/progress/{user_id}/courses/{course_id}
//
// @Summary  List a user's lesson progress for a course
// @Tags     progress
// @Produce  json
// @Security BearerAuth
// @Param    user_id    path      string  true  "User ID"
// @Param    course_id  path      string  true  "Course ID"
// @Success  200  {array}   model.LessonProgress
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Router   /api/v1/progress/{user_id}/courses/{course_id} [get]
func (h *ProgressHandler) ListByCourse(w http.ResponseWriter, r *http.Request) error {
	userID := chi.URLParam(r, "user_id")
	if !httputil.ValidateUUID(userID) {
		return fault.BadRequest("invalid user_id", nil)
	}
	courseID := chi.URLParam(r, "course_id")
	if !httputil.ValidateUUID(courseID) {
		return fault.BadRequest("invalide course_id", nil)
	}
	records, err := h.progress.ListByCourse(r.Context(), userID, courseID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, records)
}

// ============================================================================
// Save
// ============================================================================

// SaveProgressRequest holds the fields required to record lesson progress.
type SaveProgressRequest struct {
	UserID         string `json:"user_id"         validate:"required,uuid"`
	LessonID       string `json:"lesson_id"       validate:"required,uuid"`
	CourseID       string `json:"course_id"       validate:"required,uuid"`
	IsCompleted    bool   `json:"is_completed"`
	WatchedSeconds int    `json:"watched_seconds" validate:"min=0"`
}

// Save handles POST /api/v1/progress
//
// @Summary  Record or update progress on a lesson
// @Tags     progress
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      SaveProgressRequest  true  "Progress payload"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid request body"
// @Router   /api/v1/progress [post]
func (h *ProgressHandler) Save(w http.ResponseWriter, r *http.Request) error {
	var req SaveProgressRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	p := &model.LessonProgress{
		UserID:         req.UserID,
		LessonID:       req.LessonID,
		IsCompleted:    req.IsCompleted,
		WatchedSeconds: req.WatchedSeconds,
	}
	if err := h.progress.Save(r.Context(), p, req.CourseID); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
