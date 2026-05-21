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

// SubscriptionStore defines persistence operations for user subscriptions.
type SubscriptionStore interface {
	// FindActiveByUser returns the current active subscription for a user, or nil if none.
	FindActiveByUser(ctx context.Context, userID string) (*model.Subscription, error)

	// ListByUser returns all subscriptions for a user, ordered by most recent first.
	ListByUser(ctx context.Context, userID string) ([]model.Subscription, error)

	// Create inserts a new subscription.
	Create(ctx context.Context, s *model.Subscription) error

	// Cancel marks a subscription as cancelled.
	Cancel(ctx context.Context, id string) error
}

type subscriptionStore struct {
	db database.Querier
}

// NewSubscriptionStore creates a SubscriptionStore backed by the given database connection.
func NewSubscriptionStore(db database.Querier) SubscriptionStore {
	return &subscriptionStore{db: db}
}

func (s *subscriptionStore) FindActiveByUser(ctx context.Context, userID string) (*model.Subscription, error) {
	var sub model.Subscription
	err := s.db.GetContext(ctx, &sub, `
		SELECT id, user_id, plan, status, started_at, expires_at, cancelled_at
		FROM subscriptions
		WHERE user_id = $1
			AND status = 'active'
			AND expires > 'now()'
		ORDER BY expires_at DESC
		LIMIT 1`,
		userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, err
}

func (s *subscriptionStore) ListByUser(ctx context.Context, userID string) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := s.db.SelectContext(ctx, &subs, `
		SELECT id, user_id, plan, status, started_at, expires_at, cancelled_at
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY started_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *subscriptionStore) Create(ctx context.Context, sub *model.Subscription) error {
	return s.db.GetContext(ctx, sub, `
		INSERT INTO subscriptions (user_id, plan, status, started_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, plan, status, started_at, expires_at, cancelled_at`,
		sub.UserID,
		sub.Plan,
		sub.Status,
		sub.StartedAt,
		sub.ExpiresAt,
	)
}

func (s *subscriptionStore) Cancel(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET
			status       = 'cancelled',
			cancelled_at = now()
		WHERE id = $1`,
		id,
	)
	return err
}
