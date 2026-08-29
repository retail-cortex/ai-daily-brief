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

package agent

import (
	"os"
	"strings"
	"testing"

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/security"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test in-memory database: %v", err)
	}

	err = db.AutoMigrate(&database.Setting{}, &database.NewsItem{}, &database.ChatMessage{})
	if err != nil {
		t.Fatalf("Failed to auto-migrate database: %v", err)
	}
	return db
}

func TestGetAgentSettings_APIKeyDecryption(t *testing.T) {
	db := setupTestDB(t)

	rawKey := "AIzaSy_Secret_Test_Key_12345"
	encryptedKey, err := security.Encrypt(rawKey)
	if err != nil {
		t.Fatalf("Failed to encrypt test API key: %v", err)
	}

	// Save encrypted key in DB
	db.Save(&database.Setting{Key: "gemini_api_key", Value: encryptedKey})
	db.Save(&database.Setting{Key: "gemini_auth_mode", Value: "api_key"})
	db.Save(&database.Setting{Key: "gemini_model", Value: "gemini-2.5-pro"})

	model, authMode, apiKey, _, _ := GetAgentSettings(db)

	if authMode != "api_key" {
		t.Errorf("Expected authMode 'api_key', got '%s'", authMode)
	}
	if apiKey != rawKey {
		t.Errorf("Expected decrypted apiKey '%s', got '%s'", rawKey, apiKey)
	}
	if model != "gemini-2.5-pro" {
		t.Errorf("Expected model 'gemini-2.5-pro', got '%s'", model)
	}
}

func TestGetAgentSettings_DBOverridesConfigDefaults(t *testing.T) {
	db := setupTestDB(t)

	// Set static AppConfig defaults
	config.AppConfig = &config.Config{
		GoogleCloud: config.GoogleCloudConfig{
			ProjectID:     "config-project",
			ProjectRegion: "us-east1",
		},
		Gemini: map[string]config.GeminiAgentConfig{
			"default": {
				AuthMode: "api_key",
				APIKey:   "config-default-key",
				Model:    "gemini-1.5-flash",
				Region:   "us-east1",
			},
		},
	}

	// User configures Vertex ADC in DB
	db.Save(&database.Setting{Key: "gemini_auth_mode", Value: "vertex_adc"})
	db.Save(&database.Setting{Key: "vertex_project_id", Value: "override-project-id"})

	model, authMode, apiKey, projectID, location := GetAgentSettings(db)

	if authMode != "vertex_adc" {
		t.Errorf("Expected DB setting 'vertex_adc' to override config 'api_key', got '%s'", authMode)
	}
	if projectID != "override-project-id" {
		t.Errorf("Expected DB project 'override-project-id', got '%s'", projectID)
	}
	// Unset fields in DB should fall back to config values
	if model != "gemini-1.5-flash" {
		t.Errorf("Expected fallback model 'gemini-1.5-flash', got '%s'", model)
	}
	if location != "us-east1" {
		t.Errorf("Expected fallback location 'us-east1', got '%s'", location)
	}
	_ = apiKey
}

func TestGenerateRawContent_ValidationMessages(t *testing.T) {
	db := setupTestDB(t)
	os.Unsetenv("GEMINI_API_KEY")

	// 1. API Key mode without key should demand API Key
	db.Save(&database.Setting{Key: "gemini_auth_mode", Value: "api_key"})
	db.Save(&database.Setting{Key: "gemini_api_key", Value: ""})
	config.AppConfig = nil

	_, err := GenerateRawContent(db, "", "Hello")
	if err == nil || !strings.Contains(err.Error(), "Gemini API key is not configured") {
		t.Errorf("Expected 'Gemini API key is not configured' error, got: %v", err)
	}

	// 2. Vertex ADC mode should NOT error about API key
	db.Save(&database.Setting{Key: "gemini_auth_mode", Value: "vertex_adc"})
	db.Save(&database.Setting{Key: "vertex_project_id", Value: "my-gcp-project"})

	_, err = GenerateRawContent(db, "", "Hello")
	if err != nil && strings.Contains(err.Error(), "API key") {
		t.Errorf("Vertex ADC mode must NOT require or check API key, got error: %v", err)
	}
}
