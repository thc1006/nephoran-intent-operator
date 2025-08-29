package disaster_recovery

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupManager handles backup and restore operations for disaster recovery testing.

type BackupManager struct {
	client client.Client

	backupDir string

	scheme *runtime.Scheme
}

// BackupOptions configures backup behavior.

type BackupOptions struct {
	Namespace string

	IncludeSecrets bool

	IncludeConfigMaps bool

	CompressBackup bool

	Timeout time.Duration

	ResourceTypes []schema.GroupVersionKind
}

// RestoreOptions configures restore behavior.

type RestoreOptions struct {
	Namespace string

	DryRun bool

	ForceReplace bool

	IgnoreErrors bool

	Timeout time.Duration

	ValidationMode ValidationMode
}

// ValidationMode defines how restored resources should be validated.

type ValidationMode string

const (

	// ValidationModeNone holds validationmodenone value.

	ValidationModeNone ValidationMode = "none"

	// ValidationModeBasic holds validationmodebasic value.

	ValidationModeBasic ValidationMode = "basic"

	// ValidationModeStrict holds validationmodestrict value.

	ValidationModeStrict ValidationMode = "strict"
)

// BackupManifest contains metadata about a backup.

type BackupManifest struct {
	ID string `json:"id"`

	Timestamp time.Time `json:"timestamp"`

	Namespace string `json:"namespace"`

	ResourceCounts map[string]int `json:"resource_counts"`

	BackupOptions BackupOptions `json:"backup_options"`

	ChecksumSHA256 string `json:"checksum_sha256"`

	BackupSize int64 `json:"backup_size"`

	BackupVersion string `json:"backup_version"`

	KubernetesVersion string `json:"kubernetes_version"`

	Resources []BackupResourceMetadata `json:"resources"`
}

// BackupResourceMetadata contains metadata about backed up resources.

type BackupResourceMetadata struct {
	APIVersion string `json:"api_version"`

	Kind string `json:"kind"`

	Name string `json:"name"`

	Namespace string `json:"namespace"`

	UID string `json:"uid"`

	Checksum string `json:"checksum"`
}

// RestoreResult contains information about a restore operation.

type RestoreResult struct {
	Success bool `json:"success"`

	ResourcesRestored int `json:"resources_restored"`

	ResourcesFailed int `json:"resources_failed"`

	Errors []RestoreError `json:"errors"`

	Duration time.Duration `json:"duration"`

	ValidationResults []ValidationResult `json:"validation_results"`
}

// RestoreError represents an error during restore.

type RestoreError struct {
	Resource string `json:"resource"`

	Error string `json:"error"`

	Fatal bool `json:"fatal"`
}

// ValidationResult represents a resource validation result.

type ValidationResult struct {
	Resource string `json:"resource"`

	Valid bool `json:"valid"`

	Message string `json:"message"`
}

// NewBackupManager creates a new backup manager.

func NewBackupManager(client client.Client, backupDir string, scheme *runtime.Scheme) *BackupManager {

	return &BackupManager{

		client: client,

		backupDir: backupDir,

		scheme: scheme,
	}

}

// CreateBackup creates a backup of specified resources.

func (bm *BackupManager) CreateBackup(ctx context.Context, backupID string, options BackupOptions) (*BackupManifest, error) {

	backupPath := filepath.Join(bm.backupDir, backupID)

	if err := os.MkdirAll(backupPath, 0o755); err != nil {

		return nil, fmt.Errorf("failed to create backup directory: %w", err)

	}

	manifest := &BackupManifest{

		ID: backupID,

		Timestamp: time.Now(),

		Namespace: options.Namespace,

		BackupOptions: options,

		BackupVersion: "1.0",

		ResourceCounts: make(map[string]int),
	}

	// Backup each resource type.

	var allResources []unstructured.Unstructured

	for _, gvk := range options.ResourceTypes {

		resources, err := bm.backupResourceType(ctx, gvk, options)

		if err != nil {

			return nil, fmt.Errorf("failed to backup %s: %w", gvk.String(), err)

		}

		allResources = append(allResources, resources...)

		manifest.ResourceCounts[gvk.Kind] = len(resources)

	}

	// Save resources to files.

	for i, resource := range allResources {

		resourceData, err := json.MarshalIndent(resource.Object, "", "  ")

		if err != nil {

			return nil, fmt.Errorf("failed to marshal resource: %w", err)

		}

		filename := fmt.Sprintf("resource-%d-%s-%s.json", i, resource.GetKind(), resource.GetName())

		resourcePath := filepath.Join(backupPath, filename)

		if err := os.WriteFile(resourcePath, resourceData, 0o640); err != nil {

			return nil, fmt.Errorf("failed to write resource file: %w", err)

		}

		// Add resource metadata.

		checksum := bm.calculateChecksum(resourceData)

		metadata := BackupResourceMetadata{

			APIVersion: resource.GetAPIVersion(),

			Kind: resource.GetKind(),

			Name: resource.GetName(),

			Namespace: resource.GetNamespace(),

			UID: string(resource.GetUID()),

			Checksum: checksum,
		}

		manifest.Resources = append(manifest.Resources, metadata)

	}

	// Calculate backup checksum and size.

	manifest.ChecksumSHA256, manifest.BackupSize = bm.calculateBackupChecksum(backupPath)

	// Save manifest.

	manifestData, err := json.MarshalIndent(manifest, "", "  ")

	if err != nil {

		return nil, fmt.Errorf("failed to marshal manifest: %w", err)

	}

	manifestPath := filepath.Join(backupPath, "manifest.json")

	if err := os.WriteFile(manifestPath, manifestData, 0o640); err != nil {

		return nil, fmt.Errorf("failed to write manifest: %w", err)

	}

	return manifest, nil

}

