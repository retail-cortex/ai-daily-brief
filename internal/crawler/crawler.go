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

package crawler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"ai-daily-brief/internal/database"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

type BatchResult struct {
	RunID             string `json:"run_id"`
	RunDate           string `json:"run_date"`
	NewItemsInserted  int    `json:"new_items_inserted"`
	SkippedDuplicates int    `json:"skipped_duplicates"`
	TotalInDB         int    `json:"total_in_db"`
	Status            string `json:"status"`
	Log               string `json:"log"`
}

var (
	htmlTagRegex  = regexp.MustCompile(`<[^>]*>`)
	wsRegex       = regexp.MustCompile(`\s+`)
	sourceSuffix  = regexp.MustCompile(`(?i)\s*-\s*(Reuters|TechCrunch|The Verge|Forbes|WSJ|VentureBeat|Decrypt|SiliconANGLE|Built In|infoq\.com|CNBC|Bloomberg|AWS|Google|OpenAI|Anthropic|xAI|Mashable|9to5Toys|Chrome Unboxed|Yahoo Finance|Motley Fool|Search Engine Roundtable|Android Police|24/7 Wall St).*$`)
	httpClient    = &http.Client{Timeout: 6 * time.Second}
	rssFeedParser = gofeed.NewParser()
)

func makeID(prefix, link string) string {
	hasher := sha256.New()
	hasher.Write([]byte(strings.ToLower(strings.TrimSpace(link))))
	hash := hex.EncodeToString(hasher.Sum(nil))
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return fmt.Sprintf("%s-%s", prefix, hash)
}

func cleanTitle(raw string) string {
	t := html.UnescapeString(raw)
	t = htmlTagRegex.ReplaceAllString(t, "")
	t = sourceSuffix.ReplaceAllString(t, "")
	return strings.TrimSpace(wsRegex.ReplaceAllString(t, " "))
}

