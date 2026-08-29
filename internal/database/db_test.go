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

package database

import (
	"os"
	"testing"
	"time"
)

func TestInitDBInMemory(t *testing.T) {
	// Initialize in-memory SQLite database
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize in-memory database: %v", err)
	}
	if db == nil {
		t.Fatal("InitDB returned nil db")
	}

	// Verify tables are created by inserting a test subscriber
	sub := Subscriber{
		ID:    "sub-1",
		Email: "test@example.com",
		Name:  "Test User",
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Errorf("Failed to create record in Subscriber table: %v", err)
	}

	// Verify default settings are seeded
	var setting Setting
	if err := db.First(&setting, "key = ?", "gemini_model").Error; err != nil {
		t.Errorf("Failed to find seeded gemini_model setting: %v", err)
	}
	if setting.Value != "gemini-3.7-flash" {
		t.Errorf("Expected default gemini_model to be gemini-3.7-flash, got %s", setting.Value)
	}
}

// TestInitDBAlloyDB runs an integration test against a live AlloyDB or PostgreSQL instance if configured
func TestInitDBAlloyDB(t *testing.T) {
	dsn := os.Getenv("ALLOYDB_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("Skipping live AlloyDB test: ALLOYDB_DATABASE_URL or DATABASE_URL not set")
	}

	db, err := InitDB(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to AlloyDB instance (%s): %v", dsn, err)
	}
	if db == nil {
		t.Fatal("InitDB returned nil db for AlloyDB")
	}

	// Validate DB ping & connection pool
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to obtain generic database object: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("AlloyDB ping failed: %v", err)
	}

	// Test writing & querying a test run log record
	testRun := RunLog{
		ID:         "test-alloydb-run-" + time.Now().Format("20060102150405"),
		RunDate:    time.Now().Format("2006-01-02"),
		ItemsCount: 1,
		Status:     "success",
		Log:        "AlloyDB connectivity verification",
		CreatedAt:  time.Now(),
	}
	if err := db.Create(&testRun).Error; err != nil {
		t.Fatalf("Failed to insert test record into AlloyDB: %v", err)
	}

	var fetched RunLog
	if err := db.First(&fetched, "id = ?", testRun.ID).Error; err != nil {
		t.Fatalf("Failed to query inserted test record from AlloyDB: %v", err)
	}

	// Clean up test record
	_ = db.Delete(&testRun)
}
