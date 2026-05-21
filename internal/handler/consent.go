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

// ConsentHandler handles HTTP requests for GPRD consent records.
type ConsentHandler struct {
	consent service.ConsentService
}

// NewConsentHandler creates a ConsentHandler with the given service.
func NewConsentHandler(consent service.ConsentService) *ConsentHandler {
	return &ConsentHandler{consent: consent}
}

// ListByUser handles GET /api/v1/consents
//
// @Summary   List consent records for the authenticated user
// @Tags      consents
// @Produce   json
// @Security  BearerAuth
// @Success   200  {array}   model.Consent
// @Failure   401  {object}  fault.ErrorResponse
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/consents [get]
func (h *ConsentHandler) ListByUser(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	consents, err := h.consent.ListByUser(r.Context(), userID)
	if err != nil {
		return err
	}
	return httputil.OK(w, consents)
}

// GetByID handles GET /api/v1/consents/{id}
//
// @Summary   Get a consent record by ID
// @Tags      consents
// @Produce   json
// @Security  BearerAuth
// @Param     id  path  string  true  "Consent ID"
// @Success   200  {object}  model.Consent
// @Failure   401  {object}  fault.ErrorResponse
// @Failure   404  {object}  fault.ErrorResponse "consent not found"
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/consents/{id} [get]
func (h *ConsentHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	id := chi.URLParam(r, "id")
	c, err := h.consent.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	if c.UserID != userID {
		return toFault(service.ErrConsentNotFound)
	}
	return httputil.OK(w, c)
}

// Create handles POST /api/v1/consents
//
// @Summary   Record a new consent action
// @Tags      consents
// @Accept    json
// @Produce   json
// @Security  BearerAuth
// @Param     consent  body      model.Consent  true  "Consent payload"
// @Success   201  {object}  model.Consent
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   401  {object}  fault.ErrorResponse
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/consents [post]
func (h *ConsentHandler) Create(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	var c model.Consent
	if err := httputil.Bind(r, &c); err != nil {
		return err
	}
	c.UserID = userID

	if err := h.consent.Create(r.Context(), &c); err != nil {
		return err
	}

	return httputil.Created(w, c)
}
