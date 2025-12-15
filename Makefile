.PHONY: help build run dev clean test deps

help: ## Display this help screen
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/server cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

dev: ## Run in development mode with hot reload (requires air)
	air

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

fmt: ## Format code
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

# ---------------- Docker ----------------

DOCKER_USER := your-dockerhub-username
IMAGE_NAME := uptime-monitor
IMAGE_TAG ?= latest
IMAGE := $(DOCKER_USER)/$(IMAGE_NAME):$(IMAGE_TAG)

docker-build: ## Build Docker image
	docker build -t $(IMAGE) .

docker-run: ## Run Docker container
	docker run -p 5000:5000 --env-file .env $(IMAGE)

docker-push: ## Push Docker image to Docker Hub
	docker push $(IMAGE)

docker-release: docker-build docker-push ## Build and push Docker image
	@echo "Docker image $(IMAGE) pushed successfully."