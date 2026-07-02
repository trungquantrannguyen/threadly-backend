# ============================================================
# Threadly Backend Makefile
# ============================================================

SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -c

# Load local env by default for local Go commands and migrations.
# You can override with: make run-user ENV_FILE=.env.docker
ENV_FILE ?= .env.local
-include $(ENV_FILE)

# Docker Compose env file
DOCKER_ENV_FILE ?= .env.docker
DOCKER_COMPOSE := docker compose --env-file $(DOCKER_ENV_FILE)

# Database migrations
MIGRATIONS_PATH ?= ./db/migrations
DB_URL ?= $(DATABASE_URL)

# Go test/cache/coverage
GOCACHE ?= $(CURDIR)/tmp/go-build
COVERAGE_DIR ?= coverage
COVERAGE_PROFILE ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_THRESHOLD ?= 80

# Test packages
TEST_PACKAGES := $(shell go list ./... \
	| grep -v '/proto/' \
	| grep -v '/docs' \
	| grep -v '/cmd/' \
	| grep -v '/provider')

API_GATEWAY_PACKAGES := $(shell go list ./services/api-gateway/internal/...)
USER_PACKAGES := $(shell go list ./services/user-service/internal/...)
CONTENT_PACKAGES := $(shell go list ./services/content-service/internal/...)
FEED_PACKAGES := $(shell go list ./services/feed-service/internal/...)
NOTIFICATION_PACKAGES := $(shell go list ./services/notification-service/internal/...)
STORAGE_PACKAGES := $(shell go list ./services/storage-service/internal/... | grep -v '/provider')

# ============================================================
# Help
# ============================================================

.PHONY: help
help:
	@echo "Threadly Backend commands:"
	@echo ""
	@echo "Local run:"
	@echo "  make run-gateway"
	@echo "  make run-user"
	@echo "  make run-content"
	@echo "  make run-feed"
	@echo "  make run-storage"
	@echo "  make run-notification"
	@echo ""
	@echo "Air dev mode:"
	@echo "  make dev-gateway"
	@echo "  make dev-user"
	@echo "  make dev-content"
	@echo "  make dev-feed"
	@echo "  make dev-storage"
	@echo "  make dev-notification"
	@echo "  make dev-all"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-config"
	@echo "  make docker-build"
	@echo "  make docker-up"
	@echo "  make docker-down"
	@echo "  make docker-logs"
	@echo "  make docker-ps"
	@echo "  make docker-clean"
	@echo ""
	@echo "Database migrations:"
	@echo "  make migrate-create name=create_users_table"
	@echo "  make migrate-up"
	@echo "  make migrate-down"
	@echo "  make migrate-version"
	@echo "  make migrate-force version=1"
	@echo "  make migrate-drop"
	@echo ""
	@echo "Testing:"
	@echo "  make test"
	@echo "  make test-coverage"
	@echo "  make coverage-html"
	@echo "  make coverage-check"
	@echo "  make test-api-gateway"
	@echo "  make test-user"
	@echo "  make test-content"
	@echo "  make test-feed"
	@echo "  make test-notification"
	@echo "  make test-storage"
	@echo ""
	@echo "Code quality:"
	@echo "  make tidy"
	@echo "  make fmt"
	@echo "  make vet"
	@echo ""
	@echo "Proto:"
	@echo "  make proto-user"
	@echo "  make proto-content"
	@echo "  make proto-feed"
	@echo "  make proto-notification"
	@echo "  make proto-storage"
	@echo "  make proto-all"
	@echo ""
	@echo "Swagger:"
	@echo "  make swagger-gen"

# ============================================================
# Local service run
# ============================================================

.PHONY: run-gateway
run-gateway:
	go run ./services/api-gateway/cmd/server

.PHONY: run-user
run-user:
	go run ./services/user-service/cmd/server

.PHONY: run-content
run-content:
	go run ./services/content-service/cmd/server

.PHONY: run-feed
run-feed:
	go run ./services/feed-service/cmd/server

.PHONY: run-storage
run-storage:
	go run ./services/storage-service/cmd/server

.PHONY: run-notification
run-notification:
	go run ./services/notification-service/cmd/server

# ============================================================
# Air dev mode
# ============================================================

.PHONY: dev-gateway
dev-gateway:
	air -c .air.api-gateway.toml

.PHONY: dev-user
dev-user:
	air -c .air.user-service.toml

.PHONY: dev-content
dev-content:
	air -c .air.content-service.toml

.PHONY: dev-feed
dev-feed:
	air -c .air.feed-service.toml

.PHONY: dev-storage
dev-storage:
	air -c .air.storage-service.toml

.PHONY: dev-notification
dev-notification:
	air -c .air.notification-service.toml

.PHONY: dev-all
dev-all:
	$(MAKE) -j6 dev-gateway dev-user dev-content dev-feed dev-storage dev-notification

# ============================================================
# Docker Compose
# ============================================================

.PHONY: docker-config
docker-config:
	$(DOCKER_COMPOSE) config

.PHONY: docker-build
docker-build:
	$(DOCKER_COMPOSE) build

.PHONY: docker-up
docker-up:
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-up-attached
docker-up-attached:
	$(DOCKER_COMPOSE) up

.PHONY: docker-down
docker-down:
	$(DOCKER_COMPOSE) down

.PHONY: docker-logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

.PHONY: docker-logs-service
docker-logs-service:
	@if [ -z "$(service)" ]; then \
		echo "Usage: make docker-logs-service service=api-gateway"; \
		exit 1; \
	fi
	$(DOCKER_COMPOSE) logs -f $(service)

