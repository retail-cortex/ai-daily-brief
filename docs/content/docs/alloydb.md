---
title: "Google Cloud AlloyDB"
weight: 7
---

# 💾 Google Cloud AlloyDB for PostgreSQL

AI Daily Brief uses **Google Cloud AlloyDB for PostgreSQL** as its primary production data tier for enterprise scalability, fast transactional indexing, and connection pooling.

---

## Connection Configuration

The database connection is configured via `ALLOYDB_DATABASE_URL`, `DATABASE_URL`, or `alloydb_url` in `.env.integration.toml`:

```toml
# .env.integration.toml
port = "8080"
cron_schedule = "0 8 * * *"
alloydb_url = "postgres://aibrief:YOUR_PASSWORD@10.0.0.1:5432/ai_daily_brief?sslmode=require"

[gemini]
model = "gemini-3.7-flash"
auth_mode = "vertex_adc"
vertex_project_id = "YOUR_PROJECT_ID"
vertex_location = "us-central1"
```

---

## Connecting Locally with AlloyDB Auth Proxy

To securely access the private AlloyDB instance from your local workstation:

1. **Launch the proxy**:
   ```bash
   alloydb-auth-proxy \
     "projects/${PROJECT_ID}/locations/us-central1/clusters/ai-daily-brief-alloydb-dev/instances/ai-daily-brief-alloydb-dev-primary" \
     --port=5432
   ```

2. **Run MCP server locally against the proxy**:
   ```bash
   export ALLOYDB_DATABASE_URL="postgres://postgres:YOUR_PASSWORD@127.0.0.1:5432/ai_daily_brief?sslmode=disable"
   bazel run //cmd/mcp-server
   ```

---

## Connection Pooling Settings

To maximize throughput under high concurrent agent calls, connection pooling is automatically applied:

- **Max Open Connections**: `25`
- **Max Idle Connections**: `10`
- **Max Connection Lifetime**: `15 minutes`

---

## Schema & Tables

All tables auto-migrate on startup using GORM:

- `news_items`: Stored intelligence items with SHA-256 deduplication index on `link`.
- `subscribers`: Email newsletter subscribers.
- `run_logs`: Crawler batch execution records and logs.
- `settings`: Key-value dynamic settings (encrypted API keys, cron schedules, latest TL;DR).
- `chat_messages`: Multi-turn conversational research history.
