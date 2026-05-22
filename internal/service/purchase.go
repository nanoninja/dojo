// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"fmt"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

// PurchaseService defines business operations for one-time purchases.
type PurchaseService interface {
	// GetByID returns a purchase by ID, or ErrPurchaseNotFound if not found.
	GetByID(ctx context.Context, id string) (*model.Purchase, error)

	// ListByUser returns all purchases for a user.
	ListByUser(ctx context.Context, userID string) ([]model.Purchase, error)

	// BuyCourse processes a course purchase and creates the enrollment atomically.
	BuyCourse(ctx context.Context, userID, courseID string, amountCents int64, currency string) (*model.Purchase, error)

	// BuyBundle processes a bundle purchase and creates enrollments for all courses atomically.
	BuyBundle(ctx context.Context, userID, bundleID string, amountCents int64, currency string) (*model.Purchase, error)

	// Refund marks a purchase as refunded and cancels the associated enrollment(s).
	Refund(ctx context.Context, purchaseID string) error
}

type purchaseService struct {
	db          database.TxRunner
	purchases   store.PurchaseStore
	enrollments store.EnrollmentStore
	bundles     store.BundleCourseStore
}

// NewPurchaseService creates a PurchaseService backed by the given stores.
func NewPurchaseService(
	db database.TxRunner,
	purchases store.PurchaseStore,
	enrollments store.EnrollmentStore,
	bundles store.BundleCourseStore,
) PurchaseService {
	return &purchaseService{
		db:          db,
		purchases:   purchases,
		enrollments: enrollments,
		bundles:     bundles,
	}
}

func (s *purchaseService) GetByID(ctx context.Context, id string) (*model.Purchase, error) {
	p, err := s.purchases.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPurchaseNotFound
	}
	return p, nil
}

func (s *purchaseService) ListByUser(ctx context.Context, userID string) ([]model.Purchase, error) {
	return s.purchases.ListByUser(ctx, userID)
}

func (s *purchaseService) BuyCourse(ctx context.Context, userID, courseID string, amountCents int64, currency string) (*model.Purchase, error) {
	var p model.Purchase

	err := s.db.WithTx(ctx, func(q database.Querier) error {
		ps := store.NewPurchaseStore(q)
		es := store.NewEnrollmentStore(q)

		purchase := &model.Purchase{
			UserID:      userID,
			Type:        model.PurchaseTypeCourse,
			ItemID:      courseID,
			Status:      model.PurchaseStatusCompleted,
			AmountCents: amountCents,
			Currency:    currency,
		}
		if err := ps.Create(ctx, purchase); err != nil {
			return fmt.Errorf("create purchase: %w", err)
		}

		enrollment := &model.CourseEnrollment{
			UserID:     userID,
			CourseID:   courseID,
			PurchaseID: &purchase.ID,
			Status:     model.EnrollmentStatusActive,
			Source:     model.EnrollmentSourcePurchase,
		}
		if err := es.Create(ctx, enrollment); err != nil {
			return fmt.Errorf("create enrollment: %w", err)
		}

		p = *purchase
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (s *purchaseService) BuyBundle(ctx context.Context, userID, bundleID string, amountCents int64, currency string) (*model.Purchase, error) {
	var p model.Purchase

	err := s.db.WithTx(ctx, func(q database.Querier) error {
		ps := store.NewPurchaseStore(q)
		es := store.NewEnrollmentStore(q)
		bcs := store.NewBundleCourseStore(q)

		purchase := &model.Purchase{
			UserID:      userID,
			Type:        model.PurchaseTypeBundle,
			ItemID:      bundleID,
			Status:      model.PurchaseStatusCompleted,
			AmountCents: amountCents,
			Currency:    currency,
		}
		if err := ps.Create(ctx, purchase); err != nil {
			return fmt.Errorf("create purchase: %w", err)
		}

		assignments, err := bcs.List(ctx, bundleID)
		if err != nil {
			return fmt.Errorf("list bundle courses: %w", err)
		}

		for _, a := range assignments {
			enrollment := &model.CourseEnrollment{
				UserID:     userID,
				CourseID:   a.CourseID,
				PurchaseID: &purchase.ID,
				Status:     model.EnrollmentStatusActive,
				Source:     model.EnrollmentSourcePurchase,
			}
			if err := es.Create(ctx, enrollment); err != nil {
				return fmt.Errorf("create enrollment for course %s: %w", a.CourseID, err)
			}
		}

		p = *purchase
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (s *purchaseService) Refund(ctx context.Context, purchaseID string) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		ps := store.NewPurchaseStore(q)
		es := store.NewEnrollmentStore(q)

		if err := ps.Refund(ctx, purchaseID); err != nil {
			return fmt.Errorf("refund purchase: %w", err)
		}
		if err := es.CancelByPurchase(ctx, purchaseID); err != nil {
			return fmt.Errorf("cancel enrollments: %w", err)
		}
		return nil
	})
}
