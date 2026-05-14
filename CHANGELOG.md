# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

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
