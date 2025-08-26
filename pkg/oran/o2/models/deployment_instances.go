/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import (
	"time"
)

// Deployment represents an active deployment instance following O2 IMS specification
type Deployment struct {
	DeploymentID      string                 `json:"deploymentId"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	TemplateID        string                 `json:"templateId"`
	TemplateVersion   string                 `json:"templateVersion,omitempty"`
	
	// Deployment configuration
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	PoolID           string                 `json:"poolId"`
	Environment      string                 `json:"environment,omitempty"`
	
	// Runtime status
	Status           string                 `json:"status"` // creating, active, updating, deleting, failed, terminated
	Phase            string                 `json:"phase,omitempty"`
	Health           *DeploymentHealth      `json:"health,omitempty"`
	
	// Resources and scaling
	ResourceUtilization *ResourceUtilization `json:"resourceUtilization,omitempty"`
	ScalingPolicy    *ScalingPolicy         `json:"scalingPolicy,omitempty"`
	
	// Lifecycle metadata
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	StartedAt        *time.Time             `json:"startedAt,omitempty"`
	TerminatedAt     *time.Time             `json:"terminatedAt,omitempty"`
	
	// Ownership and management
	CreatedBy        string                 `json:"createdBy,omitempty"`
	UpdatedBy        string                 `json:"updatedBy,omitempty"`
	OwnerID          string                 `json:"ownerId,omitempty"`
	
	// Networking and connectivity
	Endpoints        []*DeploymentEndpoint  `json:"endpoints,omitempty"`
	NetworkPolicies  []string               `json:"networkPolicies,omitempty"`
	
	// Monitoring and observability
	MetricsEnabled   bool                   `json:"metricsEnabled"`
	LoggingEnabled   bool                   `json:"loggingEnabled"`
	TracingEnabled   bool                   `json:"tracingEnabled"`
	
	// Security and compliance
	SecurityContext  *SecurityContext       `json:"securityContext,omitempty"`
	ComplianceStatus *ComplianceStatus      `json:"complianceStatus,omitempty"`
	
	// Extension and customization
	Labels           map[string]string      `json:"labels,omitempty"`
	Annotations      map[string]string      `json:"annotations,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// DeploymentHealth represents the health status of a deployment
type DeploymentHealth struct {
	Status           string                 `json:"status"` // healthy, degraded, unhealthy, unknown
	LastCheck        time.Time              `json:"lastCheck"`
	HealthScore      float64                `json:"healthScore"` // 0.0 to 1.0
	
	// Component health
	ComponentHealth  map[string]string      `json:"componentHealth,omitempty"`
	
	// Health indicators
	ReadinessProbes  *ProbeStatus           `json:"readinessProbes,omitempty"`
	LivenessProbes   *ProbeStatus           `json:"livenessProbes,omitempty"`
	StartupProbes    *ProbeStatus           `json:"startupProbes,omitempty"`
	
	// Performance metrics
	ResponseTime     *time.Duration         `json:"responseTime,omitempty"`
	ErrorRate        *float64               `json:"errorRate,omitempty"`
	Throughput       *float64               `json:"throughput,omitempty"`
	
	// Issues and alerts
	ActiveAlerts     []string               `json:"activeAlerts,omitempty"`
	RecentErrors     []string               `json:"recentErrors,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
}

// ProbeStatus represents the status of health probes
type ProbeStatus struct {
	Status        string    `json:"status"` // passing, failing, unknown
	LastCheck     time.Time `json:"lastCheck"`
	SuccessCount  int       `json:"successCount"`
	FailureCount  int       `json:"failureCount"`
	LastFailure   *time.Time `json:"lastFailure,omitempty"`
	FailureReason string    `json:"failureReason,omitempty"`
}

// ResourceUtilization represents the resource utilization status
type ResourceUtilization struct {
	// Resource allocation
	AllocatedCPU     string                 `json:"allocatedCpu,omitempty"`
	AllocatedMemory  string                 `json:"allocatedMemory,omitempty"`
	AllocatedStorage string                 `json:"allocatedStorage,omitempty"`
	
	// Resource utilization
	UsedCPU          string                 `json:"usedCpu,omitempty"`
	UsedMemory       string                 `json:"usedMemory,omitempty"`
	UsedStorage      string                 `json:"usedStorage,omitempty"`
	
	// Utilization percentages
	CPUUtilization   float64                `json:"cpuUtilization,omitempty"`
	MemoryUtilization float64               `json:"memoryUtilization,omitempty"`
	StorageUtilization float64              `json:"storageUtilization,omitempty"`
	
	// Instance counts
	DesiredInstances int                    `json:"desiredInstances"`
	CurrentInstances int                    `json:"currentInstances"`
	ReadyInstances   int                    `json:"readyInstances"`
	AvailableInstances int                  `json:"availableInstances"`
	
	// Custom resource metrics
	CustomMetrics    map[string]interface{} `json:"customMetrics,omitempty"`
}


// DeploymentEndpoint represents a network endpoint exposed by the deployment
type DeploymentEndpoint struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // service, ingress, loadbalancer, nodeport
	Protocol     string                 `json:"protocol"` // HTTP, HTTPS, TCP, UDP, gRPC
	Port         int                    `json:"port"`
	TargetPort   int                    `json:"targetPort,omitempty"`
	ExternalURL  string                 `json:"externalUrl,omitempty"`
	InternalURL  string                 `json:"internalUrl,omitempty"`
	
	// Endpoint configuration
	PathPrefix   string                 `json:"pathPrefix,omitempty"`
	TLS          *TLSConfig             `json:"tls,omitempty"`
	
	// Load balancing
	LoadBalancer *LoadBalancerConfig    `json:"loadBalancer,omitempty"`
	
	// Health checks
	HealthCheck  *EndpointHealthCheck   `json:"healthCheck,omitempty"`
	
	// Metadata
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
}

// TLSConfig represents TLS configuration for an endpoint
type TLSConfig struct {
	Enabled     bool     `json:"enabled"`
	SecretName  string   `json:"secretName,omitempty"`
	Hosts       []string `json:"hosts,omitempty"`
	Issuer      string   `json:"issuer,omitempty"`
	AutoTLS     bool     `json:"autoTls"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Type              string                 `json:"type"` // round-robin, least-conn, ip-hash, weighted
	SessionAffinity   string                 `json:"sessionAffinity,omitempty"`
	StickySession     bool                   `json:"stickySession"`
	HealthCheckPath   string                 `json:"healthCheckPath,omitempty"`
	Configuration     map[string]interface{} `json:"configuration,omitempty"`
}

// EndpointHealthCheck represents health check configuration for an endpoint
type EndpointHealthCheck struct {
	Enabled         bool           `json:"enabled"`
	Path            string         `json:"path,omitempty"`
	Port            int            `json:"port,omitempty"`
	Protocol        string         `json:"protocol"` // HTTP, TCP
	IntervalSeconds int            `json:"intervalSeconds"`
	TimeoutSeconds  int            `json:"timeoutSeconds"`
	FailureThreshold int           `json:"failureThreshold"`
	SuccessThreshold int           `json:"successThreshold"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// SecurityContext represents security configuration for the deployment
type SecurityContext struct {
	RunAsUser              *int64              `json:"runAsUser,omitempty"`
	RunAsGroup             *int64              `json:"runAsGroup,omitempty"`
	RunAsNonRoot           *bool               `json:"runAsNonRoot,omitempty"`
	ReadOnlyRootFilesystem *bool               `json:"readOnlyRootFilesystem,omitempty"`
	
	// Capabilities and privileges
	AllowPrivilegeEscalation *bool             `json:"allowPrivilegeEscalation,omitempty"`
	Privileged               *bool             `json:"privileged,omitempty"`
	Capabilities             *SecurityCapabilities `json:"capabilities,omitempty"`
	
	// SELinux and AppArmor
	SELinuxOptions           *SELinuxOptions   `json:"seLinuxOptions,omitempty"`
	AppArmorProfile          string            `json:"appArmorProfile,omitempty"`
	SeccompProfile           string            `json:"seccompProfile,omitempty"`
	
	// Network security
	NetworkPolicies          []string          `json:"networkPolicies,omitempty"`
	PodSecurityPolicy        string            `json:"podSecurityPolicy,omitempty"`
	ServiceAccount           string            `json:"serviceAccount,omitempty"`
}

// SecurityCapabilities represents Linux capabilities
type SecurityCapabilities struct {
	Add    []string `json:"add,omitempty"`
	Drop   []string `json:"drop,omitempty"`
}


// ComplianceStatus represents compliance and audit information
type ComplianceStatus struct {
	Status           string                 `json:"status"` // compliant, non-compliant, unknown
	LastAudit        time.Time              `json:"lastAudit"`
	ComplianceScore  float64                `json:"complianceScore"` // 0.0 to 1.0
	
	// Compliance frameworks
	Frameworks       []string               `json:"frameworks,omitempty"` // SOC2, PCI-DSS, HIPAA, etc.
	
	// Policy compliance
	PolicyCompliance map[string]string      `json:"policyCompliance,omitempty"`
	Violations       []ComplianceViolation  `json:"violations,omitempty"`
	Recommendations  []string               `json:"recommendations,omitempty"`
	
	// Security scans
	VulnerabilityStatus *VulnerabilityStatus `json:"vulnerabilityStatus,omitempty"`
	
	// Audit trail
	AuditEvents      []AuditEvent           `json:"auditEvents,omitempty"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string    `json:"id"`
	Rule        string    `json:"rule"`
	Severity    string    `json:"severity"` // critical, high, medium, low
	Description string    `json:"description"`
	Component   string    `json:"component,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
	Status      string    `json:"status"` // open, acknowledged, resolved, suppressed
}

// VulnerabilityStatus represents vulnerability scan results
type VulnerabilityStatus struct {
	LastScan        time.Time              `json:"lastScan"`
	ScanStatus      string                 `json:"scanStatus"` // completed, in-progress, failed
	
	// Vulnerability counts by severity
	Critical        int                    `json:"critical"`
	High            int                    `json:"high"`
	Medium          int                    `json:"medium"`
	Low             int                    `json:"low"`
	
	// Detailed vulnerabilities
	Vulnerabilities []Vulnerability        `json:"vulnerabilities,omitempty"`
	
	// Scan metadata
	ScannerVersion  string                 `json:"scannerVersion,omitempty"`
	DatabaseVersion string                 `json:"databaseVersion,omitempty"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string    `json:"id"` // CVE ID or scanner-specific ID
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Version     string    `json:"version,omitempty"`
	FixVersion  string    `json:"fixVersion,omitempty"`
	CVSS        *CVSSInfo `json:"cvss,omitempty"`
	References  []string  `json:"references,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// CVSSInfo represents Common Vulnerability Scoring System information
type CVSSInfo struct {
	Version string  `json:"version"` // 2.0, 3.0, 3.1
	Vector  string  `json:"vector"`
	Score   float64 `json:"score"`
	Severity string `json:"severity"`
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // access, change, compliance, security
	Action    string                 `json:"action"`
	User      string                 `json:"user,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Result    string                 `json:"result"` // success, failure, error
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}