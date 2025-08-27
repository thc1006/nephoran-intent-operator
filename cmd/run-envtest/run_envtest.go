// Package main provides a standalone runner for the NetworkIntent controller envtest.
// This can be used to run the envtest independently or in specific CI scenarios.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	var (
		verbose      = flag.Bool("v", false, "Enable verbose output")
		focus        = flag.String("focus", "", "Focus on specific test specs (Ginkgo focus)")
		skip         = flag.String("skip", "", "Skip specific test specs (Ginkgo skip)")
		parallel     = flag.Int("p", 1, "Number of parallel test processes")
		timeout      = flag.String("timeout", "30m", "Test timeout")
		coverMode    = flag.String("covermode", "atomic", "Coverage mode")
		coverProfile = flag.String("coverprofile", "", "Coverage profile file")
		dryRun       = flag.Bool("dry-run", false, "Show commands that would be executed")
	)
	flag.Parse()

	// Get project root
	projectRoot, err := getProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	// Change to the controllers test directory
	testDir := filepath.Join(projectRoot, "tests", "unit", "controllers")
	if err := os.Chdir(testDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to test directory %s: %v\n", testDir, err)
		os.Exit(1)
	}

	// Set up environment
	setupEnv()

	// Build the ginkgo command
	cmd := buildGinkgoCommand(*verbose, *focus, *skip, *parallel, *timeout, *coverMode, *coverProfile)

	if *dryRun {
		fmt.Println("Command that would be executed:")
		fmt.Println(strings.Join(cmd.Args, " "))
		return
	}

	fmt.Printf("Running NetworkIntent controller envtest from %s\n", testDir)
	fmt.Printf("Command: %s\n", strings.Join(cmd.Args, " "))

	// Run the command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Test execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("NetworkIntent controller envtest completed successfully!")
}

// getProjectRoot finds the project root directory by looking for go.mod
func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find project root (no go.mod found)")
}

// setupEnv sets up necessary environment variables
func setupEnv() {
	// Set test-specific environment variables
	_ = os.Setenv("KUBEBUILDER_ASSETS", getKubebuilderAssets())
	_ = os.Setenv("TEST_ENV", "true")

	// Disable LLM by default for testing unless explicitly set
	if os.Getenv("ENABLE_LLM_INTENT") == "" {
		_ = os.Setenv("ENABLE_LLM_INTENT", "false")
	}

	// Set reasonable defaults for test timeouts
	if os.Getenv("ENVTEST_TIMEOUT") == "" {
		_ = os.Setenv("ENVTEST_TIMEOUT", "300s")
	}

	fmt.Println("Environment setup:")
	fmt.Printf("  KUBEBUILDER_ASSETS=%s\n", os.Getenv("KUBEBUILDER_ASSETS"))
	fmt.Printf("  ENABLE_LLM_INTENT=%s\n", os.Getenv("ENABLE_LLM_INTENT"))
	fmt.Printf("  TEST_ENV=%s\n", os.Getenv("TEST_ENV"))
}

// getKubebuilderAssets returns the path to kubebuilder assets
func getKubebuilderAssets() string {
	// Check if already set
	if assets := os.Getenv("KUBEBUILDER_ASSETS"); assets != "" {
		return assets
	}

	// Common paths based on OS
	var commonPaths []string

	if runtime.GOOS == "windows" {
		commonPaths = []string{
			filepath.Join(os.Getenv("USERPROFILE"), ".local", "share", "kubebuilder-envtest", "k8s", "1.28.0-windows-amd64"),
			filepath.Join(os.Getenv("USERPROFILE"), ".local", "share", "kubebuilder-envtest", "k8s", "1.27.1-windows-amd64"),
			"C:\\kubebuilder\\bin",
		}
	} else {
		commonPaths = []string{
			filepath.Join(os.Getenv("HOME"), ".local", "share", "kubebuilder-envtest", "k8s", "1.28.0-linux-amd64"),
			filepath.Join(os.Getenv("HOME"), ".local", "share", "kubebuilder-envtest", "k8s", "1.27.1-linux-amd64"),
			"/usr/local/kubebuilder/bin",
		}
	}

	// Find the first existing path
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Return empty and let envtest discover
	return ""
}

// buildGinkgoCommand builds the ginkgo command with appropriate flags
func buildGinkgoCommand(verbose bool, focus, skip string, parallel int, timeout, coverMode, coverProfile string) *exec.Cmd {
	args := []string{"ginkgo"}

	// Basic flags
	if verbose {
		args = append(args, "-v")
	}

	// Focus and skip patterns
	if focus != "" {
		args = append(args, "--focus", focus)
	}
	if skip != "" {
		args = append(args, "--skip", skip)
	}

	// Parallelization
	if parallel > 1 {
		args = append(args, fmt.Sprintf("-p=%d", parallel))
	}

	// Coverage
	if coverProfile != "" {
		args = append(args, "--cover", "--coverprofile", coverProfile)
		if coverMode != "" {
			args = append(args, "--covermode", coverMode)
		}
	}

	// Output format
	args = append(args, "--progress")

	// Timeout
	args = append(args, "--timeout", timeout)

	// Run specific test file
	args = append(args, "networkintent_envtest.go")

	return exec.Command("ginkgo", args[1:]...)
}
