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
	ID                 string      `db:"id"`
	InstructorID       string      `db:"instructor_id"`
	Slug               string      `db:"slug"`
	Title              string      `db:"title"`
	Subtitle           *string     `db:"subtitle"`
	Description        *string     `db:"description"`
	Prerequisites      *string     `db:"prerequisites"`
	Objectives         *string     `db:"objectives"`
	MetaTitle          *string     `db:"meta_title"`
	MetaDescription    *string     `db:"meta_description"`
	MetaKeywords       *string     `db:"meta_keywords"`
	ThumbnailURL       *string     `db:"thumbnail_url"`
	TrailerURL         *string     `db:"trailer_url"`
	Level              CourseLevel `db:"level"`
	ContentType        ContentType `db:"content_type"`
	Language           string      `db:"language"`
	DurationMinutes    int         `db:"duration_minutes"`
	IsFree             bool        `db:"is_free"`
	SubscriptionOnly   bool        `db:"subscription_only"`
	PriceCents         int         `db:"price_cents"`
	Currency           string      `db:"currency"`
	IsPublished        bool        `db:"is_published"`
	IsFeatured         bool        `db:"is_featured"`
	CertificateEnabled bool        `db:"certificate_enabled"`
	SortOrder          int16       `db:"sort_order"`
	StudentCount       int         `db:"student_count"`
	RatingAverage      float64     `db:"rating_average"`
	RatingCount        int         `db:"rating_count"`
	PublishedAt        *time.Time  `db:"published_at"`
	CreatedAt          time.Time   `db:"created_at"`
	UpdatedAt          *time.Time  `db:"updated_at"`
	DeletedAt          *time.Time  `db:"deleted_at"`
}
