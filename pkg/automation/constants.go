package automation

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
