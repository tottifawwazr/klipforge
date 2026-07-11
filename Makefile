COMPOSE := docker compose

.PHONY: help dev infra-up infra-down up down ps logs build test lint format clean migrate-up migrate-down migrate-status seed db-reset

help:
	@echo "KlipForge development commands:"
	@echo "  make dev         Build and run the complete Phase 1 stack"
	@echo "  make infra-up    Start PostgreSQL and Redis in the background"
	@echo "  make infra-down  Stop PostgreSQL and Redis"
	@echo "  make up          Build and run the Phase 1 stack in the background"
	@echo "  make down        Stop the Phase 1 stack (data volumes are preserved)"
	@echo "  make ps          Show service status"
	@echo "  make logs        Follow stack logs"
	@echo "  make build       Build API and web images"
	@echo "  make test        Run current backend and frontend tests"
	@echo "  make lint        Run current backend and frontend linters"
	@echo "  make format      Format current backend and frontend sources"
	@echo "  make clean       Stop containers and remove generated containers/networks"
	@echo "  make migrate-up  Apply all pending PostgreSQL migrations"
	@echo "  make migrate-down Roll back the most recently applied migration"
	@echo "  make migrate-status Show applied and pending migrations"
	@echo "  make seed        Insert deterministic local development seed data"
	@echo "  make db-reset    Remove local Compose volumes, migrate, and seed"

dev:
	$(COMPOSE) up --build

infra-up:
	$(COMPOSE) up -d postgres redis

infra-down:
	$(COMPOSE) stop postgres redis

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down --remove-orphans

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs --follow

build:
	$(COMPOSE) build api web

test:
	cd services/api && go test ./...
	cd apps/web && npm run test --if-present

lint:
	cd services/api && go vet ./...
	cd apps/web && npm run lint

format:
	cd services/api && gofmt -w .
	cd apps/web && npm run format --if-present

clean:
	$(COMPOSE) down --remove-orphans

migrate-up:
	$(COMPOSE) up -d postgres
	$(COMPOSE) --profile tools run --rm migrate up

migrate-down:
	$(COMPOSE) up -d postgres
	$(COMPOSE) --profile tools run --rm migrate down

migrate-status:
	$(COMPOSE) up -d postgres
	$(COMPOSE) --profile tools run --rm migrate status

seed:
	$(COMPOSE) up -d postgres
	$(COMPOSE) --profile tools run --rm seed

db-reset:
	$(COMPOSE) down --remove-orphans --volumes
	$(COMPOSE) up -d postgres redis
	$(COMPOSE) --profile tools run --rm migrate up
	$(COMPOSE) --profile tools run --rm seed
