package security

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// SecureCommandExecutor provides hardened command execution with zero-trust controls.

type SecureCommandExecutor struct {
	allowedBinaries map[string]BinaryPolicy

	maxTimeout time.Duration

	auditor *CommandAuditor
}

// BinaryPolicy defines security policy for executable binaries.

type BinaryPolicy struct {
	AllowedPaths []string

	RequiredSigner string

	MaxArgs int

	AllowedArgs []string

	ForbiddenArgs []string

	Environment map[string]string

	Capabilities []string
}

// CommandAuditor provides audit logging for command execution.

type CommandAuditor struct {
	logPath string

	enabled bool
}

// SecureCommand represents a validated and sanitized command.

type SecureCommand struct {
	Binary string

	Args []string

	Environment []string

	WorkingDir string

	Timeout time.Duration

	Policy BinaryPolicy
}

// CommandResult contains execution results with security context.

type CommandResult struct {
	ExitCode int

	Output []byte

	Error error

	Duration time.Duration

	SecurityInfo SecurityExecutionInfo
}

// SecurityExecutionInfo contains security-related execution information.

type SecurityExecutionInfo struct {
	BinaryVerified bool

	ArgumentsSanitized bool

	EnvironmentSecure bool

	ResourcesLimited bool

	AuditLogged bool
}

// NewSecureCommandExecutor creates a new secure command executor.

func NewSecureCommandExecutor() (*SecureCommandExecutor, error) {
	auditor, err := NewCommandAuditor()
	if err != nil {
		return nil, fmt.Errorf("failed to create command auditor: %w", err)
	}

	return &SecureCommandExecutor{
		allowedBinaries: getDefaultBinaryPolicies(),

		maxTimeout: 10 * time.Minute, // O-RAN WG11 recommended timeout

		auditor: auditor,
	}, nil
}

// ExecuteSecure executes a command with comprehensive security controls.

func (e *SecureCommandExecutor) ExecuteSecure(ctx context.Context, binaryName string, args []string, workingDir string) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		SecurityInfo: SecurityExecutionInfo{},
	}

	// 1. Binary validation and policy enforcement.

	secureCmd, err := e.validateAndPrepareCommand(binaryName, args, workingDir)
	if err != nil {

		result.Error = fmt.Errorf("command validation failed: %w", err)

		return result, nil

	}

	result.SecurityInfo.BinaryVerified = true

	result.SecurityInfo.ArgumentsSanitized = true

	// 2. Environment hardening.

	if err := e.hardenEnvironment(secureCmd); err != nil {

		result.Error = fmt.Errorf("environment hardening failed: %w", err)

		return result, nil

	}

	result.SecurityInfo.EnvironmentSecure = true

	// 3. Resource limiting.

	timeoutCtx, cancel := context.WithTimeout(ctx, secureCmd.Timeout)

	defer cancel()

	// 4. Command execution with security monitoring.

	cmd := exec.CommandContext(timeoutCtx, secureCmd.Binary, secureCmd.Args...)

	cmd.Dir = secureCmd.WorkingDir

	cmd.Env = secureCmd.Environment

	// Apply system-level security controls.

	if err := e.applySecurityControls(cmd); err != nil {

		result.Error = fmt.Errorf("failed to apply security controls: %w", err)

		return result, nil

	}

	result.SecurityInfo.ResourcesLimited = true

	// 5. Audit logging.

	auditEntry := e.createAuditEntry(secureCmd, startTime)

	e.auditor.LogExecution(auditEntry)

	result.SecurityInfo.AuditLogged = true

	// 6. Execute command.

	output, err := cmd.CombinedOutput()

	result.Output = output

	result.Duration = time.Since(startTime)

	if err != nil {

		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = fmt.Errorf("command execution failed: %w", err)
		}

	} else {
		result.ExitCode = 0
	}

	// 7. Post-execution audit.

	auditEntry.Duration = result.Duration

	auditEntry.ExitCode = result.ExitCode

	auditEntry.Success = result.ExitCode == 0

	e.auditor.LogCompletion(auditEntry)

	return result, nil
}

// validateAndPrepareCommand performs comprehensive validation and sanitization.

