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
	"fmt"
	"strings"

	"ai-daily-brief/internal/database"
)

// A2UICard represents a structured visual card for agent-to-UI rendering
type A2UICard struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "article_card", "tldr_card", "grounding_card", "telemetry_card"
	Title       string            `json:"title"`
	Subtitle    string            `json:"subtitle,omitempty"`
	Category    string            `json:"category,omitempty"`
	Badge       string            `json:"badge,omitempty"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Actions     []A2UIAction      `json:"actions,omitempty"`
}

// A2UIAction represents an interactive action button attached to a card
type A2UIAction struct {
	Label       string            `json:"label"`
	ActionType  string            `json:"action_type"` // "load_context", "open_url", "generate_tldr", "voice_stream"
	Payload     map[string]string `json:"payload"`
}

// CategoryBadge returns an emoji badge and styling tag for each stream category
func CategoryBadge(category string) (string, string) {
	cat := strings.ToLower(category)
	switch {
	case strings.Contains(cat, "frontier") || strings.Contains(cat, "model"):
		return "⚡ Frontier Models", "frontier"
	case strings.Contains(cat, "cloud") || strings.Contains(cat, "google"):
		return "☁️ Google Cloud & Compute", "cloud"
	case strings.Contains(cat, "paper") || strings.Contains(cat, "research") || strings.Contains(cat, "arxiv"):
		return "🔬 Research Papers", "papers"
	case strings.Contains(cat, "business") || strings.Contains(cat, "venture"):
		return "💼 AI Business & Market", "business"
	case strings.Contains(cat, "tool") || strings.Contains(cat, "oss") || strings.Contains(cat, "huggingface"):
		return "🛠️ Open-Source & Tooling", "tooling"
	default:
		return "🌐 AI Intelligence", "general"
	}
}

// FormatArticleCards renders a slice of NewsItem records into formatted A2UI markdown cards
func FormatArticleCards(items []database.NewsItem) string {
	if len(items) == 0 {
		return "📭 *No intelligence items found matching your query.*"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 📰 Intelligence Stream (%d records)\n\n", len(items)))

	for i, it := range items {
		badge, _ := CategoryBadge(string(it.Category))
		dateStr := it.PubDate
		if dateStr == "" {
			dateStr = it.CreatedAt.Format("Mon Jan 02, 15:04 MST")
		}

		sb.WriteString(fmt.Sprintf("┌─────────────────────────────────────────────────────────────\n"))
		sb.WriteString(fmt.Sprintf("│ **#%d: %s**\n", i+1, it.Title))
		sb.WriteString(fmt.Sprintf("│ %s • **%s** • *%s*\n", badge, it.Company, dateStr))
		sb.WriteString(fmt.Sprintf("├─────────────────────────────────────────────────────────────\n"))
		sb.WriteString(fmt.Sprintf("│ %s\n", it.Summary))
		sb.WriteString(fmt.Sprintf("├─────────────────────────────────────────────────────────────\n"))
		sb.WriteString(fmt.Sprintf("│ ⚡ **Actions:**\n"))
		sb.WriteString(fmt.Sprintf("│ • `[⚡ Load Article Context: %s]` -> Deep ground this article in conversation\n", it.ID))
		sb.WriteString(fmt.Sprintf("│ • `[🔗 Source URL]` -> %s\n", it.Link))
		sb.WriteString(fmt.Sprintf("└─────────────────────────────────────────────────────────────\n\n"))
	}

	return sb.String()
}

// FormatGroundedContextCard formats the fetched live webpage context into an A2UI grounding inspector card
func FormatGroundedContextCard(item database.NewsItem, bodyText string) string {
	badge, _ := CategoryBadge(string(item.Category))
	var sb strings.Builder

	sb.WriteString("┌─────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("│ 🔬 **GROUNDED CONTEXT INSPECTOR: %s**\n", item.Title))
	sb.WriteString(fmt.Sprintf("│ %s • Source: **%s** • URL: %s\n", badge, item.Company, item.Link))
	sb.WriteString("├─────────────────────────────────────────────────────────────\n")
	sb.WriteString("│ 📄 **Sanitized Extracted Text Body:**\n│\n")

	lines := strings.Split(bodyText, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			sb.WriteString(fmt.Sprintf("│ %s\n", l))
		}
	}

	sb.WriteString("├─────────────────────────────────────────────────────────────\n")
	sb.WriteString("│ ⚡ **Suggested Agent Followups:**\n")
	sb.WriteString("│ 1. \"Summarize the core technical architecture and benchmark scores.\"\n")
	sb.WriteString("│ 2. \"How does this compare to competitor frontier releases?\"\n")
	sb.WriteString("│ 3. \"What are the enterprise cloud infrastructure implications?\"\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────\n")

	return sb.String()
}

// FormatExecutiveTLDRCard formats the daily executive TL;DR into an A2UI briefing card
func FormatExecutiveTLDRCard(tldrText string, dateStr string) string {
	var sb strings.Builder
	sb.WriteString("┌─────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("│ ⚡ **EXECUTIVE STRATEGIC BRIEFING (%s)**\n", dateStr))
	sb.WriteString("├─────────────────────────────────────────────────────────────\n")
	sb.WriteString(tldrText + "\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────\n")
	sb.WriteString("│ ⚡ **Actions:**\n")
	sb.WriteString("│ • `[🎙️ Voice Dialogue]` -> Start live conversational audio stream on this briefing\n")
	sb.WriteString("│ • `[📧 Send Digest]` -> Dispatch HTML briefing to all subscribers\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────\n")
	return sb.String()
}

// FormatTelemetryCard formats server & model telemetry for the control plane
func FormatTelemetryCard(totalItems int64, model string, authMode string, projectID string, location string) string {
	var sb strings.Builder
	sb.WriteString("┌─────────────────────────────────────────────────────────────\n")
	sb.WriteString("│ 🛡️ **AI DAILY BRIEF MCP CONTROL PLANE TELEMETRY**\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("│ • **Indexed Database Records:** %d articles\n", totalItems))
	sb.WriteString(fmt.Sprintf("│ • **Active Model Engine:**       `%s`\n", model))
	sb.WriteString(fmt.Sprintf("│ • **Auth Mode:**                 `%s`\n", authMode))
	if authMode == "vertex_adc" {
		projDisplay := projectID
		if projDisplay == "" {
			projDisplay = "(Auto-detected from GCP ADC / Compute Metadata)"
		}
		sb.WriteString(fmt.Sprintf("│ • **GCP Project ID:**            %s\n", projDisplay))
		sb.WriteString(fmt.Sprintf("│ • **GCP Region / Location:**     %s\n", location))
	}
	sb.WriteString("│ • **Runtime Protocol:**          Model Context Protocol (JSON-RPC 2.0 / SSE)\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────\n")
	return sb.String()
}
