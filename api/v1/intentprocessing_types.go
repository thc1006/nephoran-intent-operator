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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Intent processing phase constants
const (
	IntentProcessingPhasePending    = "Pending"
	IntentProcessingPhaseProcessing = "Processing"
	IntentProcessingPhaseInProgress = "InProgress"
	IntentProcessingPhaseCompleted  = "Completed"
	IntentProcessingPhaseFailed     = "Failed"
	IntentProcessingPhaseRetrying   = "Retrying"
)

// IntentProcessingSpec defines the desired state of IntentProcessing
type IntentProcessingSpec struct {
	// Intent is the natural language intent to process
	Intent string `json:"intent"`

	// ParentIntentRef references the parent NetworkIntent
	// +optional
	ParentIntentRef *ObjectReference `json:"parentIntentRef,omitempty"`

	// OriginalIntent stores the original intent text
	// +optional
	OriginalIntent string `json:"originalIntent,omitempty"`

	// Priority processing priority
	// +optional
	// +kubebuilder:default="Medium"
	Priority Priority `json:"priority,omitempty"`

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

	// MaxRetries maximum number of retries
	// +optional
	// +kubebuilder:default=3
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// ProcessingConfiguration additional processing configuration
	// +optional
	ProcessingConfiguration *ProcessingConfig `json:"processingConfiguration,omitempty"`
}

// IntentProcessingStatus defines the observed state of IntentProcessing
type IntentProcessingStatus struct {
	// Phase current processing phase
	// +kubebuilder:validation:Enum=Pending;Processing;Completed;Failed
	Phase string `json:"phase,omitempty"`

	// ProcessedIntent structured intent result
	// +optional
	ProcessedIntent *ProcessedParameters `json:"processedIntent,omitempty"`

	// LLMResponse contains the raw LLM response
	// +optional
	LLMResponse *runtime.RawExtension `json:"llmResponse,omitempty"`

	// ProcessedParameters contains the processed parameters from LLM
	// +optional
	ProcessedParameters *ProcessedParameters `json:"processedParameters,omitempty"`

	// ExtractedEntities contains entities extracted from the intent
	// +optional
	ExtractedEntities map[string]runtime.RawExtension `json:"extractedEntities,omitempty"`

	// QualityScore indicates the quality of processing (as string to avoid float issues)
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.\d+)?|1(\.0+)?)$`
	QualityScore *string `json:"qualityScore,omitempty"`

	// Metrics processing metrics
	// +optional
	Metrics *ProcessingMetrics `json:"metrics,omitempty"`

	// Error processing error if any
	// +optional
	Error string `json:"error,omitempty"`

	// LastUpdated timestamp of last update
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// ProcessingStartTime when processing started
	// +optional
	ProcessingStartTime *metav1.Time `json:"processingStartTime,omitempty"`

	// ProcessingCompletionTime when processing completed
	// +optional
	ProcessingCompletionTime *metav1.Time `json:"processingCompletionTime,omitempty"`

	// ValidationErrors validation errors if any
	// +optional
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// TokenUsage token usage information
	// +optional
	TokenUsage *TokenUsageInfo `json:"tokenUsage,omitempty"`

	// RAGMetrics RAG processing metrics
	// +optional
	RAGMetrics *RAGMetrics `json:"ragMetrics,omitempty"`

	// TelecomContext telecom-specific context
	// +optional
	TelecomContext map[string]string `json:"telecomContext,omitempty"`

	// ProcessingDuration total processing duration
	// +optional
	ProcessingDuration *metav1.Duration `json:"processingDuration,omitempty"`

	// RetryCount number of retries attempted
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`

	// LastRetryTime timestamp of last retry
	// +optional
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`

	// Conditions represents the current conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// LLMProcessingConfig defines LLM processing configuration
type LLMProcessingConfig struct {
	// Model LLM model to use
	// +kubebuilder:default="gpt-4"
	Model string `json:"model,omitempty"`

	// Temperature for text generation (as string to avoid float issues)
	// +optional
	// +kubebuilder:validation:Pattern=`^([01](\.[0-9]+)?|2(\.0+)?)$`
	Temperature *string `json:"temperature,omitempty"`

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

	// SimilarityThreshold threshold for similarity matching (as string to avoid float issues)
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.\d+)?|1(\.0+)?)$`
	SimilarityThreshold *string `json:"similarityThreshold,omitempty"`

	// MaxDocuments maximum number of documents to retrieve
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	MaxDocuments *int32 `json:"maxDocuments,omitempty"`

	// RetrievalThreshold threshold for retrieval (as string to avoid float issues)
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.\d+)?|1(\.0+)?)$`
	RetrievalThreshold *string `json:"retrievalThreshold,omitempty"`
}

// ProcessingConfig defines general processing configuration
type ProcessingConfig struct {
	// Provider specifies the processing provider
	// +optional
	Provider string `json:"provider,omitempty"`

	// Model specifies the model to use
	// +optional
	Model string `json:"model,omitempty"`

	// Temperature for text generation (as string to avoid float issues)
	// +optional
	// +kubebuilder:validation:Pattern=`^([01](\.[0-9]+)?|2(\.0+)?)$`
	Temperature *string `json:"temperature,omitempty"`

	// MaxTokens maximum tokens in response
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=8192
	MaxTokens *int32 `json:"maxTokens,omitempty"`

	// RAGConfiguration for RAG-specific settings
	// +optional
	RAGConfiguration *RAGProcessingConfig `json:"ragConfiguration,omitempty"`
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

<<<<<<< HEAD
	// ConfidenceScore confidence in the response (0.0-1.0)
	// +optional
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	ConfidenceScore *float64 `json:"confidenceScore,omitempty"`
=======
	// ConfidenceScore confidence in the response (0.0-1.0) as string to avoid float issues
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`
	ConfidenceScore *string `json:"confidenceScore,omitempty"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

// RAGMetrics contains metrics for RAG processing
type RAGMetrics struct {
	// RetrievalTimeMs time spent on retrieval in milliseconds
	RetrievalTimeMs int64 `json:"retrievalTimeMs,omitempty"`

	// RetrievalDuration duration of retrieval process
	// +optional
	RetrievalDuration int64 `json:"retrievalDuration,omitempty"`

	// DocumentsRetrieved number of documents retrieved
	DocumentsRetrieved int32 `json:"documentsRetrieved,omitempty"`

<<<<<<< HEAD
	// AverageRelevanceScore average relevance score of retrieved documents (0.0-1.0)
	// +optional
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	AverageRelevanceScore *float64 `json:"averageRelevanceScore,omitempty"`

	// TopRelevanceScore highest relevance score (0.0-1.0)
	// +optional
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	TopRelevanceScore *float64 `json:"topRelevanceScore,omitempty"`
=======
	// AverageRelevanceScore average relevance score of retrieved documents (0.0-1.0) as string to avoid float issues
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`
	AverageRelevanceScore *string `json:"averageRelevanceScore,omitempty"`

	// TopRelevanceScore highest relevance score (0.0-1.0) as string to avoid float issues
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`
	TopRelevanceScore *string `json:"topRelevanceScore,omitempty"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// SourcesUsed list of knowledge sources used
	SourcesUsed []string `json:"sourcesUsed,omitempty"`

	// IndexHits number of index hits
	IndexHits int64 `json:"indexHits,omitempty"`

	// CacheHits number of cache hits
	CacheHits int64 `json:"cacheHits,omitempty"`

	// QueryEnhancement information about query enhancement
	// +optional
	QueryEnhancement string `json:"queryEnhancement,omitempty"`
}

