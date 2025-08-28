// Package security provides enhanced certificate management with automation.
package security

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// CertManager provides comprehensive certificate lifecycle management.
type CertManager struct {
	// ACME/Let's Encrypt configuration.
	acmeManager *autocert.Manager
	acmeClient  *acme.Client

	// Certificate storage.
	certStore CertificateStore

	// Root and intermediate CAs.
	rootCA          *x509.Certificate
	rootKey         crypto.PrivateKey
	intermediateCA  *x509.Certificate
	intermediateKey crypto.PrivateKey

	// Certificate transparency.
	ctLogs      []string
	ctSubmitter *CTSubmitter

	// OCSP responder.
	ocspResponder *OCSPResponder

	// CRL management.
	crlManager *CRLManager

	// Certificate rotation.
	rotationScheduler *CertRotationScheduler

	// Certificate pinning.
	pinnedCerts map[string][]byte

	// Monitoring.
	monitor *CertMonitor

	mu sync.RWMutex
}

// CertificateStore interface for certificate storage.
type CertificateStore interface {
	Get(ctx context.Context, name string) (*tls.Certificate, error)
	Put(ctx context.Context, name string, cert *tls.Certificate) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]string, error)
}

// CTSubmitter handles Certificate Transparency submissions.
type CTSubmitter struct {
	logs    []string
	client  *http.Client
	timeout time.Duration
}

// OCSPResponder provides OCSP responses.
type OCSPResponder struct {
	signer    crypto.Signer
	cert      *x509.Certificate
	issuer    *x509.Certificate
	responses map[string]*OCSPResponse
	mu        sync.RWMutex
}

// OCSPResponse represents an OCSP response.
type OCSPResponse struct {
	Status    int
	RevokedAt time.Time
	Reason    int
	UpdatedAt time.Time
}

// CRLManager manages Certificate Revocation Lists.
type CRLManager struct {
	revoked    map[string]*RevokedCert
	crl        *x509.RevocationList
	signer     crypto.Signer
	issuer     *x509.Certificate
	nextUpdate time.Time
	mu         sync.RWMutex
}

// RevokedCert represents a revoked certificate.
type RevokedCert struct {
	SerialNumber *big.Int
	RevokedAt    time.Time
	Reason       int
}

// CertRotationScheduler handles automatic certificate rotation.
type CertRotationScheduler struct {
	rotations map[string]*RotationConfig
	ticker    *time.Ticker
	done      chan bool
	mu        sync.RWMutex
}

// RotationConfig defines certificate rotation parameters.
type RotationConfig struct {
	Name          string
	CheckInterval time.Duration
	RenewBefore   time.Duration
	RenewCallback func(name string, cert *tls.Certificate) error
	ErrorCallback func(name string, err error)
}

// CertMonitor monitors certificate health and expiry.
type CertMonitor struct {
	certificates map[string]*MonitoredCert
	alerts       chan *CertAlert
	mu           sync.RWMutex
}

// MonitoredCert represents a monitored certificate.
type MonitoredCert struct {
	Name       string
	Cert       *x509.Certificate
	ExpiryTime time.Time
	Healthy    bool
	LastCheck  time.Time
}

// CertAlert represents a certificate alert.
type CertAlert struct {
	Type      string
	Severity  string
	Name      string
	Message   string
	Timestamp time.Time
}

// NewCertManager creates a new certificate manager.
func NewCertManager(store CertificateStore) *CertManager {
	return &CertManager{
		certStore: store,
		ctLogs: []string{
			"https://ct.googleapis.com/logs/xenon2024/",
			"https://ct.cloudflare.com/logs/nimbus2024/",
		},
		ctSubmitter: &CTSubmitter{
			client:  &http.Client{Timeout: 10 * time.Second},
			timeout: 10 * time.Second,
		},
		ocspResponder: &OCSPResponder{
			responses: make(map[string]*OCSPResponse),
		},
		crlManager: &CRLManager{
			revoked: make(map[string]*RevokedCert),
		},
		rotationScheduler: &CertRotationScheduler{
			rotations: make(map[string]*RotationConfig),
		},
		pinnedCerts: make(map[string][]byte),
		monitor: &CertMonitor{
			certificates: make(map[string]*MonitoredCert),
			alerts:       make(chan *CertAlert, 100),
		},
	}
}

// SetupACME configures ACME/Let's Encrypt.
func (cm *CertManager) SetupACME(email string, domains []string, cacheDir string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.acmeManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Email:      email,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(cacheDir),
	}

	// Create ACME client.
	cm.acmeClient = &acme.Client{
		DirectoryURL: acme.LetsEncryptURL,
	}

	return nil
}

