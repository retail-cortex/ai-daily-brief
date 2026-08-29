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
	"fmt"
	"log"
	"net/url"
	"os"

	"ai-daily-brief/internal/security"

	"github.com/rrmcguinness/modenv/pkg/modenv"
)

type ServerConfig struct {
	IPAddress    string `toml:"ip_address"`
	Port         string `toml:"port"`
	CronSchedule string `toml:"cron_schedule"`
}

type DatabaseConfig struct {
	Dialect    string `toml:"dialect"`
	Username   string `toml:"username"`
	Password   string `toml:"password"`
	Address    string `toml:"address"`
	Port       int    `toml:"port"`
	Database   string `toml:"database"`
	RequireSSL bool   `toml:"require_ssl"`
}

func (d *DatabaseConfig) GetDSN() string {
	if d.Dialect == "sqlite" {
		if d.Address == ":memory:" {
			return ":memory:"
		}
		if d.Database != "" {
			return d.Database
		}
		return "data/ai_daily_brief.db"
	}
	if d.Address == "" {
		return ""
	}
	sslMode := "disable"
	if d.RequireSSL {
		sslMode = "require"
	}
	port := d.Port
	if port <= 0 {
		port = 5432
	}
	user := d.Username
	if user == "" {
		user = "postgres"
	}
	dbName := d.Database
	if dbName == "" {
		dbName = "ai_daily_brief"
	}
	escapedPass := url.QueryEscape(d.Password)
	if escapedPass != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", user, escapedPass, d.Address, port, dbName, sslMode)
	}
	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", user, d.Address, port, dbName, sslMode)
}

type GoogleCloudConfig struct {
	ProjectID     string `toml:"project_id"`
	ProjectRegion string `toml:"project_region"`
}

type GeminiAgentConfig struct {
	Model            string   `toml:"model"`
	Region           string   `toml:"region"`
	AuthMode         string   `toml:"auth_mode"`
	APIKey           string   `toml:"api_key"`
	Instructions     string   `toml:"instructions"`
	Temperature      *float32 `toml:"temperature"`
	TopP             *float32 `toml:"top_p"`
	TopK             *int     `toml:"top_k"`
	GroundWithGoogle bool     `toml:"ground_with_google"`
}

