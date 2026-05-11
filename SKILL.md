# Dojo — Project Knowledge Base

## Overview

Application de formation en ligne (type Udemy/Coursera) construite en Go.
Issu d'un boilerplate personnel (`github.com/nanoninja/boilerplate`).

## Stack

| Composant     | Technologie                        |
|---------------|------------------------------------|
| Language      | Go 1.26                            |
| Router        | go-chi/chi v5                      |
| Base de données | PostgreSQL 18 + pgx/v5 + sqlx    |
| Migrations    | Goose v3 (`db/migrations/`)        |
| Cache         | Redis 8 + go-redis/v9              |
| Auth          | JWT (golang-jwt/jwt v5)            |
| Validation    | go-playground/validator v10        |
| Docs API      | Swagger (swaggo/swag)              |
| Métriques     | Prometheus + Grafana               |
| Mail          | SMTP + Mailpit (dev)               |
| Dev live reload | Air (`air.toml`)                 |

## Architecture interne (`internal/`)

```
config/      — configuration (env vars)
fault/       — erreurs métier typées
handler/     — handlers HTTP (chi)
httputil/    — helpers request/response
middleware/  — auth, CSRF, body, security, metrics
model/       — structs domaine (User, Course, etc.)
platform/    — adapters infrastructure (database, cache, mailer)
service/     — logique métier
store/       — accès base de données (SQL)
testutil/    — helpers de test (db, mocks)
```

## Commandes principales

```bash
make up               # démarre l'env dev (Air + Docker)
make down             # arrête les containers
make migrate-up       # applique les migrations en attente
make migrate-down     # rollback dernière migration
make seed             # insère superadmin + compte système
make test             # lance les tests (container db-test dédié)
make coverage         # tests avec couverture
make lint             # go vet + golangci-lint
make swagger          # génère la doc Swagger
```

## Docker

- Base : `deployments/docker/compose.prod.yaml` (db, cache, prometheus, grafana)
- Overlay dev : `deployments/docker/compose.dev.yaml` (Air, Mailpit, Adminer, ports exposés)
- Les volumes sont préfixés avec `${APP_NAME}` pour éviter les conflits entre projets
- Healthcheck PostgreSQL : `pg_isready -U ${DB_USER} -d ${DB_NAME}`

## Base de données

- Extension UUID : `pg_uuidv7` — fonction `uuidv7()`
- Soft delete : colonne `deleted_at TIMESTAMPTZ` (NULL = actif)
- `updated_at` géré par trigger `update_updated_at_column()`
- Schéma visuel : `db/schemas/schema.dbml` (visualisable sur dbdiagram.io)

## Domaine métier

### Hiérarchie du contenu
```
courses
  └── course_chapters
        └── lessons
              └── lesson_resources
```

### Relations catalogue
- `courses` ↔ `course_categories` via `course_category_assignments` (many-to-many, `is_primary` pour la catégorie principale)
- `courses` ↔ `tags` via `course_tag_assignments` (many-to-many)

### Modèle de prix
- `price_cents INTEGER` + `currency CHAR(3)` — pas de float pour éviter les erreurs d'arrondi
- `is_free BOOLEAN` — accès gratuit
- `subscription_only BOOLEAN` — réservé aux abonnés

## Tables à venir (non encore créées)
- `enrollments` — inscriptions des apprenants
- `reviews` — avis et notes (source pour `rating_average` / `rating_count`)
- `coupons` / `discounts` — codes promotionnels
