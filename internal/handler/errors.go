// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"errors"

	"github.com/nanoninja/dojo/internal/fault"
	"github.com/nanoninja/dojo/internal/service"
)

// toFault maps service-level errors to the appropriate HTTP fault.
// It is shared across all handlers to avoid duplicating error mapping logic.
func toFault(err error) error {
	switch {

	case errors.Is(err, service.ErrEmailTaken):
		return fault.Conflict("email already in use", err)
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrWrongPassword),
		errors.Is(err, service.ErrInvalidRefreshToken),
		errors.Is(err, service.ErrTokenInvalid):
		return fault.Unauthorized(err)
	case errors.Is(err, service.ErrAccountNotVerified),
		errors.Is(err, service.ErrAccountNotActive),
		errors.Is(err, service.ErrAccountSuspended),
		errors.Is(err, service.ErrTokenMaxAttempts):
		return fault.Forbidden(err)
	case errors.Is(err, service.ErrAccountLocked):
		return fault.TooManyRequests(err)
	case errors.Is(err, service.ErrUserNotFound):
		return fault.NotFound("user", err)

	case errors.Is(err, service.ErrCourseNotFound):
		return fault.NotFound("course", err)
	case errors.Is(err, service.ErrCategoryNotFound):
		return fault.NotFound("category", err)
	case errors.Is(err, service.ErrTagNotFound):
		return fault.NotFound("tag", err)
	case errors.Is(err, service.ErrChapterNotFound):
		return fault.NotFound("chapter", err)
	case errors.Is(err, service.ErrLessonNotFound):
		return fault.NotFound("lesson", err)
	case errors.Is(err, service.ErrLessonResourceNotFound):
		return fault.NotFound("lesson resource", err)

	case errors.Is(err, service.ErrEnrollmentNotFound):
		return fault.NotFound("enrollment", err)
	case errors.Is(err, service.ErrAlreadyEnrolled):
		return fault.Conflict("user already enrolled in this course", err)

	case errors.Is(err, service.ErrBundleNotFound):
		return fault.NotFound("bundle", err)
	case errors.Is(err, service.ErrBundleSlugTaken):
		return fault.Conflict("bundle slug already taken", err)

	default:
		return fault.Internal(err)
	}
}
