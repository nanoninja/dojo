// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// EnrollmentStatus represents the lifecycle state of a course enrollment.
type EnrollmentStatus string

// Enrollment status values.
const (
	EnrollmentStatusActive    EnrollmentStatus = "active"
	EnrollmentStatusCompleted EnrollmentStatus = "completed"
	EnrollmentStatusExpired   EnrollmentStatus = "expired"
	EnrollmentStatusRefunded  EnrollmentStatus = "refunded"
)

// CourseEnrollment represents a user's registration to a course.
type CourseEnrollment struct {
	ID              string           `db:"id"                json:"id"`
	UserID          string           `db:"user_id"           json:"user_id"`
	CourseID        string           `db:"course_id"         json:"course_id"`
	Status          EnrollmentStatus `db:"status"            json:"status"`
	ProgressPercent float64          `db:"progress_percent"  json:"progress_percent"`
	LastAccessedAt  *time.Time       `db:"last_accessed_at"  json:"last_accessed_at"`
	EnrolledAt      time.Time        `db:"enrolled_at"       json:"enrolled_at"`
	CompletedAt     *time.Time       `db:"completed_at"      json:"completed_at"`
	ExpiresAt       *time.Time       `db:"expires_at"        json:"expires_at"`
}
