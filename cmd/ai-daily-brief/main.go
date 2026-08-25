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
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/crawler"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/server"
)

//go:embed dist/*
var embedDist embed.FS

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		log.Printf("[Launcher] Open browser hint: Navigate to %s", url)
	}
}

func main() {
	batchOnly := flag.Bool("batch", false, "Run batch crawler headlessly and exit")
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open browser on startup")
	port := flag.String("port", "3001", "Port to listen on")
	flag.Parse()

	log.Println("====================================================")
	log.Println("⚡ AI Daily Brief (Go Standalone)")
	log.Println("====================================================")

	cfg := config.LoadConfig()

	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Fatal: Database initialization error: %v", err)
	}

	// If headless batch mode requested
	if *batchOnly {
		log.Println("[Mode] Headless Goroutine Batch Execution...")
		res, err := crawler.ExecuteBatchRun(db)
		if err != nil {
			log.Fatalf("Batch failed: %v", err)
		}
		fmt.Println("\n----------------------------------------------------")
		fmt.Printf("Run Date:            %s\n", res.RunDate)
		fmt.Printf("Status:              %s\n", res.Status)
		fmt.Printf("New Non-Repeated:    %d\n", res.NewItemsInserted)
		fmt.Printf("Duplicates Skipped:  %d\n", res.SkippedDuplicates)
		fmt.Printf("Total DB Articles:   %d\n", res.TotalInDB)
		fmt.Println("----------------------------------------------------")
		fmt.Println(res.Log)
		return
	}

	// Check if DB is empty, run initial crawl in background
	var count int64
	db.Model(&database.NewsItem{}).Count(&count)
	if count == 0 {
		log.Println("[Init] Empty database detected. Running initial Goroutine crawl...")
		go func() {
			crawler.ExecuteBatchRun(db)
		}()
	}

	srv := server.NewServer(db, embedDist)
	portVal := *port
	if portVal == "3001" && cfg.Port != "" {
		portVal = cfg.Port
	}
	addr := fmt.Sprintf("0.0.0.0:%s", portVal)

	// Auto-open browser if running interactively
	if !*noBrowser {
		go func() {
			time.Sleep(1 * time.Second)
			openBrowser(fmt.Sprintf("http://localhost:%s", portVal))
		}()
	}

	log.Printf("🚀 Unified Go Server listening on http://localhost:%s", portVal)
	if err := srv.Router.Run(addr); err != nil && err != http.ErrServerClosed {
		// Try fallback port if in use
		if _, ok := err.(*net.OpError); ok {
			fallbackAddr := "0.0.0.0:3002"
			log.Printf("Port %s busy, falling back to http://localhost:3002", portVal)
			_ = srv.Router.Run(fallbackAddr)
		} else {
			log.Fatalf("Fatal server error: %v", err)
		}
	}
}
