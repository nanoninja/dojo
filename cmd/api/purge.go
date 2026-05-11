// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

// runAuditPurge starts a background goroutine that deletes old login audit
// log entries once per day at the configured hour, in small batches to avoid
// locking the table.
func runAuditPurge(
	ctx context.Context,
	cfg config.AuditPurge,
	s store.LoginAuditStore,
	db *database.DB,
	logger *slog.Logger,
) {
	if !cfg.Enabled {
		return
	}

	go func() {
		for {
			next := nextSchedule(cfg.ScheduleHour)
			logger.Info("audit purge scheduled", "next", next.Format(time.RFC3339))

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(next)):
			}

			runPurgeBatches(ctx, cfg, s, db, logger)
		}
	}()
}

// runPurgeBatches deletes old audit entries in batches until none remain,
// pausing between each batch to reduce database pressure.
func runPurgeBatches(
	ctx context.Context,
	cfg config.AuditPurge,
	s store.LoginAuditStore,
	db *database.DB,
	logger *slog.Logger,
) {
	acquired, err := tryAcquireLock(ctx, db)
	if err != nil {
		logger.Error("audit purge lock failed", "error", err)
		return
	}
	if !acquired {
		logger.Info("audit purge skipped: another instance is running")
		return
	}
	defer releaseLock(ctx, db)

	total := int64(0)

	for {
		deleted, err := s.Purge(ctx, cfg.Retention, cfg.BatchSize)
		if err != nil {
			logger.Error("audit purge failed", "error", err)
			return
		}

		total += deleted

		if deleted == 0 {
			logger.Info("audit purge complete", "deleted", total)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(cfg.BatchPause):
		}
	}
}

// nextSchedule returns the next occurrence of the given hour (local time).
// If that hour has already passed today, it returns tomorrow's occurrence.
func nextSchedule(hour int) time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// tryAcquireLock attempts to acquire a PostgreSQL session-level advisory lock.
// Returns false without error if another instance already holds the lock.
func tryAcquireLock(ctx context.Context, db *database.DB) (bool, error) {
	var acquired bool
	err := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext('audit_purge'))").Scan(&acquired)
	return acquired, err
}

// releaseLock releases the advisory lock acquired by tryAcquireLock.
func releaseLock(ctx context.Context, db *database.DB) {
	_, _ = db.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext('audit_purge'))")
}
