# User-Core Service Makefile

.PHONY: build test clean proto server client install-deps

# Default target
all: proto build test

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	cd backend && go mod tidy
	go mod tidy

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	./generate-proto.ps1

# Build the server
build: proto
	@echo "Building server..."
	cd backend && go build -o ../bin/user-core-server ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	cd backend && go test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd backend && go test ./... -coverprofile=coverage.out
	cd backend && go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf pkg/proto/
	cd backend && rm -f coverage.out coverage.html

# Run the server
server: build
	@echo "Starting User-Core server..."
	./bin/user-core-server

# Development server with debug logging
dev-server: build
	@echo "Starting User-Core server in development mode..."
	DEBUG=true ./bin/user-core-server

# Format code
fmt:
	@echo "Formatting code..."
	cd backend && go fmt ./...
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	cd backend && golangci-lint run ./...

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t user-core:latest .

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Generate proto, build, and test"
	@echo "  install-deps - Install Go dependencies"
	@echo "  proto        - Generate protobuf code"
	@echo "  build        - Build the server binary"
	@echo "  test         - Run all tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  clean        - Clean build artifacts"
	@echo "  server       - Run the server"
	@echo "  dev-server   - Run server with debug logging"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Lint Go code"
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help"
