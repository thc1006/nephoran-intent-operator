
package mtls



import (

	"context"

	"fmt"

	"path/filepath"

	"sync"

	"time"



	"github.com/thc1006/nephoran-intent-operator/pkg/logging"

	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"

)



// ServiceIdentity represents a service identity with associated certificates.

type ServiceIdentity struct {

	ServiceName   string            `json:"service_name"`

	TenantID      string            `json:"tenant_id"`

	Role          ServiceRole       `json:"role"`

	CertificateID string            `json:"certificate_id"`

	SerialNumber  string            `json:"serial_number"`

	IssuedAt      time.Time         `json:"issued_at"`

	ExpiresAt     time.Time         `json:"expires_at"`

	Status        IdentityStatus    `json:"status"`

	Metadata      map[string]string `json:"metadata"`



	// Certificate paths.

	CertPath   string `json:"cert_path"`

	KeyPath    string `json:"key_path"`

	CACertPath string `json:"ca_cert_path"`



	// Last rotation tracking.

	LastRotation  time.Time `json:"last_rotation"`

	RotationCount int       `json:"rotation_count"`

}



// ServiceRole defines the role of a service.

type ServiceRole string



const (

	// RoleController holds rolecontroller value.

	RoleController ServiceRole = "controller"

	// RoleLLMService holds rolellmservice value.

	RoleLLMService ServiceRole = "llm-service"

	// RoleRAGService holds roleragservice value.

	RoleRAGService ServiceRole = "rag-service"

	// RoleGitClient holds rolegitclient value.

	RoleGitClient ServiceRole = "git-client"

	// RoleDatabaseClient holds roledatabaseclient value.

	RoleDatabaseClient ServiceRole = "database-client"

	// RoleNephioBridge holds rolenephiobridge value.

	RoleNephioBridge ServiceRole = "nephio-bridge"

	// RoleORANAdaptor holds roleoranadaptor value.

	RoleORANAdaptor ServiceRole = "oran-adaptor"

	// RoleMonitoring holds rolemonitoring value.

	RoleMonitoring ServiceRole = "monitoring"

)



// IdentityStatus represents the status of a service identity.

type IdentityStatus string



const (

	// StatusActive holds statusactive value.

	StatusActive IdentityStatus = "active"

	// StatusExpiring holds statusexpiring value.

	StatusExpiring IdentityStatus = "expiring"

	// StatusExpired holds statusexpired value.

	StatusExpired IdentityStatus = "expired"

	// StatusRevoked holds statusrevoked value.

	StatusRevoked IdentityStatus = "revoked"

	// StatusPending holds statuspending value.

	StatusPending IdentityStatus = "pending"

	// StatusFailed holds statusfailed value.

	StatusFailed IdentityStatus = "failed"

)



// IdentityManagerConfig holds configuration for the identity manager.

type IdentityManagerConfig struct {

	CAManager               *ca.CAManager

	BaseDir                 string

	DefaultTenantID         string

	DefaultPolicyTemplate   string

	DefaultValidityDuration time.Duration

	RenewalThreshold        time.Duration

	RotationInterval        time.Duration

	CleanupInterval         time.Duration

	MaxIdentities           int

	BackupEnabled           bool

}



// IdentityManager manages service identities and their certificates.

type IdentityManager struct {

	config    *IdentityManagerConfig

	logger    *logging.StructuredLogger

	caManager *ca.CAManager



	// Identity tracking.

	identities map[string]*ServiceIdentity

	mu         sync.RWMutex



	// Background processes.

	ctx            context.Context

	cancel         context.CancelFunc

	rotationTicker *time.Ticker

	cleanupTicker  *time.Ticker

}



// NewIdentityManager creates a new service identity manager.

func NewIdentityManager(config *IdentityManagerConfig, logger *logging.StructuredLogger) (*IdentityManager, error) {

	if config == nil {

		return nil, fmt.Errorf("identity manager config is required")

	}



	if config.CAManager == nil {

		return nil, fmt.Errorf("CA manager is required")

	}



	if logger == nil {

		logger = logging.NewStructuredLogger(logging.DefaultConfig("mtls-identity", "1.0.0", "production"))

	}



	// Set defaults.

	if config.BaseDir == "" {

		config.BaseDir = "/etc/nephoran/certs"

	}

	if config.DefaultTenantID == "" {

		config.DefaultTenantID = "default"

	}

	if config.DefaultPolicyTemplate == "" {

		config.DefaultPolicyTemplate = "service-auth"

	}

	if config.DefaultValidityDuration == 0 {

		config.DefaultValidityDuration = 24 * time.Hour

	}

	if config.RenewalThreshold == 0 {

		config.RenewalThreshold = 6 * time.Hour

	}

	if config.RotationInterval == 0 {

		config.RotationInterval = 30 * time.Minute

	}

	if config.CleanupInterval == 0 {

		config.CleanupInterval = 1 * time.Hour

	}

	if config.MaxIdentities == 0 {

		config.MaxIdentities = 1000

	}



	ctx, cancel := context.WithCancel(context.Background())



	manager := &IdentityManager{

		config:     config,

		logger:     logger,

		caManager:  config.CAManager,

		identities: make(map[string]*ServiceIdentity),

		ctx:        ctx,

		cancel:     cancel,

	}



	// Start background processes.

	manager.startBackgroundProcesses()



	logger.Info("identity manager initialized",

		"base_dir", config.BaseDir,

		"default_tenant_id", config.DefaultTenantID,

		"rotation_interval", config.RotationInterval)



	return manager, nil

}



