package ca

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthChecker performs health checks for certificate rotation
type HealthChecker struct {
	logger *logging.StructuredLogger
	config *HealthCheckerConfig
	
	// Health check state
	activeChecks map[string]*HealthCheckSession
	mu           sync.RWMutex
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// HealthCheckerConfig configures the health checker
type HealthCheckerConfig struct {
	Enabled                bool          `yaml:"enabled"`
	DefaultTimeout         time.Duration `yaml:"default_timeout"`
	DefaultInterval        time.Duration `yaml:"default_interval"`
	DefaultHealthyThreshold int          `yaml:"default_healthy_threshold"`
	DefaultUnhealthyThreshold int        `yaml:"default_unhealthy_threshold"`
	ConcurrentChecks       int           `yaml:"concurrent_checks"`
	RetryAttempts          int           `yaml:"retry_attempts"`
	RetryDelay             time.Duration `yaml:"retry_delay"`
	TLSVerificationEnabled bool          `yaml:"tls_verification_enabled"`
	MetricsEnabled         bool          `yaml:"metrics_enabled"`
}

// HealthCheckSession represents an active health check session
type HealthCheckSession struct {
	ID            string                  `json:"id"`
	Type          HealthCheckType         `json:"type"`
	Target        *HealthCheckTarget      `json:"target"`
	Config        *HealthCheckConfig      `json:"config"`
	Status        HealthCheckStatus       `json:"status"`
	Results       []*HealthCheckResult    `json:"results"`
	StartTime     time.Time               `json:"start_time"`
	LastCheckTime time.Time               `json:"last_check_time"`
	TotalChecks   int                     `json:"total_checks"`
	SuccessCount  int                     `json:"success_count"`
	FailureCount  int                     `json:"failure_count"`
	mu            sync.RWMutex
}

// HealthCheckType represents the type of health check
type HealthCheckType string

const (
	HealthCheckTypeHTTP  HealthCheckType = "http"
	HealthCheckTypeHTTPS HealthCheckType = "https"
	HealthCheckTypeGRPC  HealthCheckType = "grpc"
	HealthCheckTypeTCP   HealthCheckType = "tcp"
	HealthCheckTypeUDP   HealthCheckType = "udp"
)

// HealthCheckTarget represents the target of a health check
type HealthCheckTarget struct {
	Name       string            `json:"name"`
	Address    string            `json:"address"`
	Port       int               `json:"port"`
	Path       string            `json:"path,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	TLSConfig  *TLSCheckConfig   `json:"tls_config,omitempty"`
	GRPCService string           `json:"grpc_service,omitempty"`
}

// TLSCheckConfig configures TLS-specific health checks
type TLSCheckConfig struct {
	Enabled              bool     `json:"enabled"`
	VerifyCertificate   bool     `json:"verify_certificate"`
	CertificateChain    bool     `json:"certificate_chain"`
	ExpiresSoon         bool     `json:"expires_soon"`
	ExpiryThreshold     time.Duration `json:"expiry_threshold"`
	AllowedSANs         []string `json:"allowed_sans"`
	RequiredKeyUsages   []string `json:"required_key_usages"`
}

// HealthCheckStatus represents the health check status
type HealthCheckStatus string

const (
	HealthCheckStatusHealthy     HealthCheckStatus = "healthy"
	HealthCheckStatusUnhealthy   HealthCheckStatus = "unhealthy"
	HealthCheckStatusUnknown     HealthCheckStatus = "unknown"
	HealthCheckStatusDegraded    HealthCheckStatus = "degraded"
)

// HealthCheckResult represents a single health check result
type HealthCheckResult struct {
	Timestamp       time.Time         `json:"timestamp"`
	Success         bool              `json:"success"`
	ResponseTime    time.Duration     `json:"response_time"`
	StatusCode      int               `json:"status_code,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	TLSInfo         *TLSCheckInfo     `json:"tls_info,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// TLSCheckInfo contains TLS-specific check information
type TLSCheckInfo struct {
	Version            string    `json:"version"`
	CipherSuite        string    `json:"cipher_suite"`
	CertificateValid   bool      `json:"certificate_valid"`
	CertificateExpiry  time.Time `json:"certificate_expiry"`
	CertificateSubject string    `json:"certificate_subject"`
	SANs               []string  `json:"sans"`
	ChainLength        int       `json:"chain_length"`
	ExpiresWithin      time.Duration `json:"expires_within,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(
	logger *logging.StructuredLogger,
	config *HealthCheckerConfig,
) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthChecker{
		logger:       logger,
		config:       config,
		activeChecks: make(map[string]*HealthCheckSession),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) error {
	if !hc.config.Enabled {
		hc.logger.Info("health checker is disabled")
		return nil
	}

	hc.logger.Info("starting health checker",
		"concurrent_checks", hc.config.ConcurrentChecks,
		"tls_verification", hc.config.TLSVerificationEnabled)

	// Start metrics collector if enabled
	if hc.config.MetricsEnabled {
		hc.wg.Add(1)
		go hc.runMetricsCollector()
	}

	// Start session cleanup
	hc.wg.Add(1)
	go hc.runSessionCleanup()

	// Wait for context cancellation
	<-ctx.Done()
	hc.cancel()
	hc.wg.Wait()

	return nil
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.logger.Info("stopping health checker")
	hc.cancel()
	hc.wg.Wait()
}

// StartHealthCheck starts a new health check session
func (hc *HealthChecker) StartHealthCheck(
	target *HealthCheckTarget,
	config *HealthCheckConfig,
) (*HealthCheckSession, error) {
	if !hc.config.Enabled {
		return nil, fmt.Errorf("health checker is disabled")
	}

	sessionID := fmt.Sprintf("hc-%s-%d", target.Name, time.Now().Unix())
	
	session := &HealthCheckSession{
		ID:        sessionID,
		Type:      hc.determineCheckType(target),
		Target:    target,
		Config:    config,
		Status:    HealthCheckStatusUnknown,
		Results:   make([]*HealthCheckResult, 0),
		StartTime: time.Now(),
	}

	hc.mu.Lock()
	hc.activeChecks[sessionID] = session
	hc.mu.Unlock()

	hc.logger.Info("started health check session",
		"session_id", sessionID,
		"target", target.Name,
		"type", session.Type,
		"address", fmt.Sprintf("%s:%d", target.Address, target.Port))

	return session, nil
}

// PerformHealthCheck performs a single health check
func (hc *HealthChecker) PerformHealthCheck(session *HealthCheckSession) (*HealthCheckResult, error) {
	startTime := time.Now()
	
	result := &HealthCheckResult{
		Timestamp:    startTime,
		Metadata:     make(map[string]string),
	}

	var err error
	switch session.Type {
	case HealthCheckTypeHTTP, HealthCheckTypeHTTPS:
		err = hc.performHTTPCheck(session, result)
	case HealthCheckTypeGRPC:
		err = hc.performGRPCCheck(session, result)
	case HealthCheckTypeTCP:
		err = hc.performTCPCheck(session, result)
	case HealthCheckTypeUDP:
		err = hc.performUDPCheck(session, result)
	default:
		err = fmt.Errorf("unsupported health check type: %s", session.Type)
	}

	result.ResponseTime = time.Since(startTime)
	result.Success = err == nil
	if err != nil {
		result.ErrorMessage = err.Error()
	}

	// Update session
	session.mu.Lock()
	session.Results = append(session.Results, result)
	session.LastCheckTime = time.Now()
	session.TotalChecks++
	if result.Success {
		session.SuccessCount++
	} else {
		session.FailureCount++
	}
	
	// Determine overall health status
	session.Status = hc.determineHealthStatus(session)
	session.mu.Unlock()

	hc.logger.Debug("performed health check",
		"session_id", session.ID,
		"success", result.Success,
		"response_time", result.ResponseTime,
		"status", session.Status)

	return result, err
}

// PerformContinuousHealthCheck performs continuous health checking
func (hc *HealthChecker) PerformContinuousHealthCheck(
	session *HealthCheckSession,
	duration time.Duration,
) error {
	if session.Config == nil {
		return fmt.Errorf("health check config is required")
	}

	timeout := time.After(duration)
	interval := session.Config.CheckInterval
	if interval == 0 {
		interval = hc.config.DefaultInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	hc.logger.Info("starting continuous health check",
		"session_id", session.ID,
		"duration", duration,
		"interval", interval)

	for {
		select {
		case <-hc.ctx.Done():
			return hc.ctx.Err()
		case <-timeout:
			hc.logger.Info("continuous health check completed",
				"session_id", session.ID,
				"total_checks", session.TotalChecks,
				"success_rate", hc.calculateSuccessRate(session))
			return nil
		case <-ticker.C:
			_, err := hc.PerformHealthCheck(session)
			if err != nil {
				hc.logger.Debug("health check failed",
					"session_id", session.ID,
					"error", err)
			}
		}
	}
}

// performHTTPCheck performs HTTP/HTTPS health check
func (hc *HealthChecker) performHTTPCheck(session *HealthCheckSession, result *HealthCheckResult) error {
	var scheme string
	if session.Type == HealthCheckTypeHTTPS {
		scheme = "https"
	} else {
		scheme = "http"
	}

	url := fmt.Sprintf("%s://%s:%d%s", scheme, session.Target.Address, session.Target.Port, session.Target.Path)
	
	// Create HTTP client with timeout
	timeout := hc.config.DefaultTimeout
	if session.Config != nil && session.Config.TimeoutPerCheck > 0 {
		timeout = session.Config.TimeoutPerCheck
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// Configure TLS if HTTPS
	if session.Type == HealthCheckTypeHTTPS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: !hc.config.TLSVerificationEnabled,
		}
		
		if session.Target.TLSConfig != nil && session.Target.TLSConfig.VerifyCertificate {
			tlsConfig.InsecureSkipVerify = false
		}

		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(hc.ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range session.Target.Headers {
		req.Header.Set(key, value)
	}

	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Metadata["url"] = url
	result.Metadata["method"] = "GET"

	// Check TLS information if HTTPS
	if session.Type == HealthCheckTypeHTTPS && resp.TLS != nil {
		tlsInfo := hc.extractTLSInfo(resp.TLS, session.Target.TLSConfig)
		result.TLSInfo = tlsInfo
		
		// Perform TLS-specific checks
		if err := hc.validateTLSInfo(tlsInfo, session.Target.TLSConfig); err != nil {
			return fmt.Errorf("TLS validation failed: %w", err)
		}
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

// performGRPCCheck performs gRPC health check
func (hc *HealthChecker) performGRPCCheck(session *HealthCheckSession, result *HealthCheckResult) error {
	address := fmt.Sprintf("%s:%d", session.Target.Address, session.Target.Port)
	
	timeout := hc.config.DefaultTimeout
	if session.Config != nil && session.Config.TimeoutPerCheck > 0 {
		timeout = session.Config.TimeoutPerCheck
	}

	// Create connection
	conn, err := grpc.Dial(address, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(timeout))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Create health client
	client := grpc_health_v1.NewHealthClient(conn)

	// Perform health check
	checkCtx, cancel := context.WithTimeout(hc.ctx, timeout)
	defer cancel()

	serviceName := ""
	if session.Target.GRPCService != "" {
		serviceName = session.Target.GRPCService
	}

	resp, err := client.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{
		Service: serviceName,
	})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	result.Metadata["grpc_service"] = serviceName
	result.Metadata["grpc_status"] = resp.Status.String()

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service not serving: %s", resp.Status)
	}

	return nil
}

// performTCPCheck performs TCP connectivity check
func (hc *HealthChecker) performTCPCheck(session *HealthCheckSession, result *HealthCheckResult) error {
	address := fmt.Sprintf("%s:%d", session.Target.Address, session.Target.Port)
	
	timeout := hc.config.DefaultTimeout
	if session.Config != nil && session.Config.TimeoutPerCheck > 0 {
		timeout = session.Config.TimeoutPerCheck
	}

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	defer conn.Close()

	result.Metadata["protocol"] = "tcp"
	result.Metadata["address"] = address

	return nil
}

// performUDPCheck performs UDP connectivity check
func (hc *HealthChecker) performUDPCheck(session *HealthCheckSession, result *HealthCheckResult) error {
	address := fmt.Sprintf("%s:%d", session.Target.Address, session.Target.Port)
	
	timeout := hc.config.DefaultTimeout
	if session.Config != nil && session.Config.TimeoutPerCheck > 0 {
		timeout = session.Config.TimeoutPerCheck
	}

	conn, err := net.DialTimeout("udp", address, timeout)
	if err != nil {
		return fmt.Errorf("UDP connection failed: %w", err)
	}
	defer conn.Close()

	result.Metadata["protocol"] = "udp"
	result.Metadata["address"] = address

	return nil
}

// extractTLSInfo extracts TLS information from connection state
func (hc *HealthChecker) extractTLSInfo(connState *tls.ConnectionState, tlsConfig *TLSCheckConfig) *TLSCheckInfo {
	info := &TLSCheckInfo{
		Version:     hc.tlsVersionString(connState.Version),
		CipherSuite: tls.CipherSuiteName(connState.CipherSuite),
		ChainLength: len(connState.PeerCertificates),
	}

	if len(connState.PeerCertificates) > 0 {
		cert := connState.PeerCertificates[0]
		info.CertificateValid = true
		info.CertificateExpiry = cert.NotAfter
		info.CertificateSubject = cert.Subject.String()
		info.SANs = cert.DNSNames
		
		// Check if certificate expires soon
		if tlsConfig != nil && tlsConfig.ExpiresSoon {
			threshold := tlsConfig.ExpiryThreshold
			if threshold == 0 {
				threshold = 30 * 24 * time.Hour // Default 30 days
			}
			
			if time.Until(cert.NotAfter) < threshold {
				info.ExpiresWithin = time.Until(cert.NotAfter)
			}
		}
	}

	return info
}

// validateTLSInfo validates TLS information against configuration
func (hc *HealthChecker) validateTLSInfo(tlsInfo *TLSCheckInfo, tlsConfig *TLSCheckConfig) error {
	if tlsConfig == nil {
		return nil
	}

	// Check certificate expiry
	if tlsConfig.ExpiresSoon && tlsInfo.ExpiresWithin > 0 {
		return fmt.Errorf("certificate expires in %v", tlsInfo.ExpiresWithin)
	}

	// Check allowed SANs
	if len(tlsConfig.AllowedSANs) > 0 {
		foundValidSAN := false
		for _, allowedSAN := range tlsConfig.AllowedSANs {
			for _, certSAN := range tlsInfo.SANs {
				if allowedSAN == certSAN {
					foundValidSAN = true
					break
				}
			}
			if foundValidSAN {
				break
			}
		}
		
		if !foundValidSAN {
			return fmt.Errorf("certificate SANs do not match allowed values")
		}
	}

	return nil
}

// determineCheckType determines the appropriate check type for a target
func (hc *HealthChecker) determineCheckType(target *HealthCheckTarget) HealthCheckType {
	// Check if gRPC service is specified
	if target.GRPCService != "" {
		return HealthCheckTypeGRPC
	}

	// Check common HTTPS ports
	if target.Port == 443 || target.Port == 8443 {
		return HealthCheckTypeHTTPS
	}

	// Check common HTTP ports
	if target.Port == 80 || target.Port == 8080 || target.Port == 3000 {
		return HealthCheckTypeHTTP
	}

	// If path is specified, assume HTTP
	if target.Path != "" {
		return HealthCheckTypeHTTP
	}

	// Default to TCP
	return HealthCheckTypeTCP
}

// determineHealthStatus determines the overall health status for a session
func (hc *HealthChecker) determineHealthStatus(session *HealthCheckSession) HealthCheckStatus {
	if session.TotalChecks == 0 {
		return HealthCheckStatusUnknown
	}

	successRate := hc.calculateSuccessRate(session)
	
	// Use thresholds from config or defaults
	healthyThreshold := hc.config.DefaultHealthyThreshold
	unhealthyThreshold := hc.config.DefaultUnhealthyThreshold
	
	if session.Config != nil {
		if session.Config.HealthyThreshold > 0 {
			healthyThreshold = session.Config.HealthyThreshold
		}
		if session.Config.UnhealthyThreshold > 0 {
			unhealthyThreshold = session.Config.UnhealthyThreshold
		}
	}

	// Simple logic: if recent checks are mostly successful, consider healthy
	recentChecks := 10
	if len(session.Results) < recentChecks {
		recentChecks = len(session.Results)
	}

	if recentChecks == 0 {
		return HealthCheckStatusUnknown
	}

	recentSuccess := 0
	for i := len(session.Results) - recentChecks; i < len(session.Results); i++ {
		if session.Results[i].Success {
			recentSuccess++
		}
	}

	recentSuccessRate := float64(recentSuccess) / float64(recentChecks)

	if recentSuccessRate >= 0.8 { // 80% success rate
		return HealthCheckStatusHealthy
	} else if recentSuccessRate >= 0.5 { // 50% success rate
		return HealthCheckStatusDegraded
	} else {
		return HealthCheckStatusUnhealthy
	}
}

// calculateSuccessRate calculates the success rate for a session
func (hc *HealthChecker) calculateSuccessRate(session *HealthCheckSession) float64 {
	if session.TotalChecks == 0 {
		return 0.0
	}
	return float64(session.SuccessCount) / float64(session.TotalChecks)
}

// GetHealthCheckSession gets a health check session
func (hc *HealthChecker) GetHealthCheckSession(sessionID string) (*HealthCheckSession, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	session, exists := hc.activeChecks[sessionID]
	return session, exists
}

// GetActiveHealthCheckSessions returns all active health check sessions
func (hc *HealthChecker) GetActiveHealthCheckSessions() []*HealthCheckSession {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	sessions := make([]*HealthCheckSession, 0, len(hc.activeChecks))
	for _, session := range hc.activeChecks {
		sessions = append(sessions, session)
	}
	
	return sessions
}

// Background processes

func (hc *HealthChecker) runMetricsCollector() {
	defer hc.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.collectMetrics()
		}
	}
}

func (hc *HealthChecker) collectMetrics() {
	hc.mu.RLock()
	activeCount := len(hc.activeChecks)
	hc.mu.RUnlock()
	
	hc.logger.Debug("health check metrics",
		"active_sessions", activeCount)
}

func (hc *HealthChecker) runSessionCleanup() {
	defer hc.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.cleanupSessions()
		}
	}
}

func (hc *HealthChecker) cleanupSessions() {
	cutoff := time.Now().Add(-1 * time.Hour) // Keep sessions for 1 hour
	
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	for sessionID, session := range hc.activeChecks {
		if session.LastCheckTime.Before(cutoff) {
			delete(hc.activeChecks, sessionID)
			hc.logger.Debug("cleaned up inactive health check session",
				"session_id", sessionID)
		}
	}
}

// Helper methods

func (hc *HealthChecker) tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}