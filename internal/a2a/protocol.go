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
	"fmt"
	"strings"
	"time"
)

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
	Task      string                 `json:"task,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Prompt    string                 `json:"prompt,omitempty"`
	Query     string                 `json:"query,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Input     string                 `json:"input,omitempty"`
}

// A2AResponse represents the standard JSON-RPC or REST response
type A2AResponse struct {
	JSONRPC string      `json:"jsonrpc,omitempty"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *A2AError   `json:"error,omitempty"`
}

type A2AError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ExtractQuery resolves the user prompt or query regardless of how the payload was shaped
func (r *A2ARequest) ExtractQuery() string {
	if r.Task != "" {
		return r.Task
	}
	if r.Message != "" {
		return r.Message
	}
	if r.Prompt != "" {
		return r.Prompt
	}
	if r.Query != "" {
		return r.Query
	}
	if r.Params != nil {
		if r.Params.Task != "" {
			return r.Params.Task
		}
		if r.Params.Message != "" {
			return r.Params.Message
		}
		if r.Params.Prompt != "" {
			return r.Params.Prompt
		}
		if r.Params.Query != "" {
			return r.Params.Query
		}
		if r.Params.Input != "" {
			return r.Params.Input
		}
		if q, ok := r.Params.Arguments["query"].(string); ok && q != "" {
			return q
		}
		if t, ok := r.Params.Arguments["task"].(string); ok && t != "" {
			return t
		}
	}
	return "Generate today's AI intelligence summary"
}

// HandleA2A processes the request and returns structured output compliant with A2A protocol
func (a *Agent) HandleA2A(ctx context.Context, req *A2ARequest) (*A2AResponse, error) {
	query := req.ExtractQuery()
	taskRes, err := a.ExecuteTask(ctx, query)
	if err != nil {
		return &A2AResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &A2AError{
				Code:    -32603,
				Message: fmt.Sprintf("Agent execution failed: %v", err),
			},
		}, nil
	}

	outputContent := taskRes.Output
	if strings.TrimSpace(outputContent) == "" {
		outputContent = "Execution completed successfully with no additional output."
	}

	resultMap := map[string]interface{}{
		"status":      "COMPLETED",
		"content":     outputContent,
		"text":        outputContent,
		"output":      outputContent,
		"task_name":   taskRes.TaskName,
		"tool_calls":  taskRes.ToolCalls,
		"executed_at": time.Now().UTC().Format(time.RFC3339),
	}

	return &A2AResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultMap,
	}, nil
}
