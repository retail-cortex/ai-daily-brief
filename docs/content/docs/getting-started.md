---
title: "Getting Started"
weight: 1
---

# 🚀 Getting Started

This guide walks you through compiling, configuring, and running the **AI Daily Brief** MCP & REST Server locally using Bazel.

---

## Prerequisites

- **Bazel** (or **Bazelisk**): `brew install bazelisk`
- **Go 1.26+** (Hermetically provided by Bazel rules_go toolchain)
- **Google Cloud SDK** (`gcloud`) for Vertex AI Application Default Credentials:
  ```bash
  gcloud auth application-default login
  ```

---

## Running Locally

To build and launch the unified MCP & REST server:

```bash
# Clone the repository
git clone https://github.com/retail-cortex/ai-daily-brief.git
cd ai-daily-brief

# Run server on port 8080 (using local SQLite by default)
bazel run //:run -- -port 8080
```

Once running, the server will output:

```text
====================================================
🛡️ AI Daily Brief MCP Server (Cloud Run Ready)
🚀 Listening on http://0.0.0.0:8080
📡 Direct JSON-RPC: POST http://0.0.0.0:8080/mcp
🌊 SSE Stream:      GET  http://0.0.0.0:8080/sse
🩺 Health Probe:    GET  http://0.0.0.0:8080/healthz
====================================================
```

---

## Environment Profiles

Configuration is powered by **`modenv`**, enabling seamless switching between development, testing, and production environments:

| Environment | Config File | Database Engine | Auth Mode |
| :--- | :--- | :--- | :--- |
| **Default** | `.env.toml` | Local SQLite (`data/ai_daily_brief.db`) | Vertex AI ADC |
| **Testing** | `.env.test.toml` | In-Memory SQLite (`:memory:`) | Vertex AI ADC |
| **Integration / Prod** | `.env.integration.toml` | Google Cloud AlloyDB / PostgreSQL | Vertex AI ADC |

### Activating an Environment Profile

```bash
# Run with integration profile (AlloyDB)
ENV=integration bazel run //:run

# Run with test profile
ENV=test bazel run //:run
```

---

## Command-Line Options

The `mcp-server` binary accepts the following flags:

```text
Usage of mcp-server:
  -port string
        Port to listen on (defaults to $PORT or 8080 for Cloud Run)
  -db string
        Path to SQLite database file or AlloyDB/PostgreSQL connection string
  -dsn string
        Explicit Google Cloud AlloyDB / PostgreSQL DSN
  -stdio
        Run in stdio mode for local desktop MCP clients (Antigravity/Claude/Cursor)
```
