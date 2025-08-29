package ca

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificateDistributor manages certificate distribution and hot-reload.

type CertificateDistributor struct {
	config *DistributionConfig

	logger *logging.StructuredLogger

	client client.Client

	watchers map[string]*FileWatcher

	distributionJobs map[string]*DistributionJob

	notifier *CertificateNotifier

	mu sync.RWMutex

	ctx context.Context

	cancel context.CancelFunc
}

// DistributionJob represents a certificate distribution job.

type DistributionJob struct {
	ID string `json:"id"`

	Certificate *CertificateResponse `json:"certificate"`

	Targets []DistributionTarget `json:"targets"`

	Status DistributionStatus `json:"status"`

	Progress int `json:"progress"`

	StartedAt time.Time `json:"started_at"`

	CompletedAt time.Time `json:"completed_at"`

	Errors []string `json:"errors"`

	RetryCount int `json:"retry_count"`
}

// DistributionTarget represents a target for certificate distribution.

type DistributionTarget struct {
	Type TargetType `json:"type"`

	Name string `json:"name"`

	Namespace string `json:"namespace,omitempty"`

	Path string `json:"path,omitempty"`

	Config TargetConfig `json:"config,omitempty"`

	Status DistributionStatus `json:"status"`

	LastUpdated time.Time `json:"last_updated"`

	Hash string `json:"hash"`

	ErrorMessage string `json:"error_message,omitempty"`
}

// TargetType represents different distribution target types.

type TargetType string

const (

	// TargetTypeSecret holds targettypesecret value.

	TargetTypeSecret TargetType = "secret"

	// TargetTypeConfigMap holds targettypeconfigmap value.

	TargetTypeConfigMap TargetType = "configmap"

	// TargetTypeFile holds targettypefile value.

	TargetTypeFile TargetType = "file"

	// TargetTypePod holds targettypepod value.

	TargetTypePod TargetType = "pod"

	// TargetTypeDeployment holds targettypedeployment value.

	TargetTypeDeployment TargetType = "deployment"

	// TargetTypeService holds targettypeservice value.

	TargetTypeService TargetType = "service"

	// TargetTypeIngress holds targettypeingress value.

	TargetTypeIngress TargetType = "ingress"
)

// DistributionStatus represents distribution status.

type DistributionStatus string

const (

	// StatusPendingDist holds statuspendingdist value.

	StatusPendingDist DistributionStatus = "pending"

	// StatusInProgress holds statusinprogress value.

	StatusInProgress DistributionStatus = "in_progress"

	// DistStatusCompleted holds diststatuscompleted value.

	DistStatusCompleted DistributionStatus = "completed"

	// StatusFailedDist holds statusfaileddist value.

	StatusFailedDist DistributionStatus = "failed"

	// StatusRetrying holds statusretrying value.

	StatusRetrying DistributionStatus = "retrying"
)

// TargetConfig holds target-specific configuration.

