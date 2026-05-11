// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Chapter represents a chapter within a course.
type Chapter struct {
	ID              string     `db:"id"`
	CourseID        string     `db:"course_id"`
	Title           string     `db:"title"`
	Slug            string     `db:"slug"`
	Description     *string    `db:"description"`
	SortOrder       int16      `db:"sort_order"`
	IsFree          bool       `db:"is_free"`
	IsPublished     bool       `db:"is_published"`
	DurationMinutes int        `db:"duration_minutes"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       *time.Time `db:"updated_at"`
}
