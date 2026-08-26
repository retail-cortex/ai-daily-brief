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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/mcp"
)

func setupMockMCPServer(t *testing.T) (*httptest.Server, *mcp.Server) {
	t.Helper()
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory db: %v", err)
	}

	// Seed test item
	db.Create(&database.NewsItem{
		Title:    "Gemini 3.7 Released",
		Company:  "Google",
		Category: "Frontier Models",
		Summary:  "Google announced Gemini 3.7 Flash reasoning model.",
		Link:     "https://blog.google/gemini-37",
	})

	mcpSrv := mcp.NewServer(db)
	ts := httptest.NewServer(mcpSrv.Router)
	return ts, mcpSrv
}

func TestMCPClient_ToolInvocation(t *testing.T) {
	ts, _ := setupMockMCPServer(t)
	defer ts.Close()

	client := NewMCPClient(ts.URL, 5)
	ctx := context.Background()

	// 1. Test ListArticles
	out, err := client.ListArticles(ctx, "Frontier Models", "Google", "", 5)
	if err != nil {
		t.Fatalf("ListArticles failed: %v", err)
	}
	if !strings.Contains(out, "Gemini 3.7") {
		t.Errorf("Expected output to contain 'Gemini 3.7', got: %s", out)
	}

	// 2. Test GetSystemStatus
	statusOut, err := client.GetSystemStatus(ctx)
	if err != nil {
		t.Fatalf("GetSystemStatus failed: %v", err)
	}
	if !strings.Contains(statusOut, "TELEMETRY") {
		t.Errorf("Expected status output, got: %s", statusOut)
	}
}

func TestAgent_ExecuteTask(t *testing.T) {
	ts, _ := setupMockMCPServer(t)
	defer ts.Close()

	cfg := &config.AgentConfig{
		AgentName:      "test-agent",
		MCPServerURL:   ts.URL,
		TimeoutSeconds: 5,
		Gemini: config.GeminiConfig{
			Model:    "gemini-3.7-flash",
			AuthMode: "vertex_adc",
		},
	}

	agent := NewAgent(cfg)
	ctx := context.Background()

	// Execute search task
	res, err := agent.ExecuteTask(ctx, "search frontier models from Google")
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %s", res.Status)
	}
	if len(res.ToolCalls) == 0 {
		t.Error("Expected tool calls to be recorded")
	}
	if !strings.Contains(res.Output, "Gemini 3.7") {
		t.Errorf("Expected output to contain article details, got: %s", res.Output)
	}
}

func TestServer_HTTPRoutes(t *testing.T) {
	ts, _ := setupMockMCPServer(t)
	defer ts.Close()

	cfg := &config.AgentConfig{
		AgentName:      "test-a2a-server",
		MCPServerURL:   ts.URL,
		TimeoutSeconds: 5,
	}

	srv := NewServer(cfg)

	// 1. Health check
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /healthz, got %d", w.Code)
	}

	// 2. Invoke task
	body := strings.NewReader(`{"task": "list frontier models"}`)
	req, _ = http.NewRequest(http.MethodPost, "/agent/invoke", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /agent/invoke, got %d (body: %s)", w.Code, w.Body.String())
	}

	var jsonRes struct {
		Success bool        `json:"success"`
		Result  *TaskResult `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonRes); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if !jsonRes.Success || jsonRes.Result == nil {
		t.Error("Expected successful result payload")
	}
}
