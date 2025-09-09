<<<<<<< HEAD
// Package writer provides filesystem writing utilities for KRM packages.
=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
package writer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thc1006/nephoran-intent-operator/internal/generator"
)

<<<<<<< HEAD
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
=======
// FilesystemWriter handles idempotent writing of KRM packages to the filesystem
type FilesystemWriter struct {
	baseDir string
	dryRun  bool
}

// WriteResult contains the result of a write operation
type WriteResult struct {
	PackagePath   string
	FilesWritten  []string
	FilesSkipped  []string
	FilesUpdated  []string
	TotalFiles    int
	WasIdempotent bool
}

// NewFilesystemWriter creates a new filesystem writer
func NewFilesystemWriter(baseDir string, dryRun bool) *FilesystemWriter {
	return &FilesystemWriter{
		baseDir: baseDir,
		dryRun:  dryRun,
	}
}

// WritePackage writes a KRM package to the filesystem idempotently
func (w *FilesystemWriter) WritePackage(pkg *generator.Package) (*WriteResult, error) {
	result := &WriteResult{
		PackagePath:   pkg.Directory,
		FilesWritten:  make([]string, 0),
		FilesSkipped:  make([]string, 0),
		FilesUpdated:  make([]string, 0),
		TotalFiles:    pkg.GetFileCount(),
		WasIdempotent: true,
	}

	// Ensure package directory exists
	if !w.dryRun {
		if err := os.MkdirAll(pkg.Directory, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			return result, fmt.Errorf("failed to create package directory %s: %w", pkg.Directory, err)
		}
	}

<<<<<<< HEAD
	// Process each file.

	for _, file := range pkg.GetPackageFiles() {

		filePath := filepath.Join(pkg.Directory, file.Path)

=======
	// Process each file
	for _, file := range pkg.GetPackageFiles() {
		filePath := filepath.Join(pkg.Directory, file.Path)
		
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		action, err := w.writeFile(filePath, file.Content)
		if err != nil {
			return result, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}

		switch action {
<<<<<<< HEAD

		case "written":

			result.FilesWritten = append(result.FilesWritten, file.Path)

			result.WasIdempotent = false

		case "updated":

			result.FilesUpdated = append(result.FilesUpdated, file.Path)

			result.WasIdempotent = false

		case "skipped":

			result.FilesSkipped = append(result.FilesSkipped, file.Path)

		}

=======
		case "written":
			result.FilesWritten = append(result.FilesWritten, file.Path)
			result.WasIdempotent = false
		case "updated":
			result.FilesUpdated = append(result.FilesUpdated, file.Path)
			result.WasIdempotent = false
		case "skipped":
			result.FilesSkipped = append(result.FilesSkipped, file.Path)
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	return result, nil
}

<<<<<<< HEAD
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

=======
// writeFile writes a single file idempotently
func (w *FilesystemWriter) writeFile(filePath string, content []byte) (string, error) {
	// Calculate content hash for comparison
	newHash := sha256.Sum256(content)

	// Check if file exists
	existingContent, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, write it
			if w.dryRun {
				return "written", nil
			}
			
			// Ensure directory exists
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}
			return "written", nil
		}
		return "", fmt.Errorf("failed to read existing file: %w", err)
	}

	// Compare content hashes
	existingHash := sha256.Sum256(existingContent)
	if newHash == existingHash {
		// Content is identical, skip writing
		return "skipped", nil
	}

	// Content differs, update the file
	if w.dryRun {
		return "updated", nil
	}
	
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to update file: %w", err)
	}
	return "updated", nil
}

// CleanupPackage removes a package directory if it exists
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (w *FilesystemWriter) CleanupPackage(packageDir string) error {
	if w.dryRun {
		return nil
	}
<<<<<<< HEAD

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

=======
	
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}
	
	return os.RemoveAll(packageDir)
}

// ListPackages returns all package directories in the base directory
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
			// Check if it contains a Kptfile
			kptfilePath := filepath.Join(w.baseDir, entry.Name(), "Kptfile")
			if _, err := os.Stat(kptfilePath); err == nil {
				packages = append(packages, entry.Name())
			}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	}

	return packages, nil
}

<<<<<<< HEAD
// ValidatePackage checks if a package directory contains valid KRM files.

func (w *FilesystemWriter) ValidatePackage(packageDir string) error {
	// Check if Kptfile exists.

	kptfilePath := filepath.Join(packageDir, "Kptfile")

=======
// ValidatePackage checks if a package directory contains valid KRM files
func (w *FilesystemWriter) ValidatePackage(packageDir string) error {
	// Check if Kptfile exists
	kptfilePath := filepath.Join(packageDir, "Kptfile")
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if _, err := os.Stat(kptfilePath); os.IsNotExist(err) {
		return fmt.Errorf("package is missing Kptfile")
	}

<<<<<<< HEAD
	// Check if at least one YAML file exists.

=======
	// Check if at least one YAML file exists
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return fmt.Errorf("failed to read package directory: %w", err)
	}

	hasYAML := false
<<<<<<< HEAD

	for _, entry := range entries {
		if !entry.IsDir() {

			ext := filepath.Ext(entry.Name())

			if ext == ".yaml" || ext == ".yml" {

				hasYAML = true

				break

			}

=======
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if ext == ".yaml" || ext == ".yml" {
				hasYAML = true
				break
			}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	}

	if !hasYAML {
		return fmt.Errorf("package contains no YAML files")
	}

	return nil
}

<<<<<<< HEAD
// GetBaseDir returns the base directory for package output.

=======
// GetBaseDir returns the base directory for package output
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (w *FilesystemWriter) GetBaseDir() string {
	return w.baseDir
}

<<<<<<< HEAD
// IsDryRun returns whether this writer is in dry-run mode.

=======
// IsDryRun returns whether this writer is in dry-run mode
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (w *FilesystemWriter) IsDryRun() bool {
	return w.dryRun
}

<<<<<<< HEAD
// SetDryRun enables or disables dry-run mode.

func (w *FilesystemWriter) SetDryRun(dryRun bool) {
	w.dryRun = dryRun
}
=======
// SetDryRun enables or disables dry-run mode
func (w *FilesystemWriter) SetDryRun(dryRun bool) {
	w.dryRun = dryRun
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
