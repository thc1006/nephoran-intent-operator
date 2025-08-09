// Package abstraction provides a no-op service mesh implementation
package abstraction

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// NoOpMesh provides a no-op implementation of ServiceMeshInterface
type NoOpMesh struct {
	certProvider *NoOpCertificateProvider
}

// NewNoOpMesh creates a new no-op service mesh
func NewNoOpMesh() *NoOpMesh {
	return &NoOpMesh{
		certProvider: &NoOpCertificateProvider{},
	}
}

// Initialize initializes the no-op mesh
func (n *NoOpMesh) Initialize(ctx context.Context, config *ServiceMeshConfig) error {
	// No-op: always succeeds
	return nil
}

// GetCertificateProvider returns the certificate provider
func (n *NoOpMesh) GetCertificateProvider() CertificateProvider {
	return n.certProvider
}

// RotateCertificates rotates certificates (no-op)
func (n *NoOpMesh) RotateCertificates(ctx context.Context, namespace string) error {
	// No-op: always succeeds
	return nil
}

// ValidateCertificateChain validates the certificate chain (no-op)
func (n *NoOpMesh) ValidateCertificateChain(ctx context.Context, namespace string) error {
	// No-op: always succeeds
	return nil
}

// ApplyMTLSPolicy applies an mTLS policy (no-op)
func (n *NoOpMesh) ApplyMTLSPolicy(ctx context.Context, policy *MTLSPolicy) error {
	// No-op: always succeeds
	return nil
}

// ApplyAuthorizationPolicy applies an authorization policy (no-op)
func (n *NoOpMesh) ApplyAuthorizationPolicy(ctx context.Context, policy *AuthorizationPolicy) error {
	// No-op: always succeeds
	return nil
}

// ApplyTrafficPolicy applies a traffic policy (no-op)
func (n *NoOpMesh) ApplyTrafficPolicy(ctx context.Context, policy *TrafficPolicy) error {
	// No-op: always succeeds
	return nil
}

// ValidatePolicies validates policies (no-op)
func (n *NoOpMesh) ValidatePolicies(ctx context.Context, namespace string) (*PolicyValidationResult, error) {
	return &PolicyValidationResult{
		Valid:    true,
		Coverage: 0,
		Compliance: PolicyCompliance{
			MTLSCompliant:      false,
			ZeroTrustCompliant: false,
			NetworkSegmented:   false,
			ComplianceScore:    0,
		},
	}, nil
}

// RegisterService registers a service (no-op)
func (n *NoOpMesh) RegisterService(ctx context.Context, service *ServiceRegistration) error {
	// No-op: always succeeds
	return nil
}

// UnregisterService unregisters a service (no-op)
func (n *NoOpMesh) UnregisterService(ctx context.Context, serviceName string, namespace string) error {
	// No-op: always succeeds
	return nil
}

// GetServiceStatus gets service status (no-op)
func (n *NoOpMesh) GetServiceStatus(ctx context.Context, serviceName string, namespace string) (*ServiceStatus, error) {
	return &ServiceStatus{
		Name:        serviceName,
		Namespace:   namespace,
		Healthy:     true,
		MTLSEnabled: false,
	}, nil
}

// GetMetrics returns metrics collectors (empty for no-op)
func (n *NoOpMesh) GetMetrics() []prometheus.Collector {
	return []prometheus.Collector{}
}

// GetServiceDependencies gets service dependencies (no-op)
func (n *NoOpMesh) GetServiceDependencies(ctx context.Context, namespace string) (*DependencyGraph, error) {
	return &DependencyGraph{
		Nodes: []ServiceNode{},
		Edges: []ServiceEdge{},
	}, nil
}

// GetMTLSStatus gets mTLS status (no-op)
func (n *NoOpMesh) GetMTLSStatus(ctx context.Context, namespace string) (*MTLSStatusReport, error) {
	return &MTLSStatusReport{
		TotalServices:    0,
		MTLSEnabledCount: 0,
		Coverage:         0,
	}, nil
}

// IsHealthy checks if the mesh is healthy (always true for no-op)
func (n *NoOpMesh) IsHealthy(ctx context.Context) error {
	return nil
}

// IsReady checks if the mesh is ready (always true for no-op)
func (n *NoOpMesh) IsReady(ctx context.Context) error {
	return nil
}

// GetProvider returns the provider type
func (n *NoOpMesh) GetProvider() ServiceMeshProvider {
	return ProviderNone
}

// GetVersion returns the version
func (n *NoOpMesh) GetVersion() string {
	return "none"
}

// GetCapabilities returns capabilities (empty for no-op)
func (n *NoOpMesh) GetCapabilities() []Capability {
	return []Capability{}
}

// NoOpCertificateProvider provides a no-op certificate provider
type NoOpCertificateProvider struct{}

// IssueCertificate issues a certificate (returns error for no-op)
func (n *NoOpCertificateProvider) IssueCertificate(ctx context.Context, service string, namespace string) (*x509.Certificate, error) {
	return nil, fmt.Errorf("certificate management not available without service mesh")
}

// GetRootCA gets the root CA (returns error for no-op)
func (n *NoOpCertificateProvider) GetRootCA(ctx context.Context) (*x509.Certificate, error) {
	return nil, fmt.Errorf("certificate management not available without service mesh")
}

// GetIntermediateCA gets the intermediate CA (returns error for no-op)
func (n *NoOpCertificateProvider) GetIntermediateCA(ctx context.Context) (*x509.Certificate, error) {
	return nil, fmt.Errorf("certificate management not available without service mesh")
}

// ValidateCertificate validates a certificate (returns error for no-op)
func (n *NoOpCertificateProvider) ValidateCertificate(ctx context.Context, cert *x509.Certificate) error {
	return fmt.Errorf("certificate management not available without service mesh")
}

// RotateCertificate rotates a certificate (returns error for no-op)
func (n *NoOpCertificateProvider) RotateCertificate(ctx context.Context, service string, namespace string) (*x509.Certificate, error) {
	return nil, fmt.Errorf("certificate management not available without service mesh")
}

// GetCertificateChain gets the certificate chain (returns error for no-op)
func (n *NoOpCertificateProvider) GetCertificateChain(ctx context.Context, service string, namespace string) ([]*x509.Certificate, error) {
	return nil, fmt.Errorf("certificate management not available without service mesh")
}

// GetSPIFFEID gets the SPIFFE ID for a service
func (n *NoOpCertificateProvider) GetSPIFFEID(service string, namespace string, trustDomain string) string {
	return fmt.Sprintf("spiffe://%s/ns/%s/sa/%s", trustDomain, namespace, service)
}
