# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY backend/go.mod backend/go.sum ./backend/

# Download dependencies
RUN go mod download
RUN cd backend && go mod download

# Copy source code
COPY . .

# Generate protobuf code
RUN apk add --no-cache bash curl
RUN curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Linux-x86_64" -o "/usr/local/bin/buf"
RUN chmod +x /usr/local/bin/buf
RUN buf generate

# Build the application
RUN cd backend && go build -ldflags="-w -s" -o ../bin/user-core-server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S usercore && adduser -u 1001 -S usercore -G usercore

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
  CMD nc -z localhost $PORT || exit 1

# Run the application
CMD ["./user-core-server"]
