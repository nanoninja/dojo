// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package payment

import (
	"context"
	"errors"
)

// Provider is the interface that payment backends must implement.
type Provider interface {
	// CreateCheckout initiates a payment session and returns a redirect URL
	// for the user to complete payment on the provider's hosted page.
	CreateCheckout(ctx context.Context, order Order) (Session, error)

	// HandleWebhook parses and validates an inbound webhook payload.
	// signature is the provider-specific header value used to verify authenticity.
	HandleWebhook(payload []byte, signature string) (Event, error)

	// Refund issues a full or partial refund for a completed payment.
	Refund(ctx context.Context, paymentID string, amountCents int64) error
}

// Order hold the information needed to create a checkout session.
type Order struct {
	Currency      string
	Items         []LineItem
	CustomerEmail string

	// Metadata is passed through to the provider and returned in webhook events.
	// Use it to carry purchase_id, user_id, and any other context needed to
	// process the webhook without a database lookup on the session ID.
	Metadata map[string]string
}

// LineItem represents a single product in an order.
type LineItem struct {
	Name        string
	Description string
	AmountCents int64
}

// Session is returned by CreateCheckout.
type Session struct {
	ID  string // ID is the provider's session identifier (e.g. Stripe checkout session ID).
	URL string // URL is the hosted payment page the user must be redirected to.
}

// EventType classifies inbound webhook events into provide-agnostic categories.
type EventType string

const (
	// EventPaymentSucceeded is fired when a payment is confirmed by the provider.
	EventPaymentSucceeded EventType = "payment_succeeded"

	// EventPaymentFailed is fired when a payment attempt is declined or times out.
	EventPaymentFailed EventType = "payment_failed"

	// EventRefundSucceeded is fired when a refund is confirmed by the provider.
	EventRefundSucceeded EventType = "refund_succeeded"
)

// Event is the normalised representation of a provider webhook event.
type Event struct {
	Type EventType

	// PaymentID is the provider's payment intent or transaction ID.
	PaymentID string

	// SessionID is the checkout session ID that originated this payment.
	SessionID string

	// Metadata contains the key/value pairs set when creating the checkout session.
	Metadata map[string]string
}

// ProviderStripe is the identifier for the Stripe payment provider.
const ProviderStripe = "stripe"

// ErrInvalidSignature is returned by HandleWebhook when the provider signature
// cannot be verified.
var ErrInvalidSignature = errors.New("payment: invalid webhook signature")
