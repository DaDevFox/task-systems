# Build stage
FROM golang:1.24-alpine AS builder

RUN  test MONOREPO_ROOT  || { echo "ERROR: Build context must be monorepo root (run docker build -f inventory-core/backend/Dockerfile -t [tag] .)" ; exit 1; }

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev netcat-openbsd
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Set working directory
WORKDIR /app

# Generate protos if missing (for CI/local dev parity)
RUN if [ ! -d inventory-core/backend/pkg/proto/inventory/v1 ] || [ ! -d shared/pkg/proto/events/v1 ]; then \
    echo "Generating protos using PowerShell script..." && \
    apk add --no-cache powershell && \
    pwsh ./generate-proto.ps1 -Verbose; \
  fi

# Copy source code (generated proto ource in pkg/ included)
COPY . .

RUN test -d shared || { echo "ERROR: Build context must be monorepo root (run docker build -f inventory-core/backend/Dockerfile -t [tag] .)" ; exit 1; }
COPY shared ./shared

# Build the application
WORKDIR /app/inventory-core/backend
RUN go mod tidy 
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
