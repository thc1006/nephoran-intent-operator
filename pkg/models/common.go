package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        string    `json:"id" yaml:"id"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	Version   int       `json:"version" yaml:"version"`
}

// NewBaseModel creates a new BaseModel with defaults
func NewBaseModel() BaseModel {
	now := time.Now()
	return BaseModel{
		ID:        uuid.New().String(),
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

// NetworkIntent represents a network configuration intent
type NetworkIntent struct {
	BaseModel `json:",inline" yaml:",inline"`
	Spec   NetworkIntentSpec   `json:"spec" yaml:"spec"`
	Status NetworkIntentStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// NetworkIntentSpec defines the desired state of NetworkIntent
type NetworkIntentSpec struct {
	Intent          string            `json:"intent" yaml:"intent"`
	NetworkType     string            `json:"network_type" yaml:"network_type"`
	Requirements    map[string]string `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Constraints     []string          `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Configuration   json.RawMessage   `json:"configuration,omitempty" yaml:"configuration,omitempty"`
	ValidationRules []string          `json:"validation_rules,omitempty" yaml:"validation_rules,omitempty"`
}

// NetworkIntentStatus defines the observed state of NetworkIntent
type NetworkIntentStatus struct {
	Phase              string             `json:"phase" yaml:"phase"`
	Conditions         []Condition        `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	ProcessedIntent    json.RawMessage    `json:"processed_intent,omitempty" yaml:"processed_intent,omitempty"`
	GeneratedConfig    json.RawMessage    `json:"generated_config,omitempty" yaml:"generated_config,omitempty"`
	ValidationResults  []ValidationResult `json:"validation_results,omitempty" yaml:"validation_results,omitempty"`
	LastProcessedAt    *time.Time         `json:"last_processed_at,omitempty" yaml:"last_processed_at,omitempty"`
	ObservedGeneration int64              `json:"observed_generation,omitempty" yaml:"observed_generation,omitempty"`
}

// Condition represents a condition of a NetworkIntent
type Condition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastTransitionTime time.Time `json:"last_transition_time" yaml:"last_transition_time"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Rule      string    `json:"rule" yaml:"rule"`
	Status    string    `json:"status" yaml:"status"`
	Message   string    `json:"message,omitempty" yaml:"message,omitempty"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// NetworkConfiguration represents a processed network configuration
type NetworkConfiguration struct {
	BaseModel `json:",inline" yaml:",inline"`
	IntentID      string          `json:"intent_id" yaml:"intent_id"`
	ConfigType    string          `json:"config_type" yaml:"config_type"`
	Configuration json.RawMessage `json:"configuration" yaml:"configuration"`
	Status        ConfigStatus    `json:"status" yaml:"status"`
}

// ConfigStatus represents the status of a network configuration
type ConfigStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Applied    bool        `json:"applied" yaml:"applied"`
	AppliedAt  *time.Time  `json:"applied_at,omitempty" yaml:"applied_at,omitempty"`
	Errors     []string    `json:"errors,omitempty" yaml:"errors,omitempty"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ExecutionPolicy represents an execution policy for intents
type ExecutionPolicy struct {
	BaseModel `json:",inline" yaml:",inline"`
	Name        string                `json:"name" yaml:"name"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Rules       []ExecutionRule       `json:"rules" yaml:"rules"`
	Metadata    map[string]string     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Status      ExecutionPolicyStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// ExecutionRule represents a rule in an execution policy
type ExecutionRule struct {
	ID         string            `json:"id" yaml:"id"`
	Type       string            `json:"type" yaml:"type"`
	Condition  string            `json:"condition" yaml:"condition"`
	Action     string            `json:"action" yaml:"action"`
	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Priority   int               `json:"priority" yaml:"priority"`
	Enabled    bool              `json:"enabled" yaml:"enabled"`
}

// ExecutionPolicyStatus represents the status of an execution policy
type ExecutionPolicyStatus struct {
	Active         bool       `json:"active" yaml:"active"`
	LastExecuted   *time.Time `json:"last_executed,omitempty" yaml:"last_executed,omitempty"`
	ExecutionCount int        `json:"execution_count" yaml:"execution_count"`
	Errors         []string   `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// ProcessingMetrics contains metrics for intent processing
type ProcessingMetrics struct {
	BaseModel `json:",inline" yaml:",inline"`
	IntentID       string        `json:"intent_id" yaml:"intent_id"`
	ProcessingTime time.Duration `json:"processing_time" yaml:"processing_time"`
	ValidationTime time.Duration `json:"validation_time" yaml:"validation_time"`
	ConfigGenTime  time.Duration `json:"config_generation_time" yaml:"config_generation_time"`
	TotalTime      time.Duration `json:"total_time" yaml:"total_time"`
	TokensUsed     int           `json:"tokens_used,omitempty" yaml:"tokens_used,omitempty"`
	Cost           float64       `json:"cost,omitempty" yaml:"cost,omitempty"`
	Status         string        `json:"status" yaml:"status"`
}

// Constants for status and phase values
const (
	// Phases
	PhasePending    = "Pending"
	PhaseProcessing = "Processing"
	PhaseValidating = "Validating"
	PhaseReady      = "Ready"
	PhaseFailed     = "Failed"
	PhaseApplied    = "Applied"

	// Condition types
	ConditionTypeProcessed = "Processed"
	ConditionTypeValidated = "Validated"
	ConditionTypeReady     = "Ready"
	ConditionTypeApplied   = "Applied"

	// Condition statuses
	ConditionStatusTrue    = "True"
	ConditionStatusFalse   = "False"
	ConditionStatusUnknown = "Unknown"

	// Validation statuses
	ValidationStatusPass = "Pass"
	ValidationStatusFail = "Fail"
	ValidationStatusWarn = "Warning"
)
