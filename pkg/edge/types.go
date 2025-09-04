package edge

import "context"

// EdgeNodePhase represents the lifecycle phase of an edge node
type EdgeNodePhase string

const (
	EdgeNodePhaseRegistering EdgeNodePhase = "Registering"
	EdgeNodePhaseReady       EdgeNodePhase = "Ready"
	EdgeNodePhaseFailed      EdgeNodePhase = "Failed"
	EdgeNodePhaseOffline     EdgeNodePhase = "Offline"
)

// Condition represents a condition of an edge node
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// EdgeServiceInterface defines the interface for edge node management
type EdgeServiceInterface interface {
	RegisterNode(ctx context.Context, node *EdgeNode) error
	GetNode(ctx context.Context, id string) (*EdgeNode, error)
	ListNodes(ctx context.Context) ([]*EdgeNode, error)
	UpdateNodeStatus(ctx context.Context, id string, status EdgeNodeStatus) error
	DeleteNode(ctx context.Context, id string) error
}

// EdgeCompute represents computational resources at the edge
type EdgeCompute struct {
	CPU     ResourceSpec `json:"cpu"`
	Memory  ResourceSpec `json:"memory"`
	Storage ResourceSpec `json:"storage"`
}

// ResourceSpec defines resource specifications
type ResourceSpec struct {
	Available int64  `json:"available"`
	Used      int64  `json:"used"`
	Unit      string `json:"unit"`
}

// Location represents geographical location of an edge node
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Region    string  `json:"region"`
	Zone      string  `json:"zone"`
}
