# Frontend build stage
FROM node:20-alpine AS frontend
WORKDIR /web-ui
COPY web-ui/package*.json ./
RUN npm ci
COPY web-ui/ ./
RUN npm run build

# Backend build stage (Go version tracks go.mod)
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed the freshly built web UI so the -tags webui go:embed succeeds.
COPY --from=frontend /web-ui/dist ./internal/webui/dist
RUN CGO_ENABLED=1 GOOS=linux go build -tags webui -trimpath -ldflags "-s -w" -o proxyd ./cmd/proxyd

# Runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates wget
WORKDIR /opt/proxyd
COPY --from=builder /build/proxyd /opt/proxyd/bin/proxyd
COPY config.example.yaml /opt/proxyd/config.yaml.example
# Seed a default config so the container runs out of the box; mount your own
# /opt/proxyd/config.yaml (with a strong jwt_secret) for production.
RUN mkdir -p /opt/proxyd/data/db /opt/proxyd/data/mihomo /opt/proxyd/logs \
 && cp /opt/proxyd/config.yaml.example /opt/proxyd/config.yaml

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# -web serves the embedded UI (the binary is built with -tags webui above).
CMD ["/opt/proxyd/bin/proxyd", "-c", "/opt/proxyd/config.yaml", "-web"]
