// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// SubscriptionPlan identifies the billing period of a subscription.
type SubscriptionPlan string

// Subscription plan values.
const (
	SubscriptionPlanMonthly SubscriptionPlan = "monthly"
	SubscriptionPlanAnnual  SubscriptionPlan = "annual"
)

// SubscriptionStatus identifies the current state of a subscription.
type SubscriptionStatus string

// Subscription status values.
const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
)

// Subscription represents a user subscription granting full catalog access.
type Subscription struct {
	ID          string             `db:"id"           json:"id"`
	UserID      string             `db:"user_id"      json:"user_id"`
	Plan        SubscriptionPlan   `db:"plan"         json:"plan"`
	Status      SubscriptionStatus `db:"status"       json:"status"`
	StartedAt   time.Time          `db:"started_at"   json:"started_at"`
	ExpiresAt   time.Time          `db:"expires_at"   json:"expires_at"`
	CancelledAt *time.Time         `db:"cancelled_at" json:"cancelled_at"`
}
