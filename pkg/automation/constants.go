package automation

import "time"

// AutomationType defines types of automation requests.

type AutomationType string

const (

	// AutomationTypeProvisioning represents certificate provisioning automation.

	AutomationTypeProvisioning AutomationType = "provisioning"

	// AutomationTypeRenewal represents certificate renewal automation.

	AutomationTypeRenewal AutomationType = "renewal"

	// AutomationTypeRevocation represents certificate revocation automation.

	AutomationTypeRevocation AutomationType = "revocation"

	// AutomationTypeDiscovery represents service discovery automation.

	AutomationTypeDiscovery AutomationType = "discovery"

	// AutomationTypeHealthCheck represents health check automation.

	AutomationTypeHealthCheck AutomationType = "health_check"
)

// AutomationStatus defines the status of automation operations.

type AutomationStatus string

const (

	// AutomationStatusPending indicates an automation request is pending.

	AutomationStatusPending AutomationStatus = "pending"

	// AutomationStatusProcessing indicates an automation request is being processed.

	AutomationStatusProcessing AutomationStatus = "processing"

	// AutomationStatusCompleted indicates an automation request completed successfully.

	AutomationStatusCompleted AutomationStatus = "completed"

	// AutomationStatusFailed indicates an automation request failed.

	AutomationStatusFailed AutomationStatus = "failed"

	// AutomationStatusCanceled indicates an automation request was canceled.

	AutomationStatusCanceled AutomationStatus = "canceled"
)

// AutomationPriority defines automation request priorities.

type AutomationPriority int

const (

	// AutomationPriorityLow represents low priority automation.

	AutomationPriorityLow AutomationPriority = iota

	// AutomationPriorityNormal represents normal priority automation.

	AutomationPriorityNormal

	// AutomationPriorityHigh represents high priority automation.

	AutomationPriorityHigh

	// AutomationPriorityCritical represents critical priority automation.

	AutomationPriorityCritical
)

// 2025 Modern Automation Constants
const (
	// DefaultRemediationTimeout is the default timeout for remediation operations
	DefaultRemediationTimeout = 10 * time.Minute

	// DefaultLearningBatchSize is the default batch size for ML learning operations
	DefaultLearningBatchSize = 100

	// MaxConcurrentRemediations is the maximum number of concurrent remediation sessions
	MaxConcurrentRemediations = 5

	// DefaultHealthCheckInterval is the default interval for health checks
	DefaultHealthCheckInterval = 30 * time.Second

	// AIModelUpdateInterval is the interval for updating AI models
	AIModelUpdateInterval = 6 * time.Hour
)

// 2025 Automation Features
const (
	// FeatureAIPoweredRemediation enables AI-powered automated remediation
	FeatureAIPoweredRemediation = "ai_powered_remediation"

	// FeaturePredictiveScaling enables predictive auto-scaling
	FeaturePredictiveScaling = "predictive_scaling"

	// FeatureAnomalyDetection enables ML-based anomaly detection
	FeatureAnomalyDetection = "anomaly_detection"

	// FeatureSmartAlerts enables intelligent alert filtering
	FeatureSmartAlerts = "smart_alerts"

	// FeatureAutomatedTesting enables automated testing in production
	FeatureAutomatedTesting = "automated_testing"
)
