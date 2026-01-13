# Base image for Go application
FROM golang:1.19 AS base

# Set the working directory
WORKDIR /app

# Install buf for protocol buffer generation
RUN curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.0.0/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf \
    && chmod +x /usr/local/bin/buf

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies to leverage Docker layer caching
RUN go mod download

# Copy the entire application source
COPY . .

# Run protobuf generation script as part of build phase
RUN pwsh ./generate-proto.ps1

# Build the application
RUN go build -o user-core ./cmd/user-core

# Create minimal runtime image
FROM gcr.io/distroless/base-debian11

# Set the working directory
WORKDIR /app

# Copy built binary to runtime image
COPY --from=base /app/user-core /app/user-core

# Expose necessary ports
EXPOSE 8080

# Command to run the Go application
CMD ["/app/user-core"]