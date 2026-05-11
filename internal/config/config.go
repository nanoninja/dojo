// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	App           App
	Database      DB
	Redis         Redis
	JWT           JWT
	CORS          CORS
	SMTP          SMTP
	MailDispatch  MailDispatch
	AuthTransport AuthTransport
	AuditPurge    AuditPurge
}

// App contains HTTP server settings.
type App struct {
	Name              string   // Application name, used in logs and HTTP headers.
	Version           string   // Application version, used in health checks and HTTP headers.
	Env               string   // Runtime environment: development, production or test.
	Host              string   // Host address to listen on: 0.0.0.0 in Docker/prod, localhost in dev.
	Port              string   // TCP port the HTTP server listens on.
	EncryptionKey     string   // 32-byte key for AES-256-GCM field encryption.
	Debug             bool     // Enables verbose logging and stack traces when true.
	MetricsAllowedIPs []string // Comma-separated IPs allowed to access /metrics. Empty = unrestricted.
}

// DB contains PostgreSQL connection settings.
type DB struct {
	User            string        // Database user name.
	Password        string        // Database user password.
	Host            string        // Database server host.
	Port            string        // Database server port.
	Name            string        // Database name.
	SSLMode         string        // TLS mode: disable, require, verify-ca, verify-full.
	SSLRootCert     string        // Optional CA certificate path for server cert verification.
	SSLCert         string        // Optional client certificate path (mTLS).
	SSLKey          string        // Optional client private key path (mTLS).
	MaxOpenConns    int           // Maximum number of open connections to the database.
	MaxIdleConns    int           // Maximum number of connections in the idle pool.
	ConnMaxLifetime time.Duration // Maximum amount of time a connection may be reused.
	ConnMaxIdleTime time.Duration // Maximum amount of time a connection may be idle.
	DSN             string        // Data Source Name built from the fields above
}

func (db DB) buildDSN() string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		db.User, db.Password, db.Host, db.Port, db.Name, db.SSLMode,
	)
	if db.SSLRootCert != "" {
		dsn += "&sslrootcert=" + db.SSLRootCert
	}
	if db.SSLCert != "" {
		dsn += "&sslcert=" + db.SSLCert
	}
	if db.SSLKey != "" {
		dsn += "&sslkey=" + db.SSLKey
	}
	return dsn
}

// AuditPurge contains settings for the background login audit log cleanup job.
type AuditPurge struct {
	Enabled      bool
	Retention    time.Duration // how old an entry must be to be deleted
	BatchSize    int           // rows deleted per iteration
	BatchPause   time.Duration // wait between batches
	ScheduleHour int           // hour of day to start the purge (0-23)
}

// Redis contains Redis connection settings.
type Redis struct {
	Addr         string        // Redis server address in host:port format.
	Password     string        // Redis authentication password, empty if none.
	DB           int           // Redis database index (0-15).
	PoolSize     int           // Maximum number of socket connections per CPU.
	MaxRetries   int           // Maximum number of retries before giving up.
	DialTimeout  time.Duration // Timeout for establishing new connections.
	ReadTimeout  time.Duration // Timeout for socket reads.
	WriteTimeout time.Duration // Timeout for socket writes.
}

// JWT contains JSON Web Token settings.
type JWT struct {
	Secret            string // Signing secret, must be at least 32 characters in production.
	ExpiryHours       int    // Token validity duration in hours.
	RefreshExpiryDays int    // Refresh token validity duration in days.
}

// CORS contains cross-origin resource sharing settings.
type CORS struct {
	AllowedOrigins []string // Allowed origins, e.g. https://example.com or https://*.example.com
	MaxAge         int      // Preflight cache duration in seconds.
}

// SMTP contains outgoing mail server settings.
type SMTP struct {
	Host     string // SMTP server host.
	Port     int    // SMTP server port (25, 465 or 587).
	Username string // SMTP authentication username.
	Password string // SMTP authentication password.
	From     string // Sender address used in the From header.
}

// MailDispatch contains reliability settings for synchronous mail sending
// (timeouts + retries), while keeping the architecture ready for async later.
type MailDispatch struct {
	Enabled        bool // Enables resilient mail dispatch wrapper.
	TimeoutMS      int  // Per-attempt timeout in milliseconds.
	RetryAttempts  int  // Total attempts (including the first try).
	RetryBaseDelay int  // Initial backoff delay in milliseconds.
}

