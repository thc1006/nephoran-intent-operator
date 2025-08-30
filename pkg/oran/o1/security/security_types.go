// Package security implements O1 interface security types and authentication for O-RAN networks.
// This package provides comprehensive security primitives including access control, policy evaluation,
// certificate management, audit logging, and compliance reporting for O-RAN security frameworks.
// It supports SAML, OIDC, OAuth2 authentication providers and implements WG11 security specifications.


package security



import (

	"time"

)



// StreamFilter defines filtering criteria for streaming data.

type StreamFilter struct {

	// Filter criteria.

	AttributeFilters map[string]string `json:"attributeFilters,omitempty"`

	ValueRanges      map[string]Range  `json:"valueRanges,omitempty"`

	TimeRange        *TimeRange        `json:"timeRange,omitempty"`



	// Stream configuration.

	BufferSize    int           `json:"bufferSize,omitempty"`

	FlushInterval time.Duration `json:"flushInterval,omitempty"`

	MaxBatchSize  int           `json:"maxBatchSize,omitempty"`



	// Security settings.

	RequireAuth  bool       `json:"requireAuth"`

	AllowedRoles []string   `json:"allowedRoles,omitempty"`

	RateLimit    *RateLimit `json:"rateLimit,omitempty"`

}



// Range defines a numeric range filter.

type Range struct {

	Min *float64 `json:"min,omitempty"`

	Max *float64 `json:"max,omitempty"`

}



// TimeRange defines a time range filter.

type TimeRange struct {

	Start time.Time `json:"start"`

	End   time.Time `json:"end"`

}



// PushProvider interface for push notification providers.

type PushProvider interface {

	// SendPush sends a push notification.

	SendPush(token string, message *PushMessage) error

	// GetProviderName returns the provider name.

	GetProviderName() string

	// ValidateToken validates a push token.

	ValidateToken(token string) error

}



// PushMessage represents a push notification message.

type PushMessage struct {

	Title    string            `json:"title"`

	Body     string            `json:"body"`

	Data     map[string]string `json:"data,omitempty"`

	Badge    int               `json:"badge,omitempty"`

	Sound    string            `json:"sound,omitempty"`

	Priority string            `json:"priority,omitempty"`

}



// PushConfig defines push notification configuration.

type PushConfig struct {

	DefaultProvider string            `json:"defaultProvider"`

	Providers       map[string]string `json:"providers"` // provider name -> config key

	RetryAttempts   int               `json:"retryAttempts"`

	Timeout         time.Duration     `json:"timeout"`

	RateLimit       *RateLimit        `json:"rateLimit,omitempty"`

}



// RateLimit defines rate limiting configuration for push notifications.

type RateLimit struct {

	RequestsPerSecond float64       `json:"requestsPerSecond"`

	Burst             int           `json:"burst"`

	TimeWindow        time.Duration `json:"timeWindow"`

}



// SAMLProvider defines SAML authentication provider configuration.

type SAMLProvider struct {

	EntityID             string            `json:"entityID"`

	MetadataURL          string            `json:"metadataURL"`

	Certificate          string            `json:"certificate"`

	PrivateKey           string            `json:"privateKey"`

	AssertionConsumerURL string            `json:"assertionConsumerURL"`

	SingleLogoutURL      string            `json:"singleLogoutURL"`

	AttributeMapping     map[string]string `json:"attributeMapping"`

	SignAuthRequests     bool              `json:"signAuthRequests"`

	WantAssertionsSigned bool              `json:"wantAssertionsSigned"`

	SessionTimeout       time.Duration     `json:"sessionTimeout"`

	ClockSkewTolerance   time.Duration     `json:"clockSkewTolerance"`

}



// OIDCProvider defines OpenID Connect authentication provider configuration.

type OIDCProvider struct {

	Issuer                 string            `json:"issuer"`

	ClientID               string            `json:"clientID"`

	ClientSecret           string            `json:"clientSecret"`

	RedirectURL            string            `json:"redirectURL"`

	Scopes                 []string          `json:"scopes"`

	AuthURL                string            `json:"authURL"`

	TokenURL               string            `json:"tokenURL"`

	UserInfoURL            string            `json:"userInfoURL"`

	JWKSUrl                string            `json:"jwksUrl"`

	ClaimMapping           map[string]string `json:"claimMapping"`

	UserinfoEndpoint       string            `json:"userinfoEndpoint"`

	EndSessionEndpoint     string            `json:"endSessionEndpoint"`

	ResponseType           string            `json:"responseType"`

	ResponseMode           string            `json:"responseMode"`

	SkipIssuerVerification bool              `json:"skipIssuerVerification"`

}



// OAuth2Config defines OAuth2 authentication configuration.

type OAuth2Config struct {

	ClientID          string            `json:"clientID"`

	ClientSecret      string            `json:"clientSecret"`

	AuthURL           string            `json:"authURL"`

	TokenURL          string            `json:"tokenURL"`

	RedirectURL       string            `json:"redirectURL"`

	Scopes            []string          `json:"scopes"`

	State             string            `json:"state,omitempty"`

	AuthStyle         int               `json:"authStyle"`

	Endpoint          *OAuth2Endpoint   `json:"endpoint"`

	TokenType         string            `json:"tokenType"`

	AccessTokenLife   time.Duration     `json:"accessTokenLife"`

	RefreshTokenLife  time.Duration     `json:"refreshTokenLife"`

	AllowedGrantTypes []string          `json:"allowedGrantTypes"`

	ExtraParams       map[string]string `json:"extraParams,omitempty"`

}