// CreateServiceIdentity creates a new service identity.

func (im *IdentityManager) CreateServiceIdentity(serviceName string, role ServiceRole, tenantID string) (*ServiceIdentity, error) {

	if serviceName == "" {

		return nil, fmt.Errorf("service name is required")

	}



	if tenantID == "" {

		tenantID = im.config.DefaultTenantID

	}



	identityKey := fmt.Sprintf("%s-%s-%s", serviceName, string(role), tenantID)



	im.mu.Lock()

	defer im.mu.Unlock()



	// Check if identity already exists.

	if existing, exists := im.identities[identityKey]; exists {

		if existing.Status == StatusActive {

			return existing, nil

		}

	}



	// Check identity limit.

	if len(im.identities) >= im.config.MaxIdentities {

		return nil, fmt.Errorf("maximum number of identities reached: %d", im.config.MaxIdentities)

	}



	im.logger.Info("creating service identity",

		"service_name", serviceName,

		"role", role,

		"tenant_id", tenantID)



	// Generate certificate paths.

	certDir := filepath.Join(im.config.BaseDir, tenantID, serviceName)

	certPath := filepath.Join(certDir, "tls.crt")

	keyPath := filepath.Join(certDir, "tls.key")

	caCertPath := filepath.Join(certDir, "ca.crt")



	// Create certificate request based on role.

	req := im.createCertificateRequest(serviceName, role, tenantID)



	// Issue certificate.

	resp, err := im.caManager.IssueCertificate(context.Background(), req)

	if err != nil {

		return nil, fmt.Errorf("failed to issue certificate for service identity: %w", err)

	}



	// Create service identity.

	identity := &ServiceIdentity{

		ServiceName:   serviceName,

		TenantID:      tenantID,

		Role:          role,

		CertificateID: req.ID,

		SerialNumber:  resp.SerialNumber,

		IssuedAt:      resp.CreatedAt,

		ExpiresAt:     resp.ExpiresAt,

		Status:        StatusActive,

		CertPath:      certPath,

		KeyPath:       keyPath,

		CACertPath:    caCertPath,

		LastRotation:  time.Now(),

		RotationCount: 0,

		Metadata: map[string]string{

			"role":        string(role),

			"issuer":      resp.IssuedBy,

			"fingerprint": resp.Fingerprint,

		},

	}



	// Store certificate files.

	if err := im.storeCertificateFiles(identity, resp); err != nil {

		return nil, fmt.Errorf("failed to store certificate files: %w", err)

	}



	// Add to tracking.

	im.identities[identityKey] = identity



	im.logger.Info("service identity created",

		"service_name", serviceName,

		"role", role,

		"serial_number", resp.SerialNumber,

		"expires_at", resp.ExpiresAt)



	return identity, nil

}



// GetServiceIdentity retrieves a service identity.

func (im *IdentityManager) GetServiceIdentity(serviceName string, role ServiceRole, tenantID string) (*ServiceIdentity, error) {

	if tenantID == "" {

		tenantID = im.config.DefaultTenantID

	}



	identityKey := fmt.Sprintf("%s-%s-%s", serviceName, string(role), tenantID)



	im.mu.RLock()

	identity, exists := im.identities[identityKey]

	im.mu.RUnlock()



	if !exists {

		return nil, fmt.Errorf("service identity not found: %s", identityKey)

	}



	return identity, nil

}



// ListServiceIdentities lists all service identities.

func (im *IdentityManager) ListServiceIdentities() []*ServiceIdentity {

	im.mu.RLock()

	defer im.mu.RUnlock()



	identities := make([]*ServiceIdentity, 0, len(im.identities))

	for _, identity := range im.identities {

		identities = append(identities, identity)

	}



	return identities

}



// RevokeServiceIdentity revokes a service identity.

