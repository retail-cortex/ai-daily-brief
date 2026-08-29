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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-daily-brief/internal/config"
	"ai-daily-brief/internal/database"
	"ai-daily-brief/internal/security"

	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

var geminiHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

type ChatResponse struct {
	Response     string `json:"response"`
	Grounded     bool   `json:"grounded"`
	ArticleTitle string `json:"article_title,omitempty"`
	ArticleURL   string `json:"article_url,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiRequest struct {
	SystemInstruction *GeminiContent  `json:"system_instruction,omitempty"`
	Contents          []GeminiContent `json:"contents"`
	GenerationConfig  *struct {
		Temperature     float32 `json:"temperature,omitempty"`
		MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	} `json:"generationConfig,omitempty"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func getSetting(db *gorm.DB, key, defaultValue string) string {
	var s database.Setting
	if err := db.First(&s, "key = ?", key).Error; err == nil && strings.TrimSpace(s.Value) != "" {
		val := strings.TrimSpace(s.Value)
		if key == "gemini_api_key" {
			if decrypted, err := security.Decrypt(val); err == nil {
				return decrypted
			}
		}
		return val
	}
	return defaultValue
}

// GetAgentSettings retrieves LLM configuration prioritizing runtime database settings over static config
func GetAgentSettings(db *gorm.DB) (model, authMode, apiKey, projectID, modelRegion string) {
	// 1. Read DB settings first (User GUI configured settings)
	model = getSetting(db, "gemini_model", "")
	authMode = getSetting(db, "gemini_auth_mode", "")
	apiKey = getSetting(db, "gemini_api_key", "")
	projectID = getSetting(db, "vertex_project_id", "")
	modelRegion = getSetting(db, "vertex_location", "")
	if modelRegion == "" {
		modelRegion = getSetting(db, "model_region", "")
	}

	// 2. Check environment variables
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" && apiKey == "" {
		apiKey = envKey
	}
	if envProject := os.Getenv("GOOGLE_CLOUD_PROJECT"); envProject != "" && projectID == "" {
		projectID = envProject
	} else if envGCP := os.Getenv("GCP_PROJECT"); envGCP != "" && projectID == "" {
		projectID = envGCP
	}
	if envRegion := os.Getenv("GEMINI_MODEL_REGION"); envRegion != "" && modelRegion == "" {
		modelRegion = envRegion
	}

	// 3. Fallback to static .env.toml config file if not set in DB
	if config.AppConfig != nil {
		defaultGemini := config.AppConfig.GetGeminiConfig("default")
		if model == "" {
			model = defaultGemini.Model
		}
		if authMode == "" {
			authMode = defaultGemini.AuthMode
		}
		if apiKey == "" {
			apiKey = defaultGemini.APIKey
		}
		if projectID == "" {
			projectID = config.AppConfig.GoogleCloud.ProjectID
		}
		if modelRegion == "" {
			modelRegion = defaultGemini.Region
		}
		if modelRegion == "" {
			modelRegion = config.AppConfig.GoogleCloud.ProjectRegion
		}
	}

	// 4. Default values
	if model == "" {
		model = "gemini-3.7-flash"
	}
	if authMode == "" {
		authMode = "vertex_adc"
	}
	if modelRegion == "" {
		modelRegion = "global"
	}
	return
}

// getVertexCredentials fetches GCP Application Default Credentials (ADC) and OAuth2 token
func getVertexCredentials(ctx context.Context) (*google.Credentials, string, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, "", fmt.Errorf("Google Cloud Application Default Credentials (ADC) not found: %w. Run 'gcloud auth application-default login' to authenticate", err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve OAuth2 token from ADC: %w", err)
	}
	return creds, token.AccessToken, nil
}

// GenerateRawContent sends the request using configured model and auth method
func GenerateRawContent(db *gorm.DB, systemInstruction, userPrompt string) (string, error) {
	return GenerateRawContentWithModel(db, "", systemInstruction, userPrompt)
}

