// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/platform/security"
)

// ConsentStore defines persistence operations for user consent records.
type ConsentStore interface {
	// ListByUser returns all consent records for the given user, ordered by most recent first.
	ListByUser(ctx context.Context, userID string) ([]model.Consent, error)

	// FindByID returns a consent record by ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Consent, error)

	// Create inserts a new consent record.
	Create(ctx context.Context, c *model.Consent) error
}

type consentStore struct {
	db     database.Querier
	cipher *security.Cipher
}

// NewConsentStore creates a ConsentStore backed by the given database connection.
func NewConsentStore(db database.Querier, cipher *security.Cipher) ConsentStore {
	return &consentStore{db: db, cipher: cipher}
}

func (s *consentStore) ListByUser(ctx context.Context, userID string) ([]model.Consent, error) {
	var consents []model.Consent
	err := s.db.SelectContext(ctx, &consents, `
		SELECT
			id, user_id, type, version, is_accepted,
			ip_address, user_agent, source, created_at
		FROM user_consents
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	if err := s.decryptConsents(consents); err != nil {
		return nil, err
	}
	return consents, err
}

func (s *consentStore) FindByID(ctx context.Context, id string) (*model.Consent, error) {
	var c model.Consent
	err := s.db.GetContext(ctx, &c, `
		SELECT
			id, user_id, type, version, is_accepted,
			ip_address, user_agent, source, created_at
		FROM user_consents
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ip, err := decrypt(s.cipher, c.IPAddress)
	if err != nil {
		return nil, fmt.Errorf("decrypting ip_address: %w", err)
	}
	c.IPAddress = ip
	return &c, nil
}

func (s *consentStore) Create(ctx context.Context, c *model.Consent) error {
	ip, err := encrypt(s.cipher, c.IPAddress)
	if err != nil {
		return err
	}
	return s.db.GetContext(ctx, c, `
		INSERT INTO user_consents (
			user_id, type, version, is_accepted, ip_address, user_agent, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, type, version, is_accepted, user_agent, source, created_at`,
		c.UserID,
		c.Type,
		c.Version,
		c.IsAccepted,
		ip,
		c.UserAgent,
		c.Source,
	)
}

func (s *consentStore) decryptConsents(consents []model.Consent) error {
	for i := range consents {
		ip, err := decrypt(s.cipher, consents[i].IPAddress)
		if err != nil {
			return fmt.Errorf("decrypting ip_address: %w", err)
		}
		consents[i].IPAddress = ip
	}
	return nil
}