func (im *IdentityManager) RevokeServiceIdentity(serviceName string, role ServiceRole, tenantID string, reason int) error {

	if tenantID == "" {

		tenantID = im.config.DefaultTenantID

	}



	identityKey := fmt.Sprintf("%s-%s-%s", serviceName, string(role), tenantID)



	im.mu.Lock()

	defer im.mu.Unlock()



	identity, exists := im.identities[identityKey]

	if !exists {

		return fmt.Errorf("service identity not found: %s", identityKey)

	}



	// Revoke certificate.

	if err := im.caManager.RevokeCertificate(context.Background(), identity.SerialNumber, reason, tenantID); err != nil {

		return fmt.Errorf("failed to revoke certificate: %w", err)

	}



	// Update status.

	identity.Status = StatusRevoked



	im.logger.Info("service identity revoked",

		"service_name", serviceName,

		"role", role,

		"serial_number", identity.SerialNumber,

		"reason", reason)



	return nil

}



// RotateServiceIdentity rotates the certificate for a service identity.

func (im *IdentityManager) RotateServiceIdentity(serviceName string, role ServiceRole, tenantID string) error {

	if tenantID == "" {

		tenantID = im.config.DefaultTenantID

	}



	identityKey := fmt.Sprintf("%s-%s-%s", serviceName, string(role), tenantID)



	im.mu.Lock()

	defer im.mu.Unlock()



	identity, exists := im.identities[identityKey]

	if !exists {

		return fmt.Errorf("service identity not found: %s", identityKey)

	}



	im.logger.Info("rotating service identity certificate",

		"service_name", serviceName,

		"role", role,

		"current_serial_number", identity.SerialNumber)



	// Renew certificate.

	resp, err := im.caManager.RenewCertificate(context.Background(), identity.SerialNumber)

	if err != nil {

		return fmt.Errorf("failed to renew certificate: %w", err)

	}



	// Store new certificate files.

	if err := im.storeCertificateFiles(identity, resp); err != nil {

		return fmt.Errorf("failed to store rotated certificate files: %w", err)

	}



	// Update identity.

	identity.SerialNumber = resp.SerialNumber

	identity.ExpiresAt = resp.ExpiresAt

	identity.LastRotation = time.Now()

	identity.RotationCount++

	identity.Metadata["fingerprint"] = resp.Fingerprint



	im.logger.Info("service identity certificate rotated",

		"service_name", serviceName,

		"role", role,

		"new_serial_number", resp.SerialNumber,

		"expires_at", resp.ExpiresAt,

		"rotation_count", identity.RotationCount)



	return nil

}



// createCertificateRequest creates a certificate request based on service role.

func (im *IdentityManager) createCertificateRequest(serviceName string, role ServiceRole, tenantID string) *ca.CertificateRequest {

	req := &ca.CertificateRequest{

		ID:               generateRequestID(),

		TenantID:         tenantID,

		CommonName:       serviceName,

		ValidityDuration: im.config.DefaultValidityDuration,

		PolicyTemplate:   im.config.DefaultPolicyTemplate,

		AutoRenew:        true,

		Metadata: map[string]string{

			"service_name": serviceName,

			"role":         string(role),

			"component":    "nephoran-intent-operator",

		},

	}



	// Role-specific configuration.

	switch role {

	case RoleController:

		req.DNSNames = []string{

			serviceName,

			fmt.Sprintf("%s.%s", serviceName, tenantID),

			fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, tenantID),

		}

		req.KeyUsage = []string{"digital_signature", "key_encipherment", "client_auth", "server_auth"}

		req.ExtKeyUsage = []string{"client_auth", "server_auth"}



	case RoleLLMService, RoleRAGService, RoleNephioBridge, RoleORANAdaptor:

		req.DNSNames = []string{

			serviceName,

			fmt.Sprintf("%s.%s", serviceName, tenantID),

			fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, tenantID),

		}

		req.KeyUsage = []string{"digital_signature", "key_encipherment", "server_auth"}

		req.ExtKeyUsage = []string{"server_auth"}



	case RoleGitClient, RoleDatabaseClient, RoleMonitoring:

		req.DNSNames = []string{serviceName}

		req.KeyUsage = []string{"digital_signature", "key_encipherment", "client_auth"}

		req.ExtKeyUsage = []string{"client_auth"}



	default:

		// Default to client authentication.

		req.DNSNames = []string{serviceName}

		req.KeyUsage = []string{"digital_signature", "key_encipherment", "client_auth"}

		req.ExtKeyUsage = []string{"client_auth"}

	}



	return req

}



// startBackgroundProcesses starts background maintenance processes.

func (im *IdentityManager) startBackgroundProcesses() {

	// Certificate rotation checker.

	im.rotationTicker = time.NewTicker(im.config.RotationInterval)

	go im.rotationLoop()



	// Cleanup process.

	im.cleanupTicker = time.NewTicker(im.config.CleanupInterval)

	go im.cleanupLoop()

}



