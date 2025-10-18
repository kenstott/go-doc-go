# Multi-stage Dockerfile for Go application
# Build stage
FROM --platform=linux/amd64 golang:1.21-alpine AS builder

# Set build arguments
ARG VERSION=1.0.0
ARG BUILD_DATE

# Add metadata labels
LABEL maintainer="devops@example.com"
LABEL version="${VERSION}"
LABEL description="Example microservice application"

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
    -o /app/bin/server ./cmd/server

# Runtime stage
FROM alpine:3.18 AS runtime

# Set runtime arguments
ARG USER_ID=1001
ARG GROUP_ID=1001

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g ${GROUP_ID} appgroup && \
    adduser -D -u ${USER_ID} -G appgroup appuser

# Set working directory
WORKDIR /home/appuser

# Copy built binary from builder stage
COPY --from=builder /app/bin/server ./server

# Copy configuration files
COPY --from=builder /app/config ./config

# Set environment variables
ENV APP_ENV=production
ENV LOG_LEVEL=info
ENV SERVER_PORT=8080

# Expose application port
EXPOSE 8080

# Health check port
EXPOSE 9090

# Switch to non-root user
USER appuser

# Set entrypoint
ENTRYPOINT ["./server"]

# Default command arguments
CMD ["--config", "config/prod.yaml"]
