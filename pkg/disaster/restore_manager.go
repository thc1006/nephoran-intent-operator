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
	
	"encoding/json"
"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Security constants for decompression bomb prevention (G110).
const (
	MaxFileSize       = 100 * 1024 * 1024  // 100MB per file
	MaxExtractionSize = 1024 * 1024 * 1024 // 1GB total extraction size
	MaxFileCount      = 10000              // Maximum number of files to extract
)

var (
	restoreAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "disaster_recovery_restore_attempts_total",
		Help: "Total number of restore attempts",
	}, []string{"type"})

	restoreSuccesses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "disaster_recovery_restore_successes_total",
		Help: "Total number of successful restores",
	}, []string{"type"})

	restoreFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "disaster_recovery_restore_failures_total",
		Help: "Total number of failed restores",
	}, []string{"type"})

	restoreProgress = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "disaster_recovery_restore_progress",
		Help: "Progress of current restore operation (0-100%)",
	}, []string{"type"})
)

type RestoreManager struct {
	Client    client.Client
	k8sClient kubernetes.Interface
	logger    *slog.Logger
	config    RestoreConfig

	// Metrics tracking.
	restoreAttempts  map[string]int
	restoreSuccesses map[string]int
	restoreFailures  map[string]int

	// Progress tracking.
	progress map[string]*RestoreProgress
	mu       sync.RWMutex

	// Rate limiting.
	rateLimiter chan struct{}

	// Context for cancellation.
	ctx    context.Context
	cancel context.CancelFunc

	// HTTP client with timeouts for external calls
	httpClient *http.Client

	// AWS S3 client for backup retrieval
	s3Client *s3.Client
}

// RestoreConfig holds configuration for restore operations.

type RestoreConfig struct {
	// Storage backend configuration

	StorageType string `json:"storageType"` // local, s3, gcs, azure

	// Local storage path

	LocalPath string `json:"localPath,omitempty"`

	// S3 configuration

	S3Bucket string `json:"s3Bucket,omitempty"`

	S3Region string `json:"s3Region,omitempty"`

	S3Prefix string `json:"s3Prefix,omitempty"`

	// GCS configuration

	GCSBucket string `json:"gcsBucket,omitempty"`

	GCSPrefix string `json:"gcsPrefix,omitempty"`

	// Azure configuration

	AzureContainer string `json:"azureContainer,omitempty"`

	AzurePrefix string `json:"azurePrefix,omitempty"`

	// Workers and concurrency

	Workers int `json:"workers"`

	// Retry configuration

	MaxRetries int           `json:"maxRetries"`
	RetryDelay time.Duration `json:"retryDelay"`

	// Verification

	VerifyBackup bool `json:"verifyBackup"`
}

// RestoreProgress tracks the progress of a restore operation.

