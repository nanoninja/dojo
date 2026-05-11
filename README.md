# dojo

[![Version](https://img.shields.io/badge/version-v0.1.0-blue)](CHANGELOG.md)
[![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go)](https://golang.org/dl/)
[![CI](https://github.com/nanoninja/dojo/actions/workflows/ci.yaml/badge.svg)](https://github.com/nanoninja/dojo/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/nanoninja/dojo/branch/main/graph/badge.svg)](https://codecov.io/gh/nanoninja/dojo)
[![License](https://img.shields.io/badge/license-Proprietary-red)](LICENSE)

A production-ready Go 1.26 REST API template with authentication, encryption, and Docker support.

## Features

**Language**: Go 1.26+
- **Database**: PostgreSQL 18 with UUID v7 and `RETURNING` support
- **Cache**: Redis 8
- **Authentication**: JWT access tokens + rotating refresh tokens
- **2FA**: Email OTP flow
- **Encryption**: AES-256-GCM for sensitive database fields
- **Mailer**: SMTP with multipart HTML/text support — Mailpit for local dev
- **Migrations**: [Goose](https://github.com/pressly/goose)
- **Hot Reload**: [Air](https://github.com/air-verse/air)
- **RBAC**: Hierarchical roles (`user`, `moderator`, `manager`, `admin`, `superadmin`, `system`)
- **Security**: HTTP security headers (X-Frame-Options, HSTS, etc.)
- **Rate Limiting**: Per-IP on all sensitive auth routes
- **Linting**: golangci-lint with `errcheck`, `staticcheck`, `govet`, `revive`, `gofmt`
- **Docker**: Multi-environment Compose (dev, prod) — scratch-based production image with `ca-certificates`
- **Kubernetes**: Base manifests with init container for zero-downtime migrations
- **CI**: GitHub Actions with Codecov coverage gate (70% project / 60% patch)

## Prerequisites

- [Go 1.26+](https://golang.org/dl/)
- [Docker](https://www.docker.com/)
- [golangci-lint](https://golangci-lint.run/welcome/install/) — required for `make lint`

  ```bash
  brew install golangci-lint
  ```

## Getting Started

### 1. Use this Template

Click the **"Use this template"** button on GitHub to create your own repository.

### 2. Initialize the Project

```bash
make init
make up
```

`make init` copies `.env.example` to `.env` and renames the Go module.  
`make up` starts PostgreSQL, Redis, Mailpit, Adminer and the API with hot reload.

### 3. Run Migrations

```bash
make migrate-up
```

### 4. Seed Essential Data

```bash
SEED_SUPERADMIN_EMAIL=admin@example.com \
SEED_SUPERADMIN_PASSWORD=ChangeMe1! \
make seed
```

This creates the `superadmin` and `system` accounts. Safe to run multiple times — existing accounts are skipped.

## Environment Notes

### Application Environment (`APP_ENV`)

Possible values:

- `development`: local development defaults
- `test`: used by automated tests
- `production`: hardened runtime checks and security constraints

Key behavior controlled by `APP_ENV`:

- Swagger UI route is available only in `development` and `test`.
- `Strict-Transport-Security` is enabled only in `production`.
- `DB_SSLMODE=disable` is rejected in `production`.
- `JWT_SECRET` is required in `production`.

### Database TLS (`DB_SSLMODE`)

- Local development and tests usually run with `DB_SSLMODE=disable`.
- Production should use TLS (`DB_SSLMODE=require` at minimum).
- For certificate validation (`verify-ca` / `verify-full`), also provide:
  - `DB_SSLROOTCERT`
  - `DB_SSLCERT`
  - `DB_SSLKEY`

#### Examples

```env
# Local development
DB_SSLMODE=disable
```

```env
# Production (minimum)
DB_SSLMODE=require
```

```env
# Production with server certificate validation
DB_SSLMODE=verify-full
DB_SSLROOTCERT=/etc/ssl/db/ca.pem
DB_SSLCERT=/etc/ssl/db/client-cert.pem
DB_SSLKEY=/etc/ssl/db/client-key.pem
```

### Token security

- Verification and OTP tokens are stored as hashes (raw tokens are never persisted).
- Verification flows track failed attempts and stop after a threshold.
- Password reset endpoints return generic responses to reduce account enumeration risk.

### Frontend Integration (Cookie/Dual Mode)

When `AUTH_TRANSPORT_MODE` is `cookie` or `dual`, frontend clients should:

- Send requests with credentials enabled (`withCredentials: true`).
- Read `csrf_token` cookie and mirror it in `X-CSRF-Token` for state-changing requests.
- Include `X-CSRF-Token` on endpoints such as:
  - `POST /auth/token/refresh`
  - `POST /auth/logout`
  - `PUT /api/v1/users/{id}/profile`
  - `PUT /api/v1/users/{id}/password`
  - `DELETE /api/v1/users/{id}`

### Mail Dispatch Reliability

The API sends authentication emails asynchronously (fire-and-forget from the HTTP handler) but ensures graceful shutdown via `sync.WaitGroup`:

- Handlers use the `sendAsync` helper to dispatch emails without blocking the HTTP response.
- Per-attempt timeout (`MAIL_TIMEOUT_MS`)
- Retries with exponential backoff (`MAIL_RETRY_ATTEMPTS`, `MAIL_RETRY_BASE_DELAY_MS`)
- Feature flag (`MAIL_DISPATCH_ENABLED`) to disable the wrapper if needed

If an email is not received, users can request a new one via the resend endpoints (`POST /auth/verify/resend`, `POST /auth/otp/resend`). Critical information (billing, etc.) is always available directly on the site.

Recommended defaults:

```env
MAIL_DISPATCH_ENABLED=true
MAIL_TIMEOUT_MS=3000
MAIL_RETRY_ATTEMPTS=3
MAIL_RETRY_BASE_DELAY_MS=200
```

## Operations Guide

### Production Checklist

**Configuration**
- [ ] `APP_ENV=production` is set.
- [ ] `JWT_SECRET` is at least 32 characters and managed by a secret manager.
- [ ] `APP_ENCRYPTION_KEY` is exactly 32 bytes and stored securely.
- [ ] `DB_SSLMODE` is not `disable`.
- [ ] If `AUTH_TRANSPORT_MODE` is `cookie` or `dual`, set `AUTH_COOKIE_SECURE=true` and `CORS_ALLOWED_ORIGINS` does not contain `*`.
- [ ] If `AUTH_TRANSPORT_MODE` is `cookie` or `dual`, ensure clients send `X-CSRF-Token` on state-changing requests.
- [ ] DB credentials and TLS material are mounted as K8s Secrets (not committed to git).
- [ ] `AUDIT_PURGE_ENABLED=true` and retention period is set (`AUDIT_PURGE_RETENTION_DAYS`).

**CI / Quality**
- [ ] CI passes on the branch being deployed (tests, vet, lint, Codecov gates).
- [ ] Project coverage ≥ 70%, patch coverage ≥ 60% (enforced by Codecov).

**Infrastructure**
- [ ] Image built from `Dockerfile.prod` — includes `/api`, `/migrate`, `/seed` and `ca-certificates`.
- [ ] K8s init container runs `/migrate up` automatically on each rollout.
- [ ] Seed Job (`seed-job.yaml`) applied once after first deployment.
- [ ] `/livez` and `/readyz` endpoints are monitored by the platform.
- [ ] Logs are centralized and searchable.
- [ ] Backup and restore procedure is tested.

### Deployment Order

**First deployment only:**

1. Build and publish image.
2. Configure K8s Secrets (`api-secret`) with all sensitive values.
3. Apply manifests: `kubectl apply -k deployments/k8s/prod` — the init container runs migrations automatically.
4. Apply seed job: `kubectl apply -f deployments/k8s/base/app/seed-job.yaml`
5. Verify `/livez` and `/readyz`, then basic auth flows.

**Subsequent deployments:**

1. Build and publish new image.
2. Update image tag: `kustomize edit set image your-repo/api=ghcr.io/you/api:SHA`
3. Apply: `kubectl apply -k deployments/k8s/prod` — migrations run automatically via init container.
4. Monitor rollout: `kubectl rollout status deployment/api -n dojo`
5. Check logs and error rates for several minutes.

### Rollback Procedure (Simple)

1. Roll back API to previous image tag.
2. If the last migration is not backward-compatible, run a targeted DB rollback plan.
3. Re-check `/health`, authentication, and token refresh.
4. Keep incident notes (time, scope, root cause, follow-up actions).

### Incident Triage

- Check API logs for `internal error` and request path frequency.
- Check DB connectivity and TLS errors first.
- Check Redis availability for auth/rate-limit related failures.
- Verify recent config/secret changes before code rollback.

## Usage

```bash
make build                              # compile binary to bin/api
make run                                # run locally without Docker
make up                                 # start all containers          (ENV=dev by default)
make up ENV=prod                        # start production containers
make down                               # stop containers               (ENV=dev by default)
make restart                            # restart containers            (ENV=dev by default)
make logs                               # stream API logs               (ENV=dev by default)
make test                               # run tests in the test environment
make coverage                           # run tests with coverage and print summary
make coverage-check                     # fail if total coverage is below COVERAGE_MIN (default: 70)
make lint                               # run go vet and golangci-lint
make check                              # run lint then test
make seed                               # insert superadmin and system accounts
make tidy                               # run go mod tidy and verify

make migrate-up                         # apply pending migrations
make migrate-down                       # rollback the last migration
make migrate-status                     # show migration status
make migrate-reset                      # rollback all migrations
make migrate-create NAME=x              # create a new SQL migration file
```

Coverage notes:

```bash
make coverage                           # writes coverage.out
make coverage-check                     # uses COVERAGE_MIN=70 by default
make coverage-check COVERAGE_MIN=80     # custom threshold
```

`coverage` and `coverage-check` focus on `./internal/...` (application logic) by default.

## Stack

| Service    | URL                                      |
|------------|------------------------------------------|
| API        | http://localhost:8000                    |
| Adminer    | http://localhost:8081                    |
| Mailpit    | http://localhost:8025                    |
| Swagger UI | http://localhost:8000/swagger/index.html |

`Swagger UI` is exposed only when `APP_ENV` is `development` or `test`.

## Debugging (VSCode)

The project includes a **"Debug API"** launch configuration that lets you
run the API locally with breakpoints, while keeping Docker for the backing
services.

**Prerequisites**: [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go) and [Delve](https://github.com/go-delve/delve) installed.

**Steps:**

1. Press `F5` (or `Fn+F5`) — VSCode automatically runs `make debug-up`
   which starts db, cache and mailpit in Docker, then launches the API
   with the debugger attached.

2. Set a breakpoint in any handler (e.g. [internal/handler/auth.go](internal/handler/auth.go)).

3. Send a request to trigger the breakpoint:
   ```bash
   curl -X POST http://localhost:8000/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"Password1"}'
   ```

4. Stop the debugger (`Shift+F5`) — `make debug-down` stops the containers automatically.

> **Note:** `DB_HOST` and `REDIS_ADDR` are automatically overridden to
> `localhost` by the launch configuration, so your `.env` file does not
> need to be modified.

## Database Schema

The schema is defined in [`db/schemas/schema.dbml`](db/schemas/schema.dbml) and exported as [`db/schemas/schema.svg`](db/schemas/schema.svg).

To visualize and edit it directly in VSCode, install the [DBML Previewer](https://marketplace.visualstudio.com/items?itemName=rizkykurniawan.dbml-previewer) extension. It renders the diagram inline and saves table positions in `schema.dbml.layout.json`.

You can also paste the file content on [dbdiagram.io](https://dbdiagram.io) for an online view.

## Documentation

- [Architecture](docs/architecture.md)
- [API Reference](docs/api.md)
- [API Versioning Strategy](docs/api-versioning.md)
- [Roles and Permissions](docs/roles.md)
- [Sensitive field encryption](docs/encryption.md)
- [Transactional emails](docs/emails.md)
- [Login audit logs and purge](docs/audit.md)

## License

**Proprietary and Confidential.** All rights reserved by Nanoninja. This Dojo is intended for internal use only.
