---
title: "Google Cloud AlloyDB"
weight: 5
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
alloydb_url = "postgres://postgres:SECURE_PASSWORD@10.0.0.1:5432/ai_daily_brief?sslmode=require"

[gemini]
model = "gemini-3.7-flash"
auth_mode = "vertex_adc"
vertex_project_id = "retail-cortex-prod"
vertex_location = "us-central1"
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
