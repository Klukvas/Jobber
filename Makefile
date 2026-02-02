.PHONY: help dev build up down logs clean deploy terraform-init terraform-apply terraform-destroy

# Load environment variables
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Variables
SERVER_IP ?= $(shell cd terraform && terraform output -raw server_ip 2>/dev/null)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## Development
dev: ## Start services for local development (with override)
	docker compose -f docker-compose.yml -f docker-compose.override.yml up -d
	@echo "Services started. Backend: http://localhost:8080, Frontend: http://localhost:5173"

dev-logs: ## Follow logs for local development
	docker compose -f docker-compose.yml -f docker-compose.override.yml logs -f

## Docker operations
build: ## Build all Docker images
	docker compose build

up: ## Start all services in background
	docker compose up -d

down: ## Stop all services
	docker compose down

restart: ## Restart all services
	docker compose restart

logs: ## Follow logs from all services
	docker compose logs -f

logs-backend: ## Follow backend logs only
	docker compose logs -f backend

logs-frontend: ## Follow frontend logs only
	docker compose logs -f frontend

ps: ## Show running containers
	docker compose ps

clean: ## Stop services and remove volumes (⚠️  deletes data!)
	@echo "⚠️  This will delete ALL data (database, redis)!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker compose down -v; \
		echo "✓ Services stopped and volumes removed"; \
	fi

## Terraform operations
terraform-init: ## Initialize Terraform
	cd terraform && terraform init

terraform-plan: ## Preview infrastructure changes
	cd terraform && terraform plan

terraform-apply: ## Create/update infrastructure
	cd terraform && terraform apply

terraform-destroy: ## Destroy infrastructure
	cd terraform && terraform destroy

terraform-output: ## Show Terraform outputs
	cd terraform && terraform output

## Deployment
deploy: ## Deploy to server (requires SERVER_IP)
	@if [ -z "$(SERVER_IP)" ]; then \
		echo "Error: SERVER_IP not set"; \
		echo "Either:"; \
		echo "  1. Set SERVER_IP environment variable"; \
		echo "  2. Run 'make terraform-apply' first"; \
		exit 1; \
	fi
	./scripts/deploy.sh $(SERVER_IP)

ssh: ## SSH to server
	@if [ -z "$(SERVER_IP)" ]; then \
		echo "Error: SERVER_IP not set. Run 'make terraform-apply' first"; \
		exit 1; \
	fi
	ssh root@$(SERVER_IP)

## Database operations
db-shell: ## Connect to PostgreSQL shell
	docker compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

db-backup: ## Backup database to file
	docker compose exec postgres pg_dump -U $(DB_USER) $(DB_NAME) > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✓ Database backed up"

db-restore: ## Restore database from file (usage: make db-restore FILE=backup.sql)
	@if [ -z "$(FILE)" ]; then \
		echo "Error: FILE not specified. Usage: make db-restore FILE=backup.sql"; \
		exit 1; \
	fi
	cat $(FILE) | docker compose exec -T postgres psql -U $(DB_USER) $(DB_NAME)
	@echo "✓ Database restored"

redis-cli: ## Connect to Redis CLI
	docker compose exec redis redis-cli

## Utilities
setup-env: ## Copy .env.example to .env
	cp -n .env.example .env || true
	@echo "✓ .env file created. Please edit it with your values."

check-env: ## Verify required environment variables
	@echo "Checking environment variables..."
	@test -n "$(JWT_ACCESS_SECRET)" || (echo "❌ JWT_ACCESS_SECRET not set" && exit 1)
	@test -n "$(JWT_REFRESH_SECRET)" || (echo "❌ JWT_REFRESH_SECRET not set" && exit 1)
	@test -n "$(S3_ENDPOINT)" || (echo "❌ S3_ENDPOINT not set" && exit 1)
	@test -n "$(S3_ACCESS_KEY)" || (echo "❌ S3_ACCESS_KEY not set" && exit 1)
	@echo "✓ All required variables set"

generate-secrets: ## Generate random JWT secrets
	@echo "Add these to your .env file:"
	@echo ""
	@echo "JWT_ACCESS_SECRET=$$(openssl rand -base64 32)"
	@echo "JWT_REFRESH_SECRET=$$(openssl rand -base64 32)"

## Info
status: ## Show system status
	@echo "=== Docker Status ==="
	@docker compose ps
	@echo ""
	@echo "=== Volumes ==="
	@docker volume ls | grep jobber || echo "No volumes found"
	@echo ""
	@if [ -n "$(SERVER_IP)" ]; then \
		echo "=== Server Info ==="; \
		echo "IP: $(SERVER_IP)"; \
		echo "URL: http://$(SERVER_IP)"; \
	fi
