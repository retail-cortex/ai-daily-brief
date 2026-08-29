# Copyright 2026 Retail Cortex
#
# Multi-stage Dockerfile for AI Daily Brief MCP Server on Cloud Run

# Stage 1: Build binary with Go
FROM golang:1.26-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/mcp-server \
    ./cmd/mcp-server

# Stage 2: Production Distroless Runtime
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/mcp-server /app/mcp-server

# Copy base and environment configurations (excluding developer local overrides)
COPY configs/mcp/example-dev.toml /app/configs/mcp/.env.dev.toml
COPY configs/mcp/example-prod.toml /app/configs/mcp/.env.prod.toml
COPY configs/mcp/example-integration.toml /app/configs/mcp/.env.integration.toml

ENV PORT=8080
ENV MODENV_PREFIX=/app/configs/mcp

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/mcp-server"]
