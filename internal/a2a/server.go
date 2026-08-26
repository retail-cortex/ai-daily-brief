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

package a2a

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-daily-brief/internal/config"

	"github.com/gin-gonic/gin"
)

// Server exposes the A2A Agent as a Cloud Run service
type Server struct {
	Agent  *Agent
	Router *gin.Engine
}

func NewServer(cfg *config.AgentConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Cloud Run CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	s := &Server{
		Agent:  NewAgent(cfg),
		Router: r,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// 1. Cloud Run Readiness/Liveness Probe
	s.Router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   s.Agent.Config.AgentName,
			"mcp_url":   s.Agent.Config.MCPServerURL,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 2. Status inspection
	s.Router.GET("/status", func(c *gin.Context) {
		mcpStatus, err := s.Agent.MCPClient.GetSystemStatus(c.Request.Context())
		mcpAvailable := (err == nil)
		c.JSON(http.StatusOK, gin.H{
			"agent":         s.Agent.Config.AgentName,
			"mcp_server":    s.Agent.Config.MCPServerURL,
			"mcp_connected": mcpAvailable,
			"mcp_status":    mcpStatus,
			"model":         s.Agent.Config.Gemini.Model,
			"auth_mode":     s.Agent.Config.Gemini.AuthMode,
		})
	})

	// 3. A2A Inter-Agent Task Invocation
	invokeHandler := func(c *gin.Context) {
		var req struct {
			Task   string `json:"task"`
			Prompt string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		taskStr := req.Task
		if taskStr == "" {
			taskStr = req.Prompt
		}
		if taskStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "task or prompt is required"})
			return
		}

		res, err := s.Agent.ExecuteTask(c.Request.Context(), taskStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "result": res})
	}

	s.Router.POST("/run", invokeHandler)
	s.Router.POST("/agent/invoke", invokeHandler)

	// 4. Interactive Chat Invocation over MCP
	s.Router.POST("/chat", func(c *gin.Context) {
		var req struct {
			Message   string `json:"message"`
			SessionID string `json:"session_id"`
			ArticleID string `json:"article_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "message is required"})
			return
		}

		args := map[string]interface{}{
			"message": req.Message,
		}
		if req.SessionID != "" {
			args["session_id"] = req.SessionID
		}
		if req.ArticleID != "" {
			args["article_id"] = req.ArticleID
		}

		chatOut, err := s.Agent.MCPClient.CallTool(c.Request.Context(), "agent_chat", args)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "reply": chatOut})
	})
}

// RunHTTPServer starts the A2A Agent HTTP server
func RunHTTPServer(cfg *config.AgentConfig, port string) error {
	srv := NewServer(cfg)

	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" && cfg != nil && cfg.Port != "" {
		port = cfg.Port
	}
	if port == "" {
		port = "8081"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", port))
	if err != nil {
		return fmt.Errorf("failed to bind to port %s: %w", port, err)
	}

	httpSrv := &http.Server{
		Handler: srv.Router,
	}

	log.Printf("====================================================")
	log.Printf("🤖 AI Daily Brief A2A Agent (Cloud Run Ready)")
	log.Printf("🚀 Listening on http://0.0.0.0:%s", port)
	log.Printf("🎯 Consuming MCP Server at %s", srv.Agent.Config.MCPServerURL)
	log.Printf("🩺 Health Probe: GET http://0.0.0.0:%s/healthz", port)
	log.Printf("====================================================")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("A2A Agent HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("[A2A Agent] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpSrv.Shutdown(ctx)
}
