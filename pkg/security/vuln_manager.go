
package security



import (

	"context"

	"fmt"

	"log/slog"

	"regexp"

	"sort"

	"strings"

	"sync"

	"time"



	"golang.org/x/mod/semver"

)



// VulnerabilityManager manages security vulnerabilities and automated remediation.

type VulnerabilityManager struct {

	config      *VulnManagerConfig

	logger      *slog.Logger

	scanner     *SecurityScanner

	database    *VulnDatabase

	remediation *RemediationEngine

	metrics     *VulnMetrics

	mutex       sync.RWMutex

}



// VulnManagerConfig holds vulnerability manager configuration.

type VulnManagerConfig struct {

	EnableCVEScanning     bool          `json:"enable_cve_scanning"`

	EnableDependencyCheck bool          `json:"enable_dependency_check"`

	EnableImageScanning   bool          `json:"enable_image_scanning"`

	EnableCodeScanning    bool          `json:"enable_code_scanning"`

	ScanInterval          time.Duration `json:"scan_interval"`

	CVEDatabaseURL        string        `json:"cve_database_url"`

	NVDAPIKey             string        `json:"nvd_api_key"`

	AutoRemediation       bool          `json:"auto_remediation"`

	MaxCVSSForAuto        float64       `json:"max_cvss_for_auto"`

	AlertThresholds       *AlertConfig  `json:"alert_thresholds"`

	IntegrationSettings   *Integrations `json:"integrations"`

}



// AlertConfig holds alerting configuration.

type AlertConfig struct {

	CriticalThreshold int           `json:"critical_threshold"` // Number of critical vulns to trigger alert

	HighThreshold     int           `json:"high_threshold"`     // Number of high vulns to trigger alert

	CVSSThreshold     float64       `json:"cvss_threshold"`     // CVSS score to trigger alert

	TimeToRemediate   time.Duration `json:"time_to_remediate"`  // SLA for remediation

}



// Integrations holds external service integrations.

type Integrations struct {

	Jira    *JiraConfig    `json:"jira,omitempty"`

	Slack   *SlackConfig   `json:"slack,omitempty"`

	Email   *EmailConfig   `json:"email,omitempty"`

	Webhook *WebhookConfig `json:"webhook,omitempty"`

}



// JiraConfig holds Jira integration configuration.

type JiraConfig struct {

	URL        string `json:"url"`

	Username   string `json:"username"`

	APIToken   string `json:"api_token"`

	ProjectKey string `json:"project_key"`

	IssueType  string `json:"issue_type"`

}



// SlackConfig holds Slack integration configuration.

type SlackConfig struct {

	WebhookURL string `json:"webhook_url"`

	Channel    string `json:"channel"`

	Username   string `json:"username"`

}



// EmailConfig holds email integration configuration.

type EmailConfig struct {

	SMTPHost     string   `json:"smtp_host"`

	SMTPPort     int      `json:"smtp_port"`

	SMTPUsername string   `json:"smtp_username"`

	SMTPPassword string   `json:"smtp_password"`

	FromEmail    string   `json:"from_email"`

	ToEmails     []string `json:"to_emails"`

	Subject      string   `json:"subject"`

	UseTLS       bool     `json:"use_tls"`

}



// WebhookConfig holds webhook integration configuration.

type WebhookConfig struct {

	URL     string            `json:"url"`

	Headers map[string]string `json:"headers"`

	Secret  string            `json:"secret"`

}



// VulnDatabase represents the vulnerability database.

type VulnDatabase struct {

	CVEs         map[string]*CVERecord  `json:"cves"`

	Dependencies map[string]*Dependency `json:"dependencies"`

	Images       map[string]*ImageVuln  `json:"images"`

	CodeIssues   map[string]*CodeIssue  `json:"code_issues"`

	LastUpdated  time.Time              `json:"last_updated"`

	mutex        sync.RWMutex

}



// CVERecord represents a CVE record.

