# Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache gcc musl-dev git

# Enable Go Modules
ENV GO111MODULE=on

# Copy module files first for efficient caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=1 GOOS=linux go build -o garminsync .

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/garminsync /app/garminsync

# Copy templates
COPY internal/web/templates ./internal/web/templates

# Set timezone and environment
ENV TZ=UTC \
    DATA_DIR=/data \
    DB_PATH=/data/garmin.db \
    TEMPLATE_DIR=/app/internal/web/templates

# Create data volume and set permissions
RUN mkdir /data && chown nobody:nobody /data
VOLUME /data

# Run as non-root user
USER nobody

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=30s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8888/health || exit 1

# Expose web port
EXPOSE 8888

# Start the application
ENTRYPOINT ["/app/garminsync"]
