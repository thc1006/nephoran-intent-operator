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

package krm

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"k8s.io/apimachinery/pkg/api/resource"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ContainerRuntime provides secure container-based KRM function execution.

type ContainerRuntime struct {
	config *ContainerRuntimeConfig

	containerMgr *ContainerManager

	sandboxMgr *SandboxManager

	resourceMgr *ResourceManager

	securityMgr *SecurityManager

	networkMgr *NetworkManager

	metrics *ContainerRuntimeMetrics

	tracer trace.Tracer

	mu sync.RWMutex
}

// ContainerRuntimeConfig defines configuration for container runtime.

type ContainerRuntimeConfig struct {

	// Container runtime settings.

	RuntimeType string `json:"runtimeType" yaml:"runtimeType"` // docker, containerd, cri-o

	RuntimeSocket string `json:"runtimeSocket" yaml:"runtimeSocket"` // Runtime socket path

	TimeoutDefault time.Duration `json:"timeoutDefault" yaml:"timeoutDefault"` // Default execution timeout

	TimeoutMax time.Duration `json:"timeoutMax" yaml:"timeoutMax"` // Maximum allowed timeout

	// Resource limits.

	DefaultCPULimit string `json:"defaultCpuLimit" yaml:"defaultCpuLimit"` // Default CPU limit

	DefaultMemoryLimit string `json:"defaultMemoryLimit" yaml:"defaultMemoryLimit"` // Default memory limit

	DefaultDiskLimit string `json:"defaultDiskLimit" yaml:"defaultDiskLimit"` // Default disk limit

	MaxCPULimit string `json:"maxCpuLimit" yaml:"maxCpuLimit"` // Maximum CPU limit

	MaxMemoryLimit string `json:"maxMemoryLimit" yaml:"maxMemoryLimit"` // Maximum memory limit

	MaxDiskLimit string `json:"maxDiskLimit" yaml:"maxDiskLimit"` // Maximum disk limit

	// Security settings.

	SandboxEnabled bool `json:"sandboxEnabled" yaml:"sandboxEnabled"` // Enable sandboxing

	AllowPrivileged bool `json:"allowPrivileged" yaml:"allowPrivileged"` // Allow privileged containers

	AllowedRegistries []string `json:"allowedRegistries" yaml:"allowedRegistries"` // Allowed container registries

	TrustedImages []string `json:"trustedImages" yaml:"trustedImages"` // Pre-approved trusted images

	RequiredCapabilities []string `json:"requiredCapabilities" yaml:"requiredCapabilities"` // Required capabilities

	DroppedCapabilities []string `json:"droppedCapabilities" yaml:"droppedCapabilities"` // Capabilities to drop

	ReadOnlyRootFS bool `json:"readOnlyRootFS" yaml:"readOnlyRootFS"` // Read-only root filesystem

	NoNewPrivileges bool `json:"noNewPrivileges" yaml:"noNewPrivileges"` // No new privileges

	// Networking.

	NetworkMode string `json:"networkMode" yaml:"networkMode"` // Network mode (none, bridge, host)

	AllowedNetworks []string `json:"allowedNetworks" yaml:"allowedNetworks"` // Allowed networks

	DNSServers []string `json:"dnsServers" yaml:"dnsServers"` // DNS servers

	// Storage and filesystem.

	WorkspaceDir string `json:"workspaceDir" yaml:"workspaceDir"` // Base workspace directory

	TempDir string `json:"tempDir" yaml:"tempDir"` // Temporary directory

	AllowedMountPaths []string `json:"allowedMountPaths" yaml:"allowedMountPaths"` // Allowed mount paths

	ReadOnlyMounts []string `json:"readOnlyMounts" yaml:"readOnlyMounts"` // Read-only mounts

	// Monitoring and observability.

	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"` // Enable metrics collection

	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"` // Enable tracing

	EnableProfiling bool `json:"enableProfiling" yaml:"enableProfiling"` // Enable profiling

	EnableAuditLogging bool `json:"enableAuditLogging" yaml:"enableAuditLogging"` // Enable audit logging

	// Performance optimization.

	CacheEnabled bool `json:"cacheEnabled" yaml:"cacheEnabled"` // Enable caching

	CacheSize int64 `json:"cacheSize" yaml:"cacheSize"` // Cache size in bytes

	ImagePullPolicy string `json:"imagePullPolicy" yaml:"imagePullPolicy"` // Image pull policy

	EnableParallelism bool `json:"enableParallelism" yaml:"enableParallelism"` // Enable parallel execution

	MaxConcurrentContainers int `json:"maxConcurrentContainers" yaml:"maxConcurrentContainers"` // Max concurrent containers

	CleanupInterval time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"` // Cleanup interval

}

