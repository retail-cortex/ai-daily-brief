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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-daily-brief/internal/config"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Agent  *Agent
	Router *gin.Engine
}

type InvokeRequest struct {
	Task string `json:"task" binding:"required"`
}

type ChatRequest struct {
	Message   string `json:"message" binding:"required"`
	SessionID string `json:"session_id,omitempty"`
	ArticleID string `json:"article_id,omitempty"`
}

func NewServer(cfg *config.AgentConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Simple logger
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		if status >= 400 {
			fmt.Printf("[A2A Server] %s %s | %d | %v\n", c.Request.Method, c.Request.URL.Path, status, latency)
		}
	})

	s := &Server{
		Agent:  NewAgent(cfg),
		Router: r,
	}

	s.setupRoutes()
	return s
}

// AgentSkill defines a discrete capability in compliance with Google Agent Registry
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// AgentCapabilities defines supported operational capabilities
type AgentCapabilities struct {
	Streaming         bool             `json:"streaming"`
	PushNotifications bool             `json:"pushNotifications,omitempty"`
	ExtendedAgentCard bool             `json:"extendedAgentCard,omitempty"`
	Extensions        []AgentExtension `json:"extensions,omitempty"`
}

// AgentExtension defines an A2A protocol extension such as A2UI
type AgentExtension struct {
	URI         string                 `json:"uri"`
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

// AgentCard defines the canonical Google Agent Registry A2A Agent Card schema
type AgentCard struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Version            string            `json:"version"`
	ProtocolVersion    string            `json:"protocolVersion"`
	URL                string            `json:"url"`
	DefaultInputModes  []string          `json:"defaultInputModes"`
	DefaultOutputModes []string          `json:"defaultOutputModes"`
	Skills             []AgentSkill      `json:"skills"`
	Capabilities       AgentCapabilities `json:"capabilities"`
	Extensions         []AgentExtension  `json:"extensions,omitempty"`
}

func (s *Server) buildAgentCard(baseURL string) AgentCard {
	skills := []AgentSkill{
		{
			ID:          "research_topic",
			Name:        "Research AI Topic",
			Description: "Perform deep research on a specific AI model, research paper, or cloud infrastructure topic",
			Tags:        []string{"research", "grounding", "search"},
			Examples: []string{
				"Perform deep research on Gemini 3.7 reasoning capabilities",
				"Search latest research papers on multimodal mixture-of-experts",
			},
		},
		{
			ID:          "generate_tldr",
			Name:        "Generate Strategic TLDR",
			Description: "Generate an executive 3-section strategic intelligence summary of today's key developments",
			Tags:        []string{"summary", "executive", "briefing"},
			Examples: []string{
				"Generate today's executive intelligence summary",
				"Summarize key AI infrastructure developments for today",
			},
		},
		{
			ID:          "trigger_crawl",
			Name:        "Trigger Live Intelligence Crawl",
			Description: "Execute an immediate live crawl across all 5 intelligence streams into AlloyDB",
			Tags:        []string{"crawler", "ingestion", "refresh"},
			Examples: []string{
				"Trigger an immediate live crawl of AI feeds",
				"Refresh feeds from arXiv, blogs, and official releases",
			},
		},
		{
			ID:          "chat",
			Name:        "Interactive Grounded Research Chat",
			Description: "Interactive multi-turn research grounded in today's digest or specific articles",
			Tags:        []string{"chat", "dialogue", "grounding"},
			Examples: []string{
				"Tell me more about the architecture behind Gemini 3.7",
				"Compare recent frontier model releases",
			},
		},
	}

	a2uiExt := AgentExtension{
		URI:         "https://a2ui.org/a2a-extension/a2ui/v0.9",
		Description: "Provides agent driven UI using the A2UI JSON format.",
		Required:    false,
		Params: map[string]interface{}{
			"supportedCatalogIds": []string{
				"https://a2ui.org/specification/v0_9/material_catalog.json",
				"https://www.gstatic.com/vertexaisearch/a2ui/v0_9/gemini_enterprise_composite_catalog.json",
				"https://a2ui.org/specification/v0_9/basic_catalog.json",
				"https://a2ui.org",
			},
			"acceptsInlineCatalogs": true,
		},
	}

	return AgentCard{
		Name:               s.Agent.Config.AgentName,
		Description:        "Autonomous AI intelligence agent providing deep research, structured synthesis, executive TL;DR briefings, and live grounding across frontier AI models, academic research papers, open-source tooling, and Google Cloud infrastructure releases backed by the AI Daily Brief MCP control plane.",
		Version:            "1.0.0",
		ProtocolVersion:    "1.0.0",
		URL:                baseURL,
		DefaultInputModes:  []string{"text/plain", "application/a2ui+json", "application/json+a2ui"},
		DefaultOutputModes: []string{"application/json+a2ui", "text/plain", "text/markdown", "application/a2ui+json"},
		Skills:             skills,
		Capabilities: AgentCapabilities{
			Streaming:  true,
			Extensions: []AgentExtension{a2uiExt},
		},
		Extensions: []AgentExtension{a2uiExt},
	}
}

