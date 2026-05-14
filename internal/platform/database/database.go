// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package database provides a configured PostgreSQL connection pool via sqlx.
package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers pgx as the database/sql driver
	"github.com/jmoiron/sqlx"
	"github.com/nanoninja/dojo/internal/config"
)

// Querier is implemented by both *DB and *sqlx.Tx.
// Stores accept a Querier so they can operate within or outside a transaction.
type Querier interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Rebind(query string) string
}

// TxRunner can execute a function within a database transaction.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(q Querier) error) error
}

// DB wraps sqlx.DB to provide a configured PostgreSQL connection pool.
type DB struct {
	*sqlx.DB
}

// Open opens a new database connection pool using the provided DSN.
// It verifies the connection with a ping before returning.
func Open(cfg config.DB) (*DB, error) {
	db, err := sqlx.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return &DB{db}, nil
}

// WithTx executes fn within a database transaction.
// It commits if fn returns nil, rolls back otherwise.
func (db *DB) WithTx(ctx context.Context, fn func(q Querier) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %w (original error: %v)", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}
