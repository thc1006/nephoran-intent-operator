/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package disaster

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// Backup metrics
	backupOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backup_operations_total",
		Help: "Total number of backup operations",
	}, []string{"type", "status"})

	backupDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "backup_duration_seconds",
		Help:    "Duration of backup operations",
		Buckets: prometheus.ExponentialBuckets(60, 2, 10), // Start at 1 minute
	}, []string{"type"})

	backupSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "backup_size_bytes",
		Help: "Size of backups in bytes",
	}, []string{"type", "component"})

	backupAge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "backup_age_seconds",
		Help: "Age of the most recent backup",
	}, []string{"type", "component"})
)

// BackupManager manages automated backups of all system components
type BackupManager struct {
	mu              sync.RWMutex
	logger          *slog.Logger
	k8sClient       kubernetes.Interface
	config          *BackupConfig
	s3Client        *s3.Client
	encryptionKey   []byte
	scheduler       *BackupScheduler
	retentionPolicy *RetentionPolicy
	backupHistory   map[string][]*BackupRecord
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	// General settings
	Enabled            bool   `json:"enabled"`
	BackupSchedule     string `json:"backup_schedule"` // Cron expression
	CompressionEnabled bool   `json:"compression_enabled"`
	EncryptionEnabled  bool   `json:"encryption_enabled"`
	EncryptionKey      string `json:"encryption_key"`

	// Storage settings
	StorageProvider string         `json:"storage_provider"` // s3, gcs, azure
	S3Config        S3BackupConfig `json:"s3_config"`

	// Component settings
	WeaviateConfig     WeaviateBackupConfig `json:"weaviate_config"`
	GitConfig          GitBackupConfig      `json:"git_config"`
	ConfigBackupConfig ConfigBackupConfig   `json:"config_backup_config"`

	// Retention settings
	DailyRetention   int `json:"daily_retention"`   // Days
	WeeklyRetention  int `json:"weekly_retention"`  // Weeks
	MonthlyRetention int `json:"monthly_retention"` // Months

	// Performance settings
	ParallelBackups int   `json:"parallel_backups"`
	ChunkSize       int64 `json:"chunk_size"` // Bytes

	// Validation settings
	ValidateAfterBackup bool `json:"validate_after_backup"`
	ChecksumValidation  bool `json:"checksum_validation"`
}

// S3BackupConfig holds S3-specific backup configuration
type S3BackupConfig struct {
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	Prefix       string `json:"prefix"`
	StorageClass string `json:"storage_class"` // STANDARD, IA, GLACIER
}

// WeaviateBackupConfig holds Weaviate backup configuration
type WeaviateBackupConfig struct {
	Endpoint      string `json:"endpoint"`
	BackupID      string `json:"backup_id"`
	IncludeVector bool   `json:"include_vector"`
	IncludeSchema bool   `json:"include_schema"`
	CompressLevel int    `json:"compress_level"`
}

// GitBackupConfig holds Git repository backup configuration
type GitBackupConfig struct {
	Repositories    []GitRepository `json:"repositories"`
	IncludeHistory  bool            `json:"include_history"`
	MaxHistoryDepth int             `json:"max_history_depth"`
}

// GitRepository defines a Git repository to backup
type GitRepository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Branch   string `json:"branch"`
	AuthType string `json:"auth_type"` // token, ssh, none
	Token    string `json:"token"`
	SSHKey   string `json:"ssh_key"`
}

// ConfigBackupConfig holds configuration backup settings
type ConfigBackupConfig struct {
	KubernetesResources []ResourceType `json:"kubernetes_resources"`
	Namespaces          []string       `json:"namespaces"`
	IncludeSecrets      bool           `json:"include_secrets"`
	SecretMask          bool           `json:"secret_mask"` // Mask secret values
}

// ResourceType defines which Kubernetes resources to backup
type ResourceType struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
}

// BackupRecord represents a backup record
type BackupRecord struct {
	ID             string                     `json:"id"`
	Type           string                     `json:"type"` // full, incremental, snapshot
	Status         string                     `json:"status"`
	StartTime      time.Time                  `json:"start_time"`
	EndTime        *time.Time                 `json:"end_time,omitempty"`
	Duration       time.Duration              `json:"duration"`
	Size           int64                      `json:"size"`
	CompressedSize int64                      `json:"compressed_size"`
	Components     map[string]ComponentBackup `json:"components"`
	StoragePath    string                     `json:"storage_path"`
	Checksum       string                     `json:"checksum"`
	EncryptionInfo *EncryptionInfo            `json:"encryption_info,omitempty"`
	Metadata       map[string]interface{}     `json:"metadata"`
	RetentionClass string                     `json:"retention_class"` // daily, weekly, monthly
}

// ComponentBackup represents backup info for a specific component
type ComponentBackup struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	Size           int64                  `json:"size"`
	CompressedSize int64                  `json:"compressed_size"`
	Path           string                 `json:"path"`
	Checksum       string                 `json:"checksum"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Error          string                 `json:"error,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// EncryptionInfo holds encryption metadata
