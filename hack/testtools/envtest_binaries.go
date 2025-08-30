// Package testtools provides envtest binary management utilities.

package testtools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// EnvtestBinaryManager manages kubebuilder assets for envtest.

type EnvtestBinaryManager struct {
	kubernetesVersion string

	binDirectory string

	cacheDirectory string

	platform string

	arch string
}

// NewEnvtestBinaryManager creates a new binary manager.

func NewEnvtestBinaryManager() *EnvtestBinaryManager {

	return &EnvtestBinaryManager{

		kubernetesVersion: getEnvOrDefault("ENVTEST_K8S_VERSION", "1.28.0"),

		binDirectory: getEnvOrDefault("ENVTEST_BIN_DIR", "/usr/local/kubebuilder/bin"),

		cacheDirectory: getEnvOrDefault("CACHE_DIR", filepath.Join(os.Getenv("HOME"), ".cache", "kubebuilder-envtest")),

		platform: runtime.GOOS,

		arch: runtime.GOARCH,
	}

}

// EnsureBinariesInstalled checks if required binaries are available and installs them if needed.

func (ebm *EnvtestBinaryManager) EnsureBinariesInstalled(ctx context.Context) error {

	// Check if binaries already exist.

	if ebm.binariesExist() {

		return nil

	}

	fmt.Printf("envtest binaries not found, installing for Kubernetes %s...\n", ebm.kubernetesVersion)

	// Try using setup-envtest first.

	if err := ebm.installWithSetupEnvtest(ctx); err != nil {

		fmt.Printf("setup-envtest installation failed: %v, trying manual installation...\n", err)

		return ebm.installManually(ctx)

	}

	return nil

}

// binariesExist checks if all required binaries are present.

func (ebm *EnvtestBinaryManager) binariesExist() bool {

	requiredBinaries := []string{"etcd", "kube-apiserver", "kubectl"}

	for _, binary := range requiredBinaries {

		binaryPath := filepath.Join(ebm.binDirectory, binary)

		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {

			return false

		}

		// Check if binary is executable.

		if !ebm.isBinaryExecutable(binaryPath) {

			return false

		}

	}

	return true

}

// isBinaryExecutable checks if a binary is executable.

func (ebm *EnvtestBinaryManager) isBinaryExecutable(path string) bool {

	info, err := os.Stat(path)

	if err != nil {

		return false

	}

	// Check if file has execute permission.

	return info.Mode()&0o111 != 0

}

// installWithSetupEnvtest uses the setup-envtest tool to install binaries.

func (ebm *EnvtestBinaryManager) installWithSetupEnvtest(ctx context.Context) error {

	// Install setup-envtest if not present.

	if err := ebm.installSetupEnvtest(ctx); err != nil {

		return fmt.Errorf("failed to install setup-envtest: %w", err)

	}

	// Use setup-envtest to download assets.

	cmd := exec.CommandContext(ctx, "setup-envtest", "use", ebm.kubernetesVersion,

		"--bin-dir", ebm.cacheDirectory, "-p", "path")

	output, err := cmd.Output()

	if err != nil {

		return fmt.Errorf("setup-envtest failed: %w", err)

	}

	assetsPath := strings.TrimSpace(string(output))

	if assetsPath == "" {

		return fmt.Errorf("setup-envtest returned empty path")

	}

	// Copy binaries to target directory.

	return ebm.copyBinaries(assetsPath)

}

// installSetupEnvtest installs the setup-envtest tool.

func (ebm *EnvtestBinaryManager) installSetupEnvtest(ctx context.Context) error {

	// Check if already installed.

	if _, err := exec.LookPath("setup-envtest"); err == nil {

		return nil

	}

	fmt.Println("Installing setup-envtest tool...")

	cmd := exec.CommandContext(ctx, "go", "install",

		"sigs.k8s.io/controller-runtime/tools/setup-envtest@latest")

	if err := cmd.Run(); err != nil {

		return fmt.Errorf("failed to install setup-envtest: %w", err)

	}

	return nil

}

// installManually installs binaries by downloading them directly.

func (ebm *EnvtestBinaryManager) installManually(ctx context.Context) error {

	// Create cache directory.

	if err := os.MkdirAll(ebm.cacheDirectory, 0o755); err != nil {

		return fmt.Errorf("failed to create cache directory: %w", err)

	}

	// Download tarball.

	tarball := fmt.Sprintf("kubebuilder-tools-%s-%s-%s.tar.gz",

		ebm.kubernetesVersion, ebm.platform, ebm.arch)

	downloadURL := fmt.Sprintf("https://go.kubebuilder.io/test-tools/%s/%s/%s/%s",

		ebm.kubernetesVersion, ebm.platform, ebm.arch, tarball)

	tarballPath := filepath.Join(ebm.cacheDirectory, tarball)

	fmt.Printf("Downloading %s...\n", downloadURL)

	if err := ebm.downloadFile(ctx, downloadURL, tarballPath); err != nil {

		return fmt.Errorf("failed to download tarball: %w", err)

	}

	// Extract tarball.

	fmt.Printf("Extracting %s...\n", tarball)

	if err := ebm.extractTarball(ctx, tarballPath, ebm.cacheDirectory); err != nil {

		return fmt.Errorf("failed to extract tarball: %w", err)

	}

	// Copy binaries.

	assetsPath := filepath.Join(ebm.cacheDirectory, "kubebuilder", "bin")

	return ebm.copyBinaries(assetsPath)

}

