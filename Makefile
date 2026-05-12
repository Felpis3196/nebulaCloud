# ============================================================================
# NebulaCloud — Makefile
# Convenience targets for local development of the platform.
# ============================================================================

SHELL := /bin/sh
.DEFAULT_GOAL := help

COMPOSE       ?= docker compose
COMPOSE_DEV   := $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
GO            ?= go
GOFLAGS       ?=
PKG           := ./...
COVER_FILE    := backend/coverage.out

# ----------------------------------------------------------------------------
# Help
# ----------------------------------------------------------------------------
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nNebulaCloud — make targets\n\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_.\/-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Bootstrap
.PHONY: env
env: ## Copy .env.example to .env if missing
	@test -f .env || cp .env.example .env && echo ".env ready"

.PHONY: hosts
hosts: ## Print /etc/hosts entries needed for local dev
	@echo "Add the following entries to your hosts file:"
	@echo "127.0.0.1  api.nebula.localhost app.nebula.localhost grafana.nebula.localhost traefik.nebula.localhost registry.nebula.localhost"

##@ Stack lifecycle
.PHONY: up
up: env ## Start the full stack (detached)
	$(COMPOSE_DEV) up -d

.PHONY: up-fg
up-fg: env ## Start the full stack (foreground)
	$(COMPOSE_DEV) up

.PHONY: down
down: ## Stop the stack and remove containers
	$(COMPOSE_DEV) down

.PHONY: nuke
nuke: ## Stop the stack AND drop all volumes (DESTRUCTIVE)
	$(COMPOSE_DEV) down -v

.PHONY: ps
ps: ## Show stack status
	$(COMPOSE_DEV) ps

.PHONY: logs
logs: ## Tail logs (set SVC=name to filter)
	$(COMPOSE_DEV) logs -f $(SVC)

.PHONY: restart
restart: ## Restart a service (SVC=api)
	$(COMPOSE_DEV) restart $(SVC)

##@ Backend (Go)
.PHONY: tidy
tidy: ## go mod tidy
	cd backend && $(GO) mod tidy

.PHONY: build
build: ## Build all backend binaries into backend/bin/
	cd backend && mkdir -p bin && $(GO) build $(GOFLAGS) -o bin/api ./cmd/api && \
	$(GO) build $(GOFLAGS) -o bin/build-worker ./cmd/build-worker && \
	$(GO) build $(GOFLAGS) -o bin/runtime-agent ./cmd/runtime-agent

.PHONY: run-api
run-api: ## Run the API locally (uses .env)
	cd backend && $(GO) run ./cmd/api

.PHONY: test
test: ## Run unit tests
	cd backend && $(GO) test -race -count=1 $(PKG)

.PHONY: test-cover
test-cover: ## Run tests with coverage
	cd backend && $(GO) test -race -count=1 -coverprofile=$(notdir $(COVER_FILE)) $(PKG) && \
	$(GO) tool cover -func=$(notdir $(COVER_FILE)) | tail -n 1

.PHONY: lint
lint: ## Run golangci-lint (must be installed)
	cd backend && golangci-lint run ./...

.PHONY: fmt
fmt: ## Format Go code
	cd backend && $(GO) fmt ./... && goimports -w .

##@ Database
.PHONY: psql
psql: ## Open psql against the dev DB
	$(COMPOSE_DEV) exec postgres psql -U nebula -d nebula

.PHONY: migrate-up
migrate-up: ## Apply pending migrations (uses backend tool)
	$(COMPOSE_DEV) exec api /app/api migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	$(COMPOSE_DEV) exec api /app/api migrate down

##@ Misc
.PHONY: openapi
openapi: ## Print the OpenAPI spec path
	@echo "backend/api/openapi.yaml"

.PHONY: clean
clean: ## Remove backend build artifacts
	rm -rf backend/bin backend/dist backend/coverage.out backend/coverage.html