// GenerateRawContentWithModel sends the request to either Google AI Studio or Vertex AI ADC with a specific model
func GenerateRawContentWithModel(db *gorm.DB, requestedModel, systemInstruction, userPrompt string) (string, error) {
	model, authMode, apiKey, projectID, location := GetAgentSettings(db)
	if requestedModel != "" {
		model = requestedModel
	}
	model = strings.TrimPrefix(model, "models/")

	reqPayload := GeminiRequest{
		Contents: []GeminiContent{
			{
				Role:  "user",
				Parts: []GeminiPart{{Text: userPrompt}},
			},
		},
		GenerationConfig: &struct {
			Temperature     float32 `json:"temperature,omitempty"`
			MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
		}{
			Temperature:     0.4,
			MaxOutputTokens: 2048,
		},
	}

	if systemInstruction != "" {
		reqPayload.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemInstruction}},
		}
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", err
	}

	var endpoint string
	var req *http.Request

	if authMode == "vertex_adc" {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		creds, token, err := getVertexCredentials(ctx)
		if err != nil {
			return "", err
		}

		if projectID == "" && creds.ProjectID != "" {
			projectID = creds.ProjectID
		}
		if projectID == "" {
			return "", fmt.Errorf("Google Cloud Project ID is required for Vertex AI ADC mode. Please enter your GCP Project ID in Agent Settings")
		}

		if location == "" || strings.HasPrefix(model, "gemini-3.") {
			location = "global"
		}

		var endpointHost string
		if location == "global" {
			endpointHost = "aiplatform.googleapis.com"
		} else {
			endpointHost = fmt.Sprintf("%s-aiplatform.googleapis.com", location)
		}

		endpoint = fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent", endpointHost, projectID, location, model)

		req, err = http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
	} else {
		// API Key Mode
		if apiKey == "" {
			return "", fmt.Errorf("Gemini API key is not configured. Please enter your API key in Agent Settings or switch to Vertex AI (ADC)")
		}
		endpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		req, err = http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := geminiHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("Gemini API Error (%d): %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini model")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// BuildNewsletterContext generates a structured text summary of the entire daily newsletter for base context
func BuildNewsletterContext(db *gorm.DB) (string, string) {
	var tldrSetting database.Setting
	tldrText := ""
	if err := db.First(&tldrSetting, "key = ?", "latest_tldr").Error; err == nil && tldrSetting.Value != "" {
		tldrText = tldrSetting.Value
	}

	var items []database.NewsItem
	db.Order("pub_date DESC").Limit(30).Find(&items)

	var sb strings.Builder
	sb.WriteString("--- BASE CONTEXT: DAILY AI & CLOUD INTELLIGENCE NEWSLETTER ---\n")
	if tldrText != "" {
		sb.WriteString("Executive TL;DR & Strategic Analysis:\n" + tldrText + "\n\n")
	}

	sb.WriteString("Indexed Intelligence Items:\n")
	for i, it := range items {
		sb.WriteString(fmt.Sprintf("%d. [%s | %s] %s\n   Summary: %s\n   Link: %s\n", i+1, it.Category, it.Company, it.Title, it.Summary, it.Link))
	}
	sb.WriteString("--- END OF NEWSLETTER BASE CONTEXT ---")

	return "Daily AI & Cloud Intelligence Digest (Full Newsletter)", sb.String()
}

// GenerateChatResponse processes an interactive user message with optional article or newsletter grounding
func GenerateChatResponse(db *gorm.DB, sessionID, userMessage, articleID string) (*ChatResponse, error) {
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess_%d", time.Now().UnixMilli())
	}

	var articleTitle, articleURL, articleText string
	isGrounded := false

	if articleID != "" && articleID != "newsletter" {
		var item database.NewsItem
		if err := db.First(&item, "id = ?", articleID).Error; err == nil {
			articleTitle = item.Title
			articleURL = item.Link

			// Fetch live webpage text for context grounding
			fetchedText, err := FetchFullArticleText(item.Link)
			if err == nil && len(fetchedText) > 50 {
				articleText = fetchedText
				isGrounded = true
			} else {
				// Fallback to title and summary
				articleText = fmt.Sprintf("Title: %s\nCompany/Source: %s\nCategory: %s\nSummary: %s\nLink: %s", item.Title, item.Company, item.Category, item.Summary, item.Link)
				isGrounded = true
			}
		}
	} else {
		// Automatically include the current full newsletter as the base context
		nlTitle, nlContent := BuildNewsletterContext(db)
		articleTitle = nlTitle
		articleURL = "/newsletter"
		articleText = nlContent
		isGrounded = true
	}

	// Fetch recent chat history
	var history []database.ChatMessage
	db.Where("session_id = ?", sessionID).Order("created_at ASC").Limit(10).Find(&history)

	var conversationText strings.Builder
	for _, msg := range history {
		if msg.Role == "user" {
			conversationText.WriteString("User: " + msg.Content + "\n\n")
		} else {
			conversationText.WriteString("Assistant: " + msg.Content + "\n\n")
		}
	}

	systemInstruction := `You are an expert AI & Cloud Intelligence Research Assistant.
You have access to live articles, technical release notes, and research papers across Google, Anthropic, OpenAI, X AI, Meta, Google Cloud, arXiv, and open-source models.
Always answer questions factually, thoroughly, and professionally. Format all code, models, and comparisons in clear GitHub-style Markdown.`

	var userPrompt strings.Builder
	if isGrounded {
		userPrompt.WriteString(fmt.Sprintf("--- GROUNDED CONTEXT FROM ARTICLE: \"%s\" ---\nURL: %s\n\n%s\n--- END OF GROUNDED CONTEXT ---\n\n", articleTitle, articleURL, articleText))
	}

	if conversationText.Len() > 0 {
		userPrompt.WriteString("Recent Conversation History:\n" + conversationText.String() + "\n")
	}

	userPrompt.WriteString("User's Question:\n" + userMessage)

	// Save user turn to DB
	db.Create(&database.ChatMessage{
		ID:           fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		SessionID:    sessionID,
		Role:         "user",
		Content:      userMessage,
		ArticleID:    articleID,
		ArticleTitle: articleTitle,
		ArticleURL:   articleURL,
		CreatedAt:    time.Now(),
	})

	// Call Gemini model
	responseTxt, err := GenerateRawContent(db, systemInstruction, userPrompt.String())
	if err != nil {
		// Return graceful error response
		responseTxt = fmt.Sprintf("⚠️ **Agent Notification**: Could not complete Gemini model request.\n\n*Details*: %v\n\n*Tip*: Check your **Agent Settings** in the top bar to verify your API Key or Vertex AI ADC configuration.", err)
	}

	// Save model response to DB
	db.Create(&database.ChatMessage{
		ID:           fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		SessionID:    sessionID,
		Role:         "model",
		Content:      responseTxt,
		ArticleID:    articleID,
		ArticleTitle: articleTitle,
		ArticleURL:   articleURL,
		CreatedAt:    time.Now(),
	})

	return &ChatResponse{
		Response:     responseTxt,
		Grounded:     isGrounded,
		ArticleTitle: articleTitle,
		ArticleURL:   articleURL,
	}, nil
}

