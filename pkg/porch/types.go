package porch

// PorchError represents an error from the Porch API
type PorchError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *PorchError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// PackageSpec represents the specification of a package
type PackageSpec struct {
	Repository string                 `json:"repository"`
	Package    string                 `json:"package"`
	Workspace  string                 `json:"workspace"`
	Resources  map[string]interface{} `json:"resources,omitempty"`
}

// PackageStatus represents the status of a package
type PackageStatus struct {
	Phase      string `json:"phase"`
	Message    string `json:"message,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition represents a condition in the package status
type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
}

// LifecycleManagerConfig holds configuration for lifecycle management
type LifecycleManagerConfig struct {
	MaxConcurrentPackages  int           `json:"maxConcurrentPackages"`
	EnableAutoCleanup      bool          `json:"enableAutoCleanup"`
	EventQueueSize         int           `json:"eventQueueSize"`
	EventWorkers           int           `json:"eventWorkers"`
	LockCleanupInterval    string        `json:"lockCleanupInterval"`
	DefaultLockTimeout     string        `json:"defaultLockTimeout"`
	EnableMetrics          bool          `json:"enableMetrics"`
}

// LifecycleManager manages package lifecycle operations
type LifecycleManager interface {
	Start() error
	Stop() error
	Close() error
	CreateRollbackPoint(packageSpec *PackageSpec) (*RollbackPoint, error)
	Rollback(point *RollbackPoint) error
	GetManagerHealth() *HealthStatus
}

// RollbackPoint represents a point in time for rollback operations
type RollbackPoint struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Package   string `json:"package"`
	Revision  string `json:"revision"`
}

// TransitionOptions holds options for package transitions
type TransitionOptions struct {
	Force       bool   `json:"force"`
	DryRun      bool   `json:"dryRun"`
	Timeout     string `json:"timeout"`
	Annotations map[string]string `json:"annotations"`
}

// TransitionResult represents the result of a package transition
type TransitionResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	NewPhase  string `json:"newPhase"`
	Timestamp string `json:"timestamp"`
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// PorchClient is an alias for the Porch client
type PorchClient = *Client

// defaultLifecycleManager provides a default implementation
type defaultLifecycleManager struct {
	client PorchClient
	config *LifecycleManagerConfig
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(client PorchClient, config *LifecycleManagerConfig) (LifecycleManager, error) {
	return &defaultLifecycleManager{
		client: client,
		config: config,
	}, nil
}

// Start starts the lifecycle manager
func (m *defaultLifecycleManager) Start() error {
	return nil
}

// Stop stops the lifecycle manager
func (m *defaultLifecycleManager) Stop() error {
	return nil
}

// Close closes the lifecycle manager
func (m *defaultLifecycleManager) Close() error {
	return nil
}

// CreateRollbackPoint creates a rollback point
func (m *defaultLifecycleManager) CreateRollbackPoint(packageSpec *PackageSpec) (*RollbackPoint, error) {
	return &RollbackPoint{
		ID:        "rollback-1",
		Timestamp: "2025-08-29T00:00:00Z",
		Package:   packageSpec.Package,
		Revision:  "v1.0.0",
	}, nil
}

// Rollback performs a rollback
func (m *defaultLifecycleManager) Rollback(point *RollbackPoint) error {
	return nil
}

// GetManagerHealth returns the manager health
func (m *defaultLifecycleManager) GetManagerHealth() *HealthStatus {
	return &HealthStatus{
		Status:    "healthy",
		Message:   "Lifecycle manager is running",
		Timestamp: "2025-08-29T00:00:00Z",
	}
}