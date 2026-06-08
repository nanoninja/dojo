// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/nanoninja/dojo/internal/payment"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/checkout/session"
	"github.com/stripe/stripe-go/v85/refund"
	"github.com/stripe/stripe-go/v85/webhook"
)

// Config holds the Stripe-specific configuration.
type Config struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

// Client implements payment.Provider using Stripe Checkout Sessions.
type Client struct {
	cfg Config
}

// New returns a Stripe Client configured with the given credentials.
func New(cfg Config) *Client {
	stripe.Key = cfg.SecretKey
	return &Client{cfg: cfg}
}

// CreateCheckout creates a Stripe Checkout Session and returns the hosted payment URL.
func (c *Client) CreateCheckout(_ context.Context, order payment.Order) (payment.Session, error) {
	items := make([]*stripe.CheckoutSessionLineItemParams, len(order.Items))
	for i, item := range order.Items {
		items[i] = &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(order.Currency),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(item.Name),
					Description: stripe.String(item.Description),
				},
				UnitAmount: stripe.Int64(item.AmountCents),
			},
			Quantity: stripe.Int64(1),
		}
	}

	metadata := make(map[string]string, len(order.Metadata))
	maps.Copy(metadata, order.Metadata)

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems:          items,
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		CustomerEmail:      stripe.String(order.CustomerEmail),
		SuccessURL:         stripe.String(c.cfg.SuccessURL),
		CancelURL:          stripe.String(c.cfg.CancelURL),
		Metadata:           metadata,
	}

	s, err := session.New(params)
	if err != nil {
		return payment.Session{}, fmt.Errorf("stripe: create checkout session: %w", err)
	}

	return payment.Session{ID: s.ID, URL: s.URL}, nil
}

// HandleWebhook validates the Stripe webhook signature and returns a normalised Event.
func (c *Client) HandleWebhook(payload []byte, signature string) (payment.Event, error) {
	e, err := webhook.ConstructEvent(payload, signature, c.cfg.WebhookSecret)
	if err != nil {
		return payment.Event{}, fmt.Errorf("stripe: webhook signature: %w", err)
	}

	switch e.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var s stripe.CheckoutSession
		if err := json.Unmarshal(e.Data.Raw, &s); err != nil {
			return payment.Event{}, fmt.Errorf("stripe: unmarshal checkout session: %w", err)
		}
		return payment.Event{
			Type:      payment.EventPaymentSucceeded,
			PaymentID: s.PaymentIntent.ID,
			SessionID: s.ID,
			Metadata:  s.Metadata,
		}, nil

	case stripe.EventTypeCheckoutSessionExpired:
		var s stripe.CheckoutSession
		if err := json.Unmarshal(e.Data.Raw, &s); err != nil {
			return payment.Event{}, fmt.Errorf("stripe: unmarshal checkout session: %w", err)
		}
		return payment.Event{
			Type:      payment.EventPaymentFailed,
			SessionID: s.ID,
			Metadata:  s.Metadata,
		}, nil

	case stripe.EventTypeChargeRefunded:
		var ch stripe.Charge
		if err := json.Unmarshal(e.Data.Raw, &ch); err != nil {
			return payment.Event{}, fmt.Errorf("stripe: unmarshal charge: %w", err)
		}
		return payment.Event{
			Type:      payment.EventRefundSucceeded,
			PaymentID: ch.PaymentIntent.ID,
		}, nil

	default:
		return payment.Event{}, fmt.Errorf("stripe: unhandled event type: %s", e.Type)
	}
}

// Refund issues a full or partial refund via Stripe.
func (c *Client) Refund(_ context.Context, paymentID string, amountCents int64) error {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentID),
		Amount:        stripe.Int64(amountCents),
	}
	if _, err := refund.New(params); err != nil {
		return fmt.Errorf("stripe: refund: %w", err)
	}
	return nil
}
