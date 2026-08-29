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

package main

import (
	"flag"
	"log"
	"os"

	"ai-daily-brief/internal/a2a"
	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/database"
)

func main() {
	portFlag := flag.String("port", "", "Port to listen on (defaults to $PORT or 8081 for Cloud Run)")
	mcpURLFlag := flag.String("mcp-url", "", "URL of the AI Daily Brief MCP server")
	dbFlag := flag.String("db", "", "Path to SQLite database or AlloyDB / PostgreSQL DSN for direct in-process execution")
	dsnFlag := flag.String("dsn", "", "Google Cloud AlloyDB / PostgreSQL DSN")
	inProcessFlag := flag.Bool("in-process", false, "Force direct in-process tool execution against database")
	flag.Parse()

	cfg := config.LoadAgentConfig()
	if *mcpURLFlag != "" {
		cfg.MCPServerURL = *mcpURLFlag
		if cfg.MCPServers == nil {
			cfg.MCPServers = make(map[string]config.MCPServerConfig)
		}
		srv := cfg.GetMCPServer("daily_brief")
		srv.URL = *mcpURLFlag
		cfg.MCPServers["daily_brief"] = srv
	}

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = cfg.GetPort()
	}
	if port == "" {
		port = "8081"
	}

	// Check if in-process execution is explicitly requested or available
	dbPath := ""
	if *dsnFlag != "" {
		dbPath = *dsnFlag
	} else if *dbFlag != "" {
		dbPath = *dbFlag
	} else if envDSN := os.Getenv("ALLOYDB_DATABASE_URL"); envDSN != "" {
		dbPath = envDSN
	}

	shouldRunInProcess := *inProcessFlag || (*mcpURLFlag == "" && dbPath != "")

	if shouldRunInProcess {
		if dbPath == "" {
			appCfg := config.LoadConfig()
			dbPath = appCfg.GetDatabaseDSN()
		}
		log.Printf("[A2A Agent] Initializing agent '%s' in DIRECT IN-PROCESS mode (Database: %s)...", cfg.AgentName, dbPath)
		db, err := database.InitDB(dbPath)
		if err != nil {
			log.Fatalf("Fatal: Database initialization error for in-process agent: %v", err)
		}
		if err := a2a.RunInProcessHTTPServer(cfg, db, port); err != nil {
			log.Fatalf("Fatal A2A agent error: %v", err)
		}
		return
	}

	mcpCfg := cfg.GetMCPServer("daily_brief")
	log.Printf("[A2A Agent] Initializing agent '%s' in REMOTE MCP mode (MCP Server: %s)...", cfg.AgentName, mcpCfg.URL)
	if err := a2a.RunHTTPServer(cfg, port); err != nil {
		log.Fatalf("Fatal A2A agent error: %v", err)
	}
}