type RestoreProgress struct {
	Type         string    `json:"type"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime,omitempty"`
	Status       string    `json:"status"` // running, completed, failed
	Progress     int       `json:"progress"`
	TotalSteps   int       `json:"totalSteps"`
	CurrentStep  string    `json:"currentStep"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}

// BackupMetadata represents metadata for a backup

type BackupMetadata struct {
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Components  []string               `json:"components"`
	Size        int64                  `json:"size"`
	Checksums   map[string]string      `json:"checksums"`
	Environment string                 `json:"environment"`
	Metadata    json.RawMessage `json:"metadata"`
}

// RestoreResult represents the result of a restore operation

type RestoreResult struct {
	Type      string                 `json:"type"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Duration  time.Duration          `json:"duration"`
	Metadata  json.RawMessage `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// RestoreStatus represents the status of restore operations

type RestoreStatus struct {
	Operations []RestoreResult `json:"operations"`
	Summary    RestoreSummary  `json:"summary"`
}

// RestoreSummary provides a summary of restore operations

type RestoreSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Count     int `json:"count"`
}

// NewRestoreManager creates a new RestoreManager instance.

func NewRestoreManager(drConfig *DisasterRecoveryConfig, k8sClient kubernetes.Interface, logger *slog.Logger) (*RestoreManager, error) {
	// Create default restore config from disaster recovery config
	config := RestoreConfig{
		StorageType:  "local",
		LocalPath:    "/var/lib/nephoran/backups", // Default backup path
		Workers:      5,
		MaxRetries:   3,
		RetryDelay:   time.Second * 30,
		VerifyBackup: true,
	}

	// Create a fake controller-runtime client for compatibility
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	ctx, cancel := context.WithCancel(context.Background())

	rm := &RestoreManager{
		Client:    fakeClient,
		k8sClient: k8sClient,

		logger: logger,

		config: config,

		restoreAttempts: make(map[string]int),

		restoreSuccesses: make(map[string]int),

		restoreFailures: make(map[string]int),

		progress: make(map[string]*RestoreProgress),

		rateLimiter: make(chan struct{}, config.Workers),

		ctx: ctx,

		cancel: cancel,

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initialize AWS client if needed

	if config.StorageType == "s3" {
		if err := rm.initAWSClient(); err != nil {

			logger.Error("Failed to initialize AWS client", "error", err)

			return nil, fmt.Errorf("failed to initialize AWS client: %w", err)

		}
	}

	// Fill rate limiter

	for range config.Workers {
		rm.rateLimiter <- struct{}{}
	}

	logger.Info("Restore manager initialized", "workers", config.Workers, "storage", config.StorageType)

	return rm, nil
}

// limitedReader prevents decompression bombs by limiting the number of bytes read.
type limitedReader struct {
	reader io.Reader
	limit  int64
	read   int64
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	if lr.read >= lr.limit {
		return 0, fmt.Errorf("read limit exceeded: %d bytes", lr.limit)
	}

	// Limit the read to not exceed our limit
	remaining := lr.limit - lr.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err := lr.reader.Read(p)
	lr.read += int64(n)
	return n, err
}

// initAWSClient initializes the AWS S3 client

func (rm *RestoreManager) initAWSClient() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),

		config.WithRegion(rm.config.S3Region),

		config.WithEC2IMDSClientEnableState(imds.ClientDefaultEnableState),
	)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	rm.s3Client = s3.NewFromConfig(cfg)

	return nil
}

// RestoreFromBackup restores from a backup archive

func (rm *RestoreManager) RestoreFromBackup(backupPath string) error {
	rm.logger.Info("Starting restore from backup", "path", backupPath)

	// Acquire rate limiter token

	<-rm.rateLimiter

	defer func() {
		rm.rateLimiter <- struct{}{}
	}()

	// Track metrics

	restoreAttempts.WithLabelValues("full").Inc()

	// Initialize progress

	progress := &RestoreProgress{
		Type:        "full",
		StartTime:   time.Now(),
		Status:      "running",
		Progress:    0,
		TotalSteps:  5,
		CurrentStep: "Validating backup",
	}

	rm.mu.Lock()

	rm.progress["full"] = progress

	rm.mu.Unlock()

	defer func() {
		rm.mu.Lock()

		if progress.Status == "running" {
			progress.Status = "failed"
		}

		progress.EndTime = time.Now()

		restoreProgress.WithLabelValues("full").Set(float64(progress.Progress))

		rm.mu.Unlock()
	}()

	// Step 1: Validate backup

	if err := rm.validateBackup(backupPath); err != nil {

		rm.logger.Error("Backup validation failed", "error", err)

		progress.ErrorMessage = err.Error()

		restoreFailures.WithLabelValues("full").Inc()

		return fmt.Errorf("backup validation failed: %w", err)

	}

	progress.Progress = 20

	progress.CurrentStep = "Extracting backup"

	// Step 2: Extract backup

	if err := rm.extractBackup(backupPath); err != nil {

		rm.logger.Error("Backup extraction failed", "error", err)

		progress.ErrorMessage = err.Error()

		restoreFailures.WithLabelValues("full").Inc()

		return fmt.Errorf("backup extraction failed: %w", err)

	}

	progress.Progress = 40

	progress.CurrentStep = "Restoring configurations"

	// Step 3: Restore configurations

	if err := rm.restoreConfigurations(); err != nil {

		rm.logger.Error("Configuration restore failed", "error", err)

		progress.ErrorMessage = err.Error()

		restoreFailures.WithLabelValues("full").Inc()

		return fmt.Errorf("configuration restore failed: %w", err)

	}

	progress.Progress = 60

	progress.CurrentStep = "Restoring persistent data"

	// Step 4: Restore persistent data

	if err := rm.restorePersistentData(); err != nil {

		rm.logger.Error("Persistent data restore failed", "error", err)

		progress.ErrorMessage = err.Error()

		restoreFailures.WithLabelValues("full").Inc()

		return fmt.Errorf("persistent data restore failed: %w", err)

	}

	progress.Progress = 80

	progress.CurrentStep = "Verifying restore"

	// Step 5: Verify restore

	if err := rm.verifyRestore(); err != nil {

		rm.logger.Error("Restore verification failed", "error", err)

		progress.ErrorMessage = err.Error()

		restoreFailures.WithLabelValues("full").Inc()

		return fmt.Errorf("restore verification failed: %w", err)

	}

	progress.Progress = 100

	progress.Status = "completed"

	progress.CurrentStep = "Completed"

	restoreSuccesses.WithLabelValues("full").Inc()

	rm.logger.Info("Restore completed successfully", "duration", time.Since(progress.StartTime))

	return nil
}

// validateBackup validates a backup archive

func (rm *RestoreManager) validateBackup(backupPath string) error {
	// Check if backup file exists

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Open backup file

	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}

	defer file.Close()

	// Check if it's a valid gzip file

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %w", err)
	}

	defer gzr.Close()

	// Check if it's a valid tar archive

	tr := tar.NewReader(gzr)

	// Read at least one header to validate

	_, err = tr.Next()

	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid tar format: %w", err)
	}

	rm.logger.Info("Backup validation completed", "path", backupPath)

	return nil
}

// extractBackup extracts a backup archive with security protection against decompression bombs

func (rm *RestoreManager) extractBackup(backupPath string) error {
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}

	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}

	defer gzr.Close()

	tr := tar.NewReader(gzr)

	extractPath := "/tmp/restore"

	if err := os.MkdirAll(extractPath, 0o755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	var totalExtracted int64
	var fileCount int

	for {

		header, err := tr.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Security check: Prevent zip bomb by limiting file count
		fileCount++
		if fileCount > MaxFileCount {
			return fmt.Errorf("too many files in archive: %d (max %d)", fileCount, MaxFileCount)
		}

		// Security check: Prevent directory traversal

		if !strings.HasPrefix(header.Name, "./") && !strings.HasPrefix(header.Name, "/") {
			header.Name = "./" + header.Name
		}

		target := filepath.Join(extractPath, header.Name)

		if !strings.HasPrefix(target, filepath.Clean(extractPath)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {

		case tar.TypeDir:

			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:

			// Security check: Prevent excessively large files
			if header.Size > MaxFileSize {
				return fmt.Errorf("file too large: %s (%d bytes, max %d)", header.Name, header.Size, MaxFileSize)
			}

			// Security check: Prevent decompression bomb by tracking total extracted size
			totalExtracted += header.Size
			if totalExtracted > MaxExtractionSize {
				return fmt.Errorf("total extraction size too large: %d bytes (max %d)", totalExtracted, MaxExtractionSize)
			}

			// Security fix (G115): Validate file mode before conversion
			// Unix file modes are typically 12 bits (0777 for permissions + special bits)
			// tar.Header.Mode is int64 but should contain valid Unix mode values
			if header.Mode < 0 || header.Mode > 0o777777 { // Max octal mode with all special bits
				return fmt.Errorf("invalid file mode %o for %s", header.Mode, header.Name)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode)) //nolint:gosec // G115: mode validated above
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			// Security fix: Use limited reader to prevent decompression bomb (G110)
			limitedReader := &limitedReader{
				reader: tr,
				limit:  MaxFileSize,
			}
			_, err = io.Copy(f, limitedReader)

			f.Close()

			if err != nil {
				return fmt.Errorf("failed to copy file content: %w", err)
			}

		}

	}

	rm.logger.Info("Backup extraction completed", "files", fileCount, "totalSize", totalExtracted, "path", extractPath)

	return nil
}

// restoreConfigurations restores Kubernetes configurations

func (rm *RestoreManager) restoreConfigurations() error {
	// Implementation for restoring Kubernetes configurations

	// This would restore ConfigMaps, Secrets, etc.

	rm.logger.Info("Restoring configurations")

	// Placeholder implementation

	return nil
}

// restorePersistentData restores persistent volumes and data

func (rm *RestoreManager) restorePersistentData() error {
	// Implementation for restoring persistent data

	// This would restore PVCs, databases, etc.

	rm.logger.Info("Restoring persistent data")

	// Placeholder implementation

	return nil
}

// verifyRestore verifies that the restore was successful

func (rm *RestoreManager) verifyRestore() error {
	if !rm.config.VerifyBackup {
		return nil
	}

	// Implementation for verifying restore

	// This would check that all components are healthy

	rm.logger.Info("Verifying restore")

	// Placeholder implementation

	return nil
}

// GetRestoreProgress returns the current restore progress

func (rm *RestoreManager) GetRestoreProgress(restoreType string) (*RestoreProgress, error) {
	rm.mu.RLock()

	defer rm.mu.RUnlock()

	progress, exists := rm.progress[restoreType]

	if !exists {
		return nil, fmt.Errorf("no restore progress found for type: %s", restoreType)
	}

	return progress, nil
}

// ListBackups lists available backups

func (rm *RestoreManager) ListBackups() ([]BackupMetadata, error) {
	switch rm.config.StorageType {

	case "local":

		return rm.listLocalBackups()

	case "s3":

		return rm.listS3Backups()

	default:

		return nil, fmt.Errorf("unsupported storage type: %s", rm.config.StorageType)

	}
}

// listLocalBackups lists backups from local storage

func (rm *RestoreManager) listLocalBackups() ([]BackupMetadata, error) {
	files, err := filepath.Glob(filepath.Join(rm.config.LocalPath, "*.tar.gz"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}

	// Preallocate slice with known capacity
	backups := make([]BackupMetadata, 0, len(files))

	for _, file := range files {

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		backup := BackupMetadata{
			Timestamp: info.ModTime(),

			Size: info.Size(),

			Metadata: json.RawMessage("{}"),
		}

		backups = append(backups, backup)

	}

	return backups, nil
}

// listS3Backups lists backups from S3 storage

func (rm *RestoreManager) listS3Backups() ([]BackupMetadata, error) {
	if rm.s3Client == nil {
		return nil, errors.New("S3 client not initialized")
	}

	// Implementation for listing S3 backups

	// This would use ListObjectsV2 to get backup files

	rm.logger.Info("Listing S3 backups", "bucket", rm.config.S3Bucket, "prefix", rm.config.S3Prefix)

	// Placeholder implementation

	return []BackupMetadata{}, nil
}

// Stop gracefully stops the restore manager

func (rm *RestoreManager) Stop() {
	rm.logger.Info("Stopping restore manager")

	rm.cancel()
}

// GetStatus returns the current restore status

func (rm *RestoreManager) GetStatus() RestoreStatus {
	rm.mu.RLock()

	defer rm.mu.RUnlock()

	// Preallocate slice with known capacity
	operations := make([]RestoreResult, 0, len(rm.progress))

	// Convert progress to results

	for _, progress := range rm.progress {

		result := RestoreResult{
			Type:      progress.Type,
			Success:   progress.Status == "completed",
			Message:   progress.CurrentStep,
			Duration:  time.Since(progress.StartTime),
			Timestamp: progress.StartTime,
		}

		if progress.ErrorMessage != "" {
			result.Message = progress.ErrorMessage
		}

		operations = append(operations, result)

	}

	// Calculate summary

	var succeeded, failed int

	for _, op := range operations {
		if op.Success {
			succeeded++
		} else {
			failed++
		}
	}

	return RestoreStatus{
		Operations: operations,

		Summary: RestoreSummary{
			Total:     len(operations),
			Succeeded: succeeded,
			Failed:    failed,
			Count:     len(operations),
		},
	}
}
