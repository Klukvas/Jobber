.PHONY: help run dev build test lint clean migrate-up migrate-down migrate-create sqlc docker-up docker-down

# Variables
APP_NAME=jobber
MAIN_PATH=cmd/api/main.go
BINARY_PATH=bin/$(APP_NAME)
MIGRATIONS_PATH=migrations
DATABASE_URL=postgresql://jobber:jobber@localhost:5432/jobber?sslmode=disable

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the application
	go run $(MAIN_PATH)

dev: ## Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
	air

build: ## Build the application
	@mkdir -p bin
	go build -o $(BINARY_PATH) $(MAIN_PATH)

test: ## Run all tests
	go test -v -race ./...

test-unit: ## Run unit tests
	go test -v -race -short ./...

test-integration: ## Run integration tests
	go test -v -race -run Integration ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	go mod download
	go mod tidy

sqlc: ## Generate sqlc code
	sqlc generate

swagger: ## Generate Swagger documentation
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

migrate-up: ## Run database migrations
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down: ## Rollback database migrations
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-force: ## Force migration version (usage: make migrate-force version=1)
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(version)

docker-up: ## Start Docker services (PostgreSQL + Redis)
	docker-compose up -d

docker-down: ## Stop Docker services
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

setup: deps docker-up ## Setup project (install deps and start services)
	@echo "Waiting for database to be ready..."
	@sleep 3
	@make migrate-up
	@echo "Setup complete! Run 'make run' to start the server."
