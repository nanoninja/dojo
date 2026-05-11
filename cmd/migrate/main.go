// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package main is the database migration CLI entry point (goose).
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/pressly/goose/v3"
)

const migrationDir = "db/migrations"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: migrate [up|down|status|reset]")
	}
	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close() //nolint:errcheck

	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting dialect: %w", err)
	}

	sqlDB := db.DB.DB

	switch command {
	case "up":
		err = goose.Up(sqlDB, migrationDir)
	case "down":
		err = goose.Down(sqlDB, migrationDir)
	case "status":
		err = goose.Status(sqlDB, migrationDir)
	case "reset":
		err = goose.Reset(sqlDB, migrationDir)
	default:
		err = fmt.Errorf("unknown command: %s", command)
	}

	if err != nil {
		return fmt.Errorf("running migration %q: %w", command, err)
	}

	logger.Info("migration done", "command", command)
	return nil
}
