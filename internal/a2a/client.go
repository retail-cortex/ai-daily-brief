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
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
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

// getIdTokenForAudience retrieves an OIDC token for Cloud Run service-to-service auth
func (c *MCPClient) getIdTokenForAudience(audience string) string {
	if token := os.Getenv("MCP_AUTH_TOKEN"); token != "" {
		return strings.TrimSpace(token)
	}
	if strings.HasPrefix(audience, "http://localhost") || strings.HasPrefix(audience, "http://127.0.0.1") {
		return ""
	}
	if metadata.OnGCE() {
		token, err := metadata.Get("instance/service-accounts/default/identity?audience=" + audience)
		if err == nil && token != "" {
			return strings.TrimSpace(token)
		}
	}
	return ""
}

// MCPContentBlock represents a typed item in an MCP response
type MCPContentBlock struct {
	Type     string                  `json:"type"`
	Text     string                  `json:"text,omitempty"`
	Resource *MCPResourceContent     `json:"resource,omitempty"`
}

type MCPResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// CallToolBlocks executes a tool on the MCP server and returns the raw structured content blocks
func (c *MCPClient) CallToolBlocks(ctx context.Context, toolName string, arguments map[string]interface{}) ([]MCPContentBlock, error) {
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
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if token := c.getIdTokenForAudience(c.BaseURL); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to MCP server failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rpcResp struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			Content []MCPContentBlock `json:"content"`
			IsError bool              `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP RPC error [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result.Content, nil
}

// CallTool executes a tool on the MCP server via JSON-RPC 2.0 and returns the primary text representation
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	blocks, err := c.CallToolBlocks(ctx, toolName, arguments)
	if err != nil {
		return "", err
	}

	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			return b.Text, nil
		}
	}

	return "", nil
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

// GenerateTLDR generates executive TLDR summary
func (c *MCPClient) GenerateTLDR(ctx context.Context) (string, error) {
	return c.CallTool(ctx, "generate_tldr", nil)
}

// TriggerCrawl triggers immediate crawl run
func (c *MCPClient) TriggerCrawl(ctx context.Context) (string, error) {
	return c.CallTool(ctx, "trigger_crawl", nil)
}

// GetSystemStatus queries system status
func (c *MCPClient) GetSystemStatus(ctx context.Context) (string, error) {
	return c.CallTool(ctx, "get_system_status", nil)
}
