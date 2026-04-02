.PHONY: all test lint sec check install-tools build run docker-up docker-down

# Default target
all: check build

# Run unit tests with race detector and coverage
test:
	@echo "==== Running tests ===="
	go test -v -race -cover ./...

# Run the golangci-lint linter (requires installation)
lint:
	@echo "==== Running linter ===="
	golangci-lint run ./...

# Run the gosec security checker (requires installation)
sec:
	@echo "==== Running security checks ===="
	gosec ./...

# Run the complete test suite (lint, sec, and unit tests)
check: lint sec test
	@echo "==== All code checks passed! ===="

# Build the main server binary
build:
	@echo "==== Building backend server ===="
	CGO_ENABLED=0 go build -o heimdall-server ./cmd/server/main.go

# Run the binary locally natively
run: build
	@echo "==== Starting backend server ===="
	./heimdall-server

# Install the necessary CI tools locally to your Go bin
install-tools:
	@echo "==== Installing golangci-lint and gosec ===="
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

# Start the application using Docker Compose (builds the image first)
docker-up: check
	@echo "==== Starting backend server with Docker Compose ===="
	docker-compose up -d --build

# Stop and remove the Docker Compose containers
docker-down:
	@echo "==== Stopping Docker Compose containers ===="
	docker-compose down