// downloadFile downloads a file from a URL.

func (ebm *EnvtestBinaryManager) downloadFile(ctx context.Context, url, dest string) error {

	var cmd *exec.Cmd

	if _, err := exec.LookPath("curl"); err == nil {

		cmd = exec.CommandContext(ctx, "curl", "-L", url, "-o", dest)

	} else if _, err := exec.LookPath("wget"); err == nil {

		cmd = exec.CommandContext(ctx, "wget", url, "-O", dest)

	} else {

		return fmt.Errorf("neither curl nor wget found")

	}

	return cmd.Run()

}

// extractTarball extracts a tarball.

func (ebm *EnvtestBinaryManager) extractTarball(ctx context.Context, tarball, dest string) error {

	cmd := exec.CommandContext(ctx, "tar", "-xzf", tarball, "-C", dest)

	return cmd.Run()

}

// copyBinaries copies binaries from source to target directory.

func (ebm *EnvtestBinaryManager) copyBinaries(sourcePath string) error {

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {

		return fmt.Errorf("source path %s does not exist", sourcePath)

	}

	// Create target directory with proper permissions.

	if err := os.MkdirAll(ebm.binDirectory, 0o755); err != nil {

		// Try with sudo if regular creation fails (for system directories).

		if strings.Contains(ebm.binDirectory, "/usr/local") {

			cmd := exec.Command("sudo", "mkdir", "-p", ebm.binDirectory)

			if err := cmd.Run(); err != nil {

				return fmt.Errorf("failed to create directory %s: %w", ebm.binDirectory, err)

			}

		} else {

			return fmt.Errorf("failed to create directory %s: %w", ebm.binDirectory, err)

		}

	}

	// Copy files.

	requiredBinaries := []string{"etcd", "kube-apiserver", "kubectl"}

	for _, binary := range requiredBinaries {

		srcPath := filepath.Join(sourcePath, binary)

		destPath := filepath.Join(ebm.binDirectory, binary)

		if err := ebm.copyFile(srcPath, destPath); err != nil {

			return fmt.Errorf("failed to copy %s: %w", binary, err)

		}

		// Make executable.

		if err := os.Chmod(destPath, 0o755); err != nil {

			// Try with sudo if regular chmod fails.

			cmd := exec.Command("sudo", "chmod", "+x", destPath)

			if err := cmd.Run(); err != nil {

				return fmt.Errorf("failed to make %s executable: %w", binary, err)

			}

		}

	}

	return nil

}

// copyFile copies a file from src to dest.

func (ebm *EnvtestBinaryManager) copyFile(src, dest string) error {

	// Try regular copy first.

	if err := ebm.regularCopy(src, dest); err == nil {

		return nil

	}

	// Try with sudo if regular copy fails (for system directories).

	if strings.Contains(dest, "/usr/local") {

		cmd := exec.Command("sudo", "cp", src, dest)

		return cmd.Run()

	}

	return fmt.Errorf("failed to copy %s to %s", src, dest)

}

// regularCopy performs a regular file copy.

func (ebm *EnvtestBinaryManager) regularCopy(src, dest string) error {

	sourceFile, err := os.Open(src)

	if err != nil {

		return err

	}

	defer sourceFile.Close()

	destFile, err := os.Create(dest)

	if err != nil {

		return err

	}

	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)

	return err

}

// GetBinaryPath returns the path to a specific binary.

func (ebm *EnvtestBinaryManager) GetBinaryPath(binary string) string {

	return filepath.Join(ebm.binDirectory, binary)

}

// GetBinaryDirectory returns the binary directory path.

func (ebm *EnvtestBinaryManager) GetBinaryDirectory() string {

	return ebm.binDirectory

}

// SetupEnvironment sets up the KUBEBUILDER_ASSETS environment variable.

func (ebm *EnvtestBinaryManager) SetupEnvironment() {

	os.Setenv("KUBEBUILDER_ASSETS", ebm.binDirectory)

}

// ValidateInstallation validates that all required binaries are installed and working.

func (ebm *EnvtestBinaryManager) ValidateInstallation(ctx context.Context) error {

	requiredBinaries := []string{"etcd", "kube-apiserver", "kubectl"}

	for _, binary := range requiredBinaries {

		binaryPath := ebm.GetBinaryPath(binary)

		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {

			return fmt.Errorf("%s not found at %s", binary, binaryPath)

		}

		if !ebm.isBinaryExecutable(binaryPath) {

			return fmt.Errorf("%s is not executable", binary)

		}

		// Quick version check with timeout.

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "--version")

		if err := cmd.Run(); err != nil {

			// Some binaries might not support --version, try --help.

			cmd = exec.CommandContext(ctx, binaryPath, "--help")

			if err := cmd.Run(); err != nil {

				// etcd might need version command instead of --version.

				if binary == "etcd" {

					cmd = exec.CommandContext(ctx, binaryPath, "version")

					if err := cmd.Run(); err != nil {

						fmt.Printf("Warning: %s may not be fully functional: %v\n", binary, err)

					}

				} else {

					fmt.Printf("Warning: %s may not be fully functional: %v\n", binary, err)

				}

			}

		}

	}

	return nil

}

// getEnvOrDefault returns environment variable value or default.

func getEnvOrDefault(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {

		return value

	}

	return defaultValue

}
