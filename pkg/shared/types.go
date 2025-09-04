package shared

import (
	
	"encoding/json"
"time"
)

// SearchResponse represents the response from a search operation.

type SearchResponse struct {
	Results []*SearchResult `json:"results"`

	Took int64 `json:"took"`

	Total int64 `json:"total"`
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
	Type ComponentType `json:"type"`

	Name string `json:"name"`

	Status string `json:"status"`

	Healthy bool `json:"healthy"`

	LastUpdate time.Time `json:"lastUpdate"`

	Version string `json:"version,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`

	Metrics map[string]float64 `json:"metrics,omitempty"`

	Errors []string `json:"errors,omitempty"`
}

// SystemHealth represents the overall health of the system.

type SystemHealth struct {
	OverallStatus string `json:"overallStatus"`

	Healthy bool `json:"healthy"`

	Components map[string]*ComponentStatus `json:"components"`

	LastUpdate time.Time `json:"lastUpdate"`

	ActiveIntents int `json:"activeIntents"`

	ProcessingRate float64 `json:"processingRate"`

	ErrorRate float64 `json:"errorRate"`

	ResourceUsage ResourceUsage `json:"resourceUsage"`
}

// ResourceUsage represents resource utilization.

type ResourceUsage struct {
	CPUPercent float64 `json:"cpuPercent"`

	MemoryPercent float64 `json:"memoryPercent"`

	DiskPercent float64 `json:"diskPercent"`

	NetworkInMBps float64 `json:"networkInMBps"`

	NetworkOutMBps float64 `json:"networkOutMBps"`

	ActiveConnections int `json:"activeConnections"`
}
