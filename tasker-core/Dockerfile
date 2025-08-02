# Build stage
FROM golang:1.24-alpine AS builder

# Install dependencies for building
RUN apk add --no-cache git make

# Install buf
RUN wget -O /tmp/buf https://github.com/bufbuild/buf/releases/latest/download/buf-Linux-x86_64 && \
    chmod +x /tmp/buf && \
    mv /tmp/buf /usr/local/bin/buf

# Install protobuf compiler
RUN apk add --no-cache protobuf protobuf-dev

# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install protobuf generators
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy source code
COPY . .

# Generate protobuf code and build
RUN make proto && make build

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S taskuser && \
    adduser -u 1001 -S taskuser -G taskuser

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /app/bin/task-server .

# Change ownership to non-root user
RUN chown taskuser:taskuser /app/task-server

# Switch to non-root user
USER taskuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 8080 || exit 1

# Run the server
CMD ["./task-server", "-port", "8080"]
