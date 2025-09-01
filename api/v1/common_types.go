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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ... (previous content)

// ManagedElementSpec defines the desired state of ManagedElement
type ManagedElementSpec struct {
	// ID is the unique identifier for the managed element
	ID string `json:"id"`
	// Name is the display name of the managed element
	Name string `json:"name"`
	// Type specifies the type of managed element (e.g., "gNB", "CU", "DU")
	Type string `json:"type"`
	// Credentials for accessing the managed element
	// +optional
	Credentials *ManagedElementCredentials `json:"credentials,omitempty"`
	// Configuration for the managed element
	// +optional
	Config map[string]string `json:"config,omitempty"`
	// Tags for organizing managed elements
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
	// Host specifies the hostname or IP address of the managed element
	// +optional
	Host string `json:"host,omitempty"`
	// Port specifies the port number for the managed element
	// +optional
	Port int `json:"port,omitempty"`
	// O1Config specifies the O1 interface configuration for the managed element
	// +optional
	O1Config string `json:"o1Config,omitempty"`
	// A1Policy specifies the A1 policy configuration
	// +optional
	A1Policy runtime.RawExtension `json:"a1Policy,omitempty"`
}

// ManagedElementCredentials defines authentication credentials
type ManagedElementCredentials struct {
	// Username for basic authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password reference for basic authentication
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"passwordRef,omitempty"`

	// Certificate reference for certificate-based authentication
	// +optional
	CertificateRef *corev1.SecretKeySelector `json:"certificateRef,omitempty"`

	// Private key reference for certificate-based authentication
	// +optional
	PrivateKeyRef *corev1.SecretKeySelector `json:"privateKeyRef,omitempty"`

	// Token reference for token-based authentication
	// +optional
	TokenRef *corev1.SecretKeySelector `json:"tokenRef,omitempty"`

	// SSH key reference for SSH-based authentication
	// +optional
	SSHKeyRef *corev1.SecretKeySelector `json:"sshKeyRef,omitempty"`

	// Client certificate reference for mTLS
	// +optional
	ClientCertRef *corev1.SecretKeySelector `json:"clientCertRef,omitempty"`

	// Client key reference for mTLS
	// +optional
	ClientKeyRef *corev1.SecretKeySelector `json:"clientKeyRef,omitempty"`

	// CA certificate reference
	// +optional
	CACertificateRef *corev1.SecretKeySelector `json:"caCertificateRef,omitempty"`

	// Client certificate reference for authentication
	// +optional
	ClientCertificateRef *corev1.SecretKeySelector `json:"clientCertificateRef,omitempty"`

	// Username reference for basic authentication
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"usernameRef,omitempty"`
}

// Priority defines processing priority levels
type Priority string

const (
	PriorityLow      Priority = "Low"
	PriorityMedium   Priority = "Medium"
	PriorityHigh     Priority = "High"
	PriorityUrgent   Priority = "Urgent"
	PriorityCritical Priority = "Critical"
)

// ObjectReference represents a reference to a Kubernetes object
type ObjectReference struct {
	// API version of the referent
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind of the referent
	Kind string `json:"kind,omitempty"`
	// Name of the referent
	Name string `json:"name,omitempty"`
	// Namespace of the referent
	Namespace string `json:"namespace,omitempty"`
	// UID of the referent
	UID string `json:"uid,omitempty"`
}

// ResourceConstraints defines resource limits and requests
type ResourceConstraints struct {
	// CPU constraints
	CPU *ResourceConstraint `json:"cpu,omitempty"`
	// Memory constraints
	Memory *ResourceConstraint `json:"memory,omitempty"`
	// Storage constraints
	Storage *ResourceConstraint `json:"storage,omitempty"`
	// Bandwidth constraints
	Bandwidth *ResourceConstraint `json:"bandwidth,omitempty"`
	// CustomResources for custom resource constraints
	CustomResources map[string]*ResourceConstraint `json:"customResources,omitempty"`
}

