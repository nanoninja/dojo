// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Certificate represents a completion certificate earned by a user for a course.
type Certificate struct {
	ID       string    `json:"id"        db:"id"`
	UserID   string    `json:"user_id"   db:"user_id"`
	CourseID string    `json:"course_id" db:"course_id"`
	UUID     string    `json:"uuid"      db:"uuid"`
	IssuedAt time.Time `json:"issued_at" db:"issued_at"`
}