// GetACMECertificate obtains a certificate via ACME.
func (cm *CertManager) GetACMECertificate(domain string) (*tls.Certificate, error) {
	if cm.acmeManager == nil {
		return nil, errors.New("ACME not configured")
	}

	// Get certificate from cache or request new one.
	cert, err := cm.acmeManager.GetCertificate(&tls.ClientHelloInfo{
		ServerName: domain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ACME certificate: %w", err)
	}

	// Submit to CT logs.
	if err := cm.submitToCTLogs(cert); err != nil {
		// Log error but don't fail certificate issuance.
		fmt.Printf("CT submission failed: %v\n", err)
	}

	// Store certificate.
	if err := cm.certStore.Put(context.Background(), domain, cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	// Start monitoring.
	cm.monitor.AddCertificate(domain, cert)

	return cert, nil
}

// GenerateRootCA generates a new root CA.
func (cm *CertManager) GenerateRootCA(commonName string, validYears int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Generate key.
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}

	// Create certificate template.
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Nephoran Intent Operator"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(validYears, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            2,
	}

	// Create certificate.
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create root certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse root certificate: %w", err)
	}

	cm.rootCA = cert
	cm.rootKey = key

	return nil
}

// GenerateIntermediateCA generates an intermediate CA.
func (cm *CertManager) GenerateIntermediateCA(commonName string, validYears int) error {
	if cm.rootCA == nil || cm.rootKey == nil {
		return errors.New("root CA not configured")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Generate key.
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate intermediate key: %w", err)
	}

	// Create certificate template.
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Nephoran Intent Operator"},
			Country:      []string{"US"},
			CommonName:   commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(validYears, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
	}

	// Create certificate signed by root CA.
	certDER, err := x509.CreateCertificate(rand.Reader, template, cm.rootCA, &key.PublicKey, cm.rootKey)
	if err != nil {
		return fmt.Errorf("failed to create intermediate certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse intermediate certificate: %w", err)
	}

	cm.intermediateCA = cert
	cm.intermediateKey = key

	return nil
}

// IssueCertificate issues a new certificate.
func (cm *CertManager) IssueCertificate(commonName string, hosts []string, validDays int) (*tls.Certificate, error) {
	if cm.intermediateCA == nil || cm.intermediateKey == nil {
		return nil, errors.New("intermediate CA not configured")
	}

	// Generate key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Create certificate template.
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Nephoran Intent Operator"},
			CommonName:   commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, validDays),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	// Add hosts.
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, host)
		}
	}

	// Create certificate signed by intermediate CA.
	certDER, err := x509.CreateCertificate(rand.Reader, template, cm.intermediateCA, &key.PublicKey, cm.intermediateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create TLS certificate.
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certDER, cm.intermediateCA.Raw},
		PrivateKey:  key,
	}

	// Parse leaf certificate.
	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	tlsCert.Leaf = leaf

	// Store certificate.
	if err := cm.certStore.Put(context.Background(), commonName, tlsCert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	// Submit to CT logs.
	if err := cm.submitToCTLogs(tlsCert); err != nil {
		fmt.Printf("CT submission failed: %v\n", err)
	}

	// Start monitoring.
	cm.monitor.AddCertificate(commonName, tlsCert)

	return tlsCert, nil
}

// RevokeCertificate revokes a certificate.
func (cm *CertManager) RevokeCertificate(serialNumber *big.Int, reason int) error {
	cm.crlManager.mu.Lock()
	defer cm.crlManager.mu.Unlock()

	// Add to revoked list.
	cm.crlManager.revoked[serialNumber.String()] = &RevokedCert{
		SerialNumber: serialNumber,
		RevokedAt:    time.Now(),
		Reason:       reason,
	}

	// Update CRL.
	if err := cm.crlManager.updateCRL(); err != nil {
		return fmt.Errorf("failed to update CRL: %w", err)
	}

	// Update OCSP response.
	cm.ocspResponder.mu.Lock()
	cm.ocspResponder.responses[serialNumber.String()] = &OCSPResponse{
		Status:    1, // Revoked
		RevokedAt: time.Now(),
		Reason:    reason,
		UpdatedAt: time.Now(),
	}
	cm.ocspResponder.mu.Unlock()

	return nil
}

// ScheduleRotation schedules automatic certificate rotation.
func (cm *CertManager) ScheduleRotation(config *RotationConfig) error {
	cm.rotationScheduler.mu.Lock()
	defer cm.rotationScheduler.mu.Unlock()

	cm.rotationScheduler.rotations[config.Name] = config

	// Start rotation scheduler if not running.
	if cm.rotationScheduler.ticker == nil {
		cm.rotationScheduler.Start()
	}

	return nil
}

// Start rotation scheduler.
func (rs *CertRotationScheduler) Start() {
	rs.ticker = time.NewTicker(1 * time.Hour)
	rs.done = make(chan bool)

	go func() {
		for {
			select {
			case <-rs.ticker.C:
				rs.checkRotations()
			case <-rs.done:
				return
			}
		}
	}()
}

