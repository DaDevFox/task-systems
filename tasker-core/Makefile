.PHONY: build test clean proto deps run help lint build-client build-enhanced run-enhanced demo-enhanced

# Default target
help:
	@echo "Available targets:"
	@echo "  deps              - Install dependencies"
	@echo "  proto             - Generate protobuf code"
	@echo "  build             - Build the original server binary"
	@echo "  build-client      - Build the original client binary"
	@echo "  build-enhanced    - Build the enhanced server binary"
	@echo "  build-enhanced-client - Build the enhanced client binary"
	@echo "  test              - Run tests"
	@echo "  lint              - Run linters"
	@echo "  run               - Run the original server"
	@echo "  run-enhanced      - Run the enhanced server"
	@echo "  demo-enhanced     - Run enhanced demo workflow"
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

# Build the original server
build: proto
	go build -o bin/task-server.exe ./cmd/server

# Build the original client
build-client: proto
	go build -o bin/task-client.exe ./cmd/client

# Build the enhanced server
build-enhanced: proto
	go build -o bin/enhanced-server.exe ./cmd/enhanced_server

# Build the enhanced client
build-enhanced-client: proto
	go build -o bin/enhanced-client.exe ./cmd/enhanced_client

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linters
lint:
	go vet ./...
	go fmt ./...

# Run the original server
run: build
	./bin/task-server.exe

# Run the enhanced server with environment variables
run-enhanced: build-enhanced
	@echo "Starting Enhanced Task Server..."
	@echo "Set these environment variables for full functionality:"
	@echo "  GOOGLE_CLIENT_ID=your_google_client_id"
	@echo "  GOOGLE_CLIENT_SECRET=your_google_client_secret"
	@echo "  EMAIL_USERNAME=your_gmail@gmail.com"
	@echo "  EMAIL_PASSWORD=your_app_password"
	@echo "  EMAIL_FROM=your_gmail@gmail.com"
	@echo "  AUTO_SYNC_ENABLED=true"
	./bin/enhanced-server.exe

# Demo the enhanced features
demo-enhanced: build-enhanced-client
	@echo "Enhanced Demo Workflow"
	@echo "======================"
	@echo "Adding a task for user1..."
	./bin/enhanced-client.exe --user user1 add "Review Code" "Review the new feature implementation"
	@echo ""
	@echo "Adding another task..."
	./bin/enhanced-client.exe --user user1 add "Write Tests" "Write unit tests for the new feature"
	@echo ""
	@echo "Listing pending tasks..."
	./bin/enhanced-client.exe --user user1 list pending
	@echo ""
	@echo "Showing DAG view..."
	./bin/enhanced-client.exe --user user1 view
	@echo ""
	@echo "Showing user information..."
	./bin/enhanced-client.exe --user user1 user show

# Demo calendar sync (requires setup)
demo-sync: build-enhanced-client
	@echo "Syncing with Google Calendar..."
	./bin/enhanced-client.exe --user user1 sync

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
demo-add-task: build-client
	./bin/task-client.exe -cmd add -name "Example Task" -desc "This is an example task"

demo-list-pending: build-client
	./bin/task-client.exe -cmd list -stage pending

demo-workflow: build-client
	@echo "Running demo workflow..."
	./bin/task-client.exe -cmd add -name "Demo Task" -desc "A task for demonstration"
	@echo "Task added. Moving to staging..."
	@timeout /t 1 >nul
	./bin/task-client.exe -cmd list -stage pending

# Development pipeline
dev: clean proto build build-client test
	@echo "Development build complete!"
