package mtls

import (
	"fmt"
	"time"
)

// ConnectionMetricCollector collects metrics related to mTLS connections.

type ConnectionMetricCollector struct{}

// GetName returns the name of the collector.

func (c *ConnectionMetricCollector) GetName() string {
	return "connection_metrics"
}

// CollectMetrics collects connection-related metrics.

func (c *ConnectionMetricCollector) CollectMetrics(monitor *MTLSMonitor) ([]*Metric, error) {
	stats := monitor.GetConnectionStats()

	timestamp := time.Now()

	metrics := []*Metric{
		{
			Name: "mtls_connections_total",

			Value: float64(stats.TotalConnections),

			Type: "gauge",

			Help: "Total number of active mTLS connections",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_requests_total",

			Value: float64(stats.TotalRequests),

			Type: "counter",

			Help: "Total number of mTLS requests processed",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_errors_total",

			Value: float64(stats.TotalErrors),

			Type: "counter",

			Help: "Total number of mTLS connection errors",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_error_rate",

			Value: stats.ErrorRate,

			Type: "gauge",

			Help: "Error rate for mTLS connections (0.0-1.0)",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_bytes_sent_total",

			Value: float64(stats.TotalBytesSent),

			Type: "counter",

			Help: "Total bytes sent over mTLS connections",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_bytes_received_total",

			Value: float64(stats.TotalBytesReceived),

			Type: "counter",

			Help: "Total bytes received over mTLS connections",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},
	}

	// Add per-service connection metrics.

	for serviceName, count := range stats.ServiceCounts {
		metrics = append(metrics, &Metric{
			Name: "mtls_connections_by_service",

			Value: float64(count),

			Type: "gauge",

			Help: "Number of mTLS connections by service",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"service": serviceName,
			},
		})
	}

	// Add TLS version metrics.

	for version, count := range stats.TLSVersionCounts {

		versionStr := fmt.Sprintf("0x%04x", version)

		tlsVersionName := getTLSVersionName(version)

		metrics = append(metrics, &Metric{
			Name: "mtls_connections_by_tls_version",

			Value: float64(count),

			Type: "gauge",

			Help: "Number of mTLS connections by TLS version",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"tls_version": versionStr,

				"version_name": tlsVersionName,
			},
		})

	}

	// Add cipher suite metrics.

	for cipher, count := range stats.CipherCounts {

		cipherStr := fmt.Sprintf("0x%04x", cipher)

		cipherName := getCipherSuiteName(cipher)

		metrics = append(metrics, &Metric{
			Name: "mtls_connections_by_cipher_suite",

			Value: float64(count),

			Type: "gauge",

			Help: "Number of mTLS connections by cipher suite",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"cipher_suite": cipherStr,

				"cipher_name": cipherName,
			},
		})

	}

	return metrics, nil
}

// CertificateMetricCollector collects metrics related to certificates.

type CertificateMetricCollector struct{}

// GetName returns the name of the collector.

func (c *CertificateMetricCollector) GetName() string {
	return "certificate_metrics"
}

// CollectMetrics collects certificate-related metrics.

