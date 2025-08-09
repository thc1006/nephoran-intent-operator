package ca

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RotationCoordinator handles coordinated certificate rotation across clustered deployments
type RotationCoordinator struct {
	logger     *logging.StructuredLogger
	kubeClient kubernetes.Interface

	// Coordination state
	activeRotations map[string]*RotationSession
	rotationLocks   map[string]*sync.RWMutex
	mu              sync.RWMutex
}

// RotationSession represents an active rotation session
type RotationSession struct {
	ID             string             `json:"id"`
	ServiceName    string             `json:"service_name"`
	Namespace      string             `json:"namespace"`
	Instances      []*ServiceInstance `json:"instances"`
	Phase          RotationPhase      `json:"phase"`
	StartTime      time.Time          `json:"start_time"`
	CompletedCount int                `json:"completed_count"`
	FailedCount    int                `json:"failed_count"`
	HealthChecks   *HealthCheckConfig `json:"health_checks,omitempty"`
	RollbackPlan   *RollbackPlan      `json:"rollback_plan,omitempty"`
	mu             sync.RWMutex
}

// ServiceInstance represents a service instance in the rotation
type ServiceInstance struct {
	Name      string            `json:"name"`
	Endpoint  string            `json:"endpoint"`
	Healthy   bool              `json:"healthy"`
	Rotated   bool              `json:"rotated"`
	Metadata  map[string]string `json:"metadata"`
	LastCheck time.Time         `json:"last_check"`
}

// RotationPhase represents the current phase of rotation
type RotationPhase string

const (
	PhaseInitializing RotationPhase = "initializing"
	PhaseValidating   RotationPhase = "validating"
	PhaseRotating     RotationPhase = "rotating"
	PhaseVerifying    RotationPhase = "verifying"
	PhaseCompleted    RotationPhase = "completed"
	PhaseFailed       RotationPhase = "failed"
	PhaseRollingBack  RotationPhase = "rolling_back"
)

// RollbackPlan defines the rollback strategy
type RollbackPlan struct {
	Enabled            bool              `json:"enabled"`
	MaxFailureCount    int               `json:"max_failure_count"`
	FailureThreshold   float64           `json:"failure_threshold"`
	RollbackTimeout    time.Duration     `json:"rollback_timeout"`
	BackupCertificates map[string]string `json:"backup_certificates"`
}

// NewRotationCoordinator creates a new rotation coordinator
func NewRotationCoordinator(
	logger *logging.StructuredLogger,
	kubeClient kubernetes.Interface,
) *RotationCoordinator {
	return &RotationCoordinator{
		logger:          logger,
		kubeClient:      kubeClient,
		activeRotations: make(map[string]*RotationSession),
		rotationLocks:   make(map[string]*sync.RWMutex),
	}
}

// StartCoordinatedRotation starts a coordinated rotation session
func (rc *RotationCoordinator) StartCoordinatedRotation(
	ctx context.Context,
	req *RenewalRequest,
) (*RotationSession, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	sessionID := fmt.Sprintf("%s-%s-%d", req.ServiceName, req.Namespace, time.Now().Unix())

	// Check if rotation is already in progress for this service
	for _, session := range rc.activeRotations {
		if session.ServiceName == req.ServiceName && session.Namespace == req.Namespace {
			if session.Phase != PhaseCompleted && session.Phase != PhaseFailed {
				return nil, fmt.Errorf("rotation already in progress for service %s/%s", req.Namespace, req.ServiceName)
			}
		}
	}

	// Discover service instances
	instances, err := rc.discoverServiceInstances(ctx, req.ServiceName, req.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service instances: %w", err)
	}

	// Create rotation session
	session := &RotationSession{
		ID:           sessionID,
		ServiceName:  req.ServiceName,
		Namespace:    req.Namespace,
		Instances:    instances,
		Phase:        PhaseInitializing,
		StartTime:    time.Now(),
		HealthChecks: req.HealthCheckConfig,
		RollbackPlan: &RollbackPlan{
			Enabled:            req.ZeroDowntime,
			MaxFailureCount:    len(instances) / 2, // Allow up to 50% failures
			FailureThreshold:   0.3,                // 30% failure threshold
			RollbackTimeout:    5 * time.Minute,
			BackupCertificates: make(map[string]string),
		},
	}

	// Create rotation lock for this service
	serviceKey := fmt.Sprintf("%s/%s", req.Namespace, req.ServiceName)
	rc.rotationLocks[serviceKey] = &sync.RWMutex{}

	rc.activeRotations[sessionID] = session

	rc.logger.Info("started coordinated rotation session",
		"session_id", sessionID,
		"service", req.ServiceName,
		"namespace", req.Namespace,
		"instance_count", len(instances),
		"zero_downtime", req.ZeroDowntime)

	return session, nil
}