type TargetConfig struct {
	SecretType corev1.SecretType `json:"secret_type,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	FileName string `json:"file_name,omitempty"`

	FileMode os.FileMode `json:"file_mode,omitempty"`

	RestartPolicy RestartPolicy `json:"restart_policy,omitempty"`

	HealthCheckPath string `json:"health_check_path,omitempty"`

	GracefulTimeout time.Duration `json:"graceful_timeout,omitempty"`
}

// RestartPolicy defines how to restart services after certificate updates.

type RestartPolicy string

const (

	// RestartPolicyNone holds restartpolicynone value.

	RestartPolicyNone RestartPolicy = "none"

	// RestartPolicyRolling holds restartpolicyrolling value.

	RestartPolicyRolling RestartPolicy = "rolling"

	// RestartPolicyRecreate holds restartpolicyrecreate value.

	RestartPolicyRecreate RestartPolicy = "recreate"

	// RestartPolicySignal holds restartpolicysignal value.

	RestartPolicySignal RestartPolicy = "signal"
)

// FileWatcher watches certificate files for changes.

type FileWatcher struct {
	path string

	watcher *fsnotify.Watcher

	callback func(path string)

	logger *logging.StructuredLogger
}

// CertificateNotifier handles certificate-related notifications.

type CertificateNotifier struct {
	config *NotificationConfig

	logger *logging.StructuredLogger

	client *http.Client
}

// NewCertificateDistributor creates a new certificate distributor.

func NewCertificateDistributor(config *DistributionConfig, logger *logging.StructuredLogger, client client.Client) (*CertificateDistributor, error) {

	ctx, cancel := context.WithCancel(context.Background())

	distributor := &CertificateDistributor{

		config: config,

		logger: logger,

		client: client,

		watchers: make(map[string]*FileWatcher),

		distributionJobs: make(map[string]*DistributionJob),

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize notifier if configured.

	if config.NotificationConfig != nil && config.NotificationConfig.Enabled {

		notifier := &CertificateNotifier{

			config: config.NotificationConfig,

			logger: logger,

			client: &http.Client{Timeout: 30 * time.Second},
		}

		distributor.notifier = notifier

	}

	return distributor, nil

}

// Start starts the certificate distributor.

func (d *CertificateDistributor) Start(ctx context.Context) {

	d.logger.Info("starting certificate distributor")

	// Start distribution job processor.

	go d.processDistributionJobs()

	// Start file watchers if hot-reload is enabled.

	if d.config.HotReloadEnabled {

		go d.runFileWatchers()

	}

	// Start notification processor.

	if d.notifier != nil {

		go d.notifier.processNotifications(ctx)

	}

	<-ctx.Done()

	d.logger.Info("certificate distributor stopped")

}

// Stop stops the certificate distributor.

func (d *CertificateDistributor) Stop() {

	d.logger.Info("stopping certificate distributor")

	d.cancel()

	// Close file watchers.

	d.mu.Lock()

	for _, watcher := range d.watchers {

		watcher.Close()

	}

	d.mu.Unlock()

}

// DistributeCertificate distributes a certificate to configured targets.

func (d *CertificateDistributor) DistributeCertificate(cert *CertificateResponse) error {

	d.logger.Info("distributing certificate",

		"serial_number", cert.SerialNumber,

		"request_id", cert.RequestID)

	// Create distribution job.

	job := &DistributionJob{

		ID: generateJobID(cert),

		Certificate: cert,

		Targets: d.buildDistributionTargets(cert),

		Status: StatusPendingDist,

		StartedAt: time.Now(),
	}

	// Queue job for processing.

	d.mu.Lock()

	d.distributionJobs[job.ID] = job

	d.mu.Unlock()

	d.logger.Info("certificate distribution job queued",

		"job_id", job.ID,

		"targets", len(job.Targets))

	return nil

}

// processDistributionJobs processes queued distribution jobs.

func (d *CertificateDistributor) processDistributionJobs() {

	ticker := time.NewTicker(5 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-d.ctx.Done():

			return

		case <-ticker.C:

			d.processPendingJobs()

		}

	}

}

func (d *CertificateDistributor) processPendingJobs() {

	d.mu.Lock()

	var pendingJobs []*DistributionJob

	for _, job := range d.distributionJobs {

		if job.Status == StatusPendingDist || job.Status == StatusRetrying {

			pendingJobs = append(pendingJobs, job)

		}

	}

	d.mu.Unlock()

	for _, job := range pendingJobs {

		go d.processDistributionJob(job)

	}

}

func (d *CertificateDistributor) processDistributionJob(job *DistributionJob) {

	d.logger.Info("processing distribution job",

		"job_id", job.ID,

		"certificate", job.Certificate.SerialNumber)

	job.Status = StatusInProgress

	job.Errors = []string{}

	successCount := 0

	totalTargets := len(job.Targets)

	for i := range job.Targets {

		target := &job.Targets[i]

		if err := d.distributeToTarget(job.Certificate, target); err != nil {

			d.logger.Error("failed to distribute to target",

				"job_id", job.ID,

				"target", target.Name,

				"type", target.Type,

				"error", err)

			target.Status = StatusFailedDist

			target.ErrorMessage = err.Error()

			job.Errors = append(job.Errors, fmt.Sprintf("%s: %v", target.Name, err))

		} else {

			target.Status = DistStatusCompleted

			target.LastUpdated = time.Now()

			target.Hash = d.calculateCertificateHash(job.Certificate)

			successCount++

		}

		job.Progress = (successCount * 100) / totalTargets

	}

	// Update job status.

	if successCount == totalTargets {

		job.Status = DistStatusCompleted

		d.logger.Info("distribution job completed successfully",

			"job_id", job.ID,

			"targets", totalTargets)

		// Send success notification.

		if d.notifier != nil {

			d.notifier.sendDistributionNotification(job, true)

		}

	} else {

		if job.RetryCount < 3 {

			job.Status = StatusRetrying

			job.RetryCount++

			d.logger.Warn("distribution job partially failed, will retry",

				"job_id", job.ID,

				"success_count", successCount,

				"total_targets", totalTargets,

				"retry_count", job.RetryCount)

			// Schedule retry after delay.

			go func() {

				time.Sleep(time.Duration(job.RetryCount*30) * time.Second)

				job.Status = StatusPendingDist

			}()

		} else {

			job.Status = StatusFailedDist

			d.logger.Error("distribution job failed after retries",

				"job_id", job.ID,

				"success_count", successCount,

				"total_targets", totalTargets,

				"errors", job.Errors)

			// Send failure notification.

			if d.notifier != nil {

				d.notifier.sendDistributionNotification(job, false)

			}

		}

	}

	job.CompletedAt = time.Now()

}

func (d *CertificateDistributor) distributeToTarget(cert *CertificateResponse, target *DistributionTarget) error {

	switch target.Type {

	case TargetTypeSecret:

		return d.distributeToSecret(cert, target)

	case TargetTypeConfigMap:

		return d.distributeToConfigMap(cert, target)

	case TargetTypeFile:

		return d.distributeToFile(cert, target)

	case TargetTypePod:

		return d.distributeToPod(cert, target)

	case TargetTypeDeployment:

		return d.distributeToDeployment(cert, target)

	default:

		return fmt.Errorf("unsupported target type: %s", target.Type)

	}

}

func (d *CertificateDistributor) distributeToSecret(cert *CertificateResponse, target *DistributionTarget) error {

	secret := &corev1.Secret{

		ObjectMeta: metav1.ObjectMeta{

			Name: target.Name,

			Namespace: target.Namespace,

			Labels: target.Config.Labels,

			Annotations: target.Config.Annotations,
		},

		Type: target.Config.SecretType,

		Data: map[string][]byte{

			"tls.crt": []byte(cert.CertificatePEM),

			"tls.key": []byte(cert.PrivateKeyPEM),
		},
	}

	if cert.CACertificatePEM != "" {

		secret.Data["ca.crt"] = []byte(cert.CACertificatePEM)

	}

	if target.Config.SecretType == "" {

		secret.Type = corev1.SecretTypeTLS

	}

	// Add distribution metadata.

	if secret.Annotations == nil {

		secret.Annotations = make(map[string]string)

	}

	secret.Annotations["nephoran.io/certificate-serial"] = cert.SerialNumber

	secret.Annotations["nephoran.io/distributed-at"] = time.Now().Format(time.RFC3339)

	// Create or update secret.

	err := d.client.Create(d.ctx, secret)

	if err != nil {

		// Try update if create failed.

		existingSecret := &corev1.Secret{}

		if getErr := d.client.Get(d.ctx, types.NamespacedName{

			Name: target.Name,

			Namespace: target.Namespace,
		}, existingSecret); getErr == nil {

			secret.ResourceVersion = existingSecret.ResourceVersion

			err = d.client.Update(d.ctx, secret)

		}

	}

	return err

}

func (d *CertificateDistributor) distributeToConfigMap(cert *CertificateResponse, target *DistributionTarget) error {

	configMap := &corev1.ConfigMap{

		ObjectMeta: metav1.ObjectMeta{

			Name: target.Name,

			Namespace: target.Namespace,

			Labels: target.Config.Labels,

			Annotations: target.Config.Annotations,
		},

		Data: map[string]string{

			"certificate.pem": cert.CertificatePEM,
		},
	}

	if cert.CACertificatePEM != "" {

		configMap.Data["ca.pem"] = cert.CACertificatePEM

	}

	// Add trust chain if available.

	if len(cert.TrustChainPEM) > 0 {

		configMap.Data["chain.pem"] = strings.Join(cert.TrustChainPEM, "\n")

	}

	// Add distribution metadata.

	if configMap.Annotations == nil {

		configMap.Annotations = make(map[string]string)

	}

	configMap.Annotations["nephoran.io/certificate-serial"] = cert.SerialNumber

	configMap.Annotations["nephoran.io/distributed-at"] = time.Now().Format(time.RFC3339)

	// Create or update configmap.

	err := d.client.Create(d.ctx, configMap)

	if err != nil {

		// Try update if create failed.

		existingConfigMap := &corev1.ConfigMap{}

		if getErr := d.client.Get(d.ctx, types.NamespacedName{

			Name: target.Name,

			Namespace: target.Namespace,
		}, existingConfigMap); getErr == nil {

			configMap.ResourceVersion = existingConfigMap.ResourceVersion

			err = d.client.Update(d.ctx, configMap)

		}

	}

	return err

}

func (d *CertificateDistributor) distributeToFile(cert *CertificateResponse, target *DistributionTarget) error {

	// Ensure target directory exists.

	dir := filepath.Dir(target.Path)

	if err := os.MkdirAll(dir, 0o755); err != nil {

		return fmt.Errorf("failed to create directory %s: %w", dir, err)

	}

	// Determine file content based on filename.

	var content string

	fileName := target.Config.FileName

	if fileName == "" {

		fileName = filepath.Base(target.Path)

	}

	switch {

	case strings.Contains(fileName, "cert") || strings.Contains(fileName, ".crt"):

		content = cert.CertificatePEM

	case strings.Contains(fileName, "key"):

		content = cert.PrivateKeyPEM

	case strings.Contains(fileName, "ca"):

		content = cert.CACertificatePEM

	case strings.Contains(fileName, "chain"):

		content = strings.Join(cert.TrustChainPEM, "\n")

	default:

		// Default to certificate.

		content = cert.CertificatePEM

	}

	// Set file mode.

	mode := target.Config.FileMode

	if mode == 0 {

		if strings.Contains(fileName, "key") {

			mode = 0o600 // Private key should be restrictive

		} else {

			mode = 0o644

		}

	}

	// Write file.

	if err := os.WriteFile(target.Path, []byte(content), mode); err != nil {

		return fmt.Errorf("failed to write certificate file %s: %w", target.Path, err)

	}

	// Set up file watcher for hot reload if enabled.

	if d.config.HotReloadEnabled {

		d.setupFileWatcher(target.Path, cert)

	}

	return nil

}

func (d *CertificateDistributor) distributeToPod(cert *CertificateResponse, target *DistributionTarget) error {

	// Find pods matching the target.

	podList := &corev1.PodList{}

	listOpts := &client.ListOptions{

		Namespace: target.Namespace,

		LabelSelector: labels.SelectorFromSet(target.Config.Labels),
	}

	if err := d.client.List(d.ctx, podList, listOpts); err != nil {

		return fmt.Errorf("failed to list pods: %w", err)

	}

	// Update pod annotations to trigger restart if needed.

	for _, pod := range podList.Items {

		if target.Config.RestartPolicy == RestartPolicySignal {

			// Add annotation to trigger restart.

			if pod.Annotations == nil {

				pod.Annotations = make(map[string]string)

			}

			pod.Annotations["nephoran.io/certificate-updated"] = time.Now().Format(time.RFC3339)

			pod.Annotations["nephoran.io/certificate-serial"] = cert.SerialNumber

			if err := d.client.Update(d.ctx, &pod); err != nil {

				d.logger.Warn("failed to update pod annotation",

					"pod", pod.Name,

					"namespace", pod.Namespace,

					"error", err)

			}

		}

	}

	return nil

}

func (d *CertificateDistributor) distributeToDeployment(cert *CertificateResponse, target *DistributionTarget) error {

	deployment := &appsv1.Deployment{}

	if err := d.client.Get(d.ctx, types.NamespacedName{

		Name: target.Name,

		Namespace: target.Namespace,
	}, deployment); err != nil {

		return fmt.Errorf("failed to get deployment: %w", err)

	}

	// Update deployment to trigger rolling update.

	if target.Config.RestartPolicy == RestartPolicyRolling {

		if deployment.Spec.Template.Annotations == nil {

			deployment.Spec.Template.Annotations = make(map[string]string)

		}

		deployment.Spec.Template.Annotations["nephoran.io/certificate-updated"] = time.Now().Format(time.RFC3339)

		deployment.Spec.Template.Annotations["nephoran.io/certificate-serial"] = cert.SerialNumber

		if err := d.client.Update(d.ctx, deployment); err != nil {

			return fmt.Errorf("failed to update deployment: %w", err)

		}

	}

	return nil

}

func (d *CertificateDistributor) buildDistributionTargets(cert *CertificateResponse) []DistributionTarget {

	var targets []DistributionTarget

	// Build targets based on certificate metadata and configuration.

	tenantID := cert.Metadata["tenant_id"]

	// Add configured distribution paths.

	for targetName, targetPath := range d.config.DistributionPaths {

		targets = append(targets, DistributionTarget{

			Type: TargetTypeSecret, // Default type

			Name: fmt.Sprintf("%s-%s", targetName, cert.SerialNumber),

			Path: targetPath,

			Status: StatusPendingDist,
		})

	}

	// Add tenant-specific targets if configured.

	if tenantID != "" {

		targets = append(targets, DistributionTarget{

			Type: TargetTypeSecret,

			Name: fmt.Sprintf("tenant-%s-cert", tenantID),

			Namespace: fmt.Sprintf("tenant-%s", tenantID),

			Status: StatusPendingDist,

			Config: TargetConfig{

				Labels: map[string]string{

					"nephoran.io/tenant": tenantID,

					"nephoran.io/cert": "true",
				},
			},
		})

	}

	return targets

}

func (d *CertificateDistributor) setupFileWatcher(path string, cert *CertificateResponse) {

	d.mu.Lock()

	defer d.mu.Unlock()

	// Skip if watcher already exists.

	if _, exists := d.watchers[path]; exists {

		return

	}

	watcher, err := NewFileWatcher(path, func(watchPath string) {

		d.logger.Info("certificate file changed, triggering reload",

			"path", watchPath,

			"serial_number", cert.SerialNumber)

		// Trigger hot reload for services using this certificate.

		d.triggerHotReload(cert, watchPath)

	}, d.logger)

	if err != nil {

		d.logger.Error("failed to create file watcher",

			"path", path,

			"error", err)

		return

	}

	d.watchers[path] = watcher

}

func (d *CertificateDistributor) triggerHotReload(cert *CertificateResponse, path string) {

	// Send notification about certificate change.

	if d.notifier != nil {

		d.notifier.sendHotReloadNotification(cert, path)

	}

	// Trigger any configured hot reload actions.

	// This could include sending signals to processes, updating load balancers, etc.

}

func (d *CertificateDistributor) runFileWatchers() {

	// File watchers run in their own goroutines.

	// This function could be used for watcher management.

}

func (d *CertificateDistributor) calculateCertificateHash(cert *CertificateResponse) string {

	data := fmt.Sprintf("%s%s", cert.CertificatePEM, cert.PrivateKeyPEM)

	hash := sha256.Sum256([]byte(data))

	return fmt.Sprintf("%x", hash)

}

// GetDistributionStatus returns the status of a distribution job.

func (d *CertificateDistributor) GetDistributionStatus(jobID string) (*DistributionJob, error) {

	d.mu.RLock()

	defer d.mu.RUnlock()

	job, exists := d.distributionJobs[jobID]

	if !exists {

		return nil, fmt.Errorf("distribution job %s not found", jobID)

	}

	return job, nil

}

// ListDistributionJobs returns all distribution jobs.

func (d *CertificateDistributor) ListDistributionJobs() []*DistributionJob {

	d.mu.RLock()

	defer d.mu.RUnlock()

	jobs := make([]*DistributionJob, 0, len(d.distributionJobs))

	for _, job := range d.distributionJobs {

		jobs = append(jobs, job)

	}

	return jobs

}

// Helper functions.

func generateJobID(cert *CertificateResponse) string {

	return fmt.Sprintf("dist-%s-%d", cert.SerialNumber, time.Now().Unix())

}

// NewFileWatcher creates a new file watcher.

func NewFileWatcher(path string, callback func(string), logger *logging.StructuredLogger) (*FileWatcher, error) {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {

		return nil, err

	}

	fw := &FileWatcher{

		path: path,

		watcher: watcher,

		callback: callback,

		logger: logger,
	}

	// Add path to watcher.

	if err := watcher.Add(path); err != nil {

		watcher.Close()

		return nil, err

	}

	// Start watching.

	go fw.watch()

	return fw, nil

}

func (fw *FileWatcher) watch() {

	for {

		select {

		case event, ok := <-fw.watcher.Events:

			if !ok {

				return

			}

			if event.Op&fsnotify.Write == fsnotify.Write {

				fw.callback(event.Name)

			}

		case err, ok := <-fw.watcher.Errors:

			if !ok {

				return

			}

			fw.logger.Error("file watcher error", "error", err)

		}

	}

}

// Close performs close operation.

func (fw *FileWatcher) Close() error {

	return fw.watcher.Close()

}

// CertificateNotifier methods.

func (n *CertificateNotifier) processNotifications(ctx context.Context) {

	// Process queued notifications.

}

func (n *CertificateNotifier) sendDistributionNotification(job *DistributionJob, success bool) {

	if n.config == nil {

		return

	}

	message := fmt.Sprintf("Certificate distribution %s for serial %s",

		map[bool]string{true: "succeeded", false: "failed"}[success],

		job.Certificate.SerialNumber)

	n.sendNotification(message, job)

}

func (n *CertificateNotifier) sendHotReloadNotification(cert *CertificateResponse, path string) {

	if n.config == nil {

		return

	}

	message := fmt.Sprintf("Certificate hot reload triggered for %s (serial: %s)",

		path, cert.SerialNumber)

	n.sendNotification(message, cert)

}

func (n *CertificateNotifier) sendNotification(message string, data interface{}) {

	// Send webhooks.

	for _, webhookURL := range n.config.Webhooks {

		go n.sendWebhook(webhookURL, message, data)

	}

	// Send email if configured.

	if n.config.EmailSMTP != nil {

		go n.sendEmail(message, data)

	}

	// Send Slack if configured.

	if n.config.SlackConfig != nil {

		go n.sendSlack(message, data)

	}

}

func (n *CertificateNotifier) sendWebhook(url, message string, data interface{}) {

	payload := map[string]interface{}{

		"message": message,

		"data": data,

		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(payload)

	if err != nil {

		n.logger.Error("failed to marshal webhook payload", "error", err)

		return

	}

	resp, err := n.client.Post(url, "application/json", strings.NewReader(string(jsonData)))

	if err != nil {

		n.logger.Error("failed to send webhook", "url", url, "error", err)

		return

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		n.logger.Warn("webhook returned error status",

			"url", url,

			"status", resp.StatusCode)

	}

}

func (n *CertificateNotifier) sendEmail(message string, data interface{}) {

	// Implementation for email notifications.

	n.logger.Debug("sending email notification", "message", message)

}

func (n *CertificateNotifier) sendSlack(message string, data interface{}) {

	// Implementation for Slack notifications.

	n.logger.Debug("sending Slack notification", "message", message)

}
