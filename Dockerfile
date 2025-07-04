FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go && \
    chmod +x main

# Final stage
FROM alpine:latest

# Install ca-certificates for SSL/TLS and wget for health check
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory and copy binary
WORKDIR /app
COPY --from=builder /app/main .
RUN chmod +x main && chown appuser:appuser main

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["./main"]
