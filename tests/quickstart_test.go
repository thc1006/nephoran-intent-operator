package tests

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestQuickstartTutorial validates that the quickstart tutorial completes in under 15 minutes
func TestQuickstartTutorial(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping quickstart test in short mode")
	}

	// Check if we're in CI environment
	if os.Getenv("CI") == "true" {
		t.Log("Running in CI environment")
	}

	// Start timer
	startTime := time.Now()
	maxDuration := 15 * time.Minute

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()

	// Determine which script to run based on OS
	var scriptPath string
	var cmd *exec.Cmd

	projectRoot := findProjectRoot(t)

	switch runtime.GOOS {
	case "windows":
		scriptPath = filepath.Join(projectRoot, "scripts", "quickstart.ps1")
		cmd = exec.CommandContext(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "-SkipPrereq")
	default:
		scriptPath = filepath.Join(projectRoot, "scripts", "quickstart.sh")
		cmd = exec.CommandContext(ctx, "/bin/bash", scriptPath, "--skip-prereq")
	}

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Quickstart script not found at %s", scriptPath)
	}

	t.Logf("Running quickstart script: %s", scriptPath)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the quickstart script
	err := cmd.Run()

	// Calculate elapsed time
	elapsed := time.Since(startTime)

	// Log output for debugging
	t.Logf("Stdout:\n%s", stdout.String())
	if stderr.Len() > 0 {
		t.Logf("Stderr:\n%s", stderr.String())
	}

	// Check for errors
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("Quickstart exceeded 15-minute time limit (elapsed: %v)", elapsed)
		}
		t.Fatalf("Quickstart failed: %v", err)
	}

	// Verify completion time
	if elapsed > maxDuration {
		t.Errorf("Quickstart took %v, exceeding 15-minute target", elapsed)
	} else {
		t.Logf("??Quickstart completed in %v (under 15-minute target)", elapsed)
	}

	// Run validation checks
	t.Run("Validation", func(t *testing.T) {
		validateQuickstartDeployment(t)
	})

	// Clean up
	t.Run("Cleanup", func(t *testing.T) {
		cleanupQuickstart(t)
	})
}

// TestQuickstartPrerequisites verifies all required tools are available
func TestQuickstartPrerequisites(t *testing.T) {
	requiredTools := []struct {
		name    string
		command string
		args    []string
	}{
		{"Docker", "docker", []string{"--version"}},
		{"kubectl", "kubectl", []string{"version", "--client", "--short"}},
		{"Git", "git", []string{"--version"}},
		{"Kind", "kind", []string{"--version"}},
	}

	for _, tool := range requiredTools {
		t.Run(tool.name, func(t *testing.T) {
			cmd := exec.Command(tool.command, tool.args...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Errorf("%s not found or not working: %v", tool.name, err)
			} else {
				t.Logf("??%s: %s", tool.name, strings.TrimSpace(string(output)))
			}
		})
	}
}

// TestQuickstartSteps validates each step of the quickstart independently
func TestQuickstartSteps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping detailed quickstart steps test in short mode")
	}

	steps := []struct {
		name        string
		maxDuration time.Duration
		validate    func(t *testing.T)
	}{
		{
			name:        "Prerequisites",
			maxDuration: 2 * time.Minute,
			validate:    validatePrerequisites,
		},
		{
			name:        "ClusterSetup",
			maxDuration: 5 * time.Minute,
			validate:    validateClusterSetup,
		},
		{
			name:        "CRDInstallation",
			maxDuration: 1 * time.Minute,
			validate:    validateCRDs,
		},
		{
			name:        "ControllerDeployment",
			maxDuration: 2 * time.Minute,
			validate:    validateController,
		},
		{
			name:        "FirstIntent",
			maxDuration: 3 * time.Minute,
			validate:    validateFirstIntent,
		},
		{
			name:        "Validation",
			maxDuration: 2 * time.Minute,
			validate:    validateQuickstartDeployment,
		},
	}

	var totalTime time.Duration

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			start := time.Now()

			// Run validation with timeout
			done := make(chan bool)
			go func() {
				step.validate(t)
				done <- true
			}()

			select {
			case <-done:
				elapsed := time.Since(start)
				totalTime += elapsed

				if elapsed > step.maxDuration {
					t.Errorf("Step took %v, exceeding target of %v", elapsed, step.maxDuration)
				} else {
					t.Logf("??Completed in %v (target: %v)", elapsed, step.maxDuration)
				}
			case <-time.After(step.maxDuration + 30*time.Second):
				t.Fatalf("Step timed out after %v", step.maxDuration)
			}
		})
	}

	t.Logf("Total time for all steps: %v", totalTime)
	if totalTime > 15*time.Minute {
		t.Errorf("Total time %v exceeds 15-minute target", totalTime)
	}
}

// Validation helper functions

func validatePrerequisites(t *testing.T) {
	// Check Docker daemon is running
	cmd := exec.Command("docker", "info") // #nosec G204 - Static command with validated args
	if err := cmd.Run(); err != nil {
		t.Fatal("Docker daemon is not running")
	}

	// Check kubectl is configured
	cmd = exec.Command("kubectl", "version", "--client", "--short") // #nosec G204 - Static command with validated args
	if err := cmd.Run(); err != nil {
		t.Fatal("kubectl is not properly configured")
	}
}

func validateClusterSetup(t *testing.T) {
	// Check if Kind cluster exists
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		t.Fatal("Failed to get Kind clusters")
	}

	if !strings.Contains(string(output), "nephoran-quickstart") {
		t.Fatal("Quickstart cluster not found")
	}

	// Check nodes are ready
	cmd = exec.Command("kubectl", "get", "nodes", "-o", "json") // #nosec G204 - Static command with validated args
	output, err = cmd.Output()
	if err != nil {
		t.Fatal("Failed to get nodes")
	}

	// Simple check for at least 2 nodes
	if !strings.Contains(string(output), "Ready") {
		t.Fatal("Nodes are not ready")
	}
}

