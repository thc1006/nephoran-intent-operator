package ca

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cert-manager API stubs for compilation

// KeyAlgorithm represents the key algorithm type
type KeyAlgorithm string

// PrivateKeyEncoding represents the private key encoding
type PrivateKeyEncoding string

// KeyUsage represents certificate key usage
type KeyUsage string

// ObjectReference represents a reference to a Kubernetes object
type ObjectReference struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Group     string `json:"group,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// Common key algorithms
const (
	RSA   KeyAlgorithm = "RSA"
	ECDSA KeyAlgorithm = "ECDSA"
	Ed25519 KeyAlgorithm = "Ed25519"
)

// Common key usages
const (
	UsageDigitalSignature  KeyUsage = "digital signature"
	UsageContentCommitment KeyUsage = "content commitment"
	UsageKeyEncipherment   KeyUsage = "key encipherment"
	UsageDataEncipherment  KeyUsage = "data encipherment"
	UsageKeyAgreement      KeyUsage = "key agreement"
	UsageCertSign          KeyUsage = "cert sign"
	UsageCRLSign           KeyUsage = "crl sign"
)

// Certificate represents a cert-manager Certificate
type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CertificateSpec      `json:"spec,omitempty"`
	Status            CertManagerStatus    `json:"status,omitempty"`
}

// CertificateSpec defines the desired certificate
type CertificateSpec struct {
	SecretName  string                 `json:"secretName"`
	IssuerRef   ObjectReference        `json:"issuerRef"`
	CommonName  string                 `json:"commonName,omitempty"`
	DNSNames    []string               `json:"dnsNames,omitempty"`
	IPAddresses []string               `json:"ipAddresses,omitempty"`
	Subject     *X509Subject           `json:"subject,omitempty"`
	Duration    *metav1.Duration       `json:"duration,omitempty"`
	PrivateKey  *CertificatePrivateKey `json:"privateKey,omitempty"`
	Usages      []KeyUsage             `json:"usages,omitempty"`
}

// CertManagerStatus represents certificate status for cert-manager resources
// (renamed from CertificateStatus to avoid conflict with string-based CertificateStatus in manager.go)
type CertManagerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// X509Subject represents X.509 certificate subject
type X509Subject struct {
	Organizations       []string `json:"organizations,omitempty"`
	Countries           []string `json:"countries,omitempty"`
	OrganizationalUnits []string `json:"organizationalUnits,omitempty"`
}

// CertificatePrivateKey represents private key configuration
type CertificatePrivateKey struct {
	Algorithm KeyAlgorithm       `json:"algorithm"`
	Size      int                `json:"size,omitempty"`
	Encoding  PrivateKeyEncoding `json:"encoding,omitempty"`
}

// Issuer represents a cert-manager Issuer
type Issuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IssuerSpec `json:"spec,omitempty"`
}

// IssuerSpec defines the desired state of Issuer
type IssuerSpec struct {
	CA     *CAIssuer     `json:"ca,omitempty"`
	SelfSigned *SelfSignedIssuer `json:"selfSigned,omitempty"`
}

// CAIssuer represents a CA issuer configuration
type CAIssuer struct {
	SecretName string `json:"secretName"`
}

// SelfSignedIssuer represents a self-signed issuer configuration  
type SelfSignedIssuer struct{}