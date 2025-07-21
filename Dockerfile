# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24.2-alpine AS builder

# Install git and build tools
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for multiple platforms
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Set build arguments for cross-compilation
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
        GOOS=linux GOARCH=amd64; \
    elif [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
        GOOS=linux GOARCH=arm64; \
    elif [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then \
        GOOS=linux GOARCH=arm GOARM=7; \
    else \
        GOOS=linux GOARCH=amd64; \
    fi && \
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Create necessary directories and set permissions
RUN mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"] 