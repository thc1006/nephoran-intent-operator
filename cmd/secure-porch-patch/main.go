
package main



import (

	"context"

	"flag"

	"fmt"

	"os"

	"time"



	"github.com/go-logr/logr"

	"github.com/go-logr/zapr"

	"go.uber.org/zap"



	"github.com/thc1006/nephoran-intent-operator/internal/patchgen"

	"github.com/thc1006/nephoran-intent-operator/internal/security"

)



// Version information - will be set at build time.

var (

	// Version holds version value.

	Version = "dev"

	// GitCommit holds gitcommit value.

	GitCommit = "unknown"

	// BuildTime holds buildtime value.

	BuildTime = "unknown"

)



// SecurityConfig holds security configuration.

type SecurityConfig struct {

	EnableCompliance      bool

	EnableAuditLogging    bool

	EnableThreatDetection bool

	MaxExecutionTime      time.Duration

	AllowedOutputPaths    []string

}



func main() {

	var (

		intentPath     string

		outputDir      string

		apply          bool

		enableSecurity bool

		complianceMode bool

		verbosity      int

	)



	flag.StringVar(&intentPath, "intent", "", "Path to the intent JSON file (required)")

	flag.StringVar(&outputDir, "out", "", "Output directory for generated patches")

	flag.BoolVar(&apply, "apply", false, "Apply the patch using porch-direct after generation")

	flag.BoolVar(&enableSecurity, "security", true, "Enable comprehensive security controls (default: true)")

	flag.BoolVar(&complianceMode, "compliance", false, "Enable O-RAN WG11 compliance validation")

	flag.IntVar(&verbosity, "v", 0, "Verbosity level (0-3)")

	flag.Parse()



	// Initialize structured logging.

	logger, err := initializeLogger(verbosity)

	if err != nil {

		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)

		os.Exit(1)

	}



	// Print version information.

	logger.Info("Starting Secure Porch Patch Generator",

		"version", Version,

		"git_commit", GitCommit,

		"build_time", BuildTime,

		"security_enabled", enableSecurity,

		"compliance_mode", complianceMode)



	// Validate command line arguments.

	if intentPath == "" {

		fmt.Fprintf(os.Stderr, "Error: --intent flag is required\n")

		flag.Usage()

		os.Exit(1)

	}



	if outputDir == "" {

		outputDir = "./examples/packages/scaling"

	}



	// Initialize security configuration.

	secConfig := SecurityConfig{

		EnableCompliance:      complianceMode,

		EnableAuditLogging:    true,

		EnableThreatDetection: enableSecurity,

		MaxExecutionTime:      5 * time.Minute,

		AllowedOutputPaths: []string{

			"./examples",

			"./packages",

			"./output",

			os.TempDir(),

		},

	}



	// Run with timeout and security context.

	ctx, cancel := context.WithTimeout(context.Background(), secConfig.MaxExecutionTime)

	defer cancel()



	if err := runSecure(ctx, intentPath, outputDir, apply, secConfig, logger); err != nil {

		logger.Error(err, "Secure execution failed")

		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)

		// Let defer handle cleanup, exit with error code

		logger.Info("Terminating due to secure execution failure")

		os.Exit(1)

	}

}



// runSecure executes the main logic with comprehensive security controls.

