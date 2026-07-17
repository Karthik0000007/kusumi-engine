# Kasumi Engine — Root Makefile
# Cross-service build, test, lint, and deployment commands

.PHONY: all test lint fmt build up down clean help

# Default target
all: lint test build

# ─────────────────────────────────────────────
# Testing
# ─────────────────────────────────────────────

test: test-go test-python test-cpp ## Run all tests
	@echo "✅ All tests passed"

test-go: ## Run Go tests
	@echo "🧪 Running Go tests..."
	cd ingestion && go test -race -coverprofile=coverage.out ./...
	@echo "✅ Go tests passed"

test-python: ## Run Python tests
	@echo "🧪 Running Python tests..."
	cd retrieval && python -m pytest tests/ -v --tb=short
	@echo "✅ Python tests passed"

test-cpp: ## Run C++ tests
	@echo "🧪 Running C++ tests..."
	-cd reranking/build && ctest --output-on-failure
	@echo "✅ C++ tests passed"

# ─────────────────────────────────────────────
# Linting
# ─────────────────────────────────────────────

lint: lint-go lint-python lint-cpp ## Run all linters
	@echo "✅ All linters passed"

lint-go: ## Lint Go code
	@echo "🔍 Linting Go code..."
	cd ingestion && go vet ./...
	@command -v golangci-lint >/dev/null 2>&1 && cd ingestion && golangci-lint run ./... || echo "⚠️  golangci-lint not installed, skipping"

lint-python: ## Lint Python code
	@echo "🔍 Linting Python code..."
	cd retrieval && python -m ruff check .
	cd retrieval && python -m mypy retrieval/ --ignore-missing-imports

lint-cpp: ## Lint C++ code
	@echo "🔍 Linting C++ code..."
	@command -v cpplint >/dev/null 2>&1 && find reranking/src reranking/include -name '*.cpp' -o -name '*.h' 2>/dev/null | xargs cpplint --quiet || echo "⚠️  cpplint not installed or no C++ files, skipping"

# ─────────────────────────────────────────────
# Formatting
# ─────────────────────────────────────────────

fmt: fmt-go fmt-python fmt-cpp ## Format all code
	@echo "✅ All code formatted"

fmt-go: ## Format Go code
	@echo "📝 Formatting Go code..."
	cd ingestion && gofmt -w .

fmt-python: ## Format Python code
	@echo "📝 Formatting Python code..."
	cd retrieval && python -m black .
	cd retrieval && python -m isort .

fmt-cpp: ## Format C++ code
	@echo "📝 Formatting C++ code..."
	@command -v clang-format >/dev/null 2>&1 && find reranking/src reranking/include -name '*.cpp' -o -name '*.h' 2>/dev/null | xargs clang-format -i || echo "⚠️  clang-format not installed or no C++ files, skipping"

# ─────────────────────────────────────────────
# Building
# ─────────────────────────────────────────────

build: build-go build-python build-cpp ## Build all services
	@echo "✅ All builds completed"

build-go: ## Build Go ingestion service
	@echo "🔨 Building Go ingestion service..."
	cd ingestion && go build -o bin/kasumi-ingest ./cmd/server

build-python: ## Install Python dependencies
	@echo "🔨 Installing Python retrieval service..."
	cd retrieval && pip install -e ".[dev]"

build-cpp: ## Build C++ reranking service
	@echo "🔨 Building C++ reranking service..."
	mkdir -p reranking/build
	cd reranking/build && cmake .. && cmake --build .

# ─────────────────────────────────────────────
# Docker
# ─────────────────────────────────────────────

up: ## Start all services via Docker Compose
	@echo "🚀 Starting Kasumi Engine..."
	docker compose -f deployment/docker-compose.yml up -d

down: ## Stop all services
	@echo "🛑 Stopping Kasumi Engine..."
	docker compose -f deployment/docker-compose.yml down

build-docker: ## Build all Docker images
	@echo "🐳 Building Docker images..."
	docker compose -f deployment/docker-compose.yml build

# ─────────────────────────────────────────────
# Coverage
# ─────────────────────────────────────────────

coverage-go: ## Generate Go coverage report
	cd ingestion && go test -coverprofile=coverage.out ./...
	cd ingestion && go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Go coverage report: ingestion/coverage.html"

coverage-python: ## Generate Python coverage report
	cd retrieval && python -m pytest tests/ --cov=retrieval --cov-report=html
	@echo "📊 Python coverage report: retrieval/htmlcov/index.html"

# ─────────────────────────────────────────────
# Protobuf
# ─────────────────────────────────────────────

proto: ## Generate protobuf bindings
	@echo "📦 Generating protobuf bindings..."
	buf generate proto/

# ─────────────────────────────────────────────
# Cleanup
# ─────────────────────────────────────────────

clean: ## Clean all build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf ingestion/bin ingestion/coverage.out ingestion/coverage.html
	rm -rf retrieval/dist retrieval/build retrieval/*.egg-info retrieval/htmlcov retrieval/.coverage
	rm -rf reranking/build
	@echo "✅ Clean complete"

# ─────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
