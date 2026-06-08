// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// PurchaseType identifies whether a purchase targets a course or a bundle.
type PurchaseType string

// Purchase type values.
const (
	PurchaseTypeCourse PurchaseType = "course"
	PurchaseTypeBundle PurchaseType = "bundle"
)

// PurchaseStatus identifies the current state of a purchase.
type PurchaseStatus string

// Purchase status values.
const (
	PurchaseStatusPending   PurchaseStatus = "pending"
	PurchaseStatusCompleted PurchaseStatus = "completed"
	PurchaseStatusFailed    PurchaseStatus = "failed"
	PurchaseStatusRefunded  PurchaseStatus = "refunded"
	PurchaseStatusDisputed  PurchaseStatus = "disputed"
)

// Purchase represents a one-time payment for a course or bundle.
type Purchase struct {
	ID                string         `db:"id"                  json:"id"`
	UserID            string         `db:"user_id"             json:"user_id"`
	Type              PurchaseType   `db:"type"                json:"type"`
	ItemID            string         `db:"item_id"             json:"item_id"`
	Status            PurchaseStatus `db:"status"              json:"status"`
	AmountCents       int64          `db:"amount_cents"        json:"amount_cents"`
	Currency          string         `db:"currency"            json:"currency"`
	Provider          string         `db:"provider"            json:"provider"`
	ProviderSessionID string         `db:"provider_session_id" json:"provider_session_id"`
	ProviderPaymentID string         `db:"provider_payment_id" json:"provider_payment_id"`
	RefundedAt        *time.Time     `db:"refunded_at"         json:"refunded_at"`
	CreatedAt         time.Time      `db:"created_at"          json:"created_at"`
	CheckoutURL       string         `db:"-"                   json:"checkout_url,omitempty"`
}