// ContainerRequest represents a container execution request.

type ContainerRequest struct {

	// Container specification.

	Image string `json:"image"`

	Tag string `json:"tag,omitempty"`

	Command []string `json:"command,omitempty"`

	Args []string `json:"args,omitempty"`

	Environment map[string]string `json:"environment,omitempty"`

	WorkingDir string `json:"workingDir,omitempty"`

	// Resource constraints.

	CPULimit string `json:"cpuLimit,omitempty"`

	MemoryLimit string `json:"memoryLimit,omitempty"`

	DiskLimit string `json:"diskLimit,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	// Security settings.

	User string `json:"user,omitempty"`

	Group string `json:"group,omitempty"`

	Capabilities []string `json:"capabilities,omitempty"`

	Privileged bool `json:"privileged,omitempty"`

	ReadOnlyRootFS bool `json:"readOnlyRootFS,omitempty"`

	// Networking.

	NetworkMode string `json:"networkMode,omitempty"`

	ExposedPorts []int `json:"exposedPorts,omitempty"`

	// Storage.

	Mounts []*VolumeMount `json:"mounts,omitempty"`

	TempFS []string `json:"tempfs,omitempty"`

	// Input/Output.

	Stdin []byte `json:"stdin,omitempty"`

	InputFiles map[string][]byte `json:"inputFiles,omitempty"`

	// Metadata.

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	// Execution context.

	Context context.Context `json:"-"`

	RequestID string `json:"requestId,omitempty"`

	FunctionName string `json:"functionName,omitempty"`
}

// ContainerResponse represents container execution response.

type ContainerResponse struct {

	// Execution results.

	ExitCode int `json:"exitCode"`

	Stdout []byte `json:"stdout,omitempty"`

	Stderr []byte `json:"stderr,omitempty"`

	OutputFiles map[string][]byte `json:"outputFiles,omitempty"`

	// Execution metadata.

	ContainerID string `json:"containerId,omitempty"`

	Duration time.Duration `json:"duration"`

	StartTime time.Time `json:"startTime"`

	EndTime time.Time `json:"endTime"`

	// Resource usage.

	CPUUsage *resource.Quantity `json:"cpuUsage,omitempty"`

	MemoryUsage *resource.Quantity `json:"memoryUsage,omitempty"`

	DiskUsage *resource.Quantity `json:"diskUsage,omitempty"`

	NetworkIO *NetworkIOStats `json:"networkIO,omitempty"`

	// Error information.

	Error error `json:"error,omitempty"`

	ErrorCode string `json:"errorCode,omitempty"`

	ErrorDetails string `json:"errorDetails,omitempty"`

	// Security and audit.

	SecurityEvents []*SecurityEvent `json:"securityEvents,omitempty"`

	AuditLogs []string `json:"auditLogs,omitempty"`
}

// VolumeMount represents a volume mount.

type VolumeMount struct {
	Source string `json:"source"`

	Target string `json:"target"`

	Type string `json:"type,omitempty"` // bind, volume, tmpfs

	ReadOnly bool `json:"readOnly,omitempty"`

	Propagation string `json:"propagation,omitempty"`
}

// NetworkIOStats represents network I/O statistics.

type NetworkIOStats struct {
	BytesReceived int64 `json:"bytesReceived"`

	BytesSent int64 `json:"bytesSent"`

	PacketsReceived int64 `json:"packetsReceived"`

	PacketsSent int64 `json:"packetsSent"`
}

// SecurityEvent represents a security-related event.

type SecurityEvent struct {
	Type string `json:"type"`

	Severity string `json:"severity"`

	Message string `json:"message"`

	Details map[string]interface{} `json:"details,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// ContainerManager manages container lifecycle.

type ContainerManager struct {
	runtimeType string

	runtimeSocket string

	containers map[string]*ContainerInstance

	mu sync.RWMutex
}

// ContainerInstance represents a running container instance.

type ContainerInstance struct {
	ID string

	Name string

	Image string

	Status string

	StartTime time.Time

	Request *ContainerRequest

	Process *exec.Cmd

	mu sync.Mutex
}

// SandboxManager manages container sandboxing.

type SandboxManager struct {
	enabled bool

	sandboxes map[string]*Sandbox

	mu sync.RWMutex
}

// Sandbox represents a container sandbox.

type Sandbox struct {
	ID string

	Name string

	NetworkNS string

	PidNS string

	UserNS string

	MountNS string

	Capabilities []string

	SELinuxLabel string

	AppArmorProfile string

	SeccompProfile string

	CreatedAt time.Time
}

// ResourceManager manages resource allocation and monitoring.

type ResourceManager struct {
	cpuQuota *resource.Quantity

	memoryQuota *resource.Quantity

	diskQuota *resource.Quantity

	allocations map[string]*ResourceAllocation

	mu sync.RWMutex
}

// ResourceAllocation tracks resource allocation for a container.

type ResourceAllocation struct {
	ContainerID string

	CPULimit *resource.Quantity

	MemoryLimit *resource.Quantity

	DiskLimit *resource.Quantity

	CPUUsage *resource.Quantity

	MemoryUsage *resource.Quantity

	DiskUsage *resource.Quantity

	AllocationTime time.Time
}

// SecurityManager enforces security policies.

type SecurityManager struct {
	allowedRegistries map[string]bool

	trustedImages map[string]bool

	securityPolicies []*SecurityPolicy

	violations []*SecurityViolation

	mu sync.RWMutex
}

// SecurityPolicy defines security constraints.

type SecurityPolicy struct {
	Name string

	AllowPrivileged bool

	AllowedCapabilities []string

	DroppedCapabilities []string

	ReadOnlyRootFS bool

	NoNewPrivileges bool

	UserNamespaces bool

	NetworkPolicies []*NetworkSecurityPolicy

	allowedImages map[string]bool

	blockedCapabilities map[string]bool

	maxResourceLimits ResourceLimits

	networkPolicy *NetworkPolicy

	fileSystemPolicy *FileSystemPolicy
}

// NetworkSecurityPolicy defines network security constraints.

type NetworkSecurityPolicy struct {
	Mode string // none, bridge, host

	AllowedPorts []int

	BlockedPorts []int

	AllowedHosts []string

	BlockedHosts []string

	DNSPolicies []string
}

// SecurityViolation records security violations.

type SecurityViolation struct {
	ContainerID string

	ViolationType string

	Message string

	Severity string

	Timestamp time.Time

	Details map[string]interface{}
}

// NetworkManager manages container networking.

type NetworkManager struct {
	networks map[string]*Network

	defaultDNS []string

	mu sync.RWMutex
}

// Network represents a container network.

type Network struct {
	Name string

	Driver string

	IPAM *IPAMConfig

	Options map[string]string

	CreatedAt time.Time
}

// IPAMConfig represents IP Address Management configuration.

type IPAMConfig struct {
	Driver string

	Options map[string]string

	Subnets []*IPAMSubnet
}

// IPAMSubnet represents an IPAM subnet.

type IPAMSubnet struct {
	Subnet string

	Gateway string
}

// ContainerRuntimeMetrics provides comprehensive metrics.

type ContainerRuntimeMetrics struct {
	ContainersStarted *prometheus.CounterVec

	ContainerExecutions *prometheus.HistogramVec

	ResourceUtilization *prometheus.GaugeVec

	SecurityViolations *prometheus.CounterVec

	NetworkIOBytes *prometheus.CounterVec

	ErrorRate *prometheus.CounterVec

	CacheHitRate prometheus.Counter

	ActiveContainers prometheus.Gauge
}

// Default configuration.

var DefaultContainerRuntimeConfig = &ContainerRuntimeConfig{

	RuntimeType: "docker",

	RuntimeSocket: "/var/run/docker.sock",

	TimeoutDefault: 5 * time.Minute,

	TimeoutMax: 30 * time.Minute,

	DefaultCPULimit: "1000m",

	DefaultMemoryLimit: "512Mi",

	DefaultDiskLimit: "1Gi",

	MaxCPULimit: "4000m",

	MaxMemoryLimit: "2Gi",

	MaxDiskLimit: "10Gi",

	SandboxEnabled: true,

	AllowPrivileged: false,

	ReadOnlyRootFS: true,

	NoNewPrivileges: true,

	NetworkMode: "none",

	WorkspaceDir: "/tmp/krm-containers",

	TempDir: "/tmp",

	EnableMetrics: true,

	EnableTracing: true,

	EnableAuditLogging: true,

	CacheEnabled: true,

	CacheSize: 1024 * 1024 * 1024, // 1GB

	ImagePullPolicy: "IfNotPresent",

	EnableParallelism: true,

	MaxConcurrentContainers: 50,

	CleanupInterval: 10 * time.Minute,

	DroppedCapabilities: []string{"ALL"},
}

// NewContainerRuntime creates a new container runtime.

func NewContainerRuntime(config *ContainerRuntimeConfig) (*ContainerRuntime, error) {

	if config == nil {

		config = DefaultContainerRuntimeConfig

	}

	// Validate configuration.

	if err := validateContainerRuntimeConfig(config); err != nil {

		return nil, fmt.Errorf("invalid container runtime configuration: %w", err)

	}

	// Initialize metrics.

	metrics := &ContainerRuntimeMetrics{

		ContainersStarted: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_container_starts_total",

				Help: "Total number of container starts",
			},

			[]string{"image", "function", "status"},
		),

		ContainerExecutions: promauto.NewHistogramVec(

			prometheus.HistogramOpts{

				Name: "krm_container_execution_duration_seconds",

				Help: "Duration of container executions",

				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},

			[]string{"image", "function"},
		),

		ResourceUtilization: promauto.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "krm_container_resource_utilization",

				Help: "Container resource utilization",
			},

			[]string{"resource_type", "container_id"},
		),

		SecurityViolations: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_container_security_violations_total",

				Help: "Total number of security violations",
			},

			[]string{"violation_type", "severity"},
		),

		NetworkIOBytes: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_container_network_io_bytes_total",

				Help: "Total container network I/O bytes",
			},

			[]string{"direction", "container_id"},
		),

		ErrorRate: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_container_errors_total",

				Help: "Total number of container execution errors",
			},

			[]string{"error_type", "image"},
		),

		CacheHitRate: promauto.NewCounter(

			prometheus.CounterOpts{

				Name: "krm_container_cache_hits_total",

				Help: "Total number of container cache hits",
			},
		),

		ActiveContainers: promauto.NewGauge(

			prometheus.GaugeOpts{

				Name: "krm_container_active_count",

				Help: "Number of active containers",
			},
		),
	}

	// Initialize managers.

	containerMgr := &ContainerManager{

		runtimeType: config.RuntimeType,

		runtimeSocket: config.RuntimeSocket,

		containers: make(map[string]*ContainerInstance),
	}

	sandboxMgr := &SandboxManager{

		enabled: config.SandboxEnabled,

		sandboxes: make(map[string]*Sandbox),
	}

	resourceMgr := &ResourceManager{

		allocations: make(map[string]*ResourceAllocation),
	}

	// Parse resource limits.

	if config.MaxCPULimit != "" {

		if cpuQuota, err := resource.ParseQuantity(config.MaxCPULimit); err == nil {

			resourceMgr.cpuQuota = &cpuQuota

		}

	}

	if config.MaxMemoryLimit != "" {

		if memoryQuota, err := resource.ParseQuantity(config.MaxMemoryLimit); err == nil {

			resourceMgr.memoryQuota = &memoryQuota

		}

	}

	if config.MaxDiskLimit != "" {

		if diskQuota, err := resource.ParseQuantity(config.MaxDiskLimit); err == nil {

			resourceMgr.diskQuota = &diskQuota

		}

	}

	securityMgr := &SecurityManager{

		allowedRegistries: make(map[string]bool),

		trustedImages: make(map[string]bool),

		securityPolicies: []*SecurityPolicy{},

		violations: []*SecurityViolation{},
	}

	// Populate security manager.

	for _, registry := range config.AllowedRegistries {

		securityMgr.allowedRegistries[registry] = true

	}

	for _, image := range config.TrustedImages {

		securityMgr.trustedImages[image] = true

	}

	networkMgr := &NetworkManager{

		networks: make(map[string]*Network),

		defaultDNS: config.DNSServers,
	}

	runtime := &ContainerRuntime{

		config: config,

		containerMgr: containerMgr,

		sandboxMgr: sandboxMgr,

		resourceMgr: resourceMgr,

		securityMgr: securityMgr,

		networkMgr: networkMgr,

		metrics: metrics,

		tracer: otel.Tracer("krm-container-runtime"),
	}

	// Create workspace directory.

	if err := os.MkdirAll(config.WorkspaceDir, 0o755); err != nil {

		return nil, fmt.Errorf("failed to create workspace directory: %w", err)

	}

	// Start background cleanup.

	go runtime.backgroundCleanup()

	return runtime, nil

}