func cleanSummary(raw string, maxLen int) string {
	s := html.UnescapeString(raw)
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = strings.TrimSpace(wsRegex.ReplaceAllString(s, " "))
	if s == "" {
		return ""
	}
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func synthesizeSmartSummary(title, company string, category database.NewsCategory) string {
	t := strings.ToLower(title)

	switch category {
	case database.CategoryGoogleCloud:
		return "Google Cloud release notes and infrastructure update covering Vertex AI, TPUs, GKE, and cloud enterprise services."

	case database.CategoryResearchPapers:
		if strings.Contains(t, "benchmark") || strings.Contains(t, "evaluat") {
			return "Academic research paper introducing new benchmark datasets and rigorous evaluation methodology for frontier foundation models."
		}
		if strings.Contains(t, "reasoning") || strings.Contains(t, "thought") {
			return "Research publication presenting novel reasoning architectures, inference-time compute scaling, and chain-of-thought paradigms."
		}
		if strings.Contains(t, "diffusion") || strings.Contains(t, "video") || strings.Contains(t, "image") {
			return "Computer vision and multimodal research paper detailing novel generative architectures, diffusion mechanisms, and visual fidelity metrics."
		}
		return "Academic preprint published on arXiv / open repositories analyzing model architectures, training dynamics, and empirical performance metrics."

	case database.CategoryBusinessInfra:
		if strings.Contains(t, "funding") || strings.Contains(t, "valuation") || strings.Contains(t, "invest") || strings.Contains(t, "round") {
			return "Venture capital and investment analysis detailing valuation multiples, funding allocations, and commercial growth strategies."
		}
		if strings.Contains(t, "gpu") || strings.Contains(t, "blackwell") || strings.Contains(t, "chip") || strings.Contains(t, "cluster") || strings.Contains(t, "datacenter") {
			return "Hardware and compute infrastructure report analyzing datacenter buildouts, next-generation accelerators, and energy scaling."
		}
		if strings.Contains(t, "bedrock") || strings.Contains(t, "azure") || strings.Contains(t, "cloud") || strings.Contains(t, "enterprise") {
			return "Enterprise cloud adoption update detailing hyperscaler model hosting, cross-region latency, and SLA guarantees."
		}
		return "Strategic business intelligence report covering commercial partnerships, enterprise contracts, and regulatory landscape shifts."

	case database.CategoryOSSTooling:
		if strings.Contains(t, "vllm") || strings.Contains(t, "ollama") || strings.Contains(t, "runtime") || strings.Contains(t, "inference") {
			return "High-throughput local inference engine update providing latency optimizations, KV-cache management, and hardware acceleration."
		}
		if strings.Contains(t, "unsloth") || strings.Contains(t, "fine-tun") || strings.Contains(t, "train") {
			return "Developer tooling update optimizing parameter-efficient fine-tuning (PEFT), memory consumption, and training throughput."
		}
		return "Open-source release providing open weights, agent orchestration libraries, and community fine-tuning toolchains."

	default: // Frontier Models
		if strings.Contains(t, "review") || strings.Contains(t, "test") || strings.Contains(t, "benchmark") || strings.Contains(t, "comparison") {
			return "In-depth capability analysis and benchmark evaluation covering reasoning accuracy, token throughput, and pricing efficiency."
		}
		if strings.Contains(t, "search") || strings.Contains(t, "agent") || strings.Contains(t, "mode") || strings.Contains(t, "feature") {
			return "Feature deployment update highlighting real-world integration, autonomous tooling, and user experience enhancements."
		}
		if strings.Contains(t, "free") || strings.Contains(t, "price") || strings.Contains(t, "discount") || strings.Contains(t, "student") || strings.Contains(t, "plan") {
			return "Pricing structure and access tier update detailing commercial API token rates, subscription tiers, and promotional offerings."
		}
		if strings.Contains(t, "watermark") || strings.Contains(t, "safety") || strings.Contains(t, "guardrail") || strings.Contains(t, "security") {
			return "Safety and alignment protocol update covering provenance watermarking, red-teaming defenses, and prompt injection mitigations."
		}
		return fmt.Sprintf("Official release announcement and technical documentation from %s detailing model weights, architecture updates, and API integration.", company)
	}
}

func resolveSummary(rawSnippet, title, company string, category database.NewsCategory) string {
	cleanedSnippet := cleanSummary(rawSnippet, 240)

	prefixLen := len(title)
	if prefixLen > 20 {
		prefixLen = 20
	}
	titlePrefix := strings.ToLower(title[:prefixLen])

	if cleanedSnippet != "" && len(cleanedSnippet) > 40 && !strings.HasPrefix(strings.ToLower(cleanedSnippet), titlePrefix) {
		return cleanedSnippet
	}

	return synthesizeSmartSummary(title, company, category)
}

func fetchGoogleNewsQuery(query, company string, category database.NewsCategory, limit int) []database.NewsItem {
	var items []database.NewsItem
	runDate := time.Now().Format("2006-01-02")
	feedURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=en-US&gl=US&ceid=US:en", url.QueryEscape(query+" when:7d"))

	feed, err := rssFeedParser.ParseURL(feedURL)
	if err != nil {
		log.Printf("[Crawler] Warning: Google News fetch failed for %s: %v", company, err)
		return items
	}

	for idx, item := range feed.Items {
		if idx >= limit {
			break
		}
		if item.Title != "" && item.Link != "" {
			pubDate := runDate
			if item.PublishedParsed != nil {
				pubDate = item.PublishedParsed.Format("2006-01-02")
			}
			cleanedTitle := cleanTitle(item.Title)
			summary := resolveSummary(item.Description, cleanedTitle, company, category)

			items = append(items, database.NewsItem{
				ID:        makeID(strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(company, "")), item.Link),
				RunDate:   runDate,
				PubDate:   pubDate,
				Company:   company,
				Category:  category,
				Title:     cleanedTitle,
				Summary:   summary,
				Link:      strings.TrimSpace(item.Link),
				RawSource: fmt.Sprintf("Google News (%s)", company),
				CreatedAt: time.Now(),
			})
		}
	}

	return items
}