// GenerateDailyTLDR generates an executive strategic briefing across all 5 streams
func GenerateDailyTLDR(db *gorm.DB, items []database.NewsItem) (string, error) {
	if len(items) == 0 {
		return "No intelligence items recorded for today.", nil
	}

	var itemsSummary strings.Builder
	for idx, it := range items {
		if idx >= 25 {
			break
		}
		itemsSummary.WriteString(fmt.Sprintf("- [%s | %s] %s: %s\n", it.Company, it.Category, it.Title, it.Summary))
	}

	systemInstruction := `You are an elite Executive AI & Cloud Intelligence Analyst.
Analyze today's news items across Frontier Models, Google Cloud, AI Research Papers, AI Business, and OSS Tooling.
Synthesize a concise, high-impact Daily Executive TL;DR & Strategic Analysis in Markdown with 3 sections:
1. ⚡ **Top Frontier Breakthroughs** (2-3 punchy bullet points)
2. ☁️ **Cloud, Compute & Infrastructure** (2-3 punchy bullet points)
3. 🔮 **Key Strategic Takeaway** (1 concise synthesis of what this means for builders and enterprise teams)`

	userPrompt := fmt.Sprintf("Here are today's top indexed intelligence records:\n\n%s\n\nPlease produce the executive Daily TL;DR synthesis.", itemsSummary.String())

	tldr, err := GenerateRawContent(db, systemInstruction, userPrompt)
	if err != nil {
		log.Printf("[Agent] Gemini TL;DR generation failed: %v. Falling back to heuristic synthesis.", err)
		// Fallback heuristic synthesis if LLM key is not configured
		dateStr := time.Now().Format("Jan 02, 2006")
		return fmt.Sprintf(`### ⚡ Daily Intelligence Overview (%s)

- **Frontier Models**: Active release cadence across Google Gemini, Anthropic Claude, OpenAI, and X AI Grok with strong focus on inference efficiency and agent workflows.
- **Google Cloud & Compute**: New documentation and GPU machine series expansions (AI Hypercomputer, A3 Edge, G4) alongside Vertex AI managed vector search.
- **Open-Source & Papers**: Continued proliferation of open-weights preprints and runtime acceleration frameworks optimizing developer fine-tuning.

*(Configure your Gemini 3.7 API Key or Vertex AI ADC in Agent Settings for live LLM executive synthesis)*`, dateStr), nil
	}

	// Persist TL;DR in settings
	dateStr := time.Now().Format("2006-01-02")
	db.Save(&database.Setting{Key: "latest_tldr", Value: tldr})
	db.Save(&database.Setting{Key: "latest_tldr_date", Value: dateStr})

	return tldr, nil
}

