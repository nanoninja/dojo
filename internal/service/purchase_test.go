// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/payment"
	"github.com/nanoninja/dojo/internal/service"
)

// ============================================================================
// fakePurchaseStore
// ============================================================================

type fakePurchaseStore struct {
	purchases map[string]*model.Purchase
	seq       int
}

func newFakePurchaseStore() *fakePurchaseStore {
	return &fakePurchaseStore{purchases: map[string]*model.Purchase{}}
}

func (f *fakePurchaseStore) nextID() string {
	f.seq++
	return fmt.Sprintf("purchase-%d", f.seq)
}

func (f *fakePurchaseStore) FindByID(_ context.Context, id string) (*model.Purchase, error) {
	p, ok := f.purchases[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (f *fakePurchaseStore) ListByUser(_ context.Context, userID string) ([]model.Purchase, error) {
	result := make([]model.Purchase, 0)
	for _, p := range f.purchases {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (f *fakePurchaseStore) Create(_ context.Context, p *model.Purchase) error {
	p.ID = f.nextID()
	cp := *p
	f.purchases[p.ID] = &cp
	return nil
}

func (f *fakePurchaseStore) Update(_ context.Context, p *model.Purchase) error {
	if existing, ok := f.purchases[p.ID]; ok {
		existing.Status = p.Status
		existing.ProviderSessionID = p.ProviderSessionID
		existing.ProviderPaymentID = p.ProviderPaymentID
	}
	// ID not found is acceptable: noopQuerier inside WithTx does not persist creates,
	// so the post-transaction Update on the outer store may target an unknown ID.
	return nil
}

func (f *fakePurchaseStore) Refund(_ context.Context, id string) error {
	p, ok := f.purchases[id]
	if !ok {
		return fmt.Errorf("purchase not found")
	}
	p.Status = model.PurchaseStatusRefunded
	return nil
}

// ============================================================================
// helpers
// ============================================================================

func newFakeBundleCourseStore(courseIDs ...string) *fakeBundleCourseStore {
	assignments := make([]model.BundleCourseAssignment, 0, len(courseIDs))
	for i, id := range courseIDs {
		assignments = append(assignments, model.BundleCourseAssignment{
			BundleID:  "bundle-1",
			CourseID:  id,
			SortOrder: i,
		})
	}
	return &fakeBundleCourseStore{assignments: assignments}
}

// fakeProvider is a no-op payment.Provider for unit tests.
type fakeProvider struct {
	session payment.Session
	err     error
}

func (f *fakeProvider) CreateCheckout(_ context.Context, _ payment.Order) (payment.Session, error) {
	return f.session, f.err
}

func (f *fakeProvider) HandleWebhook(_ []byte, _ string) (payment.Event, error) {
	return payment.Event{}, nil
}

func (f *fakeProvider) Refund(_ context.Context, _ string, _ int64) error { return nil }

func newPurchaseService(
	purchases *fakePurchaseStore,
	enrollments *fakeEnrollmentStore,
	bundles *fakeBundleCourseStore,
	txErr error,
) service.PurchaseService {
	tx := &fakeTxRunner{err: txErr}
	provider := &fakeProvider{session: payment.Session{ID: "sess_test", URL: "https://checkout.stripe.com/test"}}
	return service.NewPurchaseService(tx, provider, purchases, enrollments, bundles)
}

// ============================================================================
// Tests
// ============================================================================

func TestPurchaseService_GetByID(t *testing.T) {
	ctx := context.Background()
	ps := newFakePurchaseStore()
	svc := newPurchaseService(ps, newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)

	p := &model.Purchase{UserID: "user-1", Type: model.PurchaseTypeCourse, ItemID: "course-1", AmountCents: 1999, Currency: "EUR"}
	assert.NoError(t, ps.Create(ctx, p))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, p.ID)
		assert.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "non-existent")
		assert.ErrorIs(t, err, service.ErrPurchaseNotFound)
	})
}

func TestPurchaseService_ListByUser(t *testing.T) {
	ctx := context.Background()
	ps := newFakePurchaseStore()
	svc := newPurchaseService(ps, newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)

	assert.NoError(t, ps.Create(ctx, &model.Purchase{UserID: "user-1", ItemID: "course-1", AmountCents: 1999, Currency: "EUR"}))
	assert.NoError(t, ps.Create(ctx, &model.Purchase{UserID: "user-1", ItemID: "course-2", AmountCents: 2999, Currency: "EUR"}))
	assert.NoError(t, ps.Create(ctx, &model.Purchase{UserID: "user-2", ItemID: "course-1", AmountCents: 1999, Currency: "EUR"}))

	t.Run("returns purchases for user", func(t *testing.T) {
		got, err := svc.ListByUser(ctx, "user-1")
		assert.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		got, err := svc.ListByUser(ctx, "unknown")
		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})
}

