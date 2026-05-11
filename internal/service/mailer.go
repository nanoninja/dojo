// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
)

// MailMessage represents an email to be sent.
type MailMessage struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Mailer is the generic interface for sending emails.
type Mailer interface {
	Send(ctx context.Context, msg MailMessage) error
}

// AuthMailer defines the outgoing mail operations required by the auth service.
type AuthMailer interface {
	// SendAccountVerification sends an email verification link to a new user.
	SendAccountVerification(ctx context.Context, to, token string) error

	// SendPasswordReset sends a password reset link to the user.
	SendPasswordReset(ctx context.Context, to, token string) error

	// SendOTP sends a 6-digit one-time password for two-factor authentication.
	SendOTP(ctx context.Context, to, code string) error
}

// authMailer handles all authentication-related emails.
type authMailer struct {
	mailer    Mailer
	templates *template.Template
}

// NewAuthMailer creates a new AuthMailer using the given Mailer transport.
func NewAuthMailer(m Mailer, templates *template.Template) AuthMailer {
	return &authMailer{mailer: m, templates: templates}
}

func (m *authMailer) SendAccountVerification(ctx context.Context, to, token string) error {
	html, err := m.render("verification.html", map[string]string{"Token": token})
	if err != nil {
		return err
	}
	return m.mailer.Send(ctx, MailMessage{
		To:      to,
		Subject: "Verify your email address",
		HTML:    html,
		Text:    "Your verification token: " + token,
	})
}

func (m *authMailer) SendPasswordReset(ctx context.Context, to, token string) error {
	html, err := m.render("password_reset.html", map[string]string{"Token": token})
	if err != nil {
		return err
	}
	return m.mailer.Send(ctx, MailMessage{
		To:      to,
		Subject: "Reset your password",
		HTML:    html,
		Text:    "Your password reset token: " + token,
	})
}

func (m *authMailer) SendOTP(ctx context.Context, to, code string) error {
	html, err := m.render("otp.html", map[string]string{"Code": code})
	if err != nil {
		return err
	}
	return m.mailer.Send(ctx, MailMessage{
		To:      to,
		Subject: "Your login code",
		HTML:    html,
		Text:    "Your one-time code: " + code,
	})
}

func (m *authMailer) render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := m.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("rendering template %s: %w", name, err)
	}
	return buf.String(), nil
}
