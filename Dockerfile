# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build backend
RUN CGO_ENABLED=1 GOOS=linux go build -o proxyd ./cmd/proxyd

# Frontend stage
FROM node:20-alpine AS frontend-builder

WORKDIR /web-ui
COPY web-ui/package*.json ./
RUN npm install
COPY web-ui/ ./
RUN npm run build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache sqlite-libs ca-certificates

# Set working directory
WORKDIR /opt/proxyd

# Copy binary and frontend
COPY --from=builder /build/proxyd /opt/proxyd/bin/
COPY --from=frontend-builder /web-ui/dist /opt/proxyd/web-ui/

# Create directories
RUN mkdir -p /opt/proxyd/data/db /opt/proxyd/logs

# Copy example config
COPY config.example.yaml /opt/proxyd/config.yaml.example

# Set permissions
RUN chmod +x /opt/proxyd/bin/proxyd

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run
CMD ["/opt/proxyd/bin/proxyd", "-c", "/opt/proxyd/config.yaml"]
