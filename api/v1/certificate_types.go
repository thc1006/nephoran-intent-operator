
// Package v1 provides API types for certificate management in the Nephoran Intent Operator.
// This package defines custom resources for handling certificate generation, rotation,
// and management across telecommunications network functions, ensuring secure
// communication and identity verification in O-RAN and 5G network environments.
package v1



import (

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)



// CertificateAutomationSpec defines the desired state of CertificateAutomation.

type CertificateAutomationSpec struct {

	// ServiceName specifies the target service name.

	ServiceName string `json:"serviceName"`



	// Namespace specifies the target namespace.

	Namespace string `json:"namespace"`



	// CertType specifies the type of certificate (TLS, mTLS, etc.).

	CertType string `json:"certType,omitempty"`



	// ValidityPeriod specifies the certificate validity period.

	ValidityPeriod *metav1.Duration `json:"validityPeriod,omitempty"`



	// DNSNames specifies additional DNS names for the certificate.

	DNSNames []string `json:"dnsNames,omitempty"`



	// IPAddresses specifies IP addresses for the certificate.

	IPAddresses []string `json:"ipAddresses,omitempty"`



	// AutoRenew enables automatic certificate renewal.

	AutoRenew bool `json:"autoRenew,omitempty"`



	// RenewalThreshold specifies when to renew (e.g., "30d" before expiry).

	RenewalThreshold *metav1.Duration `json:"renewalThreshold,omitempty"`



	// KeySize specifies the private key size.

	KeySize int `json:"keySize,omitempty"`



	// Algorithm specifies the signing algorithm.

	Algorithm string `json:"algorithm,omitempty"`



	// Priority specifies the priority of this automation.

	Priority string `json:"priority,omitempty"`

}



// CertificateAutomationStatus defines the observed state of CertificateAutomation.

type CertificateAutomationStatus struct {

	// Phase represents the current phase of certificate automation.

	Phase CertificateAutomationPhase `json:"phase,omitempty"`



	// Conditions represents the current conditions.

	Conditions []CertificateAutomationCondition `json:"conditions,omitempty"`



	// CertificateSerialNumber contains the serial number of the issued certificate.

	CertificateSerialNumber string `json:"certificateSerialNumber,omitempty"`



	// ExpirationTime contains the certificate expiration time.

	ExpirationTime *metav1.Time `json:"expirationTime,omitempty"`



	// ExpiresAt contains the certificate expiration time (alternative name).

	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`



	// LastRenewalTime contains the last renewal timestamp.

	LastRenewalTime *metav1.Time `json:"lastRenewalTime,omitempty"`



	// NextRenewalTime contains the next scheduled renewal time.

	NextRenewalTime *metav1.Time `json:"nextRenewalTime,omitempty"`



	// ValidationStatus contains the certificate validation status.

	ValidationStatus string `json:"validationStatus,omitempty"`



	// SecretName contains the name of the secret storing the certificate.

	SecretName string `json:"secretName,omitempty"`



	// ObservedGeneration reflects the generation observed by the controller.

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

}



// CertificateAutomationPhase represents the phase of certificate automation.

type CertificateAutomationPhase string



const (

	// CertificateAutomationPhasePending indicates the request is pending.

	CertificateAutomationPhasePending CertificateAutomationPhase = "Pending"



	// CertificateAutomationPhaseProvisioning indicates certificate is being provisioned.

	CertificateAutomationPhaseProvisioning CertificateAutomationPhase = "Provisioning"



	// CertificateAutomationPhaseReady indicates certificate is ready.

	CertificateAutomationPhaseReady CertificateAutomationPhase = "Ready"



	// CertificateAutomationPhaseRenewing indicates certificate is being renewed.

	CertificateAutomationPhaseRenewing CertificateAutomationPhase = "Renewing"



	// CertificateAutomationPhaseExpired indicates certificate has expired.

	CertificateAutomationPhaseExpired CertificateAutomationPhase = "Expired"



	// CertificateAutomationPhaseRevoked indicates certificate has been revoked.

	CertificateAutomationPhaseRevoked CertificateAutomationPhase = "Revoked"



	// CertificateAutomationPhaseFailed indicates an error occurred.

	CertificateAutomationPhaseFailed CertificateAutomationPhase = "Failed"

)



// CertificateAutomationCondition describes the state of certificate automation.

type CertificateAutomationCondition struct {

	// Type of condition.

	Type CertificateAutomationConditionType `json:"type"`



	// Status of the condition.

	Status metav1.ConditionStatus `json:"status"`



	// LastTransitionTime is the last time the condition transitioned.

	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`



	// Reason contains a programmatic identifier indicating the reason for the condition.

	Reason string `json:"reason,omitempty"`



	// Message contains a human readable message indicating details about the transition.

	Message string `json:"message,omitempty"`

}



// CertificateAutomationConditionType represents condition types.

type CertificateAutomationConditionType string



const (

	// CertificateAutomationConditionReady indicates the certificate is ready.

	CertificateAutomationConditionReady CertificateAutomationConditionType = "Ready"



	// CertificateAutomationConditionIssued indicates the certificate has been issued.

	CertificateAutomationConditionIssued CertificateAutomationConditionType = "Issued"



	// CertificateAutomationConditionValidated indicates the certificate has been validated.

	CertificateAutomationConditionValidated CertificateAutomationConditionType = "Validated"



	// CertificateAutomationConditionRenewed indicates the certificate has been renewed.

	CertificateAutomationConditionRenewed CertificateAutomationConditionType = "Renewed"



	// CertificateAutomationConditionRevoked indicates the certificate has been revoked.

	CertificateAutomationConditionRevoked CertificateAutomationConditionType = "Revoked"

)



//+kubebuilder:object:root=true

//+kubebuilder:subresource:status

//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"

//+kubebuilder:printcolumn:name="Service",type="string",JSONPath=".spec.serviceName"

//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"



// CertificateAutomation is the Schema for the certificateautomations API.

type CertificateAutomation struct {

	metav1.TypeMeta   `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`



	Spec   CertificateAutomationSpec   `json:"spec,omitempty"`

	Status CertificateAutomationStatus `json:"status,omitempty"`

}



//+kubebuilder:object:root=true



// CertificateAutomationList contains a list of CertificateAutomation.

type CertificateAutomationList struct {

	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items           []CertificateAutomation `json:"items"`

}

func init() {
	SchemeBuilder.Register(&CertificateAutomation{}, &CertificateAutomationList{})
}

