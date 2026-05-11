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

// RefreshTokenStore defines the data access contract for refresh tokens.
type RefreshTokenStore interface {
	// Create stores a new hashed refresh token.
	Create(ctx context.Context, t *model.RefreshToken) error

	// FindByHash returns a token by its hash, or nil if not found or revoked.
	FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error)

	// Revoke marks a refresh token as revoked
	Revoke(ctx context.Context, id string) error

	// RevokeAllForUser revokes all active refresh tokens for a user (logout everywhere).
	RevokeAllForUser(ctx context.Context, userID string) error

	// RotateToken atomically revokes oldID and inserts newToken in a single transaction.
	// If either operation fails, both are rolled back — preventing the user from being locked out.
	RotateToken(ctx context.Context, oldID string, newToken *model.RefreshToken) error

	// DeleteExpired removes all expired tokens for a given user.
	DeleteExpired(ctx context.Context, userID string) error
}

type refreshTokenStore struct {
	db *database.DB
}

// NewRefreshTokenStore creates a new RefreshTokenStore.
func NewRefreshTokenStore(db *database.DB) RefreshTokenStore {
	return &refreshTokenStore{db: db}
}

func (s *refreshTokenStore) Create(ctx context.Context, t *model.RefreshToken) error {
	return s.db.QueryRowxContext(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id`,
		t.UserID,
		t.TokenHash,
		t.ExpiresAt,
	).Scan(&t.ID)
}

func (s *refreshTokenStore) FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := s.db.GetContext(ctx, &t, `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			created_at,
			revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()`,
		hash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (s *refreshTokenStore) RotateToken(ctx context.Context, oldID string, newToken *model.RefreshToken) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		_, err := q.ExecContext(ctx,
			`UPDATE refresh_tokens
				SET revoked_at = NOW()
				WHERE id = $1`,
			oldID,
		)
		if err != nil {
			return err
		}
		return q.QueryRowContext(ctx,
			`INSERT INTO refresh_tokens (
				user_id,
				token_hash,
				expires_at
			) VALUES ($1, $2, $3)
			RETURNING id`,
			newToken.UserID,
			newToken.TokenHash,
			newToken.ExpiresAt,
		).Scan(&newToken.ID)
	})
}

func (s *refreshTokenStore) Revoke(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE refresh_tokens
		   SET revoked_at = NOW()
		WHERE id = $1`, id)
	return err
}

func (s *refreshTokenStore) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
        UPDATE refresh_tokens
           SET revoked_at = NOW()
         WHERE user_id = $1
		   AND revoked_at IS NULL`, userID)
	return err
}

func (s *refreshTokenStore) DeleteExpired(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
        DELETE FROM refresh_tokens
        WHERE user_id = $1
		  AND (expires_at < NOW() OR revoked_at IS NOT NULL)`, userID)
	return err
}
