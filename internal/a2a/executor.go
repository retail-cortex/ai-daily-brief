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
	"context"

	"ai-daily-brief/internal/mcp"

	"gorm.io/gorm"
)

// ToolExecutor defines the contract for executing intelligence tools,
// enabling both zero-latency in-process dispatch and remote MCP JSON-RPC execution.
type ToolExecutor interface {
	CallToolBlocks(ctx context.Context, toolName string, arguments map[string]interface{}) ([]mcp.MCPContentBlock, error)
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error)
	GetSystemStatus(ctx context.Context) (string, error)
	GetArticleContext(ctx context.Context, articleID, url string) (string, error)
	ListArticles(ctx context.Context, category, company, query string, limit int) (string, error)
	GenerateTLDR(ctx context.Context) (string, error)
	TriggerCrawl(ctx context.Context) (string, error)
	GetNewsletter(ctx context.Context) (string, error)
}

// InProcessToolExecutor executes tools directly in-memory via mcp.Handler
type InProcessToolExecutor struct {
	Handler *mcp.Handler
}

// NewInProcessToolExecutor initializes a direct in-process executor backed by GORM
func NewInProcessToolExecutor(db *gorm.DB) *InProcessToolExecutor {
	return &InProcessToolExecutor{
		Handler: mcp.NewHandler(db),
	}
}

// CallToolBlocks executes a tool directly and produces structured content blocks
func (e *InProcessToolExecutor) CallToolBlocks(ctx context.Context, toolName string, arguments map[string]interface{}) ([]mcp.MCPContentBlock, error) {
	if arguments == nil {
		arguments = make(map[string]interface{})
	}
	return e.Handler.ExecuteToolBlocks(toolName, arguments)
}

// CallTool executes a tool directly and returns the primary text representation
func (e *InProcessToolExecutor) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	if arguments == nil {
		arguments = make(map[string]interface{})
	}
	return e.Handler.ExecuteTool(toolName, arguments)
}

// GetSystemStatus inspects system telemetry directly in-process
func (e *InProcessToolExecutor) GetSystemStatus(ctx context.Context) (string, error) {
	return e.Handler.ExecuteTool("get_system_status", nil)
}

// GetArticleContext fetches and sanitizes grounded article context directly
func (e *InProcessToolExecutor) GetArticleContext(ctx context.Context, articleID, url string) (string, error) {
	args := map[string]interface{}{}
	if articleID != "" {
		args["article_id"] = articleID
	}
	if url != "" {
		args["url"] = url
	}
	return e.Handler.ExecuteTool("get_article_context", args)
}

// ListArticles lists indexed articles directly from the database
func (e *InProcessToolExecutor) ListArticles(ctx context.Context, category, company, query string, limit int) (string, error) {
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
	return e.Handler.ExecuteTool("list_articles", args)
}

// GenerateTLDR synthesizes executive briefing directly in-process
func (e *InProcessToolExecutor) GenerateTLDR(ctx context.Context) (string, error) {
	return e.Handler.ExecuteTool("generate_tldr", nil)
}

// TriggerCrawl runs a crawl batch directly
func (e *InProcessToolExecutor) TriggerCrawl(ctx context.Context) (string, error) {
	return e.Handler.ExecuteTool("trigger_crawl", nil)
}

// GetNewsletter compiles newsletter representation directly
func (e *InProcessToolExecutor) GetNewsletter(ctx context.Context) (string, error) {
	return e.Handler.ExecuteTool("get_newsletter", nil)
}
