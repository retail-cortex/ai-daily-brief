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

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/mcp"
)

func main() {
	portFlag := flag.String("port", "", "Port to listen on (defaults to $PORT, config, or 8080 for Cloud Run)")
	stdioFlag := flag.Bool("stdio", false, "Run in stdio mode for local desktop MCP clients")
	dbFlag := flag.String("db", "", "Path to SQLite database or AlloyDB / PostgreSQL DSN")
	dsnFlag := flag.String("dsn", "", "Google Cloud AlloyDB / PostgreSQL DSN (e.g. postgres://user:pass@host:5432/dbname)")
	flag.Parse()

	cfg := config.LoadConfig()
	dbPath := cfg.GetDatabaseDSN()
	if envDSN := os.Getenv("ALLOYDB_DATABASE_URL"); envDSN != "" {
		dbPath = envDSN
	}
	if *dsnFlag != "" {
		dbPath = *dsnFlag
	} else if *dbFlag != "" {
		dbPath = *dbFlag
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal: Database initialization error: %v", err)
	}

	// 1. If Stdio mode requested
	if *stdioFlag {
		log.Println("[MCP] Starting AI Daily Brief MCP server in stdio mode...")
		if err := mcp.RunStdio(db); err != nil {
			log.Fatalf("Fatal stdio MCP error: %v", err)
		}
		return
	}

	// 2. Cloud Run HTTP Server mode
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = cfg.GetPort()
	}
	if port == "" {
		port = "8080"
	}

	if err := mcp.RunHTTPServer(db, port); err != nil {
		log.Fatalf("Fatal Cloud Run MCP server error: %v", err)
	}
}