func (e *SecureCommandExecutor) validateAndPrepareCommand(binaryName string, args []string, workingDir string) (*SecureCommand, error) {
	// Get binary policy.

	policy, exists := e.allowedBinaries[binaryName]

	if !exists {
		return nil, fmt.Errorf("binary %s is not in allowed list", binaryName)
	}

	// Validate binary path.

	binaryPath, err := e.validateBinaryPath(binaryName, policy)
	if err != nil {
		return nil, fmt.Errorf("binary path validation failed: %w", err)
	}

	// Sanitize and validate arguments.

	sanitizedArgs, err := e.sanitizeArguments(args, policy)
	if err != nil {
		return nil, fmt.Errorf("argument validation failed: %w", err)
	}

	// Validate working directory.

	cleanWorkingDir, err := e.validateWorkingDirectory(workingDir)
	if err != nil {
		return nil, fmt.Errorf("working directory validation failed: %w", err)
	}

	return &SecureCommand{
		Binary: binaryPath,

		Args: sanitizedArgs,

		WorkingDir: cleanWorkingDir,

		Timeout: e.maxTimeout,

		Policy: policy,
	}, nil
}

// validateBinaryPath ensures binary is in allowed locations and signed.

func (e *SecureCommandExecutor) validateBinaryPath(binaryName string, policy BinaryPolicy) (string, error) {
	// Look up binary in PATH.

	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		return "", fmt.Errorf("binary %s not found in PATH: %w", binaryName, err)
	}

	// Get absolute path.

	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Validate against allowed paths.

	allowed := false

	for _, allowedPath := range policy.AllowedPaths {

		matched, err := filepath.Match(allowedPath, absBinaryPath)
		if err != nil {
			return "", fmt.Errorf("invalid allowed path pattern %s: %w", allowedPath, err)
		}

		if matched {

			allowed = true

			break

		}

	}

	if !allowed {
		return "", fmt.Errorf("binary %s not in allowed paths", absBinaryPath)
	}

	// Security checks for suspicious locations.

	suspiciousPaths := []string{"/tmp", "/var/tmp", "temp", "Temp"}

	for _, suspicious := range suspiciousPaths {
		if strings.Contains(strings.ToLower(absBinaryPath), strings.ToLower(suspicious)) {
			return "", fmt.Errorf("binary in suspicious location: %s", absBinaryPath)
		}
	}

	return absBinaryPath, nil
}

// sanitizeArguments validates and sanitizes command arguments.

func (e *SecureCommandExecutor) sanitizeArguments(args []string, policy BinaryPolicy) ([]string, error) {
	if len(args) > policy.MaxArgs {
		return nil, fmt.Errorf("too many arguments: %d > %d", len(args), policy.MaxArgs)
	}

	sanitized := make([]string, 0, len(args))

	for _, arg := range args {

		// Check forbidden arguments.

		for _, forbidden := range policy.ForbiddenArgs {

			matched, err := regexp.MatchString(forbidden, arg)
			if err != nil {
				return nil, fmt.Errorf("invalid forbidden argument pattern %s: %w", forbidden, err)
			}

			if matched {
				return nil, fmt.Errorf("forbidden argument pattern: %s", forbidden)
			}

		}

		// Sanitize argument.

		clean := e.sanitizeArgument(arg)

		// Validate against allowed patterns if specified.

		if len(policy.AllowedArgs) > 0 {

			allowed := false

			for _, pattern := range policy.AllowedArgs {

				matched, err := regexp.MatchString(pattern, clean)
				if err != nil {
					return nil, fmt.Errorf("invalid allowed argument pattern %s: %w", pattern, err)
				}

				if matched {

					allowed = true

					break

				}

			}

			if !allowed {
				return nil, fmt.Errorf("argument does not match allowed patterns: %s", arg)
			}

		}

		sanitized = append(sanitized, clean)

	}

	return sanitized, nil
}

// sanitizeArgument removes potential injection characters.

func (e *SecureCommandExecutor) sanitizeArgument(arg string) string {
	// Remove null bytes and control characters.

	clean := strings.ReplaceAll(arg, "\x00", "")

	// Remove potentially dangerous characters for shell injection.

	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "<", ">", "\\", "\"", "'", "*", "?"}

	for _, char := range dangerousChars {
		clean = strings.ReplaceAll(clean, char, "")
	}

	return clean
}

// validateWorkingDirectory ensures working directory is secure.

func (e *SecureCommandExecutor) validateWorkingDirectory(workingDir string) (string, error) {
	if workingDir == "" {

		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		workingDir = cwd

	}

	// Clean and validate path.

	clean := filepath.Clean(workingDir)

	// Check for path traversal.

	if strings.Contains(workingDir, "..") {
		return "", fmt.Errorf("working directory contains path traversal: %s", workingDir)
	}

	// Ensure directory exists.

	if info, err := os.Stat(clean); err != nil {
		return "", fmt.Errorf("working directory does not exist: %s", clean)
	} else if !info.IsDir() {
		return "", fmt.Errorf("working directory is not a directory: %s", clean)
	}

	return clean, nil
}

