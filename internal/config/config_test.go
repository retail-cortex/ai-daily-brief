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
