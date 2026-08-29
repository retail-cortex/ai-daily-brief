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

package mailer

import (
	"strings"
	"testing"
	"time"

	"ai-daily-brief/internal/database"
)

func TestMarkdownToInlineHTML(t *testing.T) {
	input := `### Section Header
This is a **bold statement** with ` + "`code block`" + ` and [link](https://example.com).

> An important quote block.

- Item 1
- Item 2
`

	html := MarkdownToInlineHTML(input)

	if !strings.Contains(html, "<h3") || !strings.Contains(html, "Section Header") {
		t.Errorf("Expected H3 tag in HTML output, got: %s", html)
	}

	if !strings.Contains(html, "<strong") || !strings.Contains(html, "bold statement") {
		t.Errorf("Expected <strong> tag for bold text, got: %s", html)
	}

	if !strings.Contains(html, "<code") || !strings.Contains(html, "code block") {
		t.Errorf("Expected <code> tag for inline code, got: %s", html)
	}

	if !strings.Contains(html, `<a href="https://example.com"`) {
		t.Errorf("Expected anchor tag for link, got: %s", html)
	}

	if !strings.Contains(html, "<blockquote") || !strings.Contains(html, "An important quote block.") {
		t.Errorf("Expected blockquote in HTML output, got: %s", html)
	}

	if !strings.Contains(html, "<ul") || !strings.Contains(html, "<li") {
		t.Errorf("Expected unordered list in HTML output, got: %s", html)
	}
}

func TestGenerateNewsletterHTML(t *testing.T) {
	items := []database.NewsItem{
		{
			ID:        "item-1",
			RunDate:   "2026-08-25",
			PubDate:   "2026-08-25",
			Company:   "Google",
			Category:  database.CategoryFrontierModels,
			Title:     "Gemini 3.7 Release",
			Summary:   "New multimodal flagship release with reasoning.",
			Link:      "https://example.com/gemini-3.7",
			CreatedAt: time.Now(),
		},
		{
			ID:        "item-2",
			RunDate:   "2026-08-25",
			PubDate:   "2026-08-25",
			Company:   "Google Cloud",
			Category:  database.CategoryGoogleCloud,
			Title:     "Vertex AI Model Garden Update",
			Summary:   "New endpoint serving options.",
			Link:      "https://example.com/gcp",
			CreatedAt: time.Now(),
		},
	}

	dateStr := "Monday, Aug 25, 2026"
	html := GenerateNewsletterHTML(items, dateStr)

	if !strings.Contains(html, "Daily AI &amp; Cloud Intelligence Digest") && !strings.Contains(html, "Daily AI & Cloud Intelligence Digest") {
		t.Errorf("Expected newsletter header in HTML output")
	}

	if !strings.Contains(html, "Gemini 3.7 Release") {
		t.Errorf("Expected item 1 title in HTML output")
	}

	if !strings.Contains(html, "Vertex AI Model Garden Update") {
		t.Errorf("Expected item 2 title in HTML output")
	}

	if !strings.Contains(html, dateStr) {
		t.Errorf("Expected date string in HTML output")
	}
}
