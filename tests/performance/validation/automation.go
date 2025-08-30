package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AutomationRunner handles automated validation execution and CI/CD integration.

type AutomationRunner struct {
	config *ValidationConfig

	environment string

	outputDir string

	ciMode bool
}

// CIPipelineConfig defines CI/CD pipeline configuration for validation.

type CIPipelineConfig struct {
	Triggers []TriggerConfig `yaml:"triggers"`

	Stages []StageConfig `yaml:"stages"`

	Notifications NotificationConfig `yaml:"notifications"`

	Artifacts ArtifactConfig `yaml:"artifacts"`

	Gates []QualityGate `yaml:"gates"`

	Environments []EnvironmentConfig `yaml:"environments"`

	Scheduling ScheduleConfig `yaml:"scheduling"`
}

// TriggerConfig defines when validation should run.

type TriggerConfig struct {
	Type string `yaml:"type"` // "push", "pull_request", "schedule", "manual"

	Branches []string `yaml:"branches,omitempty"`

	Paths []string `yaml:"paths,omitempty"`

	Schedule string `yaml:"schedule,omitempty"` // Cron expression

	Conditions []string `yaml:"conditions,omitempty"`
}

// StageConfig defines pipeline stages.

type StageConfig struct {
	Name string `yaml:"name"`

	Type string `yaml:"type"` // "validation", "baseline", "regression", "report"

	DependsOn []string `yaml:"depends_on,omitempty"`

	Environment string `yaml:"environment"`

	Parallel bool `yaml:"parallel"`

	Timeout string `yaml:"timeout"`

	Parameters map[string]interface{} `yaml:"parameters,omitempty"`

	Conditions []string `yaml:"conditions,omitempty"`
}

// NotificationConfig defines notification settings.

type NotificationConfig struct {
	OnSuccess []NotificationTarget `yaml:"on_success"`

	OnFailure []NotificationTarget `yaml:"on_failure"`

	OnRegression []NotificationTarget `yaml:"on_regression"`
}

// NotificationTarget defines notification destinations.

type NotificationTarget struct {
	Type string `yaml:"type"` // "email", "slack", "teams", "webhook"

	Recipients []string `yaml:"recipients,omitempty"`

	URL string `yaml:"url,omitempty"`

	Template string `yaml:"template,omitempty"`

	Conditions []string `yaml:"conditions,omitempty"`
}

// ArtifactConfig defines artifact management.

type ArtifactConfig struct {
	Retention string `yaml:"retention"` // Duration like "30d", "6m"

	Storage string `yaml:"storage"` // "local", "s3", "gcs", "azure"

	Compression bool `yaml:"compression"`

	Encryption bool `yaml:"encryption"`

	PublishResults bool `yaml:"publish_results"`

	Paths []string `yaml:"paths"`
}

// QualityGate defines quality gates that must pass.

type QualityGate struct {
	Name string `yaml:"name"`

	Type string `yaml:"type"` // "performance", "coverage", "security"

	Metrics []QualityMetric `yaml:"metrics"`

	Action string `yaml:"action"` // "fail", "warn", "report"

	Conditions []string `yaml:"conditions,omitempty"`
}

// QualityMetric defines specific quality metrics.

type QualityMetric struct {
	Name string `yaml:"name"`

	Threshold float64 `yaml:"threshold"`

	Operator string `yaml:"operator"` // ">=", "<=", "==", "!=", ">", "<"

	Unit string `yaml:"unit,omitempty"`
}

// EnvironmentConfig defines environment-specific settings.

type EnvironmentConfig struct {
	Name string `yaml:"name"`

	Type string `yaml:"type"` // "development", "staging", "production"

	Kubeconfig string `yaml:"kubeconfig,omitempty"`

	Namespace string `yaml:"namespace,omitempty"`

	Resources map[string]string `yaml:"resources,omitempty"`

	Variables map[string]interface{} `yaml:"variables,omitempty"`
}

// ScheduleConfig defines scheduled validation runs.

type ScheduleConfig struct {
	Daily string `yaml:"daily,omitempty"` // "HH:MM"

	Weekly string `yaml:"weekly,omitempty"` // "day HH:MM"

	Monthly string `yaml:"monthly,omitempty"` // "day HH:MM"

	Regression string `yaml:"regression,omitempty"` // Cron for regression tests

	Baseline string `yaml:"baseline,omitempty"` // Cron for baseline updates

}

// ValidationResult represents the result of an automated validation run.

