// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// CourseLevel represents the difficulty level of a course.
type CourseLevel string

// Supported course levels.
const (
	CourseLevelBeginner     CourseLevel = "beginner"
	CourseLevelIntermediate CourseLevel = "intermediate"
	CourseLevelAdvanced     CourseLevel = "advanced"
	CourseLevelExpert       CourseLevel = "expert"
)

// ContentType represents the media format of a course or lesson.
type ContentType string

// Supported content types.
const (
	ContentTypeVideo    ContentType = "video"
	ContentTypeArticle  ContentType = "article"
	ContentTypeAudio    ContentType = "audio"
	ContentTypeLive     ContentType = "live"
	ContentTypeDocument ContentType = "document"
	ContentTypeMixed    ContentType = "mixed"
)

// Course represents a course in the catalog.
type Course struct {
	ID                 string      `db:"id"                  json:"id"`
	InstructorID       string      `db:"instructor_id"       json:"instructor_id"`
	Slug               string      `db:"slug"                json:"slug"`
	Title              string      `db:"title"               json:"title"`
	Subtitle           *string     `db:"subtitle"            json:"subtitle"`
	Description        *string     `db:"description"         json:"description"`
	Prerequisites      *string     `db:"prerequisites"       json:"prerequisites"`
	Objectives         *string     `db:"objectives"          json:"objectives"`
	MetaTitle          *string     `db:"meta_title"          json:"meta_title"`
	MetaDescription    *string     `db:"meta_description"    json:"meta_description"`
	MetaKeywords       *string     `db:"meta_keywords"       json:"meta_keywords"`
	ThumbnailURL       *string     `db:"thumbnail_url"       json:"thumbnail_url"`
	TrailerURL         *string     `db:"trailer_url"         json:"trailer_url"`
	Level              CourseLevel `db:"level"               json:"level"`
	ContentType        ContentType `db:"content_type"        json:"content_type"`
	Language           string      `db:"language"            json:"language"`
	DurationMinutes    int         `db:"duration_minutes"    json:"duration_minutes"`
	IsFree             bool        `db:"is_free"             json:"is_free"`
	SubscriptionOnly   bool        `db:"subscription_only"   json:"subscription_only"`
	PriceCents         int         `db:"price_cents"         json:"price_cents"`
	Currency           string      `db:"currency"            json:"currency"`
	IsPublished        bool        `db:"is_published"        json:"is_published"`
	IsFeatured         bool        `db:"is_featured"         json:"is_featured"`
	CertificateEnabled bool        `db:"certificate_enabled" json:"certificate_enabled"`
	SortOrder          int16       `db:"sort_order"          json:"sort_order"`
	StudentCount       int         `db:"student_count"       json:"student_count"`
	RatingAverage      float64     `db:"rating_average"      json:"rating_average"`
	RatingCount        int         `db:"rating_count"        json:"rating_count"`
	PublishedAt        *time.Time  `db:"published_at"        json:"published_at"`
	CreatedAt          time.Time   `db:"created_at"          json:"created_at"`
	UpdatedAt          *time.Time  `db:"updated_at"          json:"updated_at"`
	DeletedAt          *time.Time  `db:"deleted_at"          json:"deleted_at,omitempty"`
}
