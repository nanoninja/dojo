// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
)

// TokenFilter holds the criteria used to look up a verification token.
type TokenFilter struct {
	UserID string
	Token  string
	Type   model.TokenType
}

// AuthStore defines the data access contract for verification tokens.
type AuthStore interface {
	// Create inserts a new verification token and populates t.ID.
	Create(ctx context.Context, t *model.VerificationToken) error

	// FindOne returns a token matching the filter, or nil if not found.
	FindOne(ctx context.Context, f TokenFilter) (*model.VerificationToken, error)

	// FindActiveByUserAndType returns the latest active token for a user and type, or nil if not found.
	FindActiveByUserAndType(ctx context.Context, userID string, tokenType model.TokenType) (*model.VerificationToken, error)

	// MarkUsed marks a token as used at the current time.
	MarkUsed(ctx context.Context, id string) error

	// IncrementAttempts increments the failed attempt counter for a token.
	IncrementAttempts(ctx context.Context, id string) error

	// DeleteExpired removes all expired tokens for a given user.
	DeleteExpired(ctx context.Context, userID string) error
}

type authStore struct {
	db database.Querier
}

// NewAuthStore creates a new AuthStore.
func NewAuthStore(db database.Querier) AuthStore {
	return &authStore{db: db}
}

func (s *authStore) Create(ctx context.Context, t *model.VerificationToken) error {
	return s.db.QueryRowContext(ctx, `
		INSERT INTO verification_tokens (
			user_id,
			token,
			type,
			expires_at
		) VALUES (
			$1, $2, $3, $4 
		) RETURNING id`,
		t.UserID,
		t.Token,
		t.Type,
		t.ExpiresAt,
	).Scan(&t.ID)
}

func (s *authStore) FindOne(ctx context.Context, f TokenFilter) (*model.VerificationToken, error) {
	var t model.VerificationToken
	err := s.db.GetContext(ctx, &t, `
		SELECT
			id,
			user_id,
			token,
			type,
			attempts,
			expires_at,
			used_at,
			created_at
		FROM verification_tokens
		WHERE user_id = $1
		  AND token = $2
		  AND type = $3
		  AND used_at IS NULL
		  AND expires_at > NOW()`,
		f.UserID,
		f.Token,
		f.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (s *authStore) FindActiveByUserAndType(ctx context.Context, userID string, tokenType model.TokenType) (*model.VerificationToken, error) {
	var t model.VerificationToken
	err := s.db.GetContext(ctx, &t, `
		SELECT
			id,
			user_id,
			token,
			type,
			attempts,
			expires_at,
			used_at,
			created_at
		FROM verification_tokens
		WHERE user_id = $1
		  AND type = $2
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1`,
		userID,
		tokenType,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (s *authStore) MarkUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE verification_tokens
		   SET used_at = NOW()
		WHERE id = $1`, id)
	return err
}

func (s *authStore) IncrementAttempts(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE verification_tokens
		   SET attempts = attempts + 1
		WHERE id = $1`, id)
	return err
}

func (s *authStore) DeleteExpired(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM verification_tokens
		WHERE user_id = $1
		  AND (expires_at < NOW() OR used_at IS NOT NULL)`, userID)
	return err
}
