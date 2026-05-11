// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/nanoninja/dojo/internal/testutil"
)

func TestMain(m *testing.M) {
	// .env.test is optional — system environment variables are used as fallback.
	_ = godotenv.Load("../../.env.test")
	testutil.RunMigrations("../../db/migrations")
	os.Exit(m.Run())
}
