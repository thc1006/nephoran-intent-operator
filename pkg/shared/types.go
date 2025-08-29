
package shared



import (

	"context"

	"time"

)



// ClientInterface defines the interface for LLM clients.

// This interface is shared between packages to avoid circular dependencies.

type ClientInterface interface {

	ProcessIntent(ctx context.Context, prompt string) (string, error)

	ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *StreamingChunk) error

	GetSupportedModels() []string

	GetModelCapabilities(modelName string) (*ModelCapabilities, error)

	ValidateModel(modelName string) error

	EstimateTokens(text string) int

	GetMaxTokens(modelName string) int

	Close() error

}



// StreamingChunk represents a chunk of streamed response.

type StreamingChunk struct {

	Content   string

	IsLast    bool

	Metadata  map[string]interface{}

	Timestamp time.Time

}



// ModelCapabilities describes what a model can do.

type ModelCapabilities struct {

	MaxTokens         int                    `json:"max_tokens"`

	SupportsChat      bool                   `json:"supports_chat"`

	SupportsFunction  bool                   `json:"supports_function"`

	SupportsStreaming bool                   `json:"supports_streaming"`

	CostPerToken      float64                `json:"cost_per_token"`

	Features          map[string]interface{} `json:"features"`

}



// TelecomDocument represents a document in the telecom knowledge base.

type TelecomDocument struct {

	ID              string                 `json:"id"`

	Title           string                 `json:"title"`

	Content         string                 `json:"content"`

	Source          string                 `json:"source"`

	Category        string                 `json:"category"`

	Version         string                 `json:"version"`

	Keywords        []string               `json:"keywords"`

	Language        string                 `json:"language"`

	DocumentType    string                 `json:"document_type"`

	NetworkFunction []string               `json:"network_function"`

	Technology      []string               `json:"technology"`

	UseCase         []string               `json:"use_case"`

	Confidence      float32                `json:"confidence"`

	Metadata        map[string]interface{} `json:"metadata"`

	Timestamp       time.Time              `json:"timestamp"` // For compatibility with some components

	CreatedAt       time.Time              `json:"created_at"`

	UpdatedAt       time.Time              `json:"updated_at"`

}



// SearchResult represents a search result from the vector database.

type SearchResult struct {

	Document *TelecomDocument       `json:"document"`

	Score    float32                `json:"score"`

	Distance float32                `json:"distance"`

	Metadata map[string]interface{} `json:"metadata"`

}



// SearchQuery represents a search query to the vector database.

type SearchQuery struct {

	Query         string                 `json:"query"`

	Limit         int                    `json:"limit"`

	Offset        int                    `json:"offset"`

	Filters       map[string]interface{} `json:"filters,omitempty"`

	HybridSearch  bool                   `json:"hybrid_search"`

	HybridAlpha   float32                `json:"hybrid_alpha"`

	UseReranker   bool                   `json:"use_reranker"`

	MinConfidence float32                `json:"min_confidence"`

	IncludeVector bool                   `json:"include_vector"`

	ExpandQuery   bool                   `json:"expand_query"`

	TargetVectors []string               `json:"target_vectors,omitempty"` // For named vectors support

}



// SearchResponse represents the response from a search operation.

type SearchResponse struct {

	Results []*SearchResult `json:"results"`

	Took    int64           `json:"took"`

	Total   int64           `json:"total"`

}



// ComponentType represents different types of components in the system.

type ComponentType string



const (

	// Core processing components.

	ComponentTypeLLMProcessor ComponentType = "llm-processor"

	// ComponentTypeResourcePlanner holds componenttyperesourceplanner value.

	ComponentTypeResourcePlanner ComponentType = "resource-planner"

	// ComponentTypeManifestGenerator holds componenttypemanifestgenerator value.

	ComponentTypeManifestGenerator ComponentType = "manifest-generator"

	// ComponentTypeGitOpsController holds componenttypegitopscontroller value.

	ComponentTypeGitOpsController ComponentType = "gitops-controller"

	// ComponentTypeDeploymentVerifier holds componenttypedeploymentverifier value.

	ComponentTypeDeploymentVerifier ComponentType = "deployment-verifier"



	// Optimization and analysis components.

	ComponentTypeRAGSystem ComponentType = "rag-system"

	// ComponentTypeNephioIntegration holds componenttypenephiointegration value.

	ComponentTypeNephioIntegration ComponentType = "nephio-integration"

	// ComponentTypeAuthentication holds componenttypeauthentication value.

	ComponentTypeAuthentication ComponentType = "authentication"

	// ComponentTypeDatabase holds componenttypedatabase value.

	ComponentTypeDatabase ComponentType = "database"

	// ComponentTypeCache holds componenttypecache value.

	ComponentTypeCache ComponentType = "cache"

	// ComponentTypeKubernetes holds componenttypekubernetes value.

	ComponentTypeKubernetes ComponentType = "kubernetes"

	// ComponentTypeNetworking holds componenttypenetworking value.

	ComponentTypeNetworking ComponentType = "networking"

)



// ComponentStatus represents the status of a component.

type ComponentStatus struct {

	Type       ComponentType          `json:"type"`

	Name       string                 `json:"name"`

	Status     string                 `json:"status"`

	Healthy    bool                   `json:"healthy"`

	LastUpdate time.Time              `json:"lastUpdate"`

	Version    string                 `json:"version,omitempty"`

	Metadata   map[string]interface{} `json:"metadata,omitempty"`

	Metrics    map[string]float64     `json:"metrics,omitempty"`

	Errors     []string               `json:"errors,omitempty"`

}



// SystemHealth represents the overall health of the system.

type SystemHealth struct {

	OverallStatus  string                      `json:"overallStatus"`

	Healthy        bool                        `json:"healthy"`

	Components     map[string]*ComponentStatus `json:"components"`

	LastUpdate     time.Time                   `json:"lastUpdate"`

	ActiveIntents  int                         `json:"activeIntents"`

	ProcessingRate float64                     `json:"processingRate"`

	ErrorRate      float64                     `json:"errorRate"`

	ResourceUsage  ResourceUsage               `json:"resourceUsage"`

}



// ResourceUsage represents resource utilization.

type ResourceUsage struct {

	CPUPercent        float64 `json:"cpuPercent"`

	MemoryPercent     float64 `json:"memoryPercent"`

	DiskPercent       float64 `json:"diskPercent"`

	NetworkInMBps     float64 `json:"networkInMBps"`

	NetworkOutMBps    float64 `json:"networkOutMBps"`

	ActiveConnections int     `json:"activeConnections"`

}