// ExecuteContainer executes a container with the given request.

func (cr *ContainerRuntime) ExecuteContainer(ctx context.Context, req *ContainerRequest) (*ContainerResponse, error) {

	ctx, span := cr.tracer.Start(ctx, "container-execution")

	defer span.End()

	// Generate container ID.

	containerID := generateContainerID()

	req.RequestID = containerID

	span.SetAttributes(

		attribute.String("container.id", containerID),

		attribute.String("container.image", req.Image),

		attribute.String("function.name", req.FunctionName),
	)

	logger := log.FromContext(ctx).WithName("container-runtime").WithValues(

		"containerId", containerID,

		"image", req.Image,

		"function", req.FunctionName,
	)

	logger.Info("Starting container execution")

	startTime := time.Now()

	// Security validation.

	if err := cr.validateSecurity(ctx, req); err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "security validation failed")

		cr.metrics.SecurityViolations.WithLabelValues("validation_failed", "high").Inc()

		return nil, fmt.Errorf("security validation failed: %w", err)

	}

	// Resource allocation.

	if err := cr.allocateResources(ctx, req); err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "resource allocation failed")

		return nil, fmt.Errorf("resource allocation failed: %w", err)

	}

	defer cr.releaseResources(containerID)

	// Create sandbox if enabled.

	var sandbox *Sandbox

	var err error

	if cr.config.SandboxEnabled {

		sandbox, err = cr.createSandbox(ctx, req)

		if err != nil {

			span.RecordError(err)

			span.SetStatus(codes.Error, "sandbox creation failed")

			return nil, fmt.Errorf("sandbox creation failed: %w", err)

		}

		defer cr.destroySandbox(sandbox.ID)

	}

	// Execute container.

	response, err := cr.executeContainerInternal(ctx, req, sandbox)

	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "container execution failed")

		cr.metrics.ErrorRate.WithLabelValues("execution_failed", req.Image).Inc()

		return nil, err

	}

	// Record metrics.

	duration := time.Since(startTime)

	cr.metrics.ContainerExecutions.WithLabelValues(req.Image, req.FunctionName).Observe(duration.Seconds())

	status := "success"

	if response.ExitCode != 0 {

		status = "failed"

	}

	cr.metrics.ContainersStarted.WithLabelValues(req.Image, req.FunctionName, status).Inc()

	// Record resource usage.

	if response.CPUUsage != nil {

		cr.metrics.ResourceUtilization.WithLabelValues("cpu", containerID).Set(float64(response.CPUUsage.MilliValue()))

	}

	if response.MemoryUsage != nil {

		cr.metrics.ResourceUtilization.WithLabelValues("memory", containerID).Set(float64(response.MemoryUsage.Value()))

	}

	logger.Info("Container execution completed",

		"exitCode", response.ExitCode,

		"duration", duration,

		"status", status)

	span.SetStatus(codes.Ok, "container execution completed")

	return response, nil

}