// OAuth2Endpoint defines OAuth2 endpoint configuration.

type OAuth2Endpoint struct {

	AuthURL     string            `json:"authURL"`

	TokenURL    string            `json:"tokenURL"`

	AuthStyle   int               `json:"authStyle"`

	ExtraParams map[string]string `json:"extraParams,omitempty"`

}



// AccessControlRule defines an access control rule.

type AccessControlRule struct {

	ID          string            `json:"id"`

	Name        string            `json:"name"`

	Description string            `json:"description"`

	Subject     *Subject          `json:"subject"`              // Who this rule applies to

	Resource    *Resource         `json:"resource"`             // What resource is being accessed

	Action      string            `json:"action"`               // What action is being performed

	Effect      string            `json:"effect"`               // ALLOW or DENY

	Conditions  []*Condition      `json:"conditions,omitempty"` // Additional conditions

	Priority    int               `json:"priority"`             // Rule priority (higher number = higher priority)

	Enabled     bool              `json:"enabled"`

	CreatedAt   time.Time         `json:"createdAt"`

	UpdatedAt   time.Time         `json:"updatedAt"`

	Metadata    map[string]string `json:"metadata,omitempty"`

}



// Subject represents the entity the rule applies to.

type Subject struct {

	Type       string            `json:"type"` // user, group, role, service

	ID         string            `json:"id"`

	Attributes map[string]string `json:"attributes,omitempty"`

}



// Resource represents the resource being accessed.

type Resource struct {

	Type       string            `json:"type"` // endpoint, service, data

	ID         string            `json:"id"`

	Path       string            `json:"path,omitempty"`

	Attributes map[string]string `json:"attributes,omitempty"`

}



// Condition represents a rule condition.

type Condition struct {

	Type     string      `json:"type"` // time, location, attribute

	Field    string      `json:"field"`

	Operator string      `json:"operator"` // eq, ne, gt, lt, in, contains

	Value    interface{} `json:"value"`

	Values   []string    `json:"values,omitempty"`

}



// RotationPolicy defines key or certificate rotation policy.

type RotationPolicy struct {

	ID                   string               `json:"id"`

	Name                 string               `json:"name"`

	Description          string               `json:"description"`

	RotationInterval     time.Duration        `json:"rotationInterval"` // How often to rotate

	ExpiryWarning        time.Duration        `json:"expiryWarning"`    // Warning period before expiry

	GracePeriod          time.Duration        `json:"gracePeriod"`      // Grace period after rotation

	AutoRotation         bool                 `json:"autoRotation"`     // Enable automatic rotation

	NotificationChannels []string             `json:"notificationChannels,omitempty"`

	BackupCount          int                  `json:"backupCount"` // Number of old keys/certs to keep

	RotationConditions   []*RotationCondition `json:"rotationConditions,omitempty"`

	ApprovalRequired     bool                 `json:"approvalRequired"` // Require manual approval

	Approvers            []string             `json:"approvers,omitempty"`

	TestAfterRotation    bool                 `json:"testAfterRotation"` // Test functionality after rotation

	RollbackOnFailure    bool                 `json:"rollbackOnFailure"` // Rollback if post-rotation test fails

	Enabled              bool                 `json:"enabled"`

	CreatedAt            time.Time            `json:"createdAt"`

	UpdatedAt            time.Time            `json:"updatedAt"`

	Metadata             map[string]string    `json:"metadata,omitempty"`

}



// RotationCondition defines conditions that can trigger rotation.

type RotationCondition struct {

	Type        string            `json:"type"`                // expiry, compromise, policy_change, manual

	Threshold   interface{}       `json:"threshold,omitempty"` // Threshold value for condition

	Description string            `json:"description"`

	Enabled     bool              `json:"enabled"`

	Metadata    map[string]string `json:"metadata,omitempty"`

}



// SecurityPolicyEngine defines the interface for security policy evaluation.

type SecurityPolicyEngine interface {

	// EvaluatePolicy evaluates a security policy against a request.

	EvaluatePolicy(request *PolicyRequest) (*PolicyDecision, error)



	// AddPolicy adds a new security policy.

	AddPolicy(policy *SecurityPolicy) error



	// RemovePolicy removes a security policy by ID.

	RemovePolicy(policyID string) error



	// UpdatePolicy updates an existing security policy.

	UpdatePolicy(policy *SecurityPolicy) error



	// GetPolicy retrieves a security policy by ID.

	GetPolicy(policyID string) (*SecurityPolicy, error)



	// ListPolicies returns all security policies with optional filtering.

	ListPolicies(filter *PolicyFilter) ([]*SecurityPolicy, error)

}



// SecurityPolicy represents a comprehensive security policy.

