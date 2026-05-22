// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"context"
	"time"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
)

// SubscriptionService defines business operations for user subscriptions.
type SubscriptionService interface {
	// GetActive returns the active subscription for a user, or ErrSubscriptionNotFound if none.
	GetActive(ctx context.Context, userID string) (*model.Subscription, error)

	// ListByUser returns all subscriptions for a user.
	ListByUser(ctx context.Context, userID string) ([]model.Subscription, error)

	// Subscribe creates a new subscription for a user.
	Subscribe(ctx context.Context, userID string, plan model.SubscriptionPlan) (*model.Subscription, error)

	// Cancel cancels the active subscription for a user.
	Cancel(ctx context.Context, id string) error

	// IsActive reports whether a user has an active subscription.
	IsActive(ctx context.Context, userID string) (bool, error)
}

type subscriptionService struct {
	subscriptions store.SubscriptionStore
}

// NewSubscriptionService creates a SubscriptionService backed by the given store.
func NewSubscriptionService(subscriptions store.SubscriptionStore) SubscriptionService {
	return &subscriptionService{subscriptions: subscriptions}
}

func (s *subscriptionService) GetActive(ctx context.Context, userID string) (*model.Subscription, error) {
	sub, err := s.subscriptions.FindActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (s *subscriptionService) ListByUser(ctx context.Context, userID string) ([]model.Subscription, error) {
	return s.subscriptions.ListByUser(ctx, userID)
}

func (s *subscriptionService) Subscribe(ctx context.Context, userID string, plan model.SubscriptionPlan) (*model.Subscription, error) {
	now := time.Now()

	var expiresAt time.Time
	switch plan {
	case model.SubscriptionPlanMonthly:
		expiresAt = now.AddDate(0, 1, 0)
	case model.SubscriptionPlanAnnual:
		expiresAt = now.AddDate(1, 0, 0)
	}

	sub := &model.Subscription{
		UserID:    userID,
		Plan:      plan,
		Status:    model.SubscriptionStatusActive,
		StartedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := s.subscriptions.Create(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *subscriptionService) Cancel(ctx context.Context, id string) error {
	return s.subscriptions.Cancel(ctx, id)
}

func (s *subscriptionService) IsActive(ctx context.Context, userID string) (bool, error) {
	sub, err := s.subscriptions.FindActiveByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return sub != nil, nil
}