// Private methods.

func (cr *ContainerRuntime) validateSecurity(ctx context.Context, req *ContainerRequest) error {

	// Validate image registry.

	if len(cr.config.AllowedRegistries) > 0 {

		allowed := false

		for _, registry := range cr.config.AllowedRegistries {

			if strings.HasPrefix(req.Image, registry) {

				allowed = true

				break

			}

		}

		if !allowed {

			return fmt.Errorf("image %s is not from allowed registry", req.Image)

		}

	}

	// Validate privileged access.

	if req.Privileged && !cr.config.AllowPrivileged {

		return fmt.Errorf("privileged containers are not allowed")

	}

	// Validate capabilities.

	for _, cap := range req.Capabilities {

		if contains(cr.config.DroppedCapabilities, cap) {

			return fmt.Errorf("capability %s is not allowed", cap)

		}

	}

	return nil

}

func (cr *ContainerRuntime) allocateResources(ctx context.Context, req *ContainerRequest) error {

	cr.resourceMgr.mu.Lock()

	defer cr.resourceMgr.mu.Unlock()

	// Parse resource requirements.

	var cpuLimit, memoryLimit, diskLimit *resource.Quantity

	if req.CPULimit != "" {

		quantity, err := resource.ParseQuantity(req.CPULimit)

		if err != nil {

			return fmt.Errorf("invalid CPU limit: %w", err)

		}

		cpuLimit = &quantity

	} else {

		quantity, err := resource.ParseQuantity(cr.config.DefaultCPULimit)

		if err != nil {

			return fmt.Errorf("invalid default CPU limit: %w", err)

		}

		cpuLimit = &quantity

	}

	if req.MemoryLimit != "" {

		quantity, err := resource.ParseQuantity(req.MemoryLimit)

		if err != nil {

			return fmt.Errorf("invalid memory limit: %w", err)

		}

		memoryLimit = &quantity

	} else {

		quantity, err := resource.ParseQuantity(cr.config.DefaultMemoryLimit)

		if err != nil {

			return fmt.Errorf("invalid default memory limit: %w", err)

		}

		memoryLimit = &quantity

	}

	if req.DiskLimit != "" {

		quantity, err := resource.ParseQuantity(req.DiskLimit)

		if err != nil {

			return fmt.Errorf("invalid disk limit: %w", err)

		}

		diskLimit = &quantity

	} else {

		quantity, err := resource.ParseQuantity(cr.config.DefaultDiskLimit)

		if err != nil {

			return fmt.Errorf("invalid default disk limit: %w", err)

		}

		diskLimit = &quantity

	}

	// Check against quotas.

	if cr.resourceMgr.cpuQuota != nil && cpuLimit.Cmp(*cr.resourceMgr.cpuQuota) > 0 {

		return fmt.Errorf("CPU limit exceeds quota")

	}

	if cr.resourceMgr.memoryQuota != nil && memoryLimit.Cmp(*cr.resourceMgr.memoryQuota) > 0 {

		return fmt.Errorf("memory limit exceeds quota")

	}

	if cr.resourceMgr.diskQuota != nil && diskLimit.Cmp(*cr.resourceMgr.diskQuota) > 0 {

		return fmt.Errorf("disk limit exceeds quota")

	}

	// Record allocation.

	allocation := &ResourceAllocation{

		ContainerID: req.RequestID,

		CPULimit: cpuLimit,

		MemoryLimit: memoryLimit,

		DiskLimit: diskLimit,

		AllocationTime: time.Now(),
	}

	cr.resourceMgr.allocations[req.RequestID] = allocation

	return nil

}