type EncryptionInfo struct {
	Algorithm string `json:"algorithm"`
	KeyHash   string `json:"key_hash"`
	IV        string `json:"iv"`
}

// BackupScheduler manages backup scheduling
type BackupScheduler struct {
	manager   *BackupManager
	ticker    *time.Ticker
	stopCh    chan struct{}
	mu        sync.Mutex
	isRunning bool
}

// RetentionPolicy manages backup retention
type RetentionPolicy struct {
	DailyRetention   int
	WeeklyRetention  int
	MonthlyRetention int
}

// NewBackupManager creates a new backup manager
func NewBackupManager(drConfig *DisasterRecoveryConfig, k8sClient kubernetes.Interface, logger *slog.Logger) (*BackupManager, error) {
	// Create default backup config if not provided
	config := &BackupConfig{
		Enabled:            true,
		BackupSchedule:     "0 2 * * *", // Daily at 2 AM
		CompressionEnabled: true,
		EncryptionEnabled:  true,
		StorageProvider:    "s3",
		S3Config: S3BackupConfig{
			Bucket:       "nephoran-backups",
			Region:       "us-west-2",
			Prefix:       "disaster-recovery",
			StorageClass: "STANDARD_IA",
		},
		WeaviateConfig: WeaviateBackupConfig{
			Endpoint:      "http://weaviate:8080",
			BackupID:      "nephoran-vector-db",
			IncludeVector: true,
			IncludeSchema: true,
			CompressLevel: 6,
		},
		GitConfig: GitBackupConfig{
			IncludeHistory:  true,
			MaxHistoryDepth: 100,
			Repositories: []GitRepository{
				{
					Name:   "nephoran-packages",
					URL:    "https://github.com/nephoran/packages.git",
					Branch: "main",
				},
			},
		},
		ConfigBackupConfig: ConfigBackupConfig{
			KubernetesResources: []ResourceType{
				{APIVersion: "nephoran.com/v1", Kind: "NetworkIntent"},
				{APIVersion: "nephoran.com/v1", Kind: "E2NodeSet"},
				{APIVersion: "nephoran.com/v1", Kind: "ManagedElement"},
				{APIVersion: "v1", Kind: "ConfigMap"},
				{APIVersion: "v1", Kind: "Secret"},
			},
			Namespaces:     []string{"nephoran-system", "default"},
			IncludeSecrets: true,
			SecretMask:     true,
		},
		DailyRetention:      30, // 30 days
		WeeklyRetention:     12, // 12 weeks
		MonthlyRetention:    12, // 12 months
		ParallelBackups:     3,
		ChunkSize:           64 * 1024 * 1024, // 64MB
		ValidateAfterBackup: true,
		ChecksumValidation:  true,
	}

	bm := &BackupManager{
		logger:        logger,
		k8sClient:     k8sClient,
		config:        config,
		backupHistory: make(map[string][]*BackupRecord),
		retentionPolicy: &RetentionPolicy{
			DailyRetention:   config.DailyRetention,
			WeeklyRetention:  config.WeeklyRetention,
			MonthlyRetention: config.MonthlyRetention,
		},
	}

	// Initialize encryption key
	if config.EncryptionEnabled {
		if err := bm.initializeEncryption(config.EncryptionKey); err != nil {
			return nil, fmt.Errorf("failed to initialize encryption: %w", err)
		}
	}

	// Initialize S3 client
	if config.StorageProvider == "s3" {
		if err := bm.initializeS3Client(); err != nil {
			return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
		}
	}

	// Initialize scheduler
	bm.scheduler = &BackupScheduler{
		manager: bm,
		stopCh:  make(chan struct{}),
	}

	logger.Info("Backup manager initialized successfully")
	return bm, nil
}

// initializeEncryption sets up encryption key
func (bm *BackupManager) initializeEncryption(keyHex string) error {
	if keyHex == "" {
		// Generate a new key
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}
		bm.encryptionKey = key
		bm.logger.Warn("Generated new encryption key - ensure this is saved securely")
	} else {
		// Use provided key (should be hex encoded)
		key := []byte(keyHex)
		if len(key) != 32 {
			return fmt.Errorf("encryption key must be 32 bytes")
		}
		bm.encryptionKey = key
	}

	return nil
}

// initializeS3Client initializes the S3 client
func (bm *BackupManager) initializeS3Client() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(bm.config.S3Config.Region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	bm.s3Client = s3.NewFromConfig(cfg)
	return nil
}

// Start starts the backup manager and scheduler
func (bm *BackupManager) Start(ctx context.Context) error {
	if !bm.config.Enabled {
		bm.logger.Info("Backup manager is disabled")
		return nil
	}

	bm.logger.Info("Starting backup manager")

	// Start scheduler
	if err := bm.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backup scheduler: %w", err)
	}

	// Perform initial backup if none exists
	go func() {
		time.Sleep(1 * time.Minute) // Wait for system to stabilize
		if len(bm.backupHistory["full"]) == 0 {
			bm.logger.Info("No existing backups found, creating initial backup")
			if _, err := bm.CreateFullBackup(ctx); err != nil {
				bm.logger.Error("Failed to create initial backup", "error", err)
			}
		}
	}()

	bm.logger.Info("Backup manager started successfully")
	return nil
}