// GenerateTimespanOverview generates an executive multi-stream overview covering a specified time span
func GenerateTimespanOverview(db *gorm.DB, items []database.NewsItem, timeSpanTitle string) (string, error) {
	if len(items) == 0 {
		return fmt.Sprintf("No intelligence items recorded for %s.", timeSpanTitle), nil
	}

	var itemsSummary strings.Builder
	for idx, it := range items {
		if idx >= 40 {
			break
		}
		dateStr := it.PubDate
		if dateStr == "" {
			dateStr = it.CreatedAt.Format("Jan 02")
		}
		itemsSummary.WriteString(fmt.Sprintf("- [%s | %s | %s] %s: %s\n", it.Company, it.Category, dateStr, it.Title, it.Summary))
	}

	systemInstruction := `You are an elite Executive AI & Cloud Intelligence Analyst.
Analyze the multi-stream intelligence developments across Frontier Models, Google Cloud, AI Research Papers, AI Business & Infrastructure, and OSS Tooling over the specified timeframe.
Synthesize a comprehensive, high-impact Executive Intelligence Overview in GitHub-style Markdown structured in 4 sections:
1. 🚀 **Frontier & Architecture Breakthroughs** (Key model weights, multimodal reasoning, benchmark scores)
2. ☁️ **Enterprise Cloud & Infrastructure Movements** (Compute, TPUs/GPUs, managed orchestration, and platform shifts)
3. 🔬 **Notable Academic & Open-Source Advances** (Prominent preprints, agent frameworks, and tooling releases)
4. 🔮 **Strategic Executive Takeaway** (Core synthesis of what these shifts mean for enterprise technology leaders)`

	userPrompt := fmt.Sprintf("Here are the top indexed intelligence records from %s (%d articles):\n\n%s\n\nPlease produce the comprehensive Executive Timespan Overview.", timeSpanTitle, len(items), itemsSummary.String())

	overview, err := GenerateRawContent(db, systemInstruction, userPrompt)
	if err != nil {
		log.Printf("[Agent] Gemini timespan overview generation failed: %v. Falling back to heuristic synthesis.", err)
		dateStr := time.Now().Format("Jan 02, 2006")
		return fmt.Sprintf(`### ⚡ Executive Overview — %s (%s)

#### 1. Frontier & Architecture Trends
- Tracking %d indexed developments across frontier AI, research papers, and cloud releases.

#### 2. Infrastructure & Tooling
- Key movements captured across Google Cloud Vertex AI, open-source model repositories, and enterprise AI stacks.

#### 3. Strategic Summary
- Continuous velocity across foundational model reasoning, agent frameworks, and inference optimization.`, timeSpanTitle, dateStr, len(items)), nil
	}
	return overview, nil
}

// GenerateArticleSummary generates a structured technical and strategic synthesis for a specific article
func GenerateArticleSummary(db *gorm.DB, item database.NewsItem, fullText string) (string, error) {
	content := strings.TrimSpace(fullText)
	if content == "" {
		content = strings.TrimSpace(item.Summary)
	}
	if len(content) > 15000 {
		content = content[:15000] + "\n\n[Content truncated for analysis]"
	}

	systemInstruction := `You are a Principal AI Systems Architect and Senior Cloud Strategist.
Analyze the provided article/release/paper and produce a structured, high-value Technical & Strategic Article Summary in GitHub-style Markdown.
Structure your synthesis into:
1. 📌 **Executive Overview & Key Announcements** (Core focus, problems addressed, major features/findings)
2. 🏗️ **Technical Architecture & Method** (Underlying mechanisms, system topology, model architectures, algorithms, integrations)
3. 📊 **Key Capabilities & Metrics** (Reported benchmark metrics, latency/cost improvements, empirical evidence)
4. 💡 **Strategic Takeaway** (Key takeaway for AI practitioners and enterprise cloud teams)`

	userPrompt := fmt.Sprintf("Article Title: %s\nCompany / Source: %s\nCategory: %s\nDirect Link: %s\n\nFull Grounded Content:\n%s\n\nPlease produce the structured technical article summary.",
		item.Title, item.Company, item.Category, item.Link, content)

	summary, err := GenerateRawContent(db, systemInstruction, userPrompt)
	if err != nil {
		log.Printf("[Agent] Gemini article summary failed: %v. Using fallback summary.", err)
		if item.Summary != "" {
			return fmt.Sprintf("### 📌 Overview\n\n%s\n\n*Reference: [%s](%s)*", item.Summary, item.Company, item.Link), nil
		}
		return "Summary generation temporarily unavailable.", nil
	}
	return summary, nil
}

// GenerateArticleArchitectureSummary is an alias for GenerateArticleSummary for backwards compatibility
func GenerateArticleArchitectureSummary(db *gorm.DB, item database.NewsItem, fullText string) (string, error) {
	return GenerateArticleSummary(db, item, fullText)
}

// FetchAvailableModels returns a list of supported Gemini & Vertex AI models
func FetchAvailableModels(db *gorm.DB) []string {
	return []string{
		"gemini-3.7-flash",
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}
}
