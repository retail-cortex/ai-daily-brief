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

package crawler

import (
	"strings"
	"testing"
	"time"

	"ai-daily-brief/internal/database"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "OpenAI unveils GPT-5 with multimodal reasoning - TechCrunch",
			expected: "OpenAI unveils GPT-5 with multimodal reasoning",
		},
		{
			input:    "Google Cloud Vertex AI &amp; TPU v6 Update - Reuters",
			expected: "Google Cloud Vertex AI & TPU v6 Update",
		},
		{
			input:    "<b>Anthropic</b> announces Claude 3.7 - The Verge",
			expected: "Anthropic announces Claude 3.7",
		},
	}

	for _, tt := range tests {
		got := cleanTitle(tt.input)
		if got != tt.expected {
			t.Errorf("cleanTitle(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestCleanSummary(t *testing.T) {
	raw := "<p>This is a <b>test</b> description with HTML tags.</p>"
	got := cleanSummary(raw, 50)
	expected := "This is a test description with HTML tags."
	if got != expected {
		t.Errorf("cleanSummary(%q) = %q, expected %q", raw, got, expected)
	}

	// Truncation test
	longText := "A very long summary that exceeds the maximum length configured for summaries."
	truncated := cleanSummary(longText, 20)
	if len(truncated) > 20 || !strings.HasSuffix(truncated, "...") {
		t.Errorf("cleanSummary truncation failed, got: %s", truncated)
	}
}

func TestMakeID(t *testing.T) {
	id1 := makeID("google", "https://news.google.com/item1")
	id2 := makeID("google", "https://news.google.com/item1")
	id3 := makeID("google", "https://news.google.com/item2")

	if id1 != id2 {
		t.Errorf("makeID should produce deterministic hashes, got %s and %s", id1, id2)
	}
	if id1 == id3 {
		t.Errorf("makeID should produce different IDs for different links, got %s and %s", id1, id3)
	}
	if !strings.HasPrefix(id1, "google-") {
		t.Errorf("makeID should include prefix, got %s", id1)
	}
}

func TestSynthesizeSmartSummary(t *testing.T) {
	summary1 := synthesizeSmartSummary("New reasoning benchmark for LLMs", "Research Lab", database.CategoryResearchPapers)
	if !strings.Contains(summary1, "benchmark") {
		t.Errorf("Expected benchmark summary, got %s", summary1)
	}

	summary2 := synthesizeSmartSummary("Startup raises $50M in funding round", "AI Deals", database.CategoryBusinessInfra)
	if !strings.Contains(summary2, "Venture capital") && !strings.Contains(summary2, "funding") {
		t.Errorf("Expected funding summary, got %s", summary2)
	}

	summary3 := synthesizeSmartSummary("vLLM releases high throughput engine", "OSS", database.CategoryOSSTooling)
	if !strings.Contains(summary3, "inference engine") {
		t.Errorf("Expected inference summary, got %s", summary3)
	}
}

func TestBatchRunDeduplication(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory DB: %v", err)
	}

	// Insert an existing article
	link := "https://example.com/test-article"
	item := database.NewsItem{
		ID:        makeID("test", link),
		RunDate:   time.Now().Format("2006-01-02"),
		PubDate:   time.Now().Format("2006-01-02"),
		Company:   "Test Co",
		Category:  database.CategoryFrontierModels,
		Title:     "Existing Test Article",
		Summary:   "Summary of existing article",
		Link:      link,
		RawSource: "Test",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("Failed to insert item: %v", err)
	}

	var count int64
	db.Model(&database.NewsItem{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 item in DB, got %d", count)
	}
}
