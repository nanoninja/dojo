// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// ConsentService defines business operations for GDPR consent records.
type ConsentService interface {
	// ListByUser returns all consent records for the given user.
	ListByUser(ctx context.Context, userID string) ([]model.Consent, error)

	// GetByID returns a consent record by ID, or ErrConsentNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Consent, error)

	// Create records a new consent action for a user.
	Create(ctx context.Context, c *model.Consent) error
}

type consentService struct {
	consents store.ConsentStore
}

// NewConsentService creates a ConsentService backed by the given store.
func NewConsentService(consents store.ConsentStore) ConsentService {
	return &consentService{consents: consents}
}

func (s *consentService) ListByUser(ctx context.Context, userID string) ([]model.Consent, error) {
	return s.consents.ListByUser(ctx, userID)
}

func (s *consentService) GetByID(ctx context.Context, id string) (*model.Consent, error) {
	c, err := s.consents.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrConsentNotFound
	}
	return c, nil
}

func (s *consentService) Create(ctx context.Context, c *model.Consent) error {
	return s.consents.Create(ctx, c)
}
