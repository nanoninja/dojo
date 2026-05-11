// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Category represents a course category, optionally nested under a parent.
type Category struct {
	ID          string     `db:"id"`
	ParentID    *string    `db:"parent_id"`
	Slug        string     `db:"slug"`
	Name        string     `db:"name"`
	Description *string    `db:"description"`
	ColorHex    *string    `db:"color_hex"`
	IconURL     *string    `db:"icon_url"`
	SortOrder   int16      `db:"sort_order"`
	IsVisible   bool       `db:"is_visible"`
	CourseCount int        `db:"course_count"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

// CategoryAssignment represents the many-to-many relation between a course and a category.
type CategoryAssignment struct {
	CourseID   string    `db:"course_id"`
	CategoryID string    `db:"category_id"`
	IsPrimary  bool      `db:"is_primary"`
	AssignedAt time.Time `db:"assigned_at"`
}
