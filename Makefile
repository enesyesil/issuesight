.PHONY: help build build-all build-gateway build-collector build-ai-processor \
        run run-web run-gateway run-collector run-ai-processor \
        test test-go test-web lint lint-go lint-web \
        docker-build docker-up docker-down clean

# Default target
help:
	@echo "IssueSight - Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make build-all        - Build all Go services"
	@echo "  make build-gateway    - Build gateway service"
	@echo "  make build-collector  - Build collector service"
	@echo "  make build-ai-processor - Build AI processor service"
	@echo ""
	@echo "Run (development):"
	@echo "  make run-web          - Run Next.js frontend"
	@echo "  make run-gateway      - Run gateway service"
	@echo "  make run-collector    - Run collector service"
	@echo "  make run-ai-processor - Run AI processor service"
	@echo ""
	@echo "Test:"
	@echo "  make test             - Run all tests"
	@echo "  make test-go          - Run Go tests"
	@echo "  make test-web         - Run frontend tests"
	@echo ""
	@echo "Lint:"
	@echo "  make lint             - Run all linters"
	@echo "  make lint-go          - Run Go linter"
	@echo "  make lint-web         - Run frontend linter"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build     - Build Docker images"
	@echo "  make docker-up        - Start services with docker-compose"
	@echo "  make docker-down      - Stop services"
	@echo ""
	@echo "Misc:"
	@echo "  make clean            - Clean build artifacts"

# Build targets
build-all: build-gateway build-collector build-ai-processor
	@echo "All services built successfully"

build-gateway:
	@echo "Building gateway..."
	CGO_ENABLED=0 go build -o bin/gateway ./backend/gateway/main.go ./backend/gateway/server.go

build-collector:
	@echo "Building collector..."
	CGO_ENABLED=0 go build -o bin/collector ./backend/collector/main.go ./backend/collector/service.go

build-ai-processor:
	@echo "Building ai-processor..."
	CGO_ENABLED=0 go build -o bin/ai-processor ./backend/ai-processor

# Run targets (development)
run-web:
	cd web && npm run dev

run-gateway:
	go run ./backend/gateway/main.go ./backend/gateway/server.go

run-collector:
	go run ./backend/collector/main.go ./backend/collector/service.go

run-ai-processor:
	go run ./backend/ai-processor

# Test targets
test: test-go test-web
	@echo "All tests completed"

test-go:
	@echo "Running Go tests..."
	go test -v ./...

test-web:
	@echo "Running frontend tests..."
	cd web && npm test --if-present

# Lint targets
lint: lint-go lint-web
	@echo "All linting completed"

lint-go:
	@echo "Running Go linter..."
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

lint-web:
	@echo "Running frontend linter..."
	cd web && npm run lint --if-present

# Docker targets
docker-build:
	@echo "Building Docker images..."
	docker-compose -f deployments/docker-compose.yml build

docker-up:
	@echo "Starting services..."
	docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	@echo "Stopping services..."
	docker-compose -f deployments/docker-compose.yml down

# Clean target
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf web/.next
	go clean -cache