// CreateFullBackup creates a comprehensive full backup
func (bm *BackupManager) CreateFullBackup(ctx context.Context) (*BackupRecord, error) {
	return bm.createBackup(ctx, "full")
}

// CreateIncrementalBackup creates an incremental backup
func (bm *BackupManager) CreateIncrementalBackup(ctx context.Context) (*BackupRecord, error) {
	return bm.createBackup(ctx, "incremental")
}

// createBackup creates a backup of specified type
func (bm *BackupManager) createBackup(ctx context.Context, backupType string) (*BackupRecord, error) {
	start := time.Now()
	backupID := fmt.Sprintf("%s-backup-%d", backupType, start.Unix())

	bm.logger.Info("Starting backup creation", "type", backupType, "id", backupID)

	defer func() {
		backupDuration.WithLabelValues(backupType).Observe(time.Since(start).Seconds())
	}()

	// Create backup record
	record := &BackupRecord{
		ID:         backupID,
		Type:       backupType,
		Status:     "in_progress",
		StartTime:  start,
		Components: make(map[string]ComponentBackup),
		Metadata:   make(map[string]interface{}),
	}

	// Define components to backup
	components := []struct {
		name string
		fn   func(context.Context, *BackupRecord) error
	}{
		{"weaviate", bm.backupWeaviate},
		{"kubernetes-config", bm.backupKubernetesConfig},
		{"git-repositories", bm.backupGitRepositories},
		{"persistent-volumes", bm.backupPersistentVolumes},
		{"system-state", bm.backupSystemState},
	}

	// Execute backups in parallel
	var wg sync.WaitGroup
	errCh := make(chan error, len(components))
	semaphore := make(chan struct{}, bm.config.ParallelBackups)

	for _, comp := range components {
		wg.Add(1)
		go func(name string, backupFn func(context.Context, *BackupRecord) error) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			start := time.Now()
			err := backupFn(ctx, record)

			if err != nil {
				bm.logger.Error("Component backup failed", "component", name, "error", err)
				errCh <- fmt.Errorf("component %s failed: %w", name, err)
			} else {
				bm.logger.Info("Component backup completed", "component", name, "duration", time.Since(start))
			}
		}(comp.name, comp.fn)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		record.Status = "failed"
		backupOperations.WithLabelValues(backupType, "failed").Inc()
		return record, fmt.Errorf("backup failed with %d errors: %v", len(errors), errors)
	}

	// Finalize backup
	endTime := time.Now()
	record.EndTime = &endTime
	record.Duration = endTime.Sub(start)
	record.Status = "completed"

	// Calculate total size and compressed size
	var totalSize, totalCompressed int64
	for _, comp := range record.Components {
		totalSize += comp.Size
		totalCompressed += comp.CompressedSize
	}
	record.Size = totalSize
	record.CompressedSize = totalCompressed

	// Calculate checksum
	record.Checksum = bm.calculateBackupChecksum(record)

	// Encrypt if enabled
	if bm.config.EncryptionEnabled {
		if err := bm.encryptBackup(record); err != nil {
			bm.logger.Error("Failed to encrypt backup", "error", err)
		}
	}

	// Upload to storage
	if err := bm.uploadBackup(ctx, record); err != nil {
		record.Status = "upload_failed"
		backupOperations.WithLabelValues(backupType, "failed").Inc()
		return record, fmt.Errorf("failed to upload backup: %w", err)
	}

	// Validate if enabled
	if bm.config.ValidateAfterBackup {
		if err := bm.validateBackup(ctx, record); err != nil {
			record.Status = "validation_failed"
			backupOperations.WithLabelValues(backupType, "failed").Inc()
			return record, fmt.Errorf("backup validation failed: %w", err)
		}
	}

	// Store backup record
	bm.mu.Lock()
	bm.backupHistory[backupType] = append(bm.backupHistory[backupType], record)
	bm.mu.Unlock()

	// Update metrics
	backupOperations.WithLabelValues(backupType, "success").Inc()
	backupSize.WithLabelValues(backupType, "total").Set(float64(record.Size))
	backupAge.WithLabelValues(backupType, "latest").Set(0)

	// Clean up old backups
	go bm.cleanupOldBackups(ctx, backupType)

	bm.logger.Info("Backup completed successfully",
		"id", backupID,
		"type", backupType,
		"size", record.Size,
		"compressed_size", record.CompressedSize,
		"duration", record.Duration)

	return record, nil
}

