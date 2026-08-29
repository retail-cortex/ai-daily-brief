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
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/mcp"

	"gorm.io/gorm"
)

// Agent coordinates Agent-to-Agent intelligence workflows backed by ToolExecutor (direct in-process or remote MCP)
type Agent struct {
	Config   *config.AgentConfig
	Executor ToolExecutor
}

// NewAgent creates an agent using configured MCP server URL (remote fallback)
func NewAgent(cfg *config.AgentConfig) *Agent {
	if cfg == nil {
		cfg = config.LoadAgentConfig()
	}
	mcpCfg := cfg.GetMCPServer("daily_brief")
	return &Agent{
		Config:   cfg,
		Executor: NewMCPClient(mcpCfg.URL, mcpCfg.TimeoutSeconds),
	}
}

// NewInProcessAgent creates an agent that executes tools directly in-memory with zero network overhead
func NewInProcessAgent(cfg *config.AgentConfig, db *gorm.DB) *Agent {
	if cfg == nil {
		cfg = config.LoadAgentConfig()
	}
	return &Agent{
		Config:   cfg,
		Executor: NewInProcessToolExecutor(db),
	}
}

// NewAgentWithExecutor creates an agent with an explicit ToolExecutor
func NewAgentWithExecutor(cfg *config.AgentConfig, executor ToolExecutor) *Agent {
	if cfg == nil {
		cfg = config.LoadAgentConfig()
	}
	return &Agent{
		Config:   cfg,
		Executor: executor,
	}
}

// GetExecutor returns the underlying ToolExecutor
func (a *Agent) GetExecutor() ToolExecutor {
	return a.Executor
}

// TaskResult encapsulates the execution outcome of an A2A agent invocation
type TaskResult struct {
	TaskName         string                 `json:"task_name"`
	Status           string                 `json:"status"`
	ToolCalls        []string               `json:"tool_calls"`
	Output           string                 `json:"output"`
	A2UIPayload      map[string]interface{} `json:"a2ui_payload,omitempty"`
	A2UIInstructions []interface{}          `json:"a2ui_instructions,omitempty"`
	ExecutedAt       string                 `json:"executed_at"`
	DurationMs       int64                  `json:"duration_ms"`
	Metadata         map[string]string      `json:"metadata,omitempty"`
}

func parseA2UIBlocks(blocks []mcp.MCPContentBlock) (string, map[string]interface{}, []interface{}) {
	var outText strings.Builder
	var payload map[string]interface{}
	var instructions []interface{}

	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			if outText.Len() > 0 {
				outText.WriteString("\n\n")
			}
			outText.WriteString(b.Text)
		} else if b.Type == "resource" && b.Resource != nil && b.Resource.Text != "" {
			var rawMap map[string]interface{}
			if err := json.Unmarshal([]byte(b.Resource.Text), &rawMap); err == nil && len(rawMap) > 0 {
				payload = rawMap
				if instList, ok := rawMap["instructions"].([]interface{}); ok && len(instList) > 0 {
					instructions = instList
				} else {
					if cs, ok := rawMap["createSurface"]; ok {
						instructions = append(instructions, map[string]interface{}{
							"version":       mcp.A2UIVersion,
							"createSurface": cs,
						})
					}
					if uc, ok := rawMap["updateComponents"]; ok {
						instructions = append(instructions, map[string]interface{}{
							"version":          mcp.A2UIVersion,
							"updateComponents": uc,
						})
					}
				}
			} else {
				var rawList []interface{}
				if err := json.Unmarshal([]byte(b.Resource.Text), &rawList); err == nil && len(rawList) > 0 {
					instructions = rawList
					payload = map[string]interface{}{
						"version":      mcp.A2UIVersion,
						"instructions": rawList,
					}
				}
			}
		}
	}
	return outText.String(), payload, instructions
}

