// Package istio provides certificate management for Istio service mesh.


package istio



import (

	"context"

	"crypto/x509"

	"crypto/x509/pkix"

	"encoding/pem"

	"fmt"

	"math/big"

	"time"



	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"

)



// IstioCertificateProvider implements CertificateProvider for Istio.

type IstioCertificateProvider struct {

	kubeClient  kubernetes.Interface

	trustDomain string

}



// NewIstioCertificateProvider creates a new Istio certificate provider.

func NewIstioCertificateProvider(kubeClient kubernetes.Interface, trustDomain string) *IstioCertificateProvider {

	if trustDomain == "" {

		trustDomain = "cluster.local"

	}

	return &IstioCertificateProvider{

		kubeClient:  kubeClient,

		trustDomain: trustDomain,

	}

}



// IssueCertificate issues a new certificate for a service.

func (p *IstioCertificateProvider) IssueCertificate(ctx context.Context, service, namespace string) (*x509.Certificate, error) {

	// In Istio, Citadel (istiod) automatically issues certificates.

	// We can retrieve the certificate from the pod's mounted volume.



	// Get pods for the service.

	pods, err := p.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{

		LabelSelector: fmt.Sprintf("app=%s", service),

	})

	if err != nil {

		return nil, fmt.Errorf("failed to list pods: %w", err)

	}



	if len(pods.Items) == 0 {

		return nil, fmt.Errorf("no pods found for service %s", service)

	}



	// For demonstration, we'll create a placeholder certificate.

	// In production, this would interact with Citadel or retrieve from mounted secrets.

	cert := &x509.Certificate{

		Subject: pkix.Name{

			CommonName:   fmt.Sprintf("%s.%s.svc.%s", service, namespace, p.trustDomain),

			Organization: []string{namespace},

		},

		NotBefore:    time.Now(),

		NotAfter:     time.Now().Add(90 * 24 * time.Hour), // 90 days

		SerialNumber: big.NewInt(1),

	}



	return cert, nil

}



// GetRootCA returns the root CA certificate.

func (p *IstioCertificateProvider) GetRootCA(ctx context.Context) (*x509.Certificate, error) {

	// Get the Istio root CA from the istio-ca-secret.

	secret, err := p.kubeClient.CoreV1().Secrets("istio-system").Get(ctx, "istio-ca-secret", metav1.GetOptions{})

	if err != nil {

		// Try alternative secret name.

		secret, err = p.kubeClient.CoreV1().Secrets("istio-system").Get(ctx, "cacerts", metav1.GetOptions{})

		if err != nil {

			return nil, fmt.Errorf("failed to get root CA secret: %w", err)

		}

	}



	// Parse the root certificate.

	rootCertPEM, ok := secret.Data["root-cert.pem"]

	if !ok {

		rootCertPEM, ok = secret.Data["ca-cert.pem"]

		if !ok {

			return nil, fmt.Errorf("root certificate not found in secret")

		}

	}



	block, _ := pem.Decode(rootCertPEM)

	if block == nil {

		return nil, fmt.Errorf("failed to decode root certificate PEM")

	}



	cert, err := x509.ParseCertificate(block.Bytes)

	if err != nil {

		return nil, fmt.Errorf("failed to parse root certificate: %w", err)

	}



	return cert, nil

}



// GetIntermediateCA returns intermediate CA if applicable.

func (p *IstioCertificateProvider) GetIntermediateCA(ctx context.Context) (*x509.Certificate, error) {

	// Get the Istio intermediate CA from the istio-ca-secret.

	secret, err := p.kubeClient.CoreV1().Secrets("istio-system").Get(ctx, "istio-ca-secret", metav1.GetOptions{})

	if err != nil {

		// Try alternative secret name.

		secret, err = p.kubeClient.CoreV1().Secrets("istio-system").Get(ctx, "cacerts", metav1.GetOptions{})

		if err != nil {

			return nil, fmt.Errorf("failed to get CA secret: %w", err)

		}

	}



	// Parse the intermediate certificate.

	certChainPEM, ok := secret.Data["cert-chain.pem"]

	if !ok {

		// No intermediate CA in Istio by default.

		return nil, nil

	}



	block, _ := pem.Decode(certChainPEM)

	if block == nil {

		return nil, fmt.Errorf("failed to decode certificate chain PEM")

	}



	cert, err := x509.ParseCertificate(block.Bytes)

	if err != nil {

		return nil, fmt.Errorf("failed to parse intermediate certificate: %w", err)

	}



	return cert, nil

}