.PHONY: docker-ps
docker-ps:
	$(DOCKER_COMPOSE) ps

.PHONY: docker-restart
docker-restart:
	$(DOCKER_COMPOSE) restart

.PHONY: docker-clean
docker-clean:
	$(DOCKER_COMPOSE) down -v

# ============================================================
# Database migrations: golang-migrate
# ============================================================

.PHONY: migrate-create
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

.PHONY: check-db-url
check-db-url:
	@if [ -z "$(DB_URL)" ]; then \
		echo "Error: DATABASE_URL is required"; \
		echo ""; \
		echo "Option 1: put DATABASE_URL inside $(ENV_FILE)"; \
		echo "Option 2: pass it directly:"; \
		echo "make migrate-up DATABASE_URL='postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require'"; \
		exit 1; \
	fi

.PHONY: migrate-up
migrate-up: check-db-url
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down: check-db-url
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1

.PHONY: migrate-version
migrate-version: check-db-url
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version

.PHONY: migrate-force
migrate-force: check-db-url
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=1"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(version)

.PHONY: migrate-drop
migrate-drop: check-db-url
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" drop

# ============================================================
# Code quality
# ============================================================

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: fmt-check
fmt-check:
	@UNFORMATTED=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "These files are not gofmt formatted:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

.PHONY: vet
vet:
	go vet ./...

# ============================================================
# Tests
# ============================================================

.PHONY: test
test:
	GOCACHE="$(GOCACHE)" go test $(TEST_PACKAGES)

.PHONY: test-api-gateway
test-api-gateway:
	GOCACHE="$(GOCACHE)" go test $(API_GATEWAY_PACKAGES) -v

.PHONY: test-user
test-user:
	GOCACHE="$(GOCACHE)" go test $(USER_PACKAGES) -v

.PHONY: test-content
test-content:
	GOCACHE="$(GOCACHE)" go test $(CONTENT_PACKAGES) -v

.PHONY: test-feed
test-feed:
	GOCACHE="$(GOCACHE)" go test $(FEED_PACKAGES) -v

.PHONY: test-notification
test-notification:
	GOCACHE="$(GOCACHE)" go test $(NOTIFICATION_PACKAGES) -v

.PHONY: test-storage
test-storage:
	GOCACHE="$(GOCACHE)" go test $(STORAGE_PACKAGES) -v

.PHONY: test-coverage
test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(TEST_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_PROFILE)
	go tool cover -func=$(COVERAGE_PROFILE) | tee $(COVERAGE_DIR)/coverage.txt

.PHONY: coverage-html
coverage-html: test-coverage
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_DIR)/coverage.html

.PHONY: coverage-check
coverage-check: test-coverage
	@coverage=$$(go tool cover -func=$(COVERAGE_PROFILE) | awk '/total:/ { gsub("%","",$$3); print $$3 }'); \
	echo "Total coverage: $$coverage%"; \
	awk -v coverage=$$coverage -v threshold=$(COVERAGE_THRESHOLD) 'BEGIN { if (coverage < threshold) exit 1 }'

.PHONY: coverage-api-gateway
coverage-api-gateway:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(API_GATEWAY_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/api-gateway.out
	go tool cover -func=$(COVERAGE_DIR)/api-gateway.out | tee $(COVERAGE_DIR)/api-gateway.txt

.PHONY: coverage-user
coverage-user:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(USER_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/user-service.out
	go tool cover -func=$(COVERAGE_DIR)/user-service.out | tee $(COVERAGE_DIR)/user-service.txt

.PHONY: coverage-content
coverage-content:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(CONTENT_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/content-service.out
	go tool cover -func=$(COVERAGE_DIR)/content-service.out | tee $(COVERAGE_DIR)/content-service.txt

.PHONY: coverage-feed
coverage-feed:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(FEED_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/feed-service.out
	go tool cover -func=$(COVERAGE_DIR)/feed-service.out | tee $(COVERAGE_DIR)/feed-service.txt

.PHONY: coverage-notification
coverage-notification:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(NOTIFICATION_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/notification-service.out
	go tool cover -func=$(COVERAGE_DIR)/notification-service.out | tee $(COVERAGE_DIR)/notification-service.txt

.PHONY: coverage-storage
coverage-storage:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test $(STORAGE_PACKAGES) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/storage-service.out
	go tool cover -func=$(COVERAGE_DIR)/storage-service.out | tee $(COVERAGE_DIR)/storage-service.txt

.PHONY: test-report
test-report:
	./scripts/test-report.sh

# ============================================================
# Proto generation
# ============================================================

.PHONY: proto-user
proto-user:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/user/user.proto

.PHONY: proto-content
proto-content:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/content/content.proto

.PHONY: proto-feed
proto-feed:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/feed/feed.proto

.PHONY: proto-notification
proto-notification:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/notification/notification.proto

.PHONY: proto-storage
proto-storage:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/storage/storage.proto

.PHONY: proto-all
proto-all: proto-user proto-content proto-feed proto-notification proto-storage

# ============================================================
# Swagger
# ============================================================

.PHONY: swagger-gen
swagger-gen:
	swag init \
		-g main.go \
		-d services/api-gateway/cmd/server,services/api-gateway/internal/handlers,services/api-gateway/internal/dto \
		-o services/api-gateway/docs \
		--parseInternal \
		--parseDependency