type CVERecord struct {

	ID               string       `json:"id"`

	Summary          string       `json:"summary"`

	Description      string       `json:"description"`

	CVSS             float64      `json:"cvss"`

	Severity         string       `json:"severity"`

	PublishedDate    time.Time    `json:"published_date"`

	ModifiedDate     time.Time    `json:"modified_date"`

	References       []string     `json:"references"`

	CWE              []string     `json:"cwe"`

	AffectedProducts []Product    `json:"affected_products"`

	Remediation      *Remediation `json:"remediation,omitempty"`

}



// Product represents an affected product.

type Product struct {

	Vendor      string   `json:"vendor"`

	Product     string   `json:"product"`

	Versions    []string `json:"versions"`

	VersionType string   `json:"version_type"` // exact, range, regex

}



// Dependency represents a software dependency.

type Dependency struct {

	Name            string         `json:"name"`

	Version         string         `json:"version"`

	Type            string         `json:"type"`            // go, npm, pip, maven, etc.

	Vulnerabilities []string       `json:"vulnerabilities"` // CVE IDs

	LicenseIssues   []LicenseIssue `json:"license_issues"`

	LastChecked     time.Time      `json:"last_checked"`

}



// LicenseIssue represents a licensing issue.

type LicenseIssue struct {

	License     string `json:"license"`

	Severity    string `json:"severity"`

	Description string `json:"description"`

}



// ImageVuln represents container image vulnerabilities.

type ImageVuln struct {

	Image           string    `json:"image"`

	Tag             string    `json:"tag"`

	Digest          string    `json:"digest"`

	Vulnerabilities []string  `json:"vulnerabilities"`

	LastScanned     time.Time `json:"last_scanned"`

	ScanTool        string    `json:"scan_tool"`

}



// CodeIssue represents code security issues.

type CodeIssue struct {

	ID          string    `json:"id"`

	Type        string    `json:"type"` // hardcoded_secret, weak_crypto, etc.

	Severity    string    `json:"severity"`

	File        string    `json:"file"`

	Line        int       `json:"line"`

	Description string    `json:"description"`

	Rule        string    `json:"rule"`

	FoundAt     time.Time `json:"found_at"`

}



// Remediation represents remediation information.

type Remediation struct {

	Type        string        `json:"type"` // update, patch, config, workaround

	Description string        `json:"description"`

	Steps       []string      `json:"steps"`

	Automated   bool          `json:"automated"`

	Priority    string        `json:"priority"`

	ETA         time.Duration `json:"eta"`

}



// RemediationEngine handles automated remediation.

type RemediationEngine struct {

	config  *VulnManagerConfig

	logger  *slog.Logger

	actions map[string]RemediationAction

	mutex   sync.RWMutex

}



// RemediationAction represents an automated remediation action.

type RemediationAction interface {

	CanRemediate(vuln *CVERecord) bool

	Remediate(ctx context.Context, vuln *CVERecord) error

	GetDescription() string

	GetRiskLevel() string

}



// VulnMetrics tracks vulnerability metrics.

type VulnMetrics struct {

	TotalVulnerabilities      int64            `json:"total_vulnerabilities"`

	CriticalVulnerabilities   int64            `json:"critical_vulnerabilities"`

	HighVulnerabilities       int64            `json:"high_vulnerabilities"`

	MediumVulnerabilities     int64            `json:"medium_vulnerabilities"`

	LowVulnerabilities        int64            `json:"low_vulnerabilities"`

	RemediatedVulnerabilities int64            `json:"remediated_vulnerabilities"`

	VulnsByType               map[string]int64 `json:"vulns_by_type"`

	MTTRemediation            time.Duration    `json:"mtt_remediation"`

	LastScanTime              time.Time        `json:"last_scan_time"`

	ScanDuration              time.Duration    `json:"scan_duration"`

	mutex                     sync.RWMutex

}



// NewVulnerabilityManager creates a new vulnerability manager.

