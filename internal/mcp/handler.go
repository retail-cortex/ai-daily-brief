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

// MCP Content Block
type MCPContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
			Description: "Synthesize today's executive strategic intelligence brief across all 5 streams using Vertex AI ADC or Gemini 3.7.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
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
					"mimeType":    "text/markdown",
					"description": "Complete curated AI & Cloud daily intelligence briefing",
				},
				{
					"uri":         "brief://today/tldr",
					"name":        "Executive Strategic Briefing",
					"mimeType":    "text/markdown",
					"description": "Strategic synthesis across Frontier Models, Compute, and Open-Source",
				},
			},
		}
		return resp

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			return resp
		}

		content, err := h.ExecuteTool(callParams.Name, callParams.Arguments)
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
			"content": []MCPContentBlock{
				{Type: "text", Text: content},
			},
		}
		return resp

	default:
		resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("Method '%s' not found", req.Method)}
		return resp
	}
}

// ExecuteTool dispatches specific tool operations
func (h *Handler) ExecuteTool(name string, args map[string]interface{}) (string, error) {
	switch name {
	case "list_articles":
		limit := 10
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
			if limit > 50 {
				limit = 50
			}
		}

		query := h.DB.Order("pub_date DESC").Limit(limit)
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
			return "", fmt.Errorf("database query failed: %w", err)
		}

		return FormatArticleCards(items), nil

	case "get_article_context":
		articleID, _ := args["article_id"].(string)
		url, _ := args["url"].(string)

		var item database.NewsItem
		if articleID != "" {
			if err := h.DB.First(&item, "id = ?", articleID).Error; err != nil {
				return "", fmt.Errorf("article with ID '%s' not found: %w", articleID, err)
			}
			url = item.Link
		}

		if url == "" {
			return "", fmt.Errorf("either 'article_id' or 'url' must be specified")
		}

		bodyText, err := agent.FetchFullArticleText(url)
		if err != nil {
			if item.Summary != "" {
				bodyText = fmt.Sprintf("Notice: Full webpage fetch timed out. Using indexed summary:\n%s", item.Summary)
			} else {
				return "", fmt.Errorf("failed to fetch article webpage: %w", err)
			}
		}

		if item.Title == "" {
			item.Title = url
			item.Link = url
			item.Company = "Web"
			item.Category = "General"
		}

		return FormatGroundedContextCard(item, bodyText), nil

	case "generate_tldr":
		var items []database.NewsItem
		h.DB.Order("pub_date DESC").Limit(25).Find(&items)
		tldr, err := agent.GenerateDailyTLDR(h.DB, items)
		if err != nil {
			return "", fmt.Errorf("TL;DR generation failed: %w", err)
		}
		dateStr := time.Now().Format("Monday, Jan 02, 2006")
		return FormatExecutiveTLDRCard(tldr, dateStr), nil

	case "trigger_crawl":
		res, err := crawler.ExecuteBatchRun(h.DB)
		if err != nil {
			return "", fmt.Errorf("crawler execution failed: %w", err)
		}
		var sb strings.Builder
		sb.WriteString("┌─────────────────────────────────────────────────────────────\n")
		sb.WriteString("│ ⚡ **INTELLIGENCE CRAWLER BATCH SUMMARY**\n")
		sb.WriteString("├─────────────────────────────────────────────────────────────\n")
		sb.WriteString(fmt.Sprintf("│ • **Status:**             %s\n", res.Status))
		sb.WriteString(fmt.Sprintf("│ • **New Articles Added:** %d\n", res.NewItemsInserted))
		sb.WriteString(fmt.Sprintf("│ • **Duplicates Filtered:** %d\n", res.SkippedDuplicates))
		sb.WriteString(fmt.Sprintf("│ • **Total DB Articles:**  %d\n", res.TotalInDB))
		sb.WriteString("├─────────────────────────────────────────────────────────────\n")
		sb.WriteString("│ 📄 **Execution Log Preview:**\n│\n")
		for _, l := range strings.Split(res.Log, "\n") {
			if strings.TrimSpace(l) != "" {
				sb.WriteString(fmt.Sprintf("│ %s\n", l))
			}
		}
		sb.WriteString("└─────────────────────────────────────────────────────────────\n")
		return sb.String(), nil

	case "get_newsletter":
		var items []database.NewsItem
		h.DB.Order("pub_date DESC").Limit(30).Find(&items)
		dateStr := time.Now().Format("Monday, Jan 02, 2006")
		html := mailer.GenerateNewsletterHTML(items, dateStr)
		_ = html
		cards := FormatArticleCards(items)
		return fmt.Sprintf("# 📧 Daily AI & Cloud Intelligence Digest (%s)\n\n%s", dateStr, cards), nil

	case "agent_chat":
		msg, _ := args["message"].(string)
		if strings.TrimSpace(msg) == "" {
			return "", fmt.Errorf("message argument is required")
		}
		sessionID, _ := args["session_id"].(string)
		articleID, _ := args["article_id"].(string)

		chatResp, err := agent.GenerateChatResponse(h.DB, sessionID, msg, articleID)
		if err != nil {
			return "", err
		}
		return chatResp.Response, nil

	case "get_system_status":
		var totalItems int64
		h.DB.Model(&database.NewsItem{}).Count(&totalItems)
		model, authMode, _, projectID, location := agent.GetAgentSettings(h.DB)
		return FormatTelemetryCard(totalItems, model, authMode, projectID, location), nil

	default:
		return "", fmt.Errorf("unknown tool name '%s'", name)
	}
}
