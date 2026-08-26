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
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type NewsCategory string

const (
	CategoryFrontierModels NewsCategory = "Frontier Models"
	CategoryResearchPapers NewsCategory = "AI Research Papers"
	CategoryBusinessInfra  NewsCategory = "AI Business & Infra"
	CategoryOSSTooling     NewsCategory = "OSS & Tooling"
	CategoryGoogleCloud    NewsCategory = "Google Cloud"
)

type NewsItem struct {
	ID        string       `gorm:"primaryKey" json:"id"`
	RunDate   string       `gorm:"index" json:"run_date"`
	PubDate   string       `gorm:"index" json:"pub_date"`
	Company   string       `gorm:"index" json:"company"`
	Category  NewsCategory `gorm:"index" json:"category"`
	Title     string       `json:"title"`
	Summary   string       `gorm:"type:text" json:"summary"`
	Link      string       `gorm:"uniqueIndex" json:"link"`
	RawSource string       `gorm:"type:text" json:"raw_source"`
	CreatedAt time.Time    `json:"created_at"`
}

type Subscriber struct {
	ID      string    `gorm:"primaryKey" json:"id"`
	Email   string    `gorm:"uniqueIndex" json:"email"`
	Name    string    `json:"name"`
	AddedAt time.Time `json:"added_at"`
}

type RunLog struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	RunDate    string    `json:"run_date"`
	ItemsCount int       `json:"items_count"`
	Status     string    `json:"status"`
	Log        string    `gorm:"type:text" json:"log"`
	CreatedAt  time.Time `json:"created_at"`
}

type Setting struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

type ChatMessage struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	SessionID    string    `gorm:"index" json:"session_id"`
	Role         string    `json:"role"` // "user" | "model"
	Content      string    `gorm:"type:text" json:"content"`
	ArticleID    string    `json:"article_id,omitempty"`
	ArticleTitle string    `json:"article_title,omitempty"`
	ArticleURL   string    `json:"article_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

var DB *gorm.DB

// InitDB initializes database connection to Google Cloud AlloyDB / PostgreSQL or local SQLite
func InitDB(customPathOrDSN string) (*gorm.DB, error) {
	dsn := customPathOrDSN
	if dsn == "" {
		if envAlloy := os.Getenv("ALLOYDB_DATABASE_URL"); envAlloy != "" {
			dsn = envAlloy
		} else if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
			dsn = envDB
		}
	}

	var dialector gorm.Dialector
	var dbType string

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") || strings.Contains(dsn, "host=") {
		dialector = postgres.Open(dsn)
		dbType = "Google Cloud AlloyDB / PostgreSQL"
	} else {
		dbPath := dsn
		if dbPath == "" {
			dbDir := "data"
			_ = os.MkdirAll(dbDir, 0700)
			dbPath = filepath.Join(dbDir, "ai_daily_brief.db")
		}
		dialector = sqlite.Open(dbPath)
		dbType = "SQLite (" + dbPath + ")"
	}

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// Configure connection pooling for AlloyDB / PostgreSQL
	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(15 * time.Minute)
	}

	// Auto-migrate tables
	if err := DB.AutoMigrate(&NewsItem{}, &Subscriber{}, &RunLog{}, &Setting{}, &ChatMessage{}); err != nil {
		return nil, err
	}

	// Seed default settings if missing
	defaultSettings := map[string]string{
		"cron_schedule":     "0 8 * * *",
		"gemini_model":      "gemini-3.7-flash",
		"gemini_auth_mode":  "vertex_adc", // Cloud Run defaults to Vertex AI ADC
		"gemini_api_key":    "",
		"vertex_project_id": "",
		"vertex_location":   "us-central1",
		"latest_tldr":       "",
		"latest_tldr_date":  "",
	}

	for k, v := range defaultSettings {
		var s Setting
		if err := DB.First(&s, "key = ?", k).Error; err != nil {
			DB.Create(&Setting{Key: k, Value: v})
		}
	}

	log.Printf("[Database] GORM initialized successfully with %s", dbType)
	return DB, nil
}
