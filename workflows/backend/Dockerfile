# Build stage
FROM golang:1.24-alpine AS builder

RUN  test MONOREPO_ROOT  || { echo "ERROR: Build context must be monorepo root (run docker build -f inventory-core/backend/Dockerfile -t [tag] .)" ; exit 1; }

# Install build dependencies
RUN apk add --no-cache git make
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Generate protos if missing (for CI/local dev parity)
RUN if [ ! -d inventory-core/backend/pkg/proto/inventory/v1 ] || [ ! -d shared/pkg/proto/events/v1 ]; then \
    echo "Generating protos using PowerShell script..." && \
    apk add --no-cache powershell && \
    pwsh ./generate-proto.ps1 -Verbose; \
  fi

RUN test -d shared || { echo "ERROR: Build context must be monorepo root (run docker build -f inventory-core/backend/Dockerfile -t [tag] .)" ; exit 1; }

# Build the application
WORKDIR /app/workflows/backend
RUN go mod tidy
RUN go build -ldflags="-w -s" -o /app/bin/workflows-backend

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies and create a non-root user
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/bin/workflows-backend .

# Copy templates
COPY --from=builder /app/workflows/backend/templates ./templates/

# Chown the directory to the new user
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

CMD ["./workflows-backend"]