func (c *CertificateMetricCollector) CollectMetrics(monitor *MTLSMonitor) ([]*Metric, error) {
	stats := monitor.GetCertificateStats()

	timestamp := time.Now()

	metrics := []*Metric{
		{
			Name: "mtls_certificates_total",

			Value: float64(stats.TotalCertificates),

			Type: "gauge",

			Help: "Total number of tracked certificates",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_certificates_expired",

			Value: float64(stats.ExpiredCount),

			Type: "gauge",

			Help: "Number of expired certificates",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_certificates_expiring",

			Value: float64(stats.ExpiringCount),

			Type: "gauge",

			Help: "Number of certificates expiring within 7 days",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},
	}

	// Add health status metrics.

	for health, count := range stats.HealthCounts {
		metrics = append(metrics, &Metric{
			Name: "mtls_certificates_by_health",

			Value: float64(count),

			Type: "gauge",

			Help: "Number of certificates by health status",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"health": string(health),
			},
		})
	}

	// Add per-service certificate metrics.

	for serviceName, count := range stats.ServiceCounts {
		metrics = append(metrics, &Metric{
			Name: "mtls_certificates_by_service",

			Value: float64(count),

			Type: "gauge",

			Help: "Number of certificates by service",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"service": serviceName,
			},
		})
	}

	// Add per-role certificate metrics.

	for role, count := range stats.RoleCounts {
		metrics = append(metrics, &Metric{
			Name: "mtls_certificates_by_role",

			Value: float64(count),

			Type: "gauge",

			Help: "Number of certificates by service role",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"role": string(role),
			},
		})
	}

	// Add individual certificate expiry metrics.

	monitor.certMu.RLock()

	for _, cert := range monitor.certificates {

		expiresInSeconds := time.Until(cert.NotAfter).Seconds()

		metrics = append(metrics, &Metric{
			Name: "mtls_certificate_expiry_time",

			Value: expiresInSeconds,

			Type: "gauge",

			Help: "Time until certificate expiry in seconds",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"service": cert.ServiceName,

				"role": string(cert.Role),

				"serial_number": cert.SerialNumber,

				"health_status": string(cert.HealthStatus),
			},

			Metadata: map[string]interface{}{
				"certificate_path": cert.CertificatePath,

				"expires_at": cert.NotAfter,

				"rotation_count": cert.RotationCount,
			},
		})

	}

	monitor.certMu.RUnlock()

	return metrics, nil
}

// SecurityMetricCollector collects security-related metrics.

type SecurityMetricCollector struct{}

// GetName returns the name of the collector.

func (c *SecurityMetricCollector) GetName() string {
	return "security_metrics"
}

// CollectMetrics collects security-related metrics.

func (c *SecurityMetricCollector) CollectMetrics(monitor *MTLSMonitor) ([]*Metric, error) {
	timestamp := time.Now()

	metrics := []*Metric{}

	// Collect certificate rotation metrics.

	monitor.certMu.RLock()

	totalRotations := 0

	for _, cert := range monitor.certificates {

		totalRotations += cert.RotationCount

		// Individual rotation count.

		metrics = append(metrics, &Metric{
			Name: "mtls_certificate_rotations_total",

			Value: float64(cert.RotationCount),

			Type: "counter",

			Help: "Total number of certificate rotations",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"service": cert.ServiceName,

				"role": string(cert.Role),
			},
		})

	}

	monitor.certMu.RUnlock()

	// Total rotations across all certificates.

	metrics = append(metrics, &Metric{
		Name: "mtls_certificate_rotations_global_total",

		Value: float64(totalRotations),

		Type: "counter",

		Help: "Total number of certificate rotations across all services",

		Timestamp: timestamp,

		Labels: map[string]string{"component": "mtls"},
	})

	// Connection security metrics.

	monitor.connMu.RLock()

	secureConnections := 0

	weakCiphers := 0

	oldTLSVersions := 0

	for _, conn := range monitor.connections {

		secureConnections++

		// Check for weak cipher suites (simplified check).

		if isWeakCipherSuite(conn.CipherSuite) {
			weakCiphers++
		}

		// Check for old TLS versions.

		if conn.TLSVersion < 0x0303 { // Less than TLS 1.2

			oldTLSVersions++
		}

	}

	monitor.connMu.RUnlock()

	metrics = append(metrics, []*Metric{
		{
			Name: "mtls_secure_connections_total",

			Value: float64(secureConnections),

			Type: "gauge",

			Help: "Total number of secure mTLS connections",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_weak_cipher_connections",

			Value: float64(weakCiphers),

			Type: "gauge",

			Help: "Number of connections using weak cipher suites",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},

		{
			Name: "mtls_old_tls_version_connections",

			Value: float64(oldTLSVersions),

			Type: "gauge",

			Help: "Number of connections using old TLS versions",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		},
	}...)

	return metrics, nil
}

// Helper functions.

// getTLSVersionName returns a human-readable TLS version name.

func getTLSVersionName(version uint16) string {
	switch version {

	case 0x0300:

		return "SSL 3.0"

	case 0x0301:

		return "TLS 1.0"

	case 0x0302:

		return "TLS 1.1"

	case 0x0303:

		return "TLS 1.2"

	case 0x0304:

		return "TLS 1.3"

	default:

		return "Unknown"

	}
}

