# Predict-A-Trade v1.0.0 — Root Makefile
# Canonical build/lint/test commands for all four planes.

.PHONY: help all build test lint format clean infra-up infra-down \
        db-migrate db-seed \
        go-build go-test go-lint go-format \
        control-build control-test control-lint control-format \
        frontend-build frontend-test frontend-lint frontend-format \
        research-test research-lint \
        e2e-test security-scan

SHELL := /bin/bash
.DEFAULT_GOAL := help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ============================================================
# Infrastructure (local/test)
# ============================================================

infra-up: ## Start local Docker infrastructure (Postgres, Valkey, Prometheus, Grafana)
	docker compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@timeout 60 bash -c 'until docker exec pat-postgres pg_isready -U pat_admin 2>/dev/null; do sleep 2; done'
	@echo "Infrastructure is up."

infra-down: ## Stop local Docker infrastructure
	docker compose down

infra-clean: ## Stop and remove volumes (DESTRUCTIVE — local only)
	docker compose down -v

# ============================================================
# Database
# ============================================================

db-migrate: ## Run canonical forward migrations
	./scripts/migrate.sh up

db-rollback: ## Rollback last migration
	./scripts/migrate.sh down

db-seed: ## Seed initial configuration (plans, strategies, commission rules)
	./scripts/migrate.sh seed

db-test: ## Run migration tests
	./scripts/migrate.sh test

# ============================================================
# Go — Real-Time Trading Plane
# ============================================================

GO_DIR := realtime
GO_LDFLAGS := -s -w
ENGINE_VERSION ?= 1.18.0

go-build: ## Build the Go real-time engine
	cd $(GO_DIR) && go build -ldflags "$(GO_LDFLAGS) -X github.com/predictatrade/realtime/internal/version.Version=$(ENGINE_VERSION)" -o bin/realtime-engine ./cmd/realtime-engine

go-test: ## Run Go unit tests
	cd $(GO_DIR) && go test -race -count=1 -timeout=120s ./...

go-test-integration: ## Run Go integration tests (requires infra-up)
	cd $(GO_DIR) && go test -race -count=1 -timeout=300s -tags=integration ./...

go-lint: ## Lint Go code
	cd $(GO_DIR) && go vet ./... 2>&1
	@which golangci-lint >/dev/null 2>&1 && cd $(GO_DIR) && golangci-lint run ./... || echo "golangci-lint not installed, go vet only"

go-format: ## Format Go code
	cd $(GO_DIR) && gofmt -s -w .
	cd windows-agent && gofmt -s -w .

go-benchmark: ## Run Go benchmarks
	cd $(GO_DIR) && go test -bench=. -benchmem ./...

# ============================================================
# NestJS — Control Plane
# ============================================================

control-install: ## Install NestJS dependencies
	cd control && npm ci

control-build: ## Build NestJS control plane
	cd control && npm run build

control-test: ## Run NestJS unit tests
	cd control && npm test

control-test-e2e: ## Run NestJS E2E tests
	cd control && npm run test:e2e

control-lint: ## Lint NestJS code
	cd control && npm run lint

control-format: ## Format NestJS code
	cd control && npm run format

control-start: ## Start NestJS in development mode
	cd control && npm run start:dev

# ============================================================
# Next.js — Presentation Plane
# ============================================================

frontend-install: ## Install Next.js dependencies
	cd frontend && npm ci

frontend-build: ## Build Next.js frontend
	cd frontend && npm run build

frontend-test: ## Run Next.js tests
	cd frontend && npm test

frontend-lint: ## Lint Next.js code
	cd frontend && npm run lint

frontend-format: ## Format Next.js code
	cd frontend && npm run format

frontend-dev: ## Start Next.js dev server
	cd frontend && npm run dev

# ============================================================
# Python — Research Plane
# ============================================================

research-install: ## Install Python research dependencies
	cd research && pip install -e .[dev] || pip install -e .

research-test: ## Run Python research tests
	cd research && python -m pytest tests/ -v --tb=short

research-lint: ## Lint Python code
	cd research && python -m ruff check src/ tests/ 2>/dev/null || echo "ruff not installed"

# ============================================================
# Windows Agent
# ============================================================

agent-build: ## Build Windows Agent (cross-compile both roles for windows/amd64)
	cd windows-agent && GOOS=windows GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o bin/pat-agent-client.exe ./cmd/client
	cd windows-agent && GOOS=windows GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o bin/pat-agent-master.exe ./cmd/master

agent-test: ## Test Windows Agent
	cd windows-agent && go test -race -count=1 ./...

# ============================================================
# Aggregated targets
# ============================================================

all: build test lint

build: go-build control-build frontend-build ## Build all planes

test: go-test control-test frontend-test research-test agent-test ## Run all unit tests

lint: go-lint control-lint frontend-lint research-lint ## Lint all planes

format: go-format control-format frontend-format ## Format all planes

e2e-test: control-test-e2e frontend-test ## Run E2E tests

security-scan: ## Run security scans (secrets, deps, SAST)
	./scripts/security-scan.sh

clean: ## Clean build artifacts
	rm -rf realtime/bin control/dist frontend/.next frontend/out windows-agent/bin
	cd research && rm -rf build dist *.egg-info
	find . -name '__pycache__' -type d -exec rm -rf {} + 2>/dev/null || true

.PHONY: help all build test lint format clean
