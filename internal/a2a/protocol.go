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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// A2APart represents a content chunk in A2A (supports TextPart and DataPart)
type A2APart struct {
	Kind      string                 `json:"kind,omitempty"`
	Text      string                 `json:"text,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	MediaType string                 `json:"mediaType,omitempty"`
	MIMEType  string                 `json:"mimeType,omitempty"`
	Raw       string                 `json:"raw,omitempty"`
	URL       string                 `json:"url,omitempty"`
	Filename  string                 `json:"filename,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// A2AArtifact represents an artifact produced by an A2A task
type A2AArtifact struct {
	ArtifactID  string                 `json:"artifactId"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parts       []A2APart              `json:"parts"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// A2AMessage represents a discrete message in A2A
type A2AMessage struct {
	Kind             string                 `json:"kind,omitempty"`
	MessageID        string                 `json:"messageId"`
	ContextID        string                 `json:"contextId"`
	TaskID           string                 `json:"taskId,omitempty"`
	Role             string                 `json:"role"`
	Parts            []A2APart              `json:"parts"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	ReferenceTaskIDs []string               `json:"referenceTaskIds,omitempty"`
}

// A2ATaskStatus represents the status of a task
type A2ATaskStatus struct {
	State     string      `json:"state"`
	Message   *A2AMessage `json:"message,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// A2ATask represents the canonical A2A Task object
type A2ATask struct {
	Kind      string                 `json:"kind,omitempty"`
	ID        string                 `json:"id"`
	ContextID string                 `json:"contextId"`
	Status    A2ATaskStatus          `json:"status"`
	History   []A2AMessage           `json:"history,omitempty"`
	Artifacts []A2AArtifact          `json:"artifacts,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// A2ARequest represents an incoming A2A execution request (standard JSON-RPC or REST payload)
type A2ARequest struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  *A2ARequestBody `json:"params,omitempty"`

	// Direct fields if invoked as flat REST
	Task      string `json:"task,omitempty"`
	Message   string `json:"message,omitempty"`
	Prompt    string `json:"prompt,omitempty"`
	Query     string `json:"query,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// A2ARequestBody holds parameters for A2A methods
type A2ARequestBody struct {
	Tenant        string                 `json:"tenant,omitempty"`
	Message       *A2AMessageInput       `json:"message,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Task          string                 `json:"task,omitempty"`
	Prompt        string                 `json:"prompt,omitempty"`
	Query         string                 `json:"query,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	Arguments     map[string]interface{} `json:"arguments,omitempty"`
	Input         string                 `json:"input,omitempty"`
}

// A2AMessageInput accommodates nested message formats sent by A2A clients
type A2AMessageInput struct {
	ContextID string    `json:"context_id,omitempty"`
	ContextId string    `json:"contextId,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	TaskId    string    `json:"taskId,omitempty"`
	Role      string    `json:"role,omitempty"`
	Parts     []A2APart `json:"parts,omitempty"`
}

// A2AResponse represents the standard JSON-RPC 2.0 response wrapper
type A2AResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *A2AError   `json:"error,omitempty"`
}

type A2AError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func generateID(prefix string) string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bytes))
}

// ExtractQuery resolves the user prompt or query regardless of how the payload was shaped
func (r *A2ARequest) ExtractQuery() (string, string) {
	contextID := ""

	if r.Params != nil && r.Params.Message != nil {
		if r.Params.Message.ContextId != "" {
			contextID = r.Params.Message.ContextId
		} else if r.Params.Message.ContextID != "" {
			contextID = r.Params.Message.ContextID
		}

		for _, p := range r.Params.Message.Parts {
			if strings.TrimSpace(p.Text) != "" {
				return p.Text, contextID
			}
			// Handle A2UI client action DataParts (e.g. from Gemini Enterprise button clicks)
			if p.Data != nil {
				if dataMap, ok := p.Data.(map[string]interface{}); ok {
					if act, ok := dataMap["action"].(map[string]interface{}); ok {
						if ctxMap, ok := act["context"].(map[string]interface{}); ok {
							if prompt, ok := ctxMap["prompt"].(string); ok && prompt != "" {
								return prompt, contextID
							}
							if articleID, ok := ctxMap["article_id"].(string); ok && articleID != "" {
								return fmt.Sprintf("Load context for article %s", articleID), contextID
							}
						}
						if name, ok := act["name"].(string); ok && name != "" {
							return name, contextID
						}
					}
					if prompt, ok := dataMap["prompt"].(string); ok && prompt != "" {
						return prompt, contextID
					}
				}
			}
		}
	}

	if r.Task != "" {
		return r.Task, contextID
	}
	if r.Message != "" {
		return r.Message, contextID
	}
	if r.Prompt != "" {
		return r.Prompt, contextID
	}
	if r.Query != "" {
		return r.Query, contextID
	}
	if r.Params != nil {
		if r.Params.Task != "" {
			return r.Params.Task, contextID
		}
		if r.Params.Prompt != "" {
			return r.Params.Prompt, contextID
		}
		if r.Params.Query != "" {
			return r.Params.Query, contextID
		}
		if r.Params.Input != "" {
			return r.Params.Input, contextID
		}
		if q, ok := r.Params.Arguments["query"].(string); ok && q != "" {
			return q, contextID
		}
		if t, ok := r.Params.Arguments["task"].(string); ok && t != "" {
			return t, contextID
		}
	}
	return "Generate today's AI intelligence summary", contextID
}

// BuildTaskResponse constructs a schema-compliant A2ATask result
func (a *Agent) BuildTaskResponse(ctx context.Context, req *A2ARequest) (*A2ATask, error) {
	query, ctxID := req.ExtractQuery()
	if ctxID == "" {
		ctxID = generateID("ctx")
	}
	taskID := generateID("task")
	msgID := generateID("msg")

	taskRes, err := a.ExecuteTask(ctx, query)
	if err != nil {
		return nil, err
	}

	// 1. Text part with direct Markdown (never wrapped in code block fences)
	outputContent := strings.TrimSpace(taskRes.Output)
	if outputContent == "" {
		outputContent = "Execution completed successfully with no additional output."
	}

	parts := []A2APart{
		{
			Kind:      "text",
			Text:      outputContent,
			MediaType: "text/markdown",
			MIMEType:  "text/markdown",
		},
	}

	// 2. Structured A2UI instructions DataPart with application/json+a2ui media type
	var artifacts []A2AArtifact
	if len(taskRes.A2UIInstructions) > 0 {
		for _, inst := range taskRes.A2UIInstructions {
			dataPart := A2APart{
				Kind: "data",
				Metadata: map[string]interface{}{
					"mimeType": "application/json+a2ui",
				},
				Data: inst,
			}
			parts = append(parts, dataPart)
		}
		artifacts = append(artifacts, A2AArtifact{
			ArtifactID:  generateID("art"),
			Name:        "a2ui_components",
			Description: "Visual A2UI v0.9 component deck for Gemini Enterprise",
			Parts: []A2APart{
				{
					Kind: "data",
					Metadata: map[string]interface{}{
						"mimeType": "application/json+a2ui",
					},
					Data: taskRes.A2UIInstructions,
				},
			},
		})
	} else if len(taskRes.A2UIPayload) > 0 {
		dataPart := A2APart{
			Kind: "data",
			Metadata: map[string]interface{}{
				"mimeType": "application/json+a2ui",
			},
			Data: taskRes.A2UIPayload,
		}
		parts = append(parts, dataPart)

		artifacts = append(artifacts, A2AArtifact{
			ArtifactID:  generateID("art"),
			Name:        "a2ui_components",
			Description: "Visual A2UI v0.9 component deck for Gemini Enterprise",
			Parts:       []A2APart{dataPart},
		})
	}

	task := &A2ATask{
		Kind:      "task",
		ID:        taskID,
		ContextID: ctxID,
		Status: A2ATaskStatus{
			State: "completed",
			Message: &A2AMessage{
				Kind:      "message",
				MessageID: msgID,
				ContextID: ctxID,
				TaskID:    taskID,
				Role:      "agent",
				Parts:     parts,
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Artifacts: artifacts,
		Metadata: map[string]interface{}{
			"task_name":  taskRes.TaskName,
			"tool_calls": taskRes.ToolCalls,
		},
	}

	return task, nil
}

// BuildMessageResponse constructs a schema-compliant A2AMessage result for conversational turns
func (a *Agent) BuildMessageResponse(ctx context.Context, req *A2ARequest) (*A2AMessage, error) {
	query, ctxID := req.ExtractQuery()
	if ctxID == "" {
		ctxID = generateID("ctx")
	}
	msgID := generateID("msg")

	taskRes, err := a.ExecuteTask(ctx, query)
	if err != nil {
		return nil, err
	}

	// 1. Text part with direct text
	outputContent := strings.TrimSpace(taskRes.Output)
	if outputContent == "" {
		outputContent = "Execution completed successfully with no additional output."
	}

	parts := []A2APart{
		{
			Kind: "text",
			Text: outputContent,
		},
	}

	// 2. Structured A2UI payload as DataPart (application/json+a2ui)
	if len(taskRes.A2UIInstructions) > 0 {
		for _, inst := range taskRes.A2UIInstructions {
			dataPart := A2APart{
				Kind: "data",
				Metadata: map[string]interface{}{
					"mimeType": "application/json+a2ui",
				},
				Data: inst,
			}
			parts = append(parts, dataPart)
		}
	} else if len(taskRes.A2UIPayload) > 0 {
		dataPart := A2APart{
			Kind: "data",
			Metadata: map[string]interface{}{
				"mimeType": "application/json+a2ui",
			},
			Data: taskRes.A2UIPayload,
		}
		parts = append(parts, dataPart)
	}

	message := &A2AMessage{
		Kind:      "message",
		MessageID: msgID,
		ContextID: ctxID,
		Role:      "agent",
		Parts:     parts,
		Metadata: map[string]interface{}{
			"task_name":  taskRes.TaskName,
			"tool_calls": taskRes.ToolCalls,
		},
	}

	return message, nil
}

// HandleA2A processes the request and returns structured output compliant with A2A protocol
func (a *Agent) HandleA2A(ctx context.Context, req *A2ARequest) (*A2AResponse, error) {
	reqID := req.ID
	if reqID == nil {
		reqID = 1
	}

	if req.Method == "tasks/get" || req.Method == "tasks/cancel" {
		task, err := a.BuildTaskResponse(ctx, req)
		if err != nil {
			return &A2AResponse{
				JSONRPC: "2.0",
				ID:      reqID,
				Error: &A2AError{
					Code:    -32603,
					Message: fmt.Sprintf("Agent execution failed: %v", err),
				},
			}, nil
		}
		return &A2AResponse{
			JSONRPC: "2.0",
			ID:      reqID,
			Result:  task,
		}, nil
	}

	// Standard message/send or conversational turn returns A2AMessage
	msg, err := a.BuildMessageResponse(ctx, req)
	if err != nil {
		return &A2AResponse{
			JSONRPC: "2.0",
			ID:      reqID,
			Error: &A2AError{
				Code:    -32603,
				Message: fmt.Sprintf("Agent execution failed: %v", err),
			},
		}, nil
	}

	return &A2AResponse{
		JSONRPC: "2.0",
		ID:      reqID,
		Result:  msg,
	}, nil
}