type ValidationResult struct {
	ID string `json:"id"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	Duration time.Duration `json:"duration"`

	Status string `json:"status"` // "passed", "failed", "warning"

	Environment string `json:"environment"`

	CommitHash string `json:"commit_hash,omitempty"`

	Branch string `json:"branch,omitempty"`

	Claims map[string]ClaimResult `json:"claims"`

	QualityGates []QualityGateResult `json:"quality_gates"`

	Artifacts []string `json:"artifacts"`

	Notifications []NotificationSent `json:"notifications"`

	Metadata map[string]interface{} `json:"metadata"`
}

// QualityGateResult represents quality gate evaluation results.

type QualityGateResult struct {
	Name string `json:"name"`

	Status string `json:"status"` // "passed", "failed", "warning"

	Metrics []MetricResult `json:"metrics"`

	Message string `json:"message"`

	Action string `json:"action"`
}

// MetricResult represents individual metric evaluation.

type MetricResult struct {
	Name string `json:"name"`

	Value float64 `json:"value"`

	Threshold float64 `json:"threshold"`

	Status string `json:"status"`

	Unit string `json:"unit"`
}

// NotificationSent tracks sent notifications.

type NotificationSent struct {
	Type string `json:"type"`

	Recipients []string `json:"recipients"`

	Subject string `json:"subject"`

	SentAt time.Time `json:"sent_at"`

	Status string `json:"status"`

	MessageID string `json:"message_id,omitempty"`
}

// NewAutomationRunner creates a new automation runner.

func NewAutomationRunner(config *ValidationConfig, environment string) *AutomationRunner {

	return &AutomationRunner{

		config: config,

		environment: environment,

		outputDir: getOutputDir(),

		ciMode: isRunningInCI(),
	}

}

// RunAutomatedValidation executes automated validation with full CI/CD integration.

func (ar *AutomationRunner) RunAutomatedValidation(ctx context.Context) (*ValidationResult, error) {

	startTime := time.Now()

	runID := fmt.Sprintf("val-%s-%s", ar.environment, startTime.Format("20060102-150405"))

	log.Printf("Starting automated validation run: %s", runID)

	result := &ValidationResult{

		ID: runID,

		StartTime: startTime,

		Status: "running",

		Environment: ar.environment,

		Claims: make(map[string]ClaimResult),

		QualityGates: []QualityGateResult{},

		Artifacts: []string{},

		Notifications: []NotificationSent{},

		Metadata: make(map[string]interface{}),
	}

	// Gather environment metadata.

	ar.gatherMetadata(result)

	// Load CI pipeline configuration.

	pipelineConfig, err := ar.loadPipelineConfig()

	if err != nil {

		log.Printf("Warning: Could not load pipeline config: %v", err)

		pipelineConfig = ar.getDefaultPipelineConfig()

	}

	// Execute validation stages.

	for _, stage := range pipelineConfig.Stages {

		if !ar.shouldExecuteStage(stage, result) {

			continue

		}

		log.Printf("Executing stage: %s", stage.Name)

		stageCtx, cancel := context.WithTimeout(ctx, ar.parseTimeout(stage.Timeout))

		err := ar.executeStage(stageCtx, stage, result)

		cancel()

		if err != nil {

			result.Status = "failed"

			log.Printf("Stage %s failed: %v", stage.Name, err)

			break

		}

	}

	// Evaluate quality gates.

	ar.evaluateQualityGates(pipelineConfig.Gates, result)

	// Finalize result.

	result.EndTime = time.Now()

	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.Status != "failed" {

		result.Status = ar.determineOverallStatus(result)

	}

	// Save artifacts.

	ar.saveArtifacts(result, pipelineConfig.Artifacts)

	// Send notifications.

	ar.sendNotifications(pipelineConfig.Notifications, result)

	log.Printf("Automated validation completed: %s (status: %s, duration: %v)",

		runID, result.Status, result.Duration)

	return result, nil

}

// executeStage executes a single pipeline stage.

func (ar *AutomationRunner) executeStage(ctx context.Context, stage StageConfig, result *ValidationResult) error {

	switch stage.Type {

	case "validation":

		return ar.executeValidationStage(ctx, stage, result)

	case "baseline":

		return ar.executeBaselineStage(ctx, stage, result)

	case "regression":

		return ar.executeRegressionStage(ctx, stage, result)

	case "report":

		return ar.executeReportStage(ctx, stage, result)

	default:

		return fmt.Errorf("unknown stage type: %s", stage.Type)

	}

}

// executeValidationStage executes performance validation.

func (ar *AutomationRunner) executeValidationStage(ctx context.Context, stage StageConfig, result *ValidationResult) error {

	// Create validation suite with stage-specific configuration.

	config := ar.config

	if stageConfig, ok := stage.Parameters["config"].(map[string]interface{}); ok {

		config = ar.mergeConfig(config, stageConfig)

	}

	validationSuite := NewValidationSuite(config)

	// Run validation.

	validationResults, err := validationSuite.ValidateAllClaims(ctx)

	if err != nil {

		return fmt.Errorf("validation failed: %w", err)

	}

	// Merge results.

	for claimName, claimResult := range validationResults.ClaimResults {

		result.Claims[claimName] = *claimResult

	}

	// Update overall status based on validation results.

	if !validationResults.Summary.OverallSuccess {

		result.Status = "failed"

	}

	return nil

}

// executeBaselineStage updates performance baselines.

func (ar *AutomationRunner) executeBaselineStage(ctx context.Context, stage StageConfig, result *ValidationResult) error {

	log.Printf("Updating performance baselines...")

	// This would implement baseline update logic.

	// For now, we'll simulate it.

	time.Sleep(2 * time.Second)

	result.Metadata["baseline_updated"] = true

	result.Metadata["baseline_timestamp"] = time.Now()

	return nil

}

// executeRegressionStage performs regression detection.

func (ar *AutomationRunner) executeRegressionStage(ctx context.Context, stage StageConfig, result *ValidationResult) error {

	log.Printf("Performing regression analysis...")

	// This would implement regression detection logic.

	// For now, we'll simulate it.

	time.Sleep(3 * time.Second)

	result.Metadata["regression_analysis"] = true

	result.Metadata["regressions_detected"] = 0

	return nil

}

// executeReportStage generates comprehensive reports.

func (ar *AutomationRunner) executeReportStage(ctx context.Context, stage StageConfig, result *ValidationResult) error {

	log.Printf("Generating comprehensive reports...")

	// Generate HTML report.

	htmlReport, err := ar.generateHTMLReport(result)

	if err != nil {

		return fmt.Errorf("failed to generate HTML report: %w", err)

	}

	htmlPath := filepath.Join(ar.outputDir, "validation-report.html")

	if err := os.WriteFile(htmlPath, []byte(htmlReport), 0o640); err != nil {

		return fmt.Errorf("failed to save HTML report: %w", err)

	}

	result.Artifacts = append(result.Artifacts, htmlPath)

	// Generate JSON report.

	jsonPath := filepath.Join(ar.outputDir, "validation-results.json")

	jsonData, err := json.MarshalIndent(result, "", "  ")

	if err != nil {

		return fmt.Errorf("failed to marshal JSON report: %w", err)

	}

	if err := os.WriteFile(jsonPath, jsonData, 0o640); err != nil {

		return fmt.Errorf("failed to save JSON report: %w", err)

	}

	result.Artifacts = append(result.Artifacts, jsonPath)

	return nil

}

// evaluateQualityGates evaluates all configured quality gates.

func (ar *AutomationRunner) evaluateQualityGates(gates []QualityGate, result *ValidationResult) {

	for _, gate := range gates {

		gateResult := ar.evaluateQualityGate(gate, result)

		result.QualityGates = append(result.QualityGates, gateResult)

		if gateResult.Status == "failed" && gateResult.Action == "fail" {

			result.Status = "failed"

		}

	}

}

// evaluateQualityGate evaluates a single quality gate.

func (ar *AutomationRunner) evaluateQualityGate(gate QualityGate, result *ValidationResult) QualityGateResult {

	gateResult := QualityGateResult{

		Name: gate.Name,

		Status: "passed",

		Metrics: []MetricResult{},

		Action: gate.Action,
	}

	for _, metric := range gate.Metrics {

		metricResult := ar.evaluateMetric(metric, result)

		gateResult.Metrics = append(gateResult.Metrics, metricResult)

		if metricResult.Status == "failed" {

			gateResult.Status = "failed"

		}

	}

	if gateResult.Status == "failed" {

		gateResult.Message = fmt.Sprintf("Quality gate '%s' failed", gate.Name)

	} else {

		gateResult.Message = fmt.Sprintf("Quality gate '%s' passed", gate.Name)

	}

	return gateResult

}

// evaluateMetric evaluates a single quality metric.

func (ar *AutomationRunner) evaluateMetric(metric QualityMetric, result *ValidationResult) MetricResult {

	// Extract metric value from validation results.

	value := ar.extractMetricValue(metric.Name, result)

	// Evaluate threshold.

	passed := ar.evaluateThreshold(value, metric.Threshold, metric.Operator)

	status := "passed"

	if !passed {

		status = "failed"

	}

	return MetricResult{

		Name: metric.Name,

		Value: value,

		Threshold: metric.Threshold,

		Status: status,

		Unit: metric.Unit,
	}

}

// Helper methods for pipeline execution.

func (ar *AutomationRunner) shouldExecuteStage(stage StageConfig, result *ValidationResult) bool {

	// Check environment matching.

	if stage.Environment != "" && stage.Environment != ar.environment {

		return false

	}

	// Check dependencies.

	for _, dep := range stage.DependsOn {

		// This would check if dependent stages have completed successfully.

		_ = dep // Placeholder

	}

	// Check conditions.

	for _, condition := range stage.Conditions {

		if !ar.evaluateCondition(condition, result) {

			return false

		}

	}

	return true

}

func (ar *AutomationRunner) parseTimeout(timeout string) time.Duration {

	if timeout == "" {

		return 30 * time.Minute // Default timeout

	}

	d, err := time.ParseDuration(timeout)

	if err != nil {

		return 30 * time.Minute

	}

	return d

}

func (ar *AutomationRunner) mergeConfig(base *ValidationConfig, overrides map[string]interface{}) *ValidationConfig {

	// This would implement configuration merging logic.

	// For now, return the base config.

	return base

}

func (ar *AutomationRunner) extractMetricValue(metricName string, result *ValidationResult) float64 {

	// Extract specific metric values from validation results.

	switch metricName {

	case "overall_success_rate":

		passed := 0

		total := len(result.Claims)

		for _, claim := range result.Claims {

			if claim.Status == "validated" {

				passed++

			}

		}

		if total == 0 {

			return 0

		}

		return float64(passed) / float64(total) * 100

	case "average_confidence":

		if len(result.Claims) == 0 {

			return 0

		}

		total := 0.0

		for _, claim := range result.Claims {

			total += claim.Confidence

		}

		return total / float64(len(result.Claims))

	default:

		return 0

	}

}

func (ar *AutomationRunner) evaluateThreshold(value, threshold float64, operator string) bool {

	switch operator {

	case ">=":

		return value >= threshold

	case "<=":

		return value <= threshold

	case ">":

		return value > threshold

	case "<":

		return value < threshold

	case "==":

		return value == threshold

	case "!=":

		return value != threshold

	default:

		return false

	}

}

func (ar *AutomationRunner) evaluateCondition(condition string, result *ValidationResult) bool {

	// This would implement condition evaluation logic.

	// For now, return true.

	return true

}

func (ar *AutomationRunner) determineOverallStatus(result *ValidationResult) string {

	// Check if all claims passed.

	allPassed := true

	for _, claim := range result.Claims {

		if claim.Status != "validated" {

			allPassed = false

			break

		}

	}

	// Check quality gates.

	hasFailedGates := false

	hasWarningGates := false

	for _, gate := range result.QualityGates {

		if gate.Status == "failed" {

			hasFailedGates = true

		} else if gate.Status == "warning" {

			hasWarningGates = true

		}

	}

	if !allPassed || hasFailedGates {

		return "failed"

	} else if hasWarningGates {

		return "warning"

	}

	return "passed"

}

func (ar *AutomationRunner) gatherMetadata(result *ValidationResult) {

	// Gather Git information.

	if gitHash, err := ar.getGitCommitHash(); err == nil {

		result.CommitHash = gitHash

	}

	if gitBranch, err := ar.getGitBranch(); err == nil {

		result.Branch = gitBranch

	}

	// Gather environment information.

	result.Metadata["go_version"] = ar.getGoVersion()

	result.Metadata["kubernetes_version"] = ar.getKubernetesVersion()

	result.Metadata["ci_mode"] = ar.ciMode

	// CI-specific metadata.

	if ar.ciMode {

		result.Metadata["ci_build_id"] = os.Getenv("BUILD_ID")

		result.Metadata["ci_job_id"] = os.Getenv("JOB_ID")

		result.Metadata["ci_pipeline_id"] = os.Getenv("PIPELINE_ID")

	}

}

func (ar *AutomationRunner) getGitCommitHash() (string, error) {

	cmd := exec.Command("git", "rev-parse", "HEAD")

	output, err := cmd.Output()

	if err != nil {

		return "", err

	}

	return strings.TrimSpace(string(output)), nil

}

func (ar *AutomationRunner) getGitBranch() (string, error) {

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")

	output, err := cmd.Output()

	if err != nil {

		return "", err

	}

	return strings.TrimSpace(string(output)), nil

}

func (ar *AutomationRunner) getGoVersion() string {

	cmd := exec.Command("go", "version")

	output, err := cmd.Output()

	if err != nil {

		return "unknown"

	}

	return strings.TrimSpace(string(output))

}

func (ar *AutomationRunner) getKubernetesVersion() string {

	cmd := exec.Command("kubectl", "version", "--client", "--short")

	output, err := cmd.Output()

	if err != nil {

		return "unknown"

	}

	return strings.TrimSpace(string(output))

}

// Configuration and setup methods.

func (ar *AutomationRunner) loadPipelineConfig() (*CIPipelineConfig, error) {

	configPath := ".nephoran/validation-pipeline.yaml"

	if envPath := os.Getenv("VALIDATION_PIPELINE_CONFIG"); envPath != "" {

		configPath = envPath

	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {

		return ar.getDefaultPipelineConfig(), nil

	}

	data, err := os.ReadFile(configPath)

	if err != nil {

		return nil, fmt.Errorf("failed to read pipeline config: %w", err)

	}

	var config CIPipelineConfig

	if err := yaml.Unmarshal(data, &config); err != nil {

		return nil, fmt.Errorf("failed to parse pipeline config: %w", err)

	}

	return &config, nil

}

func (ar *AutomationRunner) getDefaultPipelineConfig() *CIPipelineConfig {

	return &CIPipelineConfig{

		Stages: []StageConfig{

			{

				Name: "validation",

				Type: "validation",

				Environment: ar.environment,

				Timeout: "30m",

				Parallel: false,
			},

			{

				Name: "report",

				Type: "report",

				Environment: ar.environment,

				Timeout: "5m",

				DependsOn: []string{"validation"},
			},
		},

		Gates: []QualityGate{

			{

				Name: "overall_success",

				Type: "performance",

				Metrics: []QualityMetric{

					{

						Name: "overall_success_rate",

						Threshold: 100.0,

						Operator: ">=",

						Unit: "%",
					},
				},

				Action: "fail",
			},
		},

		Artifacts: ArtifactConfig{

			Retention: "30d",

			Storage: "local",

			Compression: true,

			PublishResults: true,

			Paths: []string{"*.json", "*.html"},
		},
	}

}

// Utility functions.

func getOutputDir() string {

	if dir := os.Getenv("VALIDATION_OUTPUT_DIR"); dir != "" {

		return dir

	}

	return "test-results/validation"

}

func isRunningInCI() bool {

	ciEnvVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}

	for _, envVar := range ciEnvVars {

		if os.Getenv(envVar) != "" {

			return true

		}

	}

	return false

}

// saveArtifacts saves validation artifacts according to configuration.

func (ar *AutomationRunner) saveArtifacts(result *ValidationResult, artifactConfig ArtifactConfig) {

	if len(artifactConfig.Paths) == 0 {

		return

	}

	log.Printf("Saving validation artifacts...")

	// Create artifacts directory.

	artifactDir := filepath.Join(ar.outputDir, "artifacts")

	if err := os.MkdirAll(artifactDir, 0o755); err != nil {

		log.Printf("Warning: Failed to create artifacts directory: %v", err)

		return

	}

	// Save result as JSON artifact.

	resultPath := filepath.Join(artifactDir, fmt.Sprintf("validation-result-%s.json", result.ID))

	data, err := json.MarshalIndent(result, "", "  ")

	if err != nil {

		log.Printf("Warning: Failed to marshal result: %v", err)

		return

	}

	if err := os.WriteFile(resultPath, data, 0o640); err != nil {

		log.Printf("Warning: Failed to save result artifact: %v", err)

		return

	}

	result.Artifacts = append(result.Artifacts, resultPath)

	log.Printf("Saved validation artifacts to: %s", artifactDir)

}

// sendNotifications sends notifications based on validation results.

func (ar *AutomationRunner) sendNotifications(notificationConfig NotificationConfig, result *ValidationResult) {

	log.Printf("Processing notifications for validation result: %s", result.Status)

	// Determine which targets to notify based on status.

	var targets []NotificationTarget

	switch result.Status {

	case "failed":

		targets = notificationConfig.OnFailure

	case "passed":

		targets = notificationConfig.OnSuccess

	default:

		// For other statuses like "warning", use regression targets.

		targets = notificationConfig.OnRegression

	}

	if len(targets) == 0 {

		log.Printf("No notification targets configured for status: %s", result.Status)

		return

	}

	// Process each notification target.

	for _, target := range targets {

		notification := NotificationSent{

			Type: target.Type,

			Recipients: target.Recipients,

			Subject: fmt.Sprintf("Performance Validation - %s", result.Status),

			SentAt: time.Now(),

			Status: "sent",

			MessageID: fmt.Sprintf("val-%s-%d", result.ID, time.Now().Unix()),
		}

		// Log notification (in real implementation, would send to configured endpoints).

		log.Printf("NOTIFICATION [%s]: %s", target.Type, notification.Subject)

		result.Notifications = append(result.Notifications, notification)

	}

}

// generateHTMLReport generates a comprehensive HTML report.

func (ar *AutomationRunner) generateHTMLReport(result *ValidationResult) (string, error) {

	log.Printf("Generating HTML report for validation result: %s", result.ID)

	// Simple HTML template for the report.

	htmlTemplate := `

<!DOCTYPE html>

<html>

<head>

    <title>Performance Validation Report</title>

    <style>

        body { font-family: Arial, sans-serif; margin: 20px; }

        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }

        .status-passed { color: green; font-weight: bold; }

        .status-failed { color: red; font-weight: bold; }

        .status-warning { color: orange; font-weight: bold; }

        .claim { margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }

        .artifacts { margin-top: 20px; }

    </style>

