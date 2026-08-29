# Copyright 2026 Retail Cortex
#
# Multi-stage Dockerfile for AI Daily Brief A2A Agent on Cloud Run

# Stage 1: Build binary with Go
FROM golang:1.26-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/a2a-agent \
    ./cmd/a2a-agent

# Stage 2: Production Distroless Runtime
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/a2a-agent /app/a2a-agent

# Copy base and environment configurations
COPY configs/agent/example-dev.toml /app/configs/agent/.env.dev.toml
COPY configs/agent/example-prod.toml /app/configs/agent/.env.prod.toml
COPY configs/agent/example-integration.toml /app/configs/agent/.env.integration.toml

ENV PORT=8080
ENV MODENV_PREFIX=/app/configs/agent

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/a2a-agent"]
