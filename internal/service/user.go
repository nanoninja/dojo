// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"fmt"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user account management: registration, profile updates, and deletion.
type UserService interface {
	// List returns a paginated list of users matching the given filter, and the total count.
	List(ctx context.Context, f store.UserFilter) ([]model.User, int, error)

	// GetByID returns a user by ID, or ErrUserNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.User, error)

	// Register creates a new user account with a hashed password.
	Register(ctx context.Context, u *model.User, password string) error

	// UpdateProfile updates the profile fields of an existing user.
	UpdateProfile(ctx context.Context, u *model.User) error

	// ChangePassword verifies the current password and replaces it with a new one.
	ChangePassword(ctx context.Context, id, currentPassword, newPassword string) error

	// Delete soft-deletes a user account.
	Delete(ctx context.Context, id string) error

	// LoginHistory returns the most recent login attempts for a user.
	LoginHistory(ctx context.Context, userID string, limit int) ([]model.LoginAuditLog, error)
}

type userService struct {
	store store.UserStore
	audit store.LoginAuditStore
}

// NewUserService creates a UserService backed by the given store.
func NewUserService(s store.UserStore, audit store.LoginAuditStore) UserService {
	return &userService{store: s, audit: audit}
}

func (s *userService) List(ctx context.Context, f store.UserFilter) ([]model.User, int, error) {
	return s.store.List(ctx, f)
}

func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
	u, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *userService) Register(ctx context.Context, u *model.User, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	u.PasswordHash = string(hash)
	u.Status = model.UserStatusPending
	return s.store.Create(ctx, u)
}

func (s *userService) UpdateProfile(ctx context.Context, u *model.User) error {
	return s.store.Update(ctx, u)
}

func (s *userService) ChangePassword(ctx context.Context, id, currentPassword, newPassword string) error {
	// Fetch user to retrieve email
	u, err := s.store.FindCredentialsByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrWrongPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	return s.store.UpdatePassword(ctx, id, string(hash))
}

func (s *userService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *userService) LoginHistory(ctx context.Context, userID string, limit int) ([]model.LoginAuditLog, error) {
	return s.audit.FindByUser(ctx, userID, limit)
}
