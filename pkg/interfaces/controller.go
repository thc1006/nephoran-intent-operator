package interfaces

import (
	"context"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/contracts"
)

// Type aliases for backward compatibility
type ComponentType = contracts.ComponentType
type ProcessingPhase = contracts.ProcessingPhase

// Component type constants
const (
	ComponentUnknown    = contracts.ComponentUnknown
	ComponentIntentAPI  = contracts.ComponentIntentAPI
	ComponentVIMSIM     = contracts.ComponentVIMSIM
	ComponentNEPHIO     = contracts.ComponentNEPHIO
	ComponentORAN       = contracts.ComponentORAN
	ComponentCACHE      = contracts.ComponentCACHE
	ComponentCONTROLLER = contracts.ComponentCONTROLLER
	ComponentMETRICS    = contracts.ComponentMETRICS
	ComponentPERF       = contracts.ComponentPERF
)

// Phase constants
const (
	PhaseUnknown    = contracts.PhaseUnknown
	PhaseInit       = contracts.PhaseInit
	PhaseValidation = contracts.PhaseValidation
	PhaseProcessing = contracts.PhaseProcessing
	PhaseComplete   = contracts.PhaseComplete
	PhaseError      = contracts.PhaseError
)

// IntentProcessor is the main interface for processing network intents
type IntentProcessor interface {
	ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) error
	ValidateIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) error
}

// ComponentController manages component lifecycle
type ComponentController interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Status() ComponentStatus
}

// ComponentStatus represents component health status
type ComponentStatus struct {
	Healthy bool
	Message string
	Phase   ProcessingPhase
}

// MetricsCollector collects performance metrics
type MetricsCollector interface {
	RecordLatency(operation string, duration float64)
	RecordThroughput(operation string, value float64)
	RecordError(operation string, err error)
}

// EventHandler handles system events
type EventHandler interface {
	HandleEvent(ctx context.Context, event Event) error
}

// Event represents a system event
type Event struct {
	Type      string
	Component ComponentType
	Data      interface{}
}

// CacheManager manages caching operations
type CacheManager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	Delete(key string) error
}

// Ensure contracts types are available
var (
	_ ComponentType   = contracts.ComponentUnknown
	_ ProcessingPhase = contracts.PhaseUnknown
)