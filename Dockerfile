# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev} -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o volcano-job-analyzer .

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S analyzer && \
    adduser -u 1001 -S analyzer -G analyzer

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/volcano-job-analyzer .

# Copy example configurations
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/examples ./examples

# Change ownership
RUN chown -R analyzer:analyzer /app

# Switch to non-root user
USER analyzer

# Expose port (if needed for future web interface)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./volcano-job-analyzer --version || exit 1

# Default command
ENTRYPOINT ["./volcano-job-analyzer"]
CMD ["--help"]