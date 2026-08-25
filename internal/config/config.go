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
	"log"

	"github.com/rrmcguinness/modenv/pkg/modenv"
)

type Config struct {
	Port         string       `toml:"port"`
	CronSchedule string       `toml:"cron_schedule"`
	DatabasePath string       `toml:"database_path"`
	Gemini       GeminiConfig `toml:"gemini"`
}

type GeminiConfig struct {
	Model           string `toml:"model"`
	AuthMode        string `toml:"auth_mode"`
	APIKey          string `toml:"api_key"`
	VertexProjectID string `toml:"vertex_project_id"`
	VertexLocation  string `toml:"vertex_location"`
}

var AppConfig *Config

func LoadConfig() *Config {
	var cfg Config
	clone, err := modenv.Load(&cfg)
	if err != nil {
		log.Printf("[Config] Error loading modenv configuration: %v. Using defaults.", err)
		// Fallback defaults
		AppConfig = &Config{
			Port:         "3001",
			CronSchedule: "0 8 * * *",
			DatabasePath: "data/ai_daily_brief.db",
			Gemini: GeminiConfig{
				Model:          "gemini-3.7-flash",
				AuthMode:       "api_key",
				VertexLocation: "us-central1",
			},
		}
		return AppConfig
	}
	AppConfig = clone.(*Config)
	return AppConfig
}