func (cr *ContainerRuntime) releaseResources(containerID string) {

	cr.resourceMgr.mu.Lock()

	defer cr.resourceMgr.mu.Unlock()

	delete(cr.resourceMgr.allocations, containerID)

}

func (cr *ContainerRuntime) createSandbox(ctx context.Context, req *ContainerRequest) (*Sandbox, error) {

	sandboxID := generateSandboxID()

	sandbox := &Sandbox{

		ID: sandboxID,

		Name: fmt.Sprintf("sandbox-%s", req.FunctionName),

		Capabilities: cr.config.RequiredCapabilities,

		CreatedAt: time.Now(),
	}

	cr.sandboxMgr.mu.Lock()

	cr.sandboxMgr.sandboxes[sandboxID] = sandbox

	cr.sandboxMgr.mu.Unlock()

	return sandbox, nil

}

func (cr *ContainerRuntime) destroySandbox(sandboxID string) {

	cr.sandboxMgr.mu.Lock()

	defer cr.sandboxMgr.mu.Unlock()

	delete(cr.sandboxMgr.sandboxes, sandboxID)

}

func (cr *ContainerRuntime) executeContainerInternal(ctx context.Context, req *ContainerRequest, sandbox *Sandbox) (*ContainerResponse, error) {

	// Create workspace directory.

	workspaceDir := filepath.Join(cr.config.WorkspaceDir, req.RequestID)

	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {

		return nil, fmt.Errorf("failed to create workspace: %w", err)

	}

	defer os.RemoveAll(workspaceDir)

	// Write input files.

	if err := cr.writeInputFiles(workspaceDir, req.InputFiles); err != nil {

		return nil, fmt.Errorf("failed to write input files: %w", err)

	}

	// Create container command.

	cmd, err := cr.createContainerCommand(ctx, req, workspaceDir, sandbox)

	if err != nil {

		return nil, fmt.Errorf("failed to create container command: %w", err)

	}

	// Execute container.

	return cr.runContainer(ctx, cmd, req, workspaceDir)

}