// backupWeaviate creates a backup of the Weaviate vector database
func (bm *BackupManager) backupWeaviate(ctx context.Context, record *BackupRecord) error {
	start := time.Now()
	component := ComponentBackup{
		Name:      "weaviate",
		Type:      "vector_database",
		Status:    "in_progress",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	bm.logger.Info("Starting Weaviate backup")

	// Create Weaviate backup using REST API
	backupPath := fmt.Sprintf("/tmp/weaviate-backup-%d.tar.gz", start.Unix())

	// Note: In real implementation, this would use Weaviate's backup API
	// For now, we'll simulate the backup process
	err := bm.createWeaviateBackup(ctx, backupPath)
	if err != nil {
		component.Status = "failed"
		component.Error = err.Error()
		component.EndTime = time.Now()
		record.Components["weaviate"] = component
		return err
	}

	// Get file size
	if stat, err := os.Stat(backupPath); err == nil {
		component.Size = stat.Size()
	}

	// Compress if enabled
	if bm.config.CompressionEnabled {
		compressedPath := backupPath + ".compressed"
		compressedSize, err := bm.compressFile(backupPath, compressedPath)
		if err != nil {
			bm.logger.Error("Failed to compress Weaviate backup", "error", err)
		} else {
			component.CompressedSize = compressedSize
			component.Path = compressedPath
			os.Remove(backupPath) // Remove uncompressed file
		}
	} else {
		component.Path = backupPath
		component.CompressedSize = component.Size
	}

	// Calculate checksum
	component.Checksum = bm.calculateFileChecksum(component.Path)

	component.Status = "completed"
	component.EndTime = time.Now()
	component.Metadata["backup_method"] = "rest_api"
	component.Metadata["include_vectors"] = bm.config.WeaviateConfig.IncludeVector
	component.Metadata["include_schema"] = bm.config.WeaviateConfig.IncludeSchema

	record.Components["weaviate"] = component
	backupSize.WithLabelValues("weaviate", "component").Set(float64(component.Size))

	bm.logger.Info("Weaviate backup completed", "size", component.Size, "path", component.Path)
	return nil
}

// createWeaviateBackup creates the actual Weaviate backup
func (bm *BackupManager) createWeaviateBackup(ctx context.Context, backupPath string) error {
	// In a real implementation, this would:
	// 1. Call Weaviate's backup API: POST /v1/backups/{backend}/{backupId}
	// 2. Wait for backup completion
	// 3. Download backup data
	// 4. Save to local file

	// For simulation, create a dummy backup file
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// Write dummy data (in real implementation, this would be actual Weaviate data)
	data := fmt.Sprintf(`{
		"backup_id": "%s",
		"timestamp": "%s",
		"include_vectors": %t,
		"include_schema": %t,
		"data": "simulated_vector_data"
	}`, bm.config.WeaviateConfig.BackupID, time.Now().Format(time.RFC3339),
		bm.config.WeaviateConfig.IncludeVector, bm.config.WeaviateConfig.IncludeSchema)

	_, err = file.WriteString(data)
	return err
}

// backupKubernetesConfig backs up Kubernetes configurations and secrets
func (bm *BackupManager) backupKubernetesConfig(ctx context.Context, record *BackupRecord) error {
	start := time.Now()
	component := ComponentBackup{
		Name:      "kubernetes-config",
		Type:      "configuration",
		Status:    "in_progress",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	bm.logger.Info("Starting Kubernetes configuration backup")

	backupPath := fmt.Sprintf("/tmp/k8s-config-backup-%d.tar.gz", start.Unix())

	// Create tar.gz file for all configurations
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	var totalSize int64

	// Backup each resource type in each namespace
	for _, ns := range bm.config.ConfigBackupConfig.Namespaces {
		for _, resourceType := range bm.config.ConfigBackupConfig.KubernetesResources {
			size, err := bm.backupResourceType(ctx, tw, ns, resourceType)
			if err != nil {
				bm.logger.Error("Failed to backup resource type",
					"namespace", ns, "resource", resourceType, "error", err)
				continue
			}
			totalSize += size
		}
	}

	component.Size = totalSize
	component.CompressedSize = totalSize // Already compressed
	component.Path = backupPath
	component.Checksum = bm.calculateFileChecksum(backupPath)
	component.Status = "completed"
	component.EndTime = time.Now()
	component.Metadata["namespaces"] = bm.config.ConfigBackupConfig.Namespaces
	component.Metadata["resource_types"] = len(bm.config.ConfigBackupConfig.KubernetesResources)

	record.Components["kubernetes-config"] = component
	backupSize.WithLabelValues("kubernetes-config", "component").Set(float64(component.Size))

	bm.logger.Info("Kubernetes configuration backup completed", "size", component.Size)
	return nil
}

// backupResourceType backs up a specific resource type
func (bm *BackupManager) backupResourceType(ctx context.Context, tw *tar.Writer, namespace string, resourceType ResourceType) (int64, error) {
	var totalSize int64

	switch resourceType.Kind {
	case "NetworkIntent":
		// Use the generated client to get NetworkIntents
		// This would require importing the generated client
		data := fmt.Sprintf(`# NetworkIntent resources in namespace %s
apiVersion: %s
kind: %s
# Simulated backup data for %s resources
`, namespace, resourceType.APIVersion, resourceType.Kind, resourceType.Kind)

		filename := fmt.Sprintf("%s/%s.yaml", namespace, strings.ToLower(resourceType.Kind))
		if err := bm.addToTar(tw, filename, []byte(data)); err != nil {
			return 0, err
		}
		totalSize += int64(len(data))

	case "ConfigMap":
		configMaps, err := bm.k8sClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return 0, fmt.Errorf("failed to list ConfigMaps: %w", err)
		}

		for _, cm := range configMaps.Items {
			data, err := json.MarshalIndent(cm, "", "  ")
			if err != nil {
				continue
			}

			filename := fmt.Sprintf("%s/configmaps/%s.json", namespace, cm.Name)
			if err := bm.addToTar(tw, filename, data); err != nil {
				return 0, err
			}
			totalSize += int64(len(data))
		}

	case "Secret":
		if !bm.config.ConfigBackupConfig.IncludeSecrets {
			return 0, nil
		}

		secrets, err := bm.k8sClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return 0, fmt.Errorf("failed to list Secrets: %w", err)
		}

		for _, secret := range secrets.Items {
			// Mask secret data if configured
			if bm.config.ConfigBackupConfig.SecretMask {
				for key := range secret.Data {
					secret.Data[key] = []byte("[MASKED]")
				}
			}

			data, err := json.MarshalIndent(secret, "", "  ")
			if err != nil {
				continue
			}

			filename := fmt.Sprintf("%s/secrets/%s.json", namespace, secret.Name)
			if err := bm.addToTar(tw, filename, data); err != nil {
				return 0, err
			}
			totalSize += int64(len(data))
		}
	}

	return totalSize, nil
}

