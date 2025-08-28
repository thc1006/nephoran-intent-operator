package shared

import (
	"context"
	"time"
)

// RAG-related types and interfaces

// SearchResult represents a single search result from RAG
type SearchResult struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	Score      float32                `json:"score"`
	Distance   float32                `json:"distance"`   // Added for test compatibility
	Source     string                 `json:"source"`
	Chunk      int                    `json:"chunk"`
	Title      string                 `json:"title"`
	Summary    string                 `json:"summary"`
	Document   *TelecomDocument       `json:"document"`   // Added for test compatibility
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Confidence float32                `json:"confidence"` // Added for test compatibility
}

// SearchQuery represents a query to the RAG system
type SearchQuery struct {
	Query     string                 `json:"query"`
	Limit     int                    `json:"limit,omitempty"`
	Threshold float32                `json:"threshold,omitempty"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	ContextID string                 `json:"context_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Namespace string                 `json:"namespace,omitempty"`
}

// RAGResponse represents the response from RAG system
type RAGResponse struct {
	Query     string            `json:"query"`
	Results   []SearchResult    `json:"results"`
	Context   string            `json:"context"`
	Documents []TelecomDocument `json:"documents"`
	TotalTime time.Duration     `json:"total_time"`
	Error     *RAGError         `json:"error,omitempty"`
}

// TelecomDocument represents a telecom-specific document
type TelecomDocument struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	Type            DocumentType           `json:"type"`
	Category        string                 `json:"category"`
	Standard        string                 `json:"standard"`         // e.g., "3GPP TS 38.401"
	Version         string                 `json:"version"`          // e.g., "16.0.0"
	Section         string                 `json:"section"`          // e.g., "7.2.1"
	Keywords        []string               `json:"keywords"`
	Source          string                 `json:"source"`           // Added for test compatibility
	NetworkFunction []string               `json:"network_function"` // Added for test compatibility
	Technology      []string               `json:"technology"`       // Added for test compatibility
	Confidence      float64                `json:"confidence"`       // Added for test compatibility
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Indexed         bool                   `json:"indexed"`
}

// DocumentType represents the type of telecom document
type DocumentType string

const (
	DocumentTypeSpec     DocumentType = "specification"
	DocumentTypeStandard DocumentType = "standard"
	DocumentTypeGuide    DocumentType = "guide"
	DocumentTypeAPI      DocumentType = "api"
	DocumentTypeConfig   DocumentType = "configuration"
	DocumentTypePolicy   DocumentType = "policy"
	DocumentTypeOther    DocumentType = "other"
)

// RAGError represents an error from RAG system
type RAGError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// RAGServiceInterface defines the interface for RAG service
type RAGServiceInterface interface {
	Search(ctx context.Context, query *SearchQuery) (*RAGResponse, error)
	IndexDocument(ctx context.Context, document *TelecomDocument) error
	DeleteDocument(ctx context.Context, documentID string) error
	GetDocument(ctx context.Context, documentID string) (*TelecomDocument, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

// QueryRequest represents a request for query processing
type QueryRequest struct {
	Query     string                 `json:"query"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Threshold float32                `json:"threshold,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// EmbeddingServiceInterface defines the interface for embedding service
type EmbeddingServiceInterface interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	CreateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	GetDimensions() int
	GetModel() string
}

// VectorStoreInterface defines the interface for vector storage
type VectorStoreInterface interface {
	Store(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, vector []float32, limit int, threshold float32) ([]SearchResult, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*SearchResult, error)
	HealthCheck(ctx context.Context) error
	Close() error
}
