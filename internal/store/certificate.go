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

// CertificateStore defines persistence operations for certificates.
type CertificateStore interface {
	// FindByID returns the certificate with the given ID, or nil if not found.
	FindByID(ctx context.Context, id string) (*model.Certificate, error)

	// FindByUUID returns the certificate with the given public UUID, or nil if not found.
	FindByUUID(ctx context.Context, uuid string) (*model.Certificate, error)

	// ListByUser returns all certificates earned by the given user, ordered by most recent first.
	ListByUser(ctx context.Context, userID string) ([]model.Certificate, error)

	// Create inserts a new certificate. It is idempotent: if a certificate already exists
	// for the same user and course, the insert is silently ignored.
	Create(ctx context.Context, c *model.Certificate) error
}

type certificateStore struct {
	db database.Querier
}

// NewCertificateStore creates a new CertificateStore backed by the given database connection.
func NewCertificateStore(db database.Querier) CertificateStore {
	return &certificateStore{db: db}
}

func (s *certificateStore) FindByID(ctx context.Context, id string) (*model.Certificate, error) {
	var c model.Certificate
	err := s.db.GetContext(ctx, &c, `
		SELECT id, user_id, course_id, uuid, issued_at
		FROM certificates
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, err
}

func (s *certificateStore) FindByUUID(ctx context.Context, uuid string) (*model.Certificate, error) {
	var c model.Certificate
	err := s.db.GetContext(ctx, &c, `
		SELECT id, user_id, course_id, uuid, issued_at
		FROM certificates
		WHERE uuid = $1`,
		uuid,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, err
}

func (s *certificateStore) ListByUser(ctx context.Context, userID string) ([]model.Certificate, error) {
	var certificates []model.Certificate
	err := s.db.SelectContext(ctx, &certificates, `
		SELECT id, user_id, course_id, uuid, issued_at
		FROM certificates
		WHERE user_id = $1
		ORDER BY issued_at DESC`,
		userID,
	)
	return certificates, err
}

func (s *certificateStore) Create(ctx context.Context, c *model.Certificate) error {
	return s.db.GetContext(ctx, c, `
		INSERT INTO certificates (user_id, course_id) VALUES ($1, $2)
		ON CONFLICT (user_id, course_id) DO NOTHING
		RETURING id, user_id, course_id, uuid, issued_at`,
		c.UserID,
		c.CourseID,
	)
}
