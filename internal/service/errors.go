// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import "errors"

var (
	// ErrUserNotFound is returned when a user lookup yields no result.
	ErrUserNotFound = errors.New("user not found")

	// ErrWrongPassword is returned when the provided password does not match.
	ErrWrongPassword = errors.New("wrong password")

	// ErrEmailTaken is returned when registering with an already-used email.
	ErrEmailTaken = errors.New("email already taken")

	// ErrInvalidCredentials is returned when credentials cannot be validated.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAccountNotVerified is returned when the account email has not been confirmed.
	ErrAccountNotVerified = errors.New("account not verified")

	// ErrAccountNotActive is returned when the account is not in active state.
	ErrAccountNotActive = errors.New("account not active")

	// ErrAccountSuspended is returned when the account has been suspended.
	ErrAccountSuspended = errors.New("account suspended")

	// ErrTokenInvalid is returned when a verification token is missing or expired.
	ErrTokenInvalid = errors.New("invalid or expired token")

	// ErrTokenMaxAttempts is returned when too many failed token attempts have occurred.
	ErrTokenMaxAttempts = errors.New("too many failed attempts")

	// ErrInvalidRefreshToken is returned when the refresh token is missing or expired.
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

	// ErrAccountLocked is returned when the account is temporarily locked after too many failures.
	ErrAccountLocked = errors.New("account temporarily locked")

	// ErrCourseNotFound is returned when a course lookup yields no result.
	ErrCourseNotFound = errors.New("course not found")

	// ErrCourseSlugTaken is returned when creating or updating a course with an already-used slug.
	ErrCourseSlugTaken = errors.New("course slug already taken")

	// ErrCategoryNotFound is returned when a category lookup yields no result.
	ErrCategoryNotFound = errors.New("category not found")

	// ErrCategorySlugTaken is returned when creating or updating a category with an already-used slug.
	ErrCategorySlugTaken = errors.New("category slug already taken")

	// ErrTagNotFound is returned when a tag lookup yields no result.
	ErrTagNotFound = errors.New("tag not found")

	// ErrTagSlugTaken is returned when creating or updating a tag with an already-used slug.
	ErrTagSlugTaken = errors.New("tag slug already taken")

	// ErrChapterNotFound is returned when a chapter lookup yields no result.
	ErrChapterNotFound = errors.New("course chapter not found")

	// ErrLessonNotFound is returned when a lesson lookup yields no result.
	ErrLessonNotFound = errors.New("lesson not found")

	// ErrLessonResourceNotFound is returned when a lesson resource lookup yields no result.
	ErrLessonResourceNotFound = errors.New("lesson resource not found")
)
