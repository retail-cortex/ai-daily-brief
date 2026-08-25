package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
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
	Summary   string       `json:"summary"`
	Link      string       `gorm:"uniqueIndex" json:"link"`
	RawSource string       `json:"raw_source"`
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
	Value string `json:"value"`
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

func InitDB(customPath string) (*gorm.DB, error) {
	dbDir := "data"
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dbDir, "news_agent.db")
	if customPath != "" {
		dbPath = customPath
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate tables
	if err := DB.AutoMigrate(&NewsItem{}, &Subscriber{}, &RunLog{}, &Setting{}, &ChatMessage{}); err != nil {
		return nil, err
	}

	// Seed default settings if missing
	defaultSettings := map[string]string{
		"cron_schedule":     "0 8 * * *",
		"gemini_model":      "gemini-3.7-flash",
		"gemini_auth_mode":  "api_key", // "api_key" or "vertex_adc"
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

	log.Printf("[Database] GORM SQLite initialized successfully at %s", dbPath)
	return DB, nil
}
