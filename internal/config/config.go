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
			DatabasePath: "data/news_agent.db",
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
