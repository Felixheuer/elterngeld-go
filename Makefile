# Elterngeld Portal - Optimiertes Entwickler-Makefile

# Variables
BINARY_NAME=elterngeld-portal
MAIN_PATH=./cmd/server
BUILD_DIR=./build
DB_PATH=./data/database.db
MIGRATIONS_PATH=./migrations

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Colors
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m

.PHONY: help
help: ## Show available commands
	@echo "$(GREEN)Elterngeld Portal - Development Commands$(NC)"
	@echo ""
	@echo "$(GREEN)Core Commands:$(NC)"
	@echo "  make db         - Initialize database"
	@echo "  make reset-db   - Reset database completely"
	@echo "  make migrate    - Run database migrations"
	@echo "  make seed       - Fill database with sample data"
	@echo "  make run        - Start development server"
	@echo "  make test       - Run tests"
	@echo "  make build      - Build/compile project"
	@echo "  make clean      - Clean temporary files"
	@echo "  make lint       - Check code style"
	@echo "  make format     - Format code automatically"
	@echo ""
	@echo "$(GREEN)Setup:$(NC)"
	@echo "  make setup      - Initial project setup"
	@echo "  make deps       - Install dependencies"

# =============================================================================
# CORE DEVELOPER COMMANDS
# =============================================================================

.PHONY: db
db: deps ## Initialize database
	@echo "$(GREEN)Initializing database...$(NC)"
	@mkdir -p data storage/uploads
	@$(GOCMD) run $(MAIN_PATH)/main.go --init-db
	@echo "$(GREEN)Database initialized successfully$(NC)"

.PHONY: reset-db
reset-db: ## Reset database completely (delete + rebuild)
	@echo "$(YELLOW)Resetting database completely...$(NC)"
	@rm -f $(DB_PATH)
	@mkdir -p data
	@$(GOCMD) run $(MAIN_PATH)/main.go --init-db
	@echo "$(GREEN)Database reset completed$(NC)"

.PHONY: migrate
migrate: deps ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@mkdir -p data
	@$(GOCMD) run $(MAIN_PATH)/main.go --migrate
	@echo "$(GREEN)Migrations completed$(NC)"

.PHONY: seed
seed: deps ## Fill database with sample data
	@echo "$(GREEN)Seeding database with sample data...$(NC)"
	@SEED_DATA=true $(GOCMD) run $(MAIN_PATH)/main.go --seed
	@echo "$(GREEN)Database seeded successfully$(NC)"
	@echo "$(YELLOW)Test users created:$(NC)"
	@echo "  Admin:   admin@elterngeld-portal.de / admin123"
	@echo "  Berater: berater@elterngeld-portal.de / berater123"  
	@echo "  User:    user@example.com / user123"

.PHONY: run
run: deps ## Start development server
	@echo "$(GREEN)Starting development server...$(NC)"
	@mkdir -p data storage/uploads
	@$(GOCMD) run $(MAIN_PATH)/main.go

.PHONY: test
test: deps ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	@$(GOTEST) -v ./...
	@echo "$(GREEN)Tests completed$(NC)"

.PHONY: build
build: deps ## Build/compile project
	@echo "$(GREEN)Building project...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "$(GREEN)Build completed: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

.PHONY: clean
clean: ## Clean temporary files and build artifacts
	@echo "$(GREEN)Cleaning temporary files...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@$(GOCMD) clean -cache
	@$(GOCMD) clean -testcache
	@echo "$(GREEN)Cleanup completed$(NC)"

.PHONY: lint
lint: ## Check code style
	@echo "$(GREEN)Checking code style...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
		echo "$(GREEN)Linting completed$(NC)"; \
	else \
		echo "$(YELLOW)golangci-lint not found, running go vet instead...$(NC)"; \
		$(GOCMD) vet ./...; \
		echo "$(GREEN)Code check completed$(NC)"; \
	fi

.PHONY: format
format: ## Format code automatically
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOFMT) ./...
	@echo "$(GREEN)Code formatting completed$(NC)"

# =============================================================================
# SETUP & DEPENDENCIES
# =============================================================================

.PHONY: setup
setup: ## Initial project setup
	@echo "$(GREEN)Setting up project...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) tidy
	@mkdir -p data storage/uploads logs $(BUILD_DIR)
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(GREEN).env file created$(NC)"; \
	fi
	@echo "$(GREEN)Project setup completed!$(NC)"
	@echo "$(YELLOW)Run 'make db && make seed && make run' to start$(NC)"

.PHONY: deps
deps: ## Install dependencies
	@$(GOMOD) download
	@$(GOMOD) tidy

# =============================================================================
# QUICK WORKFLOW COMMANDS
# =============================================================================

.PHONY: dev
dev: setup db seed run ## Complete development setup and start

.PHONY: fresh
fresh: clean reset-db seed run ## Fresh start (clean + reset + seed + run)

.PHONY: check
check: format lint test ## Run all quality checks

# =============================================================================
# UTILITY COMMANDS
# =============================================================================

.PHONY: status
status: ## Show system status
	@echo "$(GREEN)System Status:$(NC)"
	@echo "Database: $(if $(wildcard $(DB_PATH)),$(GREEN)exists$(NC),$(RED)missing$(NC))"
	@echo "Binary: $(if $(wildcard $(BUILD_DIR)/$(BINARY_NAME)),$(GREEN)built$(NC),$(YELLOW)not built$(NC))"
	@echo "Env file: $(if $(wildcard .env),$(GREEN)exists$(NC),$(RED)missing$(NC))"

.PHONY: health
health: ## Check if server is running
	@echo "$(GREEN)Checking server health...$(NC)"
	@curl -s http://localhost:8080/health > /dev/null && \
		echo "$(GREEN)Server is running$(NC)" || \
		echo "$(RED)Server is not running$(NC)"

# Default target
.DEFAULT_GOAL := help