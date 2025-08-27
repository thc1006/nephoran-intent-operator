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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ProcessedParameters defines structured parameters extracted from intent processing
type ProcessedParameters struct {
	// NetworkFunction specifies the network function type
	// +optional
	NetworkFunction string `json:"networkFunction,omitempty"`

	// Region specifies the deployment region
	// +optional
	Region string `json:"region,omitempty"`

	// ScaleParameters contains scaling configuration
	// +optional
	ScaleParameters *ScaleParameters `json:"scaleParameters,omitempty"`

	// QoSParameters contains quality of service settings
	// +optional
	QoSParameters *QoSParameters `json:"qosParameters,omitempty"`

	// SecurityParameters contains security settings
	// +optional
	SecurityParameters *SecurityParameters `json:"securityParameters,omitempty"`

	// CustomParameters contains additional custom parameters
	// +optional
	CustomParameters map[string]string `json:"customParameters,omitempty"`
}

// ScaleParameters defines scaling configuration
type ScaleParameters struct {
	// MinReplicas specifies the minimum number of replicas
	// +optional
	// +kubebuilder:validation:Minimum=1
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas specifies the maximum number of replicas
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// TargetCPUUtilization specifies the target CPU utilization percentage
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	TargetCPUUtilization *int32 `json:"targetCPUUtilization,omitempty"`

	// AutoScalingEnabled indicates if auto-scaling is enabled
	// +optional
	AutoScalingEnabled *bool `json:"autoScalingEnabled,omitempty"`
}

// QoSParameters defines quality of service parameters
type QoSParameters struct {
	// Latency specifies the target latency requirement
	// +optional
	Latency string `json:"latency,omitempty"`

	// Bandwidth specifies the bandwidth requirement
	// +optional
	Bandwidth string `json:"bandwidth,omitempty"`

	// Priority specifies the traffic priority
	// +optional
	// +kubebuilder:validation:Enum=low;medium;high;critical
	Priority string `json:"priority,omitempty"`

	// ServiceLevel specifies the service level agreement
	// +optional
	ServiceLevel string `json:"serviceLevel,omitempty"`
}

// SecurityParameters defines security configuration
type SecurityParameters struct {
	// Encryption specifies encryption requirements
	// +optional
	Encryption *EncryptionConfig `json:"encryption,omitempty"`

	// NetworkPolicies specifies network security policies
	// +optional
	NetworkPolicies []string `json:"networkPolicies,omitempty"`

	// ServiceMesh indicates if service mesh should be enabled
	// +optional
	ServiceMesh *bool `json:"serviceMesh,omitempty"`

	// TLSEnabled indicates if TLS should be enabled
	// +optional
	TLSEnabled *bool `json:"tlsEnabled,omitempty"`
}

// EncryptionConfig defines encryption configuration
type EncryptionConfig struct {
	// Enabled indicates if encryption is enabled
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Algorithm specifies the encryption algorithm
	// +optional
	// +kubebuilder:validation:Enum=AES-256;AES-128;RSA-2048;RSA-4096
	Algorithm string `json:"algorithm,omitempty"`

	// KeyRotationInterval specifies the key rotation interval
	// +optional
	KeyRotationInterval string `json:"keyRotationInterval,omitempty"`
}

// IntentProcessingPhase represents the phase of LLM processing
type IntentProcessingPhase string

const (
	// IntentProcessingPhasePending indicates the processing is pending
	IntentProcessingPhasePending IntentProcessingPhase = "Pending"
	// IntentProcessingPhaseInProgress indicates the processing is in progress
	IntentProcessingPhaseInProgress IntentProcessingPhase = "InProgress"
	// IntentProcessingPhaseCompleted indicates the processing is completed
	IntentProcessingPhaseCompleted IntentProcessingPhase = "Completed"
	// IntentProcessingPhaseFailed indicates the processing has failed
	IntentProcessingPhaseFailed IntentProcessingPhase = "Failed"
	// IntentProcessingPhaseRetrying indicates the processing is retrying
	IntentProcessingPhaseRetrying IntentProcessingPhase = "Retrying"
)

// LLMProvider represents the LLM provider used for processing
type LLMProvider string

const (
	// LLMProviderOpenAI represents OpenAI GPT models
	LLMProviderOpenAI LLMProvider = "openai"
	// LLMProviderMistral represents Mistral models
	LLMProviderMistral LLMProvider = "mistral"
	// LLMProviderClaude represents Anthropic Claude models
	LLMProviderClaude LLMProvider = "claude"
	// LLMProviderLocal represents local models
	LLMProviderLocal LLMProvider = "local"
)