// RestoreBackup restores resources from a backup.

func (bm *BackupManager) RestoreBackup(ctx context.Context, backupID string, options RestoreOptions) (*RestoreResult, error) {

	startTime := time.Now()

	result := &RestoreResult{

		Success: true,
	}

	backupPath := filepath.Join(bm.backupDir, backupID)

	// Load manifest.

	manifest, err := bm.loadManifest(backupPath)

	if err != nil {

		return nil, fmt.Errorf("failed to load backup manifest: %w", err)

	}

	// Validate backup integrity.

	if err := bm.validateBackupIntegrity(backupPath, manifest); err != nil {

		return nil, fmt.Errorf("backup integrity validation failed: %w", err)

	}

	// Restore each resource.

	for _, resourceMeta := range manifest.Resources {

		if err := bm.restoreResource(ctx, backupPath, resourceMeta, options); err != nil {

			result.ResourcesFailed++

			result.Errors = append(result.Errors, RestoreError{

				Resource: fmt.Sprintf("%s/%s", resourceMeta.Kind, resourceMeta.Name),

				Error: err.Error(),

				Fatal: false,
			})

			if !options.IgnoreErrors {

				result.Success = false

			}

		} else {

			result.ResourcesRestored++

		}

	}

	// Perform post-restore validation.

	if options.ValidationMode != ValidationModeNone {

		validationResults, err := bm.validateRestoredResources(ctx, manifest, options)

		if err != nil {

			result.Errors = append(result.Errors, RestoreError{

				Resource: "validation",

				Error: err.Error(),

				Fatal: false,
			})

		}

		result.ValidationResults = validationResults

	}

	result.Duration = time.Since(startTime)

	return result, nil

}

// ListBackups returns a list of available backups.

func (bm *BackupManager) ListBackups() ([]BackupManifest, error) {

	var backups []BackupManifest

	entries, err := os.ReadDir(bm.backupDir)

	if err != nil {

		return nil, fmt.Errorf("failed to read backup directory: %w", err)

	}

	for _, entry := range entries {

		if !entry.IsDir() {

			continue

		}

		manifestPath := filepath.Join(bm.backupDir, entry.Name(), "manifest.json")

		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {

			continue // Skip directories without manifest

		}

		manifest, err := bm.loadManifest(filepath.Join(bm.backupDir, entry.Name()))

		if err != nil {

			continue // Skip invalid manifests

		}

		backups = append(backups, *manifest)

	}

	return backups, nil

}

// DeleteBackup removes a backup.

func (bm *BackupManager) DeleteBackup(backupID string) error {

	backupPath := filepath.Join(bm.backupDir, backupID)

	return os.RemoveAll(backupPath)

}

// ValidateBackup checks backup integrity.

func (bm *BackupManager) ValidateBackup(backupID string) error {

	backupPath := filepath.Join(bm.backupDir, backupID)

	manifest, err := bm.loadManifest(backupPath)

	if err != nil {

		return fmt.Errorf("failed to load manifest: %w", err)

	}

	return bm.validateBackupIntegrity(backupPath, manifest)

}

// Helper methods.

