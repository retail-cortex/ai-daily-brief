// Copyright 2026 Retail Cortex
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-daily-brief/internal/agent"
	"ai-daily-brief/internal/crawler"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/mailer"

	"gorm.io/gorm"
)

// MCP JSON-RPC 2.0 Structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP Resource Content
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// MCP Content Block
type MCPContentBlock struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	Resource *MCPResourceContent `json:"resource,omitempty"`
}

// Tool Definition Structure
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Handler dispatches MCP JSON-RPC requests
type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) GetToolDefinitions() []MCPTool {
	return []MCPTool{
		{
			Name:        "list_articles",
			Description: "List indexed AI, Cloud, and LLM news items with A2UI formatted cards and action triggers.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by stream category (e.g. 'Frontier Models', 'Google Cloud', 'Research Papers', 'AI Business', 'OSS Tooling')",
					},
					"company": map[string]interface{}{
						"type":        "string",
						"description": "Filter by organization or company name (e.g. 'Google', 'Anthropic', 'OpenAI', 'Meta')",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search keyword query across article title and summary",
					},
					"days": map[string]interface{}{
						"type":        "integer",
						"description": "Filter articles created in the past N days (e.g. 7 for past week, 30 for past month)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of articles to return (default: 10, max: 50)",
					},
				},
			},
		},
		{
			Name:        "get_article_context",
			Description: "Deep-fetch and sanitize live webpage text for an article to provide complete factual grounding for an agent.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"article_id": map[string]interface{}{
						"type":        "string",
						"description": "Database ID of the article to inspect and ground",
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Direct external URL to fetch and ground if article ID is not known",
					},
				},
			},
		},
		{
			Name:        "generate_tldr",
			Description: "Synthesize an executive strategic intelligence brief or multi-day overview across all 5 streams using Vertex AI ADC or Gemini 3.7.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of past days to synthesize into the overview (1 for daily brief, 7 for weekly overview, 30 for monthly)",
					},
					"time_span": map[string]interface{}{
						"type":        "string",
						"description": "Human-readable timeframe label (e.g. 'the Past Week', 'the Past 7 Days', 'the Past 30 Days')",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category filter (e.g. 'Frontier Models', 'AI Research Papers', 'Google Cloud')",
					},
				},
			},
		},
		{
			Name:        "trigger_crawl",
			Description: "Execute a live concurrent crawl of RSS feeds, arXiv preprints, HuggingFace releases, and Google Cloud release notes.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_newsletter",
			Description: "Retrieve today's compiled executive intelligence newsletter with HTML and Markdown representations.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "agent_chat",
			Description: "Perform interactive conversational research grounded against the daily digest or a specific article.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "User question or prompt for the research agent",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional session ID for conversation history tracking",
					},
					"article_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional article ID to ground the response against",
					},
				},
				"required": []string{"message"},
			},
		},
		{
			Name:        "get_system_status",
			Description: "Inspect control plane health, database article counts, active Gemini/Vertex model, and authentication mode.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_telemetry",
			Description: "Inspect control plane health, database article counts, active Gemini/Vertex model, and authentication mode (alias for get_system_status).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// HandleRequest processes an incoming MCP JSON-RPC message
func (h *Handler) HandleRequest(req JSONRPCRequest) JSONRPCResponse {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "ai-daily-brief-mcp",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
			},
		}
		return resp

	case "notifications/initialized":
		// Notification per MCP spec
		return resp

	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": h.GetToolDefinitions(),
		}
		return resp

	case "resources/list":
		resp.Result = map[string]interface{}{
			"resources": []map[string]interface{}{
				{
					"uri":         "brief://today/newsletter",
					"name":        "Today's Intelligence Digest",
					"mimeType":    MIMETypeMarkdown,
					"description": "Complete curated AI & Cloud daily intelligence briefing",
				},
				{
					"uri":         "brief://today/tldr",
					"name":        "Executive Strategic Briefing",
					"mimeType":    MIMETypeMarkdown,
					"description": "Strategic synthesis across Frontier Models, Compute, and Open-Source",
				},
				{
					"uri":         "brief://today/a2ui",
					"name":        "Today's A2UI Component Cards",
					"mimeType":    MIMETypeA2UIJSON,
					"description": "Structured visual cards formatted for Gemini Enterprise and Agent UI",
				},
			},
		}
		return resp

	case "resources/read":
		var resParams struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &resParams); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid resource read params"}
			return resp
		}

		switch resParams.URI {
		case "brief://today/newsletter":
			var items []database.NewsItem
			h.DB.Order("pub_date DESC").Limit(30).Find(&items)
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# 📰 Daily AI & Cloud Intelligence Digest (%d items)\n\n", len(items)))
			for i, it := range items {
				sb.WriteString(fmt.Sprintf("### %d. %s\nSource: **%s** • Category: *%s*\n%s\nLink: %s\n\n", i+1, it.Title, it.Company, it.Category, it.Summary, it.Link))
			}
			resp.Result = map[string]interface{}{
				"contents": []map[string]interface{}{
					{
						"uri":      resParams.URI,
						"mimeType": MIMETypeMarkdown,
						"text":     strings.TrimSpace(sb.String()),
					},
				},
			}
			return resp

		case "brief://today/a2ui":
			var items []database.NewsItem
			h.DB.Order("pub_date DESC").Limit(30).Find(&items)
			cards := BuildArticleCardsA2UI(items)
			resp.Result = map[string]interface{}{
				"contents": []map[string]interface{}{
					{
						"uri":      resParams.URI,
						"mimeType": MIMETypeA2UIJSON,
						"text":     MarshalA2UIJSON(cards),
					},
				},
			}
			return resp

		case "brief://today/tldr":
			var setting database.Setting
			tldrText := "No briefing generated yet today. Call tool 'generate_tldr' to synthesize."
			if err := h.DB.First(&setting, "\"key\" = ?", "latest_tldr").Error; err == nil && setting.Value != "" {
				tldrText = setting.Value
			}
			resp.Result = map[string]interface{}{
				"contents": []map[string]interface{}{
					{
						"uri":      resParams.URI,
						"mimeType": MIMETypeMarkdown,
						"text":     tldrText,
					},
				},
			}
			return resp

		default:
			resp.Error = &RPCError{Code: -32602, Message: fmt.Sprintf("Resource not found: %s", resParams.URI)}
			return resp
		}

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			return resp
		}

		blocks, err := h.ExecuteToolBlocks(callParams.Name, callParams.Arguments)
		if err != nil {
			resp.Result = map[string]interface{}{
				"content": []MCPContentBlock{
					{Type: "text", Text: fmt.Sprintf("❌ Tool execution error: %v", err)},
				},
				"isError": true,
			}
			return resp
		}

		resp.Result = map[string]interface{}{
			"content": blocks,
		}
		return resp

	default:
		resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("Method '%s' not found", req.Method)}
		return resp
	}
}