// IntentProcessingSpec defines the desired state of IntentProcessing
type IntentProcessingSpec struct {
	// ParentIntentRef references the parent NetworkIntent
	// +kubebuilder:validation:Required
	ParentIntentRef ObjectReference `json:"parentIntentRef"`

	// OriginalIntent contains the raw natural language intent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=10
	// +kubebuilder:validation:MaxLength=10000
	OriginalIntent string `json:"originalIntent"`

	// ProcessingConfiguration contains LLM processing configuration
	// +optional
	ProcessingConfiguration *LLMProcessingConfig `json:"processingConfiguration,omitempty"`

	// Priority defines processing priority
	// +optional
	// +kubebuilder:default="medium"
	Priority Priority `json:"priority,omitempty"`

	// Timeout for processing in seconds
	// +optional
	// +kubebuilder:default=120
	// +kubebuilder:validation:Minimum=30
	// +kubebuilder:validation:Maximum=1800
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// MaxRetries defines maximum retry attempts
	// +optional
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	MaxRetries *int32 `json:"maxRetries,omitempty"`

	// ContextEnrichment enables RAG-based context enrichment
	// +optional
	// +kubebuilder:default=true
	ContextEnrichment *bool `json:"contextEnrichment,omitempty"`

	// OutputFormat specifies the expected output format
	// +optional
	// +kubebuilder:default="structured"
	// +kubebuilder:validation:Enum=structured;json;yaml
	OutputFormat string `json:"outputFormat,omitempty"`
}

// LLMProcessingConfig contains configuration for LLM processing
type LLMProcessingConfig struct {
	// Provider specifies the LLM provider to use
	// +optional
	// +kubebuilder:default="openai"
	Provider LLMProvider `json:"provider,omitempty"`

	// Model specifies the model to use within the provider
	// +optional
	// +kubebuilder:default="gpt-4o-mini"
	Model string `json:"model,omitempty"`

	// Temperature controls randomness in generation
	// +optional
	// +kubebuilder:default=0.1
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens limits the response length
	// +optional
	// +kubebuilder:default=2048
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=8192
	MaxTokens *int32 `json:"maxTokens,omitempty"`

	// SystemPrompt provides system-level instructions
	// +optional
	SystemPrompt string `json:"systemPrompt,omitempty"`

	// ContextWindow defines the context window size
	// +optional
	// +kubebuilder:default=8192
	ContextWindow *int32 `json:"contextWindow,omitempty"`

	// RAGConfiguration contains RAG-specific settings
	// +optional
	RAGConfiguration *RAGConfig `json:"ragConfiguration,omitempty"`
}

// RAGConfig contains RAG system configuration
type RAGConfig struct {
	// Enabled determines if RAG is enabled
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// MaxDocuments limits the number of retrieved documents
	// +optional
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=20
	MaxDocuments *int32 `json:"maxDocuments,omitempty"`

	// RetrievalThreshold sets the minimum similarity threshold
	// +optional
	// +kubebuilder:default=0.7
	RetrievalThreshold *float64 `json:"retrievalThreshold,omitempty"`

	// KnowledgeBase specifies the knowledge base to use
	// +optional
	// +kubebuilder:default="telecom-default"
	KnowledgeBase string `json:"knowledgeBase,omitempty"`

	// EmbeddingModel specifies the embedding model for RAG
	// +optional
	// +kubebuilder:default="text-embedding-3-large"
	EmbeddingModel string `json:"embeddingModel,omitempty"`
}

