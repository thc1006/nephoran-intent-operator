// Package abstraction provides a universal interface for service mesh integration
package abstraction

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceMeshProvider defines the type of service mesh
type ServiceMeshProvider string

const (
	// ProviderIstio represents Istio service mesh
	ProviderIstio ServiceMeshProvider = "istio"
	// ProviderLinkerd represents Linkerd service mesh
	ProviderLinkerd ServiceMeshProvider = "linkerd"
	// ProviderConsul represents Consul Connect service mesh
	ProviderConsul ServiceMeshProvider = "consul"
	// ProviderNone represents no service mesh
	ProviderNone ServiceMeshProvider = "none"
)

// ServiceMeshInterface defines the universal interface for service mesh operations
type ServiceMeshInterface interface {
	// Initialize sets up the service mesh provider
	Initialize(ctx context.Context, config *ServiceMeshConfig) error

	// Certificate Management
	GetCertificateProvider() CertificateProvider
	RotateCertificates(ctx context.Context, namespace string) error
	ValidateCertificateChain(ctx context.Context, namespace string) error

	// Policy Management
	ApplyMTLSPolicy(ctx context.Context, policy *MTLSPolicy) error
	ApplyAuthorizationPolicy(ctx context.Context, policy *AuthorizationPolicy) error
	ApplyTrafficPolicy(ctx context.Context, policy *TrafficPolicy) error
	ValidatePolicies(ctx context.Context, namespace string) (*PolicyValidationResult, error)

	// Service Management
	RegisterService(ctx context.Context, service *ServiceRegistration) error
	UnregisterService(ctx context.Context, serviceName string, namespace string) error
	GetServiceStatus(ctx context.Context, serviceName string, namespace string) (*ServiceStatus, error)

	// Observability
	GetMetrics() []prometheus.Collector
	GetServiceDependencies(ctx context.Context, namespace string) (*DependencyGraph, error)
	GetMTLSStatus(ctx context.Context, namespace string) (*MTLSStatusReport, error)

	// Health and Readiness
	IsHealthy(ctx context.Context) error
	IsReady(ctx context.Context) error

	// Provider Information
	GetProvider() ServiceMeshProvider
	GetVersion() string
	GetCapabilities() []Capability
}

// ServiceMeshConfig contains configuration for service mesh initialization
type ServiceMeshConfig struct {
	Provider            ServiceMeshProvider    `json:"provider"`
	Namespace           string                 `json:"namespace"`
	ControlPlaneURL     string                 `json:"controlPlaneUrl,omitempty"`
	TrustDomain         string                 `json:"trustDomain"`
	CertificateConfig   *CertificateConfig     `json:"certificateConfig"`
	PolicyDefaults      *PolicyDefaults        `json:"policyDefaults"`
	ObservabilityConfig *ObservabilityConfig   `json:"observabilityConfig"`
	MultiCluster        *MultiClusterConfig    `json:"multiCluster,omitempty"`
	CustomConfig        map[string]interface{} `json:"customConfig,omitempty"`
}

// CertificateConfig defines certificate management configuration
type CertificateConfig struct {
	RootCAFile        string        `json:"rootCaFile"`
	CertLifetime      time.Duration `json:"certLifetime"`
	RotationThreshold float64       `json:"rotationThreshold"` // Percentage of lifetime before rotation
	SPIFFEEnabled     bool          `json:"spiffeEnabled"`
	SPIREServerURL    string        `json:"spireServerUrl,omitempty"`
	CertStorage       string        `json:"certStorage"` // "secret", "configmap", "external"
}

// PolicyDefaults defines default policy configurations
type PolicyDefaults struct {
	MTLSMode              string `json:"mtlsMode"` // "STRICT", "PERMISSIVE", "DISABLE"
	DefaultDenyAll        bool   `json:"defaultDenyAll"`
	EnableNetworkPolicies bool   `json:"enableNetworkPolicies"`
	EnableRateLimiting    bool   `json:"enableRateLimiting"`
	DefaultTimeoutSeconds int    `json:"defaultTimeoutSeconds"`
	DefaultRetryAttempts  int    `json:"defaultRetryAttempts"`
}

// ObservabilityConfig defines observability configuration
type ObservabilityConfig struct {
	EnableTracing    bool   `json:"enableTracing"`
	TracingBackend   string `json:"tracingBackend"` // "jaeger", "zipkin", "datadog"
	EnableMetrics    bool   `json:"enableMetrics"`
	MetricsPort      int    `json:"metricsPort"`
	EnableAccessLogs bool   `json:"enableAccessLogs"`
	LogLevel         string `json:"logLevel"`
}

