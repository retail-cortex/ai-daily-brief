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

package config

import (
	"testing"
)

func TestServerConfig_Getters(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			IPAddress:    "127.0.0.1",
			Port:         "9090",
			CronSchedule: "0 12 * * *",
		},
	}
	if got := cfg.GetIPAddress(); got != "127.0.0.1" {
		t.Errorf("GetIPAddress() = %s, expected 127.0.0.1", got)
	}
	if got := cfg.GetPort(); got != "9090" {
		t.Errorf("GetPort() = %s, expected 9090", got)
	}
	if got := cfg.GetCronSchedule(); got != "0 12 * * *" {
		t.Errorf("GetCronSchedule() = %s, expected 0 12 * * *", got)
	}

	// Fallback behavior
	emptyCfg := &Config{}
	if got := emptyCfg.GetIPAddress(); got != "0.0.0.0" {
		t.Errorf("GetIPAddress() default = %s, expected 0.0.0.0", got)
	}
	if got := emptyCfg.GetPort(); got != "8080" {
		t.Errorf("GetPort() default = %s, expected 8080", got)
	}
	if got := emptyCfg.GetCronSchedule(); got != "0 8 * * *" {
		t.Errorf("GetCronSchedule() default = %s, expected 0 8 * * *", got)
	}
}

func TestDatabaseConfig_GetDSN_Postgres(t *testing.T) {
	// 1. Postgres with SSL, custom port, credentials
	dbCfg := DatabaseConfig{
		Dialect:    "postgres",
		Username:   "postgres",
		Password:   "secret_pass@123",
		Address:    "10.0.0.1",
		Port:       5432,
		Database:   "ai_daily_brief",
		RequireSSL: true,
	}

	expected := "postgres://postgres:secret_pass%40123@10.0.0.1:5432/ai_daily_brief?sslmode=require"
	if got := dbCfg.GetDSN(); got != expected {
		t.Errorf("GetDSN() = %s, expected %s", got, expected)
	}

	// 2. AlloyDB dialect with require_ssl = false
	alloyCfg := DatabaseConfig{
		Dialect:    "alloydb",
		Username:   "app_user",
		Password:   "simplepass",
		Address:    "localhost",
		Port:       5433,
		Database:   "brief_db",
		RequireSSL: false,
	}

	expectedAlloy := "postgres://app_user:simplepass@localhost:5433/brief_db?sslmode=disable"
	if got := alloyCfg.GetDSN(); got != expectedAlloy {
		t.Errorf("GetDSN() = %s, expected %s", got, expectedAlloy)
	}
}

func TestDatabaseConfig_GetDSN_SQLite(t *testing.T) {
	// 1. SQLite in-memory
	dbCfg := DatabaseConfig{
		Dialect: "sqlite",
		Address: ":memory:",
	}
	if got := dbCfg.GetDSN(); got != ":memory:" {
		t.Errorf("GetDSN() = %s, expected :memory:", got)
	}

	// 2. SQLite local file
	fileCfg := DatabaseConfig{
		Dialect:  "sqlite",
		Database: "data/ai_daily_brief.db",
	}
	if got := fileCfg.GetDSN(); got != "data/ai_daily_brief.db" {
		t.Errorf("GetDSN() = %s, expected data/ai_daily_brief.db", got)
	}
}

func TestGetDatabaseDSN_Resolution(t *testing.T) {
	// 1. [database] struct takes highest priority
	cfgStruct := &Config{
		Database: DatabaseConfig{
			Dialect:    "postgres",
			Username:   "admin",
			Password:   "pass",
			Address:    "db.host",
			Port:       5432,
			Database:   "test_db",
			RequireSSL: true,
		},
		AlloyDBURL:   "postgres://user:pass@alloydb-cluster:5432/db",
		DatabaseURL:  "postgres://user:pass@other-db:5432/db",
		DatabasePath: ":memory:",
	}
	if dsn := cfgStruct.GetDatabaseDSN(); dsn != "postgres://admin:pass@db.host:5432/test_db?sslmode=require" {
		t.Errorf("Expected DSN from DatabaseConfig struct, got '%s'", dsn)
	}

	// 2. Legacy AlloyDB URL fallback
	cfgAlloy := &Config{
		AlloyDBURL:   "postgres://user:pass@alloydb-cluster:5432/db",
		DatabaseURL:  "postgres://user:pass@other-db:5432/db",
		DatabasePath: ":memory:",
	}
	if dsn := cfgAlloy.GetDatabaseDSN(); dsn != "postgres://user:pass@alloydb-cluster:5432/db" {
		t.Errorf("Expected AlloyDBURL, got '%s'", dsn)
	}

	// 3. Legacy DatabaseURL fallback
	cfgPostgres := &Config{
		DatabaseURL:  "postgres://user:pass@other-db:5432/db",
		DatabasePath: ":memory:",
	}
	if dsn := cfgPostgres.GetDatabaseDSN(); dsn != "postgres://user:pass@other-db:5432/db" {
		t.Errorf("Expected DatabaseURL, got '%s'", dsn)
	}

	// 4. Legacy DatabasePath SQLite fallback
	cfgSQLite := &Config{
		DatabasePath: ":memory:",
	}
	if dsn := cfgSQLite.GetDatabaseDSN(); dsn != ":memory:" {
		t.Errorf("Expected DatabasePath, got '%s'", dsn)
	}
}

