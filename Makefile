.PHONY: help test test-unit test-integration test-smoke test-regression test-e2e test-all
.PHONY: test-coverage test-verbose test-race lint build clean
.PHONY: build-all build-jupyter build-rstudio build-vscode install

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build parameters
BINARY_DIR=bin
PKG_DIR=pkg
APPS_DIR=apps

# Test parameters
TEST_TIMEOUT=10m
SHORT_TIMEOUT=30s
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

##@ Help

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Testing

test: test-unit ## Run all fast unit tests (default)

test-unit: ## Run unit tests that don't require AWS (fast, no external dependencies)
	@echo "Running unit tests (no AWS required)..."
	@cd $(PKG_DIR) && $(GOTEST) -short -timeout $(SHORT_TIMEOUT) ./...
	@cd $(APPS_DIR)/jupyter && $(GOTEST) -short -timeout $(SHORT_TIMEOUT) ./...
	@cd $(APPS_DIR)/rstudio && $(GOTEST) -short -timeout $(SHORT_TIMEOUT) ./...
	@cd $(APPS_DIR)/vscode && $(GOTEST) -short -timeout $(SHORT_TIMEOUT) ./...
	@echo "✓ All unit tests passed"

test-integration: ## Run integration tests with mocked AWS services (requires localstack/moto)
	@echo "Running integration tests..."
	@echo "Note: Integration tests require localstack or moto to be running"
	@cd $(PKG_DIR) && $(GOTEST) -tags=integration -timeout $(TEST_TIMEOUT) ./...
	@cd $(APPS_DIR)/jupyter && $(GOTEST) -tags=integration -timeout $(TEST_TIMEOUT) ./...
	@cd $(APPS_DIR)/rstudio && $(GOTEST) -tags=integration -timeout $(TEST_TIMEOUT) ./...
	@cd $(APPS_DIR)/vscode && $(GOTEST) -tags=integration -timeout $(TEST_TIMEOUT) ./...
	@echo "✓ All integration tests passed"

test-smoke: ## Run smoke tests against real AWS (requires AWS credentials)
	@echo "Running smoke tests..."
	@echo "Note: Smoke tests will create real AWS resources and may incur costs"
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd $(PKG_DIR) && $(GOTEST) -tags=smoke -timeout $(TEST_TIMEOUT) ./...; \
		cd $(APPS_DIR)/jupyter && $(GOTEST) -tags=smoke -timeout $(TEST_TIMEOUT) ./...; \
		cd $(APPS_DIR)/rstudio && $(GOTEST) -tags=smoke -timeout $(TEST_TIMEOUT) ./...; \
		cd $(APPS_DIR)/vscode && $(GOTEST) -tags=smoke -timeout $(TEST_TIMEOUT) ./...; \
		echo "✓ All smoke tests passed"; \
	else \
		echo "Smoke tests cancelled"; \
	fi

test-e2e: ## Run end-to-end tests (full launch → connect → terminate flow)
	@echo "Running end-to-end tests..."
	@echo "Note: E2E tests will create real AWS resources and may incur costs"
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd $(PKG_DIR) && $(GOTEST) -tags=e2e -timeout 30m ./...; \
		cd $(APPS_DIR)/jupyter && $(GOTEST) -tags=e2e -timeout 30m ./...; \
		cd $(APPS_DIR)/rstudio && $(GOTEST) -tags=e2e -timeout 30m ./...; \
		cd $(APPS_DIR)/vscode && $(GOTEST) -tags=e2e -timeout 30m ./...; \
		echo "✓ All E2E tests passed"; \
	else \
		echo "E2E tests cancelled"; \
	fi

test-regression: ## Run regression tests (all tests without long-running E2E)
	@echo "Running regression tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@echo "✓ All regression tests passed"

test-all: ## Run all tests including E2E (requires AWS credentials)
	@echo "Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@$(MAKE) test-smoke
	@$(MAKE) test-e2e
	@echo "✓ All tests passed"

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@cd $(PKG_DIR) && $(GOTEST) -short -coverprofile=$(COVERAGE_FILE) ./...
	@cd $(APPS_DIR)/jupyter && $(GOTEST) -short -coverprofile=$(COVERAGE_FILE) ./...
	@cd $(APPS_DIR)/rstudio && $(GOTEST) -short -coverprofile=$(COVERAGE_FILE) ./...
	@cd $(APPS_DIR)/vscode && $(GOTEST) -short -coverprofile=$(COVERAGE_FILE) ./...
	@echo "Generating coverage reports..."
	@cd $(PKG_DIR) && $(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "✓ Coverage report generated: $(PKG_DIR)/$(COVERAGE_HTML)"

test-verbose: ## Run unit tests with verbose output
	@echo "Running unit tests (verbose)..."
	@cd $(PKG_DIR) && $(GOTEST) -v -short -timeout $(SHORT_TIMEOUT) ./...
	@cd $(APPS_DIR)/jupyter && $(GOTEST) -v -short -timeout $(SHORT_TIMEOUT) ./...
	@cd $(APPS_DIR)/rstudio && $(GOTEST) -v -short -timeout $(SHORT_TIMEOUT) ./...
	@cd $(APPS_DIR)/vscode && $(GOTEST) -v -short -timeout $(SHORT_TIMEOUT) ./...

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@cd $(PKG_DIR) && $(GOTEST) -race -short -timeout $(TEST_TIMEOUT) ./...
	@cd $(APPS_DIR)/jupyter && $(GOTEST) -race -short -timeout $(TEST_TIMEOUT) ./...
	@cd $(APPS_DIR)/rstudio && $(GOTEST) -race -short -timeout $(TEST_TIMEOUT) ./...
	@cd $(APPS_DIR)/vscode && $(GOTEST) -race -short -timeout $(TEST_TIMEOUT) ./...
	@echo "✓ All race tests passed"

##@ Code Quality

lint: ## Run linter (golangci-lint)
	@echo "Running linter..."
	@cd $(PKG_DIR) && $(GOLINT) run
	@cd $(APPS_DIR)/jupyter && $(GOLINT) run
	@cd $(APPS_DIR)/rstudio && $(GOLINT) run
	@cd $(APPS_DIR)/vscode && $(GOLINT) run
	@echo "✓ Linting passed"

fmt: ## Format code with gofmt
	@echo "Formatting code..."
	@$(GOCMD) fmt ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOCMD) vet ./...
	@echo "✓ Vet passed"

