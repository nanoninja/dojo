// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package mailer

import (
	"context"

	"github.com/nanoninja/dojo/internal/service"
)

// Mock is a no-op Mailer for use in tests.
// It records all sent messages so they can be inspected in assertions.
type Mock struct {
	Messages []service.MailMessage
}

// NewMock creates a new Mock mailer.
func NewMock() service.Mailer {
	return &Mock{}
}

// Send records the message without delivering it.
func (m *Mock) Send(_ context.Context, msg service.MailMessage) error {
	m.Messages = append(m.Messages, msg)
	return nil
}
