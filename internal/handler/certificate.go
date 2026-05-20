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

// CertificateHandler handles HTTP requests for certificate endpoints.
type CertificateHandler struct {
	certificate service.CertificateService
}

// NewCertificateHandler creates a CertificateHandler backed by the given service.
func NewCertificateHandler(certificate service.CertificateService) *CertificateHandler {
	return &CertificateHandler{certificate: certificate}
}

// ============================================================================
// ListByUser
// ============================================================================

// ListByUser handles GET /api/v1/certificates
//
// @Summary   List certificates for the authenticated user
// @Tags      certificates
// @Produce   json
// @Security  BearerAuth
// @Success   200  {array}   model.Certificate
// @Failure   401  {object}  fault.ErrorResponse
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/certificates [get]
func (h *CertificateHandler) ListByUser(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	certificates, err := h.certificate.ListByUser(r.Context(), userID)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, certificates)
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID handles GET /api/v1/certificates/{id}
//
// @Summary   Get a certificate by ID
// @Tags      certificates
// @Produce   json
// @Security  BearerAuth
// @Param     id  path  string  true  "Certificate ID"
// @Success   200  {object}  model.Certificate
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   401  {object}  fault.ErrorResponse
// @Failure   404  {object}  fault.ErrorResponse "certificate not found"
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/certificates/{id} [get]
func (h *CertificateHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		return fault.Unauthorized(errors.New("missing user id"))
	}
	id := chi.URLParam(r, "id")
	if !httputil.ValidateUUID(id) {
		return fault.BadRequest("invalid certificate id", nil)
	}
	certificate, err := h.certificate.GetByID(r.Context(), id)
	if err != nil {
		return toFault(err)
	}
	if certificate.UserID != userID {
		return toFault(service.ErrCertificateNotFound)
	}
	return httputil.OK(w, certificate)
}

// ============================================================================
// Verify
// ============================================================================

// Verify handles GET /api/v1/certificates/verify/{uuid}
//
// @Summary   Verify a certificate by its public UUID
// @Tags      certificates
// @Produce   json
// @Param     uuid  path  string  true  "Certificate public UUID"
// @Success   200  {object}  model.Certificate
// @Failure   400  {object}  fault.ErrorResponse
// @Failure   404  {object}  fault.ErrorResponse "certificate not found"
// @Failure   500  {object}  fault.ErrorResponse
// @Router    /api/v1/certificates/verify/{uuid} [get]
func (h *CertificateHandler) Verify(w http.ResponseWriter, r *http.Request) error {
	uuid := chi.URLParam(r, "uuid")
	if !httputil.ValidateUUID(uuid) {
		return fault.BadRequest("invalid certificate uuid", nil)
	}
	certificate, err := h.certificate.GetByUUID(r.Context(), uuid)
	if err != nil {
		return toFault(err)
	}
	return httputil.OK(w, certificate)
}