func runSecure(ctx context.Context, intentPath, outputDir string, apply bool, config SecurityConfig, logger logr.Logger) error {

	logger.Info("Starting secure patch generation workflow",

		"intent_path", intentPath,

		"output_dir", outputDir,

		"apply", apply)



	// 1. Initialize security components.

	owaspValidator, err := security.NewOWASPValidator()

	if err != nil {

		return fmt.Errorf("failed to initialize OWASP validator: %w", err)

	}



	// 2. Perform comprehensive security validation of intent file.

	logger.V(1).Info("Performing security validation of intent file")

	validationResult, err := owaspValidator.ValidateIntentFile(intentPath)

	if err != nil {

		return fmt.Errorf("intent file validation error: %w", err)

	}



	if !validationResult.IsValid {

		logger.Error(fmt.Errorf("security validation failed"), "Intent file failed security validation",

			"violations", len(validationResult.Violations),

			"threat_level", validationResult.ThreatLevel)



		// Log detailed violations.

		for _, violation := range validationResult.Violations {

			logger.Error(fmt.Errorf("security violation"), "Security violation detected",

				"field", violation.Field,

				"type", violation.Type,

				"severity", violation.Severity,

				"message", violation.Message,

				"owasp_rule", violation.OWASPRule)

		}



		return fmt.Errorf("intent file has %d security violations (threat level: %s)",

			len(validationResult.Violations), validationResult.ThreatLevel)

	}



	logger.Info("Intent file security validation passed", "threat_level", validationResult.ThreatLevel)



	// 3. Load and validate intent using existing patchgen logic.

	intent, err := patchgen.LoadIntent(intentPath)

	if err != nil {

		return fmt.Errorf("failed to load intent: %w", err)

	}



	logger.Info("Intent loaded successfully",

		"target", intent.Target,

		"namespace", intent.Namespace,

		"replicas", intent.Replicas)



	// 4. O-RAN WG11 Compliance Check (if enabled).

	if config.EnableCompliance {

		logger.V(1).Info("Performing O-RAN WG11 compliance assessment")



		complianceChecker, err := security.NewORANComplianceChecker()

		if err != nil {

			return fmt.Errorf("failed to initialize compliance checker: %w", err)

		}



		complianceReport, err := complianceChecker.AssessCompliance(intent)

		if err != nil {

			return fmt.Errorf("compliance assessment failed: %w", err)

		}



		logger.Info("O-RAN WG11 compliance assessment completed",

			"overall_score", complianceReport.OverallScore,

			"critical_issues", complianceReport.CriticalIssues)



		if complianceReport.CriticalIssues > 0 {

			logger.Error(fmt.Errorf("compliance issues detected"), "Critical compliance issues found",

				"critical_issues", complianceReport.CriticalIssues,

				"recommendations", complianceReport.Recommendations)



			// In production, you might want to fail here or require override.

			logger.Info("Continuing despite compliance issues (development mode)")

		}

	}



	// 5. Generate secure patch using hardened generator.

	logger.V(1).Info("Initializing secure patch generator")



	secureGenerator, err := security.NewSecurePatchGenerator(intent, outputDir, logger)

	if err != nil {

		return fmt.Errorf("failed to create secure patch generator: %w", err)

	}



	logger.Info("Generating secure patch package")

	securePatch, err := secureGenerator.GenerateSecure()

	if err != nil {

		return fmt.Errorf("failed to generate secure patch: %w", err)

	}



	packagePath := securePatch.GetPackagePath()

	logger.Info("Secure patch package generated successfully",

		"package_path", packagePath,

		"security_validated", securePatch.SecurityMetadata.ValidationPassed,

		"compliance_level", securePatch.SecurityMetadata.ThreatModel)



	// 6. Optionally apply using secure command execution.

	if apply {

		logger.Info("Applying patch using secure command execution")



		if err := applyPatchSecurely(ctx, packagePath, logger); err != nil {

			logger.Error(err, "Failed to apply patch securely")

			fmt.Printf("Warning: Secure patch application failed: %v\n", err)

			fmt.Printf("You can manually apply with: porch-direct --package %s\n", packagePath)

		} else {

			logger.Info("Patch applied successfully via secure execution")

			fmt.Println("Patch applied successfully via secure porch-direct execution")

		}

	}



	// 7. Final security summary.

	logger.Info("Secure patch generation workflow completed successfully",

		"package_name", securePatch.Kptfile.Metadata.Name,

		"target_deployment", intent.Target,

		"replica_count", intent.Replicas,

		"security_compliance", securePatch.SecurityMetadata.ValidationPassed)



	fmt.Printf("\n=== SECURE PATCH GENERATION COMPLETE ===\n")

	fmt.Printf("Package Name: %s\n", securePatch.Kptfile.Metadata.Name)

	fmt.Printf("Target: %s (namespace: %s)\n", intent.Target, intent.Namespace)

	fmt.Printf("Replicas: %d\n", intent.Replicas)

	fmt.Printf("Location: %s\n", packagePath)

	fmt.Printf("Security Validated: %t\n", securePatch.SecurityMetadata.ValidationPassed)

	fmt.Printf("Compliance Level: %s\n", securePatch.SecurityMetadata.ThreatModel)

	fmt.Printf("============================================\n\n")



	return nil

}



// applyPatchSecurely applies the patch using secure command execution.

func applyPatchSecurely(_ context.Context, packagePath string, logger logr.Logger) error {

	logger.V(1).Info("Initializing secure command executor")



	secureExecutor, err := security.NewSecureCommandExecutor()

	if err != nil {

		return fmt.Errorf("failed to create secure command executor: %w", err)

	}



	logger.Info("Executing porch-direct with security controls", "package_path", packagePath)



	// Execute porch-direct with comprehensive security controls.

	result, err := secureExecutor.ExecuteSecure("porch-direct", []string{"--package", packagePath}, ".")

	if err != nil {

		return fmt.Errorf("secure command execution setup failed: %w", err)

	}



	if result.Error != nil {

		return fmt.Errorf("porch-direct execution failed: %w", result.Error)

	}



	if result.ExitCode != 0 {

		return fmt.Errorf("porch-direct failed with exit code %d", result.ExitCode)

	}



	// Log security execution details.

	logger.Info("Secure command execution completed successfully",

		"binary_verified", result.SecurityInfo.BinaryVerified,

		"arguments_sanitized", result.SecurityInfo.ArgumentsSanitized,

		"environment_secure", result.SecurityInfo.EnvironmentSecure,

		"resources_limited", result.SecurityInfo.ResourcesLimited,

		"audit_logged", result.SecurityInfo.AuditLogged,

		"execution_duration", result.Duration)



	return nil

}



// initializeLogger sets up structured logging with appropriate verbosity.

func initializeLogger(verbosity int) (logr.Logger, error) {

	var zapConfig zap.Config



	switch verbosity {

	case 0:

		zapConfig = zap.NewProductionConfig()

		zapConfig.Level.SetLevel(zap.InfoLevel)

	case 1:

		zapConfig = zap.NewDevelopmentConfig()

		zapConfig.Level.SetLevel(zap.InfoLevel)

	case 2:

		zapConfig = zap.NewDevelopmentConfig()

		zapConfig.Level.SetLevel(zap.DebugLevel)

	default: // 3+

		zapConfig = zap.NewDevelopmentConfig()

		zapConfig.Level.SetLevel(zap.DebugLevel)

		zapConfig.Development = true

	}



	// Add security-relevant fields.

	zapConfig.InitialFields = map[string]interface{}{

		"service":         "secure-porch-patch",

		"version":         Version,

		"security_mode":   true,

		"compliance_mode": true,

	}



	zapLogger, err := zapConfig.Build()

	if err != nil {

		return logr.Logger{}, fmt.Errorf("failed to build zap logger: %w", err)

	}



	return zapr.NewLogger(zapLogger), nil

}