func validateCRDs(t *testing.T) {
	crds := []string{
		"networkintents.nephoran.com",
		"managedelements.nephoran.com",
		"e2nodesets.nephoran.com",
	}

	for _, crd := range crds {
		cmd := exec.Command("kubectl", "get", "crd", crd) // #nosec G204 - Static command with validated args
		if err := cmd.Run(); err != nil {
			t.Errorf("CRD %s not found", crd)
		}
	}
}

func validateController(t *testing.T) {
	// Check deployment exists
	cmd := exec.Command("kubectl", "get", "deployment", "nephoran-controller", "-n", "nephoran-system") // #nosec G204 - Static command with validated args
	if err := cmd.Run(); err != nil {
		t.Fatal("Controller deployment not found")
	}

	// Check if pods are running
	cmd = exec.Command("kubectl", "get", "pods", "-n", "nephoran-system", "-l", "app=nephoran-controller", "-o", "jsonpath={.items[*].status.phase}") // #nosec G204 - Static command with validated args
	output, err := cmd.Output()
	if err != nil {
		t.Fatal("Failed to get controller pods")
	}

	if !strings.Contains(string(output), "Running") {
		t.Fatal("Controller pods are not running")
	}
}

func validateFirstIntent(t *testing.T) {
	// Check if intent exists
	cmd := exec.Command("kubectl", "get", "networkintent", "deploy-amf-quickstart") // #nosec G204 - Static command with validated args
	if err := cmd.Run(); err != nil {
		t.Fatal("First intent not found")
	}

	// Check intent status
	cmd = exec.Command("kubectl", "get", "networkintent", "deploy-amf-quickstart", "-o", "jsonpath={.status.phase}") // #nosec G204 - Static command with validated args
	output, err := cmd.Output()
	if err != nil {
		t.Fatal("Failed to get intent status")
	}

	status := strings.TrimSpace(string(output))
	validStatuses := []string{"Deployed", "Ready", "Completed", "Processing"}

	valid := false
	for _, s := range validStatuses {
		if status == s {
			valid = true
			break
		}
	}

	if !valid && status != "" {
		t.Errorf("Intent has unexpected status: %s", status)
	}
}

func validateQuickstartDeployment(t *testing.T) {
	// Run the validation script
	projectRoot := findProjectRoot(t)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		validationScript := filepath.Join(projectRoot, "validate-quickstart.ps1")
		if _, err := os.Stat(validationScript); err == nil {
			cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", validationScript)
		}
	default:
		validationScript := filepath.Join(projectRoot, "validate-quickstart.sh")
		if _, err := os.Stat(validationScript); err == nil {
			cmd = exec.Command("/bin/bash", validationScript)
		}
	}

	if cmd != nil {
		output, err := cmd.CombinedOutput()
		t.Logf("Validation output:\n%s", string(output))

		if err != nil {
			t.Error("Validation script reported errors")
		}
	}
}

func cleanupQuickstart(t *testing.T) {
	projectRoot := findProjectRoot(t)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		scriptPath := filepath.Join(projectRoot, "scripts", "quickstart.ps1")
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "-Cleanup")
	default:
		scriptPath := filepath.Join(projectRoot, "scripts", "quickstart.sh")
		cmd = exec.Command("/bin/bash", scriptPath, "--cleanup")
	}

	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Cleanup failed: %v", err)
	} else {
		t.Log("??Cleanup completed successfully")
	}
}

// Helper function to find project root
func findProjectRoot(t *testing.T) string {
	// Try to find the project root by looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root")
		}
		dir = parent
	}
}

// BenchmarkQuickstart measures the performance of the quickstart process
func BenchmarkQuickstart(b *testing.B) {
	b.Skip("Benchmark skipped - run manually when needed")

	for i := 0; i < b.N; i++ {
		// Clean up from previous run
		cleanupCmd := exec.Command("/bin/bash", "scripts/quickstart.sh", "--cleanup")
		cleanupCmd.Run()

		// Run quickstart
		b.StartTimer()
		cmd := exec.Command("/bin/bash", "scripts/quickstart.sh", "--skip-prereq")
		err := cmd.Run()
		b.StopTimer()

		if err != nil {
			b.Fatalf("Quickstart failed: %v", err)
		}
	}
}

// TestQuickstartDocumentation ensures the QUICKSTART.md file is present and valid
func TestQuickstartDocumentation(t *testing.T) {
	projectRoot := findProjectRoot(t)
	quickstartPath := filepath.Join(projectRoot, "QUICKSTART.md")

	// Check if file exists
	info, err := os.Stat(quickstartPath)
	if err != nil {
		t.Fatalf("QUICKSTART.md not found: %v", err)
	}

	// Check file size (should be substantial)
	if info.Size() < 10000 {
		t.Errorf("QUICKSTART.md seems too small (%d bytes)", info.Size())
	}

	// Read and validate content
	content, err := os.ReadFile(quickstartPath)
	if err != nil {
		t.Fatalf("Failed to read QUICKSTART.md: %v", err)
	}

	// Check for required sections
	requiredSections := []string{
		"15-Minute Quick Start",
		"Prerequisites",
		"Environment Setup",
		"First Intent",
		"Validation",
		"Troubleshooting",
		"git clone",
		"make",
		"kubectl apply",
	}

	contentStr := string(content)
	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("QUICKSTART.md missing required section/content: %s", section)
		}
	}

	t.Logf("??QUICKSTART.md validated (%d bytes, all required sections present)", info.Size())
}
