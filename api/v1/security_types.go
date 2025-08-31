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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityParameters defines structured security parameters for NetworkIntent
type SecurityParameters struct {
	// TLS configuration
	// +optional
	TLSEnabled *bool `json:"tlsEnabled,omitempty"`

	// Service mesh configuration
	// +optional
	ServiceMesh *bool `json:"serviceMesh,omitempty"`

	// Encryption configuration
	// +optional
	Encryption *EncryptionConfig `json:"encryption,omitempty"`

	// Network policies configuration
	// +optional
	NetworkPolicies []NetworkPolicyConfig `json:"networkPolicies,omitempty"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	// Whether encryption is enabled
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Encryption algorithm
	// +optional
	Algorithm string `json:"algorithm,omitempty"`

	// Key size in bits
	// +optional
	KeySize int `json:"keySize,omitempty"`
}

// NetworkPolicyConfig defines network policy settings
type NetworkPolicyConfig struct {
	// Name of the network policy
	Name string `json:"name"`

	// Type of policy (ingress, egress, or both)
	// +kubebuilder:validation:Enum=ingress;egress;both
	Type string `json:"type"`

	// Rules for the policy
	// +optional
	Rules []string `json:"rules,omitempty"`
}

// EncryptedSecret represents an encrypted secret with comprehensive metadata
type EncryptedSecret struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	EncryptedData []byte            `json:"encrypted_data"`
	Nonce         []byte            `json:"nonce,omitempty"`
	Algorithm     string            `json:"algorithm"`
	KeyID         string            `json:"key_id"`
	CreatedAt     metav1.Time       `json:"created_at"`
	UpdatedAt     metav1.Time       `json:"updated_at"`
	LastAccessed  *metav1.Time      `json:"last_accessed,omitempty"`
	ExpiresAt     *metav1.Time      `json:"expires_at,omitempty"`
	AccessCount   int64             `json:"access_count"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Version       int               `json:"version"`
	KeyVersion    int               `json:"key_version"`
}

// OPACompliancePolicyEngine represents Open Policy Agent compliance engine
type OPACompliancePolicyEngine struct {
	// Name of the OPA engine instance
	Name string `json:"name"`

	// Namespace where OPA is deployed
	Namespace string `json:"namespace"`

	// OPA server endpoint
	Endpoint string `json:"endpoint"`

	// Policy package name
	PolicyPackage string `json:"policyPackage"`

	// Policies loaded in the engine
	Policies []OPAPolicy `json:"policies,omitempty"`

	// Configuration for the OPA engine
	Config *OPAConfig `json:"config,omitempty"`

	// Status of the engine
	Status OPAEngineStatus `json:"status,omitempty"`
}

// OPAPolicy represents a single OPA policy
type OPAPolicy struct {
	// Name of the policy
	Name string `json:"name"`

	// Rego policy content
	Rego string `json:"rego"`

	// Package name for the policy
	Package string `json:"package"`

	// Version of the policy
	Version string `json:"version,omitempty"`

	// When the policy was created
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// When the policy was last updated
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}

// OPAConfig represents OPA configuration
type OPAConfig struct {
	// Bundles configuration
	Bundles map[string]OPABundle `json:"bundles,omitempty"`

	// Decision logs configuration
	DecisionLogs *OPADecisionLogsConfig `json:"decisionLogs,omitempty"`

	// Status configuration
	Status *OPAStatusConfig `json:"status,omitempty"`

	// Server configuration
	Server *OPAServerConfig `json:"server,omitempty"`
}

// OPABundle represents an OPA bundle configuration
type OPABundle struct {
	// Service name
	Service string `json:"service"`

	// Resource path
	Resource string `json:"resource,omitempty"`

	// Signing configuration
	Signing *OPABundleSigning `json:"signing,omitempty"`
}

// OPABundleSigning represents bundle signing configuration
type OPABundleSigning struct {
	// Public key for verification
	PublicKey string `json:"publicKey,omitempty"`

	// Key ID
	KeyID string `json:"keyId,omitempty"`

	// Exclude files from verification
	Exclude []string `json:"exclude,omitempty"`
}

// OPADecisionLogsConfig represents decision logs configuration
type OPADecisionLogsConfig struct {
	// Console logging
	Console bool `json:"console,omitempty"`

	// Service name for remote logging
	Service string `json:"service,omitempty"`

	// Reporting configuration
	Reporting *OPAReportingConfig `json:"reporting,omitempty"`
}

// OPAStatusConfig represents status reporting configuration
type OPAStatusConfig struct {
	// Service name for status reporting
	Service string `json:"service,omitempty"`

	// Trigger mode
	Trigger string `json:"trigger,omitempty"`
}

// OPAServerConfig represents OPA server configuration
type OPAServerConfig struct {
	// Encoding for server responses
	Encoding *OPAServerEncoding `json:"encoding,omitempty"`
}

// OPAServerEncoding represents server encoding configuration
type OPAServerEncoding struct {
	// GZIP compression
	GZIP *OPAGZIPConfig `json:"gzip,omitempty"`
}

// OPAGZIPConfig represents GZIP compression configuration
type OPAGZIPConfig struct {
	// Compression level
	Level int `json:"level,omitempty"`
}

// OPAReportingConfig represents reporting configuration
type OPAReportingConfig struct {
	// Minimum delay between reports
	MinDelaySeconds int `json:"minDelaySeconds,omitempty"`

	// Maximum delay between reports
	MaxDelaySeconds int `json:"maxDelaySeconds,omitempty"`

	// Upload size limit
	UploadSizeLimitBytes int64 `json:"uploadSizeLimitBytes,omitempty"`
}

// OPAEngineStatus represents the status of an OPA engine
type OPAEngineStatus struct {
	// Ready indicates if the engine is ready
	Ready bool `json:"ready"`

	// Healthy indicates if the engine is healthy
	Healthy bool `json:"healthy"`

	// Number of loaded policies
	PolicyCount int `json:"policyCount"`

	// Last evaluation timestamp
	LastEvaluation *metav1.Time `json:"lastEvaluation,omitempty"`

	// Error message if any
	Error string `json:"error,omitempty"`

	// Version of OPA
	Version string `json:"version,omitempty"`
}
