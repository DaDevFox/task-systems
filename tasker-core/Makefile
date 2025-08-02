.PHONY: build test clean proto deps run help lint build-client

# Default target
help:
	@echo "Available targets:"
	@echo "  deps              - Install dependencies"
	@echo "  proto             - Generate protobuf code"
	@echo "  build             - Build the server binary"
	@echo "  build-client      - Build the client binary"
	@echo "  test              - Run tests"
	@echo "  lint              - Run linters"
	@echo "  run               - Run the server"
	@echo "  demo              - Run demo workflow"
	@echo "  clean             - Clean generated files"
	@echo "  help              - Show this help"

# Install dependencies
deps:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf code
proto:
	powershell -ExecutionPolicy Bypass -File generate-proto-simple.ps1

# Build the server
build: proto
	go build -o bin/task-server.exe ./cmd/server

# Build the client
build-client: proto
	go build -o bin/task-client.exe ./cmd/client

# Run tests
test:
	@if exist coverage.out del coverage.out 2>nul
	@if exist coverage_by_funcs.txt del coverage_by_funcs.txt 2>nul
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | powershell -Command "$$input | ForEach-Object { ($$_.Split()[$$_.Split().Length-1]) + ' ' + $$_ } | Sort-Object -Descending | ForEach-Object { $$_.Substring($$_.IndexOf(' ') + 1) }" > coverage_by_funcs.txt
	@if exist coverage.out del coverage.out 2>nul
	@echo "Coverage report saved to coverage_by_funcs.txt"

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Run linters
lint:
	go vet ./...
	go fmt ./...

# Run the original server
run: build
	./bin/task-server.exe

# Run the server with custom settings
run-dev: build
	./bin/task-server.exe -port 8080 -max-inbox-size 3

# Clean generated files
clean:
	rm -rf bin/
	rm -rf proto/taskcore/
	rm -f coverage.out coverage.html

# Install buf (if not already installed)
install-buf:
	@if ! command -v buf &> /dev/null; then \
		echo "Installing buf..."; \
		curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$$(uname -s)-$$(uname -m)" -o "$${TMPDIR:-/tmp}/buf"; \
		chmod +x "$${TMPDIR:-/tmp}/buf"; \
		sudo mv "$${TMPDIR:-/tmp}/buf" /usr/local/bin/buf; \
	fi

# Docker targets
docker-build:
	docker build -t task-core:latest .

docker-run:
	docker run -p 8080:8080 task-core:latest

# Demo targets (require server to be running)
demo: build-client
	@echo "Running demo workflow..."
	@echo "========================"
	@echo "Adding a task..."
	./bin/task-client.exe add "Review Code" -d "Review the new feature implementation"
	@echo ""
	@echo "Adding another task..."
	./bin/task-client.exe add "Write Tests" -d "Write unit tests for the new feature"
	@echo ""
	@echo "Listing pending tasks..."
	./bin/task-client.exe list -s pending
	@echo ""
	@echo "Demo complete!"

demo-add-task: build-client
	./bin/task-client.exe add "Example Task" -d "This is an example task"

demo-list-pending: build-client
	./bin/task-client.exe list -s pending

demo-workflow: build-client
	@echo "Running demo workflow..."
	./bin/task-client.exe add "Demo Task" -d "A task for demonstration"
	@echo "Task added. Moving to staging..."
	@timeout /t 1 >nul
	./bin/task-client.exe list -s pending

# Development pipeline
dev: clean proto build build-client test
	@echo "Development build complete!"