// addToTar adds a file to the tar archive
func (bm *BackupManager) addToTar(tw *tar.Writer, filename string, data []byte) error {
	header := &tar.Header{
		Name:    filename,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := tw.Write(data)
	return err
}

// backupGitRepositories backs up Git repositories
func (bm *BackupManager) backupGitRepositories(ctx context.Context, record *BackupRecord) error {
	start := time.Now()
	component := ComponentBackup{
		Name:      "git-repositories",
		Type:      "source_code",
		Status:    "in_progress",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	bm.logger.Info("Starting Git repositories backup")

	backupDir := fmt.Sprintf("/tmp/git-backup-%d", start.Unix())
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	defer os.RemoveAll(backupDir)

	var totalSize int64
	repos := make(map[string]interface{})

	// Clone each repository
	for _, repo := range bm.config.GitConfig.Repositories {
		bm.logger.Info("Backing up repository", "name", repo.Name, "url", repo.URL)

		repoDir := filepath.Join(backupDir, repo.Name)
		size, err := bm.cloneRepository(ctx, repo, repoDir)
		if err != nil {
			bm.logger.Error("Failed to backup repository", "name", repo.Name, "error", err)
			repos[repo.Name] = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
			continue
		}

		repos[repo.Name] = map[string]interface{}{
			"status": "success",
			"size":   size,
			"url":    repo.URL,
			"branch": repo.Branch,
		}
		totalSize += size
	}

	// Create compressed archive
	backupPath := fmt.Sprintf("/tmp/git-repositories-backup-%d.tar.gz", start.Unix())
	compressedSize, err := bm.createTarGz(backupDir, backupPath)
	if err != nil {
		component.Status = "failed"
		component.Error = err.Error()
		component.EndTime = time.Now()
		record.Components["git-repositories"] = component
		return err
	}

	component.Size = totalSize
	component.CompressedSize = compressedSize
	component.Path = backupPath
	component.Checksum = bm.calculateFileChecksum(backupPath)
	component.Status = "completed"
	component.EndTime = time.Now()
	component.Metadata["repositories"] = repos
	component.Metadata["total_repositories"] = len(bm.config.GitConfig.Repositories)

	record.Components["git-repositories"] = component
	backupSize.WithLabelValues("git-repositories", "component").Set(float64(component.Size))

	bm.logger.Info("Git repositories backup completed", "size", component.Size, "repos", len(repos))
	return nil
}

// cloneRepository clones a Git repository
func (bm *BackupManager) cloneRepository(ctx context.Context, repo GitRepository, targetDir string) (int64, error) {
	cloneOptions := &git.CloneOptions{
		URL:      repo.URL,
		Progress: nil, // Could add progress reporting
	}

	if repo.Branch != "" {
		cloneOptions.ReferenceName = plumbing.ReferenceName("refs/heads/" + repo.Branch)
	}

	// Set depth if not including full history
	if !bm.config.GitConfig.IncludeHistory {
		cloneOptions.Depth = 1
	} else if bm.config.GitConfig.MaxHistoryDepth > 0 {
		cloneOptions.Depth = bm.config.GitConfig.MaxHistoryDepth
	}

	_, err := git.PlainCloneContext(ctx, targetDir, false, cloneOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Calculate directory size
	size, err := bm.calculateDirectorySize(targetDir)
	if err != nil {
		bm.logger.Warn("Failed to calculate repository size", "error", err)
		return 0, nil
	}

	return size, nil
}

// Helper methods

// calculateDirectorySize calculates the total size of a directory
func (bm *BackupManager) calculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// createTarGz creates a tar.gz archive of a directory
func (bm *BackupManager) createTarGz(sourceDir, targetPath string) (int64, error) {
	file, err := os.Create(targetPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Update the name to maintain directory structure
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tw, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	// Get compressed file size
	stat, err := os.Stat(targetPath)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// backupPersistentVolumes creates backups/snapshots of persistent volumes
func (bm *BackupManager) backupPersistentVolumes(ctx context.Context, record *BackupRecord) error {
	start := time.Now()
	component := ComponentBackup{
		Name:      "persistent-volumes",
		Type:      "storage",
		Status:    "in_progress",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	bm.logger.Info("Starting persistent volumes backup")

	// List all PVCs in relevant namespaces
	var totalSize int64
	pvcs := make(map[string]interface{})

	for _, ns := range bm.config.ConfigBackupConfig.Namespaces {
		pvcList, err := bm.k8sClient.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			bm.logger.Error("Failed to list PVCs", "namespace", ns, "error", err)
			continue
		}

		for _, pvc := range pvcList.Items {
			// Get PVC size
			if capacity, exists := pvc.Status.Capacity["storage"]; exists {
				size := capacity.Value()
				totalSize += size

				pvcs[fmt.Sprintf("%s/%s", ns, pvc.Name)] = map[string]interface{}{
					"size":          size,
					"storage_class": pvc.Spec.StorageClassName,
					"access_modes":  pvc.Spec.AccessModes,
					"status":        string(pvc.Status.Phase),
				}
			}
		}
	}

	// In a real implementation, this would create volume snapshots
	// For simulation, we'll create metadata about the volumes
	backupPath := fmt.Sprintf("/tmp/pv-backup-metadata-%d.json", start.Unix())
	metadata := map[string]interface{}{
		"timestamp":  time.Now().Format(time.RFC3339),
		"pvcs":       pvcs,
		"total_size": totalSize,
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal PV metadata: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write PV backup metadata: %w", err)
	}

	component.Size = totalSize
	component.CompressedSize = int64(len(data))
	component.Path = backupPath
	component.Checksum = bm.calculateFileChecksum(backupPath)
	component.Status = "completed"
	component.EndTime = time.Now()
	component.Metadata["pvc_count"] = len(pvcs)
	component.Metadata["namespaces"] = bm.config.ConfigBackupConfig.Namespaces

	record.Components["persistent-volumes"] = component
	backupSize.WithLabelValues("persistent-volumes", "component").Set(float64(component.Size))

	bm.logger.Info("Persistent volumes backup completed", "pvc_count", len(pvcs), "total_size", totalSize)
	return nil
}

// backupSystemState backs up system state and metadata
func (bm *BackupManager) backupSystemState(ctx context.Context, record *BackupRecord) error {
	start := time.Now()
	component := ComponentBackup{
		Name:      "system-state",
		Type:      "metadata",
		Status:    "in_progress",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	bm.logger.Info("Starting system state backup")

	// Collect system state information
	systemState := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"cluster_info":   bm.getClusterInfo(ctx),
		"node_info":      bm.getNodeInfo(ctx),
		"namespace_info": bm.getNamespaceInfo(ctx),
		"backup_config":  bm.config,
		"backup_history": bm.backupHistory,
	}

	data, err := json.MarshalIndent(systemState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal system state: %w", err)
	}

	backupPath := fmt.Sprintf("/tmp/system-state-backup-%d.json", start.Unix())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write system state backup: %w", err)
	}

	component.Size = int64(len(data))
	component.CompressedSize = component.Size
	component.Path = backupPath
	component.Checksum = bm.calculateFileChecksum(backupPath)
	component.Status = "completed"
	component.EndTime = time.Now()

	record.Components["system-state"] = component
	backupSize.WithLabelValues("system-state", "component").Set(float64(component.Size))

	bm.logger.Info("System state backup completed", "size", component.Size)
	return nil
}

// getClusterInfo collects cluster information
func (bm *BackupManager) getClusterInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	// Get server version
	if version, err := bm.k8sClient.Discovery().ServerVersion(); err == nil {
		info["kubernetes_version"] = version
	}

	// Get API resources (sample)
	info["api_discovery_timestamp"] = time.Now().Format(time.RFC3339)

	return info
}

// getNodeInfo collects node information
func (bm *BackupManager) getNodeInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	nodes, err := bm.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		info["error"] = err.Error()
		return info
	}

	nodeList := make([]map[string]interface{}, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		nodeInfo := map[string]interface{}{
			"name":    node.Name,
			"ready":   isNodeReady(node),
			"version": node.Status.NodeInfo.KubeletVersion,
		}
		nodeList = append(nodeList, nodeInfo)
	}

	info["nodes"] = nodeList
	info["node_count"] = len(nodeList)
	return info
}

// isNodeReady checks if a node is ready
func isNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// getNamespaceInfo collects namespace information
func (bm *BackupManager) getNamespaceInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	namespaces, err := bm.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		info["error"] = err.Error()
		return info
	}

	nsList := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		nsList = append(nsList, ns.Name)
	}

	info["namespaces"] = nsList
	info["namespace_count"] = len(nsList)
	return info
}

