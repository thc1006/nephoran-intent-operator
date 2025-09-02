package mtls

import (
	
	"encoding/json"
"context"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// MTLSMonitor provides comprehensive monitoring for mTLS connections and certificates.

type MTLSMonitor struct {
	logger *logging.StructuredLogger

	collectors []MetricCollector

	alerts []AlertRule

	// Connection tracking.

	connections map[string]*ConnectionInfo

	connMu sync.RWMutex

	// Certificate tracking.

	certificates map[string]*CertificateMonitorInfo

	certMu sync.RWMutex

	// Monitoring configuration.

	config MonitorConfig

	// Context for shutdown.

	ctx context.Context

	cancel context.CancelFunc
}

// ServiceRole represents the role of a certificate/service - unified definition for 2025.

type ServiceRole string

const (
	// Traditional TLS roles
	ServiceRoleClient ServiceRole = "client"
	ServiceRoleServer ServiceRole = "server"
	ServiceRoleCA     ServiceRole = "ca"

	// Application-specific roles for 2025 security architecture
	RoleController     ServiceRole = "controller"
	RoleLLMService     ServiceRole = "llm-service"
	RoleRAGService     ServiceRole = "rag-service"
	RoleGitClient      ServiceRole = "git-client"
	RoleDatabaseClient ServiceRole = "database-client"
	RoleNephioBridge   ServiceRole = "nephio-bridge"
	RoleORANAdaptor    ServiceRole = "oran-adaptor"
	RoleMonitoring     ServiceRole = "monitoring"
)

// ConnectionInfo holds information about an mTLS connection.

type ConnectionInfo struct {
	ID string `json:"id"`

	ServiceName string `json:"service_name"`

	RemoteAddr string `json:"remote_addr"`

	LocalAddr string `json:"local_addr"`

	Protocol string `json:"protocol"`

	CipherSuite uint16 `json:"cipher_suite"`

	TLSVersion uint16 `json:"tls_version"`

	ConnectedAt time.Time `json:"connected_at"`

	LastActivity time.Time `json:"last_activity"`

	RequestCount int64 `json:"request_count"`

	ErrorCount int64 `json:"error_count"`

	CertificateInfo *ConnectionCertInfo `json:"certificate_info,omitempty"`

	Metadata json.RawMessage `json:"metadata"`
}

// ConnectionCertInfo holds certificate information for a connection.

type ConnectionCertInfo struct {
	Subject string `json:"subject"`

	Issuer string `json:"issuer"`

	SerialNumber string `json:"serial_number"`

	NotBefore time.Time `json:"not_before"`

	NotAfter time.Time `json:"not_after"`

	IsExpired bool `json:"is_expired"`

	ExpiresIn int64 `json:"expires_in_seconds"`
}

// CertificateMonitorInfo holds certificate information for monitoring.

type CertificateMonitorInfo struct {
	ID string `json:"id"`

	ServiceName string `json:"service_name"`

	Role ServiceRole `json:"role"`

	CertificatePath string `json:"certificate_path"`

	Certificate *x509.Certificate `json:"-"` // Don't serialize the full cert

	SerialNumber string `json:"serial_number"`

	Subject string `json:"subject"`

	Issuer string `json:"issuer"`

	NotBefore time.Time `json:"not_before"`

	NotAfter time.Time `json:"not_after"`

	IsExpired bool `json:"is_expired"`

	ExpiresInDays int `json:"expires_in_days"`

	LastChecked time.Time `json:"last_checked"`

	RotationCount int `json:"rotation_count"`

	HealthStatus CertificateHealth `json:"health_status"`

	Metadata json.RawMessage `json:"metadata"`
}

// CertificateHealth represents the health status of a certificate.

type CertificateHealth string

const (

	// CertHealthHealthy holds certhealthhealthy value.

	CertHealthHealthy CertificateHealth = "healthy"

	// CertHealthWarning holds certhealthwarning value.

	CertHealthWarning CertificateHealth = "warning"

	// CertHealthCritical holds certhealthcritical value.

	CertHealthCritical CertificateHealth = "critical"

	// CertHealthExpired holds certhealthexpired value.

	CertHealthExpired CertificateHealth = "expired"
)

// AlertRule defines an alerting rule for mTLS monitoring.

type AlertRule struct {
	Name string `json:"name"`

	Severity AlertSeverity `json:"severity"`

	Condition AlertCondition `json:"condition"`

	Enabled bool `json:"enabled"`

	Description string `json:"description"`

	NotificationChannels []string `json:"notification_channels"`

	LastTriggered *time.Time `json:"last_triggered,omitempty"`

	Metadata json.RawMessage `json:"metadata"`
}

// AlertCondition defines conditions that trigger alerts.

type AlertCondition struct {
	Type AlertType `json:"type"`

	Threshold float64 `json:"threshold,omitempty"`

	Duration string `json:"duration,omitempty"`

	Field string `json:"field,omitempty"`
}

// AlertType represents different types of alert conditions.

type AlertType string

const (

	// AlertTypeCertificateExpiring holds alerttypecertificateexpiring value.

	AlertTypeCertificateExpiring AlertType = "certificate_expiring"

	// AlertTypeCertificateExpired holds alerttypecertificateexpired value.

	AlertTypeCertificateExpired AlertType = "certificate_expired"

	// AlertTypeConnectionFailure holds alerttypeconnectionfailure value.

	AlertTypeConnectionFailure AlertType = "connection_failure"

	// AlertTypeHighErrorRate holds alerttypehigherrorrate value.

	AlertTypeHighErrorRate AlertType = "high_error_rate"

	// AlertTypeWeakCipher holds alerttypeweakcipher value.

	AlertTypeWeakCipher AlertType = "weak_cipher"

	// AlertTypeOldTLSVersion holds alerttypeoldtlsversion value.

	AlertTypeOldTLSVersion AlertType = "old_tls_version"
)

// AlertSeverity represents the severity level of an alert.

type AlertSeverity string

const (

	// AlertSeverityInfo holds alertseverityinfo value.

	AlertSeverityInfo AlertSeverity = "info"

	// AlertSeverityWarning holds alertseveritywarning value.

	AlertSeverityWarning AlertSeverity = "warning"

	// AlertSeverityError holds alertseverityerror value.

	AlertSeverityError AlertSeverity = "error"

	// AlertSeverityCritical holds alertseveritycritical value.

	AlertSeverityCritical AlertSeverity = "critical"
)

// Metric represents a collected metric.

type Metric struct {
	Name string `json:"name"`

	Value float64 `json:"value"`

	Labels map[string]string `json:"labels"`

	Timestamp time.Time `json:"timestamp"`

	Type string `json:"type"` // counter, gauge, histogram

	Help string `json:"help"`

	Metadata json.RawMessage `json:"metadata"`
}

// Alert represents a triggered alert.

type Alert struct {
	Name string `json:"name"`

	Severity AlertSeverity `json:"severity"`

	Message string `json:"message"`

	Labels map[string]string `json:"labels"`

	Timestamp time.Time `json:"timestamp"`

	Resolved bool `json:"resolved"`

	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	Metadata json.RawMessage `json:"metadata"`
}

// NewMTLSMonitor creates a new mTLS monitor.

func NewMTLSMonitor(logger *logging.StructuredLogger) *MTLSMonitor {
	if logger == nil {
		logger = logging.NewStructuredLogger(logging.DefaultConfig("mtls-monitor", "1.0.0", "production"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &MTLSMonitor{
		logger: logger,

		connections: make(map[string]*ConnectionInfo),

		certificates: make(map[string]*CertificateMonitorInfo),

		collectors: make([]MetricCollector, 0),

		alerts: make([]AlertRule, 0),

		ctx: ctx,

		cancel: cancel,
	}

	return monitor
}

// AddMetricCollector adds a metric collector to the monitor.

func (m *MTLSMonitor) AddMetricCollector(collector MetricCollector) {
	m.collectors = append(m.collectors, collector)
}

// AddAlertRule adds an alerting rule to the monitor.

func (m *MTLSMonitor) AddAlertRule(rule AlertRule) {
	m.alerts = append(m.alerts, rule)
}

// TrackConnection tracks a new mTLS connection.

func (m *MTLSMonitor) TrackConnection(connID, serviceName, remoteAddr, localAddr string, tlsInfo *TLSConnectionInfo) {
	m.connMu.Lock()

	defer m.connMu.Unlock()
	
	// Create empty JSON metadata for 2025 security best practices
	emptyMetadata := json.RawMessage(`{"version":"1.0","security_level":"high"}`)

	connInfo := &ConnectionInfo{
		ID: connID,

		ServiceName: serviceName,

		RemoteAddr: remoteAddr,

		LocalAddr: localAddr,

		Protocol: "TLS",

		CipherSuite: tlsInfo.CipherSuite,

		TLSVersion: tlsInfo.Version,

		ConnectedAt: time.Now(),

		LastActivity: time.Now(),

		RequestCount: 0,

		ErrorCount: 0,

		Metadata: emptyMetadata,
	}

	// Extract certificate information if available.

	if tlsInfo.PeerCertificate != nil {
		connInfo.CertificateInfo = &ConnectionCertInfo{
			Subject: tlsInfo.PeerCertificate.Subject.String(),

			Issuer: tlsInfo.PeerCertificate.Issuer.String(),

			SerialNumber: tlsInfo.PeerCertificate.SerialNumber.String(),

			NotBefore: tlsInfo.PeerCertificate.NotBefore,

			NotAfter: tlsInfo.PeerCertificate.NotAfter,

			IsExpired: time.Now().After(tlsInfo.PeerCertificate.NotAfter),

			ExpiresIn: int64(time.Until(tlsInfo.PeerCertificate.NotAfter).Seconds()),
		}
	}

	m.connections[connID] = connInfo

	m.logger.Info("mTLS connection tracked",

		"connection_id", connID,

		"service_name", serviceName,

		"cipher_suite", fmt.Sprintf("0x%04x", tlsInfo.CipherSuite),

		"tls_version", fmt.Sprintf("0x%04x", tlsInfo.Version),

		"remote_addr", remoteAddr,
	)
}

// UpdateConnectionActivity updates the last activity time for a connection.

func (m *MTLSMonitor) UpdateConnectionActivity(connID string) {
	m.connMu.Lock()

	defer m.connMu.Unlock()

	if conn, exists := m.connections[connID]; exists {
		conn.LastActivity = time.Now()

		conn.RequestCount++
	}
}

// RecordConnectionError records an error for a connection.

func (m *MTLSMonitor) RecordConnectionError(connID string) {
	m.connMu.Lock()

	defer m.connMu.Unlock()

	if conn, exists := m.connections[connID]; exists {
		conn.ErrorCount++
	}
}

// RemoveConnection removes a connection from tracking.

func (m *MTLSMonitor) RemoveConnection(connID string) {
	m.connMu.Lock()

	defer m.connMu.Unlock()

	delete(m.connections, connID)

	m.logger.Info("mTLS connection removed", "connection_id", connID)
}

// TrackCertificate tracks a certificate for monitoring.

func (m *MTLSMonitor) TrackCertificate(serviceName string, role ServiceRole, certPath string, cert *x509.Certificate) {
	m.certMu.Lock()

	defer m.certMu.Unlock()

	key := fmt.Sprintf("%s-%s", serviceName, role)

	expiresInDays := int(time.Until(cert.NotAfter).Hours() / 24)

	healthStatus := m.calculateCertificateHealth(cert)
	
	// Create security-aware metadata for 2025 standards
	certMetadata, _ := json.Marshal(map[string]interface{}{
		"version": "1.0",
		"security_level": "enterprise",
		"fips_compliant": true,
		"key_algorithm": cert.PublicKeyAlgorithm.String(),
		"signature_algorithm": cert.SignatureAlgorithm.String(),
	})

	certInfo := &CertificateMonitorInfo{
		ServiceName: serviceName,

		Role: role,

		CertificatePath: certPath,

		Certificate: cert,

		SerialNumber: cert.SerialNumber.String(),

		Subject: cert.Subject.String(),

		Issuer: cert.Issuer.String(),

		NotBefore: cert.NotBefore,

		NotAfter: cert.NotAfter,

		IsExpired: time.Now().After(cert.NotAfter),

		ExpiresInDays: expiresInDays,

		LastChecked: time.Now(),

		HealthStatus: healthStatus,

		Metadata: json.RawMessage(certMetadata),
	}

	// Update rotation count if certificate exists.

	if existing, exists := m.certificates[key]; exists {
		if existing.SerialNumber != certInfo.SerialNumber {
			certInfo.RotationCount = existing.RotationCount + 1
		} else {
			certInfo.RotationCount = existing.RotationCount
		}
	}

	m.certificates[key] = certInfo

	m.logger.Info("Certificate tracked",

		"service_name", serviceName,

		"role", role,

		"expires_in_days", expiresInDays,

		"health_status", healthStatus,

		"serial_number", cert.SerialNumber.String(),
	)
}

// GetConnectionStats returns connection statistics.

func (m *MTLSMonitor) GetConnectionStats() *ConnectionStats {
	m.connMu.RLock()

	defer m.connMu.RUnlock()

	stats := &ConnectionStats{
		TotalConnections: len(m.connections),

		ServiceCounts: make(map[string]int),

		TLSVersionCounts: make(map[uint16]int),

		CipherCounts: make(map[uint16]int),
	}

	var totalRequests, totalErrors int64

	for _, conn := range m.connections {
		stats.ServiceCounts[conn.ServiceName]++

		stats.TLSVersionCounts[conn.TLSVersion]++

		stats.CipherCounts[conn.CipherSuite]++

		totalRequests += conn.RequestCount

		totalErrors += conn.ErrorCount
	}

	stats.TotalRequests = totalRequests

	stats.TotalErrors = totalErrors

	if totalRequests > 0 {
		stats.ErrorRate = float64(totalErrors) / float64(totalRequests)
	}

	return stats
}

// GetCertificateStats returns certificate statistics.

func (m *MTLSMonitor) GetCertificateStats() *CertificateStats {
	m.certMu.RLock()

	defer m.certMu.RUnlock()

	stats := &CertificateStats{
		TotalCertificates: len(m.certificates),

		ServiceCounts: make(map[string]int),

		RoleCounts: make(map[ServiceRole]int),

		HealthCounts: make(map[CertificateHealth]int),
	}

	now := time.Now()

	for _, cert := range m.certificates {
		stats.ServiceCounts[cert.ServiceName]++

		stats.RoleCounts[cert.Role]++

		stats.HealthCounts[cert.HealthStatus]++

		if cert.IsExpired {
			stats.ExpiredCount++
		}

		// Check if expiring within 7 days.

		if !cert.IsExpired && cert.NotAfter.Sub(now) < 7*24*time.Hour {
			stats.ExpiringCount++
		}
	}

	return stats
}

// CollectMetrics collects all metrics from registered collectors.

func (m *MTLSMonitor) CollectMetrics() ([]*Metric, error) {
	var allMetrics []*Metric

	for _, collector := range m.collectors {
		metrics, err := collector.CollectMetrics(m)
		if err != nil {
			m.logger.Error("Failed to collect metrics",

				"collector", collector.GetName(),

				"error", err,
			)

			continue
		}

		allMetrics = append(allMetrics, metrics...)
	}

	return allMetrics, nil
}

// GetMetrics returns current monitoring metrics - compatibility method for integration.

func (m *MTLSMonitor) GetMetrics() ([]*Metric, error) {
	return m.CollectMetrics()
}

// CheckAlerts checks all alert rules and returns triggered alerts.

func (m *MTLSMonitor) CheckAlerts() []*Alert {
	var triggeredAlerts []*Alert

	for _, rule := range m.alerts {
		if !rule.Enabled {
			continue
		}

		alert := m.checkAlertRule(&rule)

		if alert != nil {
			triggeredAlerts = append(triggeredAlerts, alert)

			m.triggerAlert(alert)
		}
	}

	return triggeredAlerts
}

// Start starts the monitoring loop.

func (m *MTLSMonitor) Start() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds

	defer ticker.Stop()

	for {
		select {

		case <-m.ctx.Done():

			m.logger.Info("mTLS monitor stopped")

			return

		case <-ticker.C:

			m.runMonitoringCycle()
		}
	}
}

// Stop stops the monitoring loop.

func (m *MTLSMonitor) Stop() {
	m.cancel()
}

// Close gracefully shuts down the monitor - compatibility method for integration.

func (m *MTLSMonitor) Close() error {
	m.Stop()
	return nil
}

// runMonitoringCycle runs a complete monitoring cycle.

func (m *MTLSMonitor) runMonitoringCycle() {
	// Update certificate health status.

	m.updateCertificateHealthStatus()

	// Check alerts.

	alerts := m.CheckAlerts()

	if len(alerts) > 0 {
		m.logger.Warn("mTLS alerts triggered",

			"alert_count", len(alerts),
		)
	}

	// Clean up stale connections.

	m.cleanupStaleConnections()
}

// updateCertificateHealthStatus updates the health status of all tracked certificates.

func (m *MTLSMonitor) updateCertificateHealthStatus() {
	m.certMu.Lock()

	defer m.certMu.Unlock()

	for _, cert := range m.certificates {
		cert.HealthStatus = m.calculateCertificateHealth(cert.Certificate)

		cert.LastChecked = time.Now()

		cert.IsExpired = time.Now().After(cert.NotAfter)

		cert.ExpiresInDays = int(time.Until(cert.NotAfter).Hours() / 24)
	}
}

// cleanupStaleConnections removes connections that have been inactive for too long.

func (m *MTLSMonitor) cleanupStaleConnections() {
	m.connMu.Lock()

	defer m.connMu.Unlock()

	staleThreshold := time.Now().Add(-1 * time.Hour) // 1 hour

	var staleConnections []string

	for connID, conn := range m.connections {
		if conn.LastActivity.Before(staleThreshold) {
			staleConnections = append(staleConnections, connID)
		}
	}

	for _, connID := range staleConnections {
		delete(m.connections, connID)

		m.logger.Debug("Removed stale connection", "connection_id", connID)
	}

	if len(staleConnections) > 0 {
		m.logger.Info("Cleaned up stale connections",

			"removed_count", len(staleConnections),
		)
	}
}

// calculateCertificateHealth determines the health status of a certificate.

func (m *MTLSMonitor) calculateCertificateHealth(cert *x509.Certificate) CertificateHealth {
	now := time.Now()

	if now.After(cert.NotAfter) {
		return CertHealthExpired
	}

	timeUntilExpiry := cert.NotAfter.Sub(now)

	// Critical: Less than 7 days.

	if timeUntilExpiry < 7*24*time.Hour {
		return CertHealthCritical
	}

	// Warning: Less than 30 days.

	if timeUntilExpiry < 30*24*time.Hour {
		return CertHealthWarning
	}

	return CertHealthHealthy
}

// checkAlertRule checks a specific alert rule.

func (m *MTLSMonitor) checkAlertRule(rule *AlertRule) *Alert {
	switch rule.Condition.Type {

	case AlertTypeCertificateExpiring:

		return m.checkCertificateExpiringAlert(rule)

	case AlertTypeCertificateExpired:

		return m.checkCertificateExpiredAlert(rule)

	case AlertTypeConnectionFailure:

		return m.checkConnectionFailureAlert(rule)

	case AlertTypeHighErrorRate:

		return m.checkHighErrorRateAlert(rule)

	default:

		return nil
	}
}

// checkCertificateExpiringAlert checks for certificate expiry alerts.

func (m *MTLSMonitor) checkCertificateExpiringAlert(rule *AlertRule) *Alert {
	m.certMu.RLock()

	defer m.certMu.RUnlock()

	thresholdDays := int(rule.Condition.Threshold)

	for _, cert := range m.certificates {
		if !cert.IsExpired && cert.ExpiresInDays <= thresholdDays {
			return &Alert{
				Name: rule.Name,

				Severity: rule.Severity,

				Message: fmt.Sprintf("Certificate for %s expires in %d days", cert.ServiceName, cert.ExpiresInDays),

				Labels: map[string]string{
					"service": cert.ServiceName,

					"role": string(cert.Role),

					"expires_in_days": fmt.Sprintf("%d", cert.ExpiresInDays),
				},

				Timestamp: time.Now(),

				Metadata: json.RawMessage(`{}`),
			}
		}
	}

	return nil
}

// checkCertificateExpiredAlert checks for expired certificate alerts.

func (m *MTLSMonitor) checkCertificateExpiredAlert(rule *AlertRule) *Alert {
	m.certMu.RLock()

	defer m.certMu.RUnlock()

	for _, cert := range m.certificates {
		if cert.IsExpired {
			return &Alert{
				Name: rule.Name,

				Severity: rule.Severity,

				Message: fmt.Sprintf("Certificate for %s has expired", cert.ServiceName),

				Labels: map[string]string{
					"service": cert.ServiceName,

					"role": string(cert.Role),

					"expired_days_ago": fmt.Sprintf("%d", int(time.Since(cert.NotAfter).Hours()/24)),
				},

				Timestamp: time.Now(),

				Metadata: json.RawMessage(`{}`),
			}
		}
	}

	return nil
}

// checkConnectionFailureAlert checks for connection failure alerts.

func (m *MTLSMonitor) checkConnectionFailureAlert(rule *AlertRule) *Alert {
	stats := m.GetConnectionStats()

	if stats.ErrorRate > rule.Condition.Threshold {
		return &Alert{
			Name: rule.Name,

			Severity: rule.Severity,

			Message: fmt.Sprintf("High error rate detected: %.2f%%", stats.ErrorRate*100),

			Labels: map[string]string{
				"error_rate": fmt.Sprintf("%.2f", stats.ErrorRate),
			},

			Timestamp: time.Now(),

			Metadata: json.RawMessage(`{}`),
		}
	}

	return nil
}

// checkHighErrorRateAlert checks for high error rate alerts.

func (m *MTLSMonitor) checkHighErrorRateAlert(rule *AlertRule) *Alert {
	return m.checkConnectionFailureAlert(rule) // Same logic
}

// triggerAlert triggers an alert.

func (m *MTLSMonitor) triggerAlert(alert *Alert) {
	m.logger.Warn("mTLS alert triggered",

		"alert_name", alert.Name,

		"severity", alert.Severity,

		"message", alert.Message,
	)
}

// MetricCollector is an interface for collecting metrics.

type MetricCollector interface {
	GetName() string

	CollectMetrics(monitor *MTLSMonitor) ([]*Metric, error)
}

// MonitorConfig holds configuration for the mTLS monitor.

type MonitorConfig struct {
	MetricCollectionInterval time.Duration `json:"metric_collection_interval"`

	AlertCheckInterval time.Duration `json:"alert_check_interval"`

	ConnectionTimeout time.Duration `json:"connection_timeout"`

	CertificateCheckInterval time.Duration `json:"certificate_check_interval"`
}

// DefaultMonitorConfig returns a default monitor configuration.

func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		MetricCollectionInterval: 30 * time.Second,

		AlertCheckInterval: 60 * time.Second,

		ConnectionTimeout: 5 * time.Minute,

		CertificateCheckInterval: 10 * time.Minute,
	}
}

// TLSConnectionInfo holds TLS connection information.

type TLSConnectionInfo struct {
	Version uint16

	CipherSuite uint16

	PeerCertificate *x509.Certificate
}

// ConnectionStats holds connection statistics.

type ConnectionStats struct {
	TotalConnections int `json:"total_connections"`

	TotalRequests int64 `json:"total_requests"`

	TotalErrors int64 `json:"total_errors"`

	ErrorRate float64 `json:"error_rate"`

	TotalBytesSent int64 `json:"total_bytes_sent"`

	TotalBytesReceived int64 `json:"total_bytes_received"`

	ServiceCounts map[string]int `json:"service_counts"`

	TLSVersionCounts map[uint16]int `json:"tls_version_counts"`

	CipherCounts map[uint16]int `json:"cipher_counts"`
}

// CertificateStats holds certificate statistics.

type CertificateStats struct {
	TotalCertificates int `json:"total_certificates"`

	ExpiredCount int `json:"expired_count"`

	ExpiringCount int `json:"expiring_count"`

	ServiceCounts map[string]int `json:"service_counts"`

	RoleCounts map[ServiceRole]int `json:"role_counts"`

	HealthCounts map[CertificateHealth]int `json:"health_counts"`
}