type MCPServerConfig struct {
	URL            string `toml:"url"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

type Config struct {
	Server       ServerConfig                 `toml:"server"`
	Database     DatabaseConfig               `toml:"database"`
	GoogleCloud  GoogleCloudConfig            `toml:"google_cloud"`
	Gemini       map[string]GeminiAgentConfig `toml:"gemini"`
	Port         string                       `toml:"port"`
	CronSchedule string                       `toml:"cron_schedule"`
	DatabasePath string                       `toml:"database_path"`
	DatabaseURL  string                       `toml:"database_url"`
	AlloyDBURL   string                       `toml:"alloydb_url"`
}

func (c *Config) GetPort() string {
	if c.Port != "" {
		return c.Port
	}
	if c.Server.Port != "" {
		return c.Server.Port
	}
	return "8080"
}

func (c *Config) GetIPAddress() string {
	if c.Server.IPAddress != "" {
		return c.Server.IPAddress
	}
	return "0.0.0.0"
}

func (c *Config) GetCronSchedule() string {
	if c.CronSchedule != "" {
		return c.CronSchedule
	}
	if c.Server.CronSchedule != "" {
		return c.Server.CronSchedule
	}
	return "0 8 * * *"
}

func (c *Config) GetDatabaseDSN() string {
	if env := os.Getenv("ALLOYDB_DATABASE_URL"); env != "" {
		return env
	}
	if env := os.Getenv("DATABASE_URL"); env != "" {
		return env
	}
	if dsn := c.Database.GetDSN(); dsn != "" {
		return dsn
	}
	if c.AlloyDBURL != "" {
		return c.AlloyDBURL
	}
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	if c.DatabasePath != "" {
		return c.DatabasePath
	}
	return "data/ai_daily_brief.db"
}

func (c *Config) GetGeminiConfig(agentName string) GeminiAgentConfig {
	if c.Gemini != nil {
		if agentName != "" {
			if cfg, ok := c.Gemini[agentName]; ok {
				return cfg
			}
		}
		if def, ok := c.Gemini["default"]; ok {
			return def
		}
		for _, cfg := range c.Gemini {
			return cfg
		}
	}
	return GeminiAgentConfig{
		Model:    "gemini-3.7-flash",
		Region:   "global",
		AuthMode: "vertex_adc",
	}
}

type AgentConfig struct {
	Server         ServerConfig                 `toml:"server"`
	GoogleCloud    GoogleCloudConfig            `toml:"google_cloud"`
	Gemini         map[string]GeminiAgentConfig `toml:"gemini"`
	MCPServers     map[string]MCPServerConfig   `toml:"mcp_servers"`
	Port           string                       `toml:"port"`
	AgentName      string                       `toml:"agent_name"`
	MCPServerURL   string                       `toml:"mcp_server_url"`
	TimeoutSeconds int                          `toml:"timeout_seconds"`
}

func (a *AgentConfig) GetPort() string {
	if a.Port != "" {
		return a.Port
	}
	if a.Server.Port != "" {
		return a.Server.Port
	}
	return "8081"
}

func (a *AgentConfig) GetMCPServer(name string) MCPServerConfig {
	if a.MCPServers != nil {
		if name != "" {
			if srv, ok := a.MCPServers[name]; ok && srv.URL != "" {
				return srv
			}
		}
		if srv, ok := a.MCPServers["daily_brief"]; ok && srv.URL != "" {
			return srv
		}
		if srv, ok := a.MCPServers["default"]; ok && srv.URL != "" {
			return srv
		}
		for _, srv := range a.MCPServers {
			if srv.URL != "" {
				return srv
			}
		}
	}
	url := a.MCPServerURL
	if url == "" {
		url = "http://localhost:8080"
	}
	timeout := a.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	return MCPServerConfig{
		URL:            url,
		TimeoutSeconds: timeout,
	}
}

func (a *AgentConfig) GetGeminiConfig(name string) GeminiAgentConfig {
	if a.Gemini != nil {
		if name != "" {
			if cfg, ok := a.Gemini[name]; ok {
				return cfg
			}
		}
		if def, ok := a.Gemini["default"]; ok {
			return def
		}
		for _, cfg := range a.Gemini {
			return cfg
		}
	}
	return GeminiAgentConfig{
		Model:    "gemini-3.7-flash",
		Region:   "global",
		AuthMode: "vertex_adc",
	}
}

var AppConfig *Config

func LoadConfig() *Config {
	if os.Getenv("MODENV_PREFIX") == "" {
		_ = os.Setenv("MODENV_PREFIX", "configs/mcp")
	}
	var cfg Config
	clone, err := modenv.Load(&cfg)
	if err != nil {
		log.Printf("[Config] Error loading modenv configuration: %v. Using defaults.", err)
		defaultTemp := float32(0.3)
		defaultTopP := float32(0.95)
		defaultTopK := 40
		AppConfig = &Config{
			Server: ServerConfig{
				IPAddress:    "0.0.0.0",
				Port:         "8080",
				CronSchedule: "0 8 * * *",
			},
			Database: DatabaseConfig{
				Dialect:  "sqlite",
				Database: "data/ai_daily_brief.db",
			},
			GoogleCloud: GoogleCloudConfig{
				ProjectID:     security.GetGCPProjectID(),
				ProjectRegion: "us-central1",
			},
			Gemini: map[string]GeminiAgentConfig{
				"default": {
					Model:            "gemini-3.7-flash",
					Region:           "global",
					AuthMode:         "vertex_adc",
					Instructions:     "You are an AI research intelligence assistant.",
					Temperature:      &defaultTemp,
					TopP:             &defaultTopP,
					TopK:             &defaultTopK,
					GroundWithGoogle: false,
				},
			},
		}
		return AppConfig
	}
	AppConfig = clone.(*Config)
	if AppConfig.GoogleCloud.ProjectID == "" {
		AppConfig.GoogleCloud.ProjectID = security.GetGCPProjectID()
	}
	if AppConfig.Port == "" && AppConfig.Server.Port != "" {
		AppConfig.Port = AppConfig.Server.Port
	}
	if AppConfig.Server.Port == "" && AppConfig.Port != "" {
		AppConfig.Server.Port = AppConfig.Port
	}
	return AppConfig
}

func LoadAgentConfig() *AgentConfig {
	if os.Getenv("MODENV_PREFIX") == "" {
		_ = os.Setenv("MODENV_PREFIX", "configs/agent")
	}
	var cfg AgentConfig
	clone, err := modenv.Load(&cfg)
	var agentCfg *AgentConfig
	if err != nil {
		log.Printf("[Config] Error loading agent modenv configuration: %v. Using defaults.", err)
		agentTemp := float32(0.3)
		agentTopP := float32(0.95)
		agentTopK := 40
		agentCfg = &AgentConfig{
			Server: ServerConfig{
				IPAddress: "0.0.0.0",
				Port:      "8081",
			},
			Port:      "8081",
			AgentName: "ai-daily-brief-a2a-agent",
			MCPServers: map[string]MCPServerConfig{
				"daily_brief": {
					URL:            "http://localhost:8080",
					TimeoutSeconds: 60,
				},
			},
			MCPServerURL:   "http://localhost:8080",
			TimeoutSeconds: 60,
			GoogleCloud: GoogleCloudConfig{
				ProjectID:     security.GetGCPProjectID(),
				ProjectRegion: "us-central1",
			},
			Gemini: map[string]GeminiAgentConfig{
				"default": {
					Model:            "gemini-3.7-flash",
					Region:           "global",
					AuthMode:         "vertex_adc",
					Instructions:     "You are an autonomous AI & Cloud Intelligence Research Agent.",
					Temperature:      &agentTemp,
					TopP:             &agentTopP,
					TopK:             &agentTopK,
					GroundWithGoogle: false,
				},
			},
		}
	} else {
		agentCfg = clone.(*AgentConfig)
	}

	// Environment variable overrides
	if envMCP := os.Getenv("MCP_SERVER_URL"); envMCP != "" {
		agentCfg.MCPServerURL = envMCP
		if agentCfg.MCPServers == nil {
			agentCfg.MCPServers = make(map[string]MCPServerConfig)
		}
		srv := agentCfg.GetMCPServer("daily_brief")
		srv.URL = envMCP
		agentCfg.MCPServers["daily_brief"] = srv
	} else if envMCPBrief := os.Getenv("MCP_DAILY_BRIEF_URL"); envMCPBrief != "" {
		agentCfg.MCPServerURL = envMCPBrief
		if agentCfg.MCPServers == nil {
			agentCfg.MCPServers = make(map[string]MCPServerConfig)
		}
		srv := agentCfg.GetMCPServer("daily_brief")
		srv.URL = envMCPBrief
		agentCfg.MCPServers["daily_brief"] = srv
	}

	if agentCfg.GoogleCloud.ProjectID == "" {
		agentCfg.GoogleCloud.ProjectID = security.GetGCPProjectID()
	}
	if agentCfg.Port == "" && agentCfg.Server.Port != "" {
		agentCfg.Port = agentCfg.Server.Port
	}
	if agentCfg.Server.Port == "" && agentCfg.Port != "" {
		agentCfg.Server.Port = agentCfg.Port
	}
	mcpServer := agentCfg.GetMCPServer("daily_brief")
	if agentCfg.MCPServerURL == "" {
		agentCfg.MCPServerURL = mcpServer.URL
	}
	if agentCfg.TimeoutSeconds <= 0 {
		agentCfg.TimeoutSeconds = mcpServer.TimeoutSeconds
	}
	return agentCfg
}
