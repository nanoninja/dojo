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
	"github.com/nanoninja/dojo/internal/service"
)

// PurchaseHandler handles HTTP requests for one-time purchases.
type PurchaseHandler struct {
	purchase service.PurchaseService
}

// NewPurchaseHandler returns a new PurchaseHandler.
func NewPurchaseHandler(purchase service.PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{purchase: purchase}
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/purchases/{id}
//
// @Summary  Get a purchase by ID
// @Tags     purchases
// @Produce  json
// @Security BearerAuth
// @Param    id   path      string  true  "Purchase ID"
// @Success  200  {object}  model.Purchase
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "purchase not found"
// @Router   /api/v1/purchases/{id} [get]
func (h *PurchaseHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid purchase id", nil)
	}
	p, err := h.purchase.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, p)
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/purchases
//
// @Summary  List all purchases for the authenticated user
// @Tags     purchases
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  []model.Purchase
// @Failure  500  {object}  fault.ErrorResponse
// @Router   /api/v1/purchases [get]
func (h *PurchaseHandler) List(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	purchases, err := h.purchase.ListByUser(r.Context(), userID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, purchases)
}

// ============================================================================
// BuyCourse
// ============================================================================

// BuyCourseRequest holds the fields required to purchase a course.
type BuyCourseRequest struct {
	CourseID    string `json:"course_id"    validate:"required,uuid"`
	AmountCents int64  `json:"amount_cents" validate:"required,min=0"`
	Currency    string `json:"currency"     validate:"required,len=3"`
}

// BuyCourse handles POST /api/v1/purchases/courses
//
// @Summary  Purchase a course
// @Tags     purchases
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      BuyCourseRequest  true  "Purchase payload"
// @Success  201   {object}  model.Purchase
// @Failure  400   {object}  fault.ErrorResponse  "invalid request body"
// @Router   /api/v1/purchases/courses [post]
func (h *PurchaseHandler) BuyCourse(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	var req BuyCourseRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	p, err := h.purchase.BuyCourse(
		r.Context(),
		userID,
		req.CourseID,
		req.AmountCents,
		req.Currency,
	)
	if err != nil {
		return toFault(err)
	}
	return httputil.Created(w, p)
}

// ============================================================================
// BuyBundle
// ============================================================================

// BuyBundleRequest holds the fields required to purchase a bundle.
type BuyBundleRequest struct {
	BundleID    string `json:"bundle_id"    validate:"required,uuid"`
	AmountCents int64  `json:"amount_cents" validate:"required,min=0"`
	Currency    string `json:"currency"     validate:"required,len=3"`
}

// BuyBundle handles POST /api/v1/purchases/bundles
//
// @Summary  Purchase a bundle
// @Tags     purchases
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      BuyBundleRequest  true  "Purchase payload"
// @Success  201   {object}  model.Purchase
// @Failure  400   {object}  fault.ErrorResponse  "invalid request body"
// @Router   /api/v1/purchases/bundles [post]
func (h *PurchaseHandler) BuyBundle(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	var req BuyBundleRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	p, err := h.purchase.BuyBundle(
		r.Context(),
		userID,
		req.BundleID,
		req.AmountCents,
		req.Currency,
	)
	if err != nil {
		return toFault(err)
	}
	return httputil.Created(w, p)
}

// ============================================================================
// Refund
// ============================================================================

// Refund handles POST /api/v1/purchases/{id}/refund
//
// @Summary  Refund a purchase and cancel associated enrollments
// @Tags     purchases
// @Security BearerAuth
// @Param    id  path  string  true  "Purchase ID"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "purchase not found"
// @Router   /api/v1/purchases/{id}/refund [post]
func (h *PurchaseHandler) Refund(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid purchase id", nil)
	}
	if err := h.purchase.Refund(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
