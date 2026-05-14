// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Lesson represents a lesson within a chapter.
type Lesson struct {
	ID              string      `db:"id"               json:"id"`
	ChapterID       string      `db:"chapter_id"       json:"chapter_id"`
	Title           string      `db:"title"            json:"title"`
	Slug            string      `db:"slug"             json:"slug"`
	Description     *string     `db:"description"      json:"description"`
	SortOrder       int16       `db:"sort_order"       json:"sort_order"`
	ContentType     ContentType `db:"content_type"     json:"content_type"`
	MediaURL        *string     `db:"media_url"        json:"media_url"`
	DurationMinutes int         `db:"duration_minutes" json:"duration_minutes"`
	IsFree          bool        `db:"is_free"          json:"is_free"`
	IsPublished     bool        `db:"is_published"     json:"is_published"`
	CreatedAt       time.Time   `db:"created_at"       json:"created_at"`
	UpdatedAt       *time.Time  `db:"updated_at"       json:"updated_at"`
}

// LessonResource represents a downloadable resource attached to a lesson.
type LessonResource struct {
	ID            string    `db:"id"              json:"id"`
	LessonID      string    `db:"lesson_id"       json:"lesson_id"`
	Title         string    `db:"title"           json:"title"`
	Description   *string   `db:"description"     json:"description"`
	FileURL       string    `db:"file_url"        json:"file_url"`
	FileName      string    `db:"file_name"       json:"file_name"`
	FileSizeBytes *int64    `db:"file_size_bytes" json:"file_size_bytes"`
	MimeType      *string   `db:"mime_type"       json:"mime_type"`
	SortOrder     int16     `db:"sort_order"      json:"sort_order"`
	IsPublic      bool      `db:"is_public"       json:"is_public"`
	DownloadCount int       `db:"download_count"  json:"download_count"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
}