func (cr *ContainerRuntime) createContainerCommand(ctx context.Context, req *ContainerRequest, workspaceDir string, sandbox *Sandbox) (*exec.Cmd, error) {

	args := []string{"run", "--rm"}

	// Basic container settings.

	args = append(args, "--name", fmt.Sprintf("krm-%s", req.RequestID))

	// Resource limits.

	allocation := cr.resourceMgr.allocations[req.RequestID]

	if allocation != nil {

		if allocation.CPULimit != nil {

			args = append(args, "--cpus", allocation.CPULimit.AsDec().String())

		}

		if allocation.MemoryLimit != nil {

			args = append(args, "--memory", allocation.MemoryLimit.String())

		}

	}

	// Security settings.

	if cr.config.ReadOnlyRootFS {

		args = append(args, "--read-only")

	}

	if cr.config.NoNewPrivileges {

		args = append(args, "--security-opt", "no-new-privileges:true")

	}

	// User settings.

	if req.User != "" {

		if req.Group != "" {

			args = append(args, "--user", fmt.Sprintf("%s:%s", req.User, req.Group))

		} else {

			args = append(args, "--user", req.User)

		}

	}

	// Capabilities.

	for _, cap := range cr.config.DroppedCapabilities {

		args = append(args, "--cap-drop", cap)

	}

	for _, cap := range req.Capabilities {

		args = append(args, "--cap-add", cap)

	}

	// Network settings.

	networkMode := req.NetworkMode

	if networkMode == "" {

		networkMode = cr.config.NetworkMode

	}

	args = append(args, "--network", networkMode)

	// Volume mounts.

	args = append(args, "--volume", fmt.Sprintf("%s:/workspace", workspaceDir))

	args = append(args, "--workdir", "/workspace")

	for _, mount := range req.Mounts {

		if cr.isAllowedMount(mount.Source) {

			mountStr := fmt.Sprintf("%s:%s", mount.Source, mount.Target)

			if mount.ReadOnly {

				mountStr += ":ro"

			}

			args = append(args, "--volume", mountStr)

		}

	}

	// Environment variables.

	for key, value := range req.Environment {

		args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))

	}

	// Tmpfs mounts.

	for _, tmpfs := range req.TempFS {

		args = append(args, "--tmpfs", tmpfs)

	}

	// Labels.

	for key, value := range req.Labels {

		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))

	}

	// Container image.

	image := req.Image

	if req.Tag != "" {

		image = fmt.Sprintf("%s:%s", req.Image, req.Tag)

	}

	args = append(args, image)

	// Command and arguments.

	if len(req.Command) > 0 {

		args = append(args, req.Command...)

	}

	if len(req.Args) > 0 {

		args = append(args, req.Args...)

	}

	// Create command.

	cmd := exec.CommandContext(ctx, cr.getRuntimeCommand(), args...)

	return cmd, nil

}

