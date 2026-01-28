# IndustryDB - Makefile
# Common commands for development, testing, and deployment

.PHONY: help dev stop restart logs clean build test lint fmt install \
        fetch-industry normalize-data import-db validate-data export-data \
        backend-dev frontend-dev migrate-up migrate-down db-shell redis-shell \
        docker-setup docker-start docker-stop docker-logs docker-clean

# Default target
.DEFAULT_GOAL := help

# ================================
# HELP
# ================================
help: ## Show this help message
	@echo "IndustryDB - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Quick Start:"
	@echo "  1. make docker-setup    (first time only)"
	@echo "  2. make dev             (start all services)"
	@echo "  3. make logs            (view logs)"
	@echo ""

# ================================
# DOCKER DEVELOPMENT (NEW)
# ================================
docker-setup: ## Setup Docker network (first time only)
	@echo "🔧 Setting up Docker environment..."
	@docker network create industrydb-shared 2>/dev/null && echo "✅ Network created" || echo "ℹ️  Network already exists"

docker-start: docker-setup ## Start all services with Docker
	@./start-all.sh

docker-stop: ## Stop all Docker services
	@./stop-all.sh

docker-logs: ## View all logs (requires tmux)
	@./logs-all.sh

docker-logs-backend: ## View backend logs only
	@cd backend && docker-compose logs -f

docker-logs-frontend: ## View frontend logs only
	@cd frontend && docker-compose logs -f

docker-clean: ## Stop and remove all containers and volumes
	@echo "🧹 Cleaning Docker environment..."
	@cd backend && docker-compose down -v
	@cd frontend && docker-compose down -v
	@echo "✅ Docker cleaned!"

docker-rebuild: ## Rebuild all Docker images
	@echo "🔨 Rebuilding Docker images..."
	@cd backend && docker-compose build
	@cd frontend && docker-compose build
	@echo "✅ Rebuild complete!"

# ================================
# DEVELOPMENT (Shortcuts to docker-*)
# ================================
dev: docker-start ## Start all services (alias for docker-start)

stop: docker-stop ## Stop all services (alias for docker-stop)

restart: stop dev ## Restart all services

logs: docker-logs-backend ## View backend logs (default)

logs-backend: docker-logs-backend ## View backend logs only

logs-frontend: docker-logs-frontend ## View frontend logs only

# ================================
# BUILD
# ================================
build: docker-rebuild ## Build all Docker images

build-backend: ## Build backend Docker image only
	@cd backend && docker-compose build

build-frontend: ## Build frontend Docker image only
	@cd frontend && docker-compose build

# ================================
# TESTING
# ================================
test: ## Run all tests (backend + frontend)
	@echo "🧪 Running tests..."
	@$(MAKE) test-backend
	@$(MAKE) test-frontend
	@echo "✅ All tests passed!"

test-backend: ## Run backend tests
	@echo "🧪 Running backend tests..."
	@cd backend && go test ./... -v -cover -coverprofile=coverage.out

test-frontend: ## Run frontend tests
	@echo "🧪 Running frontend tests..."
	@cd frontend && npm test

test-coverage: ## Generate test coverage report
	@cd backend && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: backend/coverage.html"

# ================================
# LINTING & FORMATTING
# ================================
lint: lint-backend lint-frontend ## Run all linters

lint-backend: ## Lint backend code
	@echo "🔍 Linting backend..."
	@cd backend && golangci-lint run ./...

lint-frontend: ## Lint frontend code
	@echo "🔍 Linting frontend..."
	@cd frontend && npm run lint

fmt: ## Format all code
	@echo "✨ Formatting code..."
	@cd backend && go fmt ./...
	@cd frontend && npm run format

# ================================
# DEPENDENCIES
# ================================
install: ## Install all dependencies
	@echo "📦 Installing dependencies..."
	@$(MAKE) install-backend
	@$(MAKE) install-frontend
	@$(MAKE) install-scripts
	@echo "✅ Dependencies installed!"

install-backend: ## Install backend dependencies
	@cd backend && go mod download && go mod tidy

install-frontend: ## Install frontend dependencies
	@cd frontend && npm install

install-scripts: ## Install Python dependencies for scripts
	@cd scripts && python3 -m venv venv && . venv/bin/activate && pip install -r requirements.txt

# ================================
# DATABASE
# ================================
migrate-up: ## Run database migrations (up)
	@echo "⬆️  Running migrations..."
	@cd backend && go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema

migrate-down: ## Rollback database migrations (down)
	@echo "⬇️  Rolling back migrations..."
	@echo "⚠️  Not implemented yet"

db-shell: ## Open PostgreSQL shell
	@cd backend && docker-compose exec db psql -U industrydb -d industrydb

db-seed: ## Seed database with sample data (15 leads)
	@echo "🌱 Seeding database with sample data..."
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/seed.go

seed-all: ## Seed all 20 industries with 2,900+ realistic leads
	@echo "🌱 Seeding all industries..."
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/seed-all.go

seed-industry: ## Seed specific industries (Usage: make seed-industry INDUSTRIES=tattoo,beauty,gym)
	@echo "🌱 Seeding industries: $(INDUSTRIES)"
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/seed-all.go --industries=$(INDUSTRIES)

