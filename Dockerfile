# Multi-stage build for optimized production image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version injection
ARG VERSION=unknown
ARG BUILD_TIME=unknown

# Build the application
# CGO_ENABLED=0 for static linking
# -ldflags="-s -w" for smaller binary size + version injection
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o bin/fizzbuzz \
    ./cmd/api

# Production stage - minimal image
FROM alpine:latest

# Build arguments for metadata labels
ARG VERSION=unknown
ARG BUILD_TIME=unknown

# Add version metadata labels
LABEL version="${VERSION}"
LABEL build_time="${BUILD_TIME}"
LABEL maintainer="Victor"
LABEL description="FizzBuzz API - Production ready containerized service"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S fizzbuzz && \
    adduser -S fizzbuzz -u 1001 -G fizzbuzz

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/fizzbuzz /app/fizzbuzz

# Change ownership to non-root user
RUN chown fizzbuzz:fizzbuzz /app/fizzbuzz

# Switch to non-root user
USER fizzbuzz

# Expose port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:4000/v1/healthcheck || exit 1

# Run the application
CMD ["/app/fizzbuzz"]