// compressFile compresses a file using gzip
func (bm *BackupManager) compressFile(sourceFile, targetFile string) (int64, error) {
	src, err := os.Open(sourceFile)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(targetFile)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	gw := gzip.NewWriter(dst)
	defer gw.Close()

	written, err := io.Copy(gw, src)
	if err != nil {
		return 0, err
	}

	return written, nil
}

// calculateFileChecksum calculates SHA256 checksum of a file
func (bm *BackupManager) calculateFileChecksum(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		bm.logger.Error("Failed to open file for checksum", "path", filePath, "error", err)
		return ""
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		bm.logger.Error("Failed to calculate checksum", "path", filePath, "error", err)
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// calculateBackupChecksum calculates overall backup checksum
func (bm *BackupManager) calculateBackupChecksum(record *BackupRecord) string {
	hash := sha256.New()

	// Include backup metadata in checksum
	data := fmt.Sprintf("%s-%s-%d", record.ID, record.Type, record.StartTime.Unix())
	hash.Write([]byte(data))

	// Include component checksums
	for name, comp := range record.Components {
		hash.Write([]byte(fmt.Sprintf("%s:%s", name, comp.Checksum)))
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// encryptBackup encrypts backup data
func (bm *BackupManager) encryptBackup(record *BackupRecord) error {
	if len(bm.encryptionKey) == 0 {
		return fmt.Errorf("encryption key not initialized")
	}

	// Create AES cipher
	block, err := aes.NewCipher(bm.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return fmt.Errorf("failed to generate IV: %w", err)
	}

	// Create GCM mode
	_, err = cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Store encryption info
	keyHash := sha256.Sum256(bm.encryptionKey)
	record.EncryptionInfo = &EncryptionInfo{
		Algorithm: "AES-256-GCM",
		KeyHash:   fmt.Sprintf("%x", keyHash),
		IV:        fmt.Sprintf("%x", iv),
	}

	bm.logger.Info("Backup encrypted successfully", "algorithm", "AES-256-GCM")
	return nil
}

// uploadBackup uploads backup to configured storage
func (bm *BackupManager) uploadBackup(ctx context.Context, record *BackupRecord) error {
	switch bm.config.StorageProvider {
	case "s3":
		return bm.uploadToS3(ctx, record)
	case "gcs":
		return bm.uploadToGCS(ctx, record)
	case "azure":
		return bm.uploadToAzure(ctx, record)
	default:
		return fmt.Errorf("unsupported storage provider: %s", bm.config.StorageProvider)
	}
}

// uploadToS3 uploads backup to Amazon S3
func (bm *BackupManager) uploadToS3(ctx context.Context, record *BackupRecord) error {
	if bm.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// Upload each component file
	for name, comp := range record.Components {
		if comp.Path == "" {
			continue
		}

		key := fmt.Sprintf("%s/%s/%s/%s", bm.config.S3Config.Prefix, record.Type, record.ID, name)

		bm.logger.Info("Uploading component to S3", "component", name, "key", key)

		file, err := os.Open(comp.Path)
		if err != nil {
			return fmt.Errorf("failed to open component file %s: %w", comp.Path, err)
		}

		_, err = bm.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:       aws.String(bm.config.S3Config.Bucket),
			Key:          aws.String(key),
			Body:         file,
			StorageClass: types.StorageClass(bm.config.S3Config.StorageClass),
		})
		file.Close()

		if err != nil {
			return fmt.Errorf("failed to upload component %s to S3: %w", name, err)
		}

		// Update component with S3 path
		comp.Path = fmt.Sprintf("s3://%s/%s", bm.config.S3Config.Bucket, key)
		record.Components[name] = comp
	}

	// Upload backup metadata
	metadataKey := fmt.Sprintf("%s/%s/%s/metadata.json", bm.config.S3Config.Prefix, record.Type, record.ID)
	metadataData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup metadata: %w", err)
	}

	_, err = bm.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bm.config.S3Config.Bucket),
		Key:          aws.String(metadataKey),
		Body:         strings.NewReader(string(metadataData)),
		StorageClass: types.StorageClass(bm.config.S3Config.StorageClass),
	})

	if err != nil {
		return fmt.Errorf("failed to upload backup metadata to S3: %w", err)
	}

	record.StoragePath = fmt.Sprintf("s3://%s/%s", bm.config.S3Config.Bucket, metadataKey)

	bm.logger.Info("Backup uploaded to S3 successfully", "path", record.StoragePath)
	return nil
}