// 1. Frontier Models
func fetchFrontierModels() []database.NewsItem {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []database.NewsItem
	runDate := time.Now().Format("2006-01-02")

	// A. Official AI Lab Feeds
	labFeeds := []struct {
		url     string
		company string
	}{
		{"https://openai.com/news/rss.xml", "OpenAI"},
		{"https://blog.google/innovation-and-ai/technology/ai/rss/", "Google"},
		{"https://huggingface.co/blog/feed.xml", "Hugging Face"},
	}

	for _, lf := range labFeeds {
		wg.Add(1)
		go func(feedURL, comp string) {
			defer wg.Done()
			feed, err := rssFeedParser.ParseURL(feedURL)
			if err != nil {
				return
			}
			var items []database.NewsItem
			for idx, item := range feed.Items {
				if idx >= 4 {
					break
				}
				if item.Title != "" && item.Link != "" {
					pubDate := runDate
					if item.PublishedParsed != nil {
						pubDate = item.PublishedParsed.Format("2006-01-02")
					}
					cleanedTitle := cleanTitle(item.Title)
					summary := resolveSummary(item.Description, cleanedTitle, comp, database.CategoryFrontierModels)
					items = append(items, database.NewsItem{
						ID:        makeID(strings.ToLower(comp), item.Link),
						RunDate:   runDate,
						PubDate:   pubDate,
						Company:   comp,
						Category:  database.CategoryFrontierModels,
						Title:     cleanedTitle,
						Summary:   summary,
						Link:      strings.TrimSpace(item.Link),
						RawSource: fmt.Sprintf("%s Official Feed", comp),
						CreatedAt: time.Now(),
					})
				}
			}
			mu.Lock()
			allItems = append(allItems, items...)
			mu.Unlock()
		}(lf.url, lf.company)
	}

	// B. Targeted News Queries
	queries := []struct {
		q       string
		company string
	}{
		{`("Google DeepMind" OR "Google AI" OR "Gemini") model`, "Google"},
		{`("Anthropic" OR "Claude") model`, "Anthropic"},
		{`("OpenAI" OR "GPT-4" OR "GPT-5" OR "o1" OR "o3" OR "Sora" OR "Operator") model`, "OpenAI"},
		{`("xAI" OR "Grok" OR "Colossus") model`, "X AI"},
		{`("Meta AI" OR "Llama 3" OR "Llama 4") model`, "Meta AI"},
	}

	for _, q := range queries {
		wg.Add(1)
		go func(qStr, comp string) {
			defer wg.Done()
			res := fetchGoogleNewsQuery(qStr, comp, database.CategoryFrontierModels, 4)
			mu.Lock()
			allItems = append(allItems, res...)
			mu.Unlock()
		}(q.q, q.company)
	}

	wg.Wait()
	return allItems
}

