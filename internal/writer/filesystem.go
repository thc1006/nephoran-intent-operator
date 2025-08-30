// Package writer provides filesystem writing utilities for KRM packages.
package writer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nephio-project/nephoran-intent-operator/internal/generator"
)

// FilesystemWriter handles idempotent writing of KRM packages to the filesystem.

type FilesystemWriter struct {
	baseDir string

	dryRun bool
}

// WriteResult contains the result of a write operation.

type WriteResult struct {
	PackagePath string

	FilesWritten []string

	FilesSkipped []string

	FilesUpdated []string

	TotalFiles int

	WasIdempotent bool
}

// NewFilesystemWriter creates a new filesystem writer.

func NewFilesystemWriter(baseDir string, dryRun bool) *FilesystemWriter {

	return &FilesystemWriter{

		baseDir: baseDir,

		dryRun: dryRun,
	}

}

// WritePackage writes a KRM package to the filesystem idempotently.

func (w *FilesystemWriter) WritePackage(pkg *generator.Package) (*WriteResult, error) {

	result := &WriteResult{

		PackagePath: pkg.Directory,

		FilesWritten: make([]string, 0),

		FilesSkipped: make([]string, 0),

		FilesUpdated: make([]string, 0),

		TotalFiles: pkg.GetFileCount(),

		WasIdempotent: true,
	}

	// Ensure package directory exists.

	if !w.dryRun {

		if err := os.MkdirAll(pkg.Directory, 0o755); err != nil {

			return result, fmt.Errorf("failed to create package directory %s: %w", pkg.Directory, err)

		}

	}

	// Process each file.

	for _, file := range pkg.GetPackageFiles() {

		filePath := filepath.Join(pkg.Directory, file.Path)

		action, err := w.writeFile(filePath, file.Content)

		if err != nil {

			return result, fmt.Errorf("failed to write file %s: %w", filePath, err)

		}

		switch action {

		case "written":

			result.FilesWritten = append(result.FilesWritten, file.Path)

			result.WasIdempotent = false

		case "updated":

			result.FilesUpdated = append(result.FilesUpdated, file.Path)

			result.WasIdempotent = false

		case "skipped":

			result.FilesSkipped = append(result.FilesSkipped, file.Path)

		}

	}

	return result, nil

}

// writeFile writes a single file idempotently.

func (w *FilesystemWriter) writeFile(filePath string, content []byte) (string, error) {

	// Calculate content hash for comparison.

	newHash := sha256.Sum256(content)

	// Check if file exists.

	existingContent, err := os.ReadFile(filePath)

	if err != nil {

		if os.IsNotExist(err) {

			// File doesn't exist, write it.

			if w.dryRun {

				return "written", nil

			}

			// Ensure directory exists.

			dir := filepath.Dir(filePath)

			if err := os.MkdirAll(dir, 0o755); err != nil {

				return "", fmt.Errorf("failed to create directory %s: %w", dir, err)

			}

			if err := os.WriteFile(filePath, content, 0o640); err != nil {

				return "", fmt.Errorf("failed to write file: %w", err)

			}

			return "written", nil

		}

		return "", fmt.Errorf("failed to read existing file: %w", err)

	}

	// Compare content hashes.

	existingHash := sha256.Sum256(existingContent)

	if newHash == existingHash {

		// Content is identical, skip writing.

		return "skipped", nil

	}

	// Content differs, update the file.

	if w.dryRun {

		return "updated", nil

	}

	if err := os.WriteFile(filePath, content, 0o640); err != nil {

		return "", fmt.Errorf("failed to update file: %w", err)

	}

	return "updated", nil

}

// CleanupPackage removes a package directory if it exists.

func (w *FilesystemWriter) CleanupPackage(packageDir string) error {

	if w.dryRun {

		return nil

	}

	if _, err := os.Stat(packageDir); os.IsNotExist(err) {

		return nil // Directory doesn't exist, nothing to clean

	}

	return os.RemoveAll(packageDir)

}

// ListPackages returns all package directories in the base directory.

func (w *FilesystemWriter) ListPackages() ([]string, error) {

	entries, err := os.ReadDir(w.baseDir)

	if err != nil {

		if os.IsNotExist(err) {

			return []string{}, nil

		}

		return nil, fmt.Errorf("failed to read base directory %s: %w", w.baseDir, err)

	}

	var packages []string

	for _, entry := range entries {

		if entry.IsDir() {

			// Check if it contains a Kptfile.

			kptfilePath := filepath.Join(w.baseDir, entry.Name(), "Kptfile")

			if _, err := os.Stat(kptfilePath); err == nil {

				packages = append(packages, entry.Name())

			}

		}

	}

	return packages, nil

}

// ValidatePackage checks if a package directory contains valid KRM files.

func (w *FilesystemWriter) ValidatePackage(packageDir string) error {

	// Check if Kptfile exists.

	kptfilePath := filepath.Join(packageDir, "Kptfile")

	if _, err := os.Stat(kptfilePath); os.IsNotExist(err) {

		return fmt.Errorf("package is missing Kptfile")

	}

	// Check if at least one YAML file exists.

	entries, err := os.ReadDir(packageDir)

	if err != nil {

		return fmt.Errorf("failed to read package directory: %w", err)

	}

	hasYAML := false

	for _, entry := range entries {

		if !entry.IsDir() {

			ext := filepath.Ext(entry.Name())

			if ext == ".yaml" || ext == ".yml" {

				hasYAML = true

				break

			}

		}

	}

	if !hasYAML {

		return fmt.Errorf("package contains no YAML files")

	}

	return nil

}

// GetBaseDir returns the base directory for package output.

func (w *FilesystemWriter) GetBaseDir() string {

	return w.baseDir

}

// IsDryRun returns whether this writer is in dry-run mode.

func (w *FilesystemWriter) IsDryRun() bool {

	return w.dryRun

}

// SetDryRun enables or disables dry-run mode.

func (w *FilesystemWriter) SetDryRun(dryRun bool) {

	w.dryRun = dryRun

}
