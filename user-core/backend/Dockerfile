# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download
WORKDIR /app

# Copy source code
COPY . .

# Generate protobuf code
RUN apk add --no-cache bash
ADD "https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Linux-x86_64" "/usr/local/bin/buf"
RUN chmod +x /usr/local/bin/buf
RUN buf generate

# Build the application
RUN go build -ldflags="-w -s" -o /app/bin/user-core-server ./backend/cmd/server

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies and create non-root user
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 -S usercore && adduser -u 1001 -S usercore -G usercore

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/user-core-server .

# Create data directory for BadgerDB (if used)
RUN mkdir -p /app/data && chown -R usercore:usercore /app

# Switch to non-root user
USER usercore

# Set environment variables
ENV PORT=50051
ENV DB_PATH=/app/data/users.db

# Expose port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD nc -z localhost "$PORT" || exit 1

# Run the application
CMD ["./user-core-server"]
