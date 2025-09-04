package nephio

import (
	
	"encoding/json"
"context"
	"time"
)

// Publisher interface for publishing Nephio packages
type Publisher interface {
	PublishPackage(ctx context.Context, pkg *Package) (*PublishResult, error)
	GetPackageStatus(ctx context.Context, packageID string) (*PackageStatus, error)
	ListPackages(ctx context.Context, filter *PackageFilter) ([]*Package, error)
	DeletePackage(ctx context.Context, packageID string) error
}

// Package represents a Nephio package
type Package struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Resources   []Resource        `json:"resources"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Resource represents a Kubernetes resource in a package
type Resource struct {
	Kind       string                 `json:"kind"`
	APIVersion string                 `json:"api_version"`
	Metadata   ResourceMetadata       `json:"metadata"`
	Spec       json.RawMessage `json:"spec"`
}

// ResourceMetadata represents resource metadata
type ResourceMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PublishResult represents the result of a package publish operation
type PublishResult struct {
	PackageID   string    `json:"package_id"`
	PublishedAt time.Time `json:"published_at"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
}

// PackageStatus represents the status of a package
type PackageStatus struct {
	PackageID   string    `json:"package_id"`
	Status      string    `json:"status"` // "pending", "published", "failed", "deleted"
	Message     string    `json:"message,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// PackageFilter represents filtering criteria for packages
type PackageFilter struct {
	Name    string            `json:"name,omitempty"`
	Version string            `json:"version,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
	Status  string            `json:"status,omitempty"`
	Limit   int               `json:"limit,omitempty"`
	Offset  int               `json:"offset,omitempty"`
}

// Intent represents a network intent for Nephio
type Intent struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	IntentType      string                 `json:"intent_type"`
	NetworkFunction string                 `json:"network_function,omitempty"`
	Parameters      json.RawMessage `json:"parameters"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// IntentProcessor interface for processing network intents
type IntentProcessor interface {
	ProcessIntent(ctx context.Context, intent *Intent) (*ProcessingResult, error)
	GetProcessingStatus(ctx context.Context, intentID string) (*ProcessingStatus, error)
	ListIntents(ctx context.Context, filter *IntentFilter) ([]*Intent, error)
}

// ProcessingResult represents the result of intent processing
type ProcessingResult struct {
	IntentID      string                 `json:"intent_id"`
	PackageID     string                 `json:"package_id,omitempty"`
	GeneratedSpec json.RawMessage `json:"generated_spec"`
	Status        string                 `json:"status"`
	Message       string                 `json:"message,omitempty"`
	ProcessedAt   time.Time              `json:"processed_at"`
}

// ProcessingStatus represents the status of intent processing
type ProcessingStatus struct {
	IntentID    string    `json:"intent_id"`
	Status      string    `json:"status"`   // "processing", "completed", "failed"
	Progress    float64   `json:"progress"` // 0-100
	Message     string    `json:"message,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// IntentFilter represents filtering criteria for intents
type IntentFilter struct {
	Name            string `json:"name,omitempty"`
	IntentType      string `json:"intent_type,omitempty"`
	NetworkFunction string `json:"network_function,omitempty"`
	Status          string `json:"status,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Offset          int    `json:"offset,omitempty"`
}

// Client interface for Nephio operations
type Client interface {
	Publisher
	IntentProcessor
}

// Config represents Nephio client configuration
type Config struct {
	PorchEndpoint   string            `json:"porch_endpoint"`
	Namespace       string            `json:"namespace"`
	Repository      string            `json:"repository"`
	Timeout         time.Duration     `json:"timeout"`
	RetryAttempts   int               `json:"retry_attempts"`
	Headers         map[string]string `json:"headers,omitempty"`
	InsecureSkipTLS bool              `json:"insecure_skip_tls"`
}

// DefaultConfig returns default Nephio configuration
func DefaultConfig() *Config {
	return &Config{
		PorchEndpoint: "http://porch-server:7007",
		Namespace:     "default",
		Repository:    "blueprints",
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		Headers:       make(map[string]string),
	}
}
