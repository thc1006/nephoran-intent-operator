package ca

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertManagerBackendStub is a stub implementation for cert-manager integration
type CertManagerBackendStub struct {
	client client.Client
	config *CertManagerConfig
}

// NewCertManagerBackendStub creates a new stubbed cert-manager backend (renamed to avoid conflict)
func NewCertManagerBackendStub(config *CertManagerConfig, kubeClient client.Client) Backend {
	return &CertManagerBackendStub{
		client: kubeClient,
		config: config,
	}
}

// Initialize initializes the cert-manager backend (stubbed)
func (b *CertManagerBackendStub) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

// IssueCertificate issues a certificate (stubbed)
func (b *CertManagerBackendStub) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	return nil, fmt.Errorf("cert-manager integration not implemented")
}

// HealthCheck performs a health check (stubbed)
func (b *CertManagerBackendStub) HealthCheck(ctx context.Context) error {
	return nil
}

// GetCAChain retrieves the CA certificate chain (stubbed)
func (b *CertManagerBackendStub) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {
	return nil, fmt.Errorf("cert-manager integration not implemented")
}

// GetCRL retrieves the Certificate Revocation List (stubbed)
func (b *CertManagerBackendStub) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {
	return nil, fmt.Errorf("cert-manager integration not implemented")
}

// GetSupportedFeatures returns list of supported features (stubbed)
func (b *CertManagerBackendStub) GetSupportedFeatures() []string {
	return []string{"stubbed"}
}

// RevokeCertificate revokes a certificate (stubbed)
func (b *CertManagerBackendStub) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {
	return fmt.Errorf("cert-manager integration not implemented")
}

// RenewCertificate renews a certificate (stubbed)
func (b *CertManagerBackendStub) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	return nil, fmt.Errorf("cert-manager integration not implemented")
}

// GetBackendInfo returns backend information (stubbed)
func (b *CertManagerBackendStub) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {
	return &BackendInfo{
		Type:    "kubernetes",
		Status:  "stubbed",
		Version: "stub-1.0.0",
	}, nil
}