// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// LoginStatus represents the outcome of a login attempt.
type LoginStatus string

// Login status values recorded in audit logs.
const (
	LoginStatusSuccess          LoginStatus = "success"
	LoginStatusFailedPassword   LoginStatus = "failed_password"
	LoginStatusFailedLocked     LoginStatus = "failed_locked"
	LoginStatusFailedNotFound   LoginStatus = "failed_not_found"
	LoginStatusFailedUnverified LoginStatus = "failed_unverified"
)

// LoginAuditLog records a single login attempt, successful or not.
// This table is append-only — rows are never updated after insert.
type LoginAuditLog struct {
	ID        string      `db:"id"`
	UserID    *string     `db:"user_id"`
	Email     string      `db:"email"`
	IPAddress string      `db:"ip_address"`
	UserAgent string      `db:"user_agent"`
	Status    LoginStatus `db:"status"`
	CreatedAt time.Time   `db:"created_at"`
}