// rotationLoop handles automatic certificate rotation.

func (im *IdentityManager) rotationLoop() {

	defer im.rotationTicker.Stop()



	for {

		select {

		case <-im.ctx.Done():

			return

		case <-im.rotationTicker.C:

			im.checkAndRotateExpiring()

		}

	}

}



// checkAndRotateExpiring checks for expiring certificates and rotates them.

func (im *IdentityManager) checkAndRotateExpiring() {

	im.mu.RLock()

	expiring := make([]*ServiceIdentity, 0)



	for _, identity := range im.identities {

		if identity.Status == StatusActive {

			timeUntilExpiry := time.Until(identity.ExpiresAt)

			if timeUntilExpiry <= im.config.RenewalThreshold {

				expiring = append(expiring, identity)

				identity.Status = StatusExpiring

			}

		}

	}

	im.mu.RUnlock()



	// Rotate expiring certificates.

	for _, identity := range expiring {

		if err := im.RotateServiceIdentity(identity.ServiceName, identity.Role, identity.TenantID); err != nil {

			im.logger.Error("failed to rotate expiring certificate",

				"service_name", identity.ServiceName,

				"role", identity.Role,

				"error", err)

			identity.Status = StatusFailed

		} else {

			identity.Status = StatusActive

		}

	}

}



// cleanupLoop handles periodic cleanup of expired identities.

func (im *IdentityManager) cleanupLoop() {

	defer im.cleanupTicker.Stop()



	for {

		select {

		case <-im.ctx.Done():

			return

		case <-im.cleanupTicker.C:

			im.cleanupExpiredIdentities()

		}

	}

}



// cleanupExpiredIdentities removes expired identities.

func (im *IdentityManager) cleanupExpiredIdentities() {

	im.mu.Lock()

	defer im.mu.Unlock()



	now := time.Now()

	cleanupThreshold := 24 * time.Hour // Keep expired identities for 24 hours



	for key, identity := range im.identities {

		if identity.Status == StatusExpired || identity.Status == StatusRevoked {

			if now.Sub(identity.ExpiresAt) > cleanupThreshold {

				// Remove from tracking.

				delete(im.identities, key)



				im.logger.Info("cleaned up expired identity",

					"service_name", identity.ServiceName,

					"role", identity.Role,

					"expired_at", identity.ExpiresAt)

			}

		}

	}

}



// Close gracefully shuts down the identity manager.

func (im *IdentityManager) Close() error {

	im.logger.Info("shutting down identity manager")



	im.cancel()



	if im.rotationTicker != nil {

		im.rotationTicker.Stop()

	}



	if im.cleanupTicker != nil {

		im.cleanupTicker.Stop()

	}



	return nil

}



// GetStats returns statistics about managed identities.

func (im *IdentityManager) GetStats() *IdentityStats {

	im.mu.RLock()

	defer im.mu.RUnlock()



	stats := &IdentityStats{

		TotalIdentities: len(im.identities),

		StatusCounts:    make(map[IdentityStatus]int),

		RoleCounts:      make(map[ServiceRole]int),

	}



	for _, identity := range im.identities {

		stats.StatusCounts[identity.Status]++

		stats.RoleCounts[identity.Role]++

	}



	return stats

}



// IdentityStats holds statistics about managed identities.

type IdentityStats struct {

	TotalIdentities int                    `json:"total_identities"`

	StatusCounts    map[IdentityStatus]int `json:"status_counts"`

	RoleCounts      map[ServiceRole]int    `json:"role_counts"`

}



// storeCertificateFiles stores certificate files for a service identity.

func (im *IdentityManager) storeCertificateFiles(identity *ServiceIdentity, resp *ca.CertificateResponse) error {

	// Ensure directory exists.

	certDir := filepath.Dir(identity.CertPath)

	if err := ensureDirectory(certDir, 0o750); err != nil {

		return fmt.Errorf("failed to create certificate directory: %w", err)

	}



	// Store certificate.

	if err := writeSecureFile(identity.CertPath, []byte(resp.CertificatePEM), 0o640); err != nil {

		return fmt.Errorf("failed to store certificate: %w", err)

	}



	// Store private key.

	if err := writeSecureFile(identity.KeyPath, []byte(resp.PrivateKeyPEM), 0o600); err != nil {

		return fmt.Errorf("failed to store private key: %w", err)

	}



	// Store CA certificate.

	if resp.CACertificatePEM != "" {

		if err := writeSecureFile(identity.CACertPath, []byte(resp.CACertificatePEM), 0o644); err != nil {

			return fmt.Errorf("failed to store CA certificate: %w", err)

		}

	}



	return nil

}