func TestGeminiConfigMap_And_GoogleCloud(t *testing.T) {
	temp := float32(0.3)
	topP := float32(0.95)
	topK := 40

	cfg := &Config{
		GoogleCloud: GoogleCloudConfig{
			ProjectID:     "retail-cortex-prod",
			ProjectRegion: "us-central1",
		},
		Gemini: map[string]GeminiAgentConfig{
			"default": {
				Model:            "gemini-3.7-flash",
				Region:           "global",
				AuthMode:         "vertex_adc",
				Instructions:     "Default instruction",
				Temperature:      &temp,
				TopP:             &topP,
				TopK:             &topK,
				GroundWithGoogle: false,
			},
			"tldr": {
				Model:        "gemini-3.7-flash",
				Region:       "us-central1",
				AuthMode:     "vertex_adc",
				Instructions: "TLDR instruction",
			},
		},
	}

	// 1. Google Cloud project vs Gemini model region separation
	if cfg.GoogleCloud.ProjectRegion != "us-central1" {
		t.Errorf("Expected project_region us-central1, got %s", cfg.GoogleCloud.ProjectRegion)
	}

	defGemini := cfg.GetGeminiConfig("default")
	if defGemini.Region != "global" {
		t.Errorf("Expected default gemini model region global, got %s", defGemini.Region)
	}

	// 2. Named agent retrieval
	tldrGemini := cfg.GetGeminiConfig("tldr")
	if tldrGemini.Region != "us-central1" || tldrGemini.Instructions != "TLDR instruction" {
		t.Errorf("Expected tldr agent config, got region %s", tldrGemini.Region)
	}

	// 3. Fallback for non-existent agent name
	unknownGemini := cfg.GetGeminiConfig("nonexistent")
	if unknownGemini.Model != "gemini-3.7-flash" {
		t.Errorf("Expected fallback to default model, got %s", unknownGemini.Model)
	}
}

func TestMCPServersMap_And_AgentConfig(t *testing.T) {
	agentCfg := &AgentConfig{
		MCPServers: map[string]MCPServerConfig{
			"daily_brief": {
				URL:            "https://mcp.internal",
				TimeoutSeconds: 45,
			},
			"external_tools": {
				URL:            "https://tools.internal",
				TimeoutSeconds: 15,
			},
		},
	}

	// 1. Retrieve specific named MCP server
	dbServer := agentCfg.GetMCPServer("daily_brief")
	if dbServer.URL != "https://mcp.internal" || dbServer.TimeoutSeconds != 45 {
		t.Errorf("Expected daily_brief MCP server config, got %+v", dbServer)
	}

	extServer := agentCfg.GetMCPServer("external_tools")
	if extServer.URL != "https://tools.internal" || extServer.TimeoutSeconds != 15 {
		t.Errorf("Expected external_tools MCP server config, got %+v", extServer)
	}

	// 2. Fallback to default/first
	fallbackServer := agentCfg.GetMCPServer("unknown")
	if fallbackServer.URL != "https://mcp.internal" {
		t.Errorf("Expected fallback to daily_brief MCP server, got %s", fallbackServer.URL)
	}
}

func TestLoadConfig_MCP(t *testing.T) {
	cfg := LoadConfig()
	if cfg == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if cfg.GetPort() == "" {
		t.Error("Expected Port to be populated")
	}
	if cfg.Server.Port == "" {
		t.Error("Expected Server.Port to be populated")
	}
	if cfg.Server.CronSchedule == "" {
		t.Error("Expected Server.CronSchedule to be populated")
	}
	defaultGemini := cfg.GetGeminiConfig("default")
	if defaultGemini.Model == "" {
		t.Error("Expected default Gemini Model to be populated")
	}
}

func TestLoadAgentConfig_A2A(t *testing.T) {
	agentCfg := LoadAgentConfig()
	if agentCfg == nil {
		t.Fatal("LoadAgentConfig returned nil")
	}
	if agentCfg.AgentName == "" {
		t.Error("Expected AgentName to be populated")
	}
	mcpServer := agentCfg.GetMCPServer("daily_brief")
	if mcpServer.URL == "" {
		t.Error("Expected MCPServer URL to be populated")
	}
	if agentCfg.MCPServerURL == "" {
		t.Error("Expected legacy MCPServerURL to be populated")
	}
	geminiCfg := agentCfg.GetGeminiConfig("")
	if geminiCfg.Model == "" {
		t.Error("Expected Gemini Model to be populated")
	}
}
