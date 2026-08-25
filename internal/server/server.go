package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-daily-brief/internal/agent"
	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/crawler"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/mailer"
	"ai-daily-brief/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Server struct {
	DB        *gorm.DB
	Router    *gin.Engine
	Cron      *cron.Cron
	EmbedDist embed.FS
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func NewServer(db *gorm.DB, embedDist embed.FS) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())

	s := &Server{
		DB:        db,
		Router:    r,
		Cron:      cron.New(),
		EmbedDist: embedDist,
	}

	s.setupRoutes()
	s.setupCron()
	return s
}

func (s *Server) setupRoutes() {
	api := s.Router.Group("/api")
	{
		// 1. GET /api/items
		api.GET("/items", func(c *gin.Context) {
			search := c.Query("search")
			company := c.Query("company")
			category := c.Query("category")
			limitStr := c.Query("limit")

			query := s.DB.Model(&database.NewsItem{})

			if company != "" && company != "All" {
				query = query.Where("company LIKE ?", "%"+company+"%")
			}
			if category != "" && category != "All" {
				query = query.Where("category = ?", category)
			}
			if search != "" {
				term := "%" + search + "%"
				query = query.Where("title LIKE ? OR summary LIKE ? OR company LIKE ?", term, term, term)
			}

			query = query.Order("pub_date DESC, created_at DESC")

			if limitStr != "" {
				if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
					query = query.Limit(l)
				}
			}

			var items []database.NewsItem
			if err := query.Find(&items).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"count":   len(items),
				"items":   items,
			})
		})

		// 2. POST /api/batch/run
		api.POST("/batch/run", func(c *gin.Context) {
			res, err := crawler.ExecuteBatchRun(s.DB)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "result": res})
				return
			}

			// Generate TL;DR asynchronously
			go func() {
				var latestItems []database.NewsItem
				s.DB.Order("pub_date DESC").Limit(25).Find(&latestItems)
				agent.GenerateDailyTLDR(s.DB, latestItems)
			}()

			c.JSON(http.StatusOK, gin.H{"success": true, "result": res})
		})

		// 3. GET /api/runs
		api.GET("/runs", func(c *gin.Context) {
			var runs []database.RunLog
			s.DB.Order("created_at DESC").Limit(20).Find(&runs)
			c.JSON(http.StatusOK, gin.H{"success": true, "runs": runs})
		})

		// 4. Newsletter Preview
		api.POST("/newsletter/preview", func(c *gin.Context) {
			var items []database.NewsItem
			s.DB.Order("pub_date DESC").Limit(30).Find(&items)
			dateStr := time.Now().Format("Monday, Jan 02, 2006")
			htmlBody := mailer.GenerateNewsletterHTML(items, dateStr)
			c.JSON(http.StatusOK, gin.H{"success": true, "html": htmlBody})
		})

		// 5. Agent Settings Endpoints
		api.GET("/settings", func(c *gin.Context) {
			var settings []database.Setting
			s.DB.Find(&settings)
			setMap := make(map[string]string)
			for _, st := range settings {
				if st.Key == "gemini_api_key" && st.Value != "" {
					setMap[st.Key] = "••••••••••••••••"
				} else {
					setMap[st.Key] = st.Value
				}
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "settings": setMap})
		})

		api.POST("/settings", func(c *gin.Context) {
			var req map[string]interface{}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
				return
			}

			for k, v := range req {
				valStr := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(v.(string), "\n", ""), "\r", ""))
				
				// Handle API Key security
				if k == "gemini_api_key" {
					if valStr == "••••••••••••••••" {
						// Don't overwrite existing encrypted key with the mask placeholder
						continue
					}
					if valStr != "" {
						encrypted, err := security.Encrypt(valStr)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Encryption failed: " + err.Error()})
							return
						}
						valStr = encrypted
					}
				}

				s.DB.Save(&database.Setting{Key: k, Value: valStr})
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Settings saved successfully"})
		})

		// 6. Agent Model Discovery
		api.GET("/agent/models", func(c *gin.Context) {
			models := agent.FetchAvailableModels(s.DB)
			c.JSON(http.StatusOK, gin.H{"success": true, "models": models})
		})

		// 7. Interactive Agent Chat Endpoint (Q&A with On-Demand Grounding)
		api.POST("/agent/chat", func(c *gin.Context) {
			var req struct {
				SessionID string `json:"session_id"`
				Message   string `json:"message"`
				ArticleID string `json:"article_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Message is required"})
				return
			}

			res, err := agent.GenerateChatResponse(s.DB, req.SessionID, req.Message, req.ArticleID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "result": res})
		})

		// 8. Agent Chat History
		api.GET("/agent/history", func(c *gin.Context) {
			sessionID := c.Query("session_id")
			var history []database.ChatMessage
			query := s.DB.Order("created_at ASC")
			if sessionID != "" {
				query = query.Where("session_id = ?", sessionID)
			} else {
				query = query.Limit(50)
			}
			query.Find(&history)
			c.JSON(http.StatusOK, gin.H{"success": true, "history": history})
		})

		api.DELETE("/agent/history", func(c *gin.Context) {
			sessionID := c.Query("session_id")
			if sessionID != "" {
				s.DB.Where("session_id = ?", sessionID).Delete(&database.ChatMessage{})
			} else {
				s.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&database.ChatMessage{})
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "History cleared"})
		})

		// 9. LLM TL;DR Endpoint
		api.POST("/agent/tldr", func(c *gin.Context) {
			var items []database.NewsItem
			s.DB.Order("pub_date DESC").Limit(25).Find(&items)
			tldr, err := agent.GenerateDailyTLDR(s.DB, items)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "tldr": tldr})
		})

		// 10. Test Connection to Gemini / Vertex AI
		api.POST("/agent/test-connection", func(c *gin.Context) {
			model := c.Query("model")
			if model == "" {
				var sModel database.Setting
				s.DB.First(&sModel, "key = ?", "gemini_model")
				model = sModel.Value
				if model == "" {
					model = "gemini-3.7-flash"
				}
			}

			res, err := agent.GenerateRawContent(s.DB, "You are a test validator.", "Respond with exactly: 'Gemini connection successful!'")
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"error":   err.Error(),
					"model":   model,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": res,
				"model":   model,
			})
		})

		// 11. Multimodal Live Bidi WebSocket Endpoint
		api.GET("/agent/live", func(c *gin.Context) {
			agent.HandleBidiWebSocket(s.DB, c.Writer, c.Request)
		})
	}

	// Embedded React Frontend SPA Handler
	distFS, err := fs.Sub(s.EmbedDist, "dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(distFS))
		s.Router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
				return
			}
			filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
			if filePath != "" {
				if f, err := distFS.Open(filePath); err == nil {
					f.Close()
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
			indexData, err := fs.ReadFile(distFS, "index.html")
			if err == nil {
				c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Writer.Header().Set("Pragma", "no-cache")
				c.Writer.Header().Set("Expires", "0")
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
				return
			}
			c.String(http.StatusOK, "AI Daily Brief - Ready")
		})
	}
}

func (s *Server) setupCron() {
	var expr string
	if config.AppConfig != nil {
		expr = config.AppConfig.CronSchedule
	}
	if expr == "" {
		var sched database.Setting
		s.DB.First(&sched, "key = ?", "cron_schedule")
		expr = sched.Value
	}
	if expr == "" {
		expr = "0 8 * * *"
	}

	_, err := s.Cron.AddFunc(expr, func() {
		log.Println("[Scheduler] Executing periodic automated Goroutine crawl...")
		res, err := crawler.ExecuteBatchRun(s.DB)
		if err == nil && res != nil {
			var latestItems []database.NewsItem
			s.DB.Order("pub_date DESC").Limit(25).Find(&latestItems)
			agent.GenerateDailyTLDR(s.DB, latestItems)
		}
	})
	if err == nil {
		s.Cron.Start()
		log.Printf("[Scheduler] Active background cron schedule: %s", expr)
	} else {
		log.Printf("[Scheduler] Cron parse warning: %v", err)
	}
}