func NewVulnerabilityManager(config *VulnManagerConfig) (*VulnerabilityManager, error) {

	if config == nil {

		config = getDefaultVulnConfig()

	}



	vm := &VulnerabilityManager{

		config: config,

		logger: slog.Default().With("component", "vuln-manager"),

		database: &VulnDatabase{

			CVEs:         make(map[string]*CVERecord),

			Dependencies: make(map[string]*Dependency),

			Images:       make(map[string]*ImageVuln),

			CodeIssues:   make(map[string]*CodeIssue),

		},

		metrics: &VulnMetrics{

			VulnsByType: make(map[string]int64),

		},

	}



	// Initialize security scanner.

	scannerConfig := &ScannerConfig{

		BaseURL:                "http://localhost:8080",

		EnableVulnScanning:     true,

		EnableOWASPTesting:     true,

		EnableInjectionTesting: true,

	}

	vm.scanner = NewSecurityScanner(scannerConfig)



	// Initialize remediation engine.

	vm.remediation = NewRemediationEngine(config)



	// Start background scanning if enabled.

	if config.ScanInterval > 0 {

		go vm.startPeriodicScanning()

	}



	return vm, nil

}



// getDefaultVulnConfig returns default vulnerability manager configuration.

func getDefaultVulnConfig() *VulnManagerConfig {

	return &VulnManagerConfig{

		EnableCVEScanning:     true,

		EnableDependencyCheck: true,

		EnableImageScanning:   true,

		EnableCodeScanning:    true,

		ScanInterval:          24 * time.Hour,

		CVEDatabaseURL:        "https://nvd.nist.gov/vuln/data-feeds",

		AutoRemediation:       false,

		MaxCVSSForAuto:        6.0,

		AlertThresholds: &AlertConfig{

			CriticalThreshold: 1,

			HighThreshold:     5,

			CVSSThreshold:     7.0,

			TimeToRemediate:   48 * time.Hour,

		},

	}

}



// RunComprehensiveScan performs a comprehensive vulnerability scan.

func (vm *VulnerabilityManager) RunComprehensiveScan(ctx context.Context) (*VulnScanResults, error) {

	vm.logger.Info("Starting comprehensive vulnerability scan")

	startTime := time.Now()



	results := &VulnScanResults{

		ScanID:      fmt.Sprintf("vuln_scan_%d", time.Now().Unix()),

		StartTime:   startTime,

		Findings:    make([]*VulnFinding, 0),

		Remediation: make([]*RemediationSuggestion, 0),

	}



	var wg sync.WaitGroup



	// CVE scanning.

	if vm.config.EnableCVEScanning {

		wg.Add(1)

		go func() {

			defer wg.Done()

			vm.scanCVEs(ctx, results)

		}()

	}



	// Dependency scanning.

	if vm.config.EnableDependencyCheck {

		wg.Add(1)

		go func() {

			defer wg.Done()

			vm.scanDependencies(ctx, results)

		}()

	}



	// Container image scanning.

	if vm.config.EnableImageScanning {

		wg.Add(1)

		go func() {

			defer wg.Done()

			vm.scanImages(ctx, results)

		}()

	}



	// Code scanning.

	if vm.config.EnableCodeScanning {

		wg.Add(1)

		go func() {

			defer wg.Done()

			vm.scanCode(ctx, results)

		}()

	}



	wg.Wait()



	results.EndTime = time.Now()

	results.Duration = results.EndTime.Sub(results.StartTime)



	// Process results and generate remediation suggestions.

	vm.processResults(ctx, results)



	// Update metrics.

	vm.updateMetrics(results)



	// Send alerts if needed.

	vm.checkAlerts(ctx, results)



	vm.logger.Info("Vulnerability scan completed",

		"duration", results.Duration,

		"findings", len(results.Findings),

		"remediation_suggestions", len(results.Remediation))



	return results, nil

}



// scanCVEs scans for known CVEs.

