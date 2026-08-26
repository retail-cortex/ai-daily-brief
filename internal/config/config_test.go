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

func TestGetDatabaseDSN_Resolution(t *testing.T) {
	// 1. AlloyDB URL takes highest priority
	cfgAlloy := &Config{
		AlloyDBURL:   "postgres://user:pass@alloydb-cluster:5432/db",
		DatabaseURL:  "postgres://user:pass@other-db:5432/db",
		DatabasePath: ":memory:",
	}
	if dsn := cfgAlloy.GetDatabaseDSN(); dsn != "postgres://user:pass@alloydb-cluster:5432/db" {
		t.Errorf("Expected AlloyDBURL, got '%s'", dsn)
	}

	// 2. DatabaseURL takes second priority
	cfgPostgres := &Config{
		DatabaseURL:  "postgres://user:pass@other-db:5432/db",
		DatabasePath: ":memory:",
	}
	if dsn := cfgPostgres.GetDatabaseDSN(); dsn != "postgres://user:pass@other-db:5432/db" {
		t.Errorf("Expected DatabaseURL, got '%s'", dsn)
	}

	// 3. DatabasePath SQLite fallback
	cfgSQLite := &Config{
		DatabasePath: ":memory:",
	}
	if dsn := cfgSQLite.GetDatabaseDSN(); dsn != ":memory:" {
		t.Errorf("Expected DatabasePath, got '%s'", dsn)
	}
}
