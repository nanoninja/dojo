// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Chapter represents a chapter within a course.
type Chapter struct {
	ID              string     `db:"id"               json:"id"`
	CourseID        string     `db:"course_id"        json:"course_id"`
	Title           string     `db:"title"            json:"title"`
	Slug            string     `db:"slug"             json:"slug"`
	Description     *string    `db:"description"      json:"description"`
	SortOrder       int16      `db:"sort_order"       json:"sort_order"`
	IsFree          bool       `db:"is_free"          json:"is_free"`
	IsPublished     bool       `db:"is_published"     json:"is_published"`
	DurationMinutes int        `db:"duration_minutes" json:"duration_minutes"`
	CreatedAt       time.Time  `db:"created_at"       json:"created_at"`
	UpdatedAt       *time.Time `db:"updated_at"       json:"updated_at"`
}