// ExecuteCoordinatedRotation executes the coordinated rotation
func (rc *RotationCoordinator) ExecuteCoordinatedRotation(
	ctx context.Context,
	session *RotationSession,
	rotationFunc func(instance *ServiceInstance) error,
) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	rc.logger.Info("executing coordinated rotation",
		"session_id", session.ID,
		"service", session.ServiceName,
		"instance_count", len(session.Instances))

	// Phase 1: Initial health check
	session.Phase = PhaseValidating
	if err := rc.performHealthChecks(ctx, session); err != nil {
		session.Phase = PhaseFailed
		return fmt.Errorf("initial health check failed: %w", err)
	}

	// Phase 2: Rolling rotation
	session.Phase = PhaseRotating
	if err := rc.performRollingRotation(ctx, session, rotationFunc); err != nil {
		// Check if rollback is needed
		if session.RollbackPlan.Enabled && rc.shouldRollback(session) {
			rc.logger.Warn("rotation failed, initiating rollback",
				"session_id", session.ID,
				"failed_count", session.FailedCount)

			session.Phase = PhaseRollingBack
			if rbErr := rc.performRollback(ctx, session); rbErr != nil {
				rc.logger.Error("rollback failed",
					"session_id", session.ID,
					"error", rbErr)
			}
		}
		session.Phase = PhaseFailed
		return err
	}

	// Phase 3: Final verification
	session.Phase = PhaseVerifying
	if err := rc.performHealthChecks(ctx, session); err != nil {
		if session.RollbackPlan.Enabled {
			rc.logger.Warn("final verification failed, initiating rollback",
				"session_id", session.ID)

			session.Phase = PhaseRollingBack
			if rbErr := rc.performRollback(ctx, session); rbErr != nil {
				rc.logger.Error("rollback failed",
					"session_id", session.ID,
					"error", rbErr)
			}
		}
		session.Phase = PhaseFailed
		return fmt.Errorf("final verification failed: %w", err)
	}

	session.Phase = PhaseCompleted
	rc.logger.Info("coordinated rotation completed successfully",
		"session_id", session.ID,
		"duration", time.Since(session.StartTime))

	return nil
}

// performRollingRotation performs rolling rotation of service instances
func (rc *RotationCoordinator) performRollingRotation(
	ctx context.Context,
	session *RotationSession,
	rotationFunc func(instance *ServiceInstance) error,
) error {
	// Calculate batch size for rolling updates
	batchSize := rc.calculateBatchSize(len(session.Instances))

	rc.logger.Info("starting rolling rotation",
		"session_id", session.ID,
		"total_instances", len(session.Instances),
		"batch_size", batchSize)

	for i := 0; i < len(session.Instances); i += batchSize {
		end := i + batchSize
		if end > len(session.Instances) {
			end = len(session.Instances)
		}

		batch := session.Instances[i:end]

		rc.logger.Debug("processing rotation batch",
			"session_id", session.ID,
			"batch_start", i,
			"batch_end", end-1,
			"batch_size", len(batch))

		// Rotate batch in parallel
		var wg sync.WaitGroup
		var batchErrors []error
		var errorMu sync.Mutex

		for _, instance := range batch {
			wg.Add(1)
			go func(inst *ServiceInstance) {
				defer wg.Done()

				if err := rotationFunc(inst); err != nil {
					errorMu.Lock()
					batchErrors = append(batchErrors, err)
					session.FailedCount++
					errorMu.Unlock()

					rc.logger.Error("instance rotation failed",
						"session_id", session.ID,
						"instance", inst.Name,
						"error", err)
				} else {
					inst.Rotated = true
					session.CompletedCount++

					rc.logger.Debug("instance rotation completed",
						"session_id", session.ID,
						"instance", inst.Name)
				}
			}(instance)
		}

		wg.Wait()

		// Check batch results
		if len(batchErrors) > 0 {
			if rc.shouldRollback(session) {
				return fmt.Errorf("batch rotation failed with %d errors, rollback triggered", len(batchErrors))
			}
			rc.logger.Warn("batch rotation had errors but continuing",
				"session_id", session.ID,
				"error_count", len(batchErrors),
				"failed_total", session.FailedCount)
		}

		// Wait between batches for system stabilization
		if i+batchSize < len(session.Instances) {
			stabilizationDelay := 30 * time.Second
			rc.logger.Debug("waiting for system stabilization",
				"session_id", session.ID,
				"delay", stabilizationDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(stabilizationDelay):
			}

			// Perform health check after each batch
			if session.HealthChecks != nil && session.HealthChecks.Enabled {
				if err := rc.performHealthChecks(ctx, session); err != nil {
					return fmt.Errorf("health check failed after batch: %w", err)
				}
			}
		}
	}

	return nil
}