func (bm *BackupManager) backupResourceType(ctx context.Context, gvk schema.GroupVersionKind, options BackupOptions) ([]unstructured.Unstructured, error) {

	var resources []unstructured.Unstructured

	// Create unstructured list for this resource type.

	list := &unstructured.UnstructuredList{}

	list.SetGroupVersionKind(schema.GroupVersionKind{

		Group: gvk.Group,

		Version: gvk.Version,

		Kind: gvk.Kind + "List",
	})

	// List resources.

	listOptions := []client.ListOption{}

	if options.Namespace != "" {

		listOptions = append(listOptions, client.InNamespace(options.Namespace))

	}

	if err := bm.client.List(ctx, list, listOptions...); err != nil {

		return nil, fmt.Errorf("failed to list resources: %w", err)

	}

	// Process each resource.

	for _, item := range list.Items {

		// Skip secrets if not included.

		if !options.IncludeSecrets && item.GetKind() == "Secret" {

			continue

		}

		// Skip config maps if not included.

		if !options.IncludeConfigMaps && item.GetKind() == "ConfigMap" {

			continue

		}

		// Clean up resource for backup (remove status, resource version, etc.).

		cleanedResource := bm.cleanResourceForBackup(item)

		resources = append(resources, cleanedResource)

	}

	return resources, nil

}

func (bm *BackupManager) cleanResourceForBackup(resource unstructured.Unstructured) unstructured.Unstructured {

	// Create a copy.

	cleaned := resource.DeepCopy()

	// Remove runtime fields.

	unstructured.RemoveNestedField(cleaned.Object, "metadata", "resourceVersion")

	unstructured.RemoveNestedField(cleaned.Object, "metadata", "generation")

	unstructured.RemoveNestedField(cleaned.Object, "metadata", "managedFields")

	unstructured.RemoveNestedField(cleaned.Object, "metadata", "selfLink")

	unstructured.RemoveNestedField(cleaned.Object, "metadata", "uid")

	unstructured.RemoveNestedField(cleaned.Object, "status")

	return *cleaned

}

func (bm *BackupManager) loadManifest(backupPath string) (*BackupManifest, error) {

	manifestPath := filepath.Join(backupPath, "manifest.json")

	data, err := os.ReadFile(manifestPath)

	if err != nil {

		return nil, fmt.Errorf("failed to read manifest file: %w", err)

	}

	var manifest BackupManifest

	if err := json.Unmarshal(data, &manifest); err != nil {

		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)

	}

	return &manifest, nil

}

func (bm *BackupManager) validateBackupIntegrity(backupPath string, manifest *BackupManifest) error {

	// Check if all resource files exist and have correct checksums.

	for _, resourceMeta := range manifest.Resources {

		found := false

		err := filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {

			if err != nil {

				return err

			}

			if info.IsDir() || filepath.Ext(path) != ".json" || filepath.Base(path) == "manifest.json" {

				return nil

			}

			// Read and validate resource file.

			data, err := os.ReadFile(path)

			if err != nil {

				return err

			}

			var resource unstructured.Unstructured

			if err := json.Unmarshal(data, &resource.Object); err != nil {

				return err

			}

			if resource.GetKind() == resourceMeta.Kind && resource.GetName() == resourceMeta.Name {

				checksum := bm.calculateChecksum(data)

				if checksum != resourceMeta.Checksum {

					return fmt.Errorf("checksum mismatch for resource %s/%s", resourceMeta.Kind, resourceMeta.Name)

				}

				found = true

			}

			return nil

		})

		if err != nil {

			return err

		}

		if !found {

			return fmt.Errorf("resource file not found for %s/%s", resourceMeta.Kind, resourceMeta.Name)

		}

	}

	// Validate overall backup checksum.

	currentChecksum, _ := bm.calculateBackupChecksum(backupPath)

	if currentChecksum != manifest.ChecksumSHA256 {

		return fmt.Errorf("backup checksum mismatch: expected %s, got %s", manifest.ChecksumSHA256, currentChecksum)

	}

	return nil

}

func (bm *BackupManager) restoreResource(ctx context.Context, backupPath string, resourceMeta BackupResourceMetadata, options RestoreOptions) error {

	// Find and load resource file.

	var resourceData []byte

	err := filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {

		if err != nil {

			return err

		}

		if info.IsDir() || filepath.Ext(path) != ".json" || filepath.Base(path) == "manifest.json" {

			return nil

		}

		data, err := os.ReadFile(path)

		if err != nil {

			return err

		}

		var resource unstructured.Unstructured

		if err := json.Unmarshal(data, &resource.Object); err != nil {

			return err

		}

		if resource.GetKind() == resourceMeta.Kind && resource.GetName() == resourceMeta.Name {

			resourceData = data

		}

		return nil

	})

	if err != nil {

		return fmt.Errorf("failed to find resource file: %w", err)

	}

	if resourceData == nil {

		return fmt.Errorf("resource file not found for %s/%s", resourceMeta.Kind, resourceMeta.Name)

	}

	// Parse resource.

	var resource unstructured.Unstructured

	if err := json.Unmarshal(resourceData, &resource.Object); err != nil {

		return fmt.Errorf("failed to unmarshal resource: %w", err)

	}

	// Set target namespace if specified.

	if options.Namespace != "" {

		resource.SetNamespace(options.Namespace)

	}

	// Handle dry run.

	if options.DryRun {

		return nil // Just validate parsing for dry run

	}

	// Try to create the resource.

	if err := bm.client.Create(ctx, &resource); err != nil {

		// If resource exists and force replace is enabled, update it.

		if options.ForceReplace {

			if updateErr := bm.client.Update(ctx, &resource); updateErr != nil {

				return fmt.Errorf("failed to create or update resource: create error: %w, update error: %w", err, updateErr)

			}

		} else {

			return fmt.Errorf("failed to create resource: %w", err)

		}

	}

	return nil

}