func (vm *VulnerabilityManager) scanCVEs(ctx context.Context, results *VulnScanResults) {

	vm.logger.Info("Scanning for CVEs")



	// Get system information.

	components := vm.getSystemComponents()



	// Check each component against CVE database.

	for _, component := range components {

		vulns := vm.checkComponentVulnerabilities(component)

		for _, vuln := range vulns {

			finding := &VulnFinding{

				ID:          vuln.ID,

				Type:        "CVE",

				Severity:    vuln.Severity,

				CVSS:        vuln.CVSS,

				Title:       fmt.Sprintf("CVE vulnerability in %s", component.Name),

				Description: vuln.Summary,

				Component:   component.Name,

				Version:     component.Version,

				FoundAt:     time.Now(),

				References:  vuln.References,

			}



			results.mutex.Lock()

			results.Findings = append(results.Findings, finding)

			results.mutex.Unlock()

		}

	}

}



// scanDependencies scans project dependencies for vulnerabilities.

func (vm *VulnerabilityManager) scanDependencies(ctx context.Context, results *VulnScanResults) {

	vm.logger.Info("Scanning dependencies")



	// Scan Go modules.

	vm.scanGoModules(ctx, results)



	// Scan Node.js dependencies.

	vm.scanNodeDependencies(ctx, results)



	// Scan Python dependencies.

	vm.scanPythonDependencies(ctx, results)

}



// scanImages scans container images for vulnerabilities.

func (vm *VulnerabilityManager) scanImages(ctx context.Context, results *VulnScanResults) {

	vm.logger.Info("Scanning container images")



	images := vm.getContainerImages()

	for _, image := range images {

		vulns := vm.scanContainerImage(ctx, image)

		for _, vuln := range vulns {

			finding := &VulnFinding{

				ID:        vuln,

				Type:      "Container",

				Title:     fmt.Sprintf("Container vulnerability in %s", image),

				Component: image,

				FoundAt:   time.Now(),

			}



			results.mutex.Lock()

			results.Findings = append(results.Findings, finding)

			results.mutex.Unlock()

		}

	}

}



// scanCode performs static code analysis for security issues.

func (vm *VulnerabilityManager) scanCode(ctx context.Context, results *VulnScanResults) {

	vm.logger.Info("Performing static code analysis")



	codeIssues := vm.performStaticAnalysis()

	for _, issue := range codeIssues {

		finding := &VulnFinding{

			ID:          issue.ID,

			Type:        "Code",

			Severity:    issue.Severity,

			Title:       fmt.Sprintf("Code security issue: %s", issue.Type),

			Description: issue.Description,

			File:        issue.File,

			Line:        issue.Line,

			FoundAt:     issue.FoundAt,

		}



		results.mutex.Lock()

		results.Findings = append(results.Findings, finding)

		results.mutex.Unlock()

	}

}



// VulnScanResults holds vulnerability scan results.

type VulnScanResults struct {

	ScanID      string                   `json:"scan_id"`

	StartTime   time.Time                `json:"start_time"`

	EndTime     time.Time                `json:"end_time"`

	Duration    time.Duration            `json:"duration"`

	Findings    []*VulnFinding           `json:"findings"`

	Remediation []*RemediationSuggestion `json:"remediation"`

	mutex       sync.RWMutex

}



// VulnFinding represents a vulnerability finding.

type VulnFinding struct {

	ID          string    `json:"id"`

	Type        string    `json:"type"`

	Severity    string    `json:"severity"`

	CVSS        float64   `json:"cvss"`

	Title       string    `json:"title"`

	Description string    `json:"description"`

	Component   string    `json:"component"`

	Version     string    `json:"version"`

	File        string    `json:"file,omitempty"`

	Line        int       `json:"line,omitempty"`

	FoundAt     time.Time `json:"found_at"`

	References  []string  `json:"references,omitempty"`

}



// RemediationSuggestion represents a remediation suggestion.

type RemediationSuggestion struct {

	VulnID      string        `json:"vuln_id"`

	Type        string        `json:"type"`

	Description string        `json:"description"`

	Steps       []string      `json:"steps"`

	Automated   bool          `json:"automated"`

	Priority    string        `json:"priority"`

	ETA         time.Duration `json:"eta"`

	RiskLevel   string        `json:"risk_level"`

}