// MultiClusterConfig defines multi-cluster mesh configuration
type MultiClusterConfig struct {
	ClusterName    string            `json:"clusterName"`
	ClusterID      string            `json:"clusterId"`
	Network        string            `json:"network"`
	RemoteClusters []RemoteCluster   `json:"remoteClusters"`
	Federation     *FederationConfig `json:"federation,omitempty"`
}

// RemoteCluster defines a remote cluster in the mesh
type RemoteCluster struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Network     string `json:"network"`
	TrustDomain string `json:"trustDomain"`
}

// FederationConfig defines federation settings
type FederationConfig struct {
	Enabled          bool     `json:"enabled"`
	TrustDomains     []string `json:"trustDomains"`
	SharedGateway    string   `json:"sharedGateway"`
	CrossClusterAuth bool     `json:"crossClusterAuth"`
}

// MTLSPolicy defines an mTLS policy
type MTLSPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MTLSPolicySpec `json:"spec"`
}

// MTLSPolicySpec defines the specification for an mTLS policy
type MTLSPolicySpec struct {
	Selector      *LabelSelector `json:"selector"`
	Mode          string         `json:"mode"` // "STRICT", "PERMISSIVE", "DISABLE"
	PortLevelMTLS []PortMTLS     `json:"portLevelMtls,omitempty"`
	Exceptions    []string       `json:"exceptions,omitempty"` // Service names exempt from policy
}

// PortMTLS defines port-specific mTLS configuration
type PortMTLS struct {
	Port int    `json:"port"`
	Mode string `json:"mode"`
}

// AuthorizationPolicy defines service-to-service authorization
type AuthorizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthorizationPolicySpec `json:"spec"`
}

// AuthorizationPolicySpec defines the specification for authorization
type AuthorizationPolicySpec struct {
	Selector *LabelSelector      `json:"selector"`
	Action   string              `json:"action"` // "ALLOW", "DENY", "AUDIT", "CUSTOM"
	Rules    []AuthorizationRule `json:"rules,omitempty"`
	Provider *ExtensionProvider  `json:"provider,omitempty"`
}

// AuthorizationRule defines an authorization rule
type AuthorizationRule struct {
	From []SecuritySource    `json:"from,omitempty"`
	To   []SecurityOperation `json:"to,omitempty"`
	When []Condition         `json:"when,omitempty"`
}

// SecuritySource defines the source of a request
type SecuritySource struct {
	Principals    []string `json:"principals,omitempty"`
	NotPrincipals []string `json:"notPrincipals,omitempty"`
	Namespaces    []string `json:"namespaces,omitempty"`
	NotNamespaces []string `json:"notNamespaces,omitempty"`
	IPBlocks      []string `json:"ipBlocks,omitempty"`
	NotIPBlocks   []string `json:"notIpBlocks,omitempty"`
}

// SecurityOperation defines the operation being accessed
type SecurityOperation struct {
	Hosts      []string `json:"hosts,omitempty"`
	NotHosts   []string `json:"notHosts,omitempty"`
	Methods    []string `json:"methods,omitempty"`
	NotMethods []string `json:"notMethods,omitempty"`
	Paths      []string `json:"paths,omitempty"`
	NotPaths   []string `json:"notPaths,omitempty"`
	Ports      []string `json:"ports,omitempty"`
	NotPorts   []string `json:"notPorts,omitempty"`
}

// Condition defines when a rule applies
type Condition struct {
	Key       string   `json:"key"`
	Values    []string `json:"values,omitempty"`
	NotValues []string `json:"notValues,omitempty"`
}

// TrafficPolicy defines traffic management policies
type TrafficPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TrafficPolicySpec `json:"spec"`
}

// TrafficPolicySpec defines traffic policy specification
type TrafficPolicySpec struct {
	Selector        *LabelSelector      `json:"selector"`
	TrafficShifting *TrafficShifting    `json:"trafficShifting,omitempty"`
	CircuitBreaker  *CircuitBreaker     `json:"circuitBreaker,omitempty"`
	Retry           *RetryPolicy        `json:"retry,omitempty"`
	Timeout         *TimeoutPolicy      `json:"timeout,omitempty"`
	LoadBalancer    *LoadBalancerPolicy `json:"loadBalancer,omitempty"`
}