// 2. Google Cloud (docs.cloud.google.com/release-notes + GCP News)
func fetchGoogleCloudReleaseNotes() []database.NewsItem {
	var items []database.NewsItem
	runDate := time.Now().Format("2006-01-02")

	// A. Official GCP Release Notes Feed (docs.cloud.google.com/release-notes)
	feedURL := "https://docs.cloud.google.com/feeds/gcp-release-notes.xml"
	feed, err := rssFeedParser.ParseURL(feedURL)
	if err == nil {
		for idx, it := range feed.Items {
			if idx >= 6 {
				break
			}
			pubDate := runDate
			if it.PublishedParsed != nil {
				pubDate = it.PublishedParsed.Format("2006-01-02")
			}

			if it.Content != "" {
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(it.Content))
				if err == nil {
					doc.Find("h2.release-note-product-title").Each(func(i int, s *goquery.Selection) {
						productName := strings.TrimSpace(s.Text())
						if productName == "" {
							return
						}

						var productSummary string
						var productLink string
						var featureTitle string

						next := s.Next()
						for next.Length() > 0 && !next.Is("h2") {
							if next.Is("h3, h4") && featureTitle == "" {
								featureTitle = strings.TrimSpace(next.Text())
							}
							if next.Is("p, li") && productSummary == "" {
								txt := strings.TrimSpace(next.Text())
								if len(txt) > 20 {
									productSummary = cleanSummary(txt, 240)
								}
							}
							if productLink == "" {
								if aHref, exists := next.Find("a[href]").First().Attr("href"); exists && aHref != "" {
									productLink = aHref
								} else if aHref, exists := next.Attr("href"); exists && aHref != "" {
									productLink = aHref
								}
							}
							next = next.Next()
						}

						if productLink == "" {
							productLink = fmt.Sprintf("https://docs.cloud.google.com/release-notes#%s_%s", strings.ReplaceAll(it.Title, " ", "_"), strings.ReplaceAll(productName, " ", "_"))
						}
						if featureTitle == "" {
							featureTitle = "Release Announcement"
						}
						if productSummary == "" {
							productSummary = fmt.Sprintf("Google Cloud update for %s covering new enterprise features, security updates, and performance improvements.", productName)
						}

						title := fmt.Sprintf("%s: %s (%s)", productName, featureTitle, it.Title)
						items = append(items, database.NewsItem{
							ID:        makeID("gcp", productLink+productName),
							RunDate:   runDate,
							PubDate:   pubDate,
							Company:   "Google Cloud",
							Category:  database.CategoryGoogleCloud,
							Title:     title,
							Summary:   productSummary,
							Link:      productLink,
							RawSource: "docs.cloud.google.com/release-notes",
							CreatedAt: time.Now(),
						})
					})
				}
			}
		}
	} else {
		log.Printf("[Crawler] Warning: GCP release notes feed error: %v", err)
	}

	// B. Google Cloud & Vertex AI News Search
	gcpNews := fetchGoogleNewsQuery(`("Google Cloud" OR "Vertex AI" OR "Cloud TPU" OR "AI Hypercomputer") (release OR update OR announcement)`, "Google Cloud", database.CategoryGoogleCloud, 4)
	items = append(items, gcpNews...)

	return items
}

