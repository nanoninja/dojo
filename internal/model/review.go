// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Review represents a rating and optional comment left by a user on a course.
type Review struct {
	ID        string     `db:"id"         json:"id"`
	UserID    string     `db:"user_id"    json:"user_id"`
	CourseID  string     `db:"course_id"  json:"course_id"`
	Rating    int        `db:"rating"     json:"rating"`
	Comment   string     `db:"comment"    json:"comment"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}
