package llm

import (
	"time"
)

// ProcessingResult represents the result of LLM processing
type ProcessingResult struct {
	Content           string                 `json:"content"`
	TokensUsed        int                    `json:"tokens_used"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	CacheHit          bool                   `json:"cache_hit"`
	Batched           bool                   `json:"batched"`
	Metadata          map[string]interface{} `json:"metadata"`
	Error             error                  `json:"error,omitempty"`
	ProcessingContext *ProcessingContext     `json:"processing_context,omitempty"` // Uses the existing ProcessingContext from processing_pipeline.go
	Success           bool                   `json:"success"`                      // Added for processing_pipeline.go
}

// NOTE: ProcessingContext is already defined in processing_pipeline.go
// NOTE: ClassificationResult is already defined in processing_pipeline.go
// NOTE: EnrichmentContext is already defined in processing_pipeline.go
// NOTE: ValidationResult is already defined in security_validator.go

// Helper functions

// NewProcessingResult creates a new processing result with default values
func NewProcessingResult(content string) *ProcessingResult {
	return &ProcessingResult{
		Content:        content,
		TokensUsed:     0,
		ProcessingTime: time.Duration(0),
		CacheHit:       false,
		Batched:        false,
		Metadata:       make(map[string]interface{}),
		Success:        true,
	}
}

// WithError sets an error on the processing result
func (pr *ProcessingResult) WithError(err error) *ProcessingResult {
	pr.Error = err
	pr.Success = false
	return pr
}

// WithMetadata adds metadata to the processing result
func (pr *ProcessingResult) WithMetadata(key string, value interface{}) *ProcessingResult {
	if pr.Metadata == nil {
		pr.Metadata = make(map[string]interface{})
	}
	pr.Metadata[key] = value
	return pr
}

// WithProcessingContext sets the processing context
func (pr *ProcessingResult) WithProcessingContext(ctx *ProcessingContext) *ProcessingResult {
	pr.ProcessingContext = ctx
	return pr
}

// WithTokenUsage sets the token usage
func (pr *ProcessingResult) WithTokenUsage(tokens int) *ProcessingResult {
	pr.TokensUsed = tokens
	return pr
}

// WithProcessingTime sets the processing time
func (pr *ProcessingResult) WithProcessingTime(duration time.Duration) *ProcessingResult {
	pr.ProcessingTime = duration
	return pr
}

// IsSuccessful returns true if the processing was successful
func (pr *ProcessingResult) IsSuccessful() bool {
	return pr.Success && pr.Error == nil
}

// GetErrorMessage returns the error message if there is one
func (pr *ProcessingResult) GetErrorMessage() string {
	if pr.Error != nil {
		return pr.Error.Error()
	}
	return ""
}
