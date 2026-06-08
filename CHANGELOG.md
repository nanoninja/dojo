# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.7.0] - 2026-06-08

### Added

- Stripe payment integration — `payment.Provider` interface, `internal/payment/stripe/` client
- `BuyCourse` / `BuyBundle` — purchase `pending` + Stripe checkout session, enrollment créé uniquement après confirmation webhook
- `POST /webhooks/stripe` — route publique, validation signature HMAC, gestion `EventPaymentSucceeded` et `EventPaymentFailed`
- `ConfirmPayment` — webhook `checkout.session.completed` → purchase `completed` + enrollment créé atomiquement via `WithTx`
- `CancelPending` — webhook `checkout.session.expired` → purchase `failed`
- `Refund` complet — appel `provider.Refund` (Stripe) avant la transaction DB ; webhook `charge.refunded` (`EventRefundSucceeded`) gère les remboursements initiés depuis le dashboard Stripe
- `payment.ErrInvalidSignature` — erreur normalisée pour les signatures webhook invalides
- `ErrPurchaseAlreadyProcessed` mappé → 409 Conflict dans `toFault`
- Ownership check sur `GET /purchases/{id}` — retourne 404 si l'utilisateur n'est pas le propriétaire
- Rate limiting par userID sur les mutations `BuyCourse`, `BuyBundle`, `Refund`, `Enroll` (fallback IP si non authentifié)
- Tests handler `internal/handler/webhook_test.go`
- Migration `db/migrations/003_payment.sql` — colonnes `provider`, `provider_session_id`, `provider_payment_id` sur `purchases`

---

## [0.6.0] - 2026-06-03

### Added

- `AccessService.CanAccess(ctx, userID, courseID)` — grants access when user has an active subscription or an active enrollment on the course; returns `fault.Forbidden` otherwise
- Access guard on `GET /chapters/{id}`, `GET /chapters/{chapter_id}/lessons`, `GET /lessons/{id}`, `GET /lessons/{id}/resources`
- `EnrollmentStatusCancelled` — new enum value in SQL migration, Go model, and DBML schema
- `middleware.RequireUserID` — centralized helper replacing inline `userID == ""` checks across all handlers

### Changed

- All instructor mutation handlers (bundle, course, chapter, lesson) migrated to `RequireUserID`
- Subscription and certificate handlers migrated to `RequireUserID`
- Swagger `@Failure 401` annotations added on subscription endpoints

---

## [0.5.0] - 2026-05-26

### Added

- Ownership checks on instructor mutations — `OwnershipChecker` interface with per-domain constructors (course, chapter, lesson, bundle); returns 404 on mismatch to avoid resource enumeration
- Rate limiting on catalog list endpoints — 60 req/min per IP on `GET /courses` and `GET /bundles` using existing Redis backend (`httprate-redis`)
- Swagger `@Failure 429` annotation on `Course.List` and `Bundle.List`

---

## [0.4.0] - 2026-05-22

### Added

- Business layer: migration `002_business_schema.sql` (subscriptions, purchases, purchase_id on enrollments)
- Subscriptions: store, service, handler, routes and tests
- Purchases: store, service, handler, routes and tests — course and bundle purchases with atomic enrollment creation via `WithTx`
- Refund: marks purchase as refunded and cancels associated enrollments atomically
- `EnrollmentStore.CancelByPurchase` — bulk status update by purchase ID
- `purchase_id` nullable FK on `course_enrollments` linking billing to access

---

## [0.3.0] - 2026-05-14

### Added

- Course enrollments: migration, model, store, service, handler and routes
- Store integration tests for tag, category, chapter, lesson, lesson resources, course, courses_categories, courses_tags

### Changed

- `httputil.ValidateUUID` now returns `bool` instead of `error`; error messages are now contextualized at each call site

---

## [0.1.0] - 2026-05-11

### Added

- Project scaffolding: chi router, pgx/v5, sqlx, PostgreSQL 18, Redis
- JWT authentication with cookie, header and dual transport modes
- CSRF protection, rate limiting, Prometheus metrics, Swagger UI
- Role-based access control: user, instructor, moderator, manager, admin, superadmin
- User management: registration, email verification, OTP, password reset, login history
- AES encryption for sensitive fields (email, IP, birth date, VAT number)
- Courses: CRUD, category and tag assignments
- Categories and tags: CRUD (admin only)
- Chapters: CRUD per course
- Lessons: CRUD per chapter, with attachable resources
- Docker Compose environments for development, test and production
- Kubernetes manifests with Kustomize overlays
- Grafana dashboard and Prometheus scrape config
- Database migrations and seed command
- CI pipeline with GitHub Actions

---
