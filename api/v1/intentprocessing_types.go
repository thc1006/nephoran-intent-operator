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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IntentProcessingSpec defines the desired state of IntentProcessing
type IntentProcessingSpec struct {
	// Intent is the natural language intent to process
	Intent string `json:"intent"`

	// LLMConfig configuration for LLM processing
	// +optional
	LLMConfig *LLMProcessingConfig `json:"llmConfig,omitempty"`

	// RAGConfig configuration for RAG processing
	// +optional
	RAGConfig *RAGProcessingConfig `json:"ragConfig,omitempty"`

	// ProcessingTimeout timeout for processing in seconds
	// +optional
	// +kubebuilder:default=300
	ProcessingTimeout int32 `json:"processingTimeout,omitempty"`
}

// IntentProcessingStatus defines the observed state of IntentProcessing
type IntentProcessingStatus struct {
	// Phase current processing phase
	// +kubebuilder:validation:Enum=Pending;Processing;Completed;Failed
	Phase string `json:"phase,omitempty"`

	// ProcessedIntent structured intent result
	// +optional
	ProcessedIntent *ProcessedParameters `json:"processedIntent,omitempty"`

	// Metrics processing metrics
	// +optional
	Metrics *ProcessingMetrics `json:"metrics,omitempty"`

	// Error processing error if any
	// +optional
	Error string `json:"error,omitempty"`

	// LastUpdated timestamp of last update
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// LLMProcessingConfig defines LLM processing configuration
type LLMProcessingConfig struct {
	// Model LLM model to use
	// +kubebuilder:default="gpt-4"
	Model string `json:"model,omitempty"`

	// Temperature for text generation
	// +optional
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=2.0
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens maximum tokens in response
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=8192
	MaxTokens *int32 `json:"maxTokens,omitempty"`

	// SystemPrompt custom system prompt
	// +optional
	SystemPrompt string `json:"systemPrompt,omitempty"`
}

// RAGProcessingConfig defines RAG processing configuration
type RAGProcessingConfig struct {
	// Enabled whether RAG is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Sources list of knowledge sources
	// +optional
	Sources []string `json:"sources,omitempty"`

	// MaxRetrievalResults maximum number of retrieved results
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=50
	MaxRetrievalResults *int32 `json:"maxRetrievalResults,omitempty"`

	// SimilarityThreshold threshold for similarity matching
	// +optional
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	SimilarityThreshold *float64 `json:"similarityThreshold,omitempty"`
}

// ProcessingMetrics contains metrics for intent processing
type ProcessingMetrics struct {
	// StartTime when processing started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// EndTime when processing completed
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// DurationMs processing duration in milliseconds
	DurationMs int64 `json:"durationMs,omitempty"`

	// LLMMetrics metrics for LLM processing
	// +optional
	LLMMetrics *LLMMetrics `json:"llmMetrics,omitempty"`

	// RAGMetrics metrics for RAG processing
	// +optional
	RAGMetrics *RAGMetrics `json:"ragMetrics,omitempty"`
}

// LLMMetrics contains metrics for LLM processing
type LLMMetrics struct {
	// TokenUsage token usage information
	TokenUsage *TokenUsageInfo `json:"tokenUsage,omitempty"`

	// ResponseTimeMs LLM response time in milliseconds
	ResponseTimeMs int64 `json:"responseTimeMs,omitempty"`

	// Model the LLM model used
	Model string `json:"model,omitempty"`

	// ConfidenceScore confidence in the response
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	ConfidenceScore float64 `json:"confidenceScore,omitempty"`
}

// RAGMetrics contains metrics for RAG processing
type RAGMetrics struct {
	// RetrievalTimeMs time spent on retrieval in milliseconds
	RetrievalTimeMs int64 `json:"retrievalTimeMs,omitempty"`

	// DocumentsRetrieved number of documents retrieved
	DocumentsRetrieved int32 `json:"documentsRetrieved,omitempty"`

	// AverageRelevanceScore average relevance score of retrieved documents
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	AverageRelevanceScore float64 `json:"averageRelevanceScore,omitempty"`

	// SourcesUsed list of knowledge sources used
	SourcesUsed []string `json:"sourcesUsed,omitempty"`

	// IndexHits number of index hits
	IndexHits int64 `json:"indexHits,omitempty"`

	// CacheHits number of cache hits
	CacheHits int64 `json:"cacheHits,omitempty"`
}

// TokenUsageInfo contains token usage information
type TokenUsageInfo struct {
	// InputTokens number of input tokens
	InputTokens int32 `json:"inputTokens,omitempty"`

	// OutputTokens number of output tokens
	OutputTokens int32 `json:"outputTokens,omitempty"`

	// TotalTokens total number of tokens
	TotalTokens int32 `json:"totalTokens,omitempty"`

	// Cost estimated cost in USD
	// +optional
	Cost *float64 `json:"cost,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ip
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Intent",type=string,JSONPath=`.spec.intent`
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=`.status.metrics.durationMs`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// IntentProcessing represents an intent processing request
type IntentProcessing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntentProcessingSpec   `json:"spec,omitempty"`
	Status IntentProcessingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntentProcessingList contains a list of IntentProcessing
type IntentProcessingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntentProcessing `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IntentProcessing{}, &IntentProcessingList{})
}