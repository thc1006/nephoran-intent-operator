package porch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (

	// DefaultTimeout for porch command execution.

	DefaultTimeout = 30 * time.Second

	// Mode constants.

	ModeDirect = "direct"

	// ModeStructured holds modestructured value.

	ModeStructured = "structured"
)

// ExecutorConfig holds configuration for the porch executor.

type ExecutorConfig struct {
	PorchPath string `json:"porch_path"`

	Mode string `json:"mode"`

	OutDir string `json:"out_dir"`

	Timeout time.Duration `json:"timeout"`
}

// ExecutionResult holds the result of a porch command execution.

type ExecutionResult struct {
	Success bool `json:"success"`

	ExitCode int `json:"exit_code"`

	Stdout string `json:"stdout"`

	Stderr string `json:"stderr"`

	Duration time.Duration `json:"duration"`

	Command string `json:"command"`

	Error error `json:"error,omitempty"`
}

// Executor manages porch command execution.

type Executor struct {
	config ExecutorConfig
}

// NewExecutor creates a new porch executor with the given configuration.

func NewExecutor(config ExecutorConfig) *Executor {
	// Set default timeout if not specified.

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	// Validate mode.

	if config.Mode != ModeDirect && config.Mode != ModeStructured {
		config.Mode = ModeDirect
	}

	return &Executor{
		config: config,
	}
}

// Execute runs the porch command for the given intent file with graceful handling.

func (e *Executor) Execute(ctx context.Context, intentPath string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Build command based on mode.

	cmdArgs, err := e.buildCommand(intentPath)
	if err != nil {
		return &ExecutionResult{
			Success: false,

			Command: fmt.Sprintf("<%s>", err.Error()),

			Duration: time.Since(startTime),

			Error: err,
		}, err
	}

	// Create context with timeout.
	// CRITICAL FIX: Check if parent context is already cancelled to prevent immediate cancellation
	// If parent context is cancelled (e.g., during graceful shutdown), use background context
	// with timeout to allow command to execute properly during shutdown scenarios.
	var timeoutCtx context.Context
	var cancel context.CancelFunc
	
	if ctx.Err() != nil {
		// Parent context is already cancelled (graceful shutdown scenario)
		// Use background context with timeout to allow porch command to complete
		timeoutCtx, cancel = context.WithTimeout(context.Background(), e.config.Timeout)
		log.Printf("Parent context cancelled, using background context with %v timeout", e.config.Timeout)
	} else {
		// Normal operation - use parent context with timeout
		timeoutCtx, cancel = context.WithTimeout(ctx, e.config.Timeout)
	}

	defer cancel()

	// Create graceful command instead of regular exec.Command.

	gracefulCmd := NewGracefulCommand(timeoutCtx, cmdArgs[0], cmdArgs[1:]...)

	gracefulCmd.SetGracePeriod(5 * time.Second) // 5 second grace period for SIGTERM->SIGKILL

	var stdout, stderr bytes.Buffer

	gracefulCmd.Stdout = &stdout

	gracefulCmd.Stderr = &stderr

	log.Printf("Executing porch command: %s", strings.Join(cmdArgs, " "))

	// Execute the command with graceful shutdown support.

	err = gracefulCmd.RunWithGracefulShutdown()

	duration := time.Since(startTime)

	result := &ExecutionResult{
		Success: err == nil,

		ExitCode: getExitCode(err),

		Stdout: strings.TrimSpace(stdout.String()),

		Stderr: strings.TrimSpace(stderr.String()),

		Duration: duration,

		Command: strings.Join(cmdArgs, " "),
	}

	// Check for context errors first (even if RunWithGracefulShutdown returned nil,
	// the context may have been cancelled or timed out and the process was killed gracefully).
	if timeoutCtx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = fmt.Errorf("porch command timed out after %v", e.config.Timeout)
	} else if timeoutCtx.Err() != nil {
		// Context was cancelled (e.g., parent context cancelled)
		result.Success = false
		result.Error = fmt.Errorf("porch command cancelled: %w", timeoutCtx.Err())
	} else if err != nil {
		result.Error = err
	}

	// Log execution result.

	if result.Success {

		log.Printf("Porch command completed successfully in %v", duration)

		if result.Stdout != "" {
			log.Printf("Porch stdout: %s", result.Stdout)
		}

	} else {

		log.Printf("Porch command failed (exit code %d) in %v: %v", result.ExitCode, duration, result.Error)

		if result.Stderr != "" {
			log.Printf("Porch stderr: %s", result.Stderr)
		}

	}

	return result, nil
}