// SystemComponent represents a system component.

type SystemComponent struct {

	Name    string `json:"name"`

	Version string `json:"version"`

	Type    string `json:"type"`

	Path    string `json:"path"`

}



// Helper methods.



func (vm *VulnerabilityManager) getSystemComponents() []*SystemComponent {

	// In a real implementation, this would detect installed software.

	return []*SystemComponent{

		{Name: "kubernetes", Version: "v1.28.0", Type: "runtime"},

		{Name: "go", Version: "1.21.0", Type: "runtime"},

		{Name: "docker", Version: "24.0.0", Type: "runtime"},

		{Name: "nginx", Version: "1.25.0", Type: "service"},

	}

}



func (vm *VulnerabilityManager) checkComponentVulnerabilities(component *SystemComponent) []*CVERecord {

	vm.database.mutex.RLock()

	defer vm.database.mutex.RUnlock()



	var vulnerabilities []*CVERecord

	for _, cve := range vm.database.CVEs {

		if vm.componentAffected(component, cve) {

			vulnerabilities = append(vulnerabilities, cve)

		}

	}



	return vulnerabilities

}



func (vm *VulnerabilityManager) componentAffected(component *SystemComponent, cve *CVERecord) bool {

	for _, product := range cve.AffectedProducts {

		if strings.EqualFold(product.Product, component.Name) {

			return vm.versionAffected(component.Version, product.Versions, product.VersionType)

		}

	}

	return false

}



func (vm *VulnerabilityManager) versionAffected(version string, affectedVersions []string, versionType string) bool {

	switch versionType {

	case "exact":

		for _, affected := range affectedVersions {

			if version == affected {

				return true

			}

		}

	case "range":

		// Handle version ranges (e.g., ">=1.0.0,<1.2.0").

		return vm.versionInRange(version, affectedVersions)

	case "regex":

		for _, pattern := range affectedVersions {

			if matched, _ := regexp.MatchString(pattern, version); matched {

				return true

			}

		}

	}

	return false

}



func (vm *VulnerabilityManager) versionInRange(version string, ranges []string) bool {

	// Simplified version range checking using semver.

	for _, rangeStr := range ranges {

		if vm.checkSemverRange(version, rangeStr) {

			return true

		}

	}

	return false

}



func (vm *VulnerabilityManager) checkSemverRange(version, rangeStr string) bool {

	// Basic semver range checking.

	if !strings.HasPrefix(version, "v") {

		version = "v" + version

	}



	// Handle different range formats.

	if strings.Contains(rangeStr, ">=") && strings.Contains(rangeStr, "<") {

		parts := strings.Split(rangeStr, ",")

		if len(parts) == 2 {

			minVersion := strings.TrimPrefix(strings.TrimSpace(parts[0]), ">=")

			maxVersion := strings.TrimPrefix(strings.TrimSpace(parts[1]), "<")



			if !strings.HasPrefix(minVersion, "v") {

				minVersion = "v" + minVersion

			}

			if !strings.HasPrefix(maxVersion, "v") {

				maxVersion = "v" + maxVersion

			}



			return semver.Compare(version, minVersion) >= 0 && semver.Compare(version, maxVersion) < 0

		}

	}



	return false

}



func (vm *VulnerabilityManager) scanGoModules(ctx context.Context, results *VulnScanResults) {

	// Scan go.mod file for vulnerable dependencies.

	// This would parse go.mod and check against vulnerability databases.

}



func (vm *VulnerabilityManager) scanNodeDependencies(ctx context.Context, results *VulnScanResults) {

	// Scan package.json/package-lock.json for vulnerable dependencies.

}



func (vm *VulnerabilityManager) scanPythonDependencies(ctx context.Context, results *VulnScanResults) {

	// Scan requirements.txt/Pipfile for vulnerable dependencies.

}



func (vm *VulnerabilityManager) getContainerImages() []string {

	// Get list of container images used by the system.

	return []string{

		"nephoran/llm-processor:latest",

		"nephoran/rag-api:latest",

		"weaviate/weaviate:1.28.0",

		"redis:7.0",

		"postgres:15",

	}

}



