// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"fmt"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/payment"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

const (
	checkoutItemCourse = "Course"
	checkoutItemBundle = "Bundle"
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

	// ConfirmPayment marks a pending purchase as completed, records the provider
	// payment ID, and creates course enrollments atomically.
	ConfirmPayment(ctx context.Context, purchaseID, providerPaymentID string) error

	// CancelPending marks a pending purchase as failed (e.g. payment expired or declined).
	CancelPending(ctx context.Context, purchaseID string) error
}

type purchaseService struct {
	db          database.TxRunner
	provider    payment.Provider
	purchases   store.PurchaseStore
	enrollments store.EnrollmentStore
	bundles     store.BundleCourseStore
}

// NewPurchaseService creates a PurchaseService backed by the given stores.
func NewPurchaseService(
	db database.TxRunner,
	provider payment.Provider,
	purchases store.PurchaseStore,
	enrollments store.EnrollmentStore,
	bundles store.BundleCourseStore,
) PurchaseService {
	return &purchaseService{
		db:          db,
		provider:    provider,
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

		purchase := &model.Purchase{
			UserID:      userID,
			Type:        model.PurchaseTypeCourse,
			ItemID:      courseID,
			Status:      model.PurchaseStatusPending,
			AmountCents: amountCents,
			Currency:    currency,
			Provider:    payment.ProviderStripe,
		}
		if err := ps.Create(ctx, purchase); err != nil {
			return fmt.Errorf("create purchase: %w", err)
		}
		p = *purchase
		return nil
	})
	if err != nil {
		return nil, err
	}

	session, err := s.provider.CreateCheckout(ctx, payment.Order{
		Currency: currency,
		Items: []payment.LineItem{{
			Name:        checkoutItemCourse,
			AmountCents: amountCents,
		}},
		Metadata: map[string]string{"purchase_id": p.ID},
	})
	if err != nil {
		_ = s.CancelPending(ctx, p.ID)
		return nil, fmt.Errorf("create checkout: %w", err)
	}

	p.ProviderSessionID = session.ID
	if err := s.purchases.Update(ctx, &p); err != nil {
		return nil, fmt.Errorf("update purchase session: %w", err)
	}

	p.CheckoutURL = session.URL
	return &p, nil
}

func (s *purchaseService) BuyBundle(ctx context.Context, userID, bundleID string, amountCents int64, currency string) (*model.Purchase, error) {
	var p model.Purchase

	err := s.db.WithTx(ctx, func(q database.Querier) error {
		ps := store.NewPurchaseStore(q)
		purchase := &model.Purchase{
			UserID:      userID,
			Type:        model.PurchaseTypeBundle,
			ItemID:      bundleID,
			Status:      model.PurchaseStatusPending,
			AmountCents: amountCents,
			Currency:    currency,
			Provider:    payment.ProviderStripe,
		}
		if err := ps.Create(ctx, purchase); err != nil {
			return fmt.Errorf("create purchase: %w", err)
		}
		p = *purchase
		return nil
	})
	if err != nil {
		return nil, err
	}

	session, err := s.provider.CreateCheckout(ctx, payment.Order{
		Currency: currency,
		Items: []payment.LineItem{{
			Name:        checkoutItemBundle,
			AmountCents: amountCents,
		}},
		Metadata: map[string]string{"purchase_id": p.ID},
	})
	if err != nil {
		_ = s.CancelPending(ctx, p.ID)
		return nil, fmt.Errorf("create checkout: %w", err)
	}

	p.ProviderSessionID = session.ID
	if err := s.purchases.Update(ctx, &p); err != nil {
		return nil, fmt.Errorf("update purchase session: %w", err)
	}

	p.CheckoutURL = session.URL
	return &p, nil
}

func (s *purchaseService) ConfirmPayment(ctx context.Context, purchaseID, providerPaymentID string) error {
	p, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPurchaseNotFound
	}
	if p.Status != model.PurchaseStatusPending {
		return ErrPurchaseAlreadyProcessed
	}
	return s.db.WithTx(ctx, func(q database.Querier) error {
		p.Status = model.PurchaseStatusCompleted
		p.ProviderPaymentID = providerPaymentID
		ps := store.NewPurchaseStore(q)
		if err := ps.Update(ctx, p); err != nil {
			return fmt.Errorf("update purchase: %w", err)
		}
		es := store.NewEnrollmentStore(q)
		return s.createEnrollments(ctx, es, p)
	})
}

func (s *purchaseService) CancelPending(ctx context.Context, purchaseID string) error {
	p, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPurchaseNotFound
	}
	if p.Status != model.PurchaseStatusPending {
		return nil
	}
	return s.db.WithTx(ctx, func(q database.Querier) error {
		p.Status = model.PurchaseStatusFailed
		return store.NewPurchaseStore(q).Update(ctx, p)
	})
}

func (s *purchaseService) createEnrollments(ctx context.Context, es store.EnrollmentStore, p *model.Purchase) error {
	courseIDs := []string{p.ItemID}
	if p.Type == model.PurchaseTypeBundle {
		assignments, err := s.bundles.List(ctx, p.ItemID)
		if err != nil {
			return fmt.Errorf("list bundle courses: %w", err)
		}
		courseIDs = make([]string, len(assignments))
		for i, a := range assignments {
			courseIDs[i] = a.CourseID
		}
	}
	for _, courseID := range courseIDs {
		enrollment := &model.CourseEnrollment{
			UserID:     p.UserID,
			CourseID:   courseID,
			PurchaseID: &p.ID,
			Status:     model.EnrollmentStatusActive,
			Source:     model.EnrollmentSourcePurchase,
		}
		if err := es.Create(ctx, enrollment); err != nil {
			return fmt.Errorf("create enrollment for course %s: %w", courseID, err)
		}
	}
	return nil
}

func (s *purchaseService) Refund(ctx context.Context, purchaseID string) error {
	p, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		return fmt.Errorf("refund: find purchase: %w", err)
	}
	if p == nil {
		return ErrPurchaseNotFound
	}
	if p.ProviderPaymentID != "" {
		if err := s.provider.Refund(ctx, p.ProviderPaymentID, p.AmountCents); err != nil {
			return fmt.Errorf("refund: provider: %w", err)
		}
	}
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