##@ Building

build: build-all ## Build all applications

build-all: build-jupyter build-rstudio build-vscode ## Build all applications

build-jupyter: ## Build aws-jupyter
	@echo "Building aws-jupyter..."
	@mkdir -p $(BINARY_DIR)
	@cd $(APPS_DIR)/jupyter && $(GOBUILD) -o ../../$(BINARY_DIR)/aws-jupyter ./cmd/aws-jupyter
	@echo "✓ Built: $(BINARY_DIR)/aws-jupyter"

build-rstudio: ## Build aws-rstudio
	@echo "Building aws-rstudio..."
	@mkdir -p $(BINARY_DIR)
	@cd $(APPS_DIR)/rstudio && $(GOBUILD) -o ../../$(BINARY_DIR)/aws-rstudio ./cmd/aws-rstudio
	@echo "✓ Built: $(BINARY_DIR)/aws-rstudio"

build-vscode: ## Build aws-vscode
	@echo "Building aws-vscode..."
	@mkdir -p $(BINARY_DIR)
	@cd $(APPS_DIR)/vscode && $(GOBUILD) -o ../../$(BINARY_DIR)/aws-vscode ./cmd/aws-vscode
	@echo "✓ Built: $(BINARY_DIR)/aws-vscode"

install: build-all ## Install binaries to /usr/local/bin
	@echo "Installing binaries..."
	@sudo cp $(BINARY_DIR)/aws-jupyter /usr/local/bin/
	@sudo cp $(BINARY_DIR)/aws-rstudio /usr/local/bin/
	@sudo cp $(BINARY_DIR)/aws-vscode /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/"

##@ Maintenance

clean: ## Clean build artifacts and test cache
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)
	@rm -f $(PKG_DIR)/$(COVERAGE_FILE) $(PKG_DIR)/$(COVERAGE_HTML)
	@rm -f $(APPS_DIR)/*/$(COVERAGE_FILE) $(APPS_DIR)/*/$(COVERAGE_HTML)
	@$(GOCMD) clean -testcache
	@echo "✓ Cleaned"

mod-tidy: ## Run go mod tidy on all modules
	@echo "Tidying modules..."
	@cd $(PKG_DIR) && $(GOMOD) tidy
	@cd $(APPS_DIR)/jupyter && $(GOMOD) tidy
	@cd $(APPS_DIR)/rstudio && $(GOMOD) tidy
	@cd $(APPS_DIR)/vscode && $(GOMOD) tidy
	@echo "✓ Modules tidied"

mod-verify: ## Verify module dependencies
	@echo "Verifying modules..."
	@cd $(PKG_DIR) && $(GOMOD) verify
	@cd $(APPS_DIR)/jupyter && $(GOMOD) verify
	@cd $(APPS_DIR)/rstudio && $(GOMOD) verify
	@cd $(APPS_DIR)/vscode && $(GOMOD) verify
	@echo "✓ Modules verified"

##@ CI/CD

ci: lint test-unit test-race ## Run CI checks (lint + unit tests + race detector)
	@echo "✓ All CI checks passed"

pre-commit: fmt lint test-unit ## Run pre-commit checks
	@echo "✓ Pre-commit checks passed"

##@ Documentation

test-docs: ## Display testing documentation
	@echo "\n=== Testing Strategy ==="
	@echo ""
	@echo "Unit Tests (make test-unit):"
	@echo "  - Fast tests (<30s total)"
	@echo "  - No AWS credentials required"
	@echo "  - No network dependencies"
	@echo "  - Struct validation, business logic"
	@echo "  - Use -short flag to skip slow tests"
	@echo ""
	@echo "Integration Tests (make test-integration):"
	@echo "  - Tests with mocked AWS services"
	@echo "  - Requires localstack or moto"
	@echo "  - Tests AWS SDK integration"
	@echo "  - Use -tags=integration"
	@echo ""
	@echo "Smoke Tests (make test-smoke):"
	@echo "  - Quick checks against real AWS"
	@echo "  - Requires valid AWS credentials"
	@echo "  - Creates minimal real resources"
	@echo "  - Use -tags=smoke"
	@echo ""
	@echo "E2E Tests (make test-e2e):"
	@echo "  - Full workflow tests"
	@echo "  - Launch → Connect → Terminate"
	@echo "  - Long-running (up to 30 minutes)"
	@echo "  - Use -tags=e2e"
	@echo ""
	@echo "Regression Tests (make test-regression):"
	@echo "  - Unit + Integration tests"
	@echo "  - Run before releases"
	@echo "  - No E2E (faster feedback)"
	@echo ""