// checkRotations checks and performs necessary rotations.
func (rs *CertRotationScheduler) checkRotations() {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	for name, config := range rs.rotations {
		go rs.checkAndRotate(name, config)
	}
}

// checkAndRotate checks and rotates a single certificate.
func (rs *CertRotationScheduler) checkAndRotate(name string, config *RotationConfig) {
	// Implementation would check certificate expiry and rotate if needed.
	// This is a placeholder for the actual implementation.
}

// submitToCTLogs submits certificate to CT logs.
func (cm *CertManager) submitToCTLogs(cert *tls.Certificate) error {
	if len(cert.Certificate) == 0 {
		return errors.New("no certificate to submit")
	}

	// Submit to each CT log.
	for _, logURL := range cm.ctLogs {
		if err := cm.ctSubmitter.Submit(logURL, cert.Certificate[0]); err != nil {
			// Continue submitting to other logs even if one fails.
			fmt.Printf("Failed to submit to CT log %s: %v\n", logURL, err)
		}
	}

	return nil
}

// Submit submits certificate to a CT log.
func (cts *CTSubmitter) Submit(logURL string, certDER []byte) error {
	// This would implement actual CT submission.
	// Placeholder for actual implementation.
	return nil
}

// updateCRL updates the Certificate Revocation List.
func (crlm *CRLManager) updateCRL() error {
	// Build revoked certificate list.
	var revokedCerts []x509.RevocationListEntry
	for _, revoked := range crlm.revoked {
		revokedCerts = append(revokedCerts, x509.RevocationListEntry{
			SerialNumber:   revoked.SerialNumber,
			RevocationTime: revoked.RevokedAt,
		})
	}

	// Create CRL template.
	template := &x509.RevocationList{
		RevokedCertificateEntries: revokedCerts,
		ThisUpdate:                time.Now(),
		NextUpdate:                time.Now().Add(24 * time.Hour),
	}

	// Sign CRL.
	crlDER, err := x509.CreateRevocationList(rand.Reader, template, crlm.issuer, crlm.signer)
	if err != nil {
		return fmt.Errorf("failed to create CRL: %w", err)
	}

	crl, err := x509.ParseRevocationList(crlDER)
	if err != nil {
		return fmt.Errorf("failed to parse CRL: %w", err)
	}

	crlm.crl = crl
	crlm.nextUpdate = template.NextUpdate

	return nil
}

// AddCertificate adds a certificate to monitoring.
func (cm *CertMonitor) AddCertificate(name string, cert *tls.Certificate) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cert.Leaf == nil && len(cert.Certificate) > 0 {
		// Parse leaf certificate if not already parsed.
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			cm.alerts <- &CertAlert{
				Type:      "PARSE_ERROR",
				Severity:  "ERROR",
				Name:      name,
				Message:   fmt.Sprintf("Failed to parse certificate: %v", err),
				Timestamp: time.Now(),
			}
			return
		}
		cert.Leaf = leaf
	}

	cm.certificates[name] = &MonitoredCert{
		Name:       name,
		Cert:       cert.Leaf,
		ExpiryTime: cert.Leaf.NotAfter,
		Healthy:    true,
		LastCheck:  time.Now(),
	}

	// Check if certificate is expiring soon.
	daysUntilExpiry := time.Until(cert.Leaf.NotAfter).Hours() / 24
	if daysUntilExpiry < 30 {
		severity := "WARNING"
		if daysUntilExpiry < 7 {
			severity = "CRITICAL"
		}

		cm.alerts <- &CertAlert{
			Type:      "EXPIRY_WARNING",
			Severity:  severity,
			Name:      name,
			Message:   fmt.Sprintf("Certificate expires in %.0f days", daysUntilExpiry),
			Timestamp: time.Now(),
		}
	}
}

// PinCertificate adds a certificate to the pinning list.
func (cm *CertManager) PinCertificate(name string, certDER []byte) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.pinnedCerts[name] = certDER
}

// ValidatePinnedCertificate validates a certificate against pinned certificates.
func (cm *CertManager) ValidatePinnedCertificate(name string, cert *x509.Certificate) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	pinnedDER, ok := cm.pinnedCerts[name]
	if !ok {
		return nil // No pinning for this certificate
	}

	if !equalDER(cert.Raw, pinnedDER) {
		return errors.New("certificate does not match pinned certificate")
	}

	return nil
}

// equalDER compares two DER-encoded certificates.
func equalDER(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ExportCertificate exports a certificate in PEM format.
func ExportCertificate(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// ExportPrivateKey exports a private key in PEM format.
func ExportPrivateKey(key crypto.PrivateKey) ([]byte, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}), nil
	default:
		return nil, errors.New("unsupported key type")
	}
}