func (cr *ContainerRuntime) runContainer(ctx context.Context, cmd *exec.Cmd, req *ContainerRequest, workspaceDir string) (*ContainerResponse, error) {

	startTime := time.Now()

	// Set up pipes.

	stdout, err := cmd.StdoutPipe()

	if err != nil {

		return nil, err

	}

	stderr, err := cmd.StderrPipe()

	if err != nil {

		return nil, err

	}

	// Set up stdin if provided.

	if len(req.Stdin) > 0 {

		stdin, err := cmd.StdinPipe()

		if err != nil {

			return nil, err

		}

		go func() {

			defer stdin.Close()

			stdin.Write(req.Stdin)

		}()

	}

	// Start container.

	if err := cmd.Start(); err != nil {

		return nil, err

	}

	cr.metrics.ActiveContainers.Inc()

	defer cr.metrics.ActiveContainers.Dec()

	// Read output.

	var stdoutBuf, stderrBuf []byte

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {

		defer wg.Done()

		if data, err := io.ReadAll(stdout); err == nil {

			stdoutBuf = data

		}

	}()

	go func() {

		defer wg.Done()

		if data, err := io.ReadAll(stderr); err == nil {

			stderrBuf = data

		}

	}()

	// Wait for completion.

	err = cmd.Wait()

	wg.Wait()

	endTime := time.Now()

	exitCode := 0

	if err != nil {

		if exitError, ok := err.(*exec.ExitError); ok {

			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {

				exitCode = status.ExitStatus()

			}

		}

	}

	// Read output files.

	outputFiles, err := cr.readOutputFiles(workspaceDir)

	if err != nil {

		log.FromContext(ctx).Error(err, "Failed to read output files")

	}

	// Get resource usage (simplified).

	cpuUsage, memoryUsage := cr.getResourceUsage(req.RequestID)

	response := &ContainerResponse{

		ExitCode: exitCode,

		Stdout: stdoutBuf,

		Stderr: stderrBuf,

		OutputFiles: outputFiles,

		ContainerID: req.RequestID,

		Duration: endTime.Sub(startTime),

		StartTime: startTime,

		EndTime: endTime,

		CPUUsage: cpuUsage,

		MemoryUsage: memoryUsage,
	}

	if err != nil && exitCode == 0 {

		response.Error = err

	}

	return response, nil

}

func (cr *ContainerRuntime) writeInputFiles(workspaceDir string, inputFiles map[string][]byte) error {

	for filename, content := range inputFiles {

		filePath := filepath.Join(workspaceDir, filename)

		// Ensure directory exists.

		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {

			return err

		}

		if err := os.WriteFile(filePath, content, 0o640); err != nil {

			return err

		}

	}

	return nil

}

func (cr *ContainerRuntime) readOutputFiles(workspaceDir string) (map[string][]byte, error) {

	outputFiles := make(map[string][]byte)

	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {

		if err != nil {

			return err

		}

		if !info.IsDir() {

			relPath, err := filepath.Rel(workspaceDir, path)

			if err != nil {

				return err

			}

			content, err := os.ReadFile(path)

			if err != nil {

				return err

			}

			outputFiles[relPath] = content

		}

		return nil

	})

	return outputFiles, err

}