// IntentProcessingStatus defines the observed state of IntentProcessing
type IntentProcessingStatus struct {
	// Phase represents the current processing phase
	// +optional
	Phase IntentProcessingPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProcessingStartTime indicates when processing started
	// +optional
	ProcessingStartTime *metav1.Time `json:"processingStartTime,omitempty"`

	// ProcessingCompletionTime indicates when processing completed
	// +optional
	ProcessingCompletionTime *metav1.Time `json:"processingCompletionTime,omitempty"`

	// LLMResponse contains the processed response from the LLM
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	LLMResponse runtime.RawExtension `json:"llmResponse,omitempty"`

	// ProcessedParameters contains structured parameters
	// +optional
	ProcessedParameters *ProcessedParameters `json:"processedParameters,omitempty"`

	// ExtractedEntities contains telecommunications entities
	// +optional
	ExtractedEntities map[string]string `json:"extractedEntities,omitempty"`

	// TelecomContext contains domain-specific context
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	TelecomContext runtime.RawExtension `json:"telecomContext,omitempty"`

	// RetryCount tracks the number of retry attempts
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`

	// LastRetryTime indicates the last retry attempt
	// +optional
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`

	// ProcessingDuration represents total processing time
	// +optional
	ProcessingDuration *metav1.Duration `json:"processingDuration,omitempty"`

	// TokenUsage tracks token consumption
	// +optional
	TokenUsage *TokenUsageInfo `json:"tokenUsage,omitempty"`

	// RAGMetrics contains retrieval-augmented generation metrics
	// +optional
	RAGMetrics *RAGMetrics `json:"ragMetrics,omitempty"`

	// QualityScore represents the quality of processing
	// +optional
	QualityScore *float64 `json:"qualityScore,omitempty"`

	// ValidationErrors contains any validation errors
	// +optional
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// ObservedGeneration reflects the generation observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// TokenUsageInfo tracks LLM token usage
type TokenUsageInfo struct {
	// PromptTokens used for the prompt
	PromptTokens int32 `json:"promptTokens"`
	// CompletionTokens used for the response
	CompletionTokens int32 `json:"completionTokens"`
	// TotalTokens used in total
	TotalTokens int32 `json:"totalTokens"`
	// EstimatedCost in USD
	// +optional
	EstimatedCost *float64 `json:"estimatedCost,omitempty"`
	// Provider used for processing
	Provider LLMProvider `json:"provider"`
	// Model used for processing
	Model string `json:"model"`
}

// RAGMetrics contains metrics for RAG processing
type RAGMetrics struct {
	// DocumentsRetrieved is the number of documents retrieved
	DocumentsRetrieved int32 `json:"documentsRetrieved"`
	// RetrievalDuration is the time taken to retrieve documents
	RetrievalDuration metav1.Duration `json:"retrievalDuration"`
	// AverageRelevanceScore is the average relevance of retrieved docs
	AverageRelevanceScore float64 `json:"averageRelevanceScore"`
	// TopRelevanceScore is the highest relevance score
	TopRelevanceScore float64 `json:"topRelevanceScore"`
	// KnowledgeBase used for retrieval
	KnowledgeBase string `json:"knowledgeBase"`
	// QueryEnhancement indicates if query was enhanced
	QueryEnhancement bool `json:"queryEnhancement"`
}

// ObjectReference represents a reference to a Kubernetes object
type ObjectReference struct {
	// APIVersion of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind of the referent
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`
	// Name of the referent
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Namespace of the referent
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// UID of the referent
	// +optional
	UID string `json:"uid,omitempty"`
	// ResourceVersion of the referent
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Parent Intent",type=string,JSONPath=`.spec.parentIntentRef.name`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Priority",type=string,JSONPath=`.spec.priority`
//+kubebuilder:printcolumn:name="Retries",type=integer,JSONPath=`.status.retryCount`
//+kubebuilder:printcolumn:name="Duration",type=string,JSONPath=`.status.processingDuration`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=ip;intentproc
//+kubebuilder:storageversion

// IntentProcessing is the Schema for the intentprocessings API
type IntentProcessing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntentProcessingSpec   `json:"spec,omitempty"`
	Status IntentProcessingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IntentProcessingList contains a list of IntentProcessing
type IntentProcessingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntentProcessing `json:"items"`
}

// GetParentIntentName returns the name of the parent NetworkIntent
func (ip *IntentProcessing) GetParentIntentName() string {
	return ip.Spec.ParentIntentRef.Name
}

// GetParentIntentNamespace returns the namespace of the parent NetworkIntent
func (ip *IntentProcessing) GetParentIntentNamespace() string {
	if ip.Spec.ParentIntentRef.Namespace != "" {
		return ip.Spec.ParentIntentRef.Namespace
	}
	return ip.GetNamespace() // Default to same namespace
}

// IsProcessingComplete returns true if processing is complete
func (ip *IntentProcessing) IsProcessingComplete() bool {
	return ip.Status.Phase == IntentProcessingPhaseCompleted
}

// IsProcessingFailed returns true if processing has failed
func (ip *IntentProcessing) IsProcessingFailed() bool {
	return ip.Status.Phase == IntentProcessingPhaseFailed
}

// CanRetry returns true if the processing can be retried
func (ip *IntentProcessing) CanRetry() bool {
	if ip.Spec.MaxRetries == nil {
		return false
	}
	return ip.Status.RetryCount < *ip.Spec.MaxRetries
}

// GetProcessingTimeout returns the timeout for processing
func (ip *IntentProcessing) GetProcessingTimeout() time.Duration {
	if ip.Spec.TimeoutSeconds == nil {
		return 120 * time.Second // Default timeout
	}
	return time.Duration(*ip.Spec.TimeoutSeconds) * time.Second
}

// ShouldEnableRAG returns true if RAG should be enabled
func (ip *IntentProcessing) ShouldEnableRAG() bool {
	if ip.Spec.ContextEnrichment == nil {
		return true // Default to enabled
	}
	return *ip.Spec.ContextEnrichment
}

func init() {
	SchemeBuilder.Register(&IntentProcessing{}, &IntentProcessingList{})
}
