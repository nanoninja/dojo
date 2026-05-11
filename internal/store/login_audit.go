// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/platform/security"
)

// LoginAuditStore defines the data access contract for login audit logs.
type LoginAuditStore interface {
	// List returns a paginated slice of audit logs matching the filter, with total count.
	// Zero-value filter fields are ignored. Results are ordered by most recent first.
	List(ctx context.Context, f AuditFilter) ([]model.LoginAuditLog, int, error)

	// Create inserts a new login audit log entry and populates log.ID.
	Create(ctx context.Context, log *model.LoginAuditLog) error

	// FindByUser returns the most recent login audit logs for a given user.
	FindByUser(ctx context.Context, userID string, limit int) ([]model.LoginAuditLog, error)

	// Purge deletes at most batchSize audit log entries older than the given
	// duration and returns the number of rows deleted.
	// Call it in a loop until it returns 0 to fully drain old entries.
	Purge(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)
}

// AuditFilter defines optional criteria for querying login audit logs.
type AuditFilter struct {
	UserID *string
	Status model.LoginStatus
	Since  time.Time
	Until  time.Time
	Limit  int
	Offset int
}

type loginAuditStore struct {
	db     *database.DB
	cipher *security.Cipher
}

// NewLoginAuditStore creates a new LoginAuditStore.
func NewLoginAuditStore(db *database.DB, cipher *security.Cipher) LoginAuditStore {
	return &loginAuditStore{db: db, cipher: cipher}
}

func (s *loginAuditStore) List(ctx context.Context, f AuditFilter) ([]model.LoginAuditLog, int, error) {
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 6)
	where := ""

	if f.UserID != nil {
		conditions = append(conditions, "user_id = ?")
		args = append(args, *f.UserID)
	}
	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}
	if !f.Since.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, f.Since)
	}
	if !f.Until.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, f.Until)
	}
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := s.db.Rebind("SELECT COUNT(*) FROM login_audit_logs " + where)
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("counting audit logs: %w", err)
	}

	query := `
		SELECT id, user_id, email, ip_address, user_agent, status, created_at
		FROM login_audit_logs ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	args = append(args, f.Limit, f.Offset)

	var logs []model.LoginAuditLog
	if err := s.db.SelectContext(ctx, &logs, s.db.Rebind(query), args...); err != nil {
		return nil, 0, fmt.Errorf("listing audit log: %w", err)
	}
	if err := s.decryptLogs(logs); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (s *loginAuditStore) Create(ctx context.Context, log *model.LoginAuditLog) error {
	encEmail, err := s.cipher.Encrypt(log.Email)
	if err != nil {
		return fmt.Errorf("encrypting email: %w", err)
	}
	encIP, err := s.cipher.Encrypt(log.IPAddress)
	if err != nil {
		return fmt.Errorf("encrypting ip_address: %w", err)
	}
	return s.db.QueryRowContext(ctx, `
        INSERT INTO login_audit_logs (
            user_id,
            email,
            ip_address,
            user_agent,
            status
        ) VALUES (
            $1, $2, $3, $4, $5
        ) RETURNING id`,
		log.UserID,
		encEmail,
		encIP,
		log.UserAgent,
		log.Status,
	).Scan(&log.ID)
}

func (s *loginAuditStore) FindByUser(ctx context.Context, userID string, limit int) ([]model.LoginAuditLog, error) {
	var logs []model.LoginAuditLog
	err := s.db.SelectContext(ctx, &logs, `
		SELECT
			id,
			user_id,
			email,
			ip_address,
			user_agent,
			status,
			created_at
		FROM login_audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`,
		userID,
		limit,
	)
	if err := s.decryptLogs(logs); err != nil {
		return nil, err
	}
	return logs, err
}

func (s *loginAuditStore) Purge(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	query := s.db.Rebind(`
		DELETE FROM login_audit_logs
		WHERE id IN (
			SELECT id FROM login_audit_logs
			WHERE created_at < NOW() - (? * INTERVAL '1 second')
			LIMIT ?
	)`)
	result, err := s.db.ExecContext(ctx, query, int(olderThan.Seconds()), batchSize)
	if err != nil {
		return 0, fmt.Errorf("purging audit logs: %w", err)
	}
	return result.RowsAffected()
}

// decryptLogs decrypts email and ip_address fields in place for each log entry.
func (s *loginAuditStore) decryptLogs(logs []model.LoginAuditLog) error {
	for i := range logs {
		email, err := s.cipher.Decrypt(logs[i].Email)
		if err != nil {
			return fmt.Errorf("decrypting email: %w", err)
		}
		logs[i].Email = email

		ip, err := s.cipher.Decrypt(logs[i].IPAddress)
		if err != nil {
			return fmt.Errorf("decrypting ip_address: %w", err)
		}
		logs[i].IPAddress = ip
	}
	return nil
}
