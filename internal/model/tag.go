// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Tag represents a keyword tag that can be assigned to courses.
type Tag struct {
	ID        string    `db:"id"         json:"id"`
	Slug      string    `db:"slug"       json:"slug"`
	Name      string    `db:"name"       json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// CourseTagAssignment represents the many-to-many relation between a course and a tag.
type CourseTagAssignment struct {
	CourseID   string    `db:"course_id"  json:"course_id"`
	TagID      string    `db:"tag_id"     json:"tag_id"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}