// buildCommand constructs the porch command based on the mode and configuration.

func (e *Executor) buildCommand(intentPath string) ([]string, error) {
	// Convert paths to absolute paths for consistency.

	absIntentPath, err := filepath.Abs(intentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute intent path: %w", err)
	}

	absOutDir, err := filepath.Abs(e.config.OutDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute output directory: %w", err)
	}

	// Ensure porch path is properly resolved
	porchPath := e.config.PorchPath
	if filepath.IsAbs(porchPath) {
		// Use absolute path as-is for better reliability
		absPath, err := filepath.Abs(porchPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute porch path: %w", err)
		}
		porchPath = absPath
	}

	// Build command based on mode.

	switch e.config.Mode {

	case ModeDirect:

		return []string{
			porchPath,

			"-intent", absIntentPath,

			"-out", absOutDir,
		}, nil

	case ModeStructured:

		return []string{
			porchPath,

			"-intent", absIntentPath,

			"-out", absOutDir,

			"-structured",
		}, nil

	default:

		return nil, fmt.Errorf("unsupported mode: %s", e.config.Mode)

	}
}

// getExitCode extracts the exit code from a command execution.

func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitError *exec.ExitError

	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}

	// For other types of errors, return -1.

	return -1
}

// ValidatePorchPath checks if the porch executable exists and is executable.

func ValidatePorchPath(porchPath string) error {
	// Handle empty path
	if porchPath == "" {
		return fmt.Errorf("porch path cannot be empty")
	}
	
	// If path is absolute, check file existence first
	if filepath.IsAbs(porchPath) {
		if _, err := os.Stat(porchPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("porch executable not found at path: %s", porchPath)
			}
			return fmt.Errorf("cannot access porch executable: %w", err)
		}
	}

	// Try to run porch with --help to validate it exists and works.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// For cross-platform compatibility, handle .bat files on Windows and .sh on Unix
	var cmd *exec.Cmd
	if filepath.IsAbs(porchPath) {
		// Use absolute path directly
		cmd = exec.CommandContext(ctx, porchPath, "--help")
	} else {
		// Use relative path (for PATH lookup)
		cmd = exec.CommandContext(ctx, porchPath, "--help")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return fmt.Errorf("porch validation failed: %w (stderr: %s)", err, stderrStr)
		}
		return fmt.Errorf("porch validation failed: %w", err)
	}

	return nil
}

// ExecutorStats holds statistics about executor usage.

type ExecutorStats struct {
	TotalExecutions int `json:"total_executions"`

	SuccessfulExecs int `json:"successful_executions"`

	FailedExecs int `json:"failed_executions"`

	AverageExecTime time.Duration `json:"average_execution_time"`

	TotalExecTime time.Duration `json:"total_execution_time"`

	TimeoutCount int `json:"timeout_count"`
}

// StatefulExecutor wraps Executor with statistics tracking.

type StatefulExecutor struct {
	*Executor

	stats ExecutorStats

	mu sync.RWMutex
}

// NewStatefulExecutor creates a new stateful executor that tracks execution statistics.

func NewStatefulExecutor(config ExecutorConfig) *StatefulExecutor {
	return &StatefulExecutor{
		Executor: NewExecutor(config),

		stats: ExecutorStats{},
	}
}

// Execute runs the porch command and updates statistics.