seed-reset: ## Reset database and reseed all industries
	@echo "⚠️  WARNING: This will delete all data and reseed!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/seed-all.go --reset; \
	fi

db-stats: ## Show database statistics
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/db-utils.go --action=stats

db-export: ## Export database to JSON (Usage: make db-export OUTPUT=backup.json)
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/db-utils.go --action=export --output=$(OUTPUT)

db-import: ## Import database from JSON (Usage: make db-import INPUT=backup.json)
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/db-utils.go --action=import --input=$(INPUT)

db-clean-test: ## Clean test data only
	@cd backend && DATABASE_URL="postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable" go run scripts/db-utils.go --action=clean-test

db-reset: ## Drop and recreate database (DANGEROUS!)
	@echo "⚠️  WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd backend && docker-compose exec db psql -U industrydb -c "DROP DATABASE IF EXISTS industrydb;"; \
		cd backend && docker-compose exec db psql -U industrydb -c "CREATE DATABASE industrydb;"; \
		cd backend && docker-compose exec db psql -U industrydb -d industrydb -c "CREATE EXTENSION IF NOT EXISTS postgis;"; \
		echo "✅ Database reset complete!"; \
	fi

redis-shell: ## Open Redis CLI
	@cd backend && docker-compose exec redis redis-cli

# ================================
# DATA PIPELINE
# ================================
fetch-industry: ## Fetch data from OSM (Usage: make fetch-industry INDUSTRY=tattoo COUNTRY=US CITY="New York")
	@echo "🌍 Fetching $(INDUSTRY) data..."
	@cd scripts/data-acquisition && \
	. venv/bin/activate && \
	python fetchers/$(INDUSTRY)_fetcher.py --country=$(COUNTRY) --city="$(CITY)"
	@echo "✅ Data fetched!"

normalize-data: ## Normalize fetched data
	@echo "🔄 Normalizing data..."
	@cd scripts/data-acquisition && \
	. venv/bin/activate && \
	python normalizer.py
	@echo "✅ Data normalized!"

import-db: ## Import normalized data to PostgreSQL
	@echo "📥 Importing data to database..."
	@cd scripts/data-import && \
	. venv/bin/activate && \
	python import_to_postgres.py
	@echo "✅ Data imported!"

validate-data: ## Validate data quality
	@echo "✅ Validating data..."
	@cd scripts/data-import && \
	. venv/bin/activate && \
	python validation.py
	@echo "✅ Validation complete!"

export-data: ## Export data from database (Usage: make export-data FORMAT=csv)
	@echo "📤 Exporting data..."
	@cd scripts/data-import && \
	. venv/bin/activate && \
	python export_data.py --format=$(FORMAT)
	@echo "✅ Data exported!"

# ================================
# STANDALONE DEVELOPMENT (Without Docker)
# ================================
backend-dev: ## Run backend in development mode (without Docker)
	@cd backend && go run cmd/api/main.go

frontend-dev: ## Run frontend in development mode (without Docker)
	@cd frontend && npm run dev

# ================================
# CLEANUP
# ================================
clean: ## Clean build artifacts and caches
	@echo "🧹 Cleaning..."
	@rm -rf backend/coverage.out backend/coverage.html
	@rm -rf frontend/.next frontend/node_modules
	@rm -rf scripts/data-acquisition/venv
	@rm -rf data/output/*.json data/output/*.csv
	@echo "✅ Cleanup complete!"

clean-all: clean docker-clean ## Clean everything including Docker volumes

# ================================
# PRODUCTION
# ================================
deploy: ## Deploy to production (requires setup)
	@echo "🚀 Deploying to production..."
	@echo "⚠️  Not implemented yet. Set up CI/CD first."

# ================================
# UTILITIES
# ================================
ps: ## Show running containers
	@echo "Backend containers:"
	@cd backend && docker-compose ps
	@echo ""
	@echo "Frontend containers:"
	@cd frontend && docker-compose ps

stats: ## Show container resource usage
	@docker stats

version: ## Show version info
	@echo "IndustryDB v0.1.0"
	@echo ""
	@echo "Go version:"
	@go version 2>/dev/null || echo "  (not installed)"
	@echo ""
	@echo "Node version:"
	@node --version 2>/dev/null || echo "  (not installed)"
	@echo ""
	@echo "Python version:"
	@python3 --version 2>/dev/null || echo "  (not installed)"
	@echo ""
	@echo "Docker version:"
	@docker --version 2>/dev/null || echo "  (not installed)"

# ================================
# QUICK WORKFLOWS
# ================================
setup: install docker-setup ## Complete initial setup
	@echo "🎉 Setup complete! Run 'make dev' to start."

workflow-data: ## Complete data workflow (fetch → normalize → import)
	@$(MAKE) fetch-industry
	@$(MAKE) normalize-data
	@$(MAKE) import-db
	@$(MAKE) validate-data
	@echo "🎉 Data workflow complete!"

# ================================
# DOCKER HELPERS
# ================================
docker-backend-shell: ## Access backend container shell
	@cd backend && docker-compose exec backend sh

docker-frontend-shell: ## Access frontend container shell
	@cd frontend && docker-compose exec frontend sh

docker-db-shell: db-shell ## Alias for db-shell

docker-redis-shell: redis-shell ## Alias for redis-shell
