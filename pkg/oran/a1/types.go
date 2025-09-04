// Package a1 implements O-RAN A1 Policy Management Service compliant with O-RAN.WG2.A1AP-v03.01 specification.

// This package provides complete A1-P (Policy), A1-C (Consumer), and A1-EI (Enrichment Information) interfaces.

package a1

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// A1Interface represents the three O-RAN A1 interface variants.

type A1Interface string

const (

	// A1PolicyInterface holds a1policyinterface value.

	A1PolicyInterface A1Interface = "A1-P" // Policy interface

	// A1ConsumerInterface holds a1consumerinterface value.

	A1ConsumerInterface A1Interface = "A1-C" // Consumer interface

	// A1EnrichmentInterface holds a1enrichmentinterface value.

	A1EnrichmentInterface A1Interface = "A1-EI" // Enrichment Information interface

)

// A1Version represents supported A1 API versions.

type A1Version string

const (

	// A1PolicyVersion holds a1policyversion value.

	A1PolicyVersion A1Version = "v2" // A1-P version 2

	// A1ConsumerVersion holds a1consumerversion value.

	A1ConsumerVersion A1Version = "v1" // A1-C version 1

	// A1EnrichmentVersion holds a1enrichmentversion value.

	A1EnrichmentVersion A1Version = "v1" // A1-EI version 1

)

// PolicyType represents an O-RAN compliant A1 policy type definition.

