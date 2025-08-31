package monitoring

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthMonitor manages health checks for O-RAN components
type HealthMonitor struct {
	checkers    map[string]HealthChecker
	lastResults map[string]*ComponentHealth
	mu          sync.RWMutex
	logger      *log.Logger
	interval    time.Duration
	stopCh      chan struct{}
}

// HTTPHealthChecker checks HTTP endpoint health
type HTTPHealthChecker struct {
	name     string
	endpoint string
	client   *http.Client
	timeout  time.Duration
}

// KubernetesHealthChecker checks Kubernetes component health
type KubernetesHealthChecker struct {
	name      string
	namespace string
	selector  string
}

// ProcessHealthChecker checks process health
type ProcessHealthChecker struct {
	name        string
	processName string
	pidFile     string
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(logger *log.Logger, checkInterval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		checkers:    make(map[string]HealthChecker),
		lastResults: make(map[string]*ComponentHealth),
		logger:      logger,
		interval:    checkInterval,
		stopCh:      make(chan struct{}),
	}
}

// RegisterChecker registers a health checker
func (hm *HealthMonitor) RegisterChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checkers[checker.GetName()] = checker
}

// UnregisterChecker removes a health checker
func (hm *HealthMonitor) UnregisterChecker(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.checkers, name)
	delete(hm.lastResults, name)
}

// Start starts the health monitoring loop
func (hm *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.runHealthChecks(ctx)
		}
	}
}

// Stop stops the health monitoring
func (hm *HealthMonitor) Stop() {
	close(hm.stopCh)
}

// runHealthChecks runs all registered health checks
func (hm *HealthMonitor) runHealthChecks(ctx context.Context) {
	hm.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range hm.checkers {
		checkers[name] = checker
	}
	hm.mu.RUnlock()

	var wg sync.WaitGroup
	results := make(map[string]*ComponentHealth)
	mu := sync.Mutex{}

	for name, checker := range checkers {
		wg.Add(1)
		go func(n string, c HealthChecker) {
			defer wg.Done()
			
			result, err := c.CheckHealth(ctx)
			if err != nil {
				result = &ComponentHealth{
					Name:      n,
					Status:    HealthStatusUnhealthy,
					Message:   fmt.Sprintf("Health check failed: %v", err),
					Timestamp: time.Now(),
				}
			}

			mu.Lock()
			results[n] = result
			mu.Unlock()
		}(name, checker)
	}

	wg.Wait()

	hm.mu.Lock()
	for name, result := range results {
		hm.lastResults[name] = result
	}
	hm.mu.Unlock()

	hm.logger.Printf("Health check completed for %d components", len(results))
}

// GetSystemHealth returns overall system health
func (hm *HealthMonitor) GetSystemHealth() *SystemHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	components := make([]ComponentHealth, 0, len(hm.lastResults))
	overallStatus := HealthStatusHealthy
	unhealthyCount := 0

	for _, result := range hm.lastResults {
		components = append(components, *result)
		
		switch result.Status {
		case HealthStatusUnhealthy:
			unhealthyCount++
			overallStatus = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}

	// If more than 50% components are unhealthy, system is unhealthy
	if float64(unhealthyCount)/float64(len(components)) > 0.5 {
		overallStatus = HealthStatusUnhealthy
	}

	return &SystemHealth{
		OverallStatus: overallStatus,
		Components:    components,
		LastChecked:   time.Now(),
	}
}

// GetComponentHealth returns health for a specific component
func (hm *HealthMonitor) GetComponentHealth(name string) (*ComponentHealth, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if result, ok := hm.lastResults[name]; ok {
		return result, nil
	}

	return nil, fmt.Errorf("component not found: %s", name)
}

