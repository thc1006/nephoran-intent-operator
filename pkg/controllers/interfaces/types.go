package interfaces

// Controller defines the interface for all Nephoran controllers
type Controller interface {
	Start() error
	Stop() error
	GetName() string
}

// Reconciler defines the interface for Kubernetes resource reconciliation
type Reconciler interface {
	Reconcile(req ReconcileRequest) (ReconcileResult, error)
}

// ReconcileRequest represents a reconciliation request
type ReconcileRequest struct {
	Namespace string
	Name      string
}

// ReconcileResult represents the result of a reconciliation
type ReconcileResult struct {
	Requeue      bool
	RequeueAfter int64 // seconds
}

// Manager defines the interface for controller managers
type Manager interface {
	AddController(Controller) error
	Start() error
	Stop() error
}

// StatusUpdater defines the interface for updating resource status
type StatusUpdater interface {
	UpdateStatus(namespace, name string, status interface{}) error
}