// 3. AI Research Papers
func fetchResearchPapers() []database.NewsItem {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []database.NewsItem
	runDate := time.Now().Format("2006-01-02")

	// A. arXiv API (Expanded to 50 preprints)
	wg.Add(1)
	go func() {
		defer wg.Done()
		arxivURL := "https://export.arxiv.org/api/query?search_query=cat:cs.CL+OR+cat:cs.AI+OR+cat:cs.CV+OR+cat:cs.LG+OR+stat.ML+OR+cs.RO&sortBy=submittedDate&sortOrder=descending&max_results=50"
		req, err := http.NewRequest("GET", arxivURL, nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
		resp, err := httpClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return
		}

		var arxivItems []database.NewsItem
		doc.Find("entry").Each(func(i int, s *goquery.Selection) {
			title := cleanTitle(s.Find("title").Text())
			idURL := strings.TrimSpace(s.Find("id").Text())
			published := strings.TrimSpace(s.Find("published").Text())
			rawSummary := s.Find("summary").Text()

			if title != "" && idURL != "" {
				pubDate := runDate
				if t, err := time.Parse(time.RFC3339, published); err == nil {
					pubDate = t.Format("2006-01-02")
				}
				summary := cleanSummary(rawSummary, 250)
				if summary == "" || strings.EqualFold(summary, title) {
					summary = synthesizeSmartSummary(title, "arXiv", database.CategoryResearchPapers)
				}

				arxivItems = append(arxivItems, database.NewsItem{
					ID:        makeID("arxiv", idURL),
					RunDate:   runDate,
					PubDate:   pubDate,
					Company:   "arXiv / Academic",
					Category:  database.CategoryResearchPapers,
					Title:     title,
					Summary:   summary,
					Link:      idURL,
					RawSource: "arXiv Repository API",
					CreatedAt: time.Now(),
				})
			}
		})

		mu.Lock()
		allItems = append(allItems, arxivItems...)
		mu.Unlock()
	}()

	// B. Hugging Face Daily Papers API (Expanded to 30 papers)
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := httpClient.Get("https://huggingface.co/api/daily_papers?limit=30")
		if err != nil {
			return
		}
		defer resp.Body.Close()

		var papers []struct {
			Paper struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Summary     string `json:"summary"`
				PublishedAt string `json:"publishedAt"`
			} `json:"paper"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&papers); err != nil {
			return
		}

		var hfItems []database.NewsItem
		for _, p := range papers {
			if p.Paper.ID != "" && p.Paper.Title != "" {
				pubDate := runDate
				if p.Paper.PublishedAt != "" {
					if t, err := time.Parse(time.RFC3339, p.Paper.PublishedAt); err == nil {
						pubDate = t.Format("2006-01-02")
					}
				}
				fullLink := fmt.Sprintf("https://huggingface.co/papers/%s", p.Paper.ID)
				cleanedTitle := cleanTitle(p.Paper.Title)
				summary := cleanSummary(p.Paper.Summary, 240)
				if summary == "" || strings.EqualFold(summary, cleanedTitle) {
					summary = synthesizeSmartSummary(cleanedTitle, "Hugging Face", database.CategoryResearchPapers)
				}

				hfItems = append(hfItems, database.NewsItem{
					ID:        makeID("hf", fullLink),
					RunDate:   runDate,
					PubDate:   pubDate,
					Company:   "Hugging Face Research",
					Category:  database.CategoryResearchPapers,
					Title:     cleanedTitle,
					Summary:   summary,
					Link:      fullLink,
					RawSource: "Hugging Face Daily Papers API",
					CreatedAt: time.Now(),
				})
			}
		}

		mu.Lock()
		allItems = append(allItems, hfItems...)
		mu.Unlock()
	}()

	// C. Academic Benchmark Queries
	wg.Add(1)
	go func() {
		defer wg.Done()
		res := fetchGoogleNewsQuery(`AI research paper reasoning foundation model benchmark`, "Research Lab", database.CategoryResearchPapers, 8)
		mu.Lock()
		allItems = append(allItems, res...)
		mu.Unlock()
	}()

	wg.Wait()
	return allItems
}

// 4. AI Business & Infrastructure
func fetchAIBusiness() []database.NewsItem {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []database.NewsItem
	runDate := time.Now().Format("2006-01-02")

	// A. Direct Industry Feeds
	businessFeeds := []struct {
		url     string
		company string
	}{
		{"https://techcrunch.com/category/artificial-intelligence/feed/", "TechCrunch AI"},
		{"https://venturebeat.com/category/ai/feed/", "VentureBeat AI"},
		{"https://siliconangle.com/category/ai/feed/", "SiliconANGLE AI"},
	}

	for _, bf := range businessFeeds {
		wg.Add(1)
		go func(feedURL, comp string) {
			defer wg.Done()
			feed, err := rssFeedParser.ParseURL(feedURL)
			if err != nil {
				return
			}
			var items []database.NewsItem
			for idx, item := range feed.Items {
				if idx >= 6 {
					break
				}
				if item.Title != "" && item.Link != "" {
					pubDate := runDate
					if item.PublishedParsed != nil {
						pubDate = item.PublishedParsed.Format("2006-01-02")
					}
					cleanedTitle := cleanTitle(item.Title)
					summary := resolveSummary(item.Description, cleanedTitle, comp, database.CategoryBusinessInfra)
					items = append(items, database.NewsItem{
						ID:        makeID(strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(comp, "")), item.Link),
						RunDate:   runDate,
						PubDate:   pubDate,
						Company:   comp,
						Category:  database.CategoryBusinessInfra,
						Title:     cleanedTitle,
						Summary:   summary,
						Link:      strings.TrimSpace(item.Link),
						RawSource: fmt.Sprintf("%s RSS", comp),
						CreatedAt: time.Now(),
					})
				}
			}
			mu.Lock()
			allItems = append(allItems, items...)
			mu.Unlock()
		}(bf.url, bf.company)
	}

	// B. Targeted News Queries
	queries := []struct {
		q       string
		company string
	}{
		{`AI startup funding valuation acquisition`, "AI Business & Deals"},
		{`AI datacenter GPU cluster Blackwell compute cloud`, "AI Compute & Infra"},
	}

	for _, q := range queries {
		wg.Add(1)
		go func(qStr, comp string) {
			defer wg.Done()
			res := fetchGoogleNewsQuery(qStr, comp, database.CategoryBusinessInfra, 6)
			mu.Lock()
			allItems = append(allItems, res...)
			mu.Unlock()
		}(q.q, q.company)
	}

	wg.Wait()
	return allItems
}

// 5. OSS & Tooling
func fetchOSSTooling() []database.NewsItem {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []database.NewsItem
	runDate := time.Now().Format("2006-01-02")

	// A. Direct Developer & OSS Feeds
	ossFeeds := []struct {
		url     string
		company string
	}{
		{"https://simonwillison.net/atom/everything/", "Simon Willison"},
		{"https://www.langchain.com/blog/rss", "LangChain"},
		{"https://huggingface.co/blog/feed.xml", "Hugging Face"},
	}

	for _, of := range ossFeeds {
		wg.Add(1)
		go func(feedURL, comp string) {
			defer wg.Done()
			feed, err := rssFeedParser.ParseURL(feedURL)
			if err != nil {
				return
			}
			var items []database.NewsItem
			for idx, item := range feed.Items {
				if idx >= 6 {
					break
				}
				if item.Title != "" && item.Link != "" {
					pubDate := runDate
					if item.PublishedParsed != nil {
						pubDate = item.PublishedParsed.Format("2006-01-02")
					}
					cleanedTitle := cleanTitle(item.Title)
					summary := resolveSummary(item.Description, cleanedTitle, comp, database.CategoryOSSTooling)
					items = append(items, database.NewsItem{
						ID:        makeID(strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(comp, "")), item.Link),
						RunDate:   runDate,
						PubDate:   pubDate,
						Company:   comp,
						Category:  database.CategoryOSSTooling,
						Title:     cleanedTitle,
						Summary:   summary,
						Link:      strings.TrimSpace(item.Link),
						RawSource: fmt.Sprintf("%s RSS", comp),
						CreatedAt: time.Now(),
					})
				}
			}
			mu.Lock()
			allItems = append(allItems, items...)
			mu.Unlock()
		}(of.url, of.company)
	}

	// B. Targeted News Queries
	queries := []struct {
		q       string
		company string
	}{
		{`DeepSeek Qwen Mistral open weights model release`, "Open Weights"},
		{`vLLM Ollama Unsloth llama.cpp fine-tuning agent framework`, "Developer Tooling"},
	}

	for _, q := range queries {
		wg.Add(1)
		go func(qStr, comp string) {
			defer wg.Done()
			res := fetchGoogleNewsQuery(qStr, comp, database.CategoryOSSTooling, 6)
			mu.Lock()
			allItems = append(allItems, res...)
			mu.Unlock()
		}(q.q, q.company)
	}

	wg.Wait()
	return allItems
}

// ExecuteBatchRun runs all crawlers in parallel goroutines and commits non-repeated articles to DB
func ExecuteBatchRun(db *gorm.DB) (*BatchResult, error) {
	runID := fmt.Sprintf("run_%d", time.Now().UnixMilli())
	runDate := time.Now().Format("2006-01-02")
	var logs []string

	startTime := time.Now()
	logs = append(logs, fmt.Sprintf("[%s] Starting parallel Goroutine AI & Cloud Intelligence crawl (%s)...", time.Now().Format(time.RFC3339), runID))

	var wg sync.WaitGroup
	var frontier, gcp, papers, business, oss []database.NewsItem

	wg.Add(5)
	go func() {
		defer wg.Done()
		frontier = fetchFrontierModels()
	}()
	go func() {
		defer wg.Done()
		gcp = fetchGoogleCloudReleaseNotes()
	}()
	go func() {
		defer wg.Done()
		papers = fetchResearchPapers()
	}()
	go func() {
		defer wg.Done()
		business = fetchAIBusiness()
	}()
	go func() {
		defer wg.Done()
		oss = fetchOSSTooling()
	}()

	wg.Wait()

	elapsed := time.Since(startTime)
	logs = append(logs, fmt.Sprintf("  🔵 [Frontier Models] Fetched %d items", len(frontier)))
	logs = append(logs, fmt.Sprintf("  ☁️ [Google Cloud] Fetched %d release notes & updates", len(gcp)))
	logs = append(logs, fmt.Sprintf("  🟣 [AI Research Papers] Fetched %d items", len(papers)))
	logs = append(logs, fmt.Sprintf("  🟢 [AI Business & Infra] Fetched %d items", len(business)))
	logs = append(logs, fmt.Sprintf("  🟡 [OSS & Tooling] Fetched %d items", len(oss)))

	allItems := append(frontier, append(gcp, append(papers, append(business, oss...)...)...)...)
	logs = append(logs, fmt.Sprintf("Total records fetched in %s: %d", elapsed.Round(time.Millisecond), len(allItems)))

	// Targeted chunked deduplication against database to avoid unbounded memory load
	linkSet := make(map[string]struct{})
	var uniqueLinks []string
	for _, item := range allItems {
		cLink := strings.ToLower(strings.TrimSpace(item.Link))
		if cLink != "" {
			if _, seen := linkSet[cLink]; !seen {
				linkSet[cLink] = struct{}{}
				uniqueLinks = append(uniqueLinks, cLink)
			}
		}
	}

	existingMap := make(map[string]struct{})
	chunkSize := 500
	for i := 0; i < len(uniqueLinks); i += chunkSize {
		end := i + chunkSize
		if end > len(uniqueLinks) {
			end = len(uniqueLinks)
		}
		var foundLinks []string
		db.Model(&database.NewsItem{}).Where("LOWER(TRIM(link)) IN ?", uniqueLinks[i:end]).Pluck("LOWER(TRIM(link))", &foundLinks)
		for _, fl := range foundLinks {
			existingMap[fl] = struct{}{}
		}
	}

	newInserted := 0
	skippedDuplicates := 0

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, item := range allItems {
			cLink := strings.ToLower(strings.TrimSpace(item.Link))
			if _, found := existingMap[cLink]; found {
				skippedDuplicates++
			} else {
				if err := tx.Create(&item).Error; err == nil {
					existingMap[cLink] = struct{}{}
					newInserted++
				} else {
					skippedDuplicates++
				}
			}
		}
		return nil
	})

	if err != nil {
		logs = append(logs, fmt.Sprintf("[ERROR] DB Transaction failed: %v", err))
		logText := strings.Join(logs, "\n")
		db.Create(&database.RunLog{ID: runID, RunDate: runDate, ItemsCount: 0, Status: "failed", Log: logText, CreatedAt: time.Now()})
		return &BatchResult{
			RunID:             runID,
			RunDate:           runDate,
			NewItemsInserted:  0,
			SkippedDuplicates: skippedDuplicates,
			TotalInDB:         0,
			Status:            "failed",
			Log:               logText,
		}, err
	}

	var totalInDB int64
	db.Model(&database.NewsItem{}).Count(&totalInDB)

	logs = append(logs, "--------------------------------------------------")
	logs = append(logs, fmt.Sprintf("[SUMMARY] Crawl Completed in %s!", elapsed.Round(time.Millisecond)))
	logs = append(logs, fmt.Sprintf("  ✨ Brand-New Non-Repeated Articles Added: %d", newInserted))
	logs = append(logs, fmt.Sprintf("  ⏩ Previously Existing Duplicates Skipped: %d", skippedDuplicates))
	logs = append(logs, fmt.Sprintf("  📚 Total Database Count: %d", totalInDB))
	logs = append(logs, "--------------------------------------------------")

	logText := strings.Join(logs, "\n")
	db.Create(&database.RunLog{
		ID:         runID,
		RunDate:    runDate,
		ItemsCount: newInserted,
		Status:     "success",
		Log:        logText,
		CreatedAt:  time.Now(),
	})

	return &BatchResult{
		RunID:             runID,
		RunDate:           runDate,
		NewItemsInserted:  newInserted,
		SkippedDuplicates: skippedDuplicates,
		TotalInDB:         int(totalInDB),
		Status:            "success",
		Log:               logText,
	}, nil
}
