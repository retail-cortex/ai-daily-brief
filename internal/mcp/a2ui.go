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

	"ai-daily-brief/internal/crawler"
	"ai-daily-brief/internal/database"
)

const (
	// A2UIVersion defines the primary A2UI specification version (v0.9 is GA on Gemini Enterprise)
	A2UIVersion = "v0.9"

	// MIMETypeA2UIJSON is the standard MIME type for Agent-to-UI JSON components
	MIMETypeA2UIJSON = "application/a2ui+json"
	// MIMETypeA2UIJSONAlt is the alternate MIME type used by Gemini Enterprise
	MIMETypeA2UIJSONAlt = "application/json+a2ui"
	// MIMETypeMarkdown is standard text/markdown
	MIMETypeMarkdown = "text/markdown"
	// MIMETypeHTML is standard text/html
	MIMETypeHTML = "text/html"
	// MIMETypeJSON is standard application/json
	MIMETypeJSON = "application/json"

	// CatalogIDMaterial is the upstream A2UI v0.9 Material Design 3 catalog identifier
	CatalogIDMaterial = "https://a2ui.org/specification/v0_9/material_catalog.json"
	// CatalogIDComposite is the Gemini Enterprise composite catalog identifier
	CatalogIDComposite = "https://www.gstatic.com/vertexaisearch/a2ui/v0_9/gemini_enterprise_composite_catalog.json"
	// CatalogIDBasic is the upstream A2UI v0.9 Basic catalog identifier
	CatalogIDBasic = "https://a2ui.org/specification/v0_9/basic_catalog.json"
	// CatalogIDA2UI is the primary A2UI upstream catalog identifier
	CatalogIDA2UI = "https://a2ui.org"
)

// DefaultA2UITheme provides standard Material 3 theme properties
var DefaultA2UITheme = map[string]interface{}{
	"primaryColor": "#1a73e8",
	"font":         "Roboto",
}

// A2UIPayload is the canonical A2UI v0.9/v0.8 compliant data container object
type A2UIPayload struct {
	Version          string                 `json:"version"`
	CreateSurface    *A2UICreateSurface     `json:"createSurface,omitempty"`
	UpdateComponents *A2UIUpdateComponents  `json:"updateComponents,omitempty"`
	Instructions     []A2UIInstruction      `json:"instructions,omitempty"`
}

// A2UIInstruction represents an instruction in the A2UI stream
type A2UIInstruction struct {
	Version          string                `json:"version"`
	CreateSurface    *A2UICreateSurface    `json:"createSurface,omitempty"`
	UpdateComponents *A2UIUpdateComponents `json:"updateComponents,omitempty"`
}

// A2UICreateSurface initializes an A2UI surface
type A2UICreateSurface struct {
	SurfaceID string                 `json:"surfaceId"`
	CatalogID string                 `json:"catalogId"`
	Theme     map[string]interface{} `json:"theme,omitempty"`
}

// A2UIUpdateComponents updates or registers components in an A2UI surface
type A2UIUpdateComponents struct {
	SurfaceID  string          `json:"surfaceId"`
	Components []A2UIComponent `json:"components"`
}

// A2UIComponent represents a single UI component in the A2UI catalog
type A2UIComponent map[string]interface{}

// CategoryBadge returns a styling badge and tag for each stream category
func CategoryBadge(category string) (string, string) {
	cat := strings.ToLower(category)
	switch {
	case strings.Contains(cat, "frontier") || strings.Contains(cat, "model"):
		return "Frontier Models", "frontier"
	case strings.Contains(cat, "cloud") || strings.Contains(cat, "google"):
		return "Google Cloud & Compute", "cloud"
	case strings.Contains(cat, "paper") || strings.Contains(cat, "research") || strings.Contains(cat, "arxiv"):
		return "Research Papers", "papers"
	case strings.Contains(cat, "business") || strings.Contains(cat, "venture"):
		return "AI Business & Market", "business"
	case strings.Contains(cat, "tool") || strings.Contains(cat, "oss") || strings.Contains(cat, "huggingface"):
		return "Open-Source & Tooling", "tooling"
	default:
		return "AI Intelligence", "general"
	}
}