// NewHTTPHealthChecker creates an HTTP health checker
func NewHTTPHealthChecker(name, endpoint string, timeout time.Duration) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		name:     name,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// CheckHealth implements HealthChecker interface for HTTP
func (hc *HTTPHealthChecker) CheckHealth(ctx context.Context) (*ComponentHealth, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hc.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return &ComponentHealth{
			Name:      hc.name,
			Status:    HealthStatusUnhealthy,
			Message:   fmt.Sprintf("HTTP request failed: %v", err),
			Timestamp: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	var status HealthStatus
	var message string

	switch resp.StatusCode {
	case 200:
		status = HealthStatusHealthy
		message = "OK"
	case 503:
		status = HealthStatusDegraded
		message = "Service Unavailable"
	default:
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return &ComponentHealth{
		Name:      hc.name,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"endpoint":    hc.endpoint,
			"status_code": fmt.Sprintf("%d", resp.StatusCode),
		},
	}, nil
}

// GetName implements HealthChecker interface for HTTP
func (hc *HTTPHealthChecker) GetName() string {
	return hc.name
}

// NewKubernetesHealthChecker creates a Kubernetes health checker
func NewKubernetesHealthChecker(name, namespace, selector string) *KubernetesHealthChecker {
	return &KubernetesHealthChecker{
		name:      name,
		namespace: namespace,
		selector:  selector,
	}
}

// CheckHealth implements HealthChecker interface for Kubernetes
func (kc *KubernetesHealthChecker) CheckHealth(ctx context.Context) (*ComponentHealth, error) {
	// This would integrate with Kubernetes client-go in real implementation
	// For now, return a mock healthy status
	return &ComponentHealth{
		Name:      kc.name,
		Status:    HealthStatusHealthy,
		Message:   "Kubernetes component is healthy",
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"namespace": kc.namespace,
			"selector":  kc.selector,
		},
	}, nil
}

// GetName implements HealthChecker interface for Kubernetes
func (kc *KubernetesHealthChecker) GetName() string {
	return kc.name
}

// NewProcessHealthChecker creates a process health checker
func NewProcessHealthChecker(name, processName, pidFile string) *ProcessHealthChecker {
	return &ProcessHealthChecker{
		name:        name,
		processName: processName,
		pidFile:     pidFile,
	}
}

// CheckHealth implements HealthChecker interface for Process
func (pc *ProcessHealthChecker) CheckHealth(ctx context.Context) (*ComponentHealth, error) {
	// This would check actual process status in real implementation
	// For now, return a mock healthy status
	return &ComponentHealth{
		Name:      pc.name,
		Status:    HealthStatusHealthy,
		Message:   "Process is running",
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"process_name": pc.processName,
			"pid_file":     pc.pidFile,
		},
	}, nil
}

// GetName implements HealthChecker interface for Process
func (pc *ProcessHealthChecker) GetName() string {
	return pc.name
}

// CreateORANHealthCheckers creates standard O-RAN component health checkers
func CreateORANHealthCheckers() []HealthChecker {
	return []HealthChecker{
		NewHTTPHealthChecker("o-cu", "http://oai-cu:8080/health", 10*time.Second),
		NewHTTPHealthChecker("o-du", "http://oai-du:8080/health", 10*time.Second),
		NewHTTPHealthChecker("o-ru", "http://oai-ru:8080/health", 10*time.Second),
		NewHTTPHealthChecker("ric-platform", "http://ric-plt-e2mgr:3800/health", 10*time.Second),
		NewHTTPHealthChecker("ves-collector", "http://ves-collector:8443/health", 10*time.Second),
		NewKubernetesHealthChecker("porch-server", "porch-system", "app=porch-server"),
		NewKubernetesHealthChecker("nephio-controller", "nephio-system", "app=nephio-controller"),
	}
}

// HealthCheckResult represents a health check result for external consumption
type HealthCheckResult struct {
	Component string            `json:"component"`
	Status    string            `json:"status"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ToHealthCheckResult converts ComponentHealth to HealthCheckResult
func (ch *ComponentHealth) ToHealthCheckResult() *HealthCheckResult {
	return &HealthCheckResult{
		Component: ch.Name,
		Status:    string(ch.Status),
		Message:   ch.Message,
		Timestamp: ch.Timestamp,
		Metadata:  ch.Metadata,
	}
}