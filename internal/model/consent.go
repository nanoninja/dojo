// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// ConsentType identifies the category of consent recorded.
type ConsentType string

// Consent type values.
const (
	ConsentTypeTermsOfService    = "terms_of_service"
	ConsentTypePrivacyPolicy     = "privacy_policy"
	ConsentTypeCookieAnalytics   = "cookie_analytics"
	ConsentTypeCookieMarketing   = "cookie_marketing"
	ConsentTypeCookiePreferences = "cookie_preferences"
	ConsentTypeThirdParty        = "cookie_social"
)

// ConsentSource identifies where the consent action was collected.
type ConsentSource string

// Consent source values.
const (
	ConsentSourceRegistration       = "registration"
	ConsentSourceBanner             = "banner"
	ConsentSourceSettings           = "settings"
	ConsentSourceCheckout           = "checkout"
	ConsentSourceUpdateNotification = "update_notification"
)

// Consent represents a single GDPR consent action recorded for a user.
type Consent struct {
	ID         string        `db:"id"          json:"id"`
	UserID     string        `db:"user_id"     json:"user_id"`
	Type       ConsentType   `db:"type"        json:"type"`
	Version    *string       `db:"version"     json:"version"`
	IsAccepted bool          `db:"is_accepted" json:"is_accepted"`
	IPAddress  *string       `db:"ip_address"  json:"ip_address"`
	UserAgent  string        `db:"user_agent"  json:"user_agent"`
	Source     ConsentSource `db:"source"      json:"source"`
	CreatedAt  time.Time     `db:"created_at"  json:"created_at"`
}
