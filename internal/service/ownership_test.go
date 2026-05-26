// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/service"
)

// ownerQuerier simulates GetContext returning a controlled EXISTS result.
type ownerQuerier struct {
	exists bool
	err    error
}

func (q *ownerQuerier) GetContext(_ context.Context, dest any, _ string, _ ...any) error {
	if q.err != nil {
		return q.err
	}
	if b, ok := dest.(*bool); ok {
		*b = q.exists
	}
	return nil
}

func (q *ownerQuerier) SelectContext(_ context.Context, _ any, _ string, _ ...any) error {
	return nil
}
func (q *ownerQuerier) QueryxContext(_ context.Context, _ string, _ ...any) (*sqlx.Rows, error) {
	return nil, nil
}
func (q *ownerQuerier) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}
func (q *ownerQuerier) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, nil
}
func (q *ownerQuerier) Rebind(s string) string { return s }

func TestOwnershipChecker_Check(t *testing.T) {
	ctx := context.Background()

	constructors := []struct {
		name string
		fn   func(*ownerQuerier) service.OwnershipChecker
	}{
		{"Course", func(q *ownerQuerier) service.OwnershipChecker { return service.NewCourseOwnership(q) }},
		{"Chapter", func(q *ownerQuerier) service.OwnershipChecker { return service.NewChapterOwnership(q) }},
		{"Lesson", func(q *ownerQuerier) service.OwnershipChecker { return service.NewLessonOwnership(q) }},
		{"Bundle", func(q *ownerQuerier) service.OwnershipChecker { return service.NewBundleOwnership(q) }},
	}

	for _, c := range constructors {
		t.Run(c.name+"/owner", func(t *testing.T) {
			checker := c.fn(&ownerQuerier{exists: true})
			assert.NoError(t, checker.Check(ctx, "resource-id", "owner-id"))
		})

		t.Run(c.name+"/not owner", func(t *testing.T) {
			checker := c.fn(&ownerQuerier{exists: false})
			assert.Error(t, checker.Check(ctx, "resource-id", "other-id"))
		})

		t.Run(c.name+"/db error", func(t *testing.T) {
			checker := c.fn(&ownerQuerier{err: errors.New("db down")})
			assert.Error(t, checker.Check(ctx, "resource-id", "owner-id"))
		})
	}
}
