package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg := LoadConfig()
	if cfg == nil {
		t.Fatal("LoadConfig returned nil")
	}

	// Verify defaults are populated
	if cfg.Port == "" {
		t.Error("Expected default Port to be populated, got empty string")
	}
	if cfg.CronSchedule == "" {
		t.Error("Expected default CronSchedule to be populated, got empty string")
	}
	if cfg.DatabasePath == "" {
		t.Error("Expected default DatabasePath to be populated, got empty string")
	}

	// Verify nested structures
	if cfg.Gemini.Model == "" {
		t.Error("Expected default Gemini.Model to be populated, got empty string")
	}
}
