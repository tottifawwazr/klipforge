COMPOSE := docker compose

.PHONY: help dev infra-up infra-down up down ps logs build test lint format clean migrate-up migrate-down seed

help:
	@echo "KlipForge Phase 1 commands:"
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

migrate-up migrate-down seed:
	@echo "This target becomes available in Phase 2; no migrations or seed data exist in Phase 1."
	@exit 1
