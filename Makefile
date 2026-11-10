# Ayaka developer workflow. Run `make help` to list targets.
# On Windows use Git Bash / WSL (or install make via `choco install make`).

APP      := ayaka
BIN      := bin/$(APP)
PKG      := ./...
SWAG     := github.com/swaggo/swag/cmd/swag@v1.16.4
AIR      := github.com/air-verse/air@latest
LINT     := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
VULN     := golang.org/x/vuln/cmd/govulncheck@latest

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ---------- run

.PHONY: run
run: ## Run the server
	go run . serve

.PHONY: dev
dev: ## Run with hot reload (air)
	go run $(AIR)

.PHONY: build
build: ## Build the binary into ./bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) .

## ---------- quality

.PHONY: fmt
fmt: ## Format code
	gofmt -w .
	go mod tidy

.PHONY: lint
lint: ## Run golangci-lint
	go run $(LINT) run

.PHONY: vuln
vuln: ## Scan dependencies for known vulnerabilities
	go run $(VULN) $(PKG)

.PHONY: test
test: ## Run unit tests
	go test -count=1 $(PKG)

.PHONY: test-race
test-race: ## Run unit tests with the race detector (requires CGO)
	CGO_ENABLED=1 go test -race -count=1 $(PKG)

.PHONY: cover
cover: ## Run tests with coverage report (coverage.html)
	go test -count=1 -covermode=atomic -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html

.PHONY: check
check: fmt lint test ## Everything CI runs (run before pushing)

## ---------- code generation

.PHONY: swag
swag: ## Regenerate Swagger docs (docs/)
	go run $(SWAG) fmt
	go run $(SWAG) init -g main.go -o docs --parseInternal --outputTypes go,yaml

## ---------- database

.PHONY: migrate-up
migrate-up: ## Apply pending migrations
	go run . migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	go run . migrate down

.PHONY: migrate-status
migrate-status: ## Show migration status
	go run . migrate status

.PHONY: migrate-create
migrate-create: ## Create a migration: make migrate-create name=add_posts_table
	@test -n "$(name)" || (echo "usage: make migrate-create name=<name>" && exit 1)
	go run . migrate create $(name)

## ---------- administration

.PHONY: create-admin
create-admin: ## Create an admin: ADMIN_PASSWORD=... make create-admin email=a@b.c username=admin
	@test -n "$(email)" -a -n "$(username)" || (echo "usage: ADMIN_PASSWORD=... make create-admin email=<email> username=<username>" && exit 1)
	go run . user create-admin --email $(email) --username $(username)

.PHONY: tokens-prune
tokens-prune: ## Delete stale refresh / one-time tokens
	go run . tokens prune

## ---------- docker

.PHONY: up
up: ## Start app + postgres with docker compose
	docker compose up --build -d

.PHONY: down
down: ## Stop docker compose services
	docker compose down

.PHONY: logs
logs: ## Tail app logs
	docker compose logs -f app
