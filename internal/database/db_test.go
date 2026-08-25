package database

import (
	"testing"
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