// getCipherSuiteName returns a human-readable cipher suite name.

func getCipherSuiteName(cipher uint16) string {
	// This is a simplified mapping - in practice, you'd want a more complete one.

	switch cipher {

	case 0x1301:

		return "TLS_AES_128_GCM_SHA256"

	case 0x1302:

		return "TLS_AES_256_GCM_SHA384"

	case 0x1303:

		return "TLS_CHACHA20_POLY1305_SHA256"

	case 0xc02f:

		return "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"

	case 0xc030:

		return "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"

	case 0xcca8:

		return "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"

	default:

		return fmt.Sprintf("Unknown_0x%04x", cipher)

	}
}

// isWeakCipherSuite checks if a cipher suite is considered weak.

func isWeakCipherSuite(cipher uint16) bool {
	// Define weak cipher suites (simplified list).

	weakCiphers := []uint16{
		0x0001, // TLS_RSA_WITH_NULL_MD5

		0x0002, // TLS_RSA_WITH_NULL_SHA

		0x0004, // TLS_RSA_WITH_RC4_128_MD5

		0x0005, // TLS_RSA_WITH_RC4_128_SHA

		0x000a, // TLS_RSA_WITH_3DES_EDE_CBC_SHA

		// Add more weak ciphers as needed.

	}

	for _, weak := range weakCiphers {
		if cipher == weak {
			return true
		}
	}

	return false
}

// PerformanceMetricCollector collects performance-related metrics.

type PerformanceMetricCollector struct{}

// GetName returns the name of the collector.

func (c *PerformanceMetricCollector) GetName() string {
	return "performance_metrics"
}

// CollectMetrics collects performance-related metrics.

func (c *PerformanceMetricCollector) CollectMetrics(monitor *MTLSMonitor) ([]*Metric, error) {
	timestamp := time.Now()

	metrics := []*Metric{}

	monitor.connMu.RLock()

	// Calculate connection duration metrics.

	var totalDuration float64

	var connectionDurations []float64

	activeConnections := 0

	for _, conn := range monitor.connections {

		duration := time.Since(conn.ConnectedAt).Seconds()

		connectionDurations = append(connectionDurations, duration)

		totalDuration += duration

		activeConnections++

		// Individual connection metrics.

		metrics = append(metrics, &Metric{
			Name: "mtls_connection_duration_seconds",

			Value: duration,

			Type: "gauge",

			Help: "Duration of individual mTLS connections in seconds",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"service": conn.ServiceName,

				"conn_id": conn.ID,
			},
		})

		// Request rate per connection.

		if duration > 0 {

			requestRate := float64(conn.RequestCount) / duration

			metrics = append(metrics, &Metric{
				Name: "mtls_connection_request_rate",

				Value: requestRate,

				Type: "gauge",

				Help: "Request rate per connection (requests per second)",

				Timestamp: timestamp,

				Labels: map[string]string{
					"component": "mtls",

					"service": conn.ServiceName,

					"conn_id": conn.ID,
				},
			})

		}

	}

	monitor.connMu.RUnlock()

	// Average connection duration.

	if activeConnections > 0 {

		avgDuration := totalDuration / float64(activeConnections)

		metrics = append(metrics, &Metric{
			Name: "mtls_connection_duration_average_seconds",

			Value: avgDuration,

			Type: "gauge",

			Help: "Average duration of mTLS connections in seconds",

			Timestamp: timestamp,

			Labels: map[string]string{"component": "mtls"},
		})

	}

	// Certificate check performance.

	monitor.certMu.RLock()

	for _, cert := range monitor.certificates {

		timeSinceLastCheck := time.Since(cert.LastChecked).Seconds()

		metrics = append(metrics, &Metric{
			Name: "mtls_certificate_check_age_seconds",

			Value: timeSinceLastCheck,

			Type: "gauge",

			Help: "Time since last certificate health check in seconds",

			Timestamp: timestamp,

			Labels: map[string]string{
				"component": "mtls",

				"service": cert.ServiceName,

				"role": string(cert.Role),
			},
		})

	}

	monitor.certMu.RUnlock()

	return metrics, nil
}
