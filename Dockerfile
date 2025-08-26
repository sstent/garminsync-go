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

# Install runtime dependencies (wget needed for healthcheck)
RUN apk add --no-cache ca-certificates tzdata wget sqlite

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/garminsync /app/garminsync

# Copy web directory (frontend assets)
COPY web ./web

# Set timezone and environment
ENV TZ=UTC \
    DATA_DIR=/data \
    DB_PATH=/data/garmin.db

# Create data volume and set permissions
RUN mkdir -p /data/activities && chown -R nobody:nobody /data
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
