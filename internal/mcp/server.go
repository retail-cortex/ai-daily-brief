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

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ai-daily-brief/internal/agent"
	"ai-daily-brief/internal/crawler"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/mailer"
	"ai-daily-brief/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Server encapsulates the MCP Server instance
type Server struct {
	Handler    *Handler
	Router     *gin.Engine
	DB         *gorm.DB
	Cron       *cron.Cron
	sseClients map[string]chan string
	mu         sync.RWMutex
}

func NewServer(db *gorm.DB) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Cloud Run CORS configuration
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	s := &Server{
		Handler:    NewHandler(db),
		Router:     r,
		DB:         db,
		Cron:       cron.New(),
		sseClients: make(map[string]chan string),
	}

	s.setupRoutes()
	s.setupCron()
	return s
}

func (s *Server) setupCron() {
	var sCron database.Setting
	schedule := "0 8 * * *"
	if err := s.DB.First(&sCron, "key = ?", "cron_schedule").Error; err == nil && sCron.Value != "" {
		schedule = sCron.Value
	}

	_, err := s.Cron.AddFunc(schedule, func() {
		log.Println("[Cron] Executing scheduled crawl batch run...")
		res, err := crawler.ExecuteBatchRun(s.DB)
		if err != nil {
			log.Printf("[Cron] Crawler error: %v", err)
			return
		}
		log.Printf("[Cron] Run complete. Inserted %d items.", res.NewItemsInserted)

		// Trigger daily executive TL;DR synthesis
		var items []database.NewsItem
		s.DB.Order("pub_date DESC").Limit(25).Find(&items)
		tldr, err := agent.GenerateDailyTLDR(s.DB, items)
		if err != nil {
			log.Printf("[Cron] TL;DR generation error: %v", err)
		} else {
			log.Printf("[Cron] Daily TL;DR generated successfully (%d bytes)", len(tldr))
		}
	})

	if err != nil {
		log.Printf("[Cron] Error scheduling cron: %v", err)
	} else {
		s.Cron.Start()
		log.Printf("[Cron] Background crawler scheduler started (Schedule: %s)", schedule)
	}
}

