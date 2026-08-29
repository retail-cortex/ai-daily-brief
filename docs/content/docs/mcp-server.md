---
title: "MCP Server & A2UI"
weight: 3
---

# 🤖 Model Context Protocol (MCP) & A2UI

AI Daily Brief implements the official **Model Context Protocol (2024-11-05)** specification. It formats responses using **dual content blocks**:
1. **`text`**: Formatted ASCII / Unicode cards for human readability, terminal logs, and standard markdown consoles.
2. **`resource` (`mimeType: application/a2ui+json`)**: Structured **Agent-to-UI (A2UI)** components natively rendered by Gemini Enterprise, Google Agent Space, and rich frontend conversational cards.

---

## Content Block Format (Gemini Enterprise Ready)

When executing `tools/call` or reading `resources/read`, the server returns structured `MCPContentBlock` items:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "### 📰 Intelligence Stream (2 records)\n\n┌─────────────────────────────────────────────────────────────\n│ **#1: Google Cloud Release: Feature (August 27, 2026)**\n│ ☁️ Google Cloud & Compute • **Google Cloud** • *2026-08-27*\n├─────────────────────────────────────────────────────────────\n│ Set up a regional external passthrough Network Load Balancer...\n├─────────────────────────────────────────────────────────────\n│ ⚡ **Actions:**\n│ • `[⚡ Load Article Context: gcp-rel-123]` -> Deep ground this article\n│ • `[🔗 Source URL]` -> https://docs.cloud.google.com/release-notes\n└─────────────────────────────────────────────────────────────\n"
      },
      {
        "type": "resource",
        "resource": {
          "uri": "brief://a2ui/articles/list",
          "mimeType": "application/a2ui+json",
          "text": "[{\"id\":\"gcp-rel-123\",\"type\":\"article_card\",\"title\":\"Google Cloud Release...\",\"category\":\"cloud\",\"badge\":\"☁️ Google Cloud & Compute\",\"content\":\"Set up a regional...\",\"actions\":[{\"label\":\"⚡ Load Article Context\",\"action_type\":\"load_context\",\"payload\":{\"article_id\":\"gcp-rel-123\"}}]}]"
        }
      }
    ]
  }
}
```

---

## Supported Tools

### 1. `list_articles`
Lists indexed AI, Cloud, and LLM news items with A2UI formatted cards and structured A2UI JSON components.

**Arguments:**
- `category` *(string, optional)*: Filter by stream (`"Frontier Models"`, `"Google Cloud"`, `"Research Papers"`, `"AI Business"`, `"OSS Tooling"`).
- `company` *(string, optional)*: Filter by company (`"Google"`, `"Anthropic"`, `"OpenAI"`, `"Meta"`).
- `query` *(string, optional)*: Keyword search query across title and summary.
- `limit` *(integer, optional)*: Maximum articles to return (default: `10`, max: `50`).

---

### 2. `get_article_context`
Deep-fetches and extracts sanitized live article body text for complete factual grounding, returning both reading card and `application/a2ui+json` grounding inspector card.

**Arguments:**
- `article_id` *(string, optional)*: Database ID of the article to inspect.
- `url` *(string, optional)*: Direct external URL to fetch and ground if article ID is not known.

---

### 3. `generate_tldr`
Synthesizes today's executive strategic intelligence brief across all 5 streams using Vertex AI ADC or Gemini 3.7 Flash and returns an interactive `application/a2ui+json` strategic card.

---

### 4. `trigger_crawl`
Executes an immediate live crawl of RSS feeds, arXiv preprints, HuggingFace releases, and Google Cloud release notes.

---

### 5. `get_newsletter`
Retrieves today's compiled executive intelligence newsletter with both HTML (`text/html`), Markdown (`text/markdown`), and `application/a2ui+json` card components.

---

### 6. `agent_chat`
Conducts conversational research grounded against the daily digest or a specific article.

**Arguments:**
- `message` *(string, required)*: User question or research prompt.
- `session_id` *(string, optional)*: Session tracking ID for multi-turn conversations.
- `article_id` *(string, optional)*: Specific article ID to ground against.

---

### 7. `get_system_status`
Inspects database row counts, active Gemini/Vertex model, and authentication mode with a telemetry card.

---

## MCP Resources

| Resource URI | MIME Type | Description |
| :--- | :--- | :--- |
| `brief://today/newsletter` | `text/markdown` | Today's complete curated daily briefing |
| `brief://today/a2ui` | `application/a2ui+json` | Today's structured visual card deck |
| `brief://today/tldr` | `text/markdown` | Executive Strategic TL;DR briefing |

---

## Invocation & Transports

### Direct JSON-RPC (`POST /mcp`)
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

### Remote Invocation on Cloud Run (Authenticated)
When invoking a deployed Cloud Run service with IAM authentication enforced:

```bash
ENDPOINT="https://ai-daily-brief-mcp-dev-${PROJECT_NUMBER}.us-central1.run.app"
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Generate an audience-bound ID token
TOKEN=$(curl -s -X POST \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"audience\": \"${ENDPOINT}\", \"includeEmail\": true}" \
  "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/ai-daily-brief-sa-dev@${PROJECT_ID}.iam.gserviceaccount.com:generateIdToken" | jq -r .token)

# Call tool
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "list_articles", "arguments": {"limit": 2}}, "id": 1}' \
  "${ENDPOINT}/mcp" | jq .
```

### Server-Sent Events (`GET /sse` & `POST /message`)
Standard streaming protocol for desktop agent integrations (Antigravity, Claude Desktop, Cursor).

---

### Gemini Enterprise & Discovery Engine Integration
To connect this MCP server directly to Google Gemini Enterprise or Discovery Engine as a managed tool connector with IAM service account authentication, refer to the [Gemini Enterprise Integration Guide](/docs/gemini_enterprise/).
