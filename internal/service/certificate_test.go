// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
)

// ============================================================================
// fakeCertificateStore
// ============================================================================

type fakeCertificateStore struct {
	certs map[string]*model.Certificate
	uuids map[string]*model.Certificate
	seq   int
}

func newFakeCertificateStore() *fakeCertificateStore {
	return &fakeCertificateStore{
		certs: make(map[string]*model.Certificate),
		uuids: make(map[string]*model.Certificate),
	}
}

func (s *fakeCertificateStore) nextID() string {
	s.seq++
	return fmt.Sprintf("cert-%d", s.seq)
}

func (s *fakeCertificateStore) FindByID(_ context.Context, id string) (*model.Certificate, error) {
	c, ok := s.certs[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (s *fakeCertificateStore) FindByUUID(_ context.Context, uuid string) (*model.Certificate, error) {
	c, ok := s.uuids[uuid]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (s *fakeCertificateStore) ListByUser(_ context.Context, userID string) ([]model.Certificate, error) {
	var result []model.Certificate
	for _, c := range s.certs {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (s *fakeCertificateStore) Create(_ context.Context, c *model.Certificate) error {
	c.ID = s.nextID()
	c.UUID = fmt.Sprintf("uuid-%d", s.seq)
	c.IssuedAt = time.Now()
	cp := *c
	s.certs[c.ID] = &cp
	s.uuids[c.UUID] = &cp
	return nil
}

// ============================================================================
// helpers
// ============================================================================

func newCertificateService(cs *fakeCertificateStore) service.CertificateService {
	return service.NewCertificateService(cs)
}

func baseCertificate() *model.Certificate {
	return &model.Certificate{
		UserID:   "user-1",
		CourseID: "course-1",
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestCertificateService_GetByID(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCertificateStore()
	svc := newCertificateService(cs)

	c := baseCertificate()
	assert.NoError(t, cs.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, c.ID)
		assert.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, "unknown-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrCertificateNotFound)
		assert.Equal(t, (*model.Certificate)(nil), got)
	})
}

func TestCertificateService_GetByUUID(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCertificateStore()
	svc := newCertificateService(cs)

	c := baseCertificate()
	assert.NoError(t, cs.Create(ctx, c))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByUUID(ctx, c.UUID)
		assert.NoError(t, err)
		assert.Equal(t, c.UUID, got.UUID)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := svc.GetByUUID(ctx, "unknown-uuid")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrCertificateNotFound)
		assert.Equal(t, (*model.Certificate)(nil), got)
	})
}

func TestCertificateService_ListByUser(t *testing.T) {
	ctx := context.Background()
	cs := newFakeCertificateStore()
	svc := newCertificateService(cs)

	c := baseCertificate()
	assert.NoError(t, cs.Create(ctx, c))

	t.Run("returns certificates for user", func(t *testing.T) {
		got, err := svc.ListByUser(ctx, "user-1")
		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, c.ID, got[0].ID)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		got, err := svc.ListByUser(ctx, "unknown-user")
		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})
}