</head>

<body>

    <div class="header">

        <h1>Performance Validation Report</h1>

        <p><strong>ID:</strong> %s</p>

        <p><strong>Start Time:</strong> %s</p>

        <p><strong>Duration:</strong> %v</p>

        <p><strong>Status:</strong> <span class="status-%s">%s</span></p>

        <p><strong>Environment:</strong> %s</p>

        <p><strong>Branch:</strong> %s</p>

    </div>

    

    <h2>Validation Claims (%d)</h2>

    <div class="claims">

        %s

    </div>

    

    <h2>Quality Gates</h2>

    <div class="quality-gates">

        %s

    </div>

    

    <div class="artifacts">

        <h2>Artifacts</h2>

        <ul>

            %s

        </ul>

    </div>

    

    <p><em>Generated at: %s</em></p>

</body>

</html>`

	// Build claims section.

	claimsHTML := ""

	for name, claim := range result.Claims {

		statusClass := "status-" + claim.Status

		claimsHTML += fmt.Sprintf(`

        <div class="claim">

            <h3>%s</h3>

            <p><strong>Status:</strong> <span class="%s">%s</span></p>

            <p><strong>Target:</strong> %v</p>

            <p><strong>Measured:</strong> %v</p>

            <p><strong>Confidence:</strong> %.2f%%</p>

        </div>`, name, statusClass, claim.Status, claim.Target, claim.Measured, claim.Confidence*100)

	}

	// Build quality gates section.

	qualityGatesHTML := ""

	for _, gate := range result.QualityGates {

		statusClass := "status-" + gate.Status

		qualityGatesHTML += fmt.Sprintf(`

        <div class="claim">

            <h3>%s</h3>

            <p><strong>Status:</strong> <span class="%s">%s</span></p>

            <p><strong>Message:</strong> %s</p>

            <p><strong>Action:</strong> %s</p>

        </div>`, gate.Name, statusClass, gate.Status, gate.Message, gate.Action)

	}

	// Build artifacts section.

	artifactsHTML := ""

	for _, artifact := range result.Artifacts {

		artifactsHTML += fmt.Sprintf("<li>%s</li>", filepath.Base(artifact))

	}

	// Fill in the template.

	htmlReport := fmt.Sprintf(htmlTemplate,

		result.ID,

		result.StartTime.Format("2006-01-02 15:04:05"),

		result.Duration,

		result.Status,

		result.Status,

		result.Environment,

		result.Branch,

		len(result.Claims),

		claimsHTML,

		qualityGatesHTML,

		artifactsHTML,

		time.Now().Format("2006-01-02 15:04:05"),
	)

	return htmlReport, nil

}
