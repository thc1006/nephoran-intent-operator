package edge

import "context"

// EdgeNode represents an edge computing node in the O-RAN network
type EdgeNode struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Location Location          `json:"location"`
	Metadata map[string]string `json:"metadata"`
	Status   EdgeNodeStatus    `json:"status"`
}

// Location represents geographical location of an edge node
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Region    string  `json:"region"`
	Zone      string  `json:"zone"`
}

// EdgeNodeStatus represents the status of an edge node
type EdgeNodeStatus struct {
	Phase      EdgeNodePhase `json:"phase"`
	Ready      bool          `json:"ready"`
	LastSeen   int64         `json:"lastSeen"`
	Conditions []Condition   `json:"conditions"`
}

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

// EdgeService defines the interface for edge node management
type EdgeService interface {
	RegisterNode(ctx context.Context, node *EdgeNode) error
	GetNode(ctx context.Context, id string) (*EdgeNode, error)
	ListNodes(ctx context.Context) ([]*EdgeNode, error)
	UpdateNodeStatus(ctx context.Context, id string, status EdgeNodeStatus) error
	DeleteNode(ctx context.Context, id string) error
}

// EdgeCompute represents computational resources at the edge
type EdgeCompute struct {
	CPU    ResourceSpec `json:"cpu"`
	Memory ResourceSpec `json:"memory"`
	Storage ResourceSpec `json:"storage"`
}

// ResourceSpec defines resource specifications
type ResourceSpec struct {
	Available int64 `json:"available"`
	Used      int64 `json:"used"`
	Unit      string `json:"unit"`
}