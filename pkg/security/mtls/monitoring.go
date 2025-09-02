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

	// Monitoring state.

	ctx context.Context

	cancel context.CancelFunc

	monitoringTicker *time.Ticker

	alertTicker *time.Ticker
}

// ConnectionInfo tracks information about mTLS connections.

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

	BytesSent int64 `json:"bytes_sent"`

	BytesReceived int64 `json:"bytes_received"`

	RequestCount int64 `json:"request_count"`

	ErrorCount int64 `json:"error_count"`

	CertificateInfo *ConnectionCertInfo `json:"certificate_info"`

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

// CertificateMonitorInfo tracks certificate monitoring information.

type CertificateMonitorInfo struct {
	ServiceName string `json:"service_name"`

	Role ServiceRole `json:"role"`

	CertificatePath string `json:"certificate_path"`

	Certificate *x509.Certificate `json:"-"`

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

	// CertHealthInvalid holds certhealthinvalid value.

	CertHealthInvalid CertificateHealth = "invalid"
)

// MetricCollector defines the interface for metric collection.

type MetricCollector interface {
	CollectMetrics(monitor *MTLSMonitor) ([]*Metric, error)

	GetName() string
}

// AlertRule defines alert rules for mTLS monitoring.

type AlertRule struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Condition AlertCondition `json:"condition"`

	Severity AlertSeverity `json:"severity"`

	Enabled bool `json:"enabled"`

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

	// AlertTypeCertificateExpiry holds alerttypecertificateexpiry value.

	AlertTypeCertificateExpiry AlertType = "certificate_expiry"

	// AlertTypeConnectionFailure holds alerttypeconnectionfailure value.

	AlertTypeConnectionFailure AlertType = "connection_failure"

	// AlertTypeHighErrorRate holds alerttypehigherrorrate value.

	AlertTypeHighErrorRate AlertType = "high_error_rate"

	// AlertTypeCertificateRotationFail holds alerttypecertificaterotationfail value.

	AlertTypeCertificateRotationFail AlertType = "certificate_rotation_fail"

	// AlertTypeSecurityViolation holds alerttypesecurityviolation value.

	AlertTypeSecurityViolation AlertType = "security_violation"
)

// AlertSeverity represents alert severity levels.

type AlertSeverity string

const (

	// AlertSeverityInfo holds alertseverityinfo value.

	AlertSeverityInfo AlertSeverity = "info"

	// AlertSeverityWarning holds alertseveritywarning value.

	AlertSeverityWarning AlertSeverity = "warning"

	// AlertSeverityCritical holds alertseveritycritical value.

	AlertSeverityCritical AlertSeverity = "critical"
)

// Metric represents a monitoring metric.

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

		alerts: getDefaultAlertRules(),

		ctx: ctx,

		cancel: cancel,
	}

	// Add default metric collectors.

	monitor.AddCollector(&ConnectionMetricCollector{})

	monitor.AddCollector(&CertificateMetricCollector{})

	monitor.AddCollector(&SecurityMetricCollector{})

	// Start monitoring routines.

	monitor.startMonitoring()

	logger.Info("mTLS monitor initialized")

	return monitor
}

// TrackConnection tracks a new mTLS connection.

func (m *MTLSMonitor) TrackConnection(connID, serviceName, remoteAddr, localAddr string, tlsInfo *TLSConnectionInfo) {
	m.connMu.Lock()

	defer m.connMu.Unlock()

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

		Metadata: make(map[string]interface{}),
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

	m.logger.Debug("tracking new mTLS connection",

		"connection_id", connID,

		"service_name", serviceName,

		"remote_addr", remoteAddr,

		"tls_version", tlsInfo.Version,

		"cipher_suite", tlsInfo.CipherSuite)
}

// UpdateConnectionActivity updates connection activity metrics.

func (m *MTLSMonitor) UpdateConnectionActivity(connID string, bytesSent, bytesReceived, requestCount, errorCount int64) {
	m.connMu.Lock()

	defer m.connMu.Unlock()

	if conn, exists := m.connections[connID]; exists {

		conn.LastActivity = time.Now()

		conn.BytesSent += bytesSent

		conn.BytesReceived += bytesReceived

		conn.RequestCount += requestCount

		conn.ErrorCount += errorCount

	}
}

// CloseConnection removes connection tracking.

func (m *MTLSMonitor) CloseConnection(connID string) {
	m.connMu.Lock()

	defer m.connMu.Unlock()

	if conn, exists := m.connections[connID]; exists {

		duration := time.Since(conn.ConnectedAt)

		m.logger.Debug("closing tracked mTLS connection",

			"connection_id", connID,

			"service_name", conn.ServiceName,

			"duration", duration,

			"requests", conn.RequestCount,

			"errors", conn.ErrorCount)

		delete(m.connections, connID)

	}
}