// ResourceConstraint defines a single resource constraint
type ResourceConstraint struct {
	// Minimum required amount
	Min *string `json:"min,omitempty"`
	// Maximum allowed amount
	Max *string `json:"max,omitempty"`
	// Requested amount
	Request *string `json:"request,omitempty"`
	// Limit amount
	Limit *string `json:"limit,omitempty"`
}

// ProcessedParameters contains structured parameters extracted from the intent
type ProcessedParameters struct {
	// NetworkFunction specifies the target network function
	NetworkFunction string `json:"networkFunction,omitempty"`
	// Region specifies the deployment region
	Region string `json:"region,omitempty"`
	// Scaling parameters
	Scaling *ScalingParameters `json:"scaling,omitempty"`
	// Resource parameters
	Resources *ResourceParameters `json:"resources,omitempty"`
	// Network parameters
	Network *NetworkParameters `json:"network,omitempty"`
	// Custom parameters as key-value pairs
	Custom map[string]string `json:"custom,omitempty"`
	// Raw parameters for backup
	Raw map[string]string `json:"raw,omitempty"`
	// Structured parameters
	Structured map[string]string `json:"structured,omitempty"`
	// CustomParameters as separate field
	CustomParameters map[string]string `json:"customParameters,omitempty"`
	// SecurityParameters for security-related settings
	SecurityParameters *SecurityParameters `json:"securityParameters,omitempty"`
	// Metadata for additional information
	Metadata *ParameterMetadata `json:"metadata,omitempty"`
}

// ScalingParameters defines scaling-related parameters
type ScalingParameters struct {
	// Minimum replicas
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// Maximum replicas
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// Target replicas
	TargetReplicas *int32 `json:"targetReplicas,omitempty"`
	// Scaling triggers
	Triggers []ScalingTrigger `json:"triggers,omitempty"`
}

// ScalingTrigger defines a scaling trigger condition
type ScalingTrigger struct {
	// Metric name
	Metric string `json:"metric"`
	// Target value
	TargetValue string `json:"targetValue"`
	// Comparison operator (>, <, >=, <=, ==)
	Operator string `json:"operator"`
}

// ResourceParameters defines resource-related parameters
type ResourceParameters struct {
	// CPU requirements
	CPU string `json:"cpu,omitempty"`
	// Memory requirements
	Memory string `json:"memory,omitempty"`
	// Storage requirements
	Storage string `json:"storage,omitempty"`
}

// NetworkParameters defines network-related parameters
type NetworkParameters struct {
	// Service ports
	Ports []ServicePortConfig `json:"ports,omitempty"`
	// Service type
	ServiceType string `json:"serviceType,omitempty"`
	// Load balancer settings
	LoadBalancer *LoadBalancerConfig `json:"loadBalancer,omitempty"`
}

// ServicePortConfig defines a service port configuration for common use
type ServicePortConfig struct {
	// Port name
	Name string `json:"name,omitempty"`
	// Port number
	Port int32 `json:"port"`
	// Target port
	TargetPort int32 `json:"targetPort,omitempty"`
	// Protocol (TCP, UDP)
	Protocol string `json:"protocol,omitempty"`
}

// LoadBalancerConfig defines load balancer configuration
type LoadBalancerConfig struct {
	// Load balancer type
	Type string `json:"type,omitempty"`
	// Health check configuration
	HealthCheck *CommonHealthCheckConfig `json:"healthCheck,omitempty"`
}

