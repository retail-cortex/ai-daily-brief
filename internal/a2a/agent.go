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
	"fmt"
	"log"
	"strings"
	"time"

	"ai-daily-brief/internal/config"
)

// Agent coordinates Agent-to-Agent intelligence workflows consuming MCP
type Agent struct {
	Config    *config.AgentConfig
	MCPClient *MCPClient
}

func NewAgent(cfg *config.AgentConfig) *Agent {
	if cfg == nil {
		cfg = config.LoadAgentConfig()
	}
	return &Agent{
		Config:    cfg,
		MCPClient: NewMCPClient(cfg.MCPServerURL, cfg.TimeoutSeconds),
	}
}

// TaskResult encapsulates the execution outcome of an A2A agent invocation
type TaskResult struct {
	TaskName   string            `json:"task_name"`
	Status     string            `json:"status"`
	ToolCalls  []string          `json:"tool_calls"`
	Output     string            `json:"output"`
	ExecutedAt string            `json:"executed_at"`
	DurationMs int64             `json:"duration_ms"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ExecuteTask runs an autonomous research task using the MCP control plane
func (a *Agent) ExecuteTask(ctx context.Context, task string) (*TaskResult, error) {
	startTime := time.Now()
	res := &TaskResult{
		TaskName:   task,
		Status:     "SUCCESS",
		ToolCalls:  []string{},
		ExecutedAt: time.Now().UTC().Format(time.RFC3339),
		Metadata: map[string]string{
			"agent":     a.Config.AgentName,
			"mcp_url":   a.Config.MCPServerURL,
			"gemini":    a.Config.Gemini.Model,
			"auth_mode": a.Config.Gemini.AuthMode,
		},
	}

	taskLower := strings.ToLower(task)
	var finalOutput strings.Builder
	finalOutput.WriteString(fmt.Sprintf("### 🤖 Agent Execution Report: %s\n\n", a.Config.AgentName))

	// 1. Crawl request check
	if strings.Contains(taskLower, "crawl") || strings.Contains(taskLower, "scrape") || strings.Contains(taskLower, "refresh") {
		res.ToolCalls = append(res.ToolCalls, "trigger_crawl")
		crawlOut, err := a.MCPClient.TriggerCrawl(ctx)
		if err != nil {
			log.Printf("[A2A Agent] TriggerCrawl error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("⚠️ Crawl trigger error: %v\n\n", err))
		} else {
			finalOutput.WriteString(fmt.Sprintf("%s\n\n", crawlOut))
		}
	}

	// 2. TL;DR or summary request check
	if strings.Contains(taskLower, "tldr") || strings.Contains(taskLower, "briefing") || strings.Contains(taskLower, "executive") || strings.Contains(taskLower, "digest") {
		res.ToolCalls = append(res.ToolCalls, "generate_tldr")
		tldrOut, err := a.MCPClient.GenerateTLDR(ctx)
		if err != nil {
			log.Printf("[A2A Agent] GenerateTLDR error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("⚠️ TL;DR error: %v\n\n", err))
		} else {
			finalOutput.WriteString(fmt.Sprintf("%s\n\n", tldrOut))
		}
	}

	// 3. Article search / list query
	if len(res.ToolCalls) == 0 || strings.Contains(taskLower, "search") || strings.Contains(taskLower, "find") || strings.Contains(taskLower, "list") || strings.Contains(taskLower, "paper") || strings.Contains(taskLower, "model") {
		category := ""
		if strings.Contains(taskLower, "frontier") || strings.Contains(taskLower, "llm") {
			category = "Frontier Models"
		} else if strings.Contains(taskLower, "paper") || strings.Contains(taskLower, "arxiv") {
			category = "AI Research Papers"
		} else if strings.Contains(taskLower, "cloud") || strings.Contains(taskLower, "gcp") {
			category = "Google Cloud"
		} else if strings.Contains(taskLower, "business") || strings.Contains(taskLower, "funding") {
			category = "AI Business & Infra"
		}

		company := ""
		if strings.Contains(taskLower, "google") {
			company = "Google"
		} else if strings.Contains(taskLower, "anthropic") {
			company = "Anthropic"
		} else if strings.Contains(taskLower, "openai") {
			company = "OpenAI"
		}

		res.ToolCalls = append(res.ToolCalls, "list_articles")
		articlesOut, err := a.MCPClient.ListArticles(ctx, category, company, "", 5)
		if err != nil {
			log.Printf("[A2A Agent] ListArticles error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("⚠️ List articles error: %v\n\n", err))
		} else {
			finalOutput.WriteString(fmt.Sprintf("%s\n\n", articlesOut))
		}
	}

	res.Output = finalOutput.String()
	res.DurationMs = time.Since(startTime).Milliseconds()
	return res, nil
}
