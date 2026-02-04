# ================================
# Backend Dockerfile - Go with Air (Hot Reload)
# ================================

# Development stage with hot reload
FROM golang:1.24-alpine AS development

WORKDIR /app

# Install Air for hot reload and other dependencies
RUN apk add --no-cache git make gcc musl-dev && \
    go install github.com/air-verse/air@v1.60.0

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Expose API port
EXPOSE 8080

# Use Air for hot reload
CMD ["air", "-c", ".air.toml"]

# ================================
# Production stage (optimized binary)
# ================================
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# ================================
# Production runtime
# ================================
FROM alpine:latest AS production

WORKDIR /root/

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy docs for Swagger
COPY --from=builder /app/docs ./docs

# Expose API port
EXPOSE 8080

# Run the binary
CMD ["./main"]