type SecurityPolicy struct {

	ID             string                 `json:"id"`

	Name           string                 `json:"name"`

	Description    string                 `json:"description"`

	Version        string                 `json:"version"`

	Type           string                 `json:"type"`     // access_control, encryption, audit, authentication

	Category       string                 `json:"category"` // technical, administrative, physical

	Scope          []string               `json:"scope"`    // What this policy applies to

	Rules          []*AccessControlRule   `json:"rules"`

	Enforcement    string                 `json:"enforcement"` // strict, permissive, monitoring

	Priority       int                    `json:"priority"`

	ValidFrom      time.Time              `json:"validFrom"`

	ValidUntil     time.Time              `json:"validUntil,omitempty"`

	ApprovalStatus string                 `json:"approvalStatus"` // draft, approved, deprecated

	ApprovedBy     string                 `json:"approvedBy,omitempty"`

	ApprovedAt     time.Time              `json:"approvedAt,omitempty"`

	EffectiveDate  time.Time              `json:"effectiveDate"`

	ReviewDate     time.Time              `json:"reviewDate,omitempty"`

	Owner          string                 `json:"owner"`

	Stakeholders   []string               `json:"stakeholders,omitempty"`

	ComplianceRef  []string               `json:"complianceRef,omitempty"` // References to compliance frameworks

	Exceptions     []*PolicyException     `json:"exceptions,omitempty"`

	AuditSettings  *AuditSettings         `json:"auditSettings,omitempty"`

	Enabled        bool                   `json:"enabled"`

	CreatedAt      time.Time              `json:"createdAt"`

	UpdatedAt      time.Time              `json:"updatedAt"`

	Metadata       map[string]interface{} `json:"metadata,omitempty"`

}



// PolicyException defines exceptions to a security policy.

type PolicyException struct {

	ID            string            `json:"id"`

	Description   string            `json:"description"`

	Justification string            `json:"justification"`

	Scope         *ExceptionScope   `json:"scope"`

	ValidUntil    time.Time         `json:"validUntil,omitempty"`

	ApprovedBy    string            `json:"approvedBy"`

	ApprovedAt    time.Time         `json:"approvedAt"`

	ReviewDate    time.Time         `json:"reviewDate,omitempty"`

	Enabled       bool              `json:"enabled"`

	Metadata      map[string]string `json:"metadata,omitempty"`

}



// ExceptionScope defines the scope of a policy exception.

type ExceptionScope struct {

	Subjects  []*Subject  `json:"subjects,omitempty"`

	Resources []*Resource `json:"resources,omitempty"`

	Actions   []string    `json:"actions,omitempty"`

	TimeRange *TimeRange  `json:"timeRange,omitempty"`

}



// TimeRange is already defined above.



// AuditSettings defines audit settings for a policy.

type AuditSettings struct {

	AuditLevel       string   `json:"auditLevel"`       // none, basic, detailed, full

	LogActions       []string `json:"logActions"`       // Which actions to log

	LogFailures      bool     `json:"logFailures"`      // Log policy violations

	LogSuccesses     bool     `json:"logSuccesses"`     // Log successful policy evaluations

	RetentionDays    int      `json:"retentionDays"`    // How long to keep audit logs

	AlertOnViolation bool     `json:"alertOnViolation"` // Send alerts on policy violations

}



// PolicyRequest represents a request for policy evaluation.

type PolicyRequest struct {

	Subject     *Subject               `json:"subject"`

	Resource    *Resource              `json:"resource"`

	Action      string                 `json:"action"`

	Context     *RequestContext        `json:"context,omitempty"`

	Timestamp   time.Time              `json:"timestamp"`

	RequestID   string                 `json:"requestID,omitempty"`

	SessionInfo map[string]interface{} `json:"sessionInfo,omitempty"`

}



// RequestContext provides additional context for policy evaluation.

type RequestContext struct {

	IPAddress   string                 `json:"ipAddress,omitempty"`

	UserAgent   string                 `json:"userAgent,omitempty"`

	Geolocation *Geolocation           `json:"geolocation,omitempty"`

	TimeOfDay   string                 `json:"timeOfDay,omitempty"`

	DayOfWeek   string                 `json:"dayOfWeek,omitempty"`

	NetworkZone string                 `json:"networkZone,omitempty"`

	DeviceInfo  *DeviceInfo            `json:"deviceInfo,omitempty"`

	TLSInfo     *TLSInfo               `json:"tlsInfo,omitempty"`

	Attributes  map[string]interface{} `json:"attributes,omitempty"`

}



// Geolocation represents geographic location information.

type Geolocation struct {

	Country   string  `json:"country,omitempty"`

	Region    string  `json:"region,omitempty"`

	City      string  `json:"city,omitempty"`

	Latitude  float64 `json:"latitude,omitempty"`

	Longitude float64 `json:"longitude,omitempty"`

	Accuracy  int     `json:"accuracy,omitempty"`

}



// DeviceInfo represents device information.

type DeviceInfo struct {

	DeviceID   string            `json:"deviceId,omitempty"`

	DeviceType string            `json:"deviceType,omitempty"`

	OS         string            `json:"os,omitempty"`

	OSVersion  string            `json:"osVersion,omitempty"`

	Browser    string            `json:"browser,omitempty"`

	AppVersion string            `json:"appVersion,omitempty"`

	Attributes map[string]string `json:"attributes,omitempty"`

}



// TLSInfo represents TLS connection information.

type TLSInfo struct {

	Version          string   `json:"version,omitempty"`

	CipherSuite      string   `json:"cipherSuite,omitempty"`

	ClientCert       string   `json:"clientCert,omitempty"`

	CertFingerprint  string   `json:"certFingerprint,omitempty"`

	CertSerialNumber string   `json:"certSerialNumber,omitempty"`

	CertIssuer       string   `json:"certIssuer,omitempty"`

	SNI              string   `json:"sni,omitempty"`

	ALPN             []string `json:"alpn,omitempty"`

}



