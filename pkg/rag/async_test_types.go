//go:build test




package rag



import (

	"context"

	"time"

)



// Type aliases for test compatibility.

// These aliases are only compiled during tests to avoid conflicts with production types.

type (

	DocumentJob = TestDocumentJob

	// QueryJob represents a queryjob.

	QueryJob = TestQueryJob

)



// Missing types for test compilation.

type DocumentMetadata struct {

	Source   string                 `json:"source"`

	Title    string                 `json:"title"`

	FilePath string                 `json:"file_path"`

	Size     int64                  `json:"size"`

	Custom   map[string]interface{} `json:"custom"`

}



// LoadedDocument represents a loadeddocument.

type LoadedDocument struct {

	ID       string                 `json:"id"`

	Content  string                 `json:"content"`

	Metadata map[string]interface{} `json:"metadata"`

	FilePath string                 `json:"file_path"`

	Size     int64                  `json:"size"`

}



// RAGService represents a ragservice.

type RAGService interface {

	ProcessIntent(ctx context.Context, intent string) (string, error)

	IsHealthy() bool

}



// ProcessedDocument represents a processeddocument.

type ProcessedDocument struct {

	ID       string                 `json:"id"`

	Content  string                 `json:"content"`

	Metadata map[string]interface{} `json:"metadata"`

	Chunks   []DocumentChunk        `json:"chunks"`

}



// QueryParameters represents a queryparameters.

type QueryParameters struct {

	Limit   int                    `json:"limit"`

	Filters map[string]interface{} `json:"filters"`

}



// QueryResult represents a queryresult.

type QueryResult struct {

	Results   []RetrievedContext `json:"results"`

	Query     string             `json:"query"`

	Timestamp time.Time          `json:"timestamp"`

}

