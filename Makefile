.PHONY: dev prod down logs build clean setup migrate-up migrate-down migrate-create scaffold test test-integration test-coverage swagger lint install-hooks

-include .env
export

setup:
	cp --update=none .env.example .env || true

install-hooks:
	@bash scripts/install-hooks.sh

dev: setup
	docker compose -f docker-compose.dev.yml up --build

dev-d: setup
	docker compose -f docker-compose.dev.yml up --build -d

prod: setup
	docker compose -f docker-compose.prod.yml up --build -d

down-dev:
	docker compose -f docker-compose.dev.yml down

down-prod:
	docker compose -f docker-compose.prod.yml down

down-dev-v:
	docker compose -f docker-compose.dev.yml down -v

down-prod-v:
	docker compose -f docker-compose.prod.yml down -v

logs-dev:
	docker compose -f docker-compose.dev.yml logs -f api

logs-prod:
	docker compose -f docker-compose.prod.yml logs -f api

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v && go clean -testcache

test-integration:
	TEST_DATABASE_URL="postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" \
	go test -tags integration ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run ./...

swagger:
	swag init -g cmd/server/main.go -o docs

clean:
	rm -rf bin/ tmp/

# ── Migrations ────────────────────────────────────────────────────────────────
# Requires the `migrate` CLI: https://github.com/golang-migrate/migrate/tree/master/cmd/migrate
# DB_HOST is overridden to localhost so the CLI can reach the Docker container from the host.

_DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

migrate-up:
	migrate -path migrations -database "$(_DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(_DB_URL)" down 1

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(NAME)

scaffold:
	@if [ -z "$(NAME)" ]; then echo "Usage: make scaffold NAME=post"; exit 1; fi
	@bash scripts/scaffold.sh $(NAME)
