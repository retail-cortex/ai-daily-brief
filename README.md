# ⚡ AI Daily Brief

[![Documentation](https://img.shields.io/badge/docs-GitHub_Pages-blue.svg)](https://retail-cortex.github.io/ai-daily-brief/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Bazel](https://img.shields.io/badge/build-Bazel_9-green.svg)](https://bazel.build)

A high-performance **Model Context Protocol (MCP)** control plane and **Agent-to-Agent (A2A)** intelligence system powered by **Go, Gin, GORM, Google Cloud AlloyDB for PostgreSQL, and Google Vertex AI**.

It continuously aggregates and synthesizes frontier model announcements, arXiv research papers, AI venture capital deals, GPU datacenter infrastructure, and open-source tooling into structured **A2UI (Agent-to-UI)** card decks, executive briefings, and automated daily digests.

---

## 📖 Complete Documentation

For detailed architecture guides, MCP tool specifications, Cloud Run deployment manifests, and REST API references, visit the **official documentation site**:

👉 **[https://retail-cortex.github.io/ai-daily-brief/](https://retail-cortex.github.io/ai-daily-brief/)**

### Quick Links to Documentation
- 🚀 **[Getting Started](https://retail-cortex.github.io/ai-daily-brief/docs/getting-started/)**: Quickstart, running locally with Bazel, and environment profile switching (`.env.test.toml` vs `.env.integration.toml`).
- 🏛️ **[System Architecture](https://retail-cortex.github.io/ai-daily-brief/docs/architecture/)**: Parallel goroutine crawler (<900ms execution), SHA-256 deduplication, dynamic HTML extraction, and dual protocol plane.
- 🤖 **[Model Context Protocol (MCP) & A2UI](https://retail-cortex.github.io/ai-daily-brief/docs/mcp-server/)**: JSON-RPC 2.0 tool definitions, A2UI card schemas, and Server-Sent Events (SSE) streaming.
- 🤖 **[Agent-to-Agent (A2A) Service](https://retail-cortex.github.io/ai-daily-brief/docs/a2a-agent/)**: Autonomous Cloud Run agent service consuming the MCP control plane with Google ADK / Gemini workflows.
- 🌐 **[Gemini Enterprise Integration](https://retail-cortex.github.io/ai-daily-brief/docs/gemini_enterprise/)**: Cloud Run IAM service identity bindings, Discovery Engine setup, and Agent Card publishing.
- ☁️ **[Google Cloud Run Deployment](https://retail-cortex.github.io/ai-daily-brief/docs/cloud-run/)**: Cross-compiling Linux container binaries, Distroless images, `gcloud run deploy`, and container probes (`/healthz`).
- 💾 **[Google Cloud AlloyDB](https://retail-cortex.github.io/ai-daily-brief/docs/alloydb/)**: AlloyDB for PostgreSQL connection strings, production connection pooling, GORM auto-migrations, and environment profiles.
- 📡 **[REST API Reference](https://retail-cortex.github.io/ai-daily-brief/docs/rest-api/)**: Complete endpoint reference for all HTTP routes and the Multimodal Live Bidi WebSocket (`/ws/live`).

---

## 🚀 Quick Start (Running via Bazel)

```bash
# 1. Run all 4 services concurrently in parallel (Dev Profile)
# (Starts Hugo Docs on :1313, Test App on :8888, MCP Server on :8080, A2A Agent on :8081)
bazel run //:dev

# Or run individual services:
bazel run //:run           # MCP & REST Server (Port 8080)
bazel run //:run-agent     # A2A Agent (Port 8081)
bazel run //:run-test-app  # A2A Diagnostic Test App (Port 8888)
bazel run //:serve-docs    # Hugo Documentation Site (Port 1313)
```

---

## 📁 Repository Layout

```text
├── cmd/
│   ├── a2a-agent/        # Cloud Run Agent-to-Agent (A2A) service entrypoint
│   └── mcp-server/       # Cloud Run MCP & REST control plane entrypoint
├── configs/
│   ├── agent/            # A2A agent configuration profiles (modenv)
│   └── mcp/              # MCP server configuration profiles (modenv)
├── docs/                 # Static documentation site powered by rules_hugo
├── internal/
│   ├── a2a/              # A2A agent engine, MCP client, and HTTP server
│   ├── agent/            # Gemini 3.7 & Vertex AI ADC client
│   ├── config/           # Modenv environment profile loader
│   ├── crawler/          # Parallel goroutine intelligence scrapers
│   ├── database/         # AlloyDB / PostgreSQL & SQLite GORM layer
│   ├── mailer/           # HTML email digest builder
│   ├── mcp/              # MCP JSON-RPC protocol, REST APIs & A2UI engine
│   └── security/         # Google Cloud Secret Manager & AES-256 encryption
├── BUILD.bazel           # Root Bazel build definitions & aliases
└── MODULE.bazel          # Bazel 9 Bzlmod dependency configuration
```

---

## 📄 License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.
