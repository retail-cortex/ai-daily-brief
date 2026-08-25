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

	"ai-news-agent/internal/config"
	"ai-news-agent/internal/database"
	"ai-news-agent/internal/security"

	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

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

// getVertexAccessToken fetches a GCP OAuth2 token using Application Default Credentials (ADC)
func getVertexAccessToken(ctx context.Context) (string, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("Google Cloud ADC not found: %w (Ensure 'gcloud auth application-default login' was executed)", err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve OAuth2 token from ADC: %w", err)
	}
	return token.AccessToken, nil
}

// GenerateRawContent sends the request to either Google AI Studio or Vertex AI ADC
func GenerateRawContent(db *gorm.DB, systemInstruction, userPrompt string) (string, error) {
	var model, authMode, apiKey, projectID, location string
	if config.AppConfig != nil {
		model = config.AppConfig.Gemini.Model
		authMode = config.AppConfig.Gemini.AuthMode
		apiKey = config.AppConfig.Gemini.APIKey
		projectID = config.AppConfig.Gemini.VertexProjectID
		location = config.AppConfig.Gemini.VertexLocation
	}

	if model == "" {
		model = getSetting(db, "gemini_model", "gemini-3.7-flash")
	}
	if authMode == "" {
		authMode = getSetting(db, "gemini_auth_mode", "api_key")
	}
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		apiKey = envKey
	} else if apiKey == "" {
		apiKey = getSetting(db, "gemini_api_key", "")
	}
	if projectID == "" {
		projectID = getSetting(db, "vertex_project_id", "")
	}
	if location == "" {
		location = getSetting(db, "vertex_location", "us-central1")
	}

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
		if projectID == "" {
			return "", fmt.Errorf("Google Cloud Project ID is required for Vertex AI ADC mode")
		}
		endpoint = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent", location, projectID, location, model)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		token, err := getVertexAccessToken(ctx)
		if err != nil {
			return "", err
		}

		req, err = http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
	} else {
		// API Key Mode
		if apiKey == "" {
			return "", fmt.Errorf("Gemini API key is not configured. Please enter your API key in Agent Settings")
		}
		endpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		req, err = http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
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

// FetchAvailableModels returns a list of supported Gemini 3 & frontier models
func FetchAvailableModels(db *gorm.DB) []string {
	return []string{
		"gemini-3.7-flash",
		"gemini-3.1-pro",
		"gemini-3.5-flash",
		"gemini-3.5-flash-lite",
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}
}
