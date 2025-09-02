// Package security implements container security scanning and RBAC enforcement
// for Nephoran Intent Operator with O-RAN WG11 compliance
package security

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// Container Security Metrics
	containerScansTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_container_scans_total",
			Help: "Total number of container scans performed",
		},
		[]string{"scanner", "status"},
	)

	vulnerabilitiesFoundTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_vulnerabilities_found_total",
			Help: "Total number of vulnerabilities found",
		},
		[]string{"severity", "scanner"},
	)

	rbacViolationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_rbac_violations_total",
			Help: "Total number of RBAC violations detected",
		},
		[]string{"violation_type", "namespace"},
	)

	securityPoliciesEnforced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_security_policies_enforced_total",
			Help: "Total number of security policies enforced",
		},
		[]string{"policy_type", "action"},
	)

	containerSecurityScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_container_security_score",
			Help: "Container security score (0-100)",
		},
		[]string{"image", "namespace"},
	)
)

// ContainerSecurityConfig contains container security configuration
type ContainerSecurityConfig struct {
	// Scanning configuration
	EnableContainerScanning bool          `json:"enable_container_scanning"`
	ScanningTools           []string      `json:"scanning_tools"` // trivy, clair, anchore
	ScanInterval            time.Duration `json:"scan_interval"`
	ScanTimeout             time.Duration `json:"scan_timeout"`
	MaxConcurrentScans      int           `json:"max_concurrent_scans"`

	// Vulnerability thresholds
	BlockCriticalVulns bool `json:"block_critical_vulns"`
	BlockHighVulns     bool `json:"block_high_vulns"`
	MaxCriticalVulns   int  `json:"max_critical_vulns"`
	MaxHighVulns       int  `json:"max_high_vulns"`
	MaxMediumVulns     int  `json:"max_medium_vulns"`

	// RBAC configuration
	EnableRBACEnforcement   bool     `json:"enable_rbac_enforcement"`
	StrictRBACMode          bool     `json:"strict_rbac_mode"`
	MinimumPrivileges       bool     `json:"minimum_privileges"`
	ForbiddenCapabilities   []string `json:"forbidden_capabilities"`
	RequiredSecurityContext bool     `json:"required_security_context"`

	// Policy enforcement
	EnablePolicyEnforcement bool     `json:"enable_policy_enforcement"`
	PolicyEngine            string   `json:"policy_engine"` // opa, gatekeeper, falco
	PolicySets              []string `json:"policy_sets"`
	EnforcementAction       string   `json:"enforcement_action"` // warn, block, audit

	// Image security
	TrustedRegistries   []string `json:"trusted_registries"`
	RequireSignedImages bool     `json:"require_signed_images"`
	AllowedBaseImages   []string `json:"allowed_base_images"`
	ForbiddenPackages   []string `json:"forbidden_packages"`

	// Runtime security
	EnableRuntimeMonitoring bool     `json:"enable_runtime_monitoring"`
	RuntimeSecurityTools    []string `json:"runtime_security_tools"` // falco, sysdig
	AnomalyDetection        bool     `json:"anomaly_detection"`

	// Compliance frameworks
	ComplianceFrameworks []string `json:"compliance_frameworks"` // pci, sox, hipaa
	AuditLogging         bool     `json:"audit_logging"`
	ComplianceReports    bool     `json:"compliance_reports"`
}

// ContainerSecurityManager manages container security and RBAC
type ContainerSecurityManager struct {
	config     *ContainerSecurityConfig
	logger     *slog.Logger
	kubeClient kubernetes.Interface

	// Scanning components
	scanners    map[string]ContainerScanner
	scanQueue   chan ScanRequest
	scanResults sync.Map // map[string]*ScanResult

	// RBAC enforcement
	rbacPolicies     []*RBACPolicy
	policyViolations []PolicyViolation

	// Security policies
	securityPolicies map[string]*SecurityPolicy
	policyEngine     PolicyEngine

	// Image verification
	imageVerifier  *ImageVerifier
	signatureCache sync.Map // map[string]*ImageSignature

	// Statistics
	stats *ContainerSecurityStats

	// Background tasks
	scanWorkers int
	shutdown    chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
}

