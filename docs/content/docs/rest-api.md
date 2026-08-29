---
title: "REST API Reference"
weight: 8
---

# 📡 REST API Reference

In addition to the Model Context Protocol (MCP), AI Daily Brief exposes a complete REST API.

---

## Endpoints

### 1. `GET /healthz`
Cloud Run health and readiness check probe.
- **Response**: `200 OK`
```json
{
  "protocol": "model-context-protocol-2024-11-05",
  "server": "ai-daily-brief-mcp",
  "status": "healthy",
  "timestamp": "2026-02-25T20:00:00Z"
}
```

---

### 2. `GET /api/items`
Query indexed intelligence records.
- **Parameters**: `search`, `company`, `category`, `limit`
- **Response**:
```json
{
  "count": 10,
  "items": [ ... ],
  "success": true
}
```

---

### 3. `POST /api/batch/run`
Trigger an immediate parallel crawl across all 5 streams.
- **Response**:
```json
{
  "result": {
    "items_inserted": 12,
    "status": "SUCCESS",
    "total_in_db": 340
  },
  "success": true
}
```

---

### 4. `POST /api/agent/chat`
Interactive context-grounded conversational agent.
- **Body**:
```json
{
  "message": "What are the latest TPU v5e benchmark results?",
  "session_id": "sess_12345"
}
```

---

### 5. `GET /ws/live`
Multimodal Live Bidi WebSocket endpoint for low-latency conversational audio research.
