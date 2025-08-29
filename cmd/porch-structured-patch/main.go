
package main



import (

	"context"

	"flag"

	"fmt"

	"os"

	"os/exec"

	"path/filepath"

	"regexp"

	"runtime"

	"strings"

	"time"



	"github.com/thc1006/nephoran-intent-operator/internal/patchgen"

)



func main() {

	var intentPath string

	var outputDir string

	var apply bool



	flag.StringVar(&intentPath, "intent", "", "Path to the intent JSON file")

	flag.StringVar(&outputDir, "out", "", "Output directory for generated patches")

	flag.BoolVar(&apply, "apply", false, "Apply the patch using porch-direct after generation")

	flag.Parse()



	if intentPath == "" {

		fmt.Fprintf(os.Stderr, "Error: --intent flag is required\n")

		flag.Usage()

		log.Fatal(1)

	}



	if outputDir == "" {

		outputDir = filepath.Join(".", "examples", "packages", "scaling")

	}



	if err := run(intentPath, outputDir, apply); err != nil {

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		log.Fatal(1)

	}

}



func run(intentPath, outputDir string, apply bool) error {

	// Validate output directory first.

	if err := validateOutputDir(outputDir); err != nil {

		return fmt.Errorf("output directory validation failed: %w", err)

	}



	// Validate intent path.

	if err := validateFilePath(intentPath); err != nil {

		return fmt.Errorf("intent path validation failed: %w", err)

	}



	// Load intent.

	intent, err := patchgen.LoadIntent(intentPath)

	if err != nil {

		return fmt.Errorf("failed to load intent: %w", err)

	}



	// Generate patch.

	generator := patchgen.NewPatchPackage(intent, outputDir)

	if err := generator.Generate(); err != nil {

		return fmt.Errorf("failed to generate patch: %w", err)

	}



	// Optionally apply using porch-direct.

	if apply {

		if err := applyWithPorchDirect(outputDir); err != nil {

			fmt.Printf("Warning: porch-direct failed: %v\n", err)

			fmt.Printf("You can manually apply with: porch-direct --package %s\n", outputDir)

		} else {

			fmt.Println("Patch applied successfully via porch-direct")

		}

	}



	return nil

}



// getAllowedBaseDirs returns a list of allowed base directories for the current platform.

func getAllowedBaseDirs() []string {

	if runtime.GOOS == "windows" {

		return []string{

			"C:\\temp",

			"C:\\tmp",

			os.TempDir(), // Allow system temp directory for tests

			".",

			".\\examples",

			".\\packages",

			".\\output",

		}

	}

	return []string{

		"/tmp",

		"/var/tmp",

		os.TempDir(), // Allow system temp directory for tests

		".",

		"./examples",

		"./packages",

		"./output",

	}

}



// validateFilePath validates file paths to prevent path traversal attacks.

func validateFilePath(filePath string) error {

	// Clean the path.

	cleanPath := filepath.Clean(filePath)



	// Convert to absolute path for better validation.

	absPath, err := filepath.Abs(cleanPath)

	if err != nil {

		return fmt.Errorf("failed to resolve absolute path: %w", err)

	}



	// Check for path traversal attempts in the original path.

	if strings.Contains(filePath, "..") {

		return fmt.Errorf("path traversal detected in file path: %s", filePath)

	}



	// Validate path format - stricter for file paths.

	validFilePattern := regexp.MustCompile(`^[a-zA-Z0-9._/\\:-]+$`)

	if !validFilePattern.MatchString(absPath) {

		return fmt.Errorf("invalid characters in file path: %s", filePath)

	}



	// Check if file exists and is readable.

	if _, err := os.Stat(absPath); os.IsNotExist(err) {

		return fmt.Errorf("file does not exist: %s", absPath)

	}



	return nil

}



// validateOutputDir validates and sanitizes the output directory path with security controls.

