# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o jcrawl .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies: PostgreSQL client, Chrome/Chromium, ca-certificates
RUN apk add --no-cache \
    postgresql-client \
    chromium \
    ca-certificates \
    tzdata

# Copy binary from builder
COPY --from=builder /app/jcrawl .

# Create app directory for data
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./jcrawl"]