// PolicyDecision represents the result of policy evaluation.

type PolicyDecision struct {

	Decision       string            `json:"decision"` // PERMIT, DENY, NOT_APPLICABLE, INDETERMINATE

	PolicyID       string            `json:"policyID,omitempty"`

	RuleID         string            `json:"ruleID,omitempty"`

	Reason         string            `json:"reason"`

	Details        []string          `json:"details,omitempty"`

	Obligations    []*Obligation     `json:"obligations,omitempty"`

	Advice         []*Advice         `json:"advice,omitempty"`

	Confidence     float64           `json:"confidence,omitempty"`

	EvaluationTime time.Duration     `json:"evaluationTime"`

	Timestamp      time.Time         `json:"timestamp"`

	Metadata       map[string]string `json:"metadata,omitempty"`

}



// Obligation represents an obligation that must be fulfilled.

type Obligation struct {

	ID          string                 `json:"id"`

	Type        string                 `json:"type"`

	Description string                 `json:"description"`

	Parameters  map[string]interface{} `json:"parameters,omitempty"`

	Mandatory   bool                   `json:"mandatory"`

	Deadline    time.Time              `json:"deadline,omitempty"`

}



// Advice represents advisory information from policy evaluation.

type Advice struct {

	ID          string                 `json:"id"`

	Type        string                 `json:"type"`

	Description string                 `json:"description"`

	Parameters  map[string]interface{} `json:"parameters,omitempty"`

	Priority    string                 `json:"priority"` // low, medium, high

}



// PolicyFilter defines filtering criteria for listing policies.

type PolicyFilter struct {

	Types          []string  `json:"types,omitempty"`

	Categories     []string  `json:"categories,omitempty"`

	Scopes         []string  `json:"scopes,omitempty"`

	Enabled        *bool     `json:"enabled,omitempty"`

	ValidAt        time.Time `json:"validAt,omitempty"`

	ApprovalStatus []string  `json:"approvalStatus,omitempty"`

	Owner          string    `json:"owner,omitempty"`

	Tags           []string  `json:"tags,omitempty"`

	Limit          int       `json:"limit,omitempty"`

	Offset         int       `json:"offset,omitempty"`

}



// NotificationChannel interface for different notification channels.

type NotificationChannel interface {

	// Send sends a notification.

	Send(message *NotificationMessage) error

	// GetChannelType returns the channel type.

	GetChannelType() string

	// IsEnabled checks if the channel is enabled.

	IsEnabled() bool

	// ValidateConfig validates the channel configuration.

	ValidateConfig() error

}



// NotificationMessage represents a notification message.

type NotificationMessage struct {

	ID         string                 `json:"id"`

	Subject    string                 `json:"subject"`

	Body       string                 `json:"body"`

	Priority   string                 `json:"priority"` // low, normal, high, urgent

	Type       string                 `json:"type"`     // info, warning, error, alert

	Data       map[string]interface{} `json:"data,omitempty"`

	Recipients []string               `json:"recipients,omitempty"`

	Timestamp  time.Time              `json:"timestamp"`

}



// EscalationRule defines escalation rules for notifications.

type EscalationRule struct {

	ID               string        `json:"id"`

	Name             string        `json:"name"`

	Description      string        `json:"description"`

	TriggerCondition *Condition    `json:"triggerCondition"`

	TimeThreshold    time.Duration `json:"timeThreshold"`

	EscalationLevel  int           `json:"escalationLevel"`

	Recipients       []string      `json:"recipients"`

	Actions          []string      `json:"actions"`

	Enabled          bool          `json:"enabled"`

	CreatedAt        time.Time     `json:"createdAt"`

	UpdatedAt        time.Time     `json:"updatedAt"`

}



// ValidationResult represents the result of security validation.

type ValidationResult struct {

	Valid    bool              `json:"valid"`

	Errors   []string          `json:"errors,omitempty"`

	Warnings []string          `json:"warnings,omitempty"`

	Details  map[string]string `json:"details,omitempty"`

}



// SecurityStatus represents overall security status.

type SecurityStatus struct {

	ComplianceLevel   string                 `json:"complianceLevel"`

	ActiveThreats     []string               `json:"activeThreats,omitempty"`

	LastAudit         time.Time              `json:"lastAudit"`

	NextAudit         time.Time              `json:"nextAudit,omitempty"`

	CertificateStatus *CertificateStatus     `json:"certificateStatus,omitempty"`

	PolicyStatus      *PolicyStatus          `json:"policyStatus,omitempty"`

	Metrics           map[string]interface{} `json:"metrics,omitempty"`

	Alerts            []*SecurityAlert       `json:"alerts,omitempty"`

	LastUpdated       time.Time              `json:"lastUpdated"`

}



// CertificateStatus represents certificate management status.

type CertificateStatus struct {

	TotalCertificates    int       `json:"totalCertificates"`

	ActiveCertificates   int       `json:"activeCertificates"`

	ExpiringCertificates int       `json:"expiringCertificates"`

	ExpiredCertificates  int       `json:"expiredCertificates"`

	RevokedCertificates  int       `json:"revokedCertificates"`

	NextExpiry           time.Time `json:"nextExpiry,omitempty"`

	HealthScore          float64   `json:"healthScore"`

}