func (cr *ContainerRuntime) getResourceUsage(containerID string) (*resource.Quantity, *resource.Quantity) {

	// This would integrate with the container runtime to get actual resource usage.

	// For now, return placeholder values.

	cpu := resource.MustParse("100m")

	memory := resource.MustParse("128Mi")

	return &cpu, &memory

}

func (cr *ContainerRuntime) getRuntimeCommand() string {

	switch cr.config.RuntimeType {

	case "containerd":

		return "ctr"

	case "cri-o":

		return "podman"

	default:

		return "docker"

	}

}

func (cr *ContainerRuntime) isAllowedMount(path string) bool {

	for _, allowedPath := range cr.config.AllowedMountPaths {

		if strings.HasPrefix(path, allowedPath) {

			return true

		}

	}

	return false

}

func (cr *ContainerRuntime) backgroundCleanup() {

	ticker := time.NewTicker(cr.config.CleanupInterval)

	defer ticker.Stop()

	for range ticker.C {

		cr.performCleanup()

	}

}

func (cr *ContainerRuntime) performCleanup() {

	// Cleanup old containers.

	cr.containerMgr.mu.Lock()

	for id, container := range cr.containerMgr.containers {

		if time.Since(container.StartTime) > cr.config.CleanupInterval {

			delete(cr.containerMgr.containers, id)

		}

	}

	cr.containerMgr.mu.Unlock()

	// Cleanup old sandboxes.

	cr.sandboxMgr.mu.Lock()

	for id, sandbox := range cr.sandboxMgr.sandboxes {

		if time.Since(sandbox.CreatedAt) > cr.config.CleanupInterval {

			delete(cr.sandboxMgr.sandboxes, id)

		}

	}

	cr.sandboxMgr.mu.Unlock()

}

// Health returns runtime health status.

func (cr *ContainerRuntime) Health() *ContainerRuntimeHealth {

	cr.mu.RLock()

	defer cr.mu.RUnlock()

	return &ContainerRuntimeHealth{

		Status: "healthy",

		ActiveContainers: len(cr.containerMgr.containers),

		ActiveSandboxes: len(cr.sandboxMgr.sandboxes),

		ResourceAllocations: len(cr.resourceMgr.allocations),

		SecurityViolations: len(cr.securityMgr.violations),

		LastHealthCheck: time.Now(),
	}

}

// ContainerRuntimeHealth represents health status.

type ContainerRuntimeHealth struct {
	Status string `json:"status"`

	ActiveContainers int `json:"activeContainers"`

	ActiveSandboxes int `json:"activeSandboxes"`

	ResourceAllocations int `json:"resourceAllocations"`

	SecurityViolations int `json:"securityViolations"`

	LastHealthCheck time.Time `json:"lastHealthCheck"`
}

// Shutdown gracefully shuts down the container runtime.

func (cr *ContainerRuntime) Shutdown(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("container-runtime")

	logger.Info("Shutting down container runtime")

	// Stop all active containers.

	cr.containerMgr.mu.Lock()

	for id, container := range cr.containerMgr.containers {

		if container.Process != nil {

			container.Process.Process.Kill()

		}

		delete(cr.containerMgr.containers, id)

	}

	cr.containerMgr.mu.Unlock()

	// Cleanup sandboxes.

	cr.sandboxMgr.mu.Lock()

	for id := range cr.sandboxMgr.sandboxes {

		delete(cr.sandboxMgr.sandboxes, id)

	}

	cr.sandboxMgr.mu.Unlock()

	logger.Info("Container runtime shutdown complete")

	return nil

}

// Helper functions.

func validateContainerRuntimeConfig(config *ContainerRuntimeConfig) error {

	if config.RuntimeType == "" {

		return fmt.Errorf("runtimeType is required")

	}

	if config.WorkspaceDir == "" {

		return fmt.Errorf("workspaceDir is required")

	}

	if config.TimeoutDefault <= 0 {

		return fmt.Errorf("timeoutDefault must be positive")

	}

	if config.MaxConcurrentContainers <= 0 {

		return fmt.Errorf("maxConcurrentContainers must be positive")

	}

	return nil

}

func generateContainerID() string {

	return fmt.Sprintf("container-%d-%d", time.Now().UnixNano(), runtime.NumGoroutine())

}

func generateSandboxID() string {

	return fmt.Sprintf("sandbox-%d-%d", time.Now().UnixNano(), runtime.NumGoroutine())

}

func contains(slice []string, item string) bool {

	for _, s := range slice {

		if s == item {

			return true

		}

	}

	return false

}