// ExecuteTool dispatches specific tool operations and returns the primary text representation
func (h *Handler) ExecuteTool(name string, args map[string]interface{}) (string, error) {
	blocks, err := h.ExecuteToolBlocks(name, args)
	if err != nil {
		return "", err
	}
	var texts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			texts = append(texts, b.Text)
		}
	}
	return strings.Join(texts, "\n\n"), nil
}

// ExecuteToolBlocks executes a tool and produces structured content blocks with precise MIME types
func (h *Handler) ExecuteToolBlocks(name string, args map[string]interface{}) ([]MCPContentBlock, error) {
	switch name {
	case "list_articles":
		limit := 10
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
			if limit > 50 {
				limit = 50
			}
		}

		query := h.DB.Order("pub_date DESC, created_at DESC").Limit(limit)
		if d, ok := args["days"].(float64); ok && d > 0 {
			cutoff := time.Now().AddDate(0, 0, -int(d))
			query = query.Where("created_at >= ?", cutoff)
		}
		if cat, ok := args["category"].(string); ok && strings.TrimSpace(cat) != "" {
			query = query.Where("category LIKE ?", "%"+strings.TrimSpace(cat)+"%")
		}
		if comp, ok := args["company"].(string); ok && strings.TrimSpace(comp) != "" {
			query = query.Where("company LIKE ?", "%"+strings.TrimSpace(comp)+"%")
		}
		if q, ok := args["query"].(string); ok && strings.TrimSpace(q) != "" {
			searchVal := "%" + strings.TrimSpace(q) + "%"
			query = query.Where("title LIKE ? OR summary LIKE ?", searchVal, searchVal)
		}

		var items []database.NewsItem
		if err := query.Find(&items).Error; err != nil {
			return nil, fmt.Errorf("database query failed: %w", err)
		}

		introText := fmt.Sprintf("I retrieved %d articles from your intelligence stream. Please inspect the cards below:", len(items))
		if len(items) == 0 {
			introText = "No intelligence items found matching your query."
		}
		structuredCards := BuildArticleCardsA2UI(items)

		return []MCPContentBlock{
			{
				Type: "text",
				Text: introText,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      "brief://a2ui/articles/list",
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(structuredCards),
				},
			},
		}, nil

	case "get_article_context":
		articleID, _ := args["article_id"].(string)
		url, _ := args["url"].(string)

		var item database.NewsItem
		if articleID != "" {
			if err := h.DB.First(&item, "id = ?", articleID).Error; err != nil {
				// Try case-insensitive or partial prefix matching
				_ = h.DB.Where("id LIKE ?", "%"+articleID+"%").First(&item).Error
			}
			if item.Link != "" {
				url = item.Link
			}
		}

		if url == "" && item.Link != "" {
			url = item.Link
		}

		if url == "" && articleID == "" {
			return nil, fmt.Errorf("either 'article_id' or 'url' must be specified")
		}

		var bodyText string
		if url != "" {
			var err error
			bodyText, err = agent.FetchFullArticleText(url)
			if err != nil {
				if item.Summary != "" {
					bodyText = fmt.Sprintf("Extracted Overview & Key Takeaways:\n\n%s\n\nFull Reference Source: %s (%s)", item.Summary, url, item.Company)
				} else {
					bodyText = fmt.Sprintf("External source reference: %s (%s)\n\nFull webpage extraction unavailable; indexed metadata preserved.", url, item.Company)
				}
			}
		} else if item.Summary != "" {
			bodyText = item.Summary
		}

		if item.Title == "" {
			if url != "" {
				item.Title = url
				item.Link = url
			} else {
				item.Title = articleID
			}
			item.Company = "AI Source"
			item.Category = "General"
		}

		mode, _ := args["mode"].(string)

		var introText string
		var payload A2UIPayload
		if mode == "summary" || mode == "summarize" || mode == "architecture" {
			summaryText, err := agent.GenerateArticleSummary(h.DB, item, bodyText)
			if err != nil || summaryText == "" {
				summaryText = item.Summary
			}
			introText = fmt.Sprintf("Here is the technical synthesis for '%s' (%s):\n\n%s", item.Title, item.ID, summaryText)
			payload = BuildArticleSummaryA2UI(item, summaryText)
		} else {
			introText = fmt.Sprintf("I loaded the grounded context for '%s' (%s). Use the card below to inspect extracted text or summarize:", item.Title, item.ID)
			payload = BuildGroundedContextA2UI(item, bodyText)
		}

		return []MCPContentBlock{
			{
				Type: "text",
				Text: introText,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      fmt.Sprintf("brief://a2ui/articles/%s/context", item.ID),
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(payload),
				},
			},
		}, nil

	case "generate_tldr":
		days := 1
		if d, ok := args["days"].(float64); ok && d > 0 {
			days = int(d)
		}
		timeSpanStr, _ := args["time_span"].(string)
		if timeSpanStr == "" {
			if days == 7 {
				timeSpanStr = "the Past Week"
			} else if days > 1 {
				timeSpanStr = fmt.Sprintf("the Past %d Days", days)
			} else {
				timeSpanStr = "Today"
			}
		}

		query := h.DB.Order("pub_date DESC, created_at DESC")
		if days > 1 {
			cutoff := time.Now().AddDate(0, 0, -days)
			query = query.Where("created_at >= ?", cutoff)
		}
		if cat, ok := args["category"].(string); ok && strings.TrimSpace(cat) != "" {
			query = query.Where("category LIKE ?", "%"+strings.TrimSpace(cat)+"%")
		}

		var items []database.NewsItem
		query.Limit(35).Find(&items)
		if len(items) < 3 {
			// Fallback to latest items if time filter yields few results
			h.DB.Order("pub_date DESC, created_at DESC").Limit(25).Find(&items)
		}

		var tldr string
		var err error
		if days > 1 {
			tldr, err = agent.GenerateTimespanOverview(h.DB, items, timeSpanStr)
		} else {
			tldr, err = agent.GenerateDailyTLDR(h.DB, items)
		}
		if err != nil {
			return nil, fmt.Errorf("overview generation failed: %w", err)
		}

		dateStr := time.Now().Format("Monday, Jan 02, 2006")
		introText := fmt.Sprintf("Here is the Executive Strategic Overview for %s (%s):\n\n%s", timeSpanStr, dateStr, tldr)
		tldrInstructions := BuildExecutiveTLDRA2UI(tldr, fmt.Sprintf("%s • %s", timeSpanStr, dateStr))

		return []MCPContentBlock{
			{
				Type: "text",
				Text: introText,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      fmt.Sprintf("brief://a2ui/tldr/%s", dateStr),
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(tldrInstructions),
				},
			},
		}, nil

	case "trigger_crawl":
		res, err := crawler.ExecuteBatchRun(h.DB)
		if err != nil {
			return nil, fmt.Errorf("crawler execution failed: %w", err)
		}
		introText := fmt.Sprintf("Intelligence crawl completed with status '%s' (%d new articles inserted).", res.Status, res.NewItemsInserted)
		crawlCard := BuildCrawlBatchA2UI(res)

		return []MCPContentBlock{
			{
				Type: "text",
				Text: introText,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      fmt.Sprintf("brief://a2ui/crawls/%s", res.RunID),
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(crawlCard),
				},
			},
		}, nil

	case "get_newsletter":
		var items []database.NewsItem
		h.DB.Order("pub_date DESC").Limit(30).Find(&items)
		dateStr := time.Now().Format("Monday, Jan 02, 2006")
		html := mailer.GenerateNewsletterHTML(items, dateStr)
		introText := fmt.Sprintf("Daily AI & Cloud Intelligence Digest for %s (%d articles prepared).", dateStr, len(items))
		structuredCards := BuildArticleCardsA2UI(items)

		return []MCPContentBlock{
			{
				Type: "text",
				Text: introText,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      "brief://newsletter/html",
					MIMEType: MIMETypeHTML,
					Text:     html,
				},
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      "brief://a2ui/newsletter/cards",
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(structuredCards),
				},
			},
		}, nil

	case "agent_chat":
		msg, _ := args["message"].(string)
		if strings.TrimSpace(msg) == "" {
			return nil, fmt.Errorf("message argument is required")
		}
		sessionID, _ := args["session_id"].(string)
		if sessionID == "" {
			sessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
		}
		articleID, _ := args["article_id"].(string)

		chatResp, err := agent.GenerateChatResponse(h.DB, sessionID, msg, articleID)
		if err != nil {
			return nil, err
		}

		return []MCPContentBlock{
			{
				Type: "text",
				Text: chatResp.Response,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      fmt.Sprintf("brief://a2ui/chat/%s", sessionID),
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(chatResp),
				},
			},
		}, nil

	case "get_system_status", "get_telemetry":
		var totalItems int64
		h.DB.Model(&database.NewsItem{}).Count(&totalItems)
		model, authMode, _, projectID, location := agent.GetAgentSettings(h.DB)
		telemetryText := fmt.Sprintf("AI Daily Brief Control Plane: %d articles indexed | Active Model: %s | Auth: %s", totalItems, model, authMode)
		telemetryCard := BuildTelemetryA2UI(totalItems, model, authMode, projectID, location)

		return []MCPContentBlock{
			{
				Type: "text",
				Text: telemetryText,
			},
			{
				Type: "resource",
				Resource: &MCPResourceContent{
					URI:      "brief://a2ui/telemetry",
					MIMEType: MIMETypeA2UIJSON,
					Text:     MarshalA2UIJSON(telemetryCard),
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool name '%s'", name)
	}
}