func (s *Server) setupRoutes() {
	// 1. Cloud Run Health Check Probe
	s.Router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"server":    "ai-daily-brief-mcp",
			"protocol":  "model-context-protocol-2024-11-05",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 2. Direct JSON-RPC POST Endpoint (/mcp)
	mcpHandler := func(c *gin.Context) {
		var req JSONRPCRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			})
			return
		}

		resp := s.Handler.HandleRequest(req)
		c.JSON(http.StatusOK, resp)
	}

	s.Router.POST("/mcp", mcpHandler)
	s.Router.POST("/api/mcp", mcpHandler)
	s.Router.POST("/api/mcp/call", func(c *gin.Context) {
		var body struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		res, err := s.Handler.ExecuteTool(body.Name, body.Arguments)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "result": res})
	})

	// 3. MCP SSE Streaming Endpoint (/sse)
	s.Router.GET("/sse", func(c *gin.Context) {
		sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
		msgChan := make(chan string, 16)

		s.mu.Lock()
		s.sseClients[sessionID] = msgChan
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.sseClients, sessionID)
			close(msgChan)
			s.mu.Unlock()
		}()

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")

		endpointURL := fmt.Sprintf("/message?session_id=%s", sessionID)
		fmt.Fprintf(c.Writer, "event: endpoint\ndata: %s\n\n", endpointURL)
		c.Writer.Flush()

		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", msg)
				c.Writer.Flush()
			}
		}
	})

	// 4. MCP SSE Message Intake Endpoint (/message)
	s.Router.POST("/message", func(c *gin.Context) {
		sessionID := c.Query("session_id")
		var req JSONRPCRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON-RPC payload"})
			return
		}

		resp := s.Handler.HandleRequest(req)
		respBytes, _ := json.Marshal(resp)

		s.mu.RLock()
		ch, exists := s.sseClients[sessionID]
		s.mu.RUnlock()

		if exists && ch != nil {
			select {
			case ch <- string(respBytes):
				c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
			default:
				log.Printf("[MCP SSE] Buffer full for session %s, dropping frame", sessionID)
				c.JSON(http.StatusOK, resp)
			}
		} else {
			c.JSON(http.StatusOK, resp)
		}
	})

	// 5. REST API Group (/api/*)
	api := s.Router.Group("/api")
	{
		// Articles List
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

			c.JSON(http.StatusOK, gin.H{"success": true, "count": len(items), "items": items})
		})

		// Crawl Trigger
		api.POST("/batch/run", func(c *gin.Context) {
			res, err := crawler.ExecuteBatchRun(s.DB)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "result": res})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "result": res})
		})

		// Crawl Runs History
		api.GET("/runs", func(c *gin.Context) {
			var runs []database.RunLog
			s.DB.Order("created_at DESC").Limit(20).Find(&runs)
			c.JSON(http.StatusOK, gin.H{"success": true, "runs": runs})
		})

		// Newsletter Preview
		api.POST("/newsletter/preview", func(c *gin.Context) {
			var items []database.NewsItem
			s.DB.Order("pub_date DESC").Limit(30).Find(&items)
			dateStr := time.Now().Format("Monday, Jan 02, 2006")
			htmlBody := mailer.GenerateNewsletterHTML(items, dateStr)
			c.JSON(http.StatusOK, gin.H{"success": true, "html": htmlBody})
		})

		// Agent Settings
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
				if k == "gemini_api_key" {
					if valStr == "••••••••••••••••" {
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

		// Agent Model Discovery
		api.GET("/agent/models", func(c *gin.Context) {
			models := agent.FetchAvailableModels(s.DB)
			c.JSON(http.StatusOK, gin.H{"success": true, "models": models})
		})

		// Agent Chat
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

		// Agent Chat History
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

		// TL;DR Generation
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

		// Test Connection
		api.POST("/agent/test-connection", func(c *gin.Context) {
			model := c.Query("model")
			res, err := agent.GenerateRawContentWithModel(s.DB, model, "You are a test validator.", "Respond with exactly: 'Gemini connection successful!'")
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

		// Multimodal Live Bidi WebSocket
		api.GET("/agent/live", func(c *gin.Context) {
			agent.HandleBidiWebSocket(s.DB, c.Writer, c.Request)
		})
	}

	// Root Level WebSocket Live Route
	s.Router.GET("/ws/live", func(c *gin.Context) {
		agent.HandleBidiWebSocket(s.DB, c.Writer, c.Request)
	})
}

// RunHTTPServer starts the MCP server over HTTP for Cloud Run and inter-agent calls
func RunHTTPServer(db *gorm.DB, port string) error {
	srv := NewServer(db)

	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", port))
	if err != nil {
		return fmt.Errorf("failed to bind to port %s: %w", port, err)
	}

	httpSrv := &http.Server{
		Handler: srv.Router,
	}

	log.Printf("====================================================")
	log.Printf("🛡️ AI Daily Brief MCP Server (Cloud Run Ready)")
	log.Printf("🚀 Listening on http://0.0.0.0:%s", port)
	log.Printf("📡 Direct JSON-RPC: POST http://0.0.0.0:%s/mcp", port)
	log.Printf("🌊 SSE Stream:      GET  http://0.0.0.0:%s/sse", port)
	log.Printf("🩺 Health Probe:    GET  http://0.0.0.0:%s/healthz", port)
	log.Printf("====================================================")

	// Graceful shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("MCP Server HTTP error: %v", err)
		}
	}()

	<-stop
	log.Println("[MCP] Shutting down Cloud Run MCP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpSrv.Shutdown(ctx)
}

// RunStdio starts the MCP server over stdin/stdout for local desktop agents
func RunStdio(db *gorm.DB) error {
	handler := NewHandler(db)
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(line) == 0 || string(line) == "\n" || string(line) == "\r\n" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			}
			out, _ := json.Marshal(resp)
			fmt.Println(string(out))
			continue
		}

		resp := handler.HandleRequest(req)
		out, err := json.Marshal(resp)
		if err == nil {
			fmt.Println(string(out))
		}
	}
}
