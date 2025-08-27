/*
Copyright 2024 Nephoran.

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
)

// CertificateAutomationSpec defines the desired state of CertificateAutomation
type CertificateAutomationSpec struct {
	// ServiceName specifies the target service name
	ServiceName string `json:"serviceName"`

	// Namespace specifies the target namespace
	Namespace string `json:"namespace"`

	// DNSNames specifies additional DNS names for the certificate
	DNSNames []string `json:"dnsNames,omitempty"`

	// IPAddresses specifies IP addresses for the certificate
	IPAddresses []string `json:"ipAddresses,omitempty"`

	// Template specifies the certificate template to use
	Template string `json:"template,omitempty"`

	// AutoRenewal enables automatic certificate renewal
	AutoRenewal bool `json:"autoRenewal,omitempty"`

	// ValidityDuration specifies certificate validity period
	ValidityDuration *metav1.Duration `json:"validityDuration,omitempty"`

	// SecretName specifies the target secret name
	SecretName string `json:"secretName,omitempty"`

	// Priority specifies provisioning priority
	Priority string `json:"priority,omitempty"`
}

// CertificateAutomationPhase represents the phase of certificate automation
type CertificateAutomationPhase string

const (
	// CertificateAutomationPhasePending indicates the request is pending
	CertificateAutomationPhasePending CertificateAutomationPhase = "Pending"

	// CertificateAutomationPhaseProvisioning indicates certificate is being provisioned
	CertificateAutomationPhaseProvisioning CertificateAutomationPhase = "Provisioning"

	// CertificateAutomationPhaseReady indicates certificate is ready
	CertificateAutomationPhaseReady CertificateAutomationPhase = "Ready"

	// CertificateAutomationPhaseRenewing indicates certificate is being renewed
	CertificateAutomationPhaseRenewing CertificateAutomationPhase = "Renewing"

	// CertificateAutomationPhaseExpired indicates certificate has expired
	CertificateAutomationPhaseExpired CertificateAutomationPhase = "Expired"

	// CertificateAutomationPhaseRevoked indicates certificate has been revoked
	CertificateAutomationPhaseRevoked CertificateAutomationPhase = "Revoked"

	// CertificateAutomationPhaseFailed indicates an error occurred
	CertificateAutomationPhaseFailed CertificateAutomationPhase = "Failed"
)

// CertificateAutomationConditionType represents condition types
type CertificateAutomationConditionType string

const (
	// CertificateAutomationConditionReady indicates the certificate is ready
	CertificateAutomationConditionReady CertificateAutomationConditionType = "Ready"

	// CertificateAutomationConditionIssued indicates the certificate has been issued
	CertificateAutomationConditionIssued CertificateAutomationConditionType = "Issued"

	// CertificateAutomationConditionValidated indicates the certificate has been validated
	CertificateAutomationConditionValidated CertificateAutomationConditionType = "Validated"

	// CertificateAutomationConditionRenewed indicates the certificate has been renewed
	CertificateAutomationConditionRenewed CertificateAutomationConditionType = "Renewed"

	// CertificateAutomationConditionRevoked indicates the certificate has been revoked
	CertificateAutomationConditionRevoked CertificateAutomationConditionType = "Revoked"
)

// CertificateAutomationCondition describes the state of certificate automation
type CertificateAutomationCondition struct {
	// Type of condition
	Type CertificateAutomationConditionType `json:"type"`

	// Status of the condition
	Status metav1.ConditionStatus `json:"status"`

	// Last time the condition transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason for the condition's last transition
	Reason string `json:"reason,omitempty"`

	// Message providing details about the transition
	Message string `json:"message,omitempty"`
}

// CertificateValidationStatus contains certificate validation details
type CertificateValidationStatus struct {
	// Valid indicates if the certificate is valid
	Valid bool `json:"valid"`

	// ChainValid indicates if the certificate chain is valid
	ChainValid bool `json:"chainValid,omitempty"`

	// CTLogVerified indicates if the certificate is in CT logs
	CTLogVerified bool `json:"ctLogVerified,omitempty"`

	// LastValidationTime contains the last validation time
	LastValidationTime *metav1.Time `json:"lastValidationTime,omitempty"`

	// ValidationErrors contains validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// ValidationWarnings contains validation warnings
	ValidationWarnings []string `json:"validationWarnings,omitempty"`
}

// CertificateAutomationStatus defines the observed state of CertificateAutomation
type CertificateAutomationStatus struct {
	// Phase represents the current phase of certificate automation
	Phase CertificateAutomationPhase `json:"phase,omitempty"`

	// Conditions represents the current conditions
	Conditions []CertificateAutomationCondition `json:"conditions,omitempty"`

	// CertificateSerialNumber contains the serial number of the issued certificate
	CertificateSerialNumber string `json:"certificateSerialNumber,omitempty"`

	// ExpiresAt contains the certificate expiration time
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// LastRenewalTime contains the last renewal time
	LastRenewalTime *metav1.Time `json:"lastRenewalTime,omitempty"`

	// NextRenewalTime contains the next scheduled renewal time
	NextRenewalTime *metav1.Time `json:"nextRenewalTime,omitempty"`

	// SecretRef contains reference to the created secret
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// ValidationStatus contains certificate validation status
	ValidationStatus *CertificateValidationStatus `json:"validationStatus,omitempty"`

	// RevocationStatus contains certificate revocation status
	RevocationStatus string `json:"revocationStatus,omitempty"`

	// ObservedGeneration represents the generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=certauto
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Service",type="string",JSONPath=".spec.serviceName"
//+kubebuilder:printcolumn:name="Expires At",type="string",JSONPath=".status.expiresAt"
//+kubebuilder:printcolumn:name="Auto Renewal",type="boolean",JSONPath=".spec.autoRenewal"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CertificateAutomation is the Schema for the certificateautomations API
type CertificateAutomation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateAutomationSpec   `json:"spec,omitempty"`
	Status CertificateAutomationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertificateAutomationList contains a list of CertificateAutomation
type CertificateAutomationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertificateAutomation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertificateAutomation{}, &CertificateAutomationList{})
}