// PolicyStatus represents security policy status.

type PolicyStatus struct {

	TotalPolicies   int       `json:"totalPolicies"`

	ActivePolicies  int       `json:"activePolicies"`

	PendingApproval int       `json:"pendingApproval"`

	ExpiredPolicies int       `json:"expiredPolicies"`

	ComplianceScore float64   `json:"complianceScore"`

	LastReview      time.Time `json:"lastReview"`

	NextReview      time.Time `json:"nextReview,omitempty"`

}



// SecurityAlert represents a security alert.

type SecurityAlert struct {

	ID          string                 `json:"id"`

	Type        string                 `json:"type"`

	Severity    string                 `json:"severity"` // low, medium, high, critical

	Title       string                 `json:"title"`

	Description string                 `json:"description"`

	Source      string                 `json:"source"`

	Target      string                 `json:"target,omitempty"`

	Status      string                 `json:"status"` // new, acknowledged, investigating, resolved

	CreatedAt   time.Time              `json:"createdAt"`

	UpdatedAt   time.Time              `json:"updatedAt"`

	ResolvedAt  time.Time              `json:"resolvedAt,omitempty"`

	AssignedTo  string                 `json:"assignedTo,omitempty"`

	Tags        []string               `json:"tags,omitempty"`

	Evidence    map[string]interface{} `json:"evidence,omitempty"`

	Actions     []*AlertAction         `json:"actions,omitempty"`

}



// AlertAction represents an action taken on a security alert.

type AlertAction struct {

	ID          string                 `json:"id"`

	Type        string                 `json:"type"` // investigate, escalate, resolve, dismiss

	Description string                 `json:"description"`

	TakenBy     string                 `json:"takenBy"`

	TakenAt     time.Time              `json:"takenAt"`

	Result      string                 `json:"result,omitempty"`

	Evidence    map[string]interface{} `json:"evidence,omitempty"`

}



// StatisticalSummary represents statistical summary data.

type StatisticalSummary struct {

	Count       int64                  `json:"count"`

	Mean        float64                `json:"mean"`

	Median      float64                `json:"median"`

	StdDev      float64                `json:"stdDev"`

	Min         float64                `json:"min"`

	Max         float64                `json:"max"`

	Percentiles map[string]float64     `json:"percentiles,omitempty"` // p50, p75, p90, p95, p99

	Histogram   map[string]int64       `json:"histogram,omitempty"`

	Metadata    map[string]interface{} `json:"metadata,omitempty"`

}



// AuditPolicy defines audit policy configuration.

type AuditPolicy struct {

	ID          string            `json:"id"`

	Name        string            `json:"name"`

	Description string            `json:"description"`

	EventTypes  []string          `json:"eventTypes"`

	Resources   []string          `json:"resources"`

	Actions     []string          `json:"actions"`

	Subjects    []string          `json:"subjects,omitempty"`

	Conditions  []*Condition      `json:"conditions,omitempty"`

	Level       string            `json:"level"` // basic, detailed, full

	Retention   time.Duration     `json:"retention"`

	Enabled     bool              `json:"enabled"`

	CreatedAt   time.Time         `json:"createdAt"`

	UpdatedAt   time.Time         `json:"updatedAt"`

	Metadata    map[string]string `json:"metadata,omitempty"`

}



// ExpirationPolicy defines expiration policy for security objects.

type ExpirationPolicy struct {

	ID                string            `json:"id"`

	Name              string            `json:"name"`

	Description       string            `json:"description"`

	ObjectType        string            `json:"objectType"` // certificate, key, token, session

	DefaultLifetime   time.Duration     `json:"defaultLifetime"`

	MaxLifetime       time.Duration     `json:"maxLifetime"`

	MinLifetime       time.Duration     `json:"minLifetime"`

	RenewalWindow     time.Duration     `json:"renewalWindow"`

	GracePeriod       time.Duration     `json:"gracePeriod"`

	AutoRenewal       bool              `json:"autoRenewal"`

	NotifyBefore      []time.Duration   `json:"notifyBefore"` // Notification schedule

	ExtensionAllowed  bool              `json:"extensionAllowed"`

	MaxExtensions     int               `json:"maxExtensions"`

	ExtensionDuration time.Duration     `json:"extensionDuration"`

	Enabled           bool              `json:"enabled"`

	CreatedAt         time.Time         `json:"createdAt"`

	UpdatedAt         time.Time         `json:"updatedAt"`

	Metadata          map[string]string `json:"metadata,omitempty"`

}



// EscrowPolicy defines key escrow policy.

type EscrowPolicy struct {

	ID                    string            `json:"id"`

	Name                  string            `json:"name"`

	Description           string            `json:"description"`

	KeyTypes              []string          `json:"keyTypes"` // Which key types require escrow

	EscrowRequired        bool              `json:"escrowRequired"`

	MinEscrowAgents       int               `json:"minEscrowAgents"`

	RequiredApprovals     int               `json:"requiredApprovals"`

	SplitKeyThreshold     int               `json:"splitKeyThreshold"` // Minimum shares needed to reconstruct

	TotalKeyShares        int               `json:"totalKeyShares"`    // Total number of key shares

	EscrowDuration        time.Duration     `json:"escrowDuration"`

	DestructionDelay      time.Duration     `json:"destructionDelay"` // Delay before destroying escrowed keys

	AuditRequired         bool              `json:"auditRequired"`

	ComplianceRequirement []string          `json:"complianceRequirement"`

	AccessConditions      []*Condition      `json:"accessConditions,omitempty"`

	NotificationChannels  []string          `json:"notificationChannels,omitempty"`

	Enabled               bool              `json:"enabled"`

	CreatedAt             time.Time         `json:"createdAt"`

	UpdatedAt             time.Time         `json:"updatedAt"`

	Metadata              map[string]string `json:"metadata,omitempty"`

}