// ContainerScanner defines the interface for container scanners
type ContainerScanner interface {
	ScanImage(ctx context.Context, image string) (*ScanResult, error)
	GetScannerInfo() ScannerInfo
	UpdateDatabase(ctx context.Context) error
	Health(ctx context.Context) error
}

// ScanRequest represents a container scan request
type ScanRequest struct {
	ID        string    `json:"id"`
	Image     string    `json:"image"`
	Namespace string    `json:"namespace"`
	Priority  int       `json:"priority"`
	Timestamp time.Time `json:"timestamp"`
	Retries   int       `json:"retries"`
}

// ScanResult represents the result of a container scan
type ScanResult struct {
	ID                string                   `json:"id"`
	Image             string                   `json:"image"`
	Scanner           string                   `json:"scanner"`
	ScanTime          time.Time                `json:"scan_time"`
	Duration          time.Duration            `json:"duration"`
	Status            string                   `json:"status"`
	SecurityScore     int                      `json:"security_score"`
	Vulnerabilities   []ContainerVulnerability `json:"vulnerabilities"`
	Misconfigurations []Misconfiguration       `json:"misconfigurations"`
	Secrets           []SecretLeak             `json:"secrets"`
	Compliance        ComplianceResult         `json:"compliance"`
	Metadata          map[string]interface{}   `json:"metadata"`
}

// ContainerVulnerability represents a security vulnerability in containers
type ContainerVulnerability struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Severity      string            `json:"severity"`
	CVSS          float64           `json:"cvss"`
	CVE           string            `json:"cve"`
	Package       string            `json:"package"`
	Version       string            `json:"version"`
	FixedVersion  string            `json:"fixed_version"`
	References    []string          `json:"references"`
	PrimaryURL    string            `json:"primary_url"`
	PublishedDate time.Time         `json:"published_date"`
	LastModified  time.Time         `json:"last_modified"`
	Exploitable   bool              `json:"exploitable"`
	InProduction  bool              `json:"in_production"`
	Metadata      map[string]string `json:"metadata"`
}

// Misconfiguration represents a security misconfiguration
type Misconfiguration struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Category    string            `json:"category"`
	Resource    string            `json:"resource"`
	Location    string            `json:"location"`
	Resolution  string            `json:"resolution"`
	References  []string          `json:"references"`
	Metadata    map[string]string `json:"metadata"`
}

// SecretLeak represents exposed secrets or credentials
type SecretLeak struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Confidence  string `json:"confidence"`
	Match       string `json:"match"`
}

// ComplianceResult represents compliance scan results
type ComplianceResult struct {
	Framework    string                 `json:"framework"`
	Version      string                 `json:"version"`
	Score        float64                `json:"score"`
	PassedChecks int                    `json:"passed_checks"`
	FailedChecks int                    `json:"failed_checks"`
	Results      []ComplianceCheck      `json:"results"`
	Summary      map[string]interface{} `json:"summary"`
}

// ComplianceCheck represents a compliance check
type ComplianceCheck struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Severity    string `json:"severity"`
	Rationale   string `json:"rationale"`
	Remediation string `json:"remediation"`
}

// ScannerInfo provides information about a scanner
type ScannerInfo struct {
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	DatabaseVersion string    `json:"database_version"`
	LastUpdated     time.Time `json:"last_updated"`
	Capabilities    []string  `json:"capabilities"`
}

// RBACPolicy represents an RBAC security policy
type RBACPolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Namespace   string                 `json:"namespace"`
	Rules       []RBACRule             `json:"rules"`
	Enforcement string                 `json:"enforcement"` // enforce, warn, audit
	Exceptions  []string               `json:"exceptions"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RBACRule represents an RBAC rule
type RBACRule struct {
	Subjects      []string `json:"subjects"`
	Resources     []string `json:"resources"`
	Verbs         []string `json:"verbs"`
	APIGroups     []string `json:"api_groups"`
	AllowedScopes []string `json:"allowed_scopes"`
	Conditions    []string `json:"conditions"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	ID          string                 `json:"id"`
	PolicyID    string                 `json:"policy_id"`
	Resource    string                 `json:"resource"`
	Namespace   string                 `json:"namespace"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	Action      string                 `json:"action"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityPolicy represents a security policy
type SecurityPolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Rules       []PolicyRule           `json:"rules"`
	Enforcement string                 `json:"enforcement"`
	Scope       []string               `json:"scope"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PolicyRule represents a policy rule
type PolicyRule struct {
	ID         string                 `json:"id"`
	Condition  string                 `json:"condition"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
}

// PolicyEngine defines the interface for policy engines
type PolicyEngine interface {
	EvaluatePolicy(ctx context.Context, policy *SecurityPolicy, resource interface{}) (bool, error)
	LoadPolicies(ctx context.Context, policies []*SecurityPolicy) error
	GetViolations(ctx context.Context) ([]PolicyViolation, error)
}

// ImageVerifier verifies image signatures and attestations
type ImageVerifier struct {
	publicKeys     map[string][]byte
	trustedIssuers []string
	mu             sync.RWMutex
}

// ImageSignature represents an image signature
type ImageSignature struct {
	Image     string            `json:"image"`
	Signature string            `json:"signature"`
	Issuer    string            `json:"issuer"`
	Valid     bool              `json:"valid"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

// ContainerSecurityStats tracks security statistics
type ContainerSecurityStats struct {
	TotalScans           int64     `json:"total_scans"`
	SuccessfulScans      int64     `json:"successful_scans"`
	FailedScans          int64     `json:"failed_scans"`
	CriticalVulns        int64     `json:"critical_vulns"`
	HighVulns            int64     `json:"high_vulns"`
	MediumVulns          int64     `json:"medium_vulns"`
	LowVulns             int64     `json:"low_vulns"`
	PolicyViolations     int64     `json:"policy_violations"`
	BlockedDeployments   int64     `json:"blocked_deployments"`
	LastScanTime         time.Time `json:"last_scan_time"`
	AverageSecurityScore float64   `json:"average_security_score"`
	ComplianceScore      float64   `json:"compliance_score"`
}

// NewContainerSecurityManager creates a new container security manager
func NewContainerSecurityManager(config *ContainerSecurityConfig, kubeClient kubernetes.Interface, logger *slog.Logger) (*ContainerSecurityManager, error) {
	if config == nil {
		config = DefaultContainerSecurityConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid container security config: %w", err)
	}

	csm := &ContainerSecurityManager{
		config:           config,
		logger:           logger.With(slog.String("component", "container_security")),
		kubeClient:       kubeClient,
		scanners:         make(map[string]ContainerScanner),
		scanQueue:        make(chan ScanRequest, 1000),
		rbacPolicies:     make([]*RBACPolicy, 0),
		policyViolations: make([]PolicyViolation, 0),
		securityPolicies: make(map[string]*SecurityPolicy),
		stats:            &ContainerSecurityStats{},
		scanWorkers:      config.MaxConcurrentScans,
		shutdown:         make(chan struct{}),
	}

	// Initialize scanners
	if err := csm.initializeScanners(); err != nil {
		return nil, fmt.Errorf("failed to initialize scanners: %w", err)
	}

	// Initialize image verifier
	if config.RequireSignedImages {
		csm.imageVerifier = &ImageVerifier{
			publicKeys:     make(map[string][]byte),
			trustedIssuers: make([]string, 0),
		}
	}

	// Initialize policy engine
	if config.EnablePolicyEnforcement {
		if err := csm.initializePolicyEngine(); err != nil {
			return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
		}
	}

	// Load RBAC policies
	if config.EnableRBACEnforcement {
		if err := csm.loadRBACPolicies(); err != nil {
			return nil, fmt.Errorf("failed to load RBAC policies: %w", err)
		}
	}

	// Start background workers
	csm.startScanWorkers()
	csm.startPeriodicTasks()

	logger.Info("Container security manager initialized",
		slog.Bool("scanning_enabled", config.EnableContainerScanning),
		slog.Bool("rbac_enabled", config.EnableRBACEnforcement),
		slog.Bool("policy_enabled", config.EnablePolicyEnforcement),
		slog.Int("scan_workers", csm.scanWorkers))

	return csm, nil
}

// initializeScanners initializes container scanners
func (csm *ContainerSecurityManager) initializeScanners() error {
	for _, tool := range csm.config.ScanningTools {
		switch tool {
		case "trivy":
			scanner, err := NewTrivyScanner(csm.logger)
			if err != nil {
				return fmt.Errorf("failed to initialize Trivy scanner: %w", err)
			}
			csm.scanners[tool] = scanner

		case "clair":
			scanner, err := NewClairScanner(csm.logger)
			if err != nil {
				return fmt.Errorf("failed to initialize Clair scanner: %w", err)
			}
			csm.scanners[tool] = scanner

		default:
			csm.logger.Warn("Unknown scanning tool", slog.String("tool", tool))
		}
	}

	if len(csm.scanners) == 0 {
		return fmt.Errorf("no scanners initialized")
	}

	return nil
}

// initializePolicyEngine initializes the policy engine
func (csm *ContainerSecurityManager) initializePolicyEngine() error {
	switch csm.config.PolicyEngine {
	case "opa":
		engine, err := NewOPAPolicyEngine(csm.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize OPA policy engine: %w", err)
		}
		csm.policyEngine = engine

	default:
		return fmt.Errorf("unsupported policy engine: %s", csm.config.PolicyEngine)
	}

	return nil
}

// loadRBACPolicies loads RBAC policies
func (csm *ContainerSecurityManager) loadRBACPolicies() error {
	// Load default RBAC policies
	policies := []*RBACPolicy{
		{
			ID:          "minimum-privileges",
			Name:        "Minimum Privileges",
			Description: "Enforce minimum privilege principle",
			Namespace:   "*",
			Rules: []RBACRule{
				{
					Subjects:  []string{"system:serviceaccounts"},
					Resources: []string{"secrets", "configmaps"},
					Verbs:     []string{"get", "list"},
					APIGroups: []string{""},
				},
			},
			Enforcement: "enforce",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "no-cluster-admin",
			Name:        "No Cluster Admin",
			Description: "Prevent cluster admin access",
			Namespace:   "*",
			Rules: []RBACRule{
				{
					Subjects:  []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
				},
			},
			Enforcement: "warn",
			CreatedAt:   time.Now(),
		},
	}

	csm.rbacPolicies = policies
	return nil
}

// ScanImage scans a container image for vulnerabilities
func (csm *ContainerSecurityManager) ScanImage(ctx context.Context, image, namespace string) (*ScanResult, error) {
	if !csm.config.EnableContainerScanning {
		return nil, fmt.Errorf("container scanning is disabled")
	}

	// Check if image is from trusted registry
	if !csm.isTrustedImage(image) {
		return nil, fmt.Errorf("image from untrusted registry: %s", image)
	}

	// Verify image signature if required
	if csm.config.RequireSignedImages {
		if err := csm.verifyImageSignature(ctx, image); err != nil {
			return nil, fmt.Errorf("image signature verification failed: %w", err)
		}
	}

	// Create scan request
	scanReq := ScanRequest{
		ID:        generateScanID(),
		Image:     image,
		Namespace: namespace,
		Priority:  1,
		Timestamp: time.Now(),
	}

	// Queue scan request
	select {
	case csm.scanQueue <- scanReq:
	default:
		return nil, fmt.Errorf("scan queue is full")
	}

	// Wait for scan result (simplified - in production, use proper synchronization)
	time.Sleep(100 * time.Millisecond)
	if result, exists := csm.scanResults.Load(scanReq.ID); exists {
		return result.(*ScanResult), nil
	}

	return nil, fmt.Errorf("scan timeout")
}

// EvaluateRBAC evaluates RBAC policies for a resource
func (csm *ContainerSecurityManager) EvaluateRBAC(ctx context.Context, resource interface{}) ([]PolicyViolation, error) {
	if !csm.config.EnableRBACEnforcement {
		return nil, nil
	}

	violations := make([]PolicyViolation, 0)

	for _, policy := range csm.rbacPolicies {
		violation := csm.checkRBACPolicy(policy, resource)
		if violation != nil {
			violations = append(violations, *violation)
			rbacViolationsTotal.WithLabelValues(violation.PolicyID, violation.Namespace).Inc()

			if policy.Enforcement == "enforce" {
				csm.stats.BlockedDeployments++
			}
		}
	}

	return violations, nil
}

// checkRBACPolicy checks a single RBAC policy
func (csm *ContainerSecurityManager) checkRBACPolicy(policy *RBACPolicy, resource interface{}) *PolicyViolation {
	// Simplified RBAC policy checking
	// In production, this would be more comprehensive

	if pod, ok := resource.(*corev1.Pod); ok {
		// Check security context
		if csm.config.RequiredSecurityContext {
			if pod.Spec.SecurityContext == nil {
				return &PolicyViolation{
					ID:          generateViolationID(),
					PolicyID:    policy.ID,
					Resource:    pod.Name,
					Namespace:   pod.Namespace,
					Severity:    "medium",
					Description: "Pod missing security context",
					Timestamp:   time.Now(),
					Action:      policy.Enforcement,
				}
			}
		}

		// Check capabilities
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil {
				for _, capability := range container.SecurityContext.Capabilities.Add {
					capStr := string(capability)
					for _, forbidden := range csm.config.ForbiddenCapabilities {
						if capStr == forbidden {
							return &PolicyViolation{
								ID:          generateViolationID(),
								PolicyID:    policy.ID,
								Resource:    pod.Name,
								Namespace:   pod.Namespace,
								Severity:    "high",
								Description: fmt.Sprintf("Forbidden capability: %s", capStr),
								Timestamp:   time.Now(),
								Action:      policy.Enforcement,
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// isTrustedImage checks if an image is from a trusted registry
func (csm *ContainerSecurityManager) isTrustedImage(image string) bool {
	if len(csm.config.TrustedRegistries) == 0 {
		return true // Allow all if no restrictions
	}

	for _, registry := range csm.config.TrustedRegistries {
		if strings.HasPrefix(image, registry) {
			return true
		}
	}

	return false
}

// verifyImageSignature verifies image signature
func (csm *ContainerSecurityManager) verifyImageSignature(ctx context.Context, image string) error {
	if csm.imageVerifier == nil {
		return nil
	}

	// Check cache first
	if sig, exists := csm.signatureCache.Load(image); exists {
		signature := sig.(*ImageSignature)
		if signature.Valid && time.Since(signature.Timestamp) < 1*time.Hour {
			return nil
		}
	}

	// Verify signature (simplified implementation)
	// In production, use cosign or similar tools
	signature := &ImageSignature{
		Image:     image,
		Valid:     true, // Placeholder
		Timestamp: time.Now(),
	}

	csm.signatureCache.Store(image, signature)
	return nil
}

// startScanWorkers starts background scan workers
func (csm *ContainerSecurityManager) startScanWorkers() {
	for i := 0; i < csm.scanWorkers; i++ {
		csm.wg.Add(1)
		go csm.scanWorker()
	}
}

// scanWorker processes scan requests
func (csm *ContainerSecurityManager) scanWorker() {
	defer csm.wg.Done()

	for {
		select {
		case scanReq := <-csm.scanQueue:
			csm.processScanRequest(scanReq)

		case <-csm.shutdown:
			return
		}
	}
}

// processScanRequest processes a single scan request
func (csm *ContainerSecurityManager) processScanRequest(req ScanRequest) {
	start := time.Now()
	csm.stats.TotalScans++

	ctx, cancel := context.WithTimeout(context.Background(), csm.config.ScanTimeout)
	defer cancel()

	// Use the first available scanner
	for name, scanner := range csm.scanners {
		result, err := scanner.ScanImage(ctx, req.Image)
		if err != nil {
			csm.logger.Error("Scan failed",
				slog.String("scanner", name),
				slog.String("image", req.Image),
				slog.String("error", err.Error()))
			csm.stats.FailedScans++
			containerScansTotal.WithLabelValues(name, "failed").Inc()
			continue
		}

		result.ID = req.ID
		result.Duration = time.Since(start)
		result.SecurityScore = csm.calculateSecurityScore(result)

		// Store result
		csm.scanResults.Store(req.ID, result)
		csm.stats.SuccessfulScans++
		csm.stats.LastScanTime = time.Now()

		// Update vulnerability counts
		for _, vuln := range result.Vulnerabilities {
			switch vuln.Severity {
			case "critical":
				csm.stats.CriticalVulns++
				vulnerabilitiesFoundTotal.WithLabelValues("critical", name).Inc()
			case "high":
				csm.stats.HighVulns++
				vulnerabilitiesFoundTotal.WithLabelValues("high", name).Inc()
			case "medium":
				csm.stats.MediumVulns++
				vulnerabilitiesFoundTotal.WithLabelValues("medium", name).Inc()
			case "low":
				csm.stats.LowVulns++
				vulnerabilitiesFoundTotal.WithLabelValues("low", name).Inc()
			}
		}

		// Update metrics
		containerScansTotal.WithLabelValues(name, "success").Inc()
		containerSecurityScore.WithLabelValues(req.Image, req.Namespace).Set(float64(result.SecurityScore))

		csm.logger.Info("Container scan completed",
			slog.String("image", req.Image),
			slog.String("scanner", name),
			slog.Int("security_score", result.SecurityScore),
			slog.Int("vulnerabilities", len(result.Vulnerabilities)),
			slog.Duration("duration", result.Duration))

		return
	}

	csm.logger.Error("All scanners failed for image", slog.String("image", req.Image))
}

// calculateSecurityScore calculates a security score based on scan results
func (csm *ContainerSecurityManager) calculateSecurityScore(result *ScanResult) int {
	score := 100 // Start with perfect score

	// Deduct points for vulnerabilities
	for _, vuln := range result.Vulnerabilities {
		switch vuln.Severity {
		case "critical":
			score -= 25
		case "high":
			score -= 15
		case "medium":
			score -= 5
		case "low":
			score -= 1
		}
	}

	// Deduct points for misconfigurations
	for _, misc := range result.Misconfigurations {
		switch misc.Severity {
		case "critical":
			score -= 20
		case "high":
			score -= 10
		case "medium":
			score -= 3
		case "low":
			score -= 1
		}
	}

	// Deduct points for secrets
	score -= len(result.Secrets) * 10

	if score < 0 {
		score = 0
	}

	return score
}

// startPeriodicTasks starts periodic maintenance tasks
func (csm *ContainerSecurityManager) startPeriodicTasks() {
	ticker := time.NewTicker(csm.config.ScanInterval)

	csm.wg.Add(1)
	go func() {
		defer csm.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				csm.performPeriodicTasks()
			case <-csm.shutdown:
				return
			}
		}
	}()
}

// performPeriodicTasks performs periodic maintenance
func (csm *ContainerSecurityManager) performPeriodicTasks() {
	// Update scanner databases
	for name, scanner := range csm.scanners {
		if err := scanner.UpdateDatabase(context.Background()); err != nil {
			csm.logger.Warn("Failed to update scanner database",
				slog.String("scanner", name),
				slog.String("error", err.Error()))
		}
	}

	// Clean up old scan results
	csm.cleanupOldResults()

	// Generate compliance reports
	if csm.config.ComplianceReports {
		csm.generateComplianceReport()
	}

	// Update statistics
	csm.updateStats()
}

// cleanupOldResults cleans up old scan results
func (csm *ContainerSecurityManager) cleanupOldResults() {
	cutoff := time.Now().Add(-24 * time.Hour)

	csm.scanResults.Range(func(key, value interface{}) bool {
		result := value.(*ScanResult)
		if result.ScanTime.Before(cutoff) {
			csm.scanResults.Delete(key)
		}
		return true
	})
}

// generateComplianceReport generates compliance reports
func (csm *ContainerSecurityManager) generateComplianceReport() {
	// Generate compliance reports for configured frameworks
	for _, framework := range csm.config.ComplianceFrameworks {
		csm.logger.Info("Generating compliance report", slog.String("framework", framework))
		// Implementation would generate actual compliance reports
	}
}

// updateStats updates internal statistics
func (csm *ContainerSecurityManager) updateStats() {
	// Calculate average security score
	totalScore := 0
	scanCount := 0

	csm.scanResults.Range(func(key, value interface{}) bool {
		result := value.(*ScanResult)
		totalScore += result.SecurityScore
		scanCount++
		return true
	})

	if scanCount > 0 {
		csm.stats.AverageSecurityScore = float64(totalScore) / float64(scanCount)
	}

	// Update compliance score (simplified)
	csm.stats.ComplianceScore = csm.stats.AverageSecurityScore
}

// GetStats returns container security statistics
func (csm *ContainerSecurityManager) GetStats() *ContainerSecurityStats {
	csm.mu.RLock()
	defer csm.mu.RUnlock()
	return csm.stats
}

// Close shuts down the container security manager
func (csm *ContainerSecurityManager) Close() error {
	close(csm.shutdown)
	csm.wg.Wait()

	// Close scanners
	for name, scanner := range csm.scanners {
		if closer, ok := scanner.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				csm.logger.Warn("Failed to close scanner",
					slog.String("scanner", name),
					slog.String("error", err.Error()))
			}
		}
	}

	csm.logger.Info("Container security manager shut down")
	return nil
}

// Helper functions

func generateScanID() string {
	return fmt.Sprintf("scan-%d", time.Now().UnixNano())
}

func generateViolationID() string {
	return fmt.Sprintf("violation-%d", time.Now().UnixNano())
}

// DefaultContainerSecurityConfig returns default configuration
func DefaultContainerSecurityConfig() *ContainerSecurityConfig {
	return &ContainerSecurityConfig{
		EnableContainerScanning: true,
		ScanningTools:           []string{"trivy"},
		ScanInterval:            6 * time.Hour,
		ScanTimeout:             10 * time.Minute,
		MaxConcurrentScans:      5,
		BlockCriticalVulns:      true,
		BlockHighVulns:          false,
		MaxCriticalVulns:        0,
		MaxHighVulns:            5,
		MaxMediumVulns:          20,
		EnableRBACEnforcement:   true,
		StrictRBACMode:          false,
		MinimumPrivileges:       true,
		ForbiddenCapabilities:   []string{"SYS_ADMIN", "NET_ADMIN", "DAC_OVERRIDE"},
		RequiredSecurityContext: true,
		EnablePolicyEnforcement: true,
		PolicyEngine:            "opa",
		EnforcementAction:       "warn",
		RequireSignedImages:     false,
		EnableRuntimeMonitoring: true,
		AnomalyDetection:        true,
		ComplianceFrameworks:    []string{"pci"},
		AuditLogging:            true,
		ComplianceReports:       true,
	}
}

// Validate validates the container security configuration
func (config *ContainerSecurityConfig) Validate() error {
	if config.MaxConcurrentScans <= 0 {
		return fmt.Errorf("max concurrent scans must be positive")
	}
	if config.ScanTimeout <= 0 {
		return fmt.Errorf("scan timeout must be positive")
	}
	return nil
}

// Stub implementations for external scanner constructors
// TODO: Implement proper Trivy and Clair scanner integrations

func NewTrivyScanner(logger interface{}) (ContainerScanner, error) {
	// Stub implementation - replace with actual Trivy integration
	return nil, fmt.Errorf("trivy scanner not implemented yet")
}

func NewClairScanner(logger interface{}) (ContainerScanner, error) {
	// Stub implementation - replace with actual Clair integration
	return nil, fmt.Errorf("clair scanner not implemented yet")
}

// OPAPolicyEngine implements PolicyEngine interface using Open Policy Agent
type OPAPolicyEngine struct {
	logger     *slog.Logger
	policies   map[string]*SecurityPolicy
	violations []PolicyViolation
	mu         sync.RWMutex
}

// NewOPAPolicyEngine creates a new OPA-based policy engine
func NewOPAPolicyEngine(logger *slog.Logger) (PolicyEngine, error) {
	return &OPAPolicyEngine{
		logger:     logger,
		policies:   make(map[string]*SecurityPolicy),
		violations: make([]PolicyViolation, 0),
	}, nil
}

// EvaluatePolicy evaluates a policy against a resource
func (o *OPAPolicyEngine) EvaluatePolicy(ctx context.Context, policy *SecurityPolicy, resource interface{}) (bool, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Simple policy evaluation - this would normally use OPA's Rego engine
	o.logger.Info("Evaluating policy", "policy", policy.ID, "resource", fmt.Sprintf("%T", resource))

	// For now, return true (allow) for all policies
	// TODO: Implement actual OPA integration
	return true, nil
}

// LoadPolicies loads security policies into the engine
func (o *OPAPolicyEngine) LoadPolicies(ctx context.Context, policies []*SecurityPolicy) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, policy := range policies {
		o.policies[policy.ID] = policy
		o.logger.Info("Loaded policy", "id", policy.ID, "name", policy.Name)
	}

	return nil
}

// GetViolations returns current policy violations
func (o *OPAPolicyEngine) GetViolations(ctx context.Context) ([]PolicyViolation, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Return a copy to avoid race conditions
	violations := make([]PolicyViolation, len(o.violations))
	copy(violations, o.violations)

	return violations, nil
}
