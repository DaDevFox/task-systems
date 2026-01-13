# Build stage
FROM golang:1.24-alpine AS builder

# Install dependencies for building
RUN apk add --no-cache git make netcat-openbsd

# Install buf for protobuf generation
RUN wget -O /tmp/buf https://github.com/bufbuild/buf/releases/latest/download/buf-Linux-x86_64 && \
    chmod +x /tmp/buf && \
    mv /tmp/buf /usr/local/bin/buf

# Install protobuf compiler
RUN apk add --no-cache protobuf protobuf-dev

# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download
WORKDIR /app

# Copy source code
COPY . .

# Generate protobuf code using buf
RUN buf generate

# Build server and client
RUN go build -o bin/server ./backend/cmd/server
RUN go build -o bin/client ./backend/cmd/client

# Runtime stage
FROM alpine:3.20

# Install ca-certificates for HTTPS and netcat for healthcheck, create non-root user
RUN apk add --no-cache ca-certificates netcat-openbsd && \
    addgroup -g 1001 -S taskuser && \
    adduser -u 1001 -S taskuser -G taskuser

# Set working directory
WORKDIR /app

# Copy binaries from build stage
COPY --from=builder /app/bin/server ./task-server
COPY --from=builder /app/bin/client ./task-client

# Change ownership to non-root user
RUN chown -R taskuser:taskuser /app

# Switch to non-root user
USER taskuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 8080 || exit 1

# Run the server
CMD ["./task-server", "--port", "8080"]