// EscrowAccessRule defines rules for accessing escrowed keys.

type EscrowAccessRule struct {

	ID                    string            `json:"id"`

	Name                  string            `json:"name"`

	Description           string            `json:"description"`

	RequiredRoles         []string          `json:"requiredRoles"`

	ApprovalChain         []string          `json:"approvalChain"` // Order of required approvals

	JustificationRequired bool              `json:"justificationRequired"`

	TimeRestrictions      *TimeRestriction  `json:"timeRestrictions,omitempty"`

	AuditTrail            bool              `json:"auditTrail"`

	Conditions            []*Condition      `json:"conditions,omitempty"`

	Enabled               bool              `json:"enabled"`

	CreatedAt             time.Time         `json:"createdAt"`

	UpdatedAt             time.Time         `json:"updatedAt"`

	Metadata              map[string]string `json:"metadata,omitempty"`

}



// TimeRestriction defines time-based access restrictions.

type TimeRestriction struct {

	AllowedDays     []string     `json:"allowedDays"`  // Monday, Tuesday, etc.

	AllowedHours    []string     `json:"allowedHours"` // 09:00-17:00 format

	Timezone        string       `json:"timezone"`

	BlackoutPeriods []*TimeRange `json:"blackoutPeriods,omitempty"`

}



// RetentionRule defines data retention rules.

type RetentionRule struct {

	ID              string            `json:"id"`

	Name            string            `json:"name"`

	Description     string            `json:"description"`

	DataType        string            `json:"dataType"` // audit_log, security_event, certificate, key

	RetentionPeriod time.Duration     `json:"retentionPeriod"`

	ArchivePeriod   time.Duration     `json:"archivePeriod,omitempty"`

	DeleteAfter     time.Duration     `json:"deleteAfter,omitempty"`

	Compression     bool              `json:"compression"`

	Encryption      bool              `json:"encryption"`

	BackupRequired  bool              `json:"backupRequired"`

	LegalHold       bool              `json:"legalHold"`

	ComplianceRefs  []string          `json:"complianceRefs,omitempty"`

	Conditions      []*Condition      `json:"conditions,omitempty"`

	Enabled         bool              `json:"enabled"`

	CreatedAt       time.Time         `json:"createdAt"`

	UpdatedAt       time.Time         `json:"updatedAt"`

	Metadata        map[string]string `json:"metadata,omitempty"`

}



// EscrowApprover represents an authorized escrow approver.

type EscrowApprover struct {

	ID           string            `json:"id"`

	Name         string            `json:"name"`

	Email        string            `json:"email"`

	Role         string            `json:"role"`

	Department   string            `json:"department,omitempty"`

	Authority    []string          `json:"authority"` // What they can approve

	RequireMFA   bool              `json:"requireMFA"`

	Active       bool              `json:"active"`

	LastActivity time.Time         `json:"lastActivity,omitempty"`

	CreatedAt    time.Time         `json:"createdAt"`

	UpdatedAt    time.Time         `json:"updatedAt"`

	Metadata     map[string]string `json:"metadata,omitempty"`

}



// EscrowAccessRequest represents a request to access escrowed keys.

type EscrowAccessRequest struct {

	ID              string            `json:"id"`

	KeyID           string            `json:"keyID"`

	RequestedBy     string            `json:"requestedBy"`

	Justification   string            `json:"justification"`

	RequestedAt     time.Time         `json:"requestedAt"`

	RequiredBy      time.Time         `json:"requiredBy,omitempty"`

	Status          string            `json:"status"` // pending, approved, denied, completed, expired

	ApprovalChain   []*ApprovalStep   `json:"approvalChain"`

	AuditTrail      []*AuditEntry     `json:"auditTrail"`

	AccessGrantedAt time.Time         `json:"accessGrantedAt,omitempty"`

	AccessExpiry    time.Time         `json:"accessExpiry,omitempty"`

	Metadata        map[string]string `json:"metadata,omitempty"`

}



// ApprovalStep represents a step in the approval chain.

type ApprovalStep struct {

	ID           string    `json:"id"`

	ApproverID   string    `json:"approverID"`

	ApproverName string    `json:"approverName"`

	Status       string    `json:"status"` // pending, approved, denied

	Decision     string    `json:"decision,omitempty"`

	Comments     string    `json:"comments,omitempty"`

	DecisionAt   time.Time `json:"decisionAt,omitempty"`

	Order        int       `json:"order"` // Order in approval chain

}



// AuditEntry represents an audit log entry (minimal definition for compilation).

type AuditEntry struct {

	ID        string                 `json:"id"`

	Timestamp time.Time              `json:"timestamp"`

	Event     string                 `json:"event"`

	Subject   string                 `json:"subject"`

	Object    string                 `json:"object"`

	Result    string                 `json:"result"`

	Details   map[string]interface{} `json:"details,omitempty"`

}



