# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# --- Configuration ---
VERSION     := $(shell git describe --tags --always --dirty)
BINARY_NAME  = api
MODULE_NAME  = $(shell head -n 1 go.mod | cut -d ' ' -f 2)
GOBIN       := $(shell go env GOPATH)/bin
export PATH := $(PATH):$(GOBIN)

# ENV controls which compose overlay is loaded (dev by default).
# Override at the command line: make up ENV=prod
ENV         ?= dev

DOCKER_DIR     = deployments/docker
K8S_DIR        = deployments/k8s
COVERAGE_FILE  ?= coverage.out
COVERAGE_MIN   ?= 70

# compose.prod.yaml is always the base layer (shared services: db, cache).
# compose.$(ENV).yaml is the overlay that adds or overrides env-specific services.
# e.g. compose.dev.yaml adds Air, Mailpit, Adminer and exposes port 5432.
COMPOSE_BASE   = -f $(DOCKER_DIR)/compose.prod.yaml
COMPOSE_ENV    = -f $(DOCKER_DIR)/compose.$(ENV).yaml
DOCKER_COMPOSE = docker compose $(COMPOSE_BASE) $(COMPOSE_ENV)

.PHONY: help init build run up down restart logs debug-up debug-down test coverage coverage-check lint check seed tidy \
        migrate-up migrate-down migrate-status migrate-reset migrate-create \
        k8s-apply k8s-delete \
        swagger

help:
	@printf "A Go REST API.\n"
	@printf "\n"
	@printf "Usage:\n"
	@printf "\n"
	@printf "        make <command> [ENV=dev|prod]\n"
	@printf "\n"
	@printf "The commands are:\n"
	@printf "\n"
	@printf "        init                       copy .env and rename Go module\n"
	@printf "        build                      compile the binary to bin/api\n"
	@printf "        run                        run the API locally (without Docker)\n"
	@printf "        debug-up                   start db, cache and mailpit for local debugging\n"
	@printf "        debug-down                 stop debug services\n"
	@printf "        up                         start containers          (default ENV=dev)\n"
	@printf "        down                       stop and remove containers (default ENV=dev)\n"
	@printf "        restart                    restart containers         (default ENV=dev)\n"
	@printf "        logs                       stream API logs            (default ENV=dev)\n"
	@printf "        test                       run tests in the test environment\n"
	@printf "        coverage                   run tests with coverage and print summary\n"
	@printf "        coverage-check             fail if total coverage is below COVERAGE_MIN (default: 70)\n"
	@printf "        lint                       run go vet and golangci-lint\n"
	@printf "        check                      run lint then test\n"
	@printf "        seed                       insert superadmin and system accounts\n"
	@printf "        tidy                       run go mod tidy and verify\n"
	@printf "        swagger                    generate swagger docs\n"
	@printf "\n"
	@printf "        migrate-up                 run all pending migrations\n"
	@printf "        migrate-down               rollback the last migration\n"
	@printf "        migrate-status             show migration status\n"
	@printf "        migrate-reset              rollback all migrations\n"
	@printf "        migrate-create NAME=x      create a new SQL migration file\n"
	@printf "\n"
	@printf "        k8s-apply                  apply Kubernetes manifests (default ENV=dev)\n"
	@printf "        k8s-delete                 delete Kubernetes resources (default ENV=dev)\n"
	@printf "\n"

# ==============================================================================
# Project Initialization
# ==============================================================================

init:
	@go run ./cmd/init && rm -rf ./cmd/init
	@printf "Initialization complete. You can now run 'make up'.\n"

# ==============================================================================
# Local Development
# ==============================================================================

build: swagger
	go build -ldflags="-X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/api

run:
	DB_HOST=localhost REDIS_ADDR=localhost:6379 go run ./cmd/api

# debug-up starts only the backing services (db, cache, mailpit) without the api
# container, freeing port 8000 for the VSCode debugger to run the API locally.
# Make sure DB_HOST=localhost is set in your .env before attaching the debugger.
debug-up:
	$(DOCKER_COMPOSE) up -d db cache mailpit

debug-down:
	$(DOCKER_COMPOSE) down

# ==============================================================================
# Docker Orchestration
#
# All targets below accept an optional ENV variable (default: dev).
# Examples:
#   make up              → starts dev environment (Air, Mailpit, Adminer, DB on :5432)
#   make up ENV=prod     → starts production environment
#   make logs ENV=prod   → stream prod API logs
# ==============================================================================

up:
	$(DOCKER_COMPOSE) up -d

down:
	$(DOCKER_COMPOSE) down

restart:
	$(DOCKER_COMPOSE) restart

logs:
	$(DOCKER_COMPOSE) logs -f api

# ==============================================================================
# Database Migrations (Goose)
# ==============================================================================

migrate-up:
	$(DOCKER_COMPOSE) exec api go run cmd/migrate/main.go up

migrate-down:
	$(DOCKER_COMPOSE) exec api go run cmd/migrate/main.go down

migrate-status:
	$(DOCKER_COMPOSE) exec api go run cmd/migrate/main.go status

migrate-reset:
	$(DOCKER_COMPOSE) exec api go run cmd/migrate/main.go reset

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		printf "Error: NAME variable is required (e.g., make migrate-create NAME=add_users_table)\n"; \
		exit 1; \
	fi
	@go run github.com/pressly/goose/v3/cmd/goose@latest -dir db/migrations create $(NAME) sql

# ==============================================================================
# Kubernetes
# ==============================================================================

k8s-apply:
	kubectl apply -f $(K8S_DIR)/$(ENV)/

k8s-delete:
	kubectl delete -f $(K8S_DIR)/$(ENV)/

# ==============================================================================
# Development Tools
# ==============================================================================

swagger:
	$(GOBIN)/swag init -g main.go -d ./cmd/api,./internal/handler,./internal/model,./internal/service,./internal/fault --output docs/swagger --parseInternal

test:
	docker compose -f $(DOCKER_DIR)/compose.test.yaml up -d --wait db-test
	go test -count=1 -v ./...
	docker compose -f $(DOCKER_DIR)/compose.test.yaml down

coverage:
	@set -e; \
	docker compose -f $(DOCKER_DIR)/compose.test.yaml up -d --wait db-test; \
	trap 'docker compose -f $(DOCKER_DIR)/compose.test.yaml down' EXIT; \
	go test -count=1 -v -coverpkg=./internal/... -coverprofile=$(COVERAGE_FILE) ./internal/...; \
	go tool cover -func=$(COVERAGE_FILE)

coverage-check: coverage
	@total=$$(go tool cover -func=$(COVERAGE_FILE) | awk '/^total:/ { gsub("%", "", $$3); print $$3 }'); \
	printf "Total coverage: %s%% (minimum: %s%%)\n" "$$total" "$(COVERAGE_MIN)"; \
	awk "BEGIN { exit !($$total >= $(COVERAGE_MIN)) }" || (printf "Coverage check failed.\n"; exit 1)

lint:
	go vet ./...
	golangci-lint run ./...

check: lint test

seed:
	$(DOCKER_COMPOSE) exec api go run ./cmd/seed

tidy:
	go mod tidy
	go mod verify
