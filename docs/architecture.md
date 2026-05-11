# Architecture

## Project Structure

```
.
├── cmd/
│   ├── api/            # Application entrypoint (main.go, purge.go)
│   ├── migrate/        # Migration runner (main.go)
│   └── seed/           # Database seeder — superadmin and system accounts (main.go)
├── db/
│   └── migrations/     # Goose SQL migration files
├── deployments/
│   └── docker/         # Dockerfiles and Compose files per environment
├── docs/               # Project documentation
├── internal/
│   ├── config/         # Environment variable loading
│   ├── fault/          # Typed HTTP errors (Fault)
│   ├── handler/        # HTTP handlers — decode request, call service, encode response
│   ├── httputil/       # HTTP helpers: Bind, Send, Handle, Error
│   ├── middleware/     # HTTP middleware: Authenticate, RequireRole, SecureHeaders
│   ├── model/          # Domain types: User, Role, VerificationToken, RefreshToken, LoginAuditLog
│   ├── platform/       # Infrastructure adapters: database, cache, mailer, security
│   ├── router/         # Route wiring and middleware chain
│   ├── service/        # Business logic — orchestrates stores and sends emails
│   ├── store/          # Database queries — one file per aggregate
│   └── testutil/       # Shared test helpers: DB setup, migrations, cipher
└── .vscode/            # VSCode launch and task configurations
```

## Layered Architecture

Requests flow through four distinct layers, each with a single responsibility:

```
HTTP Request
    ↓
[ Middleware ]      — auth, rate limiting, security headers
    ↓
[ Handler ]         — decode input, validate, call service
    ↓
[ Service ]         — business logic, orchestration
    ↓
[ Store ]           — SQL queries, data access
    ↓
[ Database / Cache ]
```

### Handler
Handles the HTTP contract: decodes the request body (`httputil.Bind`), calls the appropriate service method, and writes the response (`httputil.OK`, `httputil.Send`). Handlers do not contain business logic.

### Service
Contains all business rules: password hashing, token generation, email dispatch, validation logic. Services depend on store interfaces, not concrete implementations — making them easy to unit test with fakes.

### Store
Executes SQL queries against the database. One file per aggregate (e.g. `user.go`, `auth.go`). Stores return domain models (`model.User`, etc.) and never leak SQL types to upper layers.

### Fault
Typed errors that carry an HTTP status code and a client-safe message. Handlers convert service errors to faults via `toFault()`. The `httputil.Error` function serializes them as JSON.

```go
// Defining an error
fault.NotFound("user", err)        // → 404 { "error": "user not found" }
fault.Unauthorized(err)            // → 401 { "error": "Unauthorized" }

// Returning from a handler
return toFault(service.ErrUserNotFound)
```

## Deployment Order

```bash
make migrate-up                          # 1. apply schema
SEED_SUPERADMIN_EMAIL=admin@example.com \
SEED_SUPERADMIN_PASSWORD=ChangeMe1! \
make seed                                # 2. insert essential accounts
make up ENV=prod                         # 3. start the application
```

The seed command is idempotent — existing accounts are skipped on subsequent runs.

## Adding a New Feature

Typical workflow for adding a new endpoint (e.g. `GET /api/v1/posts`):

1. **Model** — add the domain type in `internal/model/`
2. **Migration** — create a new SQL file with `make migrate-create NAME=add_posts`
3. **Store** — add the interface and implementation in `internal/store/`
4. **Service** — add the interface and implementation in `internal/service/`
5. **Handler** — add the handler in `internal/handler/`
6. **Router** — wire the route in `internal/router/router.go`
7. **Tests** — unit tests for service (with fakes), handler tests, integration tests for store

## Error Handling

Every handler is wrapped with `httputil.Handle` which catches returned errors and writes the appropriate JSON response. Handlers never call `http.Error` directly.

```go
r.Get("/example", httputil.Handle(func(w http.ResponseWriter, r *http.Request) error {
    // return an error → httputil.Handle writes the JSON response
    return fault.NotFound("resource", nil)
}, logger))
```

## Authentication Flow

```
POST /auth/login
  → validates credentials
  → returns access token (JWT) + refresh token

Authorization: Bearer <access_token>
  → Authenticate middleware validates JWT
  → injects userID + role into context

POST /auth/token/refresh
  → validates refresh token (hash stored in DB)
  → revokes old token, issues new pair (rotation)
```

The role is embedded in the JWT at login. See [roles.md](roles.md) for the full RBAC model.
