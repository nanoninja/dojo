// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package mailer

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/nanoninja/dojo/internal/service"
)

// SMTPConfig holds the connection settings for an outgoing SMTP server.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type smtpMailer struct {
	cfg SMTPConfig
}

// NewSMTP creates a Mailer that delivers emails via SMTP.
func NewSMTP(cfg SMTPConfig) service.Mailer {
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) Send(_ context.Context, msg service.MailMessage) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	body, contentType, err := buildBody(msg)
	if err != nil {
		return fmt.Errorf("building email body: %w", err)
	}

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n\r\n",
		m.cfg.From, sanitizeHeader(msg.To), sanitizeHeader(msg.Subject), contentType,
	)

	return smtp.SendMail(addr, auth, m.cfg.From, []string{msg.To}, []byte(headers+body))
}

// sanitizeHeader removes CR and LF characters to prevent header injection.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "")
}

// buildBody returns the email body and its Content-Type header.
// If both HTML and Text are set, it builds a multipart/alternative message.
func buildBody(msg service.MailMessage) (string, string, error) {
	if msg.HTML != "" && msg.Text != "" {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)

		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/plain; charset=UTF-8")
		part, err := w.CreatePart(h)
		if err != nil {
			return "", "", err
		}
		if _, err = part.Write([]byte(msg.Text)); err != nil {
			return "", "", err
		}

		h = make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/html; charset=UTF-8")
		part, err = w.CreatePart(h)
		if err != nil {
			return "", "", err
		}
		if _, err = part.Write([]byte(msg.HTML)); err != nil {
			return "", "", err
		}

		if err = w.Close(); err != nil {
			return "", "", err
		}
		return buf.String(), fmt.Sprintf(`multipart/alternative; boundary="%s"`, w.Boundary()), nil
	}

	if msg.HTML != "" {
		return msg.HTML, "text/html; charset=UTF-8", nil
	}

	return msg.Text, "text/plain; charset=UTF-8", nil
}
