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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-daily-brief/internal/database"

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

	// Seed test data
	db.Create(&database.NewsItem{
		ID:        "item_gemini_37",
		Title:     "Google Unveils Gemini 3.7 Flash Hybrid Reasoning Model",
		Link:      "https://blog.google/technology/ai/gemini-3-7-flash",
		Summary:   "Google announces Gemini 3.7 Flash with dynamic reasoning depth and hybrid speed capabilities.",
		Category:  "Frontier Models",
		Company:   "Google",
		PubDate:   time.Now().Format("2006-01-02 15:04:05"),
		CreatedAt: time.Now(),
	})

	db.Create(&database.NewsItem{
		ID:        "item_gcp_a3",
		Title:     "Google Cloud Launches A3 Edge TPU Infrastructure",
		Link:      "https://cloud.google.com/blog/products/compute/a3-edge",
		Summary:   "Next-generation GPU and TPU clusters with 3.2 Tbps networking for hyperscale training.",
		Category:  "Google Cloud",
		Company:   "Google Cloud",
		PubDate:   time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05"),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	})

	return db
}

func TestMCP_InitializeHandshake(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHandler(db)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := handler.HandleRequest(req)
	if resp.Error != nil {
		t.Fatalf("Unexpected RPC error: %v", resp.Error)
	}

	resMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", resp.Result)
	}

	if resMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion '2024-11-05', got '%v'", resMap["protocolVersion"])
	}
}

func TestMCP_ToolsList(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHandler(db)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	resp := handler.HandleRequest(req)
	if resp.Error != nil {
		t.Fatalf("Unexpected RPC error: %v", resp.Error)
	}

	resMap := resp.Result.(map[string]interface{})
	tools := resMap["tools"].([]MCPTool)

	expectedTools := map[string]bool{
		"list_articles":       false,
		"get_article_context": false,
		"generate_tldr":       false,
		"trigger_crawl":       false,
		"get_newsletter":      false,
		"agent_chat":          false,
		"get_system_status":   false,
	}

	for _, tool := range tools {
		if _, exists := expectedTools[tool.Name]; exists {
			expectedTools[tool.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool '%s' was not registered in tools/list", name)
		}
	}
}

func TestMCP_ListArticles_A2UI(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHandler(db)

	params, _ := json.Marshal(map[string]interface{}{
		"name": "list_articles",
		"arguments": map[string]interface{}{
			"limit": 5,
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  params,
	}

	resp := handler.HandleRequest(req)
	if resp.Error != nil {
		t.Fatalf("Unexpected RPC error: %v", resp.Error)
	}

	resMap := resp.Result.(map[string]interface{})
	contentBlocks := resMap["content"].([]MCPContentBlock)
	if len(contentBlocks) < 2 {
		t.Fatalf("Expected at least 2 content blocks (text + a2ui resource)")
	}

	text := contentBlocks[0].Text
	if !strings.Contains(text, "retrieved 2 articles") {
		t.Errorf("Expected retrieval text, got:\n%s", text)
	}

	resBlock := contentBlocks[1]
	if resBlock.Type != "resource" || resBlock.Resource == nil {
		t.Fatalf("Expected second block to be resource block")
	}
	if resBlock.Resource.MIMEType != MIMETypeA2UIJSON {
		t.Errorf("Expected MIMEType %s, got %s", MIMETypeA2UIJSON, resBlock.Resource.MIMEType)
	}
	if !strings.Contains(resBlock.Resource.Text, "createSurface") || !strings.Contains(resBlock.Resource.Text, "updateComponents") {
		t.Errorf("Expected A2UI v1.0 instructions in resource text, got:\n%s", resBlock.Resource.Text)
	}
	if !strings.Contains(resBlock.Resource.Text, "Gemini 3.7") {
		t.Errorf("Expected Gemini 3.7 in A2UI resource text, got:\n%s", resBlock.Resource.Text)
	}
}

func TestMCP_SystemStatus(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHandler(db)

	params, _ := json.Marshal(map[string]interface{}{
		"name":      "get_system_status",
		"arguments": map[string]interface{}{},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  params,
	}

	resp := handler.HandleRequest(req)
	if resp.Error != nil {
		t.Fatalf("Unexpected RPC error: %v", resp.Error)
	}

	resMap := resp.Result.(map[string]interface{})
	contentBlocks := resMap["content"].([]MCPContentBlock)
	if len(contentBlocks) < 2 {
		t.Fatalf("Expected at least 2 content blocks")
	}

	text := contentBlocks[0].Text
	if !strings.Contains(text, "Control Plane: 2 articles indexed") {
		t.Errorf("Expected telemetry count, got:\n%s", text)
	}

	resBlock := contentBlocks[1]
	if resBlock.Resource == nil || resBlock.Resource.MIMEType != MIMETypeA2UIJSON {
		t.Errorf("Expected A2UI resource for telemetry")
	}
}

func TestMCP_HTTPServerEndpoints(t *testing.T) {
	db := setupTestDB(t)
	srv := NewServer(db)

	// 1. Health check
	reqHealth, _ := http.NewRequest("GET", "/healthz", nil)
	wHealth := httptest.NewRecorder()
	srv.Router.ServeHTTP(wHealth, reqHealth)

	if wHealth.Code != http.StatusOK {
		t.Errorf("GET /healthz returned status %d", wHealth.Code)
	}

	// 2. Direct JSON-RPC POST /mcp
	reqRPC, _ := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/list",
	})
	reqPost, _ := http.NewRequest("POST", "/mcp", bytes.NewReader(reqRPC))
	reqPost.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	srv.Router.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Errorf("POST /mcp returned status %d: %s", wPost.Code, wPost.Body.String())
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(wPost.Body.Bytes(), &rpcResp); err != nil {
		t.Fatalf("Failed to unmarshal JSON-RPC response: %v", err)
	}
	if rpcResp.Error != nil {
		t.Errorf("Expected success, got error: %v", rpcResp.Error)
	}
}