// CommonHealthCheckConfig defines health check configuration for common use
type CommonHealthCheckConfig struct {
	// Health check path
	Path string `json:"path,omitempty"`
	// Health check port
	Port int32 `json:"port,omitempty"`
	// Check interval in seconds
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`
	// Timeout in seconds
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
}

// TargetComponent defines a target component
type TargetComponent struct {
	// Name of the component
	Name string `json:"name"`
	// Type of the component
	Type string `json:"type"`
	// Namespace of the component
	Namespace string `json:"namespace,omitempty"`
	// Labels for component selection
	Labels map[string]string `json:"labels,omitempty"`
}

// BackupCompressionConfig defines backup compression configuration
type BackupCompressionConfig struct {
	// Compression type
	Type string `json:"type,omitempty"`
	// Compression level
	Level int `json:"level,omitempty"`
}

// ManagedElement defines a managed element
type ManagedElement struct {
	// Standard object's metadata
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the managed element
	Spec ManagedElementSpec `json:"spec,omitempty"`
	
	// Status defines the observed state of the managed element
	Status ManagedElementStatus `json:"status,omitempty"`
}

// ManagedElementStatus defines the observed state of ManagedElement
type ManagedElementStatus struct {
	// Phase represents the lifecycle phase of the managed element
	Phase string `json:"phase,omitempty"`
	// Message contains human-readable information about the status
	Message string `json:"message,omitempty"`
	// Ready indicates if the managed element is ready
	Ready bool `json:"ready,omitempty"`
	// Last update time
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
	// Conditions represent the current conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// LastConnected represents the last connection time
	LastConnected *metav1.Time `json:"lastConnected,omitempty"`
}

// ManagedElementList contains a list of ManagedElement
type ManagedElementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedElement `json:"items"`
}

// ParameterMetadata defines metadata for parameters
type ParameterMetadata struct {
	// Name of the parameter
	Name string `json:"name"`
	// Type of the parameter
	Type string `json:"type"`
	// Description of the parameter
	Description string `json:"description,omitempty"`
	// Whether the parameter is required
	Required bool `json:"required,omitempty"`
	// Default value
	Default string `json:"default,omitempty"`
	// ProcessedAt indicates when the parameter was processed
	ProcessedAt *metav1.Time `json:"processedAt,omitempty"`
	// ConfidenceScore indicates confidence in parameter extraction
	ConfidenceScore *float64 `json:"confidenceScore,omitempty"`
	// Annotations for additional metadata
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ResourceConstraintSpec defines resource constraint specifications
type ResourceConstraintSpec struct {
	// CPU constraints
	CPU string `json:"cpu,omitempty"`
	// Memory constraints
	Memory string `json:"memory,omitempty"`
	// Storage constraints
	Storage string `json:"storage,omitempty"`
	// Min constraints
	Min *string `json:"min,omitempty"`
	// Max constraints
	Max *string `json:"max,omitempty"`
	// Request constraints
	Request *string `json:"request,omitempty"`
	// Limit constraints
	Limit *string `json:"limit,omitempty"`
	// Default constraints
	Default *string `json:"default,omitempty"`
	// Preferred constraints
	Preferred *string `json:"preferred,omitempty"`
}

// Note: SecurityParameters and TLSConfig are defined in other files

// HealthCheck defines health check configuration
type HealthCheck struct {
	// Enabled indicates if health check is enabled
	Enabled bool `json:"enabled,omitempty"`
	// Path for health check endpoint
	Path string `json:"path,omitempty"`
	// Port for health check
	Port int32 `json:"port,omitempty"`
	// Interval between checks
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`
	// Timeout for each check
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
	// Resource reference for health check
	Resource string `json:"resource,omitempty"`
	// Configuration for the health check
	Configuration map[string]string `json:"configuration,omitempty"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	// Enabled indicates if health checks are enabled
	Enabled *bool `json:"enabled,omitempty"`
	// HTTP health check configuration
	HTTP *HealthCheck `json:"http,omitempty"`
	// TCP health check configuration
	TCP *HealthCheck `json:"tcp,omitempty"`
	// Exec health check configuration
	Exec *HealthCheck `json:"exec,omitempty"`
	// Checks is a list of health checks
	Checks []HealthCheck `json:"checks,omitempty"`
}

// ... (rest of the previous content)