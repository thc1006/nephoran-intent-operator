// Package o2 implements supporting types and request/response models for resource lifecycle management
package o2

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Request types for resource lifecycle operations

// ProvisionResourceRequest represents a request to provision a new resource
type ProvisionResourceRequest struct {
	Name             string                 `json:"name" validate:"required"`
	ResourceType     string                 `json:"resource_type" validate:"required"`
	ResourcePoolID   string                 `json:"resource_pool_id" validate:"required"`
	Provider         string                 `json:"provider" validate:"required"`
	Configuration    *runtime.RawExtension  `json:"configuration" validate:"required"`
	ResourceSpec     *ResourceSpec          `json:"resource_spec,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`
	Annotations      map[string]string      `json:"annotations,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	ProvisionOptions *ProvisionOptions      `json:"provision_options,omitempty"`
}

// ResourceSpec defines the specifications for a resource
type ResourceSpec struct {
	CPU             string                 `json:"cpu,omitempty"`
	Memory          string                 `json:"memory,omitempty"`
	Storage         string                 `json:"storage,omitempty"`
	Network         *NetworkSpec           `json:"network,omitempty"`
	SecurityContext *SecurityContext       `json:"security_context,omitempty"`
	Constraints     *ResourceConstraints   `json:"constraints,omitempty"`
	CustomResources map[string]interface{} `json:"custom_resources,omitempty"`
}

// NetworkSpec defines network specifications for a resource
type NetworkSpec struct {
	NetworkType    string            `json:"network_type,omitempty"`
	Subnets        []string          `json:"subnets,omitempty"`
	SecurityGroups []string          `json:"security_groups,omitempty"`
	LoadBalancer   *LoadBalancerSpec `json:"load_balancer,omitempty"`
	Ingress        *IngressSpec      `json:"ingress,omitempty"`
	DNS            *DNSSpec          `json:"dns,omitempty"`
	Bandwidth      string            `json:"bandwidth,omitempty"`
}

// LoadBalancerSpec defines load balancer specifications
type LoadBalancerSpec struct {
	Type            string            `json:"type,omitempty"`
	Algorithm       string            `json:"algorithm,omitempty"`
	HealthCheck     *HealthCheckSpec  `json:"health_check,omitempty"`
	SessionAffinity string            `json:"session_affinity,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

// HealthCheckSpec defines health check specifications
type HealthCheckSpec struct {
	Protocol           string        `json:"protocol,omitempty"`
	Port               int32         `json:"port,omitempty"`
	Path               string        `json:"path,omitempty"`
	Interval           time.Duration `json:"interval,omitempty"`
	Timeout            time.Duration `json:"timeout,omitempty"`
	HealthyThreshold   int32         `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold int32         `json:"unhealthy_threshold,omitempty"`
}

// IngressSpec defines ingress specifications
type IngressSpec struct {
	ClassName   string            `json:"class_name,omitempty"`
	Rules       []*IngressRule    `json:"rules,omitempty"`
	TLS         []*IngressTLS     `json:"tls,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// IngressRule defines an ingress rule
type IngressRule struct {
	Host  string         `json:"host,omitempty"`
	Paths []*IngressPath `json:"paths,omitempty"`
}

// IngressPath defines an ingress path
type IngressPath struct {
	Path     string          `json:"path,omitempty"`
	PathType string          `json:"path_type,omitempty"`
	Service  *IngressService `json:"service,omitempty"`
}

// IngressService defines an ingress service
type IngressService struct {
	Name string              `json:"name"`
	Port *IngressServicePort `json:"port,omitempty"`
}

// IngressServicePort defines an ingress service port
type IngressServicePort struct {
	Number int32  `json:"number,omitempty"`
	Name   string `json:"name,omitempty"`
}

// IngressTLS defines ingress TLS configuration
type IngressTLS struct {
	Hosts      []string `json:"hosts,omitempty"`
	SecretName string   `json:"secret_name,omitempty"`
}

// DNSSpec defines DNS specifications
type DNSSpec struct {
	Domain  string       `json:"domain,omitempty"`
	Records []*DNSRecord `json:"records,omitempty"`
	TTL     int32        `json:"ttl,omitempty"`
}

// DNSRecord defines a DNS record
type DNSRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int32  `json:"ttl,omitempty"`
}

// SecurityContext defines security context for resources
type SecurityContext struct {
	RunAsUser                *int64          `json:"run_as_user,omitempty"`
	RunAsGroup               *int64          `json:"run_as_group,omitempty"`
	RunAsNonRoot             *bool           `json:"run_as_non_root,omitempty"`
	ReadOnlyRootFilesystem   *bool           `json:"read_only_root_filesystem,omitempty"`
	AllowPrivilegeEscalation *bool           `json:"allow_privilege_escalation,omitempty"`
	Capabilities             *Capabilities   `json:"capabilities,omitempty"`
	SELinuxOptions           *SELinuxOptions `json:"selinux_options,omitempty"`
	WindowsOptions           *WindowsOptions `json:"windows_options,omitempty"`
}

// Capabilities defines security capabilities
type Capabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// SELinuxOptions defines SELinux options
type SELinuxOptions struct {
	User  string `json:"user,omitempty"`
	Role  string `json:"role,omitempty"`
	Type  string `json:"type,omitempty"`
	Level string `json:"level,omitempty"`
}

// WindowsOptions defines Windows-specific options
type WindowsOptions struct {
	GMSACredentialSpecName string `json:"gmsa_credential_spec_name,omitempty"`
	GMSACredentialSpec     string `json:"gmsa_credential_spec,omitempty"`
	RunAsUserName          string `json:"run_as_user_name,omitempty"`
	HostProcess            *bool  `json:"host_process,omitempty"`
}

// ResourceConstraints defines constraints for resource placement
type ResourceConstraints struct {
	NodeSelector              map[string]string           `json:"node_selector,omitempty"`
	Affinity                  *Affinity                   `json:"affinity,omitempty"`
	Tolerations               []*Toleration               `json:"tolerations,omitempty"`
	TopologySpreadConstraints []*TopologySpreadConstraint `json:"topology_spread_constraints,omitempty"`
}

// Affinity defines node affinity rules
type Affinity struct {
	NodeAffinity    *NodeAffinity    `json:"node_affinity,omitempty"`
	PodAffinity     *PodAffinity     `json:"pod_affinity,omitempty"`
	PodAntiAffinity *PodAntiAffinity `json:"pod_anti_affinity,omitempty"`
}

// NodeAffinity defines node affinity rules
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector              `json:"required,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*PreferredSchedulingTerm `json:"preferred,omitempty"`
}

// NodeSelector defines node selection criteria
type NodeSelector struct {
	NodeSelectorTerms []*NodeSelectorTerm `json:"node_selector_terms,omitempty"`
}

// NodeSelectorTerm defines a node selector term
type NodeSelectorTerm struct {
	MatchExpressions []*NodeSelectorRequirement `json:"match_expressions,omitempty"`
	MatchFields      []*NodeSelectorRequirement `json:"match_fields,omitempty"`
}

// NodeSelectorRequirement defines a node selector requirement
type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// PreferredSchedulingTerm defines a preferred scheduling term
type PreferredSchedulingTerm struct {
	Weight     int32             `json:"weight"`
	Preference *NodeSelectorTerm `json:"preference"`
}

// PodAffinity defines pod affinity rules
type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*PodAffinityTerm         `json:"required,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*WeightedPodAffinityTerm `json:"preferred,omitempty"`
}

