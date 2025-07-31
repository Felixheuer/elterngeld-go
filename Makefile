# Elterngeld Portal Makefile

# Variables
BINARY_NAME=elterngeld-portal
MAIN_PATH=./cmd/server
BUILD_DIR=./build
MIGRATIONS_PATH=./migrations
DB_URL=sqlite://./data/database.db
DOCKER_COMPOSE=docker-compose

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Elterngeld Portal - Available Commands$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: clean deps build test ## Clean, get dependencies, build and test

# Development Commands
.PHONY: run
run: ## Run the application in development mode
	@echo "$(GREEN)Starting Elterngeld Portal in development mode...$(NC)"
	@mkdir -p data storage/uploads
	@$(GOCMD) run $(MAIN_PATH)/main.go

.PHONY: dev
dev: deps run ## Install dependencies and run in development mode

.PHONY: watch
watch: ## Run with auto-reload (requires air)
	@echo "$(GREEN)Starting development server with auto-reload...$(NC)"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(RED)air not found. Install with: go install github.com/cosmtrek/air@latest$(NC)"; \
		echo "$(YELLOW)Falling back to normal run...$(NC)"; \
		make run; \
	fi

# Build Commands
.PHONY: build
build: ## Build the application
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "$(GREEN)Build completed: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

.PHONY: build-linux
build-linux: ## Build for Linux
	@echo "$(GREEN)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)/main.go

.PHONY: build-windows
build-windows: ## Build for Windows
	@echo "$(GREEN)Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe $(MAIN_PATH)/main.go

.PHONY: build-mac
build-mac: ## Build for macOS
	@echo "$(GREEN)Building for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin $(MAIN_PATH)/main.go

.PHONY: build-all
build-all: build-linux build-windows build-mac ## Build for all platforms

# Dependency Commands
.PHONY: deps
deps: ## Download and verify dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) verify
	@$(GOMOD) tidy

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade all dependencies
	@echo "$(GREEN)Upgrading dependencies...$(NC)"
	@$(GOGET) -u ./...
	@$(GOMOD) tidy

# Testing Commands
.PHONY: test
test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	@$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "$(GREEN)Running tests with race detection...$(NC)"
	@$(GOTEST) -v -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	@$(GOTEST) -v -tags=integration ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

# Database Commands
.PHONY: migrate
migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) up; \
	else \
		echo "$(RED)migrate tool not found. Install with:$(NC)"; \
		echo "$(YELLOW)go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest$(NC)"; \
	fi

.PHONY: migrate-down
migrate-down: ## Rollback database migrations
	@echo "$(YELLOW)Rolling back database migrations...$(NC)"
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) down; \
	else \
		echo "$(RED)migrate tool not found$(NC)"; \
	fi

.PHONY: migrate-create
migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "$(RED)Error: name parameter is required$(NC)"; \
		echo "$(YELLOW)Usage: make migrate-create name=migration_name$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Creating migration: $(name)$(NC)"
	@mkdir -p $(MIGRATIONS_PATH)
	@if command -v migrate > /dev/null; then \
		migrate create -ext sql -dir $(MIGRATIONS_PATH) $(name); \
	else \
		echo "$(RED)migrate tool not found$(NC)"; \
	fi

.PHONY: seed
seed: ## Seed the database with sample data
	@echo "$(GREEN)Seeding database...$(NC)"
	@mkdir -p data
	@SEED_DATA=true $(GOCMD) run $(MAIN_PATH)/main.go --seed

.PHONY: db-reset
db-reset: ## Reset database (drop and recreate)
	@echo "$(YELLOW)Resetting database...$(NC)"
	@rm -f ./data/database.db
	@make migrate
	@make seed

# Code Quality Commands
.PHONY: fmt
fmt: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOFMT) ./...

.PHONY: lint
lint: ## Run linters
	@echo "$(GREEN)Running linters...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint not found. Install with:$(NC)"; \
		echo "$(YELLOW)curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.55.2$(NC)"; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	@$(GOCMD) vet ./...

.PHONY: check
check: fmt vet lint test ## Run all code quality checks

# Documentation Commands
.PHONY: swagger
swagger: ## Generate Swagger documentation
	@echo "$(GREEN)Generating Swagger documentation...$(NC)"
	@if command -v swag > /dev/null; then \
		swag init -g cmd/server/main.go -o ./docs; \
		echo "$(GREEN)Swagger docs generated at ./docs$(NC)"; \
	else \
		echo "$(RED)swag not found. Install with:$(NC)"; \
		echo "$(YELLOW)go install github.com/swaggo/swag/cmd/swag@latest$(NC)"; \
	fi

.PHONY: docs
docs: swagger ## Generate all documentation

# Docker Commands
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(NC)"
	@docker build -t elterngeld-portal .

.PHONY: docker-run
docker-run: ## Run application in Docker
	@echo "$(GREEN)Running application in Docker...$(NC)"
	@docker run -p 8080:8080 --env-file .env elterngeld-portal

.PHONY: docker-up
docker-up: ## Start all services with Docker Compose
	@echo "$(GREEN)Starting services with Docker Compose...$(NC)"
	@$(DOCKER_COMPOSE) up -d

.PHONY: docker-down
docker-down: ## Stop all services
	@echo "$(YELLOW)Stopping services...$(NC)"
	@$(DOCKER_COMPOSE) down

.PHONY: docker-logs
docker-logs: ## Show logs from Docker Compose
	@$(DOCKER_COMPOSE) logs -f

.PHONY: docker-ps
docker-ps: ## Show running containers
	@$(DOCKER_COMPOSE) ps

# Utility Commands
.PHONY: clean
clean: ## Clean build artifacts and caches
	@echo "$(GREEN)Cleaning...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@$(GOCMD) clean -cache
	@$(GOCMD) clean -testcache

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(NC)"
	@$(GOGET) github.com/cosmtrek/air@latest
	@$(GOGET) github.com/swaggo/swag/cmd/swag@latest
	@$(GOGET) -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

.PHONY: version
version: ## Show application version
	@echo "$(GREEN)Elterngeld Portal v1.0.0$(NC)"

.PHONY: env-example
env-example: ## Copy .env.example to .env
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(GREEN).env file created from .env.example$(NC)"; \
		echo "$(YELLOW)Please update the values in .env as needed$(NC)"; \
	else \
		echo "$(YELLOW).env file already exists$(NC)"; \
	fi

.PHONY: setup
setup: deps install-tools env-example ## Initial project setup
	@echo "$(GREEN)Setting up project...$(NC)"
	@mkdir -p data storage/uploads logs
	@echo "$(GREEN)Project setup complete!$(NC)"
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Update .env with your configuration"
	@echo "  2. Run 'make migrate' to setup database"
	@echo "  3. Run 'make dev' to start development server"

# Production Commands
.PHONY: prod-build
prod-build: ## Build for production
	@echo "$(GREEN)Building for production...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -ldflags '-extldflags "-static"' -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)/main.go

.PHONY: deploy
deploy: prod-build ## Deploy to production (customize as needed)
	@echo "$(GREEN)Deploying to production...$(NC)"
	@echo "$(YELLOW)Customize this target for your deployment process$(NC)"

# Health Commands
.PHONY: health
health: ## Check application health
	@echo "$(GREEN)Checking application health...$(NC)"
	@curl -f http://localhost:8080/health || echo "$(RED)Application is not running$(NC)"

.PHONY: ready
ready: ## Check application readiness
	@echo "$(GREEN)Checking application readiness...$(NC)"
	@curl -f http://localhost:8080/ready || echo "$(RED)Application is not ready$(NC)"