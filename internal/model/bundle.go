// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Bundle represents a bundle of courses.
type Bundle struct {
	ID           string     `db:"id"            json:"id"`
	InstructorID string     `db:"instructor_id" json:"instructor_id"`
	Slug         string     `db:"slug"          json:"slug"`
	Title        string     `db:"title"         json:"title"`
	Subtitle     *string    `db:"subtitle"      json:"subtitle"`
	Description  *string    `db:"description"   json:"description"`
	ThumbnailURL *string    `db:"thumbnail_url" json:"thumbnail_url"`
	IsFree       bool       `db:"is_free"       json:"is_free"`
	PriceCents   int        `db:"price_cents"   json:"price_cents"`
	Currency     string     `db:"currency"      json:"currency"`
	IsPublished  bool       `db:"is_published"  json:"is_published"`
	SortOrder    int        `db:"sort_order"    json:"sort_order"`
	StudentCount int        `db:"student_count" json:"student_count"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"    json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"    json:"deleted_at"`
}

// BundleCourseAssignment represents the many-to-many relation between a bundle and a course.
type BundleCourseAssignment struct {
	BundleID   string    `db:"bundle_id"   json:"bundle_id"`
	CourseID   string    `db:"course_id"   json:"course_id"`
	SortOrder  int       `db:"sort_order"  json:"sort_order"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}
