.PHONY: help build run clean

help:
	@echo "Available targets:"
	@echo "  make build      - Build all services"
	@echo "  make run-web    - Run Next.js frontend"
	@echo "  make run-gateway - Run gateway service"
	@echo "  make run-collector - Run collector service"
	@echo "  make run-ai-processor - Run AI processor service"
	@echo "  make clean      - Clean build artifacts"

build:
	@echo "Building services..."

run-web:
	cd web && npm run dev

run-gateway:
	cd backend/gateway && go run main.go

run-collector:
	cd backend/collector && go run main.go

run-ai-processor:
	cd backend/ai-processor && go run main.go

clean:
	rm -rf web/.next
	rm -rf web/node_modules
	go clean -cache
