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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ai-daily-brief/internal/database"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
}

type BidiVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type BidiPrebuiltVoiceConfig struct {
	PrebuiltVoiceConfig BidiVoiceConfig `json:"prebuiltVoiceConfig"`
}

type BidiSpeechConfig struct {
	VoiceConfig BidiPrebuiltVoiceConfig `json:"voiceConfig"`
}

type BidiGenerationConfig struct {
	ResponseModalities []string          `json:"responseModalities"`
	SpeechConfig       *BidiSpeechConfig `json:"speechConfig,omitempty"`
}

type BidiSystemInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

type BidiSetupPayload struct {
	Model             string                 `json:"model"`
	GenerationConfig  *BidiGenerationConfig  `json:"generationConfig,omitempty"`
	SystemInstruction *BidiSystemInstruction `json:"systemInstruction,omitempty"`
}

type BidiSetupMessage struct {
	Setup BidiSetupPayload `json:"setup"`
}

// HandleBidiWebSocket proxies real-time bidirectional audio between the React client and Google Gemini Live API
func HandleBidiWebSocket(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Bidi] Client WebSocket upgrade failed: %v", err)
		return
	}
	defer clientConn.Close()

	articleID := r.URL.Query().Get("article_id")
	voice := r.URL.Query().Get("voice")
	if voice == "" {
		voice = "Aoede" // Default expressive neural voice
	}

	_, authMode, apiKey, projectID, location := GetAgentSettings(db)

	reqModel := r.URL.Query().Get("model")
	if reqModel == "" || reqModel == "gemini-2.0-flash-exp" {
		// Use official native audio live model
		reqModel = "gemini-2.5-flash-native-audio-latest"
	}
	rawModel := strings.TrimPrefix(reqModel, "models/")

	var upstreamURL string
	var setupModel string
	var header http.Header = make(http.Header)

	if authMode == "vertex_adc" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		creds, token, err := getVertexCredentials(ctx)
		cancel()
		if err != nil {
			clientConn.WriteJSON(map[string]interface{}{
				"error": fmt.Sprintf("Vertex AI ADC error: %v", err),
			})
			return
		}
		if projectID == "" && creds.ProjectID != "" {
			projectID = creds.ProjectID
		}
		if projectID == "" {
			clientConn.WriteJSON(map[string]interface{}{
				"error": "Google Cloud Project ID is required for Vertex AI ADC mode. Please enter your GCP Project ID in Agent Settings",
			})
			return
		}
		upstreamURL = fmt.Sprintf("wss://%s-aiplatform.googleapis.com/ws/google.cloud.aiplatform.v1beta1.LlmBidiService/BidiGenerateContent", location)
		setupModel = fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, rawModel)
		header.Set("Authorization", "Bearer "+token)
	} else {
		if apiKey == "" {
			clientConn.WriteJSON(map[string]interface{}{
				"error": "Gemini API key is not configured. Please enter your API key in Agent Settings or switch to Vertex AI (ADC)",
			})
			return
		}
		upstreamURL = fmt.Sprintf("wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent?key=%s", apiKey)
		setupModel = "models/" + rawModel
	}

	log.Printf("[Bidi] Connecting to Gemini Live upstream WebSocket: %s (Voice: %s, Article: %s)", setupModel, voice, articleID)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second
	geminiConn, resp, err := dialer.Dial(upstreamURL, header)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		log.Printf("[Bidi] Upstream Gemini Live connection failed (status %d): %v", status, err)
		clientConn.WriteJSON(map[string]interface{}{
			"error": fmt.Sprintf("Failed to connect to Gemini Live service (%d): %v", status, err),
		})
		return
	}
	defer geminiConn.Close()

	// Build Grounding System Instruction
	systemPrompt := "You are a real-time conversational AI voice agent specializing in AI and Google Cloud intelligence. Answer questions naturally, concisely, and conversationally."

	if articleID != "" && articleID != "newsletter" {
		var item database.NewsItem
		if err := db.First(&item, "id = ?", articleID).Error; err == nil {
			articleText, err := FetchFullArticleText(item.Link)
			if err == nil && len(articleText) > 50 {
				systemPrompt += fmt.Sprintf("\n\n--- GROUNDED ARTICLE CONTEXT ---\nTitle: %s\nURL: %s\nCategory: %s\n\n%s\n--- END OF GROUNDED CONTEXT ---\nPlease use the grounded article text above to answer questions accurately.", item.Title, item.Link, item.Category, articleText)
			} else {
				systemPrompt += fmt.Sprintf("\n\n--- GROUNDED ARTICLE CONTEXT ---\nTitle: %s\nURL: %s\nCategory: %s\nSummary: %s\n--- END OF GROUNDED CONTEXT ---", item.Title, item.Link, item.Category, item.Summary)
			}
		}
	} else {
		// Automatically include current daily newsletter as the base context
		_, nlContent := BuildNewsletterContext(db)
		systemPrompt += fmt.Sprintf("\n\n%s\nPlease use the daily intelligence newsletter base context above to answer user questions about recent models, Google Cloud updates, papers, and AI developments.", nlContent)
	}

	// 1. Send Initial Bidi Setup Message to Gemini Live
	setupMsg := BidiSetupMessage{
		Setup: BidiSetupPayload{
			Model: setupModel,
			GenerationConfig: &BidiGenerationConfig{
				ResponseModalities: []string{"AUDIO"},
				SpeechConfig: &BidiSpeechConfig{
					VoiceConfig: BidiPrebuiltVoiceConfig{
						PrebuiltVoiceConfig: BidiVoiceConfig{
							VoiceName: voice,
						},
					},
				},
			},
			SystemInstruction: &BidiSystemInstruction{
				Parts: []GeminiPart{
					{Text: systemPrompt},
				},
			},
		},
	}

	setupBytes, _ := json.Marshal(setupMsg)
	if err := geminiConn.WriteMessage(websocket.TextMessage, setupBytes); err != nil {
		log.Printf("[Bidi] Error sending setup message to Gemini: %v", err)
		clientConn.WriteJSON(map[string]interface{}{"error": "Failed to initialize Gemini Live session"})
		return
	}

	// Notify client that live session is connected & ready
	clientConn.WriteJSON(map[string]interface{}{
		"connected": true,
		"voice":     voice,
		"model":     setupModel,
		"grounded":  articleID != "",
	})

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Client -> Gemini Live (Audio Chunks & User Messages)
	go func() {
		defer wg.Done()
		for {
			msgType, p, err := clientConn.ReadMessage()
			if err != nil {
				geminiConn.Close()
				break
			}
			if err := geminiConn.WriteMessage(msgType, p); err != nil {
				log.Printf("[Bidi] Upstream Gemini write error: %v", err)
				break
			}
		}
	}()

	// Goroutine 2: Gemini Live -> Client (Live Audio Output & Transcripts)
	go func() {
		defer wg.Done()
		for {
			msgType, p, err := geminiConn.ReadMessage()
			if err != nil {
				log.Printf("[Bidi] Upstream Gemini read error: %v", err)
				clientConn.Close()
				break
			}
			if err := clientConn.WriteMessage(msgType, p); err != nil {
				log.Printf("[Bidi] Client write error: %v", err)
				break
			}
		}
	}()

	wg.Wait()
	log.Printf("[Bidi] Live Voice session ended gracefully")
}
