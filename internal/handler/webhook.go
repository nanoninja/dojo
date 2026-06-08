// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"io"
	"net/http"

	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/payment"
	"github.com/nanoninja/dojo/internal/service"
)

// WebhookHandler handles inbound payment provider webhooks.
type WebhookHandler struct {
	provider payment.Provider
	purchase service.PurchaseService
}

// NewWebhookHandler returns a new WebhookHandler.
func NewWebhookHandler(provider payment.Provider, purchase service.PurchaseService) *WebhookHandler {
	return &WebhookHandler{provider: provider, purchase: purchase}
}

// Stripe handles POST /webhooks/stripe
//
// @Summary  Receive and process Stripe webhook events
// @Tags     webhooks
// @Accept   application/octet-stream
// @Produce  json
// @Param    Stripe-Signature  header  string  true  "Stripe webhook signature"
// @Success  200
// @Failure  400  {object}  fault.ErrorResponse  "invalid payload or signature"
// @Failure  500  {object}  fault.ErrorResponse
// @Router   /webhooks/stripe [post]
func (h *WebhookHandler) Stripe(w http.ResponseWriter, r *http.Request) error {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		return fault.BadRequest("cannot read request body", err)
	}
	sig := r.Header.Get("Stripe-Signature")
	if sig == "" {
		return fault.BadRequest("missing Stripe-Signature header", nil)
	}
	event, err := h.provider.HandleWebhook(payload, sig)
	if err != nil {
		return fault.BadRequest("invalid webhook payload or signature", err)
	}
	switch event.Type {
	case payment.EventPaymentSucceeded:
		purchaseID := event.Metadata["purchase_id"]
		if purchaseID == "" {
			return fault.BadRequest("missing purchase_id in webhook metadata", nil)
		}
		if err := h.purchase.ConfirmPayment(r.Context(), purchaseID, event.PaymentID); err != nil {
			return toFault(err)
		}
	}
	httputil.NoContent(w)
	return nil
}