// ComplianceReportGenerator interface for generating compliance reports.

type ComplianceReportGenerator interface {

	GenerateReport(assessment *ComplianceAssessment) (*ComplianceReport, error)

	GetSupportedFormats() []string

	ValidateAssessment(assessment *ComplianceAssessment) error

}



// ComplianceAssessment represents a compliance assessment (minimal definition).

type ComplianceAssessment struct {

	ID             string                 `json:"id"`

	Framework      string                 `json:"framework"`

	AssessmentType string                 `json:"assessmentType"`

	Status         string                 `json:"status"`

	StartDate      time.Time              `json:"startDate"`

	EndDate        time.Time              `json:"endDate"`

	OverallScore   float64                `json:"overallScore"`

	Findings       []*ComplianceFinding   `json:"findings,omitempty"`

	Evidence       []*ComplianceEvidence  `json:"evidence,omitempty"`

	Metadata       map[string]interface{} `json:"metadata,omitempty"`

}



// ComplianceFinding represents a compliance finding (minimal definition).

type ComplianceFinding struct {

	ID          string    `json:"id"`

	Type        string    `json:"type"`

	Severity    string    `json:"severity"`

	Description string    `json:"description"`

	CreatedAt   time.Time `json:"createdAt"`

}



// ComplianceEvidence represents compliance evidence (minimal definition).

type ComplianceEvidence struct {

	ID          string    `json:"id"`

	Type        string    `json:"type"`

	Description string    `json:"description"`

	Source      string    `json:"source"`

	CollectedAt time.Time `json:"collectedAt"`

}



// ComplianceReport represents a compliance report (minimal definition).

type ComplianceReport struct {

	ID               string                 `json:"id"`

	Framework        string                 `json:"framework"`

	GeneratedAt      time.Time              `json:"generatedAt"`

	OverallScore     float64                `json:"overallScore"`

	ExecutiveSummary string                 `json:"executiveSummary"`

	Content          []byte                 `json:"content"`

	Format           string                 `json:"format"`

	Metadata         map[string]interface{} `json:"metadata,omitempty"`

}



// VPNConfig defines VPN configuration.

type VPNConfig struct {

	DefaultProfile    string            `json:"defaultProfile"`

	MaxConnections    int               `json:"maxConnections"`

	ConnectionTimeout time.Duration     `json:"connectionTimeout"`

	KeepAliveInterval time.Duration     `json:"keepAliveInterval"`

	RetryAttempts     int               `json:"retryAttempts"`

	RetryInterval     time.Duration     `json:"retryInterval"`

	LogLevel          string            `json:"logLevel"`

	EnableCompression bool              `json:"enableCompression"`

	DNSServers        []string          `json:"dnsServers,omitempty"`

	Routes            []string          `json:"routes,omitempty"`

	Metadata          map[string]string `json:"metadata,omitempty"`

}



// TunnelConfig defines secure tunnel configuration.

type TunnelConfig struct {

	MaxTunnels        int               `json:"maxTunnels"`

	DefaultTimeout    time.Duration     `json:"defaultTimeout"`

	KeepAliveInterval time.Duration     `json:"keepAliveInterval"`

	BufferSize        int               `json:"bufferSize"`

	LogConnections    bool              `json:"logConnections"`

	AllowedPorts      []int             `json:"allowedPorts,omitempty"`

	BlockedPorts      []int             `json:"blockedPorts,omitempty"`

	TLSConfig         *TLSConfig        `json:"tlsConfig,omitempty"`

	Metadata          map[string]string `json:"metadata,omitempty"`

}



// TLSConfig defines TLS configuration (minimal definition to avoid conflicts).

type TLSConfig struct {

	MinVersion   uint16   `json:"minVersion"`

	MaxVersion   uint16   `json:"maxVersion"`

	CipherSuites []uint16 `json:"cipherSuites,omitempty"`

	CertFile     string   `json:"certFile,omitempty"`

	KeyFile      string   `json:"keyFile,omitempty"`

	CAFile       string   `json:"caFile,omitempty"`

	SkipVerify   bool     `json:"skipVerify"`

}



// ScanScheduler interface for scheduling scans.

type ScanScheduler interface {

	ScheduleScan(scanID string, schedule *ScanSchedule) error

	UnscheduleScan(scanID string) error

	GetScheduledScans() []*ScheduledScan

	Start() error

	Stop() error

}



// ScanSchedule defines scan scheduling configuration.

type ScanSchedule struct {

	ScannerID string    `json:"scannerID"`

	Frequency string    `json:"frequency"` // DAILY, WEEKLY, MONTHLY, CRON

	Time      string    `json:"time"`      // Time of day or cron expression

	Timezone  string    `json:"timezone"`

	Enabled   bool      `json:"enabled"`

	LastRun   time.Time `json:"lastRun,omitempty"`

	NextRun   time.Time `json:"nextRun,omitempty"`

	RunCount  int64     `json:"runCount"`

}



// ScheduledScan represents a scheduled scan.

type ScheduledScan struct {

	ID       string            `json:"id"`

	Schedule *ScanSchedule     `json:"schedule"`

	Status   string            `json:"status"` // active, paused, disabled

	Metadata map[string]string `json:"metadata,omitempty"`

}