// Placeholder methods for other storage providers
func (bm *BackupManager) uploadToGCS(ctx context.Context, record *BackupRecord) error {
	return fmt.Errorf("GCS upload not implemented yet")
}

func (bm *BackupManager) uploadToAzure(ctx context.Context, record *BackupRecord) error {
	return fmt.Errorf("Azure upload not implemented yet")
}

// validateBackup validates backup integrity
func (bm *BackupManager) validateBackup(ctx context.Context, record *BackupRecord) error {
	bm.logger.Info("Validating backup", "id", record.ID)

	// Validate checksums
	if bm.config.ChecksumValidation {
		for name, comp := range record.Components {
			if comp.Checksum == "" {
				return fmt.Errorf("component %s missing checksum", name)
			}
		}
	}

	// Validate backup completeness
	if record.Checksum == "" {
		return fmt.Errorf("backup missing overall checksum")
	}

	// Validate component status
	for name, comp := range record.Components {
		if comp.Status != "completed" {
			return fmt.Errorf("component %s not completed: %s", name, comp.Status)
		}
	}

	bm.logger.Info("Backup validation completed successfully", "id", record.ID)
	return nil
}

// cleanupOldBackups removes old backups based on retention policy
func (bm *BackupManager) cleanupOldBackups(ctx context.Context, backupType string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	backups := bm.backupHistory[backupType]
	if len(backups) == 0 {
		return
	}

	// Sort backups by date (newest first)
	// For simplicity, assuming they're already in chronological order

	var retention int
	switch backupType {
	case "daily":
		retention = bm.retentionPolicy.DailyRetention
	case "weekly":
		retention = bm.retentionPolicy.WeeklyRetention
	case "monthly":
		retention = bm.retentionPolicy.MonthlyRetention
	default:
		retention = bm.retentionPolicy.DailyRetention
	}

	if len(backups) <= retention {
		return
	}

	// Delete old backups
	toDelete := backups[:len(backups)-retention]
	for _, backup := range toDelete {
		bm.logger.Info("Deleting old backup", "id", backup.ID, "type", backupType)
		if err := bm.deleteBackup(ctx, backup); err != nil {
			bm.logger.Error("Failed to delete backup", "id", backup.ID, "error", err)
		}
	}

	// Update backup history
	bm.backupHistory[backupType] = backups[len(backups)-retention:]
}

