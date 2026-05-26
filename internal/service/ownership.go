// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/platform/database"
)

// OwnershipChecker verifies that a resource belongs to a given owner.
// Returns fault.NotFound if the resource does not exist or is not owned by ownerID.
type OwnershipChecker interface {
	Check(ctx context.Context, resourceID, ownerID string) error
}

type ownershipChecker struct {
	db    database.Querier
	query string
}

func (o *ownershipChecker) Check(ctx context.Context, resourceID, ownerID string) error {
	var exists bool
	if err := o.db.GetContext(ctx, &exists, o.query, resourceID, ownerID); err != nil {
		return fault.Internal(err)
	}
	if !exists {
		return fault.NotFound("resource", nil)
	}
	return nil
}

// NewCourseOwnership checks that a course belongs to the given instructor.
func NewCourseOwnership(db database.Querier) OwnershipChecker {
	return &ownershipChecker{db: db, query: `
		SELECT EXISTS(
			SELECT 1 FROM courses
			WHERE id = $1 AND instructor_id = $2 AND deleted_at IS NULL)`,
	}
}

// NewChapterOwnership checks that a chapter's parent course belongs to the given instructor.
func NewChapterOwnership(db database.Querier) OwnershipChecker {
	return &ownershipChecker{db: db, query: `
		SELECT EXISTS(
			SELECT 1 FROM chapters ch
			JOIN courses co ON co.id = ch.course_id
			WHERE ch.id = $1 AND co.instructor_id = $2 AND co.deleted_at IS NULL)`,
	}
}

// NewLessonOwnership checks that a lesson's parent course belongs to the given instructor.
func NewLessonOwnership(db database.Querier) OwnershipChecker {
	return &ownershipChecker{db: db, query: `
		SELECT EXISTS(
			SELECT 1 FROM lessons l
			JOIN chapters ch ON ch.id = l.chapter_id
			JOIN courses co ON co.id = ch.course_id
			WHERE l.id = $1 AND co.instructor_id = $2 AND co.deleted_at IS NULL)`,
	}
}

// NewBundleOwnership checks that a bundle belongs to the given instructor.
func NewBundleOwnership(db database.Querier) OwnershipChecker {
	return &ownershipChecker{db: db, query: `
		SELECT EXISTS(
			SELECT 1 FROM bundles
			WHERE id = $1 AND instructor_id = $2 AND deleted_at IS NULL)`,
	}
}
