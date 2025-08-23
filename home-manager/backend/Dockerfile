# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN go build -o /hometasker

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies and create a non-root user
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /hometasker .
# Copy templates
COPY --from=builder /app/templates ./templates/

# Chown the directory to the new user
RUN chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

CMD ["./hometasker"]