// TrackCertificate tracks a certificate for monitoring.

func (m *MTLSMonitor) TrackCertificate(serviceName string, role ServiceRole, certPath string, cert *x509.Certificate) {
	m.certMu.Lock()

	defer m.certMu.Unlock()

	key := fmt.Sprintf("%s-%s", serviceName, role)

	expiresInDays := int(time.Until(cert.NotAfter).Hours() / 24)

	healthStatus := m.calculateCertificateHealth(cert)

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

		Metadata: make(map[string]interface{}),
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

	m.logger.Debug("tracking certificate",

		"service_name", serviceName,

		"role", role,

		"serial_number", certInfo.SerialNumber,

		"expires_in_days", expiresInDays,

		"health_status", healthStatus)
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

	var totalRequests, totalErrors, totalBytesSent, totalBytesReceived int64

	for _, conn := range m.connections {

		stats.ServiceCounts[conn.ServiceName]++

		stats.TLSVersionCounts[conn.TLSVersion]++

		stats.CipherCounts[conn.CipherSuite]++

		totalRequests += conn.RequestCount

		totalErrors += conn.ErrorCount

		totalBytesSent += conn.BytesSent

		totalBytesReceived += conn.BytesReceived

	}

	stats.TotalRequests = totalRequests

	stats.TotalErrors = totalErrors

	stats.TotalBytesSent = totalBytesSent

	stats.TotalBytesReceived = totalBytesReceived

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

		HealthCounts: make(map[CertificateHealth]int),

		ServiceCounts: make(map[string]int),

		RoleCounts: make(map[ServiceRole]int),
	}

	var expiredCount, expiringCount int

	for _, cert := range m.certificates {

		stats.HealthCounts[cert.HealthStatus]++

		stats.ServiceCounts[cert.ServiceName]++

		stats.RoleCounts[cert.Role]++

		if cert.IsExpired {
			expiredCount++
		} else if cert.ExpiresInDays <= 7 {
			expiringCount++
		}

	}

	stats.ExpiredCount = expiredCount

	stats.ExpiringCount = expiringCount

	return stats
}

// AddCollector adds a metric collector.

func (m *MTLSMonitor) AddCollector(collector MetricCollector) {
	m.collectors = append(m.collectors, collector)

	m.logger.Debug("added metric collector", "name", collector.GetName())
}

// GetMetrics collects metrics from all collectors.

func (m *MTLSMonitor) GetMetrics() ([]*Metric, error) {
	var allMetrics []*Metric

	for _, collector := range m.collectors {

		metrics, err := collector.CollectMetrics(m)
		if err != nil {

			m.logger.Error("failed to collect metrics",

				"collector", collector.GetName(),

				"error", err)

			continue

		}

		allMetrics = append(allMetrics, metrics...)

	}

	return allMetrics, nil
}

// calculateCertificateHealth calculates the health status of a certificate.

func (m *MTLSMonitor) calculateCertificateHealth(cert *x509.Certificate) CertificateHealth {
	now := time.Now()

	if now.After(cert.NotAfter) {
		return CertHealthExpired
	}

	timeUntilExpiry := time.Until(cert.NotAfter)

	daysUntilExpiry := timeUntilExpiry.Hours() / 24

	if daysUntilExpiry <= 1 {
		return CertHealthCritical
	} else if daysUntilExpiry <= 7 {
		return CertHealthWarning
	}

	return CertHealthHealthy
}

// startMonitoring starts monitoring routines.

func (m *MTLSMonitor) startMonitoring() {
	// Monitor certificates every 5 minutes.

	m.monitoringTicker = time.NewTicker(5 * time.Minute)

	go m.monitoringLoop()

	// Check alerts every minute.

	m.alertTicker = time.NewTicker(1 * time.Minute)

	go m.alertLoop()
}

// monitoringLoop runs the main monitoring loop.

func (m *MTLSMonitor) monitoringLoop() {
	defer m.monitoringTicker.Stop()

	for {
		select {

		case <-m.ctx.Done():

			return

		case <-m.monitoringTicker.C:

			m.performHealthChecks()

		}
	}
}

// alertLoop runs the alerting loop.

func (m *MTLSMonitor) alertLoop() {
	defer m.alertTicker.Stop()

	for {
		select {

		case <-m.ctx.Done():

			return

		case <-m.alertTicker.C:

			m.checkAlerts()

		}
	}
}

// performHealthChecks performs health checks on tracked certificates.

