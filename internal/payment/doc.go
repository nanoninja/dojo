// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package payment defines the abstraction layer for payment providers.
// A Provider implementation handles checkout creation, webhook processing,
// and refunds. The business logic (purchase, enrollment) is decoupled from
// any specific provider so that Stripe, PayPal, or others can be swapped
// without touching the service layer.
package payment
