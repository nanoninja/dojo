// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Category represents a course category, optionally nested under a parent.
type Category struct {
	ID          string     `db:"id"           json:"id"`
	ParentID    *string    `db:"parent_id"    json:"parent_id"`
	Slug        string     `db:"slug"         json:"slug"`
	Name        string     `db:"name"         json:"name"`
	Description *string    `db:"description"  json:"description"`
	ColorHex    *string    `db:"color_hex"    json:"color_hex"`
	IconURL     *string    `db:"icon_url"     json:"icon_url"`
	SortOrder   int16      `db:"sort_order"   json:"sort_order"`
	IsVisible   bool       `db:"is_visible"   json:"is_visible"`
	CourseCount int        `db:"course_count" json:"course_count"`
	CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"   json:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"   json:"deleted_at,omitempty"`
}

// CategoryAssignment represents the many-to-many relation between a course and a category.
type CategoryAssignment struct {
	CourseID   string    `db:"course_id"   json:"course_id"`
	CategoryID string    `db:"category_id" json:"category_id"`
	IsPrimary  bool      `db:"is_primary"  json:"is_primary"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}
