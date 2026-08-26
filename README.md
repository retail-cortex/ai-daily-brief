# ⚡ AI Daily Brief (Cloud Run MCP & REST Server)

A high-performance **Model Context Protocol (MCP)** and **REST Control Plane Server** powered by **Go, Gin, GORM, Google Cloud AlloyDB for PostgreSQL, and Vertex AI**.

It aggregates frontier model announcements, arXiv research papers, AI venture capital deals, GPU datacenter infrastructure, and open-source tooling into structured **A2UI (Agent-to-UI)** card decks, executive briefings, and automated daily digests.

---

## 🎯 Architecture & Key Features

- **Google Cloud AlloyDB for PostgreSQL Backend**: Enterprise-grade database storage with connection pooling and high-throughput query caching. Falls back seamlessly to SQLite for local development and unit testing.
- **Vertex AI ADC & Gemini 3.7 Control Plane**:
  - **Native ADC Integration**: Authenticates directly via Application Default Credentials (`gcloud auth application-default login` or Cloud Run metadata server) with zero API keys required.
  - **Interactive Research Agent**: Context-grounded dialogue with real-time web scraping and paper extraction.
  - **Automated Daily TL;DR**: Strategic 3-tier executive briefings across all 5 streams.
- **Model Context Protocol (MCP) + A2UI Formatter**:
  - Exposes standard JSON-RPC 2.0 tools with interactive **A2UI card decks** and action buttons (`[⚡ Load Article Context: <id>]`).
  - Compatible with **Vertex AI Agent Engine**, Cloud Run inter-agent callers, Antigravity, Claude Desktop, and Cursor.
- **5 Intelligence Streams**:
  - 🔵 **Frontier Models**: Google (Gemini 3.7 / 2.0), Anthropic (Claude), OpenAI (GPT-5 / o3 / Sora), X AI (Grok), Meta AI.
  - ☁️ **Google Cloud**: Official release notes, Vertex AI infrastructure, AI Hypercomputer, TPUs, and GKE.
  - 🟣 **AI Research Papers**: Multi-category arXiv API (`cs.CL`, `cs.AI`, `cs.CV`, `cs.LG`), Hugging Face Daily Papers, and academic benchmarks.
  - 🟢 **AI Business & Infra**: Funding rounds, datacenter buildouts (Nvidia Blackwell, Colossus), hyperscaler cloud deals.
  - 🟠 **OSS & Tooling**: Open weights (DeepSeek, Llama, Qwen, Mistral), local inference runtimes (vLLM, Ollama), fine-tuning harnesses.
- **Sub-Second Parallel Crawling**: Concurrent goroutines crawl all sources in parallel in **< 1 second** (`~900ms`) with SHA-256 deduplication.

---

## 🚀 Quick Start (Running via Bazel)

```bash
# Run the unified MCP & REST server locally on port 8080
bazel run //:run -- -port 8080

# Or connect to Google Cloud AlloyDB / PostgreSQL directly:
bazel run //:run -- -dsn "postgres://user:pass@alloydb-ip:5432/ai_daily_brief?sslmode=disable"
```

---

## 📡 Cloud Run Deployment & Endpoints

The server binds to `$PORT` (default `8080`) and exposes:

- `GET  /healthz` - Cloud Run container readiness & liveness probe.
- `POST /mcp` - Direct MCP JSON-RPC 2.0 endpoint.
- `GET  /sse` & `POST /message` - MCP Server-Sent Events (SSE) streaming transport.
- `GET  /api/items` - Paginated and filtered intelligence item queries.
- `POST /api/batch/run` - Trigger parallel crawler batch execution.
- `GET  /api/agent/models` - Discover available Gemini & Vertex AI models.
- `POST /api/agent/chat` - Context-grounded interactive LLM chat.
- `POST /api/agent/tldr` - Executive daily strategic synthesis.
- `GET  /ws/live` - Multimodal Live Bidi WebSocket endpoint.

---

## 🛠️ Supported MCP Tools

| Tool | Description | A2UI Output |
| :--- | :--- | :--- |
| `list_articles` | List indexed items filtered by category, company, or query. | Formatted A2UI article card deck with `[⚡ Load Article Context]` action buttons. |
| `get_article_context` | Deep-fetches and extracts full webpage body text. | Complete grounding inspector card with source metadata and suggested prompts. |
| `generate_tldr` | Generates strategic 3-tier executive briefing via Vertex AI ADC. | Structured markdown briefing card. |
| `trigger_crawl` | Runs live sub-second crawl across all 5 streams. | Batch crawler telemetry card with deduplication metrics. |
| `get_newsletter` | Retrieves today's formatted daily intelligence digest. | Executive markdown newsletter view. |
| `agent_chat` | Grounded interactive research chat with Gemini 3.7. | Dialogue card with source citations. |
| `get_system_status` | Inspects database count, active model, and auth mode. | Telemetry card. |

---

## 📁 Directory Structure

```text
├── cmd/
│   └── mcp-server/       # Dedicated Cloud Run MCP Server & REST entrypoint
├── internal/
│   ├── agent/            # Gemini & Vertex AI ADC client
│   ├── config/           # Environment & TOML config loader
│   ├── crawler/          # Parallel goroutine news crawlers
│   ├── database/         # AlloyDB / PostgreSQL & SQLite GORM layer
│   ├── mailer/           # HTML email digest builder
│   ├── mcp/              # MCP JSON-RPC protocol, REST APIs & A2UI engine
│   └── security/         # Stable AES-256 key management
├── BUILD.bazel           # Root Bazel build definitions
├── MODULE.bazel          # Bazel 9 Bzlmod dependency configuration
└── BUILD.md              # Hermetic Bazel compilation guide
```