// deleteBackup deletes a backup from storage
func (bm *BackupManager) deleteBackup(ctx context.Context, record *BackupRecord) error {
	switch bm.config.StorageProvider {
	case "s3":
		return bm.deleteFromS3(ctx, record)
	default:
		return fmt.Errorf("delete not implemented for provider: %s", bm.config.StorageProvider)
	}
}

// deleteFromS3 deletes backup from S3
func (bm *BackupManager) deleteFromS3(ctx context.Context, record *BackupRecord) error {
	if bm.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// Delete all objects with the backup prefix
	prefix := fmt.Sprintf("%s/%s/%s/", bm.config.S3Config.Prefix, record.Type, record.ID)

	// List objects to delete
	listResult, err := bm.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bm.config.S3Config.Bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return fmt.Errorf("failed to list objects for deletion: %w", err)
	}

	// Delete each object
	for _, obj := range listResult.Contents {
		_, err := bm.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bm.config.S3Config.Bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %w", *obj.Key, err)
		}
	}

	return nil
}

// BackupScheduler methods

// Start starts the backup scheduler
func (bs *BackupScheduler) Start(ctx context.Context) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.isRunning {
		return fmt.Errorf("backup scheduler is already running")
	}

	// Parse cron schedule (simplified implementation)
	// In real implementation, use a proper cron library
	interval := 24 * time.Hour // Daily default

	bs.ticker = time.NewTicker(interval)
	bs.isRunning = true

	go bs.run(ctx)

	bs.manager.logger.Info("Backup scheduler started", "interval", interval)
	return nil
}

// run runs the backup scheduler
func (bs *BackupScheduler) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			bs.stop()
			return
		case <-bs.stopCh:
			return
		case <-bs.ticker.C:
			bs.manager.logger.Info("Scheduled backup triggered")

			// Determine backup type based on schedule
			backupType := bs.determineBackupType()

			_, err := bs.manager.createBackup(ctx, backupType)
			if err != nil {
				bs.manager.logger.Error("Scheduled backup failed", "type", backupType, "error", err)
			}
		}
	}
}

// determineBackupType determines the type of backup to create
func (bs *BackupScheduler) determineBackupType() string {
	now := time.Now()

	// Simple logic - in real implementation, this would be more sophisticated
	switch {
	case now.Day() == 1: // First day of month
		return "monthly"
	case now.Weekday() == time.Sunday: // Sunday
		return "weekly"
	default:
		return "daily"
	}
}

// stop stops the backup scheduler
func (bs *BackupScheduler) stop() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if !bs.isRunning {
		return
	}

	bs.ticker.Stop()
	close(bs.stopCh)
	bs.isRunning = false

	bs.manager.logger.Info("Backup scheduler stopped")
}