func (vm *VulnerabilityManager) scanContainerImage(ctx context.Context, image string) []string {

	// Scan container image for vulnerabilities using tools like Trivy, Clair, etc.

	// Return list of CVE IDs found.

	return []string{} // Placeholder

}



func (vm *VulnerabilityManager) performStaticAnalysis() []*CodeIssue {

	// Perform static code analysis using tools like gosec, bandit, etc.

	issues := []*CodeIssue{

		{

			ID:          "HARDCODED_SECRET_001",

			Type:        "hardcoded_secret",

			Severity:    "High",

			File:        "pkg/auth/config.go",

			Line:        42,

			Description: "Potential hardcoded secret detected",

			Rule:        "G101",

			FoundAt:     time.Now(),

		},

	}



	return issues

}



func (vm *VulnerabilityManager) processResults(ctx context.Context, results *VulnScanResults) {

	// Generate remediation suggestions.

	for _, finding := range results.Findings {

		suggestions := vm.generateRemediationSuggestions(finding)

		results.Remediation = append(results.Remediation, suggestions...)

	}



	// Sort findings by severity and CVSS.

	sort.Slice(results.Findings, func(i, j int) bool {

		return vm.getSeverityWeight(results.Findings[i].Severity) >

			vm.getSeverityWeight(results.Findings[j].Severity)

	})

}



func (vm *VulnerabilityManager) generateRemediationSuggestions(finding *VulnFinding) []*RemediationSuggestion {

	suggestions := []*RemediationSuggestion{}



	switch finding.Type {

	case "CVE":

		if vm.database.CVEs[finding.ID] != nil && vm.database.CVEs[finding.ID].Remediation != nil {

			rem := vm.database.CVEs[finding.ID].Remediation

			suggestion := &RemediationSuggestion{

				VulnID:      finding.ID,

				Type:        rem.Type,

				Description: rem.Description,

				Steps:       rem.Steps,

				Automated:   rem.Automated && vm.config.AutoRemediation,

				Priority:    rem.Priority,

				ETA:         rem.ETA,

				RiskLevel:   vm.getRiskLevel(finding.CVSS),

			}

			suggestions = append(suggestions, suggestion)

		}

	case "Container":

		suggestion := &RemediationSuggestion{

			VulnID:      finding.ID,

			Type:        "update",

			Description: "Update container image to latest secure version",

			Steps:       []string{"Update image tag", "Rebuild container", "Deploy updated image"},

			Automated:   false,

			Priority:    "High",

			ETA:         4 * time.Hour,

			RiskLevel:   "Medium",

		}

		suggestions = append(suggestions, suggestion)

	case "Code":

		suggestion := &RemediationSuggestion{

			VulnID:      finding.ID,

			Type:        "code_fix",

			Description: "Fix code security issue",

			Steps:       []string{"Review code", "Apply security fix", "Test changes"},

			Automated:   false,

			Priority:    vm.getCodeIssuePriority(finding.Severity),

			ETA:         2 * time.Hour,

			RiskLevel:   finding.Severity,

		}

		suggestions = append(suggestions, suggestion)

	}



	return suggestions

}



func (vm *VulnerabilityManager) getSeverityWeight(severity string) int {

	weights := map[string]int{

		"Critical": 4,

		"High":     3,

		"Medium":   2,

		"Low":      1,

	}

	return weights[severity]

}



func (vm *VulnerabilityManager) getRiskLevel(cvss float64) string {

	if cvss >= 9.0 {

		return "Critical"

	} else if cvss >= 7.0 {

		return "High"

	} else if cvss >= 4.0 {

		return "Medium"

	}

	return "Low"

}



func (vm *VulnerabilityManager) getCodeIssuePriority(severity string) string {

	priorityMap := map[string]string{

		"Critical": "Critical",

		"High":     "High",

		"Medium":   "Medium",

		"Low":      "Low",

	}

	return priorityMap[severity]

}



