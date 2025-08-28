package ca

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificatePool manages certificate storage and retrieval.
type CertificatePool struct {
	config       *CertificateStoreConfig
	logger       *logging.StructuredLogger
	client       client.Client
	certificates map[string]*CertificateResponse
	indices      *CertificateIndices
	mu           sync.RWMutex
	encryptor    *CertificateEncryptor
	lastBackup   time.Time
}

// CertificateIndices provides efficient certificate lookup.
type CertificateIndices struct {
	ByTenant      map[string][]*CertificateResponse
	ByStatus      map[CertificateStatus][]*CertificateResponse
	ByExpiryDate  *ExpiryIndex
	ByFingerprint map[string]*CertificateResponse
	ByRequestID   map[string]*CertificateResponse
}

// ExpiryIndex provides efficient expiry-based lookups.
type ExpiryIndex struct {
	entries []ExpiryEntry
	sorted  bool
	mu      sync.RWMutex
}

// ExpiryEntry represents a certificate expiry entry.
type ExpiryEntry struct {
	SerialNumber string
	ExpiresAt    time.Time
}

// CertificateEncryptor handles certificate encryption for storage.
type CertificateEncryptor struct {
	enabled   bool
	key       []byte
	algorithm string
}

// StoredCertificate represents a certificate stored in Kubernetes.
type StoredCertificate struct {
	CertificateResponse *CertificateResponse `json:"certificate_response"`
	StoredAt            time.Time            `json:"stored_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	Version             int                  `json:"version"`
	Checksum            string               `json:"checksum"`
}

// NewCertificatePool creates a new certificate pool.
func NewCertificatePool(config *CertificateStoreConfig, logger *logging.StructuredLogger) (*CertificatePool, error) {
	pool := &CertificatePool{
		config:       config,
		logger:       logger,
		certificates: make(map[string]*CertificateResponse),
		indices: &CertificateIndices{
			ByTenant:      make(map[string][]*CertificateResponse),
			ByStatus:      make(map[CertificateStatus][]*CertificateResponse),
			ByExpiryDate:  &ExpiryIndex{entries: make([]ExpiryEntry, 0)},
			ByFingerprint: make(map[string]*CertificateResponse),
			ByRequestID:   make(map[string]*CertificateResponse),
		},
	}

	// Initialize encryptor if encryption is enabled.
	if config.EncryptionKey != "" {
		encryptor, err := NewCertificateEncryptor(config.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize certificate encryptor: %w", err)
		}
		pool.encryptor = encryptor
	}

	// Start background processes.
	go pool.runMaintenanceTasks()

	return pool, nil
}

// StoreCertificate stores a certificate in the pool.
func (p *CertificatePool) StoreCertificate(cert *CertificateResponse) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Debug("storing certificate",
		"serial_number", cert.SerialNumber,
		"request_id", cert.RequestID)

	// Store in memory.
	p.certificates[cert.SerialNumber] = cert

	// Update indices.
	p.updateIndices(cert, false)

	// Persist to Kubernetes if configured.
	if p.config.Type == "kubernetes" {
		if err := p.persistToKubernetes(cert); err != nil {
			p.logger.Error("failed to persist certificate to Kubernetes",
				"serial_number", cert.SerialNumber,
				"error", err)
			return fmt.Errorf("failed to persist certificate: %w", err)
		}
	}

	p.logger.Info("certificate stored successfully",
		"serial_number", cert.SerialNumber,
		"expires_at", cert.ExpiresAt)

	return nil
}

// GetCertificate retrieves a certificate by serial number.
func (p *CertificatePool) GetCertificate(serialNumber string) (*CertificateResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cert, exists := p.certificates[serialNumber]
	if !exists {
		// Try loading from persistent storage.
		if loadedCert, err := p.loadFromStorage(serialNumber); err == nil {
			return loadedCert, nil
		}
		return nil, fmt.Errorf("certificate with serial number %s not found", serialNumber)
	}

	return cert, nil
}

// ListCertificates lists certificates with optional filters.
func (p *CertificatePool) ListCertificates(filters map[string]string) ([]*CertificateResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []*CertificateResponse

	// If no filters, return all certificates.
	if len(filters) == 0 {
		for _, cert := range p.certificates {
			results = append(results, cert)
		}
		return results, nil
	}

	// Apply filters using indices for efficiency.
	if tenantID, ok := filters["tenant_id"]; ok {
		if tenantCerts, exists := p.indices.ByTenant[tenantID]; exists {
			results = tenantCerts
		}
	} else if status, ok := filters["status"]; ok {
		if statusCerts, exists := p.indices.ByStatus[CertificateStatus(status)]; exists {
			results = statusCerts
		}
	} else {
		// Fallback to linear search for other filters.
		for _, cert := range p.certificates {
			if p.matchesFilters(cert, filters) {
				results = append(results, cert)
			}
		}
	}

	return results, nil
}

// GetExpiringCertificates returns certificates expiring before the given threshold.
func (p *CertificatePool) GetExpiringCertificates(threshold time.Time) ([]*CertificateResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var expiring []*CertificateResponse

	// Use expiry index for efficient lookup.
	p.indices.ByExpiryDate.mu.RLock()
	defer p.indices.ByExpiryDate.mu.RUnlock()

	for _, entry := range p.indices.ByExpiryDate.entries {
		if entry.ExpiresAt.Before(threshold) {
			if cert, exists := p.certificates[entry.SerialNumber]; exists {
				expiring = append(expiring, cert)
			}
		} else {
			// Since entries are sorted, we can break early.
			break
		}
	}

	return expiring, nil
}

// UpdateCertificate updates an existing certificate.
func (p *CertificatePool) UpdateCertificate(cert *CertificateResponse) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	existing, exists := p.certificates[cert.SerialNumber]
	if !exists {
		return fmt.Errorf("certificate with serial number %s not found", cert.SerialNumber)
	}

	// Update indices (remove old entry).
	p.updateIndices(existing, true)

	// Update certificate.
	p.certificates[cert.SerialNumber] = cert

	// Update indices (add new entry).
	p.updateIndices(cert, false)

	// Persist to storage.
	if p.config.Type == "kubernetes" {
		if err := p.persistToKubernetes(cert); err != nil {
			return fmt.Errorf("failed to persist updated certificate: %w", err)
		}
	}

	return nil
}

// DeleteCertificate removes a certificate from the pool.
func (p *CertificatePool) DeleteCertificate(serialNumber string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cert, exists := p.certificates[serialNumber]
	if !exists {
		return fmt.Errorf("certificate with serial number %s not found", serialNumber)
	}

	// Remove from indices.
	p.updateIndices(cert, true)

	// Remove from memory.
	delete(p.certificates, serialNumber)

	// Remove from persistent storage.
	if p.config.Type == "kubernetes" {
		if err := p.removeFromKubernetes(serialNumber); err != nil {
			p.logger.Warn("failed to remove certificate from Kubernetes",
				"serial_number", serialNumber,
				"error", err)
		}
	}

	return nil
}

// GetCertificateByFingerprint retrieves a certificate by fingerprint.
func (p *CertificatePool) GetCertificateByFingerprint(fingerprint string) (*CertificateResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if cert, exists := p.indices.ByFingerprint[fingerprint]; exists {
		return cert, nil
	}

	return nil, fmt.Errorf("certificate with fingerprint %s not found", fingerprint)
}

// GetCertificateByRequestID retrieves a certificate by request ID.
func (p *CertificatePool) GetCertificateByRequestID(requestID string) (*CertificateResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if cert, exists := p.indices.ByRequestID[requestID]; exists {
		return cert, nil
	}

	return nil, fmt.Errorf("certificate with request ID %s not found", requestID)
}

// GetStatistics returns certificate pool statistics.
func (p *CertificatePool) GetStatistics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := map[string]interface{}{
		"total_certificates": len(p.certificates),
		"by_status":          make(map[string]int),
		"by_tenant":          make(map[string]int),
		"expiring_soon":      0,
	}

	// Count by status.
	for status, certs := range p.indices.ByStatus {
		stats["by_status"].(map[string]int)[string(status)] = len(certs)
	}

	// Count by tenant.
	for tenant, certs := range p.indices.ByTenant {
		stats["by_tenant"].(map[string]int)[tenant] = len(certs)
	}

	// Count expiring certificates (next 30 days).
	threshold := time.Now().Add(30 * 24 * time.Hour)
	for _, cert := range p.certificates {
		if cert.ExpiresAt.Before(threshold) {
			stats["expiring_soon"] = stats["expiring_soon"].(int) + 1
		}
	}

	return stats
}

// Close gracefully shuts down the certificate pool.
func (p *CertificatePool) Close() error {
	p.logger.Info("shutting down certificate pool")

	// Perform final backup if enabled.
	if p.config.BackupEnabled {
		if err := p.performBackup(); err != nil {
			p.logger.Error("final backup failed", "error", err)
		}
	}

	return nil
}

// Helper methods.

func (p *CertificatePool) updateIndices(cert *CertificateResponse, remove bool) {
	// Update tenant index.
	tenantID := cert.Metadata["tenant_id"]
	if tenantID != "" {
		if remove {
			// Get existing slice, modify it, and put it back.
			tenantCerts := p.indices.ByTenant[tenantID]
			p.removeFromSlice(&tenantCerts, cert)
			if len(tenantCerts) == 0 {
				delete(p.indices.ByTenant, tenantID)
			} else {
				p.indices.ByTenant[tenantID] = tenantCerts
			}
		} else {
			p.indices.ByTenant[tenantID] = append(p.indices.ByTenant[tenantID], cert)
		}
	}

	// Update status index.
	if remove {
		// Get existing slice, modify it, and put it back.
		statusCerts := p.indices.ByStatus[cert.Status]
		p.removeFromSlice(&statusCerts, cert)
		if len(statusCerts) == 0 {
			delete(p.indices.ByStatus, cert.Status)
		} else {
			p.indices.ByStatus[cert.Status] = statusCerts
		}
	} else {
		p.indices.ByStatus[cert.Status] = append(p.indices.ByStatus[cert.Status], cert)
	}

	// Update expiry index.
	p.indices.ByExpiryDate.mu.Lock()
	if remove {
		p.removeFromExpiryIndex(cert.SerialNumber)
	} else {
		p.indices.ByExpiryDate.entries = append(p.indices.ByExpiryDate.entries, ExpiryEntry{
			SerialNumber: cert.SerialNumber,
			ExpiresAt:    cert.ExpiresAt,
		})
		p.indices.ByExpiryDate.sorted = false
	}
	p.indices.ByExpiryDate.mu.Unlock()

	// Update fingerprint index.
	if remove {
		delete(p.indices.ByFingerprint, cert.Fingerprint)
	} else {
		p.indices.ByFingerprint[cert.Fingerprint] = cert
	}

	// Update request ID index.
	if remove {
		delete(p.indices.ByRequestID, cert.RequestID)
	} else {
		p.indices.ByRequestID[cert.RequestID] = cert
	}
}

func (p *CertificatePool) removeFromSlice(slice *[]*CertificateResponse, cert *CertificateResponse) {
	for i, c := range *slice {
		if c.SerialNumber == cert.SerialNumber {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
}

func (p *CertificatePool) removeFromExpiryIndex(serialNumber string) {
	for i, entry := range p.indices.ByExpiryDate.entries {
		if entry.SerialNumber == serialNumber {
			p.indices.ByExpiryDate.entries = append(
				p.indices.ByExpiryDate.entries[:i],
				p.indices.ByExpiryDate.entries[i+1:]...)
			break
		}
	}
}

func (p *CertificatePool) matchesFilters(cert *CertificateResponse, filters map[string]string) bool {
	for key, value := range filters {
		switch key {
		case "tenant_id":
			if cert.Metadata["tenant_id"] != value {
				return false
			}
		case "status":
			if string(cert.Status) != value {
				return false
			}
		case "issuer":
			if cert.IssuedBy != value {
				return false
			}
		case "common_name":
			if cert.Certificate.Subject.CommonName != value {
				return false
			}
		}
	}
	return true
}

func (p *CertificatePool) persistToKubernetes(cert *CertificateResponse) error {
	secretName := p.generateSecretName(cert.SerialNumber)

	storedCert := &StoredCertificate{
		CertificateResponse: cert,
		StoredAt:            time.Now(),
		UpdatedAt:           time.Now(),
		Version:             1,
	}

	// Serialize certificate data.
	data, err := json.Marshal(storedCert)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate data: %w", err)
	}

	// Encrypt if enabled.
	if p.encryptor != nil && p.encryptor.enabled {
		data, err = p.encryptor.Encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt certificate data: %w", err)
		}
	}

	// Create or update Kubernetes secret.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: p.config.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "nephoran-intent-operator",
				"app.kubernetes.io/component":  "certificate-pool",
				"app.kubernetes.io/managed-by": "nephoran-ca-manager",
				"nephoran.io/certificate":      "true",
				"nephoran.io/serial-number":    cert.SerialNumber,
				"nephoran.io/tenant-id":        cert.Metadata["tenant_id"],
				"nephoran.io/status":           string(cert.Status),
			},
			Annotations: map[string]string{
				"nephoran.io/expires-at":  cert.ExpiresAt.Format(time.RFC3339),
				"nephoran.io/issued-by":   cert.IssuedBy,
				"nephoran.io/fingerprint": cert.Fingerprint,
				"nephoran.io/request-id":  cert.RequestID,
			},
		},
		Data: map[string][]byte{
			"certificate.json": data,
		},
	}

	// Add the certificate and key data for easy access.
	secret.Data["tls.crt"] = []byte(cert.CertificatePEM)
	secret.Data["tls.key"] = []byte(cert.PrivateKeyPEM)
	if cert.CACertificatePEM != "" {
		secret.Data["ca.crt"] = []byte(cert.CACertificatePEM)
	}

	// Create or update the secret.
	err = p.client.Create(context.TODO(), secret)
	if err != nil {
		// Try update if create failed.
		existingSecret := &corev1.Secret{}
		if getErr := p.client.Get(context.TODO(), types.NamespacedName{
			Name:      secretName,
			Namespace: p.config.Namespace,
		}, existingSecret); getErr == nil {
			secret.ResourceVersion = existingSecret.ResourceVersion
			err = p.client.Update(context.TODO(), secret)
		}
	}

	return err
}

func (p *CertificatePool) loadFromStorage(serialNumber string) (*CertificateResponse, error) {
	if p.config.Type != "kubernetes" {
		return nil, fmt.Errorf("storage type %s not supported for loading", p.config.Type)
	}

	secretName := p.generateSecretName(serialNumber)
	secret := &corev1.Secret{}

	err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      secretName,
		Namespace: p.config.Namespace,
	}, secret)
	if err != nil {
		return nil, err
	}

	data, exists := secret.Data["certificate.json"]
	if !exists {
		return nil, fmt.Errorf("certificate data not found in secret")
	}

	// Decrypt if needed.
	if p.encryptor != nil && p.encryptor.enabled {
		data, err = p.encryptor.Decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt certificate data: %w", err)
		}
	}

	var storedCert StoredCertificate
	if err := json.Unmarshal(data, &storedCert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate data: %w", err)
	}

	// Add to memory cache.
	p.mu.Lock()
	p.certificates[serialNumber] = storedCert.CertificateResponse
	p.updateIndices(storedCert.CertificateResponse, false)
	p.mu.Unlock()

	return storedCert.CertificateResponse, nil
}

func (p *CertificatePool) removeFromKubernetes(serialNumber string) error {
	secretName := p.generateSecretName(serialNumber)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: p.config.Namespace,
		},
	}

	return p.client.Delete(context.TODO(), secret)
}

func (p *CertificatePool) generateSecretName(serialNumber string) string {
	prefix := p.config.SecretPrefix
	if prefix == "" {
		prefix = "nephoran-cert"
	}
	return fmt.Sprintf("%s-%s", prefix, serialNumber)
}

func (p *CertificatePool) runMaintenanceTasks() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		p.performMaintenance()
	}
}

func (p *CertificatePool) performMaintenance() {
	p.logger.Debug("performing certificate pool maintenance")

	// Sort expiry index if needed.
	p.indices.ByExpiryDate.mu.Lock()
	if !p.indices.ByExpiryDate.sorted {
		sort.Slice(p.indices.ByExpiryDate.entries, func(i, j int) bool {
			return p.indices.ByExpiryDate.entries[i].ExpiresAt.Before(p.indices.ByExpiryDate.entries[j].ExpiresAt)
		})
		p.indices.ByExpiryDate.sorted = true
	}
	p.indices.ByExpiryDate.mu.Unlock()

	// Clean up expired certificates if retention period is configured.
	if p.config.RetentionPeriod > 0 {
		p.cleanupExpiredCertificates()
	}

	// Perform backup if enabled.
	if p.config.BackupEnabled && time.Since(p.lastBackup) > p.config.BackupInterval {
		if err := p.performBackup(); err != nil {
			p.logger.Error("certificate pool backup failed", "error", err)
		}
		p.lastBackup = time.Now()
	}
}

func (p *CertificatePool) cleanupExpiredCertificates() {
	cutoff := time.Now().Add(-p.config.RetentionPeriod)

	p.mu.Lock()
	defer p.mu.Unlock()

	var toDelete []string
	for serialNumber, cert := range p.certificates {
		if cert.ExpiresAt.Before(cutoff) && cert.Status == StatusExpired {
			toDelete = append(toDelete, serialNumber)
		}
	}

	for _, serialNumber := range toDelete {
		p.logger.Info("cleaning up expired certificate",
			"serial_number", serialNumber)

		cert := p.certificates[serialNumber]
		p.updateIndices(cert, true)
		delete(p.certificates, serialNumber)

		// Remove from persistent storage.
		if err := p.removeFromKubernetes(serialNumber); err != nil {
			p.logger.Warn("failed to remove expired certificate from storage",
				"serial_number", serialNumber,
				"error", err)
		}
	}
}

func (p *CertificatePool) performBackup() error {
	p.logger.Info("performing certificate pool backup")
	// Implementation would create a backup of all certificates.
	// This is a placeholder for actual backup logic.
	return nil
}

// NewCertificateEncryptor creates a new certificate encryptor.
func NewCertificateEncryptor(key string) (*CertificateEncryptor, error) {
	return &CertificateEncryptor{
		enabled:   true,
		key:       []byte(key),
		algorithm: "AES-256-GCM",
	}, nil
}

// Encrypt encrypts certificate data.
func (e *CertificateEncryptor) Encrypt(data []byte) ([]byte, error) {
	// Implementation would use AES-GCM encryption.
	// This is a placeholder for actual encryption logic.
	return data, nil
}

// Decrypt decrypts certificate data.
func (e *CertificateEncryptor) Decrypt(data []byte) ([]byte, error) {
	// Implementation would use AES-GCM decryption.
	// This is a placeholder for actual decryption logic.
	return data, nil
}