// VulnReportTemplate defines vulnerability report templates.

type VulnReportTemplate struct {

	ID            string            `json:"id"`

	Name          string            `json:"name"`

	Description   string            `json:"description"`

	Format        string            `json:"format"`   // HTML, PDF, JSON, XML

	Template      string            `json:"template"` // Template content

	Sections      []*ReportSection  `json:"sections"`

	StyleSheet    string            `json:"styleSheet,omitempty"`

	IncludeCharts bool              `json:"includeCharts"`

	IncludeTrends bool              `json:"includeTrends"`

	Enabled       bool              `json:"enabled"`

	CreatedAt     time.Time         `json:"createdAt"`

	UpdatedAt     time.Time         `json:"updatedAt"`

	Metadata      map[string]string `json:"metadata,omitempty"`

}



// ReportSection represents a section in a report template.

type ReportSection struct {

	ID       string `json:"id"`

	Title    string `json:"title"`

	Content  string `json:"content"`

	Type     string `json:"type"` // text, chart, table, list

	Required bool   `json:"required"`

	Order    int    `json:"order"`

}



// VulnReportGenerator interface for generating vulnerability reports.

type VulnReportGenerator interface {

	GenerateReport(template *VulnReportTemplate, data interface{}) (*VulnReport, error)

	GetSupportedFormats() []string

	ValidateTemplate(template *VulnReportTemplate) error

}



// VulnReport represents a generated vulnerability report.

type VulnReport struct {

	ID          string                 `json:"id"`

	TemplateID  string                 `json:"templateID"`

	Title       string                 `json:"title"`

	GeneratedAt time.Time              `json:"generatedAt"`

	Format      string                 `json:"format"`

	Content     []byte                 `json:"content"`

	Summary     *VulnReportSummary     `json:"summary"`

	Metadata    map[string]interface{} `json:"metadata,omitempty"`

}



// VulnReportSummary provides summary information for vulnerability reports.

type VulnReportSummary struct {

	TotalVulnerabilities int                `json:"totalVulnerabilities"`

	SeverityDistribution map[string]int     `json:"severityDistribution"`

	TopVulnerabilities   []string           `json:"topVulnerabilities,omitempty"`

	AffectedSystems      int                `json:"affectedSystems"`

	NewVulnerabilities   int                `json:"newVulnerabilities"`

	ResolvedSince        int                `json:"resolvedSince"`

	TrendData            map[string]float64 `json:"trendData,omitempty"`

}



// VulnReportDistributor interface for distributing vulnerability reports.

type VulnReportDistributor interface {

	DistributeReport(report *VulnReport, recipients []string) error

	GetSupportedChannels() []string

	ValidateRecipients(recipients []string) error

}



// ResponseChecklist represents a checklist for incident response.

type ResponseChecklist struct {

	ID            string            `json:"id"`

	Name          string            `json:"name"`

	Description   string            `json:"description"`

	Items         []*ChecklistItem  `json:"items"`

	Category      string            `json:"category"` // preparation, detection, containment, eradication, recovery

	Priority      string            `json:"priority"` // low, medium, high, critical

	Mandatory     bool              `json:"mandatory"`

	Owner         string            `json:"owner,omitempty"`

	EstimatedTime time.Duration     `json:"estimatedTime,omitempty"`

	Dependencies  []string          `json:"dependencies,omitempty"` // Other checklist IDs that must complete first

	CreatedAt     time.Time         `json:"createdAt"`

	UpdatedAt     time.Time         `json:"updatedAt"`

	Metadata      map[string]string `json:"metadata,omitempty"`

}



// ChecklistItem represents a single item in a response checklist.

type ChecklistItem struct {

	ID            string            `json:"id"`

	Title         string            `json:"title"`

	Description   string            `json:"description"`

	Instructions  string            `json:"instructions"`

	Type          string            `json:"type"`   // action, verification, decision, documentation

	Status        string            `json:"status"` // pending, in_progress, completed, skipped, failed

	Assignee      string            `json:"assignee,omitempty"`

	EstimatedTime time.Duration     `json:"estimatedTime,omitempty"`

	ActualTime    time.Duration     `json:"actualTime,omitempty"`

	StartedAt     time.Time         `json:"startedAt,omitempty"`

	CompletedAt   time.Time         `json:"completedAt,omitempty"`

	Evidence      []string          `json:"evidence,omitempty"` // URLs, file paths, or descriptions of evidence

	Notes         string            `json:"notes,omitempty"`

	Prerequisites []string          `json:"prerequisites,omitempty"` // Other item IDs that must complete first

	Verification  *VerificationStep `json:"verification,omitempty"`

	Order         int               `json:"order"` // Order within the checklist

	Metadata      map[string]string `json:"metadata,omitempty"`

}



// VerificationStep represents a verification step for a checklist item.

type VerificationStep struct {

	ID         string    `json:"id"`

	Type       string    `json:"type"` // automated, manual, approval

	Criteria   string    `json:"criteria"`

	Method     string    `json:"method,omitempty"`

	VerifiedBy string    `json:"verifiedBy,omitempty"`

	VerifiedAt time.Time `json:"verifiedAt,omitempty"`

	Result     string    `json:"result,omitempty"` // pass, fail, pending

	Comments   string    `json:"comments,omitempty"`

}