// performHealthChecks performs health checks on all service instances
func (rc *RotationCoordinator) performHealthChecks(ctx context.Context, session *RotationSession) error {
	if session.HealthChecks == nil || !session.HealthChecks.Enabled {
		rc.logger.Debug("health checks disabled, skipping",
			"session_id", session.ID)
		return nil
	}

	rc.logger.Debug("performing health checks",
		"session_id", session.ID,
		"instance_count", len(session.Instances))

	var wg sync.WaitGroup
	healthyCount := 0
	var healthMu sync.Mutex

	for _, instance := range session.Instances {
		wg.Add(1)
		go func(inst *ServiceInstance) {
			defer wg.Done()

			healthy := rc.checkInstanceHealth(ctx, inst, session.HealthChecks)

			healthMu.Lock()
			inst.Healthy = healthy
			inst.LastCheck = time.Now()
			if healthy {
				healthyCount++
			}
			healthMu.Unlock()

			rc.logger.Debug("health check result",
				"session_id", session.ID,
				"instance", inst.Name,
				"healthy", healthy)
		}(instance)
	}

	wg.Wait()

	// Calculate health percentage
	healthPercentage := float64(healthyCount) / float64(len(session.Instances))
	requiredThreshold := 0.8 // 80% healthy threshold

	rc.logger.Info("health check completed",
		"session_id", session.ID,
		"healthy_count", healthyCount,
		"total_count", len(session.Instances),
		"health_percentage", healthPercentage)

	if healthPercentage < requiredThreshold {
		return fmt.Errorf("health check failed: only %.2f%% instances healthy, required %.2f%%",
			healthPercentage*100, requiredThreshold*100)
	}

	return nil
}

// checkInstanceHealth checks health of a single service instance
func (rc *RotationCoordinator) checkInstanceHealth(
	ctx context.Context,
	instance *ServiceInstance,
	healthConfig *HealthCheckConfig,
) bool {
	// HTTP health check
	if healthConfig.HTTPEndpoint != "" {
		if rc.checkHTTPHealth(ctx, instance, healthConfig) {
			return true
		}
	}

	// gRPC health check
	if healthConfig.GRPCService != "" {
		if rc.checkGRPCHealth(ctx, instance, healthConfig) {
			return true
		}
	}

	return false
}

// checkHTTPHealth performs HTTP health check
func (rc *RotationCoordinator) checkHTTPHealth(
	ctx context.Context,
	instance *ServiceInstance,
	healthConfig *HealthCheckConfig,
) bool {
	client := &http.Client{
		Timeout: healthConfig.TimeoutPerCheck,
	}

	url := fmt.Sprintf("http://%s%s", instance.Endpoint, healthConfig.HTTPEndpoint)

	for attempt := 0; attempt < healthConfig.HealthyThreshold; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(healthConfig.CheckInterval)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true
		}

		time.Sleep(healthConfig.CheckInterval)
	}

	return false
}

// checkGRPCHealth performs gRPC health check
func (rc *RotationCoordinator) checkGRPCHealth(
	ctx context.Context,
	instance *ServiceInstance,
	healthConfig *HealthCheckConfig,
) bool {
	conn, err := grpc.Dial(instance.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(healthConfig.TimeoutPerCheck))
	if err != nil {
		return false
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	for attempt := 0; attempt < healthConfig.HealthyThreshold; attempt++ {
		checkCtx, cancel := context.WithTimeout(ctx, healthConfig.TimeoutPerCheck)

		resp, err := client.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{
			Service: healthConfig.GRPCService,
		})
		cancel()

		if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
			return true
		}

		time.Sleep(healthConfig.CheckInterval)
	}

	return false
}

