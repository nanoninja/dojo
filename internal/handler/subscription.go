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
)

// SubscriptionHandler handles HTTP requests for user subscriptions.
type SubscriptionHandler struct {
	subscription service.SubscriptionService
}

// NewSubscriptionHandler returns a new SubscriptionHandler.
func NewSubscriptionHandler(subscription service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscription: subscription}
}

// ============================================================================
// GetActive
// ============================================================================

// GetActive handles GET /api/v1/subscriptions/active
//
// @Summary  Get the current active subscription for the authenticated user
// @Tags     subscriptions
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  model.Subscription
// @Failure  404  {object}  fault.ErrorResponse  "no active subscription"
// @Router   /api/v1/subscriptions/active [get]
func (h *SubscriptionHandler) GetActive(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	sub, err := h.subscription.GetActive(r.Context(), userID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, sub)
}

// ============================================================================
// List
// ============================================================================

// List handles GET /api/v1/subscriptions
//
// @Summary  List all subscriptions for the authenticated user
// @Tags     subscriptions
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  []model.Subscription
// @Failure  500  {object}  fault.ErrorResponse
// @Router   /api/v1/subscriptions [get]
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	subs, err := h.subscription.ListByUser(r.Context(), userID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, subs)
}

// ============================================================================
// Subscribe
// ============================================================================

// SubscribeRequest holds the fields required to create a subscription.
type SubscribeRequest struct {
	Plan model.SubscriptionPlan `json:"plan" validate:"required,oneof=monthly annual"`
}

// Subscribe handles POST /api/v1/subscriptions
//
// @Summary  Subscribe the authenticated user to a plan
// @Tags     subscriptions
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      SubscribeRequest  true  "Subscription payload"
// @Success  201   {object}  model.Subscription
// @Failure  400   {object}  fault.ErrorResponse  "invalid request body"
// @Router   /api/v1/subscriptions [post]
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	var req SubscribeRequest
	if err := httputil.Bind(r, &req); err != nil {
		return err
	}
	sub, err := h.subscription.Subscribe(r.Context(), userID, req.Plan)
	if err != nil {
		return toFault(err)
	}
	return httputil.Created(w, sub)
}

// ============================================================================
// Cancel
// ============================================================================

// Cancel handles DELETE /api/v1/subscriptions/{id}
//
// @Summary  Cancel a subscription
// @Tags     subscriptions
// @Security BearerAuth
// @Param    id  path  string  true  "Subscription ID"
// @Success  204
// @Failure  400  {object}  fault.ErrorResponse  "invalid uuid"
// @Failure  404  {object}  fault.ErrorResponse  "subscription not found"
// @Router   /api/v1/subscriptions/{id} [delete]
func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid subscription id", nil)
	}
	if err := h.subscription.Cancel(r.Context(), id); err != nil {
		return toFault(err)
	}
	httputil.NoContent(w)
	return nil
}