// ServiceRegistration defines a service to register with the mesh
type ServiceRegistration struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Ports       []ServicePort     `json:"ports"`
	SPIFFEID    string            `json:"spiffeId,omitempty"`
	MTLSMode    string            `json:"mtlsMode"`
	Endpoints   []string          `json:"endpoints,omitempty"`
}

// ServicePort defines a service port
type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

// ServiceStatus represents the status of a service in the mesh
type ServiceStatus struct {
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	Healthy           bool                   `json:"healthy"`
	MTLSEnabled       bool                   `json:"mtlsEnabled"`
	CertificateStatus *CertificateStatus     `json:"certificateStatus"`
	Endpoints         []EndpointStatus       `json:"endpoints"`
	Policies          []string               `json:"policies"`
	Metrics           map[string]interface{} `json:"metrics"`
}

// CertificateStatus represents certificate status for a service
type CertificateStatus struct {
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	SerialNumber    string    `json:"serialNumber"`
	NotBefore       time.Time `json:"notBefore"`
	NotAfter        time.Time `json:"notAfter"`
	DaysUntilExpiry int       `json:"daysUntilExpiry"`
	NeedsRotation   bool      `json:"needsRotation"`
}

// EndpointStatus represents the status of a service endpoint
type EndpointStatus struct {
	Address     string    `json:"address"`
	Port        int32     `json:"port"`
	Healthy     bool      `json:"healthy"`
	LastChecked time.Time `json:"lastChecked"`
	Latency     string    `json:"latency"`
}

// PolicyValidationResult contains policy validation results
type PolicyValidationResult struct {
	Valid      bool             `json:"valid"`
	Errors     []PolicyError    `json:"errors,omitempty"`
	Warnings   []PolicyWarning  `json:"warnings,omitempty"`
	Conflicts  []PolicyConflict `json:"conflicts,omitempty"`
	Coverage   float64          `json:"coverage"` // Percentage of services covered
	Compliance PolicyCompliance `json:"compliance"`
}

