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

// PurchaseStore defines persistence operations for one-time purchases.
type PurchaseStore interface {
	// FindByID returns a purchase by ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Purchase, error)

	// ListByUser returns all purchases for a user, ordered by most recent first.
	ListByUser(ctx context.Context, userID string) ([]model.Purchase, error)

	// Create inserts a new purchase.
	Create(ctx context.Context, p *model.Purchase) error

	// Refund marks a purchase as refunded.
	Refund(ctx context.Context, id string) error
}

type purchaseStore struct {
	db database.Querier
}

// NewPurchaseStore creates a PurchaseStore backed by the given database connection.
func NewPurchaseStore(db database.Querier) PurchaseStore {
	return &purchaseStore{db: db}
}

func (s *purchaseStore) FindByID(ctx context.Context, id string) (*model.Purchase, error) {
	var p model.Purchase
	err := s.db.GetContext(ctx, &p, `
		SELECT id, user_id, item_id, status, amount_cents, currency, refunded_at, created_at
		FROM purchases
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *purchaseStore) ListByUser(ctx context.Context, userID string) ([]model.Purchase, error) {
	var purchases []model.Purchase
	err := s.db.SelectContext(ctx, &purchases, `
		SELECT id, user_id, type, item_id, status, amount_cents, currency, refunded_at, created_at
		FROM purchases
		WHERE user_id = $1
		ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	return purchases, nil
}

func (s *purchaseStore) Create(ctx context.Context, p *model.Purchase) error {
	return s.db.GetContext(ctx, p, `
		INSERT INTO purchases (
			user_id, type, item_id, status, amount_cents,
			currency, refunded_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, type, item_id, status, amount_cents, currency, refunded_at, created_at`,
		p.UserID,
		p.Type,
		p.ItemID,
		p.Status,
		p.AmountCents,
		p.Currency,
	)
}

func (s *purchaseStore) Refund(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE purchases
		SET status = 'refunded', refunded_at = now()
		WHERE id = $1`,
		id,
	)
	return err
}
