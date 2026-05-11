// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package main is the database seeding CLI entry point.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/platform/security"
	"github.com/nanoninja/dojo/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close() //nolint:errcheck

	cipher, err := security.NewAESCipher(cfg.App.EncryptionKey)
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}

	users := store.NewUserStore(db, cipher)
	ctx := context.Background()

	if err := seedSuperAdmin(ctx, users, logger); err != nil {
		return err
	}
	if err := seedSystem(ctx, users, logger); err != nil {
		return err
	}

	return nil
}

func seedSuperAdmin(ctx context.Context, users store.UserStore, logger *slog.Logger) error {
	email := os.Getenv("SEED_SUPERADMIN_EMAIL")
	password := os.Getenv("SEED_SUPERADMIN_PASSWORD")

	if email == "" || password == "" {
		return fmt.Errorf("SEED_SUPERADMIN_EMAIL and SEED_SUPERADMIN_PASSWORD are required")
	}

	existing, err := users.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("checking superadmin: %w", err)
	}
	if existing != nil {
		logger.Info("superadmin already exists, skipping", "email", email)
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	u := &model.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         model.RoleSuperAdmin,
		Status:       model.UserStatusActive,
		IsVerified:   true,
		Language:     "fr-FR",
		Timezone:     "Europe/Paris",
	}
	if err := users.Create(ctx, u); err != nil {
		return fmt.Errorf("creating superadmin: %w", err)
	}

	logger.Info("superadmin created", "email", email, "id", u.ID)
	return nil
}

func seedSystem(ctx context.Context, users store.UserStore, logger *slog.Logger) error {
	email := os.Getenv("SEED_SYSTEM_EMAIL")
	if email == "" {
		email = "system@internal.local"
	}

	existing, err := users.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("checking system account: %w", err)
	}
	if existing != nil {
		logger.Info("system account already exists, skipping", "email", email)
		return nil
	}

	// System account never logs in via password - generate a random unusable one.
	raw, err := security.RandomToken(32)
	if err != nil {
		return fmt.Errorf("generating system password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing system password: %w", err)
	}

	u := &model.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         model.RoleSystem,
		Status:       model.UserStatusActive,
		IsVerified:   true,
		Language:     "fr-FR",
		Timezone:     "Europe/Paris",
	}
	if err := users.Create(ctx, u); err != nil {
		return fmt.Errorf("creating system account: %w", err)
	}

	logger.Info("system account created", "email", email, "id", u.ID)
	return nil
}