// PolicyError represents a policy validation error
type PolicyError struct {
	Policy  string `json:"policy"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// PolicyWarning represents a policy validation warning
type PolicyWarning struct {
	Policy  string `json:"policy"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// PolicyConflict represents conflicting policies
type PolicyConflict struct {
	Policy1 string `json:"policy1"`
	Policy2 string `json:"policy2"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// PolicyCompliance represents compliance status
type PolicyCompliance struct {
	MTLSCompliant      bool    `json:"mtlsCompliant"`
	ZeroTrustCompliant bool    `json:"zeroTrustCompliant"`
	NetworkSegmented   bool    `json:"networkSegmented"`
	ComplianceScore    float64 `json:"complianceScore"` // 0-100
}

// DependencyGraph represents service dependencies
type DependencyGraph struct {
	Nodes  []ServiceNode `json:"nodes"`
	Edges  []ServiceEdge `json:"edges"`
	Cycles [][]string    `json:"cycles,omitempty"`
}

// ServiceNode represents a service in the dependency graph
type ServiceNode struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	Labels      map[string]string `json:"labels"`
	MTLSEnabled bool              `json:"mtlsEnabled"`
}

// ServiceEdge represents a connection between services
type ServiceEdge struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Protocol    string  `json:"protocol"`
	Port        int32   `json:"port"`
	MTLSEnabled bool    `json:"mtlsEnabled"`
	Latency     float64 `json:"latency"`   // milliseconds
	ErrorRate   float64 `json:"errorRate"` // percentage
}

// MTLSStatusReport provides comprehensive mTLS status
type MTLSStatusReport struct {
	Timestamp         time.Time             `json:"timestamp"`
	TotalServices     int                   `json:"totalServices"`
	MTLSEnabledCount  int                   `json:"mtlsEnabledCount"`
	Coverage          float64               `json:"coverage"` // Percentage
	CertificateStatus []ServiceCertStatus   `json:"certificateStatus"`
	PolicyStatus      []ServicePolicyStatus `json:"policyStatus"`
	Issues            []MTLSIssue           `json:"issues,omitempty"`
	Recommendations   []string              `json:"recommendations,omitempty"`
}

// ServiceCertStatus represents certificate status for a service
type ServiceCertStatus struct {
	Service       string    `json:"service"`
	Namespace     string    `json:"namespace"`
	Valid         bool      `json:"valid"`
	ExpiryDate    time.Time `json:"expiryDate"`
	NeedsRotation bool      `json:"needsRotation"`
}

// ServicePolicyStatus represents policy status for a service
type ServicePolicyStatus struct {
	Service   string   `json:"service"`
	Namespace string   `json:"namespace"`
	MTLSMode  string   `json:"mtlsMode"`
	Policies  []string `json:"policies"`
}

// MTLSIssue represents an mTLS issue
type MTLSIssue struct {
	Service  string `json:"service"`
	Type     string `json:"type"`
	Severity string `json:"severity"` // "critical", "warning", "info"
	Message  string `json:"message"`
}

// Capability represents a service mesh capability
type Capability string

const (
	// CapabilityMTLS indicates mTLS support
	CapabilityMTLS Capability = "mtls"
	// CapabilityTrafficManagement indicates traffic management support
	CapabilityTrafficManagement Capability = "traffic-management"
	// CapabilityObservability indicates observability support
	CapabilityObservability Capability = "observability"
	// CapabilityMultiCluster indicates multi-cluster support
	CapabilityMultiCluster Capability = "multi-cluster"
	// CapabilitySPIFFE indicates SPIFFE/SPIRE support
	CapabilitySPIFFE Capability = "spiffe"
	// CapabilityWASM indicates WebAssembly extension support
	CapabilityWASM Capability = "wasm"
)

// CertificateProvider manages certificates for the service mesh
type CertificateProvider interface {
	// IssueCertificate issues a new certificate for a service
	IssueCertificate(ctx context.Context, service string, namespace string) (*x509.Certificate, error)

	// GetRootCA returns the root CA certificate
	GetRootCA(ctx context.Context) (*x509.Certificate, error)

	// GetIntermediateCA returns intermediate CA if applicable
	GetIntermediateCA(ctx context.Context) (*x509.Certificate, error)

	// ValidateCertificate validates a certificate
	ValidateCertificate(ctx context.Context, cert *x509.Certificate) error

	// RotateCertificate rotates a certificate for a service
	RotateCertificate(ctx context.Context, service string, namespace string) (*x509.Certificate, error)

	// GetCertificateChain returns the certificate chain for a service
	GetCertificateChain(ctx context.Context, service string, namespace string) ([]*x509.Certificate, error)

	// GetSPIFFEID returns the SPIFFE ID for a service
	GetSPIFFEID(service string, namespace string, trustDomain string) string
}

// LabelSelector defines label selection criteria
type LabelSelector struct {
	MatchLabels      map[string]string          `json:"matchLabels,omitempty"`
	MatchExpressions []LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement defines a label selector requirement
type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // "In", "NotIn", "Exists", "DoesNotExist"
	Values   []string `json:"values,omitempty"`
}

// TrafficShifting defines traffic shifting configuration
type TrafficShifting struct {
	Destinations []WeightedDestination `json:"destinations"`
}

// WeightedDestination defines a weighted traffic destination
type WeightedDestination struct {
	Service string `json:"service"`
	Weight  int    `json:"weight"`
	Version string `json:"version,omitempty"`
}

// CircuitBreaker defines circuit breaker configuration
type CircuitBreaker struct {
	ConsecutiveErrors  int    `json:"consecutiveErrors"`
	Interval           string `json:"interval"`
	BaseEjectionTime   string `json:"baseEjectionTime"`
	MaxEjectionPercent int    `json:"maxEjectionPercent"`
}

// RetryPolicy defines retry configuration
type RetryPolicy struct {
	Attempts      int      `json:"attempts"`
	PerTryTimeout string   `json:"perTryTimeout"`
	RetryOn       []string `json:"retryOn"`
	BackoffBase   string   `json:"backoffBase,omitempty"`
	BackoffMax    string   `json:"backoffMax,omitempty"`
}

// TimeoutPolicy defines timeout configuration
type TimeoutPolicy struct {
	RequestTimeout string `json:"requestTimeout"`
	IdleTimeout    string `json:"idleTimeout,omitempty"`
}

// LoadBalancerPolicy defines load balancing configuration
type LoadBalancerPolicy struct {
	Algorithm      string                `json:"algorithm"` // "round-robin", "least-conn", "random", "consistent-hash"
	ConsistentHash *ConsistentHashConfig `json:"consistentHash,omitempty"`
}

// ConsistentHashConfig defines consistent hash configuration
type ConsistentHashConfig struct {
	HashKey         string `json:"hashKey"`
	MinimumRingSize int    `json:"minimumRingSize"`
}

// ExtensionProvider defines an external authorization provider
type ExtensionProvider struct {
	Name    string            `json:"name"`
	Service string            `json:"service"`
	Port    int               `json:"port"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout string            `json:"timeout,omitempty"`
}
