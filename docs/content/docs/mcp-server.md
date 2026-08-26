---
title: "MCP Server & A2UI"
weight: 3
---

# 🤖 Model Context Protocol (MCP) & A2UI

AI Daily Brief implements the official **Model Context Protocol (2024-11-05)** specification. It formats responses using **Agent-to-UI (A2UI)** structured cards that embed visual badges, metadata, and interactive context-loading action triggers.

---

## Supported Tools

### 1. `list_articles`
Lists indexed AI, Cloud, and LLM news items with A2UI formatted cards.

**Arguments:**
- `category` *(string, optional)*: Filter by stream (e.g. `"Frontier Models"`, `"Google Cloud"`, `"AI Research Papers"`).
- `company` *(string, optional)*: Filter by company (e.g. `"Google"`, `"Anthropic"`, `"OpenAI"`).
- `query` *(string, optional)*: Keyword query across title and summary.
- `limit` *(integer, optional)*: Number of articles to return (default: `10`, max: `50`).

**A2UI Card Output Example:**
```text
┌─────────────────────────────────────────────────────────────
│ **#1: Google Unveils Gemini 3.7 Flash Hybrid Reasoning**
│ ⚡ Frontier Models • **Google** • *Tue Feb 25, 14:00 CST*
├─────────────────────────────────────────────────────────────
│ Google announces Gemini 3.7 Flash with dynamic reasoning depth and hybrid speed capabilities.
├─────────────────────────────────────────────────────────────
│ ⚡ **Actions:**
│ • `[⚡ Load Article Context: item_gemini_37]` -> Deep ground this article in conversation
│ • `[🔗 Source URL]` -> https://blog.google/technology/ai/gemini-3-7-flash
└─────────────────────────────────────────────────────────────
```

---

### 2. `get_article_context`
Deep-fetches and extracts live article body text for complete factual grounding.

**Arguments:**
- `article_id` *(string, optional)*: Database ID of the article.
- `url` *(string, optional)*: Direct URL to fetch if ID is not known.

---

### 3. `generate_tldr`
Generates a 3-section executive strategic briefing across all 5 streams using Vertex AI ADC.

---

### 4. `trigger_crawl`
Executes an immediate live crawl of all configured RSS, arXiv, Hugging Face, and GCP sources.

---

### 5. `agent_chat`
Conducts conversational research grounded against the daily newsletter or a specific article.

**Arguments:**
- `message` *(string, required)*: User question or research prompt.
- `session_id` *(string, optional)*: Multi-turn session tracking ID.
- `article_id` *(string, optional)*: Specific article to ground against.

---

### 6. `get_system_status`
Inspects database row counts, active Gemini/Vertex model, and authentication mode.

---

## Transports

### Direct JSON-RPC (`POST /mcp`)
Ideal for Cloud Run inter-agent callers:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "list_articles",
      "arguments": { "limit": 3 }
    }
  }'
```

### Server-Sent Events (`GET /sse` & `POST /message`)
Standard streaming protocol for desktop agent integrations (Antigravity, Claude Desktop, Cursor).