func TestPurchaseService_BuyCourse(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		p, err := svc.BuyCourse(ctx, "user-1", "course-1", 1999, "EUR")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, model.PurchaseStatusPending, p.Status)
		assert.Equal(t, "https://checkout.stripe.com/test", p.CheckoutURL)
		assert.Equal(t, "sess_test", p.ProviderSessionID)
	})

	t.Run("transaction failure", func(t *testing.T) {
		txErr := errors.New("db unavailable")
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), txErr)
		_, err := svc.BuyCourse(ctx, "user-1", "course-1", 1999, "EUR")
		assert.ErrorIs(t, err, txErr)
	})

	t.Run("provider failure cancels purchase", func(t *testing.T) {
		ps := newFakePurchaseStore()
		tx := &fakeTxRunner{}
		provider := &fakeProvider{err: errors.New("stripe unavailable")}
		svc := service.NewPurchaseService(tx, provider, ps, newFakeEnrollmentStore(), newFakeBundleCourseStore())
		_, err := svc.BuyCourse(ctx, "user-1", "course-1", 1999, "EUR")
		assert.Error(t, err)
	})
}

func TestPurchaseService_BuyBundle(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		bundles := newFakeBundleCourseStore("course-1", "course-2", "course-3")
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), bundles, nil)
		p, err := svc.BuyBundle(ctx, "user-1", "bundle-1", 4999, "EUR")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, model.PurchaseStatusPending, p.Status)
		assert.Equal(t, "https://checkout.stripe.com/test", p.CheckoutURL)
		assert.Equal(t, "sess_test", p.ProviderSessionID)
	})

	t.Run("transaction failure", func(t *testing.T) {
		txErr := errors.New("db unavailable")
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), txErr)
		_, err := svc.BuyBundle(ctx, "user-1", "bundle-1", 4999, "EUR")
		assert.ErrorIs(t, err, txErr)
	})
}

func TestPurchaseService_Refund(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.NoError(t, svc.Refund(ctx, "purchase-1"))
	})

	t.Run("transaction failure", func(t *testing.T) {
		txErr := errors.New("db unavailable")
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), txErr)
		err := svc.Refund(ctx, "purchase-1")
		assert.ErrorIs(t, err, txErr)
	})
}

func TestPurchaseService_ConfirmPayment(t *testing.T) {
	ctx := context.Background()

	seed := func() (*fakePurchaseStore, string) {
		ps := newFakePurchaseStore()
		p := &model.Purchase{UserID: "user-1", Type: model.PurchaseTypeCourse, ItemID: "course-1", Status: model.PurchaseStatusPending, AmountCents: 1999, Currency: "EUR"}
		_ = ps.Create(ctx, p)
		return ps, p.ID
	}

	t.Run("success", func(t *testing.T) {
		ps, id := seed()
		svc := newPurchaseService(ps, newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.NoError(t, svc.ConfirmPayment(ctx, id, "pi_test"))
	})

	t.Run("not found", func(t *testing.T) {
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.ErrorIs(t, svc.ConfirmPayment(ctx, "non-existent", "pi_test"), service.ErrPurchaseNotFound)
	})

	t.Run("already processed", func(t *testing.T) {
		ps, id := seed()
		p, _ := ps.FindByID(ctx, id)
		p.Status = model.PurchaseStatusCompleted
		_ = ps.Update(ctx, p)
		svc := newPurchaseService(ps, newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.ErrorIs(t, svc.ConfirmPayment(ctx, id, "pi_test"), service.ErrPurchaseAlreadyProcessed)
	})

	t.Run("transaction failure", func(t *testing.T) {
		ps, id := seed()
		txErr := errors.New("db unavailable")
		tx := &fakeTxRunner{err: txErr}
		provider := &fakeProvider{session: payment.Session{ID: "sess_test", URL: "https://checkout.stripe.com/test"}}
		svc := service.NewPurchaseService(tx, provider, ps, newFakeEnrollmentStore(), newFakeBundleCourseStore())
		assert.ErrorIs(t, svc.ConfirmPayment(ctx, id, "pi_test"), txErr)
	})
}

func TestPurchaseService_CancelPending(t *testing.T) {
	ctx := context.Background()

	seed := func() (*fakePurchaseStore, string) {
		ps := newFakePurchaseStore()
		p := &model.Purchase{UserID: "user-1", Type: model.PurchaseTypeCourse, ItemID: "course-1", Status: model.PurchaseStatusPending, AmountCents: 1999, Currency: "EUR"}
		_ = ps.Create(ctx, p)
		return ps, p.ID
	}

	t.Run("success", func(t *testing.T) {
		ps, id := seed()
		svc := newPurchaseService(ps, newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.NoError(t, svc.CancelPending(ctx, id))
	})

	t.Run("not found", func(t *testing.T) {
		svc := newPurchaseService(newFakePurchaseStore(), newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.ErrorIs(t, svc.CancelPending(ctx, "non-existent"), service.ErrPurchaseNotFound)
	})

	t.Run("already completed is a no-op", func(t *testing.T) {
		ps, id := seed()
		p, _ := ps.FindByID(ctx, id)
		p.Status = model.PurchaseStatusCompleted
		_ = ps.Update(ctx, p)
		svc := newPurchaseService(ps, newFakeEnrollmentStore(), newFakeBundleCourseStore(), nil)
		assert.NoError(t, svc.CancelPending(ctx, id))
	})

	t.Run("transaction failure", func(t *testing.T) {
		ps, id := seed()
		txErr := errors.New("db unavailable")
		tx := &fakeTxRunner{err: txErr}
		provider := &fakeProvider{session: payment.Session{ID: "sess_test", URL: "https://checkout.stripe.com/test"}}
		svc := service.NewPurchaseService(tx, provider, ps, newFakeEnrollmentStore(), newFakeBundleCourseStore())
		assert.ErrorIs(t, svc.CancelPending(ctx, id), txErr)
	})
}
