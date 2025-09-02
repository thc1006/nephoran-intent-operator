/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"time"
)

// TokenUsage represents token consumption details
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMConfig holds configuration for LLM services
type LLMConfig struct {
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
	APIKey      string            `json:"api_key"`
	Endpoint    string            `json:"endpoint"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float32           `json:"temperature"`
	TopP        float32           `json:"top_p"`
	Timeout     time.Duration     `json:"timeout"`
	Headers     map[string]string `json:"headers"`
}

// LLMResponse represents a response from an LLM service
type LLMResponse struct {
	Content      string        `json:"content"`
	Model        string        `json:"model"`
	Usage        TokenUsage    `json:"usage"`
	FinishReason string        `json:"finish_reason"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	RequestID    string        `json:"request_id"`
}

// LLMRequest represents a request to an LLM service
type LLMRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float32                `json:"temperature"`
	TopP        float32                `json:"top_p"`
	Stream      bool                   `json:"stream"`
	Stop        []string               `json:"stop"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}