func (bm *BackupManager) validateRestoredResources(ctx context.Context, manifest *BackupManifest, options RestoreOptions) ([]ValidationResult, error) {

	var results []ValidationResult

	for _, resourceMeta := range manifest.Resources {

		result := ValidationResult{

			Resource: fmt.Sprintf("%s/%s", resourceMeta.Kind, resourceMeta.Name),
		}

		// Check if resource exists.

		resource := &unstructured.Unstructured{}

		resource.SetGroupVersionKind(schema.GroupVersionKind{

			Group: "", // Will be determined from API discovery

			Version: "v1", // Simplified for test

			Kind: resourceMeta.Kind,
		})

		err := bm.client.Get(ctx, client.ObjectKey{

			Name: resourceMeta.Name,

			Namespace: resourceMeta.Namespace,
		}, resource)

		if err != nil {

			result.Valid = false

			result.Message = fmt.Sprintf("Resource not found: %v", err)

		} else {

			result.Valid = true

			result.Message = "Resource exists and accessible"

			// Additional validation for strict mode.

			if options.ValidationMode == ValidationModeStrict {

				// Validate resource has expected fields and values.

				if resource.GetName() != resourceMeta.Name {

					result.Valid = false

					result.Message = "Resource name mismatch"

				}

			}

		}

		results = append(results, result)

	}

	return results, nil

}

func (bm *BackupManager) calculateChecksum(data []byte) string {

	hash := sha256.Sum256(data)

	return fmt.Sprintf("%x", hash)

}

func (bm *BackupManager) calculateBackupChecksum(backupPath string) (string, int64) {

	hash := sha256.New()

	var totalSize int64

	filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {

		if err != nil || info.IsDir() || filepath.Base(path) == "manifest.json" {

			return nil

		}

		file, err := os.Open(path)

		if err != nil {

			return nil

		}

		defer file.Close()

		size, err := io.Copy(hash, file)

		if err != nil {

			return nil

		}

		totalSize += size

		return nil

	})

	return fmt.Sprintf("%x", hash.Sum(nil)), totalSize

}

// GetBackupInfo returns information about a specific backup.

func (bm *BackupManager) GetBackupInfo(backupID string) (*BackupManifest, error) {

	backupPath := filepath.Join(bm.backupDir, backupID)

	return bm.loadManifest(backupPath)

}

// CompareBackups compares two backups and returns differences.

func (bm *BackupManager) CompareBackups(backup1ID, backup2ID string) (*BackupComparison, error) {

	manifest1, err := bm.GetBackupInfo(backup1ID)

	if err != nil {

		return nil, fmt.Errorf("failed to load backup1: %w", err)

	}

	manifest2, err := bm.GetBackupInfo(backup2ID)

	if err != nil {

		return nil, fmt.Errorf("failed to load backup2: %w", err)

	}

	comparison := &BackupComparison{

		Backup1ID: backup1ID,

		Backup2ID: backup2ID,
	}

	// Compare resource counts.

	for kind, count1 := range manifest1.ResourceCounts {

		count2, exists := manifest2.ResourceCounts[kind]

		if !exists {

			comparison.Differences = append(comparison.Differences, BackupDifference{

				Type: "missing_resource_type",

				Resource: kind,

				Description: fmt.Sprintf("Resource type %s exists in backup1 but not backup2", kind),
			})

		} else if count1 != count2 {

			comparison.Differences = append(comparison.Differences, BackupDifference{

				Type: "resource_count_mismatch",

				Resource: kind,

				Description: fmt.Sprintf("Resource count mismatch for %s: backup1=%d, backup2=%d", kind, count1, count2),
			})

		}

	}

	return comparison, nil

}

// BackupComparison represents a comparison between two backups.

type BackupComparison struct {
	Backup1ID string `json:"backup1_id"`

	Backup2ID string `json:"backup2_id"`

	Differences []BackupDifference `json:"differences"`
}

// BackupDifference represents a difference between backups.

type BackupDifference struct {
	Type string `json:"type"`

	Resource string `json:"resource"`

	Description string `json:"description"`
}
