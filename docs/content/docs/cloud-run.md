---
title: "Cloud Run Deployment"
weight: 4
---

# ☁️ Google Cloud Run Deployment

Deploying AI Daily Brief as a **Google Cloud Run** service allows other agent services (such as **Vertex AI Agent Engine** or Cloud Run microservices) to call it as an intelligent control plane.

---

## 1. Container Building via Bazel

Compile the pure Linux ARM64 or AMD64 container binaries directly using Bazel:

```bash
# Build Linux x86_64 Cloud Run binary
bazel build //cmd/mcp-server:mcp_server_linux_amd64

# Build Linux ARM64 Cloud Run binary
bazel build //cmd/mcp-server:mcp_server_linux_arm64
```

---

## 2. Dockerfile Configuration

```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY bazel-bin/cmd/mcp-server/mcp_server_linux_amd64_/mcp_server_linux_amd64 /app/server
COPY .env.integration.toml /app/.env.toml
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
```

---

## 3. Deploying with gcloud

```bash
gcloud run deploy ai-daily-brief-mcp \
  --image gcr.io/YOUR_PROJECT_ID/ai-daily-brief-mcp:latest \
  --platform managed \
  --region us-central1 \
  --set-env-vars ALLOYDB_DATABASE_URL="postgres://postgres:PASS@ALLOYDB_IP:5432/ai_daily_brief?sslmode=require" \
  --set-env-vars GOOGLE_CLOUD_PROJECT="YOUR_PROJECT_ID" \
  --service-account "ai-daily-brief-sa@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
  --allow-unauthenticated
```

---

## 4. Container Probes

- **Liveness Probe**: `GET /healthz`
- **Readiness Probe**: `GET /healthz`