type PolicyType struct {
	PolicyTypeID int `json:"policy_type_id" validate:"required,min=1"`

	PolicyTypeName string `json:"policy_type_name,omitempty"`

	Description string `json:"description,omitempty"`

	Schema map[string]interface{} `json:"schema" validate:"required"`

	CreateSchema json.RawMessage `json:"create_schema,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`

	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

// PolicyInstance represents an O-RAN compliant A1 policy instance.

type PolicyInstance struct {
	PolicyID string `json:"policy_id" validate:"required"`

	PolicyTypeID int `json:"policy_type_id" validate:"required,min=1"`

	PolicyData map[string]interface{} `json:"policy_data" validate:"required"`

	PolicyInfo PolicyInstanceInfo `json:"policy_info,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`

	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

// PolicyInstanceInfo provides additional metadata for policy instances.

type PolicyInstanceInfo struct {
	NotificationDestination string `json:"notification_destination,omitempty"`

	RequestID string `json:"request_id,omitempty"`

	AdditionalParams json.RawMessage `json:"additional_params,omitempty"`
}

// PolicyStatus represents the status of an O-RAN A1 policy instance.

type PolicyStatus struct {
	EnforcementStatus string `json:"enforcement_status" validate:"required,oneof=NOT_ENFORCED ENFORCED UNKNOWN"`

	EnforcementReason string `json:"enforcement_reason,omitempty"`

	HasBeenDeleted bool `json:"has_been_deleted,omitempty"`

	Deleted bool `json:"deleted,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`

	ModifiedAt time.Time `json:"modified_at,omitempty"`

	AdditionalInfo json.RawMessage `json:"additional_info,omitempty"`
}

// EnrichmentInfoType represents A1-EI type definition per O-RAN specification.

type EnrichmentInfoType struct {
	EiTypeID string `json:"ei_type_id" validate:"required"`

	EiTypeName string `json:"ei_type_name,omitempty"`

	Description string `json:"description,omitempty"`

	EiJobDataSchema map[string]interface{} `json:"ei_job_data_schema" validate:"required"`

	EiJobResultSchema json.RawMessage `json:"ei_job_result_schema,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`

	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

// EnrichmentInfoJob represents A1-EI job definition per O-RAN specification.

type EnrichmentInfoJob struct {
	EiJobID string `json:"ei_job_id" validate:"required"`

	EiTypeID string `json:"ei_type_id" validate:"required"`

	EiJobData map[string]interface{} `json:"ei_job_data" validate:"required"`

	TargetURI string `json:"target_uri" validate:"required,url"`

	JobOwner string `json:"job_owner" validate:"required"`

	JobStatusURL string `json:"job_status_url,omitempty"`

	JobDefinition EnrichmentJobDef `json:"job_definition,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`

	ModifiedAt time.Time `json:"modified_at,omitempty"`

	LastExecutedAt time.Time `json:"last_executed_at,omitempty"`
}

// EnrichmentJobDef provides additional job definition parameters.

type EnrichmentJobDef struct {
	DeliveryInfo []DeliveryInfo `json:"delivery_info,omitempty"`

	JobParameters json.RawMessage `json:"job_parameters,omitempty"`

	JobResultSchema json.RawMessage `json:"job_result_schema,omitempty"`

	StatusNotificationURI string `json:"status_notification_uri,omitempty"`
}

// DeliveryInfo specifies how enrichment information should be delivered.

type DeliveryInfo struct {
	TopicName string `json:"topic_name,omitempty"`

	BootStrapServer string `json:"boot_strap_server,omitempty"`

	AdditionalInfo json.RawMessage `json:"additional_info,omitempty"`
}

// EnrichmentInfoJobStatus represents the status of an EI job.

type EnrichmentInfoJobStatus struct {
	Status string `json:"status" validate:"required,oneof=ENABLED DISABLED"`

	Producers []string `json:"producers,omitempty"`

	LastUpdated time.Time `json:"last_updated,omitempty"`

	StatusInfo json.RawMessage `json:"status_info,omitempty"`
}

// ConsumerInfo represents A1-C consumer information.

type ConsumerInfo struct {
	ConsumerID string `json:"consumer_id" validate:"required"`

	ConsumerName string `json:"consumer_name,omitempty"`

	CallbackURL string `json:"callback_url" validate:"required,url"`

	Capabilities []string `json:"capabilities,omitempty"`

	Metadata ConsumerMetadata `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`

	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

// ConsumerMetadata provides additional consumer metadata.

type ConsumerMetadata struct {
	Version string `json:"version,omitempty"`

	Description string `json:"description,omitempty"`

	SupportedTypes []int `json:"supported_types,omitempty"`

	AdditionalInfo json.RawMessage `json:"additional_info,omitempty"`
}

// PolicyNotification represents A1 policy notifications.

type PolicyNotification struct {
	PolicyID string `json:"policy_id" validate:"required"`

	PolicyTypeID int `json:"policy_type_id" validate:"required,min=1"`

	NotificationType string `json:"notification_type" validate:"required,oneof=CREATE UPDATE DELETE"`

	Timestamp time.Time `json:"timestamp" validate:"required"`

	Status string `json:"status" validate:"required"`

	Message string `json:"message,omitempty"`

	AdditionalDetails json.RawMessage `json:"additional_details,omitempty"`
}

// A1ServerConfig holds configuration for the A1 server.

type A1ServerConfig struct {
	Port int `json:"port" validate:"required,min=1,max=65535"`

	Host string `json:"host"`

	TLSEnabled bool `json:"tls_enabled"`

	CertFile string `json:"cert_file"`

	KeyFile string `json:"key_file"`

	ReadTimeout time.Duration `json:"read_timeout"`

	WriteTimeout time.Duration `json:"write_timeout"`

	IdleTimeout time.Duration `json:"idle_timeout"`

	MaxHeaderBytes int `json:"max_header_bytes"`

	EnableA1P bool `json:"enable_a1p"`

	EnableA1C bool `json:"enable_a1c"`

	EnableA1EI bool `json:"enable_a1ei"`

	Logger *logging.StructuredLogger `json:"-"`

	CircuitBreakerConfig *CircuitBreakerConfig `json:"circuit_breaker_config"`

	ValidationConfig *ValidationConfig `json:"validation_config"`

	AuthenticationConfig *AuthenticationConfig `json:"authentication_config"`

	RateLimitConfig *RateLimitConfig `json:"rate_limit_config"`

	MetricsConfig *MetricsConfig `json:"metrics_config"`
}

// CircuitBreakerConfig configures circuit breaker behavior.

type CircuitBreakerConfig struct {
	MaxRequests uint32 `json:"max_requests" validate:"min=1"`

	Interval time.Duration `json:"interval" validate:"min=1s"`

	Timeout time.Duration `json:"timeout" validate:"min=1s"`

	ReadyToTrip func(counts Counts) bool `json:"-"`

	OnStateChange func(name string, from, to State) `json:"-"`
}

// ValidationConfig configures JSON schema validation.

type ValidationConfig struct {
	EnableSchemaValidation bool `json:"enable_schema_validation"`

	StrictValidation bool `json:"strict_validation"`

	ValidateAdditionalFields bool `json:"validate_additional_fields"`
}

// AuthenticationConfig configures authentication mechanisms.

type AuthenticationConfig struct {
	Enabled bool `json:"enabled"`

	Method string `json:"method" validate:"oneof=basic bearer jwt oauth2"`

	JWTSecret string `json:"jwt_secret,omitempty"`

	TokenValidation bool `json:"token_validation"`

	AllowedIssuers []string `json:"allowed_issuers,omitempty"`

	RequiredClaims map[string]string `json:"required_claims,omitempty"`
}

// RateLimitConfig configures rate limiting.

type RateLimitConfig struct {
	Enabled bool `json:"enabled"`

	RequestsPerMin int `json:"requests_per_min" validate:"min=1"`

	BurstSize int `json:"burst_size" validate:"min=1"`

	WindowSize time.Duration `json:"window_size" validate:"min=1s"`
}

// MetricsConfig configures metrics collection.

type MetricsConfig struct {
	Enabled bool `json:"enabled"`

	Endpoint string `json:"endpoint"`

	Namespace string `json:"namespace"`

	Subsystem string `json:"subsystem"`
}

// Circuit breaker state definitions.

type State int

const (

	// StateClosed holds stateclosed value.

	StateClosed State = iota

	// StateHalfOpen holds statehalfopen value.

	StateHalfOpen

	// StateOpen holds stateopen value.

	StateOpen
)

// Counts holds the numbers of requests and their results.

type Counts struct {
	Requests uint32

	TotalSuccesses uint32

	TotalFailures uint32

	ConsecutiveSuccesses uint32

	ConsecutiveFailures uint32
}

// A1Service defines the core A1 service interface.

type A1Service interface {
	// A1-P Policy Interface Methods.

	GetPolicyTypes(ctx context.Context) ([]int, error)

	GetPolicyType(ctx context.Context, policyTypeID int) (*PolicyType, error)

	CreatePolicyType(ctx context.Context, policyTypeID int, policyType *PolicyType) error

	DeletePolicyType(ctx context.Context, policyTypeID int) error

	GetPolicyInstances(ctx context.Context, policyTypeID int) ([]string, error)

	GetPolicyInstance(ctx context.Context, policyTypeID int, policyID string) (*PolicyInstance, error)

	CreatePolicyInstance(ctx context.Context, policyTypeID int, policyID string, instance *PolicyInstance) error

	UpdatePolicyInstance(ctx context.Context, policyTypeID int, policyID string, instance *PolicyInstance) error

	DeletePolicyInstance(ctx context.Context, policyTypeID int, policyID string) error

	GetPolicyStatus(ctx context.Context, policyTypeID int, policyID string) (*PolicyStatus, error)

	// A1-C Consumer Interface Methods.

	RegisterConsumer(ctx context.Context, consumerID string, info *ConsumerInfo) error

	UnregisterConsumer(ctx context.Context, consumerID string) error

	GetConsumer(ctx context.Context, consumerID string) (*ConsumerInfo, error)

	ListConsumers(ctx context.Context) ([]*ConsumerInfo, error)

	NotifyConsumer(ctx context.Context, consumerID string, notification *PolicyNotification) error

	// A1-EI Enrichment Information Interface Methods.

	GetEITypes(ctx context.Context) ([]string, error)

	GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error)

	CreateEIType(ctx context.Context, eiTypeID string, eiType *EnrichmentInfoType) error

	DeleteEIType(ctx context.Context, eiTypeID string) error

	GetEIJobs(ctx context.Context, eiTypeID string) ([]string, error)

	GetEIJob(ctx context.Context, eiJobID string) (*EnrichmentInfoJob, error)

	CreateEIJob(ctx context.Context, eiJobID string, job *EnrichmentInfoJob) error

	UpdateEIJob(ctx context.Context, eiJobID string, job *EnrichmentInfoJob) error

	DeleteEIJob(ctx context.Context, eiJobID string) error

	GetEIJobStatus(ctx context.Context, eiJobID string) (*EnrichmentInfoJobStatus, error)
}

// A1Handler defines the HTTP handler interface for A1 endpoints.

type A1Handler interface {
	// A1-P Policy Handler Methods.

	HandleGetPolicyTypes(w http.ResponseWriter, r *http.Request)

	HandleGetPolicyType(w http.ResponseWriter, r *http.Request)

	HandleCreatePolicyType(w http.ResponseWriter, r *http.Request)

	HandleDeletePolicyType(w http.ResponseWriter, r *http.Request)

	HandleGetPolicyInstances(w http.ResponseWriter, r *http.Request)

	HandleGetPolicyInstance(w http.ResponseWriter, r *http.Request)

	HandleCreatePolicyInstance(w http.ResponseWriter, r *http.Request)

	HandleUpdatePolicyInstance(w http.ResponseWriter, r *http.Request)

	HandleDeletePolicyInstance(w http.ResponseWriter, r *http.Request)

	HandleGetPolicyStatus(w http.ResponseWriter, r *http.Request)

	// A1-C Consumer Handler Methods.

	HandleRegisterConsumer(w http.ResponseWriter, r *http.Request)

	HandleUnregisterConsumer(w http.ResponseWriter, r *http.Request)

	HandleGetConsumer(w http.ResponseWriter, r *http.Request)

	HandleListConsumers(w http.ResponseWriter, r *http.Request)

	// A1-EI Enrichment Information Handler Methods.

	HandleGetEITypes(w http.ResponseWriter, r *http.Request)

	HandleGetEIType(w http.ResponseWriter, r *http.Request)

	HandleCreateEIType(w http.ResponseWriter, r *http.Request)

	HandleDeleteEIType(w http.ResponseWriter, r *http.Request)

	HandleGetEIJobs(w http.ResponseWriter, r *http.Request)

	HandleGetEIJob(w http.ResponseWriter, r *http.Request)

	HandleCreateEIJob(w http.ResponseWriter, r *http.Request)

	HandleUpdateEIJob(w http.ResponseWriter, r *http.Request)

	HandleDeleteEIJob(w http.ResponseWriter, r *http.Request)

	HandleGetEIJobStatus(w http.ResponseWriter, r *http.Request)
}

// ValidationResult represents the result of a validation operation.

type ValidationResult struct {
	Valid bool `json:"valid"`

	Errors []ValidationError `json:"errors,omitempty"`

	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationError represents a validation error with detailed information.

type ValidationError struct {
	Field string `json:"field"`

	Value interface{} `json:"value,omitempty"`

	Tag string `json:"tag"`

	Message string `json:"message"`

	Param string `json:"param,omitempty"`

	StructName string `json:"struct_name,omitempty"`

	FieldPath string `json:"field_path,omitempty"`
}

// ValidationWarning represents a validation warning.

type ValidationWarning struct {
	Field string `json:"field"`

	Value interface{} `json:"value,omitempty"`

	Message string `json:"message"`

	FieldPath string `json:"field_path,omitempty"`
}

// A1Validator defines the validation interface for A1 requests.

type A1Validator interface {
	ValidatePolicyType(policyType *PolicyType) *ValidationResult

	ValidatePolicyInstance(policyTypeID int, instance *PolicyInstance) *ValidationResult

	ValidateConsumerInfo(info *ConsumerInfo) *ValidationResult

	ValidateEnrichmentInfoType(eiType *EnrichmentInfoType) *ValidationResult

	ValidateEnrichmentInfoJob(job *EnrichmentInfoJob) *ValidationResult
}

// A1Storage defines the storage interface for A1 data persistence.

type A1Storage interface {
	// Policy Type Storage.

	StorePolicyType(ctx context.Context, policyType *PolicyType) error

	GetPolicyType(ctx context.Context, policyTypeID int) (*PolicyType, error)

	ListPolicyTypes(ctx context.Context) ([]*PolicyType, error)

	DeletePolicyType(ctx context.Context, policyTypeID int) error

	// Policy Instance Storage.

	StorePolicyInstance(ctx context.Context, instance *PolicyInstance) error

	GetPolicyInstance(ctx context.Context, policyTypeID int, policyID string) (*PolicyInstance, error)

	ListPolicyInstances(ctx context.Context, policyTypeID int) ([]*PolicyInstance, error)

	DeletePolicyInstance(ctx context.Context, policyTypeID int, policyID string) error

	UpdatePolicyInstanceStatus(ctx context.Context, policyTypeID int, policyID string, status *PolicyStatus) error

	// Consumer Storage.

	StoreConsumer(ctx context.Context, consumer *ConsumerInfo) error

	GetConsumer(ctx context.Context, consumerID string) (*ConsumerInfo, error)

	ListConsumers(ctx context.Context) ([]*ConsumerInfo, error)

	DeleteConsumer(ctx context.Context, consumerID string) error

	// EI Type Storage.

	StoreEIType(ctx context.Context, eiType *EnrichmentInfoType) error

	GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error)

	ListEITypes(ctx context.Context) ([]*EnrichmentInfoType, error)

	DeleteEIType(ctx context.Context, eiTypeID string) error

	// EI Job Storage.

	StoreEIJob(ctx context.Context, job *EnrichmentInfoJob) error

	GetEIJob(ctx context.Context, eiJobID string) (*EnrichmentInfoJob, error)

	ListEIJobs(ctx context.Context, eiTypeID string) ([]*EnrichmentInfoJob, error)

	DeleteEIJob(ctx context.Context, eiJobID string) error

	UpdateEIJobStatus(ctx context.Context, eiJobID string, status *EnrichmentInfoJobStatus) error
}

// A1Metrics defines metrics collection interface.

type A1Metrics interface {
	IncrementRequestCount(interface_ A1Interface, method string, statusCode int)

	RecordRequestDuration(interface_ A1Interface, method string, duration time.Duration)

	RecordPolicyCount(policyTypeID, instanceCount int)

	RecordConsumerCount(consumerCount int)

	RecordEIJobCount(eiTypeID string, jobCount int)

	RecordCircuitBreakerState(name string, state State)

	RecordValidationErrors(interface_ A1Interface, errorType string)
}

// RequestContext holds request-specific context information.

type RequestContext struct {
	RequestID string `json:"request_id"`

	UserAgent string `json:"user_agent,omitempty"`

	RemoteAddr string `json:"remote_addr,omitempty"`

	Method string `json:"method"`

	Path string `json:"path"`

	Headers map[string][]string `json:"headers,omitempty"`

	QueryParams map[string][]string `json:"query_params,omitempty"`

	Authentication json.RawMessage `json:"authentication,omitempty"`

	StartTime time.Time `json:"start_time"`
}

// ResponseInfo holds response information for logging.

type ResponseInfo struct {
	StatusCode int `json:"status_code"`

	ContentLength int64 `json:"content_length"`

	Duration time.Duration `json:"duration"`

	Headers map[string][]string `json:"headers,omitempty"`
}

// HealthCheck represents health status information.

type HealthCheck struct {
	Status string `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`

	Timestamp time.Time `json:"timestamp"`

	Version string `json:"version,omitempty"`

	Uptime time.Duration `json:"uptime,omitempty"`

	Components json.RawMessage `json:"components,omitempty"`

	Checks []ComponentCheck `json:"checks,omitempty"`
}

// ComponentCheck represents individual component health status.

type ComponentCheck struct {
	Name string `json:"name" validate:"required"`

	Status string `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`

	Message string `json:"message,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Duration time.Duration `json:"duration,omitempty"`

	Details json.RawMessage `json:"details,omitempty"`
}

// Common HTTP status codes used in A1 interface.

const (

	// StatusOK holds statusok value.

	StatusOK = http.StatusOK // 200

	// StatusCreated holds statuscreated value.

	StatusCreated = http.StatusCreated // 201

	// StatusAccepted holds statusaccepted value.

	StatusAccepted = http.StatusAccepted // 202

	// StatusNoContent holds statusnocontent value.

	StatusNoContent = http.StatusNoContent // 204

	// StatusBadRequest holds statusbadrequest value.

	StatusBadRequest = http.StatusBadRequest // 400

	// StatusUnauthorized holds statusunauthorized value.

	StatusUnauthorized = http.StatusUnauthorized // 401

	// StatusForbidden holds statusforbidden value.

	StatusForbidden = http.StatusForbidden // 403

	// StatusNotFound holds statusnotfound value.

	StatusNotFound = http.StatusNotFound // 404

	// StatusMethodNotAllowed holds statusmethodnotallowed value.

	StatusMethodNotAllowed = http.StatusMethodNotAllowed // 405

	// StatusNotAcceptable holds statusnotacceptable value.

	StatusNotAcceptable = http.StatusNotAcceptable // 406

	// StatusConflict holds statusconflict value.

	StatusConflict = http.StatusConflict // 409

	// StatusInternalServerError holds statusinternalservererror value.

	StatusInternalServerError = http.StatusInternalServerError // 500

	// StatusNotImplemented holds statusnotimplemented value.

	StatusNotImplemented = http.StatusNotImplemented // 501

	// StatusServiceUnavailable holds statusserviceunavailable value.

	StatusServiceUnavailable = http.StatusServiceUnavailable // 503

)

// Predefined policy enforcement statuses per O-RAN specification.

const (

	// PolicyStatusNotEnforced holds policystatusnotenforced value.

	PolicyStatusNotEnforced = "NOT_ENFORCED"

	// PolicyStatusEnforced holds policystatusenforced value.

	PolicyStatusEnforced = "ENFORCED"

	// PolicyStatusUnknown holds policystatusunknown value.

	PolicyStatusUnknown = "UNKNOWN"
)

// Predefined EI job statuses per O-RAN specification.

const (

	// EIJobStatusEnabled holds eijobstatusenabled value.

	EIJobStatusEnabled = "ENABLED"

	// EIJobStatusDisabled holds eijobstatusdisabled value.

	EIJobStatusDisabled = "DISABLED"
)

// Predefined notification types per O-RAN specification.

const (

	// NotificationTypeCreate holds notificationtypecreate value.

	NotificationTypeCreate = "CREATE"

	// NotificationTypeUpdate holds notificationtypeupdate value.

	NotificationTypeUpdate = "UPDATE"

	// NotificationTypeDelete holds notificationtypedelete value.

	NotificationTypeDelete = "DELETE"
)

// Common MIME types for A1 interfaces.

const (

	// ContentTypeJSON holds contenttypejson value.

	ContentTypeJSON = "application/json"

	// ContentTypeProblemJSON holds contenttypeproblemjson value.

	ContentTypeProblemJSON = "application/problem+json"

	// ContentTypeTextPlain holds contenttypetextplain value.

	ContentTypeTextPlain = "text/plain"
)

// DefaultA1ServerConfig returns a default configuration for the A1 server.

func DefaultA1ServerConfig() *A1ServerConfig {
	return &A1ServerConfig{
		Port: 8080,

		Host: "0.0.0.0",

		TLSEnabled: false,

		ReadTimeout: 30 * time.Second,

		WriteTimeout: 30 * time.Second,

		IdleTimeout: 120 * time.Second,

		MaxHeaderBytes: 1 << 20, // 1MB

		EnableA1P: true,

		EnableA1C: true,

		EnableA1EI: true,

		CircuitBreakerConfig: &CircuitBreakerConfig{
			MaxRequests: 10,

			Interval: 60 * time.Second,

			Timeout: 30 * time.Second,
		},

		ValidationConfig: &ValidationConfig{
			EnableSchemaValidation: true,

			StrictValidation: false,

			ValidateAdditionalFields: true,
		},

		AuthenticationConfig: &AuthenticationConfig{
			Enabled: false,

			Method: "bearer",
		},

		RateLimitConfig: &RateLimitConfig{
			Enabled: false,

			RequestsPerMin: 1000,

			BurstSize: 100,

			WindowSize: time.Minute,
		},

		MetricsConfig: &MetricsConfig{
			Enabled: true,

			Endpoint: "/metrics",

			Namespace: "nephoran",

			Subsystem: "a1",
		},
	}
}

// MarshalJSON provides custom JSON marshaling for PolicyType.

func (pt *PolicyType) MarshalJSON() ([]byte, error) {
	// Alias represents a alias.

	type Alias PolicyType

	return json.Marshal(&struct {
		*Alias

		CreatedAt string `json:"created_at,omitempty"`

		ModifiedAt string `json:"modified_at,omitempty"`
	}{
		Alias: (*Alias)(pt),

		CreatedAt: pt.CreatedAt.Format(time.RFC3339),

		ModifiedAt: pt.ModifiedAt.Format(time.RFC3339),
	})
}

// UnmarshalJSON provides custom JSON unmarshaling for PolicyType.

func (pt *PolicyType) UnmarshalJSON(data []byte) error {
	// Alias represents a alias.

	type Alias PolicyType

	aux := &struct {
		*Alias

		CreatedAt string `json:"created_at,omitempty"`

		ModifiedAt string `json:"modified_at,omitempty"`
	}{
		Alias: (*Alias)(pt),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, aux.CreatedAt); err == nil {
			pt.CreatedAt = t
		}
	}

	if aux.ModifiedAt != "" {
		if t, err := time.Parse(time.RFC3339, aux.ModifiedAt); err == nil {
			pt.ModifiedAt = t
		}
	}

	return nil
}

// String returns a string representation of A1Interface.

func (ai A1Interface) String() string {
	return string(ai)
}

// String returns a string representation of A1Version.

func (av A1Version) String() string {
	return string(av)
}

// String returns a string representation of State.

func (s State) String() string {
	switch s {

	case StateClosed:

		return "CLOSED"

	case StateHalfOpen:

		return "HALF_OPEN"

	case StateOpen:

		return "OPEN"

	default:

		return "UNKNOWN"

	}
}

// IsHealthy returns true if the health check status is UP.

func (hc *HealthCheck) IsHealthy() bool {
	return hc.Status == "UP"
}

// IsDegraded returns true if the health check status is DEGRADED.

func (hc *HealthCheck) IsDegraded() bool {
	return hc.Status == "DEGRADED"
}