func validateOutputDir(outputDir string) error {

	// Clean the path to resolve any ".." or other unsafe elements.

	cleanPath := filepath.Clean(outputDir)



	// Convert to absolute path for proper validation.

	absPath, err := filepath.Abs(cleanPath)

	if err != nil {

		return fmt.Errorf("failed to resolve absolute path: %w", err)

	}



	// Check for path traversal attempts in the original path.

	if strings.Contains(outputDir, "..") {

		return fmt.Errorf("path traversal detected in output directory: %s", outputDir)

	}



	// Validate against whitelist of allowed base directories.

	allowedBaseDirs := getAllowedBaseDirs()

	isAllowed := false

	for _, allowedDir := range allowedBaseDirs {

		allowedAbs, err := filepath.Abs(allowedDir)

		if err != nil {

			continue

		}



		// Check if the path is within an allowed directory.

		if strings.HasPrefix(absPath, allowedAbs) {

			isAllowed = true

			break

		}

	}



	if !isAllowed {

		return fmt.Errorf("output directory %s is not within allowed base directories", outputDir)

	}



	// Validate path format (allow alphanumeric, hyphens, underscores, dots, and path separators).

	validPathPattern := regexp.MustCompile(`^[a-zA-Z0-9._/\\:-]+$`)

	if !validPathPattern.MatchString(absPath) {

		return fmt.Errorf("invalid characters in output directory path: %s", outputDir)

	}



	// Ensure the path is not empty after cleaning.

	if cleanPath == "" {

		return fmt.Errorf("output directory path cannot be empty")

	}



	// Additional security: prevent writing to system directories.

	systemDirs := []string{"/bin", "/boot", "/dev", "/etc", "/lib", "/proc", "/root", "/sys", "/usr/bin", "/usr/lib"}

	if runtime.GOOS == "windows" {

		systemDirs = []string{"C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)"}

	}



	for _, sysDir := range systemDirs {

		sysDirAbs, err := filepath.Abs(sysDir)

		if err != nil {

			continue

		}

		if strings.HasPrefix(absPath, sysDirAbs) {

			return fmt.Errorf("cannot write to system directory: %s", outputDir)

		}

	}



	return nil

}



// validateBinaryPath validates that the binary exists and is in an allowed location.

func validateBinaryPath(binaryName string) (string, error) {

	// Look up the binary in PATH.

	binaryPath, err := exec.LookPath(binaryName)

	if err != nil {

		return "", fmt.Errorf("binary %s not found in PATH: %w", binaryName, err)

	}



	// Get absolute path for validation.

	absBinaryPath, err := filepath.Abs(binaryPath)

	if err != nil {

		return "", fmt.Errorf("failed to resolve absolute path for binary: %w", err)

	}



	// Basic security: ensure it's not in a suspicious location.

	suspiciousPaths := []string{"/tmp", "/var/tmp", "temp", "Temp"}

	for _, suspicious := range suspiciousPaths {

		if strings.Contains(strings.ToLower(absBinaryPath), strings.ToLower(suspicious)) {

			return "", fmt.Errorf("binary in suspicious location: %s", absBinaryPath)

		}

	}



	return absBinaryPath, nil

}



// applyWithPorchDirect securely executes porch-direct command with enhanced security controls.

func applyWithPorchDirect(outputDir string) error {

	// Validate the output directory to prevent command injection.

	if err := validateOutputDir(outputDir); err != nil {

		return fmt.Errorf("output directory validation failed: %w", err)

	}



	// Validate the binary path.

	binaryPath, err := validateBinaryPath("porch-direct")

	if err != nil {

		return fmt.Errorf("binary validation failed: %w", err)

	}



	// Clean the path to prevent injection.

	cleanOutputDir := filepath.Clean(outputDir)



	fmt.Printf("Calling porch-direct (%s) to apply patch...\n", binaryPath)



	// Create a context with timeout to prevent hanging.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	defer cancel()



	// Create command with context for timeout support.

	// Use the validated binary path directly to prevent PATH manipulation.

	cmd := exec.CommandContext(ctx, binaryPath, "--package", cleanOutputDir)



	// Set up secure environment - inherit minimal environment.

	cmd.Env = []string{

		"PATH=" + os.Getenv("PATH"),

		"HOME=" + os.Getenv("HOME"),

		"USER=" + os.Getenv("USER"),

	}



	// Set working directory to a safe location.

	if wd, err := os.Getwd(); err == nil {

		cmd.Dir = wd

	}



	// Capture output for better error reporting.

	cmd.Stdout = os.Stdout

	cmd.Stderr = os.Stderr



	// Log command execution for audit trail.

	fmt.Printf("Executing: %s --package %s\n", binaryPath, cleanOutputDir)



	// Execute with timeout.

	if err := cmd.Run(); err != nil {

		// Check if it was a timeout.

		if ctx.Err() == context.DeadlineExceeded {

			return fmt.Errorf("porch-direct command timed out after 5 minutes")

		}



		// Check for specific error types.

		if exitError, ok := err.(*exec.ExitError); ok {

			return fmt.Errorf("porch-direct command failed with exit code %d: %w", exitError.ExitCode(), err)

		}



		return fmt.Errorf("porch-direct command failed: %w", err)

	}



	fmt.Println("Patch applied successfully")

	return nil

}