// AuthTransport contains token transport settings (bearer/cookie/dual).
type AuthTransport struct {
	Mode              string // bearer | cookie | dual
	AccessCookieName  string // Access token cookie name when cookie transport is enabled.
	RefreshCookieName string // Refresh token cookie name when cookie transport is enabled.
	CookieSecure      bool   // Adds the Secure attribute; must be true in production for cookie/dual mode.
	CookieDomain      string // Optional cookie domain scope.
	CookiePath        string // Cookie path scope, usually "/".
	CookieSameSite    string // lax | strict | none
}

// Load reads configuration from environment variables.
// In development, it automatically loads a .env file if present.
// In production, the .env file is intentionally skipped — variables must be
// injected by the infrastructure (Docker, Kubernetes, etc.) to avoid silently
// overriding secrets with stale file values.
func Load() (*Config, error) {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}

	sslMode := "disable"
	if getEnv("APP_ENV", "development") == "production" {
		sslMode = getEnv("DB_SSLMODE", "require")
	}

	p := &parser{}

	db := DB{
		User:            getEnv("DB_USER", "root"),
		Password:        os.Getenv("DB_PASSWORD"),
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		Name:            os.Getenv("DB_NAME"),
		SSLMode:         sslMode,
		SSLRootCert:     os.Getenv("DB_SSLROOTCERT"),
		SSLCert:         os.Getenv("DB_SSLCERT"),
		SSLKey:          os.Getenv("DB_SSLKEY"),
		MaxOpenConns:    p.int("DB_POOL_MAX_OPEN", 25),
		MaxIdleConns:    p.int("DB_POOL_MAX_IDLE", 10),
		ConnMaxLifetime: p.duration("DB_POOL_CONN_MAX_LIFETIME_SECONDS", 300, time.Second),
		ConnMaxIdleTime: p.duration("DB_POOL_CONN_MAX_IDLE_SECONDS", 120, time.Second),
	}
	db.DSN = db.buildDSN()

	cfg := &Config{
		App: App{
			Name:          getEnv("APP_NAME", "dojo"),
			Version:       getEnv("APP_VERSION", "dev"),
			Env:           getEnv("APP_ENV", "development"),
			Host:          getEnv("APP_HOST", "localhost"),
			Port:          getEnv("APP_PORT", "8080"),
			EncryptionKey: os.Getenv("APP_ENCRYPTION_KEY"),
			Debug:         getEnv("APP_DEBUG", "false") == "true",
			MetricsAllowedIPs: strings.FieldsFunc(os.Getenv("METRICS_ALLOWED_IPS"), func(r rune) bool {
				return r == ',' || r == ' '
			}),
		},
		Database: db,
		Redis: Redis{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           p.int("REDIS_DB", 0),
			PoolSize:     p.int("REDIS_POOL_SIZE", 10),
			MaxRetries:   p.int("REDIS_MAX_RETRIES", 3),
			DialTimeout:  p.duration("REDIS_DIAL_TIMEOUT_SECONDS", 5, time.Second),
			ReadTimeout:  p.duration("REDIS_READ_TIMEOUT_SECONDS", 3, time.Second),
			WriteTimeout: p.duration("REDIS_WRITE_TIMEOUT_SECONDS", 3, time.Second),
		},
		JWT: JWT{
			Secret:            os.Getenv("JWT_SECRET"),
			ExpiryHours:       p.int("JWT_EXPIRY_HOURS", 24),
			RefreshExpiryDays: p.int("JWT_REFRESH_EXPIRY_DAYS", 7),
		},
		CORS: CORS{
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
			MaxAge:         300,
		},
		SMTP: SMTP{
			Host:     getEnv("SMTP_HOST", "localhost"),
			Port:     p.int("SMTP_PORT", 1025),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     getEnv("SMTP_FROM", "noreply@example.com"),
		},
		MailDispatch: MailDispatch{
			Enabled:        getEnv("MAIL_DISPATCH_ENABLED", "true") == "true",
			TimeoutMS:      p.int("MAIL_TIMEOUT_MS", 3000),
			RetryAttempts:  p.int("MAIL_RETRY_ATTEMPTS", 3),
			RetryBaseDelay: p.int("MAIL_RETRY_BASE_DELAY_MS", 200),
		},
		AuthTransport: AuthTransport{
			Mode:              strings.ToLower(getEnv("AUTH_TRANSPORT_MODE", "dual")),
			AccessCookieName:  getEnv("AUTH_COOKIE_ACCESS_NAME", "access_token"),
			RefreshCookieName: getEnv("AUTH_COOKIE_REFRESH_NAME", "refresh_token"),
			CookieSecure:      getEnv("AUTH_COOKIE_SECURE", "false") == "true",
			CookieDomain:      os.Getenv("AUTH_COOKIE_DOMAIN"),
			CookiePath:        getEnv("AUTH_COOKIE_PATH", "/"),
			CookieSameSite:    strings.ToLower(getEnv("AUTH_COOKIE_SAMESITE", "lax")),
		},
		AuditPurge: AuditPurge{
			Enabled:      getEnv("AUDIT_PURGE_ENABLED", "true") == "true",
			Retention:    p.duration("AUDIT_PURGE_RETENTION_DAYS", 90, 24*time.Hour),
			BatchSize:    p.int("AUDIT_PURGE_BATCH_SIZE", 100),
			BatchPause:   p.duration("AUDIT_PURGE_BATCH_PAUSE_SECONDS", 300, time.Second),
			ScheduleHour: p.int("AUDIT_PURGE_SCHEDULE_HOUR", 2),
		},
	}

	if p.err != nil {
		return nil, p.err
	}
	if err := cfg.validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// IsDevelopment returns true when running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true when running in production mode.
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// validate checks that all required values are present and consistent.
func (c *Config) validate() error {
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME environment variable is required")
	}
	if c.IsProduction() && c.Database.SSLMode == "disable" {
		return fmt.Errorf("DB_SSLMODE=disable is not allowed in production")
	}

	if len(c.App.EncryptionKey) != 32 {
		return fmt.Errorf(
			"APP_ENCRYPTION_KEY must be exactly 32 bytes (got %d)",
			len(c.App.EncryptionKey),
		)
	}
	if c.IsProduction() && len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}

	switch c.AuthTransport.Mode {
	case "bearer", "cookie", "dual":
	default:
		return fmt.Errorf("AUTH_TRANSPORT_MODE must be bearer, cookie or dual")
	}

	switch c.AuthTransport.CookieSameSite {
	case "lax", "strict", "none":
	default:
		return fmt.Errorf("AUTH_COOKIE_SAMESITE must be lax, strict or none")
	}

	cookieMode := c.AuthTransport.Mode == "cookie" || c.AuthTransport.Mode == "dual"

	if c.IsProduction() && cookieMode && !c.AuthTransport.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE=true is required in production for cookie/dual mode")
	}

	if cookieMode {
		for _, origin := range c.CORS.AllowedOrigins {
			if strings.TrimSpace(origin) == "*" {
				return fmt.Errorf(
					"CORS_ALLOWED_ORIGINS cannot contain '*' when AUTH_TRANSPORT_MODE is cookie or dual",
				)
			}
		}
	}

	return nil
}

// getEnv returns the value of an environment variable,
// or a default value if the variable is not set.
func getEnv(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

// parser is an error-collecting helper for reading environment variables.
// Once a parse error occurs, all subsequent calls are no-ops and return their
// default value. The first error is retrieved via p.err after all reads are done.
// This avoids repetitive if-err-return blocks at each call site (errWriter pattern).
type parser struct {
	err error
}

// int reads an integer environment variable. Returns def if the variable is
// absent. Records an error and returns def if the value cannot be parsed.
func (p *parser) int(key string, def int) int {
	if p.err != nil {
		return def
	}
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		p.err = fmt.Errorf("invalid %s value: %w", key, err)
		return def
	}
	return i
}

// duration reads an integer environment variable and multiplies it by unit.
// It delegates to int, so parse errors are collected the same way.
func (p *parser) duration(key string, def int, unit time.Duration) time.Duration {
	return time.Duration(p.int(key, def)) * unit
}