// ExecuteTask runs an autonomous research task using the ToolExecutor
func (a *Agent) ExecuteTask(ctx context.Context, task string) (*TaskResult, error) {
	startTime := time.Now()
	geminiCfg := a.Config.GetGeminiConfig(a.Config.AgentName)
	mcpCfg := a.Config.GetMCPServer("daily_brief")

	execMode := "remote_mcp"
	if _, ok := a.Executor.(*InProcessToolExecutor); ok {
		execMode = "in_process_direct"
	}

	res := &TaskResult{
		TaskName:   task,
		Status:     "SUCCESS",
		ToolCalls:  []string{},
		ExecutedAt: time.Now().UTC().Format(time.RFC3339),
		Metadata: map[string]string{
			"agent":          a.Config.AgentName,
			"execution_mode": execMode,
			"mcp_url":        mcpCfg.URL,
			"gemini":         geminiCfg.Model,
			"auth_mode":      geminiCfg.AuthMode,
			"region":         geminiCfg.Region,
		},
	}

	taskLower := strings.ToLower(task)
	var finalOutput strings.Builder

	// 0. Greeting & orientation check
	isGreeting := taskLower == "hi" || taskLower == "hello" || taskLower == "hey" || taskLower == "help" || strings.HasPrefix(taskLower, "hi ") || strings.HasPrefix(taskLower, "hello ") || strings.Contains(taskLower, "who are you") || strings.Contains(taskLower, "what can you do")
	if isGreeting {
		res.ToolCalls = append(res.ToolCalls, "welcome_hub")
		welcomeMsg := "Welcome to AI Daily Brief! I am your autonomous enterprise AI intelligence agent. Use the control cards below to explore today's stream, browse research papers, or trigger live ingestion."
		w := mcp.BuildWelcomeA2UI()
		b, _ := json.Marshal(w)
		_ = json.Unmarshal(b, &res.A2UIPayload)
		var insts []interface{}
		bi, _ := json.Marshal(w.Instructions)
		_ = json.Unmarshal(bi, &insts)
		res.A2UIInstructions = insts

		res.Output = welcomeMsg
		res.DurationMs = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// 0. Extract potential article identifiers
	var targetArticleID string
	var targetURL string

	// A. Check for explicit "article <ID>" or "context for article <ID>" or "article_id: <ID>"
	if m := regexp.MustCompile(`(?i)(?:context\s+for(?:\s+article)?|for\s+article|article(?:\s+id)?)\s*[:=]?\s*([a-zA-Z0-9_\-]+-[a-f0-9]{8,64}|https?://\S+|[a-zA-Z0-9_\-\./]+)`).FindStringSubmatch(task); len(m) > 1 {
		candidate := strings.TrimSpace(m[1])
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			targetURL = candidate
		} else if candidate != "article" && candidate != "context" && candidate != "" {
			targetArticleID = candidate
		}
	}

	// B. Generic <prefix>-<hex> ID match (matches any prefix: gcp-rel-, gcp-, arxiv-, hf-, openai-, etc.)
	if targetArticleID == "" {
		idRegex := regexp.MustCompile(`(?i)\b([a-zA-Z0-9_\-]+-[a-f0-9]{8,64})\b`)
		if m := idRegex.FindStringSubmatch(task); len(m) > 1 {
			targetArticleID = m[1]
		}
	}

	// C. Explicit URL match
	if targetURL == "" {
		urlRegex := regexp.MustCompile(`https?://[^\s)\]]+`)
		if u := urlRegex.FindString(task); u != "" {
			targetURL = u
		}
	}

	// 1. Article-specific operations (Load context, Summarize article, Deep ground)
	isArticleOp := targetArticleID != "" || targetURL != "" || strings.Contains(taskLower, "load context") || strings.Contains(taskLower, "load article") || strings.Contains(taskLower, "deep ground") || (strings.Contains(taskLower, "summarize") && (strings.Contains(taskLower, "article") || strings.Contains(taskLower, "paper") || strings.Contains(taskLower, "architecture") || strings.Contains(taskLower, "benchmark")))

	if isArticleOp {
		res.ToolCalls = append(res.ToolCalls, "get_article_context")
		args := map[string]interface{}{}
		if targetArticleID != "" {
			args["article_id"] = targetArticleID
		}
		if targetURL != "" {
			args["url"] = targetURL
		}
		if strings.Contains(taskLower, "summar") || strings.Contains(taskLower, "bench") || strings.Contains(taskLower, "arch") {
			args["mode"] = "summary"
		}

		blocks, err := a.Executor.CallToolBlocks(ctx, "get_article_context", args)
		if err != nil {
			log.Printf("[A2A Agent] GetArticleContext error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("Load article context error: %v", err))
		} else {
			text, payload, insts := parseA2UIBlocks(blocks)
			if text != "" {
				finalOutput.WriteString(text)
			}
			if len(payload) > 0 {
				res.A2UIPayload = payload
			}
			if len(insts) > 0 {
				res.A2UIInstructions = insts
			}
		}
	}

	// 2. Global TL;DR / Executive Briefing / Multi-day Overview (ONLY when NOT targeting a single article)
	if len(res.ToolCalls) == 0 && (strings.Contains(taskLower, "tldr") || strings.Contains(taskLower, "briefing") || strings.Contains(taskLower, "overview") || strings.Contains(taskLower, "executive") || strings.Contains(taskLower, "digest") || strings.Contains(taskLower, "summary") || strings.Contains(taskLower, "summarize") || strings.Contains(taskLower, "week") || strings.Contains(taskLower, "month")) {
		res.ToolCalls = append(res.ToolCalls, "generate_tldr")
		tldrArgs := map[string]interface{}{}

		if strings.Contains(taskLower, "week") || strings.Contains(taskLower, "7 days") || strings.Contains(taskLower, "past 7") || strings.Contains(taskLower, "last 7") {
			tldrArgs["days"] = float64(7)
			tldrArgs["time_span"] = "the Past Week"
		} else if strings.Contains(taskLower, "month") || strings.Contains(taskLower, "30 days") || strings.Contains(taskLower, "past 30") || strings.Contains(taskLower, "last 30") {
			tldrArgs["days"] = float64(30)
			tldrArgs["time_span"] = "the Past 30 Days"
		}

		if strings.Contains(taskLower, "frontier") {
			tldrArgs["category"] = "Frontier Models"
		} else if strings.Contains(taskLower, "paper") || strings.Contains(taskLower, "arxiv") || strings.Contains(taskLower, "research") {
			tldrArgs["category"] = "AI Research Papers"
		} else if strings.Contains(taskLower, "cloud") || strings.Contains(taskLower, "gcp") || strings.Contains(taskLower, "infrastructure") {
			tldrArgs["category"] = "Google Cloud"
		} else if strings.Contains(taskLower, "tool") || strings.Contains(taskLower, "oss") {
			tldrArgs["category"] = "OSS & Tooling"
		}

		blocks, err := a.Executor.CallToolBlocks(ctx, "generate_tldr", tldrArgs)
		if err != nil {
			log.Printf("[A2A Agent] GenerateTLDR error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("TL;DR error: %v", err))
		} else {
			text, payload, insts := parseA2UIBlocks(blocks)
			if text != "" {
				finalOutput.WriteString(text)
			}
			if len(payload) > 0 {
				res.A2UIPayload = payload
			}
			if len(insts) > 0 {
				res.A2UIInstructions = insts
			}
		}
	}

	// 3. Crawl request check
	if len(res.ToolCalls) == 0 && (strings.Contains(taskLower, "crawl") || strings.Contains(taskLower, "scrape") || strings.Contains(taskLower, "refresh")) {
		res.ToolCalls = append(res.ToolCalls, "trigger_crawl")
		blocks, err := a.Executor.CallToolBlocks(ctx, "trigger_crawl", nil)
		if err != nil {
			log.Printf("[A2A Agent] TriggerCrawl error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("Crawl error: %v", err))
		} else {
			text, payload, insts := parseA2UIBlocks(blocks)
			if text != "" {
				finalOutput.WriteString(text)
			}
			if len(payload) > 0 {
				res.A2UIPayload = payload
			}
			if len(insts) > 0 {
				res.A2UIInstructions = insts
			}
		}
	}

	// 4. Telemetry check
	if len(res.ToolCalls) == 0 && (strings.Contains(taskLower, "telemetry") || strings.Contains(taskLower, "status") || strings.Contains(taskLower, "health") || strings.Contains(taskLower, "metric")) {
		res.ToolCalls = append(res.ToolCalls, "get_telemetry")
		blocks, err := a.Executor.CallToolBlocks(ctx, "get_telemetry", nil)
		if err != nil {
			log.Printf("[A2A Agent] GetTelemetry error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("Telemetry error: %v", err))
		} else {
			text, payload, insts := parseA2UIBlocks(blocks)
			if text != "" {
				finalOutput.WriteString(text)
			}
			if len(payload) > 0 {
				res.A2UIPayload = payload
			}
			if len(insts) > 0 {
				res.A2UIInstructions = insts
			}
		}
	}

	// 5. Newsletter request
	if len(res.ToolCalls) == 0 && strings.Contains(taskLower, "newsletter") {
		res.ToolCalls = append(res.ToolCalls, "get_newsletter")
		blocks, err := a.Executor.CallToolBlocks(ctx, "get_newsletter", nil)
		if err != nil {
			log.Printf("[A2A Agent] GetNewsletter error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("Newsletter error: %v", err))
		} else {
			text, payload, insts := parseA2UIBlocks(blocks)
			if text != "" {
				finalOutput.WriteString(text)
			}
			if len(payload) > 0 {
				res.A2UIPayload = payload
			}
			if len(insts) > 0 {
				res.A2UIInstructions = insts
			}
		}
	}

	// 6. Conversational follow-up & explanation
	if len(res.ToolCalls) == 0 && (strings.Contains(taskLower, "understand") || strings.Contains(taskLower, "explain") || strings.Contains(taskLower, "help me") || strings.Contains(taskLower, "what does") || strings.Contains(taskLower, "why") || strings.Contains(taskLower, "tell me more")) {
		res.ToolCalls = append(res.ToolCalls, "explain_concept")
		finalOutput.WriteString(fmt.Sprintf("Regarding your inquiry: \"%s\"\n\nThe AI Daily Brief platform continuously aggregates intelligence across 5 major streams: Frontier Models, Google Cloud, AI Research Papers, OSS Tooling, and Business. Use the cards below to explore details.", task))

		w := mcp.BuildWelcomeA2UI()
		b, _ := json.Marshal(w)
		_ = json.Unmarshal(b, &res.A2UIPayload)
		var insts []interface{}
		bi, _ := json.Marshal(w.Instructions)
		_ = json.Unmarshal(bi, &insts)
		res.A2UIInstructions = insts

		res.Output = strings.TrimSpace(finalOutput.String())
		res.DurationMs = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// 7. Article search / list query fallback
	if len(res.ToolCalls) == 0 {
		category := ""
		if strings.Contains(taskLower, "frontier") || strings.Contains(taskLower, "llm") || strings.Contains(taskLower, "foundation") {
			category = "Frontier Models"
		} else if strings.Contains(taskLower, "paper") || strings.Contains(taskLower, "arxiv") || strings.Contains(taskLower, "research") || strings.Contains(taskLower, "preprint") {
			category = "AI Research Papers"
		} else if strings.Contains(taskLower, "cloud") || strings.Contains(taskLower, "gcp") || strings.Contains(taskLower, "google cloud") || strings.Contains(taskLower, "infrastructure") {
			category = "Google Cloud"
		} else if strings.Contains(taskLower, "tool") || strings.Contains(taskLower, "oss") || strings.Contains(taskLower, "open-source") || strings.Contains(taskLower, "open source") || strings.Contains(taskLower, "librar") || strings.Contains(taskLower, "framework") {
			category = "OSS & Tooling"
		} else if strings.Contains(taskLower, "business") || strings.Contains(taskLower, "funding") || strings.Contains(taskLower, "venture") || strings.Contains(taskLower, "investment") {
			category = "AI Business & Infra"
		}

		company := ""
		if strings.Contains(taskLower, "google") {
			company = "Google"
		} else if strings.Contains(taskLower, "anthropic") {
			company = "Anthropic"
		} else if strings.Contains(taskLower, "openai") {
			company = "OpenAI"
		} else if strings.Contains(taskLower, "meta") {
			company = "Meta"
		} else if strings.Contains(taskLower, "deepseek") {
			company = "DeepSeek"
		} else if strings.Contains(taskLower, "mistral") {
			company = "Mistral"
		}

		res.ToolCalls = append(res.ToolCalls, "list_articles")
		args := map[string]interface{}{}
		if category != "" {
			args["category"] = category
		}
		if company != "" {
			args["company"] = company
		}
		if category == "" && company == "" && !strings.HasPrefix(taskLower, "list ") && !strings.HasPrefix(taskLower, "show ") {
			args["query"] = task
		}
		args["limit"] = 5

		blocks, err := a.Executor.CallToolBlocks(ctx, "list_articles", args)
		if err != nil {
			log.Printf("[A2A Agent] ListArticles error: %v", err)
			finalOutput.WriteString(fmt.Sprintf("List articles error: %v", err))
		} else {
			text, payload, insts := parseA2UIBlocks(blocks)
			if text != "" {
				finalOutput.WriteString(text)
			}
			if len(payload) > 0 {
				res.A2UIPayload = payload
			}
			if len(insts) > 0 {
				res.A2UIInstructions = insts
			}
		}
	}

	res.Output = strings.TrimSpace(finalOutput.String())
	res.DurationMs = time.Since(startTime).Milliseconds()
	return res, nil
}

// Chat executes interactive conversational research against the ToolExecutor
func (a *Agent) Chat(ctx context.Context, sessionID, message, articleID string) (string, error) {
	var groundPrefix string
	if articleID != "" {
		articleCtx, err := a.Executor.GetArticleContext(ctx, articleID, "")
		if err == nil && articleCtx != "" {
			groundPrefix = fmt.Sprintf("Grounded in Article [%s]:\n%s\n\n", articleID, articleCtx)
		}
	}

	taskPrompt := message
	if groundPrefix != "" {
		taskPrompt = groundPrefix + message
	}

	result, err := a.ExecuteTask(ctx, taskPrompt)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}