// PodAntiAffinity defines pod anti-affinity rules
type PodAntiAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*PodAffinityTerm         `json:"required,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*WeightedPodAffinityTerm `json:"preferred,omitempty"`
}

// PodAffinityTerm defines a pod affinity term
type PodAffinityTerm struct {
	LabelSelector *LabelSelector `json:"label_selector,omitempty"`
	Namespaces    []string       `json:"namespaces,omitempty"`
	TopologyKey   string         `json:"topology_key"`
}

// WeightedPodAffinityTerm defines a weighted pod affinity term
type WeightedPodAffinityTerm struct {
	Weight          int32            `json:"weight"`
	PodAffinityTerm *PodAffinityTerm `json:"pod_affinity_term"`
}

// Toleration defines a toleration
type Toleration struct {
	Key               string `json:"key,omitempty"`
	Operator          string `json:"operator,omitempty"`
	Value             string `json:"value,omitempty"`
	Effect            string `json:"effect,omitempty"`
	TolerationSeconds *int64 `json:"toleration_seconds,omitempty"`
}

// TopologySpreadConstraint defines topology spread constraints
type TopologySpreadConstraint struct {
	MaxSkew           int32          `json:"max_skew"`
	TopologyKey       string         `json:"topology_key"`
	WhenUnsatisfiable string         `json:"when_unsatisfiable"`
	LabelSelector     *LabelSelector `json:"label_selector,omitempty"`
}

// ProvisionOptions defines options for resource provisioning
type ProvisionOptions struct {
	WaitForReady       bool          `json:"wait_for_ready,omitempty"`
	Timeout            time.Duration `json:"timeout,omitempty"`
	RetryPolicy        *RetryPolicy  `json:"retry_policy,omitempty"`
	DryRun             bool          `json:"dry_run,omitempty"`
	ValidationOnly     bool          `json:"validation_only,omitempty"`
	PreProvisionHooks  []*Hook       `json:"pre_provision_hooks,omitempty"`
	PostProvisionHooks []*Hook       `json:"post_provision_hooks,omitempty"`
}

// Hook defines a lifecycle hook
type Hook struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // webhook, script, command
	Target        string                 `json:"target"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	FailurePolicy string                 `json:"failure_policy,omitempty"` // fail, ignore
}

