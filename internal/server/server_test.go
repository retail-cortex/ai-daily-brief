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

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"ai-daily-brief/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestServer(t *testing.T) (*Server, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open sqlite memory db: %v", err)
	}

	if err := db.AutoMigrate(&database.Setting{}, &database.NewsItem{}, &database.ChatMessage{}, &database.Subscriber{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html><body>AI Daily Brief Test</body></html>")},
	}

	srv := NewServer(db, mockFS)
	return srv, db
}

func TestSettingsEndpoints_VertexADCMode(t *testing.T) {
	srv, db := setupTestServer(t)

	// 1. POST /api/settings to switch to Vertex ADC
	body, _ := json.Marshal(map[string]string{
		"gemini_auth_mode":  "vertex_adc",
		"vertex_project_id": "test-vertex-project",
		"vertex_location":   "us-west1",
		"gemini_model":      "gemini-3.7-flash",
	})

	req, _ := http.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/settings returned status %d: %s", w.Code, w.Body.String())
	}

	// 2. GET /api/settings to verify persisted values
	reqGet, _ := http.NewRequest("GET", "/api/settings", nil)
	wGet := httptest.NewRecorder()
	srv.Router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("GET /api/settings returned status %d", wGet.Code)
	}

	var resp struct {
		Success  bool              `json:"success"`
		Settings map[string]string `json:"settings"`
	}
	if err := json.Unmarshal(wGet.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal settings response: %v", err)
	}

	if resp.Settings["gemini_auth_mode"] != "vertex_adc" {
		t.Errorf("Expected gemini_auth_mode 'vertex_adc', got '%s'", resp.Settings["gemini_auth_mode"])
	}
	if resp.Settings["vertex_project_id"] != "test-vertex-project" {
		t.Errorf("Expected vertex_project_id 'test-vertex-project', got '%s'", resp.Settings["vertex_project_id"])
	}

	// 3. Verify in DB directly
	var s database.Setting
	if err := db.First(&s, "key = ?", "gemini_auth_mode").Error; err != nil || s.Value != "vertex_adc" {
		t.Errorf("DB Setting key 'gemini_auth_mode' not saved properly: %v, val: %s", err, s.Value)
	}
}

func TestTestConnectionEndpoint(t *testing.T) {
	srv, _ := setupTestServer(t)

	req, _ := http.NewRequest("POST", "/api/agent/test-connection?model=gemini-2.5-flash", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/agent/test-connection returned status %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal test connection response: %v", err)
	}

	if resp["model"] != "gemini-2.5-flash" {
		t.Errorf("Expected model 'gemini-2.5-flash' in response, got %v", resp["model"])
	}
}
