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

package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MCPClient connects to and invokes tools on the AI Daily Brief MCP Server
type MCPClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewMCPClient(baseURL string, timeoutSeconds int) *MCPClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	return &MCPClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// CallTool executes a tool on the MCP server via JSON-RPC 2.0
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	if arguments == nil {
		arguments = make(map[string]interface{})
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request to MCP server failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MCP server returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rpcResp struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("MCP RPC error [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if len(rpcResp.Result.Content) > 0 {
		return rpcResp.Result.Content[0].Text, nil
	}

	return string(bodyBytes), nil
}

// ListArticles fetches articles from MCP
func (c *MCPClient) ListArticles(ctx context.Context, category, company, query string, limit int) (string, error) {
	args := map[string]interface{}{}
	if category != "" {
		args["category"] = category
	}
	if company != "" {
		args["company"] = company
	}
	if query != "" {
		args["query"] = query
	}
	if limit > 0 {
		args["limit"] = limit
	}
	return c.CallTool(ctx, "list_articles", args)
}

// GetArticleContext deep-extracts full webpage context
func (c *MCPClient) GetArticleContext(ctx context.Context, articleID, url string) (string, error) {
	args := map[string]interface{}{}
	if articleID != "" {
		args["article_id"] = articleID
	}
	if url != "" {
		args["url"] = url
	}
	return c.CallTool(ctx, "get_article_context", args)
}

// GenerateTLDR invokes strategic briefing generation
func (c *MCPClient) GenerateTLDR(ctx context.Context) (string, error) {
	return c.CallTool(ctx, "generate_tldr", nil)
}

// TriggerCrawl triggers immediate crawler execution
func (c *MCPClient) TriggerCrawl(ctx context.Context) (string, error) {
	return c.CallTool(ctx, "trigger_crawl", nil)
}

// GetSystemStatus fetches telemetry
func (c *MCPClient) GetSystemStatus(ctx context.Context) (string, error) {
	return c.CallTool(ctx, "get_system_status", nil)
}
