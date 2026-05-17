// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import "time"

// LessonProgress tracks a user's viewing progress on a specific lesson.
type LessonProgress struct {
	UserID         string    `db:"user_id" json:"user_id"`
	LessonID       string    `db:"lesson_id" json:"lesson_id"`
	IsCompleted    bool      `db:"is_completed" json:"is_completed"`
	WatchedSeconds int       `db:"watched_seconds" json:"watched_seconds"`
	LastWatchedAt  time.Time `db:"last_watched_at" json:"last_watched_at"`
}