// TokenUsageInfo contains token usage information
type TokenUsageInfo struct {
	// InputTokens number of input tokens
	InputTokens int32 `json:"inputTokens,omitempty"`

	// OutputTokens number of output tokens
	OutputTokens int32 `json:"outputTokens,omitempty"`

	// TotalTokens total number of tokens
	TotalTokens int32 `json:"totalTokens,omitempty"`

	// Cost estimated cost in USD (as string to avoid float issues)
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+(\.\d{1,4})?$`
	Cost *string `json:"cost,omitempty"`
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

// IsProcessingComplete checks if the intent processing is complete
func (ip *IntentProcessing) IsProcessingComplete() bool {
	return ip.Status.Phase == "Completed"
}

// IsProcessingFailed checks if the intent processing has failed
func (ip *IntentProcessing) IsProcessingFailed() bool {
	return ip.Status.Phase == "Failed"
}

// CanRetry determines if the intent processing can be retried
func (ip *IntentProcessing) CanRetry() bool {
	// Don't retry if already completed
	if ip.IsProcessingComplete() {
		return false
	}

	// Check if we have retry count annotation
	retryCountStr, exists := ip.Annotations["nephoran.com/retry-count"]
	if !exists {
		return true // First attempt, can retry
	}

	// Parse retry count and check against max retries (default 3)
	var retryCount int
	if _, err := fmt.Sscanf(retryCountStr, "%d", &retryCount); err != nil {
		return true // If we can't parse, assume first attempt
	}

	maxRetries := 3 // Default max retries
	if maxRetriesStr, exists := ip.Annotations["nephoran.com/max-retries"]; exists {
		if mr, err := fmt.Sscanf(maxRetriesStr, "%d", &maxRetries); err == nil && mr == 1 {
			// Use custom max retries
		}
	}

	return retryCount < maxRetries
}

// GetProcessingTimeout returns the processing timeout with a default value
func (ip *IntentProcessing) GetProcessingTimeout() time.Duration {
	if ip.Spec.ProcessingTimeout <= 0 {
		return 300 * time.Second // Default 5 minutes
	}
	return time.Duration(ip.Spec.ProcessingTimeout) * time.Second
}

// ShouldEnableRAG determines if RAG should be enabled for this processing
func (ip *IntentProcessing) ShouldEnableRAG() bool {
	if ip.Spec.RAGConfig != nil && ip.Spec.RAGConfig.Enabled {
		return true
	}
	if ip.Spec.ProcessingConfiguration != nil && ip.Spec.ProcessingConfiguration.RAGConfiguration != nil {
		return ip.Spec.ProcessingConfiguration.RAGConfiguration.Enabled
	}
	return false
}
