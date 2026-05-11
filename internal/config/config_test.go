// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package config

import (
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
)

// SetValidEnv defines a minimal valid environment for Load() in production mode.
// Production mode avoids loading .env automatically, which keeps tests deterministic.
func setValidEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ENCRYPTION_KEY", "12345678901234567890123456789012") // 32 bytes
	t.Setenv("JWT_SECRET", "test-jwt-secret-32-chars-min-value")
	t.Setenv("DB_NAME", "dojo")
	t.Setenv("DB_SSLMODE", "require")

	// Numeric env values parsed by Load().
	t.Setenv("JWT_EXPIRY_HOURS", "24")
	t.Setenv("JWT_REFRESH_EXPIRY_DAYS", "7")

	// Mailer
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("MAIL_TIMEOUT_MS", "3000")
	t.Setenv("MAIL_RETRY_ATTEMPTS", "3")
	t.Setenv("MAIL_RETRY_BASE_DELAY_MS", "200")

	// Auth transport
	t.Setenv("AUTH_TRANSPORT_MODE", "dual")
	t.Setenv("AUTH_COOKIE_ACCESS_NAME", "access_token")
	t.Setenv("AUTH_COOKIE_REFRESH_NAME", "refresh_token")
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	t.Setenv("AUTH_COOKIE_PATH", "/")
	t.Setenv("AUTH_COOKIE_SAMESITE", "lax")
}

func TestLoad_Success(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "require", cfg.Database.SSLMode)
	require.StringContains(t, cfg.Database.DSN, "sslmode=require")
}

func TestLoad_DSN_IncludesSSLCerts(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_SSLROOTCERT", "/path/to/ca.crt")
	t.Setenv("DB_SSLCERT", "/path/to/client.crt")
	t.Setenv("DB_SSLKEY", "/path/to/client.key")

	cfg, err := Load()
	require.NoError(t, err)
	require.StringContains(t, cfg.Database.DSN, "sslrootcert=/path/to/ca.crt")
	require.StringContains(t, cfg.Database.DSN, "sslcert=/path/to/client.crt")
	require.StringContains(t, cfg.Database.DSN, "sslkey=/path/to/client.key")
}

func TestLoad_MissingDBName_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_NAME", "")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "DB_NAME")
}

func TestLoad_InvalidEncryptionKeyLength_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ENCRYPTION_KEY", "too-short")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "APP_ENCRYPTION_KEY")
}

func TestLoad_ProductionDisableSSLMode_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_SSLMODE", "disable")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "DB_SSLMODE=disable")
}

func TestLoad_InvalidJWTExpiry_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_EXPIRY_HOURS", "invalid")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "JWT_EXPIRY_HOURS")
}

func TestLoad_InvalidMailTimeout_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MAIL_TIMEOUT_MS", "invalid")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "MAIL_TIMEOUT_MS")
}

func TestLoad_InvalidMailRetryAttempts_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MAIL_RETRY_ATTEMPTS", "invalid")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "MAIL_RETRY_ATTEMPTS")
}

func TestLoad_InvalidMailRetryBaseDelay_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MAIL_RETRY_BASE_DELAY_MS", "invalid")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "MAIL_RETRY_BASE_DELAY_MS")
}

func TestLoad_InvalidAuthTransportMode_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("AUTH_TRANSPORT_MODE", "invalid")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "AUTH_TRANSPORT_MODE")
}

func TestLoad_InvalidAuthCookieSameSite_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("AUTH_COOKIE_SAMESITE", "invalid")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "AUTH_COOKIE_SAMESITE")
}

func TestLoad_ProductionCookieModeWithoutSecure_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("AUTH_TRANSPORT_MODE", "cookie")
	t.Setenv("AUTH_COOKIE_SECURE", "false")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "AUTH_COOKIE_SECURE=true")
}

func TestLoad_ProductionJWTSecretTooShort_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_SECRET", "too-short")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
}

func TestLoad_CookieModeWithWildcardOrigin_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("AUTH_TRANSPORT_MODE", "cookie")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	_, err := Load()
	require.Error(t, err)
	require.StringContains(t, err.Error(), "CORS_ALLOWED_ORIGINS cannot contain '*'")
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := Config{App: App{Env: "development"}}
	assert.True(t, cfg.IsDevelopment())

	cfg.App.Env = "production"
	assert.False(t, cfg.IsDevelopment())
}