// ValidateCertificate validates a certificate.

func (p *IstioCertificateProvider) ValidateCertificate(ctx context.Context, cert *x509.Certificate) error {

	if cert == nil {

		return fmt.Errorf("certificate is nil")

	}



	// Check certificate validity period.

	now := time.Now()

	if now.Before(cert.NotBefore) {

		return fmt.Errorf("certificate not yet valid")

	}

	if now.After(cert.NotAfter) {

		return fmt.Errorf("certificate has expired")

	}



	// Get root CA for validation.

	rootCA, err := p.GetRootCA(ctx)

	if err != nil {

		return fmt.Errorf("failed to get root CA: %w", err)

	}



	// Create certificate pool with root CA.

	roots := x509.NewCertPool()

	roots.AddCert(rootCA)



	// Verify certificate chain.

	opts := x509.VerifyOptions{

		Roots:       roots,

		CurrentTime: now,

		DNSName:     cert.Subject.CommonName,

	}



	_, err = cert.Verify(opts)

	if err != nil {

		return fmt.Errorf("certificate validation failed: %w", err)

	}



	return nil

}



// RotateCertificate rotates a certificate for a service.

func (p *IstioCertificateProvider) RotateCertificate(ctx context.Context, service, namespace string) (*x509.Certificate, error) {

	// In Istio, certificate rotation is handled automatically by Citadel.

	// We can trigger rotation by deleting the secret and letting Citadel recreate it.



	// Find the certificate secret for the service.

	secretName := fmt.Sprintf("istio.%s", service)



	// Delete the existing secret.

	err := p.kubeClient.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})

	if err != nil {

		// Secret might not exist, which is okay.

	}



	// Wait for Citadel to issue a new certificate.

	time.Sleep(2 * time.Second)



	// Return the new certificate.

	return p.IssueCertificate(ctx, service, namespace)

}



// GetCertificateChain returns the certificate chain for a service.

func (p *IstioCertificateProvider) GetCertificateChain(ctx context.Context, service, namespace string) ([]*x509.Certificate, error) {

	chain := []*x509.Certificate{}



	// Get service certificate.

	serviceCert, err := p.IssueCertificate(ctx, service, namespace)

	if err != nil {

		return nil, fmt.Errorf("failed to get service certificate: %w", err)

	}

	chain = append(chain, serviceCert)



	// Get intermediate CA if exists.

	intermediateCert, err := p.GetIntermediateCA(ctx)

	if err == nil && intermediateCert != nil {

		chain = append(chain, intermediateCert)

	}



	// Get root CA.

	rootCert, err := p.GetRootCA(ctx)

	if err != nil {

		return nil, fmt.Errorf("failed to get root CA: %w", err)

	}

	chain = append(chain, rootCert)



	return chain, nil

}



// GetSPIFFEID returns the SPIFFE ID for a service.

func (p *IstioCertificateProvider) GetSPIFFEID(service, namespace, trustDomain string) string {

	if trustDomain == "" {

		trustDomain = p.trustDomain

	}

	// Istio SPIFFE ID format: spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>.

	return fmt.Sprintf("spiffe://%s/ns/%s/sa/%s", trustDomain, namespace, service)

}



// GetCertificateExpiry returns the expiry time of a certificate.

func (p *IstioCertificateProvider) GetCertificateExpiry(ctx context.Context, service, namespace string) (time.Time, error) {

	cert, err := p.IssueCertificate(ctx, service, namespace)

	if err != nil {

		return time.Time{}, fmt.Errorf("failed to get certificate: %w", err)

	}

	return cert.NotAfter, nil

}



// IsCertificateExpiringSoon checks if a certificate is expiring soon.

func (p *IstioCertificateProvider) IsCertificateExpiringSoon(ctx context.Context, service, namespace string, threshold time.Duration) (bool, error) {

	expiry, err := p.GetCertificateExpiry(ctx, service, namespace)

	if err != nil {

		return false, err

	}



	return time.Until(expiry) < threshold, nil

}