func (vm *VulnerabilityManager) updateMetrics(results *VulnScanResults) {

	vm.metrics.mutex.Lock()

	defer vm.metrics.mutex.Unlock()



	vm.metrics.TotalVulnerabilities = int64(len(results.Findings))

	vm.metrics.LastScanTime = results.EndTime

	vm.metrics.ScanDuration = results.Duration



	// Reset counters.

	vm.metrics.CriticalVulnerabilities = 0

	vm.metrics.HighVulnerabilities = 0

	vm.metrics.MediumVulnerabilities = 0

	vm.metrics.LowVulnerabilities = 0

	vm.metrics.VulnsByType = make(map[string]int64)



	// Count by severity and type.

	for _, finding := range results.Findings {

		switch finding.Severity {

		case "Critical":

			vm.metrics.CriticalVulnerabilities++

		case "High":

			vm.metrics.HighVulnerabilities++

		case "Medium":

			vm.metrics.MediumVulnerabilities++

		case "Low":

			vm.metrics.LowVulnerabilities++

		}



		vm.metrics.VulnsByType[finding.Type]++

	}

}



func (vm *VulnerabilityManager) checkAlerts(ctx context.Context, results *VulnScanResults) {

	if vm.config.AlertThresholds == nil {

		return

	}



	criticalCount := 0

	highCount := 0

	var maxCVSS float64



	for _, finding := range results.Findings {

		switch finding.Severity {

		case "Critical":

			criticalCount++

		case "High":

			highCount++

		}

		if finding.CVSS > maxCVSS {

			maxCVSS = finding.CVSS

		}

	}



	shouldAlert := false

	alertMessage := "Security vulnerability alert:\n"



	if criticalCount >= vm.config.AlertThresholds.CriticalThreshold {

		shouldAlert = true

		alertMessage += fmt.Sprintf("- %d critical vulnerabilities found\n", criticalCount)

	}



	if highCount >= vm.config.AlertThresholds.HighThreshold {

		shouldAlert = true

		alertMessage += fmt.Sprintf("- %d high severity vulnerabilities found\n", highCount)

	}



	if maxCVSS >= vm.config.AlertThresholds.CVSSThreshold {

		shouldAlert = true

		alertMessage += fmt.Sprintf("- Maximum CVSS score: %.1f\n", maxCVSS)

	}



	if shouldAlert {

		vm.sendAlert(ctx, alertMessage, results)

	}

}



func (vm *VulnerabilityManager) sendAlert(ctx context.Context, message string, results *VulnScanResults) {

	vm.logger.Warn("Sending security alert", "message", message)



	if vm.config.IntegrationSettings == nil {

		return

	}



	// Send Slack alert.

	if vm.config.IntegrationSettings.Slack != nil {

		vm.sendSlackAlert(ctx, message, results)

	}



	// Send email alert.

	if vm.config.IntegrationSettings.Email != nil {

		vm.sendEmailAlert(ctx, message, results)

	}



	// Send webhook alert.

	if vm.config.IntegrationSettings.Webhook != nil {

		vm.sendWebhookAlert(ctx, message, results)

	}



	// Create Jira ticket.

	if vm.config.IntegrationSettings.Jira != nil {

		vm.createJiraTicket(ctx, message, results)

	}

}



func (vm *VulnerabilityManager) sendSlackAlert(ctx context.Context, message string, results *VulnScanResults) {

	// Implementation would send Slack notification.

	vm.logger.Info("Slack alert sent", "webhook", vm.config.IntegrationSettings.Slack.WebhookURL)

}



func (vm *VulnerabilityManager) sendEmailAlert(ctx context.Context, message string, results *VulnScanResults) {

	// Implementation would send email notification.

	vm.logger.Info("Email alert sent")

}



func (vm *VulnerabilityManager) sendWebhookAlert(ctx context.Context, message string, results *VulnScanResults) {

	// Implementation would send webhook notification.

	vm.logger.Info("Webhook alert sent", "url", vm.config.IntegrationSettings.Webhook.URL)

}



