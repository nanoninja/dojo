// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// CertificateService handles certificate retrieval operations.
type CertificateService interface {
	// GetByID returns a certificate by ID, or ErrCertificateNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Certificate, error)

	// GetByUUID returns a certificate by its public UUID, or ErrCertificateNotFound if not found.
	GetByUUID(ctx context.Context, uuid string) (*model.Certificate, error)

	// ListByUser returns all certificates earned by the given user.
	ListByUser(ctx context.Context, userID string) ([]model.Certificate, error)
}

type certificateService struct {
	certificates store.CertificateStore
}

// NewCertificateService creates a CertificateService backed by the given store.
func NewCertificateService(certificates store.CertificateStore) CertificateService {
	return &certificateService{certificates: certificates}
}

func (s *certificateService) GetByID(ctx context.Context, id string) (*model.Certificate, error) {
	c, err := s.certificates.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCertificateNotFound
	}
	return c, nil
}

func (s *certificateService) GetByUUID(ctx context.Context, uuid string) (*model.Certificate, error) {
	c, err := s.certificates.FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCertificateNotFound
	}
	return c, nil
}

func (s *certificateService) ListByUser(ctx context.Context, userID string) ([]model.Certificate, error) {
	return s.certificates.ListByUser(ctx, userID)
}