func (s *Server) setupRoutes() {
	// 1. Cloud Run Readiness/Liveness Probe
	s.Router.GET("/healthz", func(c *gin.Context) {
		mcpCfg := s.Agent.Config.GetMCPServer("daily_brief")
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   s.Agent.Config.AgentName,
			"mcp_url":   mcpCfg.URL,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 2. Standardized Agent Card Endpoints (Strict Google Agent Registry A2A compliant)
	agentCardHandler := func(c *gin.Context) {
		scheme := "https"
		if c.Request.TLS == nil && c.Request.Header.Get("X-Forwarded-Proto") != "https" {
			scheme = "http"
		}
		baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
		c.JSON(http.StatusOK, s.buildAgentCard(baseURL))
	}

	// Canonical Google A2A paths
	s.Router.GET("/.well-known/agent-card.json", agentCardHandler)
	s.Router.GET("/.well-known/agent.json", agentCardHandler)
	s.Router.GET("/a2a/app/.well-known/agent-card.json", agentCardHandler)
	s.Router.GET("/a2a/.well-known/agent-card.json", agentCardHandler)
	s.Router.GET("/agent-card.json", agentCardHandler)
	s.Router.GET("/agent.json", agentCardHandler)
	s.Router.GET("/agent-card", agentCardHandler)

	// 3. A2A Invocation Handler supporting both SSE Streaming and JSON-RPC
	a2aHandler := func(c *gin.Context) {
		var a2aReq A2ARequest
		_ = c.ShouldBindJSON(&a2aReq)

		reqID := a2aReq.ID
		if reqID == nil {
			reqID = 1
		}

		accept := c.GetHeader("Accept")
		wantsSSE := strings.Contains(accept, "text/event-stream")

		// If caller requests SSE (like Gemini Enterprise SendStreamingMessage / message/stream)
		if wantsSSE || a2aReq.Method == "message/stream" {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("Transfer-Encoding", "chunked")

			msg, err := s.Agent.BuildMessageResponse(c.Request.Context(), &a2aReq)
			if err != nil {
				errResp := A2AResponse{
					JSONRPC: "2.0",
					ID:      reqID,
					Error: &A2AError{
						Code:    -32603,
						Message: err.Error(),
					},
				}
				data, _ := json.Marshal(errResp)
				_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
				c.Writer.Flush()
				return
			}

			// SendStreamingMessageSuccessResponse with Message
			successResp := A2AResponse{
				JSONRPC: "2.0",
				ID:      reqID,
				Result:  msg,
			}

			data, _ := json.Marshal(successResp)
			c.Stream(func(w io.Writer) bool {
				_, _ = fmt.Fprintf(w, "data: %s\n\n", string(data))
				return false
			})
			return
		}

		// Standard JSON-RPC response
		resp, err := s.Agent.HandleA2A(c.Request.Context(), &a2aReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}

	// Route root POST / and common protocol paths
	s.Router.POST("/", a2aHandler)
	s.Router.POST("/a2a", a2aHandler)
	s.Router.POST("/a2a/app", a2aHandler)
	s.Router.POST("/agent/invoke", a2aHandler)
	s.Router.POST("/run", a2aHandler)

	// 4. Status inspection
	s.Router.GET("/status", func(c *gin.Context) {
		mcpStatus, err := s.Agent.MCPClient.GetSystemStatus(c.Request.Context())
		mcpAvailable := (err == nil)
		geminiCfg := s.Agent.Config.GetGeminiConfig(s.Agent.Config.AgentName)
		mcpCfg := s.Agent.Config.GetMCPServer("daily_brief")
		c.JSON(http.StatusOK, gin.H{
			"agent":          s.Agent.Config.AgentName,
			"mcp_server":     mcpCfg.URL,
			"mcp_connected":  mcpAvailable,
			"mcp_status":     mcpStatus,
			"model":          geminiCfg.Model,
			"model_region":   geminiCfg.Region,
			"auth_mode":      geminiCfg.AuthMode,
			"project_id":     s.Agent.Config.GoogleCloud.ProjectID,
			"project_region": s.Agent.Config.GoogleCloud.ProjectRegion,
		})
	})

	// 5. Conversational Research Grounding (POST /chat)
	s.Router.POST("/chat", func(c *gin.Context) {
		var req ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: message is required", "details": err.Error()})
			return
		}

		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = fmt.Sprintf("a2a_session_%d", time.Now().Unix())
		}

		reply, err := s.Agent.Chat(c.Request.Context(), sessionID, req.Message, req.ArticleID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent dialogue failed", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"reply":      reply,
			"session_id": sessionID,
		})
	})
}

// Run starts the A2A HTTP Server
func (s *Server) Run(ctx context.Context, port string) error {
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	return srv.ListenAndServe()
}

// RunHTTPServer initializes and runs the A2A HTTP Server
func RunHTTPServer(cfg *config.AgentConfig, port string) error {
	srv := NewServer(cfg)
	return srv.Run(context.Background(), port)
}
