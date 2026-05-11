// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// Lesson represents a lesson within a chapter.
type Lesson struct {
	ID              string      `db:"id"`
	ChapterID       string      `db:"chapter_id"`
	Title           string      `db:"title"`
	Slug            string      `db:"slug"`
	Description     *string     `db:"description"`
	SortOrder       int16       `db:"sort_order"`
	ContentType     ContentType `db:"content_type"`
	MediaURL        *string     `db:"media_url"`
	DurationMinutes int         `db:"duration_minutes"`
	IsFree          bool        `db:"is_free"`
	IsPublished     bool        `db:"is_published"`
	CreatedAt       time.Time   `db:"created_at"`
	UpdatedAt       *time.Time  `db:"updated_at"`
}

// LessonResource represents a downloadable resource attached to a lesson.
type LessonResource struct {
	ID            string    `db:"id"`
	LessonID      string    `db:"lesson_id"`
	Title         string    `db:"title"`
	Description   *string   `db:"description"`
	FileURL       string    `db:"file_url"`
	FileName      string    `db:"file_name"`
	FileSizeBytes *int64    `db:"file_size_bytes"`
	MimeType      *string   `db:"mime_type"`
	SortOrder     int16     `db:"sort_order"`
	IsPublic      bool      `db:"is_public"`
	DownloadCount int       `db:"download_count"`
	CreatedAt     time.Time `db:"created_at"`
}