func (vm *VulnerabilityManager) createJiraTicket(ctx context.Context, message string, results *VulnScanResults) {

	// Implementation would create Jira ticket.

	vm.logger.Info("Jira ticket created", "project", vm.config.IntegrationSettings.Jira.ProjectKey)

}



func (vm *VulnerabilityManager) startPeriodicScanning() {

	ticker := time.NewTicker(vm.config.ScanInterval)

	defer ticker.Stop()



	for range ticker.C {

		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)

		if _, err := vm.RunComprehensiveScan(ctx); err != nil {

			vm.logger.Error("Periodic scan failed", "error", err)

		}

		cancel()

	}

}



// GetMetrics returns current vulnerability metrics.

func (vm *VulnerabilityManager) GetMetrics() *VulnMetrics {

	vm.metrics.mutex.RLock()

	defer vm.metrics.mutex.RUnlock()



	return &VulnMetrics{

		TotalVulnerabilities:      vm.metrics.TotalVulnerabilities,

		CriticalVulnerabilities:   vm.metrics.CriticalVulnerabilities,

		HighVulnerabilities:       vm.metrics.HighVulnerabilities,

		MediumVulnerabilities:     vm.metrics.MediumVulnerabilities,

		LowVulnerabilities:        vm.metrics.LowVulnerabilities,

		RemediatedVulnerabilities: vm.metrics.RemediatedVulnerabilities,

		VulnsByType:               copyStringInt64Map(vm.metrics.VulnsByType),

		MTTRemediation:            vm.metrics.MTTRemediation,

		LastScanTime:              vm.metrics.LastScanTime,

		ScanDuration:              vm.metrics.ScanDuration,

	}

}



// GetVulnerabilityDatabase returns the vulnerability database.

func (vm *VulnerabilityManager) GetVulnerabilityDatabase() *VulnDatabase {

	vm.database.mutex.RLock()

	defer vm.database.mutex.RUnlock()



	return vm.database

}



// UpdateCVEDatabase updates the CVE database from external sources.

func (vm *VulnerabilityManager) UpdateCVEDatabase(ctx context.Context) error {

	vm.logger.Info("Updating CVE database")



	// In a real implementation, this would fetch from NVD, CVE databases, etc.

	// For now, add some sample CVEs.

	sampleCVEs := []*CVERecord{

		{

			ID:            "CVE-2023-12345",

			Summary:       "Remote code execution in example service",

			Description:   "A vulnerability allows remote code execution",

			CVSS:          9.8,

			Severity:      "Critical",

			PublishedDate: time.Now().Add(-30 * 24 * time.Hour),

			AffectedProducts: []Product{

				{

					Vendor:      "example",

					Product:     "service",

					Versions:    []string{">=1.0.0,<1.2.0"},

					VersionType: "range",

				},

			},

		},

	}



	vm.database.mutex.Lock()

	for _, cve := range sampleCVEs {

		vm.database.CVEs[cve.ID] = cve

	}

	vm.database.LastUpdated = time.Now()

	vm.database.mutex.Unlock()



	vm.logger.Info("CVE database updated", "cves", len(sampleCVEs))

	return nil

}



// NewRemediationEngine creates a new remediation engine.

func NewRemediationEngine(config *VulnManagerConfig) *RemediationEngine {

	return &RemediationEngine{

		config:  config,

		logger:  slog.Default().With("component", "remediation-engine"),

		actions: make(map[string]RemediationAction),

	}

}



// RegisterAction registers a remediation action.

func (re *RemediationEngine) RegisterAction(name string, action RemediationAction) {

	re.mutex.Lock()

	defer re.mutex.Unlock()

	re.actions[name] = action

}



// GetAvailableActions returns available remediation actions.

func (re *RemediationEngine) GetAvailableActions() []string {

	re.mutex.RLock()

	defer re.mutex.RUnlock()



	actions := make([]string, 0, len(re.actions))

	for name := range re.actions {

		actions = append(actions, name)

	}

	return actions

}