// shouldRollback determines if rollback should be initiated
func (rc *RotationCoordinator) shouldRollback(session *RotationSession) bool {
	if !session.RollbackPlan.Enabled {
		return false
	}

	// Check absolute failure count
	if session.FailedCount >= session.RollbackPlan.MaxFailureCount {
		return true
	}

	// Check failure percentage
	totalAttempted := session.CompletedCount + session.FailedCount
	if totalAttempted == 0 {
		return false
	}

	failureRate := float64(session.FailedCount) / float64(totalAttempted)
	return failureRate >= session.RollbackPlan.FailureThreshold
}

// performRollback performs rollback of the rotation
func (rc *RotationCoordinator) performRollback(ctx context.Context, session *RotationSession) error {
	rc.logger.Info("performing rotation rollback",
		"session_id", session.ID,
		"failed_count", session.FailedCount)

	// This would implement the actual rollback logic
	// For now, we'll just log the rollback attempt

	rollbackTimeout := session.RollbackPlan.RollbackTimeout
	rollbackCtx, cancel := context.WithTimeout(ctx, rollbackTimeout)
	defer cancel()

	// Restore backup certificates for rotated instances
	for _, instance := range session.Instances {
		if instance.Rotated {
			if err := rc.rollbackInstance(rollbackCtx, instance, session); err != nil {
				rc.logger.Error("failed to rollback instance",
					"session_id", session.ID,
					"instance", instance.Name,
					"error", err)
			} else {
				instance.Rotated = false
				rc.logger.Info("rolled back instance",
					"session_id", session.ID,
					"instance", instance.Name)
			}
		}
	}

	return nil
}

// rollbackInstance rolls back a specific instance
func (rc *RotationCoordinator) rollbackInstance(ctx context.Context, instance *ServiceInstance, session *RotationSession) error {
	// Implementation would restore the backup certificate
	// This is a placeholder for the actual rollback logic
	rc.logger.Debug("rolling back instance",
		"session_id", session.ID,
		"instance", instance.Name)

	return nil
}

// discoverServiceInstances discovers service instances for rotation
func (rc *RotationCoordinator) discoverServiceInstances(ctx context.Context, serviceName, namespace string) ([]*ServiceInstance, error) {
	// Get service endpoints
	endpoints, err := rc.kubeClient.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service endpoints: %w", err)
	}

	var instances []*ServiceInstance
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			for _, port := range subset.Ports {
				instance := &ServiceInstance{
					Name:     fmt.Sprintf("%s-%s", addr.TargetRef.Name, port.Name),
					Endpoint: fmt.Sprintf("%s:%d", addr.IP, port.Port),
					Healthy:  true, // Assume healthy initially
					Rotated:  false,
					Metadata: make(map[string]string),
				}

				if addr.TargetRef != nil {
					instance.Metadata["pod_name"] = addr.TargetRef.Name
					instance.Metadata["pod_namespace"] = addr.TargetRef.Namespace
				}

				instances = append(instances, instance)
			}
		}
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no service instances found for %s/%s", namespace, serviceName)
	}

	return instances, nil
}

// calculateBatchSize calculates the optimal batch size for rolling updates
func (rc *RotationCoordinator) calculateBatchSize(totalInstances int) int {
	if totalInstances <= 2 {
		return 1
	}

	// Use 25% of instances per batch, minimum 1, maximum 5
	batchSize := totalInstances / 4
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > 5 {
		batchSize = 5
	}

	return batchSize
}

// GetRotationSession gets an active rotation session
func (rc *RotationCoordinator) GetRotationSession(sessionID string) (*RotationSession, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	session, exists := rc.activeRotations[sessionID]
	return session, exists
}

// CleanupCompletedSessions removes completed rotation sessions
func (rc *RotationCoordinator) CleanupCompletedSessions() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour) // Keep sessions for 1 hour after completion

	for sessionID, session := range rc.activeRotations {
		if (session.Phase == PhaseCompleted || session.Phase == PhaseFailed) &&
			session.StartTime.Before(cutoff) {

			// Clean up rotation lock
			serviceKey := fmt.Sprintf("%s/%s", session.Namespace, session.ServiceName)
			delete(rc.rotationLocks, serviceKey)

			delete(rc.activeRotations, sessionID)

			rc.logger.Debug("cleaned up completed rotation session",
				"session_id", sessionID,
				"phase", session.Phase)
		}
	}
}
