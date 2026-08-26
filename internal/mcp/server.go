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
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server encapsulates the MCP Server instance
type Server struct {
	Handler    *Handler
	Router     *gin.Engine
	sseClients map[string]chan string
	mu         sync.RWMutex
}

func NewServer(db *gorm.DB) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Cloud Run CORS configuration
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
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
		sseClients: make(map[string]chan string),
	}

	s.setupRoutes()
	return s
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

	// 2. Direct JSON-RPC POST Endpoint (/mcp and /api/mcp)
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

		// Send endpoint registration event per MCP SSE spec
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
			ch <- string(respBytes)
			c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
		} else {
			// Direct response fallback
			c.JSON(http.StatusOK, resp)
		}
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
