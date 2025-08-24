# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files and download dependencies
# Copy root go.mod first
COPY go.mod go.sum ./
RUN go mod download


# Copy source code
COPY . .

# Generate protobuf code
RUN apk add --no-cache protobuf protobuf-dev
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4
RUN buf generate

# Build the application
RUN go build -ldflags="-w -s" -o /app/bin/inventory-service ./cmd/server

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies and create non-root user
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/inventory-service .

# Create data directory and set permissions
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 50052

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD nc -z localhost 50052 || exit 1

# Run the application
CMD ["./inventory-service"]
