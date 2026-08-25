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

package agent

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	wsRegex      = regexp.MustCompile(`\s+`)
	enrichClient = &http.Client{
		Timeout: 7 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 6 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
)

// FetchFullArticleText downloads the live webpage at articleURL and extracts the core textual body
func FetchFullArticleText(articleURL string) (string, error) {
	if articleURL == "" {
		return "", fmt.Errorf("empty article URL")
	}

	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := enrichClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP error status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("HTML parse error: %w", err)
	}

	// Remove non-content elements
	doc.Find("script, style, nav, header, footer, aside, noscript, svg, iframe, form, button, .cookie-banner, .advertisement, .ad, .social-share").Remove()

	var paragraphs []string
	// Prefer main article container if available
	mainContent := doc.Find("article, main, .article-body, .post-content, .entry-content, [role='main']")
	if mainContent.Length() > 0 {
		mainContent.Find("p, h1, h2, h3, h4, li").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 25 && !strings.HasPrefix(strings.ToLower(text), "cookie") && !strings.HasPrefix(strings.ToLower(text), "subscribe") {
				paragraphs = append(paragraphs, text)
			}
		})
	} else {
		doc.Find("p, h1, h2, h3, li").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 30 && !strings.HasPrefix(strings.ToLower(text), "cookie") {
				paragraphs = append(paragraphs, text)
			}
		})
	}

	fullText := strings.Join(paragraphs, "\n\n")
	fullText = wsRegex.ReplaceAllString(fullText, " ")
	fullText = strings.TrimSpace(fullText)

	// Cap at ~7,000 characters for token efficiency
	if len(fullText) > 7000 {
		fullText = fullText[:7000] + "\n\n...[Content truncated for context window]..."
	}

	if fullText == "" {
		return "", fmt.Errorf("could not extract substantive text from URL")
	}

	return fullText, nil
}