func (m *MTLSMonitor) performHealthChecks() {
	m.certMu.Lock()

	defer m.certMu.Unlock()

	for _, cert := range m.certificates {

		oldHealth := cert.HealthStatus

		cert.HealthStatus = m.calculateCertificateHealth(cert.Certificate)

		cert.LastChecked = time.Now()

		cert.IsExpired = time.Now().After(cert.NotAfter)

		cert.ExpiresInDays = int(time.Until(cert.NotAfter).Hours() / 24)

		if oldHealth != cert.HealthStatus {
			m.logger.Info("certificate health status changed",

				"service_name", cert.ServiceName,

				"role", cert.Role,

				"old_status", oldHealth,

				"new_status", cert.HealthStatus,

				"expires_in_days", cert.ExpiresInDays)
		}

	}
}

// checkAlerts checks alert conditions and triggers alerts.

func (m *MTLSMonitor) checkAlerts() {
	for _, rule := range m.alerts {

		if !rule.Enabled {
			continue
		}

		if alert := m.evaluateAlertRule(&rule); alert != nil {
			m.triggerAlert(alert)
		}

	}
}

// evaluateAlertRule evaluates an alert rule.

func (m *MTLSMonitor) evaluateAlertRule(rule *AlertRule) *Alert {
	switch rule.Condition.Type {

	case AlertTypeCertificateExpiry:

		return m.checkCertificateExpiryAlert(rule)

	case AlertTypeConnectionFailure:

		return m.checkConnectionFailureAlert(rule)

	case AlertTypeHighErrorRate:

		return m.checkHighErrorRateAlert(rule)

	default:

		return nil

	}
}

// checkCertificateExpiryAlert checks for certificate expiry alerts.

func (m *MTLSMonitor) checkCertificateExpiryAlert(rule *AlertRule) *Alert {
	m.certMu.RLock()

	defer m.certMu.RUnlock()

	for _, cert := range m.certificates {
		if cert.ExpiresInDays <= int(rule.Condition.Threshold) {
			return &Alert{
				Name: rule.Name,

				Severity: rule.Severity,

				Message: fmt.Sprintf("Certificate for %s (%s) expires in %d days", cert.ServiceName, cert.Role, cert.ExpiresInDays),

				Labels: map[string]string{
					"service_name": cert.ServiceName,

					"role": string(cert.Role),

					"serial_number": cert.SerialNumber,
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

		"labels", alert.Labels)

	// Here you would typically send the alert to your alerting system.

	// For now, we just log it.
}

// Close stops the monitor and cleans up resources.

func (m *MTLSMonitor) Close() error {
	m.logger.Info("shutting down mTLS monitor")

	m.cancel()

	if m.monitoringTicker != nil {
		m.monitoringTicker.Stop()
	}

	if m.alertTicker != nil {
		m.alertTicker.Stop()
	}

	return nil
}

// getDefaultAlertRules returns default alert rules.

func getDefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			Name: "certificate_expiring_soon",

			Description: "Certificate expires within 7 days",

			Condition: AlertCondition{
				Type: AlertTypeCertificateExpiry,

				Threshold: 7,
			},

			Severity: AlertSeverityWarning,

			Enabled: true,
		},

		{
			Name: "certificate_expiring_critical",

			Description: "Certificate expires within 1 day",

			Condition: AlertCondition{
				Type: AlertTypeCertificateExpiry,

				Threshold: 1,
			},

			Severity: AlertSeverityCritical,

			Enabled: true,
		},

		{
			Name: "high_error_rate",

			Description: "High error rate on mTLS connections",

			Condition: AlertCondition{
				Type: AlertTypeHighErrorRate,

				Threshold: 0.05, // 5% error rate

			},

			Severity: AlertSeverityCritical,

			Enabled: true,
		},
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

	TotalBytesSent int64 `json:"total_bytes_sent"`

	TotalBytesReceived int64 `json:"total_bytes_received"`

	ErrorRate float64 `json:"error_rate"`

	ServiceCounts map[string]int `json:"service_counts"`

	TLSVersionCounts map[uint16]int `json:"tls_version_counts"`

	CipherCounts map[uint16]int `json:"cipher_counts"`
}

// CertificateStats holds certificate statistics.

type CertificateStats struct {
	TotalCertificates int `json:"total_certificates"`

	ExpiredCount int `json:"expired_count"`

	ExpiringCount int `json:"expiring_count"`

	HealthCounts map[CertificateHealth]int `json:"health_counts"`

	ServiceCounts map[string]int `json:"service_counts"`

	RoleCounts map[ServiceRole]int `json:"role_counts"`
}