// ScaleResourceRequest represents a request to scale a resource
type ScaleResourceRequest struct {
	ScaleType       string                 `json:"scale_type" validate:"required,oneof=horizontal vertical"`
	TargetReplicas  *int32                 `json:"target_replicas,omitempty"`
	TargetResources map[string]string      `json:"target_resources,omitempty"`
	ScalingPolicy   *ScalingPolicy         `json:"scaling_policy,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// MigrateResourceRequest represents a request to migrate a resource
type MigrateResourceRequest struct {
	SourceProvider   string                 `json:"source_provider" validate:"required"`
	TargetProvider   string                 `json:"target_provider" validate:"required"`
	TargetLocation   string                 `json:"target_location,omitempty"`
	MigrationType    string                 `json:"migration_type,omitempty"` // live, offline, blue_green
	MigrationOptions *MigrationOptions      `json:"migration_options,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// MigrationOptions defines options for resource migration
type MigrationOptions struct {
	PreMigrationValidation  bool          `json:"pre_migration_validation,omitempty"`
	PostMigrationValidation bool          `json:"post_migration_validation,omitempty"`
	RollbackOnFailure       bool          `json:"rollback_on_failure,omitempty"`
	DataMigration           bool          `json:"data_migration,omitempty"`
	NetworkMigration        bool          `json:"network_migration,omitempty"`
	Timeout                 time.Duration `json:"timeout,omitempty"`
	PreMigrationHooks       []*Hook       `json:"pre_migration_hooks,omitempty"`
	PostMigrationHooks      []*Hook       `json:"post_migration_hooks,omitempty"`
}

// BackupResourceRequest represents a request to backup a resource
type BackupResourceRequest struct {
	BackupType      string                 `json:"backup_type" validate:"required,oneof=full incremental differential"`
	BackupLocation  string                 `json:"backup_location,omitempty"`
	BackupOptions   *BackupOptions         `json:"backup_options,omitempty"`
	RetentionPeriod time.Duration          `json:"retention_period,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// BackupOptions defines options for resource backup
type BackupOptions struct {
	Compression     bool          `json:"compression,omitempty"`
	Encryption      bool          `json:"encryption,omitempty"`
	EncryptionKey   string        `json:"encryption_key,omitempty"`
	VerifyBackup    bool          `json:"verify_backup,omitempty"`
	Timeout         time.Duration `json:"timeout,omitempty"`
	PreBackupHooks  []*Hook       `json:"pre_backup_hooks,omitempty"`
	PostBackupHooks []*Hook       `json:"post_backup_hooks,omitempty"`
}

// BackupInfo represents information about a resource backup
type BackupInfo struct {
	BackupID       string                 `json:"backup_id"`
	ResourceID     string                 `json:"resource_id"`
	BackupType     string                 `json:"backup_type"`
	Status         string                 `json:"status"`
	BackupLocation string                 `json:"backup_location,omitempty"`
	Size           int64                  `json:"size,omitempty"`
	Checksum       string                 `json:"checksum,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	RetentionUntil time.Time              `json:"retention_until"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Error          string                 `json:"error,omitempty"`
}

// Infrastructure discovery types

// InfrastructureDiscovery represents the result of infrastructure discovery
type InfrastructureDiscovery struct {
	ProviderID    string                    `json:"provider_id"`
	DiscoveryID   string                    `json:"discovery_id"`
	Status        string                    `json:"status"`
	Resources     []*DiscoveredResource     `json:"resources"`
	ResourcePools []*DiscoveredResourcePool `json:"resource_pools"`
	Summary       *DiscoverySummary         `json:"summary"`
	StartedAt     time.Time                 `json:"started_at"`
	CompletedAt   *time.Time                `json:"completed_at,omitempty"`
	Duration      time.Duration             `json:"duration,omitempty"`
	Error         string                    `json:"error,omitempty"`
}

// DiscoveredResource represents a resource discovered during infrastructure discovery
type DiscoveredResource struct {
	ResourceID     string                 `json:"resource_id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	Health         string                 `json:"health"`
	Provider       string                 `json:"provider"`
	Location       string                 `json:"location,omitempty"`
	Specifications map[string]interface{} `json:"specifications,omitempty"`
	Labels         map[string]string      `json:"labels,omitempty"`
	Annotations    map[string]string      `json:"annotations,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	LastSeen       time.Time              `json:"last_seen"`
}

// DiscoveredResourcePool represents a resource pool discovered during infrastructure discovery
type DiscoveredResourcePool struct {
	ResourcePoolID string               `json:"resource_pool_id"`
	Name           string               `json:"name"`
	Provider       string               `json:"provider"`
	Location       string               `json:"location,omitempty"`
	Capacity       *ResourceCapacity    `json:"capacity,omitempty"`
	Utilization    *ResourceUtilization `json:"utilization,omitempty"`
	ResourceCount  int                  `json:"resource_count"`
	Status         string               `json:"status"`
	LastSeen       time.Time            `json:"last_seen"`
}

// ResourceUtilization represents resource utilization metrics
type ResourceUtilization struct {
	CPU         float64   `json:"cpu"`     // Percentage 0-100
	Memory      float64   `json:"memory"`  // Percentage 0-100
	Storage     float64   `json:"storage"` // Percentage 0-100
	Network     float64   `json:"network"` // Percentage 0-100
	LastUpdated time.Time `json:"last_updated"`
}

// DiscoverySummary provides a summary of discovery results
type DiscoverySummary struct {
	TotalResources     int            `json:"total_resources"`
	ResourcesByType    map[string]int `json:"resources_by_type"`
	ResourcesByStatus  map[string]int `json:"resources_by_status"`
	ResourcesByHealth  map[string]int `json:"resources_by_health"`
	TotalResourcePools int            `json:"total_resource_pools"`
	DiscoveryErrors    int            `json:"discovery_errors"`
}

// Inventory update types

// InventoryUpdate represents an inventory update
type InventoryUpdate struct {
	UpdateID     string                 `json:"update_id"`
	UpdateType   string                 `json:"update_type"` // create, update, delete, sync
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Data         map[string]interface{} `json:"data"`
	Timestamp    time.Time              `json:"timestamp"`
	Source       string                 `json:"source"`
}

// Monitoring and health types

// ResourceHealth represents the health status of a resource
type ResourceHealth struct {
	ResourceID    string                 `json:"resource_id"`
	OverallHealth string                 `json:"overall_health"`
	HealthChecks  []*HealthCheck         `json:"health_checks"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	Alarms        []*Alarm               `json:"alarms,omitempty"`
	LastUpdated   time.Time              `json:"last_updated"`
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration,omitempty"`
}

// Alarm represents a resource alarm
type Alarm struct {
	AlarmID        string                 `json:"alarm_id"`
	ResourceID     string                 `json:"resource_id"`
	AlarmType      string                 `json:"alarm_type"`
	Severity       string                 `json:"severity"`
	Status         string                 `json:"status"`
	Message        string                 `json:"message"`
	Details        map[string]interface{} `json:"details,omitempty"`
	RaisedAt       time.Time              `json:"raised_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	ClearedAt      *time.Time             `json:"cleared_at,omitempty"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string                 `json:"acknowledged_by,omitempty"`
}

// MetricsData represents collected metrics data
type MetricsData struct {
	ResourceID  string                 `json:"resource_id"`
	MetricType  string                 `json:"metric_type"`
	Timestamps  []time.Time            `json:"timestamps"`
	Values      []float64              `json:"values"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CollectedAt time.Time              `json:"collected_at"`
}

// Supporting types

// ResourceHistoryEntry represents a resource history entry
type ResourceHistoryEntry struct {
	ResourceID string                 `json:"resource_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Operation  string                 `json:"operation"`
	Status     string                 `json:"status"`
	Changes    map[string]interface{} `json:"changes"`
	User       string                 `json:"user,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
}

// CapacityPrediction represents a capacity prediction
type CapacityPrediction struct {
	ResourcePoolID       string                 `json:"resource_pool_id"`
	PredictionHorizon    time.Duration          `json:"prediction_horizon"`
	PredictedCapacity    *ResourceCapacity      `json:"predicted_capacity"`
	PredictedUtilization *ResourceUtilization   `json:"predicted_utilization"`
	Confidence           float64                `json:"confidence"`
	Algorithm            string                 `json:"algorithm"`
	PredictedAt          time.Time              `json:"predicted_at"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// Constants for resource lifecycle operations
const (
	// Resource statuses
	ResourceStatusPending      = "PENDING"
	ResourceStatusProvisioning = "PROVISIONING"
	ResourceStatusActive       = "ACTIVE"
	ResourceStatusScaling      = "SCALING"
	ResourceStatusMigrating    = "MIGRATING"
	ResourceStatusRestoring    = "RESTORING"
	ResourceStatusTerminating  = "TERMINATING"
	ResourceStatusTerminated   = "TERMINATED"
	ResourceStatusError        = "ERROR"

	// Operation types
	OperationTypeProvision = "PROVISION"
	OperationTypeConfigure = "CONFIGURE"
	OperationTypeScale     = "SCALE"
	OperationTypeMigrate   = "MIGRATE"
	OperationTypeBackup    = "BACKUP"
	OperationTypeRestore   = "RESTORE"
	OperationTypeTerminate = "TERMINATE"

	// Operation statuses
	OperationStatusQueued     = "QUEUED"
	OperationStatusInProgress = "IN_PROGRESS"
	OperationStatusCompleted  = "COMPLETED"
	OperationStatusFailed     = "FAILED"
	OperationStatusCancelled  = "CANCELLED"
	OperationStatusTimeout    = "TIMEOUT"

	// Health statuses
	HealthStatusHealthy   = "HEALTHY"
	HealthStatusDegraded  = "DEGRADED"
	HealthStatusUnhealthy = "UNHEALTHY"
	HealthStatusUnknown   = "UNKNOWN"

	// Alarm severities
	AlarmSeverityCritical = "CRITICAL"
	AlarmSeverityMajor    = "MAJOR"
	AlarmSeverityMinor    = "MINOR"
	AlarmSeverityWarning  = "WARNING"
	AlarmSeverityInfo     = "INFO"

	// Alarm statuses
	AlarmStatusActive       = "ACTIVE"
	AlarmStatusAcknowledged = "ACKNOWLEDGED"
	AlarmStatusCleared      = "CLEARED"

	// Scale types
	ScaleTypeHorizontal = "horizontal"
	ScaleTypeVertical   = "vertical"

	// Migration types
	MigrationTypeLive      = "live"
	MigrationTypeOffline   = "offline"
	MigrationTypeBlueGreen = "blue_green"

	// Backup types
	BackupTypeFull         = "full"
	BackupTypeIncremental  = "incremental"
	BackupTypeDifferential = "differential"
)
