// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package testutil

import (
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers pgx as the database/sql driver

	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/platform/security"
	"github.com/pressly/goose/v3"
)

// defaultTestDSN is the fallback DSN when TEST_DB_DSN is not set.
// Uses port 5433 to avoid conflict with the dev database on 5432.
const defaultTestDSN = "postgres://admin:admin@localhost:5433/dojo_test?sslmode=disable"

// OpenTestDB opens a connection to the test database.
// The connection is automatically closed when the test ends.
func OpenTestDB(t testing.TB) *database.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	db, err := database.Open(config.DB{
		DSN:             dsn,
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { db.Close() }) //nolint:errcheck
	return db
}

// RunMigrations applies all pending goose migrations from the given directory.
func RunMigrations(dir string) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("opening DB for migrations: %v", err)
	}
	defer db.Close() //nolint:errcheck

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("setting goose dialect: %v", err)
	}
	if err := goose.Up(db, dir); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
}

// NewTestCipher creates an AES cipher using APP_ENCRYPTION_KEY env var.
func NewTestCipher(t testing.TB) *security.Cipher {
	t.Helper()

	key := os.Getenv("APP_ENCRYPTION_KEY")
	if key == "" {
		key = "test-key-32-bytes-for-testing!!!"
	}

	c, err := security.NewAESCipher(key)
	if err != nil {
		t.Fatalf("creating cipher: %v", err)
	}
	return c
}

// TruncateTable deletes all rows from the given tables.
// In PostgreSQL, we use TRUNCATE ... CASCADE to handle foreign key constraints
// or we truncate multiple tables in a single command.
func TruncateTable(t testing.TB, db *database.DB, tables ...string) {
	t.Helper()

	for _, table := range tables {
		// CASCADE removes dependent rows automatically
		// without manually disabling foreign key constraints.
		query := "TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE"

		if _, err := db.Exec(query); err != nil {
			t.Fatalf("truncating table %s: %v", table, err)
		}
	}
}
