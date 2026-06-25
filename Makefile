-include .env

MIGRATIONS_PATH=./db/migrations
DB_URL=$(DATABASE_URL)

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make run-gateway"
	@echo "  make run-user"
	@echo "  make run-content"
	@echo "  make run-feed"
	@echo "  make run-storage"
	@echo "  make run-notification"
	@echo "  make dev-gateway"
	@echo "  make dev-user"
	@echo "  make dev-content"
	@echo "  make dev-feed"
	@echo "  make dev-storage"
	@echo "  make dev-notification"
	@echo "  make dev-all"
	@echo "  make docker-up"
	@echo "  make docker-down"
	@echo "  make docker-logs"
	@echo "  make migrate-create name=create_users_table"
	@echo "  make migrate-up"
	@echo "  make migrate-down"
	@echo "  make migrate-version"
	@echo "  make migrate-force version=1"
	@echo "  make test"
	@echo "  make tidy"
	@echo "  make fmt"
	@echo "  make proto-user"
	@echo "  make swagger-gen"

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

.PHONY: docker-up
docker-up:
	docker compose -f deployments/docker-compose.yml up -d

.PHONY: docker-down
docker-down:
	docker compose -f deployments/docker-compose.yml down

.PHONY: docker-logs
docker-logs:
	docker compose -f deployments/docker-compose.yml logs -f

.PHONY: migrate-create
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

.PHONY: migrate-up
migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1

.PHONY: migrate-version
migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version

.PHONY: migrate-force
migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=1"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(version)

.PHONY: test
test:
	go test ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: fmt
fmt:
	go fmt ./...

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
.PHONY: swagger-gen
swagger-gen:
	swag init \
		-g cmd/server/main.go \
		-d services/api-gateway,services/api-gateway/internal/handlers,services/api-gateway/internal/dto \
		-o services/api-gateway/docs \
		--parseInternal \
		--parseDependency