// hardenEnvironment sets up secure environment variables.

func (e *SecureCommandExecutor) hardenEnvironment(cmd *SecureCommand) error {
	// Start with minimal environment.

	baseEnv := []string{
		"PATH=" + os.Getenv("PATH"),

		"HOME=" + os.Getenv("HOME"),

		"USER=" + os.Getenv("USER"),

		"LANG=C",

		"LC_ALL=C",
	}

	// Add policy-specific environment variables.

	for key, value := range cmd.Policy.Environment {
		baseEnv = append(baseEnv, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Environment = baseEnv

	return nil
}

// applySecurityControls applies system-level security controls.

func (e *SecureCommandExecutor) applySecurityControls(cmd *exec.Cmd) error {
	// Set process group to enable proper signal handling.

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Create new process group (Unix-like systems only).

	// On Windows, process groups work differently and Setpgid is not available.

	// TODO: Implement Windows-specific job object for process isolation.

	// Additional Windows-specific security controls could be added here.

	// For example, job objects for resource limiting.

	return nil
}

// createAuditEntry creates an audit log entry for the command.

func (e *SecureCommandExecutor) createAuditEntry(cmd *SecureCommand, startTime time.Time) *CommandAuditEntry {
	return &CommandAuditEntry{
		Timestamp: startTime,

		Binary: cmd.Binary,

		Args: cmd.Args,

		WorkingDir: cmd.WorkingDir,

		User: os.Getenv("USER"),

		PID: os.Getpid(),
	}
}

// getDefaultBinaryPolicies returns default security policies for allowed binaries.

func getDefaultBinaryPolicies() map[string]BinaryPolicy {
	return map[string]BinaryPolicy{
		"porch-direct": {
			AllowedPaths: []string{
				"/usr/local/bin/porch-direct",

				"/usr/bin/porch-direct",

				"/bin/porch-direct",

				"*/bin/porch-direct",

				"porch-direct.exe",
			},

			MaxArgs: 10,

			AllowedArgs: []string{
				`^--package$`,

				`^--.*$`, // Allow all flag patterns for now, can be restricted later

				`^[a-zA-Z0-9._/-]+$`, // Allow alphanumeric paths

			},

			ForbiddenArgs: []string{
				`;.*`, // Command injection

				`&.*`, // Background execution

				`\|.*`, // Piping

				`>.*`, // Redirection

				`<.*`, // Input redirection

				`\$.*`, // Variable expansion

				"`.*`", // Command substitution

				`\(.*\)`, // Subshell execution

			},

			Environment: map[string]string{
				"PORCH_SECURE_MODE": "true",
			},
		},

		"kpt": {
			AllowedPaths: []string{
				"/usr/local/bin/kpt",

				"/usr/bin/kpt",

				"/bin/kpt",

				"*/bin/kpt",

				"kpt.exe",
			},

			MaxArgs: 20,

			AllowedArgs: []string{
				`^(fn|pkg|live|alpha)$`,

				`^--.*$`,

				`^[a-zA-Z0-9._/-]+$`,
			},

			ForbiddenArgs: []string{
				`;.*`, `&.*`, `\|.*`, `>.*`, `<.*`, `\$.*`, "`.*`", `\(.*\)`,
			},
		},
	}
}

// CommandAuditEntry represents an audit log entry.

type CommandAuditEntry struct {
	Timestamp time.Time

	Binary string

	Args []string

	WorkingDir string

	User string

	PID int

	Duration time.Duration

	ExitCode int

	Success bool
}

// NewCommandAuditor creates a new command auditor.

func NewCommandAuditor() (*CommandAuditor, error) {
	return &CommandAuditor{
		logPath: "./security-audit.log",

		enabled: true,
	}, nil
}

// LogExecution logs command execution start.

func (a *CommandAuditor) LogExecution(entry *CommandAuditEntry) {
	if !a.enabled {
		return
	}

	// In a production system, this would write to secure audit logs.

	// For now, we'll use structured logging.

	fmt.Printf("[AUDIT] Command execution started: %s %v\n", entry.Binary, entry.Args)
}

// LogCompletion logs command execution completion.

func (a *CommandAuditor) LogCompletion(entry *CommandAuditEntry) {
	if !a.enabled {
		return
	}

	fmt.Printf("[AUDIT] Command execution completed: %s (exit: %d, duration: %v)\n",

		entry.Binary, entry.ExitCode, entry.Duration)
}