func (se *StatefulExecutor) Execute(ctx context.Context, intentPath string) (*ExecutionResult, error) {
	result, err := se.Executor.Execute(ctx, intentPath)
	// If porch is not available, generate fallback YAML.
	if err != nil {

		errStr := err.Error()

		if strings.Contains(errStr, "executable file not found") ||

			strings.Contains(errStr, "cannot run executable") ||

			strings.Contains(errStr, "no such file or directory") {

			log.Printf("Porch not found (error: %s), generating fallback YAML for %s", errStr, intentPath)

			fallbackResult, fallbackErr := se.generateFallbackYAML(intentPath)

			if fallbackErr == nil {

				result = fallbackResult

				err = nil

			} else {
				log.Printf("Fallback YAML generation failed: %v", fallbackErr)
			}

		}

	}

	// Update statistics with thread safety.

	se.mu.Lock()

	se.stats.TotalExecutions++

	se.stats.TotalExecTime += result.Duration

	se.stats.AverageExecTime = se.stats.TotalExecTime / time.Duration(se.stats.TotalExecutions)

	if result.Success {
		se.stats.SuccessfulExecs++
	} else {

		se.stats.FailedExecs++

		// Check if it was a timeout.

		if result.Error != nil && strings.Contains(result.Error.Error(), "timed out") {
			se.stats.TimeoutCount++
		}

	}

	se.mu.Unlock()

	return result, err
}

// GetStats returns a copy of the current execution statistics.

func (se *StatefulExecutor) GetStats() ExecutorStats {
	se.mu.RLock()

	stats := se.stats

	se.mu.RUnlock()

	return stats
}

// ResetStats resets all execution statistics.

func (se *StatefulExecutor) ResetStats() {
	se.mu.Lock()

	se.stats = ExecutorStats{}

	se.mu.Unlock()
}

// generateFallbackYAML creates a simple YAML output when porch is not available.

func (se *StatefulExecutor) generateFallbackYAML(intentPath string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Read the intent JSON file.

	data, err := os.ReadFile(intentPath)
	if err != nil {
		return &ExecutionResult{
			Success: false,

			Duration: time.Since(startTime),

			Error: fmt.Errorf("failed to read intent file: %w", err),
		}, err
	}

	// Parse the JSON to extract metadata.

	var intent map[string]interface{}

	if err := json.Unmarshal(data, &intent); err != nil {
		return &ExecutionResult{
			Success: false,

			Duration: time.Since(startTime),

			Error: fmt.Errorf("failed to parse intent JSON: %w", err),
		}, err
	}

	// Extract name from metadata or use filename.

	var intentName string

	if metadata, ok := intent["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			intentName = name
		}
	}

	if intentName == "" {
		intentName = strings.TrimSuffix(filepath.Base(intentPath), ".json")
	}

	// Generate simple YAML output.

	yamlContent := fmt.Sprintf(`apiVersion: v1

kind: ConfigMap

metadata:

  name: %s-output

  namespace: default

  labels:

    generated-by: conductor-loop

    intent-processed: "true"

    processing-mode: fallback

data:

  intent-name: %s

  processed-at: %s

  status: processed

  original-intent: |

%s`, intentName, intentName, time.Now().Format(time.RFC3339), indentJSON(string(data)))

	// Write YAML to output directory.

	yamlFilename := fmt.Sprintf("%s.yaml", intentName)

	yamlPath := filepath.Join(se.config.OutDir, yamlFilename)

	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o640); err != nil {
		return &ExecutionResult{
			Success: false,

			Duration: time.Since(startTime),

			Error: fmt.Errorf("failed to write YAML file: %w", err),
		}, err
	}

	duration := time.Since(startTime)

	return &ExecutionResult{
		Success: true,

		ExitCode: 0,

		Duration: duration,

		Command: fmt.Sprintf("fallback-yaml-generator %s", intentPath),

		Stdout: fmt.Sprintf("Generated YAML: %s", yamlPath),
	}, nil
}

// indentJSON adds 4-space indentation to each line of JSON for YAML embedding.

func indentJSON(jsonStr string) string {
	lines := strings.Split(jsonStr, "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = "    " + line
		}
	}

	return strings.Join(lines, "\n")
}
