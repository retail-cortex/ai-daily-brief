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
	"time"

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/mcp"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test in-memory database: %v", err)
	}

	if err := db.AutoMigrate(&database.Setting{}, &database.NewsItem{}, &database.ChatMessage{}, &database.Subscriber{}); err != nil {
		t.Fatalf("Failed to auto-migrate schema: %v", err)
	}

	// Seed dummy article for testing
	db.Create(&database.NewsItem{
		ID:        "art_test_1",
		Title:     "Google announces Gemini 3.7 Flash and Pro models",
		Link:      "https://blog.google/technology/ai/gemini-3-7-announcement",
		Company:   "Google",
		Category:  "Frontier Models",
		Summary:   "Next-generation multimodal reasoning and code execution capabilities.",
		PubDate:   time.Now().Format(time.RFC3339),
		CreatedAt: time.Now(),
	})

	return db
}

func setupMockMCPServer(t *testing.T) (*httptest.Server, *mcp.Handler) {
	db := setupTestDB(t)
	handler := mcp.NewHandler(db)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mcp.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := handler.HandleRequest(req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	return ts, handler
}

func TestMCPClient_ListArticles(t *testing.T) {
	ts, _ := setupMockMCPServer(t)
	defer ts.Close()

	client := NewMCPClient(ts.URL, 5)
	ctx := context.Background()

	articlesOut, err := client.ListArticles(ctx, "Frontier Models", "", "", 10)
	if err != nil {
		t.Fatalf("ListArticles failed: %v", err)
	}

	if !strings.Contains(articlesOut, "retrieved 1 articles") {
		t.Errorf("Expected mock response to contain retrieval count, got: %s", articlesOut)
	}
}

func TestMCPClient_GetSystemStatus(t *testing.T) {
	ts, _ := setupMockMCPServer(t)
	defer ts.Close()

	client := NewMCPClient(ts.URL, 5)
	ctx := context.Background()

	statusOut, err := client.GetSystemStatus(ctx)
	if err != nil {
		t.Fatalf("GetSystemStatus failed: %v", err)
	}
	if !strings.Contains(statusOut, "Control Plane:") {
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
		Gemini: map[string]config.GeminiAgentConfig{
			"default": {
				Model:    "gemini-3.7-flash",
				Region:   "global",
				AuthMode: "vertex_adc",
			},
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
	if len(res.A2UIPayload) == 0 {
		t.Error("Expected A2UI payload to be populated")
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

	// 2. Standardized Agent Card check
	req, _ = http.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /.well-known/agent-card.json, got %d", w.Code)
	}
	var card AgentCard
	if err := json.Unmarshal(w.Body.Bytes(), &card); err != nil {
		t.Fatalf("Failed to parse agent card: %v", err)
	}
	if card.Name == "" {
		t.Error("Expected agent card to specify name")
	}
	if card.Description == "" {
		t.Error("Expected agent card to specify a description")
	}
	if card.Version == "" {
		t.Error("Expected agent card to specify version")
	}
	if card.ProtocolVersion == "" {
		t.Error("Expected agent card to specify protocolVersion")
	}
	if card.URL == "" {
		t.Error("Expected agent card to specify url")
	}
	if len(card.DefaultInputModes) == 0 {
		t.Error("Expected agent card to specify defaultInputModes")
	}
	if len(card.DefaultOutputModes) == 0 {
		t.Error("Expected agent card to specify defaultOutputModes")
	}
	if len(card.Skills) == 0 {
		t.Error("Expected agent card to define skills")
	} else {
		for _, sk := range card.Skills {
			if sk.ID == "" {
				t.Errorf("Skill %s missing ID", sk.Name)
			}
			if sk.Name == "" {
				t.Errorf("Skill %s missing Name", sk.ID)
			}
			if sk.Description == "" {
				t.Errorf("Skill %s missing Description", sk.ID)
			}
			if len(sk.Tags) == 0 {
				t.Errorf("Skill %s missing Tags", sk.ID)
			}
			if len(sk.Examples) == 0 {
				t.Errorf("Skill %s missing Examples", sk.ID)
			}
		}
	}

	// 3. Invoke task via standard A2A JSON-RPC / REST
	body := strings.NewReader(`{"jsonrpc": "2.0", "id": 1, "task": "list frontier models"}`)
	req, _ = http.NewRequest(http.MethodPost, "/agent/invoke", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /agent/invoke, got %d (body: %s)", w.Code, w.Body.String())
	}

	var jsonRes struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Result  *A2AMessage `json:"result"`
	}
	if err := json.NewDecoder(w.Body).Decode(&jsonRes); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	msg := jsonRes.Result
	if msg == nil || msg.MessageID == "" || msg.ContextID == "" {
		t.Fatalf("Expected valid A2A Message result, got: %+v", jsonRes)
	}
	if msg.Role != "agent" {
		t.Errorf("Expected message role 'agent', got %s", msg.Role)
	}
	if len(msg.Parts) == 0 {
		t.Fatalf("Expected message parts to be populated")
	}

	textPart := msg.Parts[0]
	if textPart.Kind != "text" {
		t.Errorf("Expected first part to be kind 'text', got %s", textPart.Kind)
	}
	if strings.HasPrefix(textPart.Text, "```markdown") {
		t.Errorf("Expected text part NOT to be wrapped in markdown code fences, got: %s", textPart.Text)
	}

	if len(msg.Parts) > 1 {
		dataPart := msg.Parts[1]
		if dataPart.Kind != "data" {
			t.Errorf("Expected second part to be kind 'data', got %s", dataPart.Kind)
		}
		if dataPart.Data == nil {
			t.Errorf("Expected data payload to be populated")
		}
	}
}
