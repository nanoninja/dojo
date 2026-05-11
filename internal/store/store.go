// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

// SortDir represents the sort direction for list queries.
type SortDir string

// Supported sort directions.
const (
	SortDirAsc  SortDir = "ASC"
	SortDirDesc SortDir = "DESC"
)
