// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/payment"
	"github.com/nanoninja/dojo/internal/service"
)

// ============================================================================
// mockPaymentProvider
// ============================================================================

type mockPaymentProvider struct {
	event payment.Event
	err   error
}

func (m *mockPaymentProvider) CreateCheckout(_ context.Context, _ payment.Order) (payment.Session, error) {
	return payment.Session{}, m.err
}

func (m *mockPaymentProvider) HandleWebhook(_ []byte, _ string) (payment.Event, error) {
	return m.event, m.err
}

func (m *mockPaymentProvider) Refund(_ context.Context, _ string, _ int64) error {
	return m.err
}

// ============================================================================
// helpers
// ============================================================================

func newWebhookHandler(p *mockPaymentProvider, s *mockPurchaseService) *handler.WebhookHandler {
	return handler.NewWebhookHandler(p, s)
}

func stripeRequest() *http.Request {
	r := httptest.NewRequest("POST", "/webhooks/stripe", nil)
	r.Header.Set("Stripe-Signature", "t=123,v1=abc")
	return r
}

// ============================================================================
// Stripe — EventPaymentSucceeded
// ============================================================================

func TestWebhookHandler_Stripe_PaymentSucceeded(t *testing.T) {
	p := &mockPaymentProvider{event: payment.Event{
		Type:      payment.EventPaymentSucceeded,
		PaymentID: "pi_test_123",
		Metadata:  map[string]string{"purchase_id": testPurchaseID},
	}}
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestWebhookHandler_Stripe_PaymentSucceeded_MissingPurchaseID(t *testing.T) {
	p := &mockPaymentProvider{event: payment.Event{
		Type:     payment.EventPaymentSucceeded,
		Metadata: map[string]string{},
	}}
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookHandler_Stripe_PaymentSucceeded_ServiceError(t *testing.T) {
	p := &mockPaymentProvider{event: payment.Event{
		Type:     payment.EventPaymentSucceeded,
		Metadata: map[string]string{"purchase_id": testPurchaseID},
	}}
	s := &mockPurchaseService{err: service.ErrPurchaseNotFound}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// Stripe — EventRefundSucceeded
// ============================================================================

func TestWebhookHandler_Stripe_RefundSucceeded(t *testing.T) {
	p := &mockPaymentProvider{event: payment.Event{
		Type:     payment.EventRefundSucceeded,
		Metadata: map[string]string{"purchase_id": testPurchaseID},
	}}
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestWebhookHandler_Stripe_RefundSucceeded_MissingPurchaseID(t *testing.T) {
	p := &mockPaymentProvider{event: payment.Event{
		Type:     payment.EventRefundSucceeded,
		Metadata: map[string]string{},
	}}
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookHandler_Stripe_RefundSucceeded_ServiceError(t *testing.T) {
	p := &mockPaymentProvider{event: payment.Event{
		Type:     payment.EventRefundSucceeded,
		Metadata: map[string]string{"purchase_id": testPurchaseID},
	}}
	s := &mockPurchaseService{err: service.ErrPurchaseAlreadyProcessed}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ============================================================================
// Stripe — invalid signature / payload
// ============================================================================

func TestWebhookHandler_Stripe_InvalidSignature(t *testing.T) {
	p := &mockPaymentProvider{err: payment.ErrInvalidSignature}
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()

	serve(newWebhookHandler(p, s).Stripe, w, stripeRequest())

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookHandler_Stripe_MissingSignatureHeader(t *testing.T) {
	p := &mockPaymentProvider{}
	s := &mockPurchaseService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/webhooks/stripe", nil) // no Stripe-Signature

	serve(newWebhookHandler(p, s).Stripe, w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