// BuildArticleCardsA2UI constructs canonical A2UI v0.9 Material component tree instructions for article cards
func BuildArticleCardsA2UI(items []database.NewsItem) A2UIPayload {
	surfaceID := "article_list_surface"
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	if len(items) == 0 {
		components := []A2UIComponent{
			{
				"id":        "root",
				"component": "MaterialCard",
				"children":  []string{"empty-text"},
			},
			{
				"id":        "empty-text",
				"component": "MaterialText",
				"text":      "No intelligence items found matching your filter.",
				"usageHint": "body1",
			},
		}
		updateComponents := &A2UIUpdateComponents{
			SurfaceID:  surfaceID,
			Components: components,
		}
		return A2UIPayload{
			Version:          A2UIVersion,
			CreateSurface:    createSurface,
			UpdateComponents: updateComponents,
			Instructions: []A2UIInstruction{
				{Version: A2UIVersion, CreateSurface: createSurface},
				{Version: A2UIVersion, UpdateComponents: updateComponents},
			},
		}
	}

	colChildren := []string{"list-header"}
	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"list-col"},
		},
		{
			"id":        "list-col",
			"component": "MaterialColumn",
			"align":     "stretch",
		},
		{
			"id":        "list-header",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"list-meta", "list-badge"},
		},
		{
			"id":        "list-meta",
			"component": "MaterialColumn",
			"children":  []string{"list-title", "list-sub"},
		},
		{
			"id":        "list-title",
			"component": "MaterialText",
			"text":      "Intelligence Stream Results",
			"usageHint": "h2",
		},
		{
			"id":        "list-sub",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Showing %d top intelligence briefings from persistent storage", len(items)),
			"usageHint": "caption",
		},
		{
			"id":        "list-badge",
			"component": "MaterialText",
			"text":      fmt.Sprintf("[%d Briefings]", len(items)),
			"usageHint": "caption",
		},
	}

	for i, it := range items {
		idx := i + 1
		badge, _ := CategoryBadge(string(it.Category))
		dateStr := it.PubDate
		if dateStr == "" {
			dateStr = it.CreatedAt.Format("Mon Jan 02, 15:04 MST")
		}
		summaryText := it.Summary
		if len(summaryText) > 300 {
			summaryText = summaryText[:297] + "..."
		}

		divID := fmt.Sprintf("art-div-%d", idx)
		titleID := fmt.Sprintf("art-title-%d", idx)
		metaID := fmt.Sprintf("art-meta-%d", idx)
		summaryID := fmt.Sprintf("art-summary-%d", idx)
		actionsID := fmt.Sprintf("art-actions-%d", idx)
		actContextID := fmt.Sprintf("act-context-%d", idx)
		actSummaryID := fmt.Sprintf("act-summary-%d", idx)
		actOpenID := fmt.Sprintf("act-open-%d", idx)

		colChildren = append(colChildren, divID, titleID, metaID, summaryID, actionsID)

		components = append(components,
			A2UIComponent{
				"id":        divID,
				"component": "MaterialDivider",
			},
			A2UIComponent{
				"id":        titleID,
				"component": "MaterialText",
				"text":      it.Title,
				"usageHint": "h2",
			},
			A2UIComponent{
				"id":        metaID,
				"component": "MaterialText",
				"text":      fmt.Sprintf("[%s] • %s • %s", badge, it.Company, dateStr),
				"usageHint": "caption",
			},
			A2UIComponent{
				"id":        summaryID,
				"component": "MaterialText",
				"text":      summaryText,
				"usageHint": "body1",
			},
			A2UIComponent{
				"id":        actionsID,
				"component": "MaterialRow",
				"children":  []string{actContextID, actSummaryID, actOpenID},
			},
			A2UIComponent{
				"id":        actContextID,
				"component": "MaterialButton",
				"label":     "Load Article Context",
				"variant":   "flat",
				"action": map[string]interface{}{
					"event": map[string]interface{}{
						"name": "load_context",
						"context": map[string]string{
							"article_id": it.ID,
							"url":        it.Link,
							"prompt":     fmt.Sprintf("Load context for article %s", it.ID),
						},
					},
				},
			},
			A2UIComponent{
				"id":        actSummaryID,
				"component": "MaterialButton",
				"label":     "Summarize Article",
				"variant":   "stroked",
				"action": map[string]interface{}{
					"event": map[string]interface{}{
						"name": "agent_prompt",
						"context": map[string]string{
							"article_id": it.ID,
							"url":        it.Link,
							"prompt":     fmt.Sprintf("Summarize article %s", it.ID),
						},
					},
				},
			},
			A2UIComponent{
				"id":        actOpenID,
				"component": "MaterialButton",
				"label":     "Open Source",
				"variant":   "stroked",
				"action": map[string]interface{}{
					"event": map[string]interface{}{
						"name": "open_url",
						"context": map[string]string{
							"url": it.Link,
						},
					},
				},
			},
		)
	}

	for j := range components {
		if components[j]["id"] == "list-col" {
			components[j]["children"] = colChildren
			break
		}
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// BuildGroundedContextA2UI constructs canonical A2UI v0.9 instructions for grounded article inspection
func BuildGroundedContextA2UI(item database.NewsItem, bodyText string) A2UIPayload {
	surfaceID := fmt.Sprintf("grounding_surface_%s", item.ID)
	badge, _ := CategoryBadge(string(item.Category))
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	clampedBody := bodyText
	if len(clampedBody) > 1200 {
		clampedBody = clampedBody[:1197] + "..."
	}

	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"ground-col"},
		},
		{
			"id":        "ground-col",
			"component": "MaterialColumn",
			"align":     "stretch",
			"children":  []string{"ground-hdr", "ground-body", "ground-div", "ground-actions"},
		},
		{
			"id":        "ground-hdr",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"ground-meta", "ground-badge"},
		},
		{
			"id":        "ground-meta",
			"component": "MaterialColumn",
			"children":  []string{"ground-title", "ground-sub"},
		},
		{
			"id":        "ground-title",
			"component": "MaterialText",
			"text":      item.Title,
			"usageHint": "h2",
		},
		{
			"id":        "ground-sub",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Source: %s • %s", item.Company, item.Link),
			"usageHint": "caption",
		},
		{
			"id":        "ground-badge",
			"component": "MaterialText",
			"text":      fmt.Sprintf("[%s]", badge),
			"usageHint": "caption",
		},
		{
			"id":        "ground-body",
			"component": "MaterialText",
			"text":      clampedBody,
			"usageHint": "body1",
		},
		{
			"id":        "ground-div",
			"component": "MaterialDivider",
		},
		{
			"id":        "ground-actions",
			"component": "MaterialRow",
			"children":  []string{"btn-arch", "btn-comp", "btn-source"},
		},
		{
			"id":        "btn-arch",
			"component": "MaterialButton",
			"label":     "Summarize Article",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "agent_prompt",
					"context": map[string]string{
						"prompt":     fmt.Sprintf("Summarize article %s", item.ID),
						"article_id": item.ID,
						"url":        item.Link,
					},
				},
			},
		},
		{
			"id":        "btn-comp",
			"component": "MaterialButton",
			"label":     "Compare Competitor Models",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "agent_prompt",
					"context": map[string]string{
						"prompt":     fmt.Sprintf("Compare the findings in %s with recent competitor releases", item.Title),
						"article_id": item.ID,
					},
				},
			},
		},
		{
			"id":        "btn-source",
			"component": "MaterialButton",
			"label":     "Open Source",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "open_url",
					"context": map[string]string{
						"url": item.Link,
					},
				},
			},
		},
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// BuildArticleSummaryA2UI constructs canonical A2UI v0.9 instructions for technical summary cards
func BuildArticleSummaryA2UI(item database.NewsItem, summaryText string) A2UIPayload {
	surfaceID := fmt.Sprintf("summary_surface_%s", item.ID)
	badge, _ := CategoryBadge(string(item.Category))
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	clampedSummary := summaryText
	if len(clampedSummary) > 1500 {
		clampedSummary = clampedSummary[:1497] + "..."
	}

	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"sum-col"},
		},
		{
			"id":        "sum-col",
			"component": "MaterialColumn",
			"align":     "stretch",
			"children":  []string{"sum-hdr", "sum-content", "sum-div", "sum-actions"},
		},
		{
			"id":        "sum-hdr",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"sum-meta", "sum-badge"},
		},
		{
			"id":        "sum-meta",
			"component": "MaterialColumn",
			"children":  []string{"sum-title", "sum-sub"},
		},
		{
			"id":        "sum-title",
			"component": "MaterialText",
			"text":      item.Title,
			"usageHint": "h2",
		},
		{
			"id":        "sum-sub",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Technical Summary • %s", item.Company),
			"usageHint": "caption",
		},
		{
			"id":        "sum-badge",
			"component": "MaterialText",
			"text":      fmt.Sprintf("[%s]", badge),
			"usageHint": "caption",
		},
		{
			"id":        "sum-content",
			"component": "MaterialText",
			"text":      clampedSummary,
			"usageHint": "body1",
		},
		{
			"id":        "sum-div",
			"component": "MaterialDivider",
		},
		{
			"id":        "sum-actions",
			"component": "MaterialRow",
			"children":  []string{"btn-src", "btn-raw", "btn-tldr"},
		},
		{
			"id":        "btn-src",
			"component": "MaterialButton",
			"label":     "Open Source Paper",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "open_url",
					"context": map[string]string{
						"url": item.Link,
					},
				},
			},
		},
		{
			"id":        "btn-raw",
			"component": "MaterialButton",
			"label":     "Inspect Raw Grounding Context",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "load_context",
					"context": map[string]string{
						"article_id": item.ID,
						"url":        item.Link,
						"prompt":     fmt.Sprintf("Load context for article %s", item.ID),
					},
				},
			},
		},
		{
			"id":        "btn-tldr",
			"component": "MaterialButton",
			"label":     "Today's Stream Briefing",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"prompt": "Generate today's executive intelligence summary",
					},
				},
			},
		},
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// BuildExecutiveTLDRA2UI constructs canonical A2UI v0.9 instructions for daily executive briefings
func BuildExecutiveTLDRA2UI(tldrText string, dateStr string) A2UIPayload {
	surfaceID := fmt.Sprintf("tldr_surface_%s", dateStr)
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	clampedTLDR := tldrText
	if len(clampedTLDR) > 2000 {
		clampedTLDR = clampedTLDR[:1997] + "..."
	}

	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"tldr-col"},
		},
		{
			"id":        "tldr-col",
			"component": "MaterialColumn",
			"align":     "stretch",
			"children":  []string{"tldr-hdr", "tldr-body", "tldr-div", "tldr-actions"},
		},
		{
			"id":        "tldr-hdr",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"tldr-meta", "tldr-badge"},
		},
		{
			"id":        "tldr-meta",
			"component": "MaterialColumn",
			"children":  []string{"tldr-title", "tldr-sub"},
		},
		{
			"id":        "tldr-title",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Executive Strategic Briefing (%s)", dateStr),
			"usageHint": "h2",
		},
		{
			"id":        "tldr-sub",
			"component": "MaterialText",
			"text":      "Synthesized cross-stream intelligence via Vertex AI Gemini",
			"usageHint": "caption",
		},
		{
			"id":        "tldr-badge",
			"component": "MaterialText",
			"text":      "[Strategic Brief]",
			"usageHint": "caption",
		},
		{
			"id":        "tldr-body",
			"component": "MaterialText",
			"text":      clampedTLDR,
			"usageHint": "body1",
		},
		{
			"id":        "tldr-div",
			"component": "MaterialDivider",
		},
		{
			"id":        "tldr-actions",
			"component": "MaterialRow",
			"children":  []string{"btn-week", "btn-month"},
		},
		{
			"id":        "btn-week",
			"component": "MaterialButton",
			"label":     "Weekly Overview (7 Days)",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"days":      "7",
						"time_span": "the Past Week",
						"prompt":    "Show me the overview for the last week",
					},
				},
			},
		},
		{
			"id":        "btn-month",
			"component": "MaterialButton",
			"label":     "Monthly Overview (30 Days)",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"days":      "30",
						"time_span": "the Past 30 Days",
						"prompt":    "Show me the overview for the last 30 days",
					},
				},
			},
		},
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// BuildTelemetryA2UI constructs canonical A2UI v0.9 instructions for control plane telemetry
func BuildTelemetryA2UI(totalItems int64, model string, authMode string, projectID string, location string) A2UIPayload {
	surfaceID := "telemetry_surface"
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"telem-col"},
		},
		{
			"id":        "telem-col",
			"component": "MaterialColumn",
			"align":     "stretch",
			"children":  []string{"telem-hdr", "telem-body", "telem-div", "telem-actions"},
		},
		{
			"id":        "telem-hdr",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"telem-meta", "telem-badge"},
		},
		{
			"id":        "telem-meta",
			"component": "MaterialColumn",
			"children":  []string{"telem-title", "telem-sub"},
		},
		{
			"id":        "telem-title",
			"component": "MaterialText",
			"text":      "AI Daily Brief Control Plane Telemetry",
			"usageHint": "h2",
		},
		{
			"id":        "telem-sub",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Model: %s • Auth: %s • Region: %s", model, authMode, location),
			"usageHint": "caption",
		},
		{
			"id":        "telem-badge",
			"component": "MaterialText",
			"text":      fmt.Sprintf("[%d Articles]", totalItems),
			"usageHint": "caption",
		},
		{
			"id":        "telem-body",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Active GCP Project: %s\nVertex Model: %s\nAlloyDB Persistent Articles: %d", projectID, model, totalItems),
			"usageHint": "body1",
		},
		{
			"id":        "telem-div",
			"component": "MaterialDivider",
		},
		{
			"id":        "telem-actions",
			"component": "MaterialRow",
			"children":  []string{"btn-crawl-telem"},
		},
		{
			"id":        "btn-crawl-telem",
			"component": "MaterialButton",
			"label":     "Trigger Live Crawl",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "trigger_crawl",
					"context": map[string]string{
						"prompt": "Trigger live crawl of AI feeds",
					},
				},
			},
		},
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// BuildCrawlBatchA2UI constructs canonical A2UI v0.9 instructions for crawl batch results
func BuildCrawlBatchA2UI(res *crawler.BatchResult) A2UIPayload {
	surfaceID := "crawl_batch_surface"
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	if res == nil {
		components := []A2UIComponent{
			{
				"id":        "root",
				"component": "MaterialCard",
				"children":  []string{"crawl-empty-text"},
			},
			{
				"id":        "crawl-empty-text",
				"component": "MaterialText",
				"text":      "No crawl result available.",
				"usageHint": "body1",
			},
		}
		updateComponents := &A2UIUpdateComponents{
			SurfaceID:  surfaceID,
			Components: components,
		}
		return A2UIPayload{
			Version:          A2UIVersion,
			CreateSurface:    createSurface,
			UpdateComponents: updateComponents,
			Instructions: []A2UIInstruction{
				{
					Version:       A2UIVersion,
					CreateSurface: createSurface,
				},
				{
					Version:          A2UIVersion,
					UpdateComponents: updateComponents,
				},
			},
		}
	}

	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"crawl-col"},
		},
		{
			"id":        "crawl-col",
			"component": "MaterialColumn",
			"align":     "stretch",
			"children":  []string{"crawl-hdr", "crawl-body", "crawl-div", "crawl-actions"},
		},
		{
			"id":        "crawl-hdr",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"crawl-meta", "crawl-badge"},
		},
		{
			"id":        "crawl-meta",
			"component": "MaterialColumn",
			"children":  []string{"crawl-title", "crawl-sub"},
		},
		{
			"id":        "crawl-title",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Crawl Batch Run: %s", res.RunID),
			"usageHint": "h2",
		},
		{
			"id":        "crawl-sub",
			"component": "MaterialText",
			"text":      fmt.Sprintf("Status: %s • Inserted: %d • Total in DB: %d", res.Status, res.NewItemsInserted, res.TotalInDB),
			"usageHint": "caption",
		},
		{
			"id":        "crawl-badge",
			"component": "MaterialText",
			"text":      fmt.Sprintf("[%s]", res.Status),
			"usageHint": "caption",
		},
		{
			"id":        "crawl-body",
			"component": "MaterialText",
			"text":      res.Log,
			"usageHint": "body1",
		},
		{
			"id":        "crawl-div",
			"component": "MaterialDivider",
		},
		{
			"id":        "crawl-actions",
			"component": "MaterialRow",
			"children":  []string{"btn-tldr-crawl"},
		},
		{
			"id":        "btn-tldr-crawl",
			"component": "MaterialButton",
			"label":     "Generate Executive Briefing",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"prompt": "Generate today's executive intelligence summary",
					},
				},
			},
		},
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// BuildWelcomeA2UI constructs canonical A2UI v0.9 Material component tree instructions for the initial welcome hub
func BuildWelcomeA2UI() A2UIPayload {
	surfaceID := "welcome_surface"
	createSurface := &A2UICreateSurface{
		SurfaceID: surfaceID,
		CatalogID: CatalogIDMaterial,
		Theme:     DefaultA2UITheme,
	}

	components := []A2UIComponent{
		{
			"id":        "root",
			"component": "MaterialCard",
			"children":  []string{"welcome-col"},
		},
		{
			"id":        "welcome-col",
			"component": "MaterialColumn",
			"align":     "stretch",
			"children": []string{
				"welcome-header",
				"welcome-desc",
				"welcome-div-1",
				"sec-1-label",
				"actions-row-1",
				"welcome-div-2",
				"sec-2-label",
				"actions-row-2",
				"welcome-div-3",
				"sec-3-label",
				"actions-row-3",
			},
		},
		{
			"id":        "welcome-header",
			"component": "MaterialRow",
			"justify":   "spaceBetween",
			"align":     "center",
			"children":  []string{"welcome-meta", "welcome-badge"},
		},
		{
			"id":        "welcome-meta",
			"component": "MaterialColumn",
			"children":  []string{"welcome-title", "welcome-sub"},
		},
		{
			"id":        "welcome-title",
			"component": "MaterialText",
			"text":      "AI Daily Brief Control Center",
			"usageHint": "h2",
		},
		{
			"id":        "welcome-sub",
			"component": "MaterialText",
			"text":      "Enterprise AI intelligence stream & multi-model research assistant",
			"usageHint": "caption",
		},
		{
			"id":        "welcome-badge",
			"component": "MaterialText",
			"text":      "[Live Agent]",
			"usageHint": "caption",
		},
		{
			"id":        "welcome-desc",
			"component": "MaterialText",
			"text":      "Select a strategic briefing, explore domain streams, or trigger on-demand control plane operations below:",
			"usageHint": "body1",
		},
		{
			"id":        "welcome-div-1",
			"component": "MaterialDivider",
		},
		{
			"id":        "sec-1-label",
			"component": "MaterialText",
			"text":      "EXECUTIVE BRIEFINGS & SYNTHESES",
			"usageHint": "caption",
		},
		{
			"id":        "actions-row-1",
			"component": "MaterialRow",
			"children":  []string{"btn-today-tldr", "btn-week-tldr", "btn-month-tldr"},
		},
		{
			"id":        "btn-today-tldr",
			"component": "MaterialButton",
			"label":     "Today's Briefing",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"prompt": "Generate today's executive intelligence summary",
					},
				},
			},
		},
		{
			"id":        "btn-week-tldr",
			"component": "MaterialButton",
			"label":     "Weekly Overview (7 Days)",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"days":      "7",
						"time_span": "the Past Week",
						"prompt":    "Show me the overview for the last week",
					},
				},
			},
		},
		{
			"id":        "btn-month-tldr",
			"component": "MaterialButton",
			"label":     "Monthly Overview (30 Days)",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "generate_tldr",
					"context": map[string]string{
						"days":      "30",
						"time_span": "the Past 30 Days",
						"prompt":    "Show me the overview for the last 30 days",
					},
				},
			},
		},
		{
			"id":        "welcome-div-2",
			"component": "MaterialDivider",
		},
		{
			"id":        "sec-2-label",
			"component": "MaterialText",
			"text":      "INTELLIGENCE STREAMS",
			"usageHint": "caption",
		},
		{
			"id":        "actions-row-2",
			"component": "MaterialRow",
			"children":  []string{"btn-frontier", "btn-papers", "btn-gcp", "btn-oss"},
		},
		{
			"id":        "btn-frontier",
			"component": "MaterialButton",
			"label":     "Frontier Models",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "list_articles",
					"context": map[string]string{
						"category": "Frontier Models",
						"prompt":   "List latest Frontier Models intelligence",
					},
				},
			},
		},
		{
			"id":        "btn-papers",
			"component": "MaterialButton",
			"label":     "Research Papers",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "list_articles",
					"context": map[string]string{
						"category": "AI Research Papers",
						"prompt":   "List latest AI research papers",
					},
				},
			},
		},
		{
			"id":        "btn-gcp",
			"component": "MaterialButton",
			"label":     "Google Cloud & Infra",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "list_articles",
					"context": map[string]string{
						"category": "Google Cloud",
						"prompt":   "List latest Google Cloud AI and infrastructure releases",
					},
				},
			},
		},
		{
			"id":        "btn-oss",
			"component": "MaterialButton",
			"label":     "OSS & Tooling",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "list_articles",
					"context": map[string]string{
						"category": "OSS & Tooling",
						"prompt":   "List latest open-source AI tooling and libraries",
					},
				},
			},
		},
		{
			"id":        "welcome-div-3",
			"component": "MaterialDivider",
		},
		{
			"id":        "sec-3-label",
			"component": "MaterialText",
			"text":      "OPERATIONS & CONTROL PLANE",
			"usageHint": "caption",
		},
		{
			"id":        "actions-row-3",
			"component": "MaterialRow",
			"children":  []string{"btn-crawl", "btn-telemetry"},
		},
		{
			"id":        "btn-crawl",
			"component": "MaterialButton",
			"label":     "Trigger Live Ingestion Crawl",
			"variant":   "flat",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "trigger_crawl",
					"context": map[string]string{
						"prompt": "Trigger live crawl of AI feeds",
					},
				},
			},
		},
		{
			"id":        "btn-telemetry",
			"component": "MaterialButton",
			"label":     "System & DB Telemetry",
			"variant":   "stroked",
			"action": map[string]interface{}{
				"event": map[string]interface{}{
					"name": "get_system_status",
					"context": map[string]string{
						"prompt": "Inspect system telemetry, database status, and active models",
					},
				},
			},
		},
	}

	updateComponents := &A2UIUpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
	}

	return A2UIPayload{
		Version:          A2UIVersion,
		CreateSurface:    createSurface,
		UpdateComponents: updateComponents,
		Instructions: []A2UIInstruction{
			{
				Version:       A2UIVersion,
				CreateSurface: createSurface,
			},
			{
				Version:          A2UIVersion,
				UpdateComponents: updateComponents,
			},
		},
	}
}

// MarshalA2UIJSON serializes an A2UI instruction collection or payload to a JSON string
func MarshalA2UIJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
