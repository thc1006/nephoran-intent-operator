package llm

import (
	"context"
	"encoding/json"
	"time"
)

//go:generate mockgen -source=client.go -destination=mock_llm_client.go -package=llm

// LLMClient defines the interface for LLM operations
type LLMClient interface {
	ProcessIntent(ctx context.Context, request *ProcessIntentRequest) (*ProcessIntentResponse, error)
	GenerateNetworkConfig(ctx context.Context, request *NetworkConfigRequest) (*NetworkConfigResponse, error)
	ValidateIntent(ctx context.Context, request *ValidateIntentRequest) (*ValidateIntentResponse, error)
}

// ProcessIntentRequest represents a request to process natural language intent
type ProcessIntentRequest struct {
	Intent      string            `json:"intent"`
	Context     map[string]string `json:"context,omitempty"`
	Metadata    RequestMetadata   `json:"metadata"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ProcessIntentResponse represents the processed intent response
type ProcessIntentResponse struct {
	StructuredIntent map[string]interface{} `json:"structured_intent"`
	Confidence      float64                `json:"confidence"`
	Reasoning       string                 `json:"reasoning,omitempty"`
	Metadata        ResponseMetadata       `json:"metadata"`
	Timestamp       time.Time              `json:"timestamp"`
}

// NetworkConfigRequest represents a request to generate network configuration
type NetworkConfigRequest struct {
	Intent          string            `json:"intent"`
	NetworkType     string            `json:"network_type"`
	Requirements    map[string]string `json:"requirements,omitempty"`
	Constraints     []string          `json:"constraints,omitempty"`
	Metadata        RequestMetadata   `json:"metadata"`
	Timestamp       time.Time         `json:"timestamp"`
}

// NetworkConfigResponse represents the generated network configuration
type NetworkConfigResponse struct {
	Configuration   json.RawMessage  `json:"configuration"`
	ConfigType      string           `json:"config_type"`
	ValidationRules []string         `json:"validation_rules,omitempty"`
	Metadata        ResponseMetadata `json:"metadata"`
	Timestamp       time.Time        `json:"timestamp"`
}

// ValidateIntentRequest represents a request to validate an intent
type ValidateIntentRequest struct {
	Intent      string            `json:"intent"`
	Context     map[string]string `json:"context,omitempty"`
	Schema      json.RawMessage   `json:"schema,omitempty"`
	Metadata    RequestMetadata   `json:"metadata"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ValidateIntentResponse represents the intent validation response
type ValidateIntentResponse struct {
	IsValid     bool             `json:"is_valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	Suggestions []string         `json:"suggestions,omitempty"`
	Metadata    ResponseMetadata `json:"metadata"`
	Timestamp   time.Time        `json:"timestamp"`
}

// RequestMetadata contains metadata for LLM requests
type RequestMetadata struct {
	RequestID   string            `json:"request_id"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Source      string            `json:"source"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// ResponseMetadata contains metadata for LLM responses
type ResponseMetadata struct {
	RequestID    string  `json:"request_id"`
	ProcessingTime float64 `json:"processing_time_ms"`
	ModelUsed    string  `json:"model_used"`
	TokensUsed   int     `json:"tokens_used,omitempty"`
	Cost         float64 `json:"cost,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}