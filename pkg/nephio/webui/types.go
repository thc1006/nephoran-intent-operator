
package webui



import (

	"time"



	"github.com/google/uuid"

)



// APIError represents a standardized error response.

type APIError struct {

	Code    string `json:"code"`

	Message string `json:"message"`

	Details string `json:"details,omitempty"`

}



// PaginationRequest represents pagination parameters for API requests.

type PaginationRequest struct {

	Page     int `json:"page"`

	PageSize int `json:"page_size"`

}



// PaginationResponse wraps paginated results.

type PaginationResponse struct {

	Total      int64 `json:"total"`

	Page       int   `json:"page"`

	PageSize   int   `json:"page_size"`

	TotalPages int   `json:"total_pages"`

}



// PackageRevisionSpec represents the specification for a package revision.

type PackageRevisionSpec struct {

	Name        string            `json:"name"`

	Repository  string            `json:"repository"`

	Version     string            `json:"version"`

	Description string            `json:"description"`

	Labels      map[string]string `json:"labels"`

	Annotations map[string]string `json:"annotations"`

}



// PackageRevisionStatus represents the status of a package revision.

type PackageRevisionStatus struct {

	Phase       string      `json:"phase"`

	Conditions  []Condition `json:"conditions"`

	LastUpdated time.Time   `json:"last_updated"`

}



// Condition represents a condition in a resource's status.

type Condition struct {

	Type    string `json:"type"`

	Status  string `json:"status"`

	Reason  string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

}



// PackageRevision represents a complete package revision.

type PackageRevision struct {

	ID        uuid.UUID             `json:"id"`

	Spec      PackageRevisionSpec   `json:"spec"`

	Status    PackageRevisionStatus `json:"status"`

	CreatedAt time.Time             `json:"created_at"`

	UpdatedAt time.Time             `json:"updated_at"`

}



// WorkloadCluster represents a managed workload cluster.

type WorkloadCluster struct {

	Name        string            `json:"name"`

	Namespace   string            `json:"namespace"`

	Environment string            `json:"environment"`

	Status      string            `json:"status"`

	Labels      map[string]string `json:"labels"`

	LastSync    time.Time         `json:"last_sync"`

}



// NetworkIntent represents a network configuration intent.

type NetworkIntent struct {

	ID          uuid.UUID           `json:"id"`

	Description string              `json:"description"`

	Spec        NetworkIntentSpec   `json:"spec"`

	Status      NetworkIntentStatus `json:"status"`

	CreatedAt   time.Time           `json:"created_at"`

}



// NetworkIntentSpec defines the desired network configuration.

type NetworkIntentSpec struct {

	Type           string            `json:"type"`

	Parameters     map[string]string `json:"parameters"`

	TargetClusters []string          `json:"target_clusters"`

}



// NetworkIntentStatus represents the processing status of a network intent.

type NetworkIntentStatus struct {

	Phase       string      `json:"phase"`

	Progress    float64     `json:"progress"`

	Conditions  []Condition `json:"conditions"`

	LastUpdated time.Time   `json:"last_updated"`

}



// GitRepository represents a configured Git repository.

type GitRepository struct {

	Name        string    `json:"name"`

	URL         string    `json:"url"`

	Branch      string    `json:"branch"`

	Credentials string    `json:"credentials"`

	LastSync    time.Time `json:"last_sync"`

	Status      string    `json:"status"`

}



// WebSocketMessage represents a message sent over WebSocket.

type WebSocketMessage struct {

	Type    string      `json:"type"`

	Payload interface{} `json:"payload"`

}



// WebSocketEventType defines types of WebSocket events.

type WebSocketEventType string



const (

	// EventTypePackageUpdate holds eventtypepackageupdate value.

	EventTypePackageUpdate WebSocketEventType = "package_update"

	// EventTypeClusterStatus holds eventtypeclusterstatus value.

	EventTypeClusterStatus WebSocketEventType = "cluster_status"

	// EventTypeIntentProcessing holds eventtypeintentprocessing value.

	EventTypeIntentProcessing WebSocketEventType = "intent_processing"

	// EventTypeSystemNotification holds eventtypesystemnotification value.

	EventTypeSystemNotification WebSocketEventType = "system_notification"

)



// APIHealthStatus represents the health status of the API service.

type APIHealthStatus struct {

	Status           string            `json:"status"`

	Version          string            `json:"version"`

	Uptime           time.Duration     `json:"uptime"`

	Components       map[string]string `json:"components"`

	DatabaseStatus   string            `json:"database_status"`

	CacheStatus      string            `json:"cache_status"`

	ConnectionStatus string            `json:"connection_status"`

}

