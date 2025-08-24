package telecom

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// TelecomKnowledgeBase contains comprehensive 5G network function specifications
// Based on 3GPP Release 16/17 and O-RAN Alliance specifications
type TelecomKnowledgeBase struct {
	NetworkFunctions map[string]*NetworkFunctionSpec
	Interfaces       map[string]*InterfaceSpec
	QosProfiles      map[string]*QosProfile
	SliceTypes       map[string]*SliceTypeSpec
	PerformanceKPIs  map[string]*KPISpec
	DeploymentTypes  map[string]*DeploymentPattern
	initialized      bool
}

// NetworkFunctionSpec defines comprehensive 5G Core network function specifications
type NetworkFunctionSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Vendor      string `json:"vendor"`

	// 3GPP Specifications
	Specification3GPP []string `json:"specification_3gpp"`

	// Network Interfaces based on 3GPP TS 23.501
	Interfaces        []string `json:"interfaces"`
	ServiceInterfaces []string `json:"service_interfaces"`

	// Dependencies and relationships
	Dependencies []string `json:"dependencies"`
	OptionalDeps []string `json:"optional_dependencies"`

	// Resource requirements (production-grade)
	Resources ResourceRequirements `json:"resources"`

	// Scaling configuration
	Scaling ScalingParameters `json:"scaling"`

	// Performance baselines from real deployments
	Performance PerformanceBaseline `json:"performance"`

	// Security requirements
	Security SecurityRequirements `json:"security"`

	// Deployment patterns
	DeploymentPatterns map[string]DeploymentConfig `json:"deployment_patterns"`

	// Configuration parameters
	Configuration map[string]ConfigParameter `json:"configuration"`

	// Health and monitoring
	HealthChecks      []HealthCheck      `json:"health_checks"`
	MonitoringMetrics []MetricDefinition `json:"monitoring_metrics"`
}

// InterfaceSpec defines 5G network interfaces per 3GPP specifications
type InterfaceSpec struct {
	Name            string          `json:"name"`
	Type            string          `json:"type"` // Reference Point, Service Based
	Protocol        []string        `json:"protocol"`
	Specification   string          `json:"specification"`
	Description     string          `json:"description"`
	Endpoints       []EndpointSpec  `json:"endpoints"`
	SecurityModel   SecurityModel   `json:"security_model"`
	QosRequirements QosRequirements `json:"qos_requirements"`
	Reliability     ReliabilitySpec `json:"reliability"`
}

// ResourceRequirements defines compute, memory, and storage requirements
type ResourceRequirements struct {
	MinCPU      string `json:"min_cpu"`     // e.g., "2" cores
	MinMemory   string `json:"min_memory"`  // e.g., "4Gi"
	MaxCPU      string `json:"max_cpu"`     // e.g., "8" cores
	MaxMemory   string `json:"max_memory"`  // e.g., "16Gi"
	Storage     string `json:"storage"`     // e.g., "100Gi"
	NetworkBW   string `json:"network_bw"`  // e.g., "10Gbps"
	DiskIOPS    int    `json:"disk_iops"`   // e.g., 1000
	Accelerator string `json:"accelerator"` // e.g., "nvidia.com/gpu"
}

// ScalingParameters defines auto-scaling configuration
type ScalingParameters struct {
	MinReplicas      int            `json:"min_replicas"`
	MaxReplicas      int            `json:"max_replicas"`
	TargetCPU        int            `json:"target_cpu"`         // percentage
	TargetMemory     int            `json:"target_memory"`      // percentage
	ScaleUpThreshold int            `json:"scale_up_threshold"` // percentage
	ScaleDownDelay   int            `json:"scale_down_delay"`   // seconds
	CustomMetrics    []CustomMetric `json:"custom_metrics"`
}

// PerformanceBaseline defines expected performance characteristics
type PerformanceBaseline struct {
	MaxThroughputRPS      int     `json:"max_throughput_rps"` // Requests per second
	AvgLatencyMs          float64 `json:"avg_latency_ms"`     // Average latency in ms
	P95LatencyMs          float64 `json:"p95_latency_ms"`     // P95 latency in ms
	P99LatencyMs          float64 `json:"p99_latency_ms"`     // P99 latency in ms
	MaxConcurrentSessions int     `json:"max_concurrent_sessions"`
	ProcessingTimeMs      float64 `json:"processing_time_ms"`
	MemoryUsageMB         int     `json:"memory_usage_mb"`
	CPUUsagePercent       int     `json:"cpu_usage_percent"`
}

// SecurityRequirements defines security specifications
type SecurityRequirements struct {
	TLSVersion      []string        `json:"tls_version"`
	Authentication  []string        `json:"authentication"` // OAuth2, mTLS, etc.
	Encryption      EncryptionSpec  `json:"encryption"`
	RBAC            bool            `json:"rbac"`
	NetworkPolicies []NetworkPolicy `json:"network_policies"`
	SecurityContext SecurityContext `json:"security_context"`
}

// DeploymentConfig defines deployment patterns
type DeploymentConfig struct {
	Name            string           `json:"name"`
	Replicas        int              `json:"replicas"`
	AntiAffinity    bool             `json:"anti_affinity"`
	ResourceProfile string           `json:"resource_profile"` // small, medium, large
	StorageClass    string           `json:"storage_class"`
	NetworkPolicy   string           `json:"network_policy"`
	ServiceMesh     bool             `json:"service_mesh"`
	LoadBalancer    LoadBalancerSpec `json:"load_balancer"`
	Monitoring      MonitoringConfig `json:"monitoring"`
	BackupPolicy    BackupPolicy     `json:"backup_policy"`
}

// Supporting types
type ConfigParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Validation  string      `json:"validation"`
}

type HealthCheck struct {
	Type         string `json:"type"` // http, tcp, grpc
	Path         string `json:"path"`
	Port         int    `json:"port"`
	Interval     int    `json:"interval"` // seconds
	Timeout      int    `json:"timeout"`  // seconds
	Retries      int    `json:"retries"`
	InitialDelay int    `json:"initial_delay"` // seconds
}

type MetricDefinition struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // counter, gauge, histogram
	Description string      `json:"description"`
	Labels      []string    `json:"labels"`
	Alerts      []AlertRule `json:"alerts"`
}

type AlertRule struct {
	Name      string  `json:"name"`
	Condition string  `json:"condition"`
	Threshold float64 `json:"threshold"`
	Duration  string  `json:"duration"`
	Severity  string  `json:"severity"`
}

type CustomMetric struct {
	Name   string `json:"name"`
	Target int    `json:"target"`
	Type   string `json:"type"`
}

type EncryptionSpec struct {
	AtRest        string   `json:"at_rest"`
	InTransit     []string `json:"in_transit"`
	KeyManagement string   `json:"key_management"`
}

type NetworkPolicy struct {
	Name      string   `json:"name"`
	Ingress   []string `json:"ingress"`
	Egress    []string `json:"egress"`
	Protocols []string `json:"protocols"`
}

type SecurityContext struct {
	RunAsNonRoot   bool              `json:"run_as_non_root"`
	RunAsUser      int64             `json:"run_as_user"`
	FSGroup        int64             `json:"fs_group"`
	Capabilities   []string          `json:"capabilities"`
	SeLinuxOptions map[string]string `json:"selinux_options"`
}

type LoadBalancerSpec struct {
	Type            string        `json:"type"`      // ClusterIP, NodePort, LoadBalancer
	Algorithm       string        `json:"algorithm"` // round-robin, least-connections
	SessionAffinity string        `json:"session_affinity"`
	HealthCheck     HealthCheckLB `json:"health_check"`
}

type HealthCheckLB struct {
	Enabled  bool   `json:"enabled"`
	Path     string `json:"path"`
	Interval int    `json:"interval"`
	Timeout  int    `json:"timeout"`
}

type MonitoringConfig struct {
	Enabled    bool   `json:"enabled"`
	Prometheus bool   `json:"prometheus"`
	Grafana    bool   `json:"grafana"`
	Jaeger     bool   `json:"jaeger"`
	LogLevel   string `json:"log_level"`
}

type BackupPolicy struct {
	Enabled   bool   `json:"enabled"`
	Schedule  string `json:"schedule"`  // cron format
	Retention int    `json:"retention"` // days
}

type EndpointSpec struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Path     string `json:"path"`
}

type SecurityModel struct {
	Authentication string   `json:"authentication"`
	Authorization  string   `json:"authorization"`
	Encryption     []string `json:"encryption"`
}

type QosRequirements struct {
	Bandwidth  string  `json:"bandwidth"`
	Latency    float64 `json:"latency"`     // ms
	Jitter     float64 `json:"jitter"`      // ms
	PacketLoss float64 `json:"packet_loss"` // percentage
}

type ReliabilitySpec struct {
	Availability float64 `json:"availability"` // percentage (99.99)
	MTBF         int     `json:"mtbf"`         // hours
	MTTR         int     `json:"mttr"`         // minutes
}

type QosProfile struct {
	QCI          int     `json:"qci"`            // QoS Class Identifier
	QFI          int     `json:"qfi"`            // QoS Flow Identifier
	Resource     string  `json:"resource"`       // GBR, Non-GBR
	Priority     int     `json:"priority"`       // 1-9 (1 highest)
	DelayBudget  int     `json:"delay_budget"`   // ms
	ErrorRate    float64 `json:"error_rate"`     // 10^-x
	MaxBitrateUL string  `json:"max_bitrate_ul"` // Mbps
	MaxBitrateDL string  `json:"max_bitrate_dl"` // Mbps
	GuaranteedBR string  `json:"guaranteed_br"`  // Mbps
}

type SliceTypeSpec struct {
	SST              int               `json:"sst"` // Slice/Service Type
	Description      string            `json:"description"`
	UseCase          string            `json:"use_case"`
	Requirements     SliceRequirements `json:"requirements"`
	ResourceProfile  ResourceProfile   `json:"resource_profile"`
	QosProfile       string            `json:"qos_profile"`
	NetworkFunctions []string          `json:"network_functions"`
}

type SliceRequirements struct {
	Throughput  ThroughputReq  `json:"throughput"`
	Latency     LatencyReq     `json:"latency"`
	Reliability ReliabilityReq `json:"reliability"`
	Density     DensityReq     `json:"density"`
}

type ThroughputReq struct {
	Min     string `json:"min"`     // Mbps
	Max     string `json:"max"`     // Mbps
	Typical string `json:"typical"` // Mbps
}

type LatencyReq struct {
	UserPlane    float64 `json:"user_plane"`    // ms
	ControlPlane float64 `json:"control_plane"` // ms
	MaxJitter    float64 `json:"max_jitter"`    // ms
}

type ReliabilityReq struct {
	Availability float64 `json:"availability"` // percentage
	PacketLoss   float64 `json:"packet_loss"`  // percentage
	ErrorRate    float64 `json:"error_rate"`   // 10^-x
}

type DensityReq struct {
	ConnectedDevices int `json:"connected_devices"` // per km²
	ActiveDevices    int `json:"active_devices"`    // per km²
	SessionDensity   int `json:"session_density"`   // per km²
}

type ResourceProfile struct {
	Compute ComputeProfile `json:"compute"`
	Network NetworkProfile `json:"network"`
	Storage StorageProfile `json:"storage"`
}

type ComputeProfile struct {
	CPUIntensive bool   `json:"cpu_intensive"`
	GPURequired  bool   `json:"gpu_required"`
	MemoryType   string `json:"memory_type"`  // standard, high-memory
	Architecture string `json:"architecture"` // x86, ARM
}

type NetworkProfile struct {
	BandwidthIntensive bool `json:"bandwidth_intensive"`
	LatencySensitive   bool `json:"latency_sensitive"`
	JitterSensitive    bool `json:"jitter_sensitive"`
	EdgePlacement      bool `json:"edge_placement"`
	Multicast          bool `json:"multicast"`
}

type StorageProfile struct {
	PersistentStorage bool   `json:"persistent_storage"`
	HighIOPS          bool   `json:"high_iops"`
	StorageType       string `json:"storage_type"` // ssd, nvme, hdd
	BackupRequired    bool   `json:"backup_required"`
}

type KPISpec struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"` // counter, gauge, rate
	Unit        string     `json:"unit"`
	Description string     `json:"description"`
	Category    string     `json:"category"` // performance, reliability, quality
	Thresholds  Thresholds `json:"thresholds"`
	SLA         SLASpec    `json:"sla"`
}

type Thresholds struct {
	Critical float64 `json:"critical"`
	Warning  float64 `json:"warning"`
	Target   float64 `json:"target"`
}

type SLASpec struct {
	Availability float64 `json:"availability"`
	Performance  float64 `json:"performance"`
	Capacity     float64 `json:"capacity"`
}

type DeploymentPattern struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	UseCase      []string               `json:"use_case"`
	Architecture DeploymentArchitecture `json:"architecture"`
	Scaling      ScalingPattern         `json:"scaling"`
	Resilience   ResiliencePattern      `json:"resilience"`
	Performance  PerformancePattern     `json:"performance"`
}

type DeploymentArchitecture struct {
	Type         string `json:"type"`          // single-zone, multi-zone, multi-region
	Redundancy   string `json:"redundancy"`    // active-active, active-passive
	DataPlane    string `json:"data_plane"`    // centralized, distributed, hybrid
	ControlPlane string `json:"control_plane"` // centralized, distributed
	ServiceMesh  bool   `json:"service_mesh"`
}

type ScalingPattern struct {
	Horizontal bool `json:"horizontal"`
	Vertical   bool `json:"vertical"`
	Predictive bool `json:"predictive"`
	Reactive   bool `json:"reactive"`
}

type ResiliencePattern struct {
	CircuitBreaker bool `json:"circuit_breaker"`
	Retry          bool `json:"retry"`
	Timeout        bool `json:"timeout"`
	Bulkhead       bool `json:"bulkhead"`
	HealthCheck    bool `json:"health_check"`
}

type PerformancePattern struct {
	Caching           bool `json:"caching"`
	LoadBalancing     bool `json:"load_balancing"`
	ConnectionPooling bool `json:"connection_pooling"`
	Compression       bool `json:"compression"`
}

// NewTelecomKnowledgeBase creates and initializes the knowledge base
func NewTelecomKnowledgeBase() *TelecomKnowledgeBase {
	kb := &TelecomKnowledgeBase{
		NetworkFunctions: make(map[string]*NetworkFunctionSpec),
		Interfaces:       make(map[string]*InterfaceSpec),
		QosProfiles:      make(map[string]*QosProfile),
		SliceTypes:       make(map[string]*SliceTypeSpec),
		PerformanceKPIs:  make(map[string]*KPISpec),
		DeploymentTypes:  make(map[string]*DeploymentPattern),
	}

	kb.initializeNetworkFunctions()
	kb.initializeInterfaces()
	kb.initializeQosProfiles()
	kb.initializeSliceTypes()
	kb.initializeKPIs()
	kb.initializeDeploymentPatterns()

	kb.initialized = true
	return kb
}

// initializeNetworkFunctions initializes 5G Core network function specifications
func (kb *TelecomKnowledgeBase) initializeNetworkFunctions() {
	// AMF - Access and Mobility Management Function (3GPP TS 23.501)
	kb.NetworkFunctions["amf"] = &NetworkFunctionSpec{
		Name:              "AMF",
		Type:              "5gc-control-plane",
		Description:       "Access and Mobility Management Function - Handles UE registration, authentication, mobility management, and access authorization",
		Version:           "R17",
		Vendor:            "multi-vendor",
		Specification3GPP: []string{"TS 23.501", "TS 23.502", "TS 29.518", "TS 29.518"},
		Interfaces:        []string{"N1", "N2", "N8", "N11", "N12", "N14", "N15", "N20", "N22"},
		ServiceInterfaces: []string{"Namf_Communication", "Namf_EventExposure", "Namf_MT", "Namf_Location"},
		Dependencies:      []string{"AUSF", "UDM", "PCF", "SMF"},
		OptionalDeps:      []string{"NSSF", "NEF", "NRF"},
		Resources: ResourceRequirements{
			MinCPU:      "2",
			MinMemory:   "4Gi",
			MaxCPU:      "8",
			MaxMemory:   "16Gi",
			Storage:     "100Gi",
			NetworkBW:   "10Gbps",
			DiskIOPS:    1000,
			Accelerator: "",
		},
		Scaling: ScalingParameters{
			MinReplicas:      3,
			MaxReplicas:      20,
			TargetCPU:        70,
			TargetMemory:     80,
			ScaleUpThreshold: 80,
			ScaleDownDelay:   300,
			CustomMetrics: []CustomMetric{
				{Name: "active_ue_sessions", Target: 10000, Type: "average"},
				{Name: "registration_rate", Target: 1000, Type: "rate"},
			},
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS:      10000,
			AvgLatencyMs:          50,
			P95LatencyMs:          100,
			P99LatencyMs:          200,
			MaxConcurrentSessions: 100000,
			ProcessingTimeMs:      25,
			MemoryUsageMB:         8192,
			CPUUsagePercent:       60,
		},
		Security: SecurityRequirements{
			TLSVersion:     []string{"TLS1.3", "TLS1.2"},
			Authentication: []string{"OAuth2", "mTLS", "JWT"},
			Encryption: EncryptionSpec{
				AtRest:        "AES-256",
				InTransit:     []string{"TLS1.3", "IPSec"},
				KeyManagement: "HSM",
			},
			RBAC: true,
			NetworkPolicies: []NetworkPolicy{
				{Name: "amf-ingress", Ingress: []string{"N1", "N2", "SBI"}, Egress: []string{"SBI"}, Protocols: []string{"HTTPS", "HTTP2"}},
			},
			SecurityContext: SecurityContext{
				RunAsNonRoot: true,
				RunAsUser:    1000,
				FSGroup:      1000,
				Capabilities: []string{"NET_BIND_SERVICE"},
			},
		},
		DeploymentPatterns: map[string]DeploymentConfig{
			"production": {
				Name:            "production",
				Replicas:        3,
				AntiAffinity:    true,
				ResourceProfile: "large",
				StorageClass:    "ssd",
				NetworkPolicy:   "strict",
				ServiceMesh:     true,
				LoadBalancer: LoadBalancerSpec{
					Type:            "LoadBalancer",
					Algorithm:       "least-connections",
					SessionAffinity: "ClientIP",
					HealthCheck:     HealthCheckLB{Enabled: true, Path: "/health", Interval: 30, Timeout: 10},
				},
				Monitoring: MonitoringConfig{
					Enabled:    true,
					Prometheus: true,
					Grafana:    true,
					Jaeger:     true,
					LogLevel:   "info",
				},
				BackupPolicy: BackupPolicy{
					Enabled:   true,
					Schedule:  "0 2 * * *",
					Retention: 30,
				},
			},
			"development": {
				Name:            "development",
				Replicas:        1,
				AntiAffinity:    false,
				ResourceProfile: "small",
				StorageClass:    "standard",
				NetworkPolicy:   "permissive",
				ServiceMesh:     false,
				LoadBalancer: LoadBalancerSpec{
					Type:      "ClusterIP",
					Algorithm: "round-robin",
				},
				Monitoring: MonitoringConfig{
					Enabled:    true,
					Prometheus: true,
					LogLevel:   "debug",
				},
			},
		},
		Configuration: map[string]ConfigParameter{
			"max_ue_contexts": {
				Name:        "max_ue_contexts",
				Type:        "integer",
				Default:     100000,
				Required:    true,
				Description: "Maximum number of concurrent UE contexts",
				Validation:  "range:1000-1000000",
			},
			"registration_timer": {
				Name:        "registration_timer",
				Type:        "duration",
				Default:     "3600s",
				Required:    true,
				Description: "UE registration timer duration",
				Validation:  "range:300s-7200s",
			},
		},
		HealthChecks: []HealthCheck{
			{Type: "http", Path: "/health", Port: 8080, Interval: 30, Timeout: 10, Retries: 3, InitialDelay: 60},
			{Type: "tcp", Port: 38412, Interval: 30, Timeout: 5, Retries: 3, InitialDelay: 30},
		},
		MonitoringMetrics: []MetricDefinition{
			{
				Name:        "amf_registered_ues",
				Type:        "gauge",
				Description: "Number of registered UEs",
				Labels:      []string{"plmn_id", "amf_region_id"},
				Alerts: []AlertRule{
					{Name: "high_ue_count", Condition: ">", Threshold: 80000, Duration: "5m", Severity: "warning"},
				},
			},
		},
	}

	// SMF - Session Management Function (3GPP TS 23.501)
	kb.NetworkFunctions["smf"] = &NetworkFunctionSpec{
		Name:              "SMF",
		Type:              "5gc-control-plane",
		Description:       "Session Management Function - Manages PDU sessions, IP address allocation, and traffic routing policies",
		Version:           "R17",
		Vendor:            "multi-vendor",
		Specification3GPP: []string{"TS 23.501", "TS 23.502", "TS 29.502", "TS 29.508"},
		Interfaces:        []string{"N4", "N7", "N10", "N11", "N16"},
		ServiceInterfaces: []string{"Nsmf_PDUSession", "Nsmf_EventExposure"},
		Dependencies:      []string{"UPF", "PCF", "UDM", "AMF"},
		OptionalDeps:      []string{"CHF", "NEF"},
		Resources: ResourceRequirements{
			MinCPU:    "2",
			MinMemory: "4Gi",
			MaxCPU:    "6",
			MaxMemory: "12Gi",
			Storage:   "50Gi",
			NetworkBW: "5Gbps",
			DiskIOPS:  800,
		},
		Scaling: ScalingParameters{
			MinReplicas:      2,
			MaxReplicas:      15,
			TargetCPU:        70,
			TargetMemory:     75,
			ScaleUpThreshold: 75,
			ScaleDownDelay:   300,
			CustomMetrics: []CustomMetric{
				{Name: "active_pdu_sessions", Target: 50000, Type: "average"},
				{Name: "session_establishment_rate", Target: 500, Type: "rate"},
			},
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS:      5000,
			AvgLatencyMs:          100,
			P95LatencyMs:          200,
			P99LatencyMs:          400,
			MaxConcurrentSessions: 200000,
			ProcessingTimeMs:      75,
			MemoryUsageMB:         6144,
			CPUUsagePercent:       65,
		},
		Security: SecurityRequirements{
			TLSVersion:     []string{"TLS1.3", "TLS1.2"},
			Authentication: []string{"OAuth2", "mTLS"},
			Encryption: EncryptionSpec{
				AtRest:        "AES-256",
				InTransit:     []string{"TLS1.3", "PFCP"},
				KeyManagement: "HSM",
			},
			RBAC: true,
		},
	}

	// UPF - User Plane Function (3GPP TS 23.501)
	kb.NetworkFunctions["upf"] = &NetworkFunctionSpec{
		Name:              "UPF",
		Type:              "5gc-user-plane",
		Description:       "User Plane Function - Handles packet forwarding, QoS enforcement, and traffic accounting",
		Version:           "R17",
		Vendor:            "multi-vendor",
		Specification3GPP: []string{"TS 23.501", "TS 23.502", "TS 29.244"},
		Interfaces:        []string{"N3", "N4", "N6", "N9"},
		Dependencies:      []string{"SMF"},
		Resources: ResourceRequirements{
			MinCPU:      "4",
			MinMemory:   "8Gi",
			MaxCPU:      "16",
			MaxMemory:   "32Gi",
			Storage:     "200Gi",
			NetworkBW:   "100Gbps",
			DiskIOPS:    2000,
			Accelerator: "intel.com/qat",
		},
		Scaling: ScalingParameters{
			MinReplicas:      2,
			MaxReplicas:      10,
			TargetCPU:        60,
			TargetMemory:     70,
			ScaleUpThreshold: 70,
			ScaleDownDelay:   600,
			CustomMetrics: []CustomMetric{
				{Name: "throughput_mbps", Target: 10000, Type: "average"},
				{Name: "packet_processing_rate", Target: 1000000, Type: "rate"},
			},
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS:      100000,
			AvgLatencyMs:          10,
			P95LatencyMs:          20,
			P99LatencyMs:          50,
			MaxConcurrentSessions: 1000000,
			ProcessingTimeMs:      1,
			MemoryUsageMB:         16384,
			CPUUsagePercent:       55,
		},
	}

	// PCF - Policy Control Function (3GPP TS 23.501)
	kb.NetworkFunctions["pcf"] = &NetworkFunctionSpec{
		Name:              "PCF",
		Type:              "5gc-control-plane",
		Description:       "Policy Control Function - Provides policy rules for access and mobility management",
		Version:           "R17",
		Specification3GPP: []string{"TS 23.501", "TS 23.503", "TS 29.507"},
		Interfaces:        []string{"N5", "N7", "N15", "N16"},
		ServiceInterfaces: []string{"Npcf_AM_PolicyControl", "Npcf_SM_PolicyControl", "Npcf_PolicyAuthorization"},
		Dependencies:      []string{"UDR"},
		OptionalDeps:      []string{"BSF", "NEF"},
		Resources: ResourceRequirements{
			MinCPU:    "1",
			MinMemory: "2Gi",
			MaxCPU:    "4",
			MaxMemory: "8Gi",
			Storage:   "50Gi",
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS: 2000,
			AvgLatencyMs:     75,
			P95LatencyMs:     150,
			ProcessingTimeMs: 50,
		},
	}

	// UDM - Unified Data Management (3GPP TS 23.501)
	kb.NetworkFunctions["udm"] = &NetworkFunctionSpec{
		Name:              "UDM",
		Type:              "5gc-control-plane",
		Description:       "Unified Data Management - Manages user subscription data and authentication credentials",
		Version:           "R17",
		Specification3GPP: []string{"TS 23.501", "TS 29.503"},
		Interfaces:        []string{"N8", "N10", "N13"},
		ServiceInterfaces: []string{"Nudm_SubscriberDataManagement", "Nudm_UEAuthentication", "Nudm_UEContextManagement"},
		Dependencies:      []string{"UDR"},
		Resources: ResourceRequirements{
			MinCPU:    "2",
			MinMemory: "4Gi",
			MaxCPU:    "6",
			MaxMemory: "12Gi",
			Storage:   "100Gi",
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS: 8000,
			AvgLatencyMs:     60,
			P95LatencyMs:     120,
			ProcessingTimeMs: 30,
		},
	}

	// AUSF - Authentication Server Function (3GPP TS 23.501)
	kb.NetworkFunctions["ausf"] = &NetworkFunctionSpec{
		Name:              "AUSF",
		Type:              "5gc-control-plane",
		Description:       "Authentication Server Function - Performs UE authentication and key generation",
		Version:           "R17",
		Specification3GPP: []string{"TS 23.501", "TS 33.501", "TS 29.509"},
		Interfaces:        []string{"N12", "N13"},
		ServiceInterfaces: []string{"Nausf_UEAuthentication", "Nausf_SoRProtection"},
		Dependencies:      []string{"UDM"},
		Resources: ResourceRequirements{
			MinCPU:    "1",
			MinMemory: "2Gi",
			MaxCPU:    "4",
			MaxMemory: "8Gi",
			Storage:   "20Gi",
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS: 5000,
			AvgLatencyMs:     80,
			P95LatencyMs:     160,
			ProcessingTimeMs: 40,
		},
	}

	// NRF - Network Repository Function (3GPP TS 23.501)
	kb.NetworkFunctions["nrf"] = &NetworkFunctionSpec{
		Name:              "NRF",
		Type:              "5gc-control-plane",
		Description:       "Network Repository Function - Service discovery and registration for 5G Core network functions",
		Version:           "R17",
		Specification3GPP: []string{"TS 23.501", "TS 29.510"},
		ServiceInterfaces: []string{"Nnrf_NFManagement", "Nnrf_NFDiscovery", "Nnrf_AccessToken", "Nnrf_Bootstrapping"},
		Resources: ResourceRequirements{
			MinCPU:    "1",
			MinMemory: "2Gi",
			MaxCPU:    "3",
			MaxMemory: "6Gi",
			Storage:   "50Gi",
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS: 15000,
			AvgLatencyMs:     30,
			P95LatencyMs:     60,
			ProcessingTimeMs: 10,
		},
	}

	// NSSF - Network Slice Selection Function (3GPP TS 23.501)
	kb.NetworkFunctions["nssf"] = &NetworkFunctionSpec{
		Name:              "NSSF",
		Type:              "5gc-control-plane",
		Description:       "Network Slice Selection Function - Selects appropriate network slice instances for UEs",
		Version:           "R17",
		Specification3GPP: []string{"TS 23.501", "TS 23.502", "TS 29.531"},
		Interfaces:        []string{"N22"},
		ServiceInterfaces: []string{"Nnssf_NSSelection", "Nnssf_NSSAI_Availability"},
		Dependencies:      []string{"NRF"},
		Resources: ResourceRequirements{
			MinCPU:    "1",
			MinMemory: "1Gi",
			MaxCPU:    "2",
			MaxMemory: "4Gi",
			Storage:   "20Gi",
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS: 3000,
			AvgLatencyMs:     40,
			P95LatencyMs:     80,
			ProcessingTimeMs: 20,
		},
	}

	// gNodeB - 5G Base Station (3GPP TS 38.401)
	kb.NetworkFunctions["gnb"] = &NetworkFunctionSpec{
		Name:              "gNodeB",
		Type:              "ran",
		Description:       "5G New Radio Base Station - Provides radio access for 5G UEs",
		Version:           "R17",
		Specification3GPP: []string{"TS 38.401", "TS 38.423", "TS 38.473"},
		Interfaces:        []string{"N2", "N3", "Xn", "F1", "E1"},
		Dependencies:      []string{"AMF", "UPF"},
		Resources: ResourceRequirements{
			MinCPU:      "4",
			MinMemory:   "8Gi",
			MaxCPU:      "12",
			MaxMemory:   "24Gi",
			Storage:     "100Gi",
			NetworkBW:   "25Gbps",
			Accelerator: "xilinx.com/fpga-xilinx_u50_gen3x16_xdma_base_1",
		},
		Performance: PerformanceBaseline{
			MaxThroughputRPS:      50000,
			AvgLatencyMs:          5,
			P95LatencyMs:          15,
			MaxConcurrentSessions: 500000,
		},
	}
}

// initializeInterfaces initializes 5G network interface specifications
func (kb *TelecomKnowledgeBase) initializeInterfaces() {
	// N1 Interface (UE ↔ AMF)
	kb.Interfaces["n1"] = &InterfaceSpec{
		Name:          "N1",
		Type:          "reference-point",
		Protocol:      []string{"NAS"},
		Specification: "3GPP TS 24.501",
		Description:   "Interface between UE and AMF for NAS signaling",
		Endpoints: []EndpointSpec{
			{Name: "ue", Port: 0, Protocol: "NAS"},
			{Name: "amf", Port: 38412, Protocol: "NAS"},
		},
		SecurityModel: SecurityModel{
			Authentication: "5G-AKA",
			Authorization:  "AUSF",
			Encryption:     []string{"5G-EA0", "5G-EA1", "5G-EA2", "5G-EA3"},
		},
		QosRequirements: QosRequirements{
			Bandwidth:  "1Mbps",
			Latency:    50.0,
			Jitter:     10.0,
			PacketLoss: 0.01,
		},
	}

	// N2 Interface (gNB ↔ AMF)
	kb.Interfaces["n2"] = &InterfaceSpec{
		Name:          "N2",
		Type:          "reference-point",
		Protocol:      []string{"NGAP"},
		Specification: "3GPP TS 38.413",
		Description:   "Interface between gNB and AMF for control plane signaling",
		SecurityModel: SecurityModel{
			Authentication: "IPSec",
			Encryption:     []string{"AES-256"},
		},
		QosRequirements: QosRequirements{
			Bandwidth:  "100Mbps",
			Latency:    20.0,
			PacketLoss: 0.001,
		},
	}

	// N3 Interface (gNB ↔ UPF)
	kb.Interfaces["n3"] = &InterfaceSpec{
		Name:          "N3",
		Type:          "reference-point",
		Protocol:      []string{"GTP-U"},
		Specification: "3GPP TS 29.281",
		Description:   "Interface between gNB and UPF for user plane data",
		QosRequirements: QosRequirements{
			Bandwidth:  "100Gbps",
			Latency:    1.0,
			PacketLoss: 0.00001,
		},
	}

	// N4 Interface (SMF ↔ UPF)
	kb.Interfaces["n4"] = &InterfaceSpec{
		Name:          "N4",
		Type:          "reference-point",
		Protocol:      []string{"PFCP"},
		Specification: "3GPP TS 29.244",
		Description:   "Interface between SMF and UPF for session management",
		QosRequirements: QosRequirements{
			Bandwidth:  "10Gbps",
			Latency:    10.0,
			PacketLoss: 0.0001,
		},
	}

	// Service Based Interfaces
	kb.Interfaces["namf"] = &InterfaceSpec{
		Name:          "Namf",
		Type:          "service-based",
		Protocol:      []string{"HTTP/2", "JSON"},
		Specification: "3GPP TS 29.518",
		Description:   "AMF services for communication, event exposure, location, and MT services",
		SecurityModel: SecurityModel{
			Authentication: "OAuth2",
			Authorization:  "RBAC",
			Encryption:     []string{"TLS1.3"},
		},
	}
}

// initializeQosProfiles initializes QoS profiles based on 3GPP TS 23.501
func (kb *TelecomKnowledgeBase) initializeQosProfiles() {
	// 5QI 1 - Conversational Voice (GBR)
	kb.QosProfiles["5qi_1"] = &QosProfile{
		QCI:          1,
		QFI:          1,
		Resource:     "GBR",
		Priority:     2,
		DelayBudget:  100,
		ErrorRate:    0.01,
		MaxBitrateUL: "150",
		MaxBitrateDL: "150",
		GuaranteedBR: "64",
	}

	// 5QI 2 - Conversational Video (GBR)
	kb.QosProfiles["5qi_2"] = &QosProfile{
		QCI:          2,
		QFI:          2,
		Resource:     "GBR",
		Priority:     4,
		DelayBudget:  150,
		ErrorRate:    0.001,
		MaxBitrateUL: "1000",
		MaxBitrateDL: "1000",
		GuaranteedBR: "500",
	}

	// 5QI 5 - IMS Signaling (Non-GBR)
	kb.QosProfiles["5qi_5"] = &QosProfile{
		QCI:          5,
		QFI:          5,
		Resource:     "Non-GBR",
		Priority:     1,
		DelayBudget:  100,
		ErrorRate:    0.000001,
		MaxBitrateUL: "256",
		MaxBitrateDL: "256",
	}

	// 5QI 9 - Default Bearer (Non-GBR)
	kb.QosProfiles["5qi_9"] = &QosProfile{
		QCI:          9,
		QFI:          9,
		Resource:     "Non-GBR",
		Priority:     9,
		DelayBudget:  300,
		ErrorRate:    0.000001,
		MaxBitrateUL: "unlimited",
		MaxBitrateDL: "unlimited",
	}

	// 5QI 82 - Discrete Automation (URLLC)
	kb.QosProfiles["5qi_82"] = &QosProfile{
		QCI:          82,
		QFI:          82,
		Resource:     "GBR",
		Priority:     1,
		DelayBudget:  10,
		ErrorRate:    0.000001,
		MaxBitrateUL: "100",
		MaxBitrateDL: "100",
		GuaranteedBR: "50",
	}
}

// initializeSliceTypes initializes network slice type specifications
func (kb *TelecomKnowledgeBase) initializeSliceTypes() {
	// eMBB - Enhanced Mobile Broadband (SST=1)
	kb.SliceTypes["embb"] = &SliceTypeSpec{
		SST:         1,
		Description: "Enhanced Mobile Broadband",
		UseCase:     "High-speed data services, streaming, web browsing",
		Requirements: SliceRequirements{
			Throughput: ThroughputReq{
				Min:     "100",
				Max:     "20000",
				Typical: "1000",
			},
			Latency: LatencyReq{
				UserPlane:    10.0,
				ControlPlane: 100.0,
				MaxJitter:    5.0,
			},
			Reliability: ReliabilityReq{
				Availability: 99.9,
				PacketLoss:   0.1,
				ErrorRate:    0.00001,
			},
			Density: DensityReq{
				ConnectedDevices: 10000,
				ActiveDevices:    5000,
				SessionDensity:   2000,
			},
		},
		ResourceProfile: ResourceProfile{
			Compute: ComputeProfile{
				CPUIntensive: false,
				GPURequired:  false,
				MemoryType:   "standard",
				Architecture: "x86",
			},
			Network: NetworkProfile{
				BandwidthIntensive: true,
				LatencySensitive:   false,
				JitterSensitive:    false,
				EdgePlacement:      false,
			},
			Storage: StorageProfile{
				PersistentStorage: false,
				HighIOPS:          false,
				StorageType:       "ssd",
			},
		},
		QosProfile:       "5qi_9",
		NetworkFunctions: []string{"amf", "smf", "upf", "pcf", "udm"},
	}

	// URLLC - Ultra-Reliable Low Latency Communications (SST=2)
	kb.SliceTypes["urllc"] = &SliceTypeSpec{
		SST:         2,
		Description: "Ultra-Reliable Low Latency Communications",
		UseCase:     "Industrial automation, autonomous vehicles, remote surgery",
		Requirements: SliceRequirements{
			Throughput: ThroughputReq{
				Min:     "1",
				Max:     "50",
				Typical: "10",
			},
			Latency: LatencyReq{
				UserPlane:    1.0,
				ControlPlane: 10.0,
				MaxJitter:    0.1,
			},
			Reliability: ReliabilityReq{
				Availability: 99.999,
				PacketLoss:   0.00001,
				ErrorRate:    0.000000001,
			},
		},
		ResourceProfile: ResourceProfile{
			Compute: ComputeProfile{
				CPUIntensive: true,
				MemoryType:   "high-memory",
			},
			Network: NetworkProfile{
				BandwidthIntensive: false,
				LatencySensitive:   true,
				JitterSensitive:    true,
				EdgePlacement:      true,
			},
			Storage: StorageProfile{
				HighIOPS:    true,
				StorageType: "nvme",
			},
		},
		QosProfile:       "5qi_82",
		NetworkFunctions: []string{"amf", "smf", "upf", "pcf"},
	}

	// mMTC - Massive Machine Type Communications (SST=3)
	kb.SliceTypes["mmtc"] = &SliceTypeSpec{
		SST:         3,
		Description: "Massive Machine Type Communications",
		UseCase:     "IoT sensors, smart city, environmental monitoring",
		Requirements: SliceRequirements{
			Throughput: ThroughputReq{
				Min:     "0.1",
				Max:     "10",
				Typical: "1",
			},
			Latency: LatencyReq{
				UserPlane:    1000.0,
				ControlPlane: 10000.0,
			},
			Reliability: ReliabilityReq{
				Availability: 99.0,
				PacketLoss:   1.0,
			},
			Density: DensityReq{
				ConnectedDevices: 1000000,
				ActiveDevices:    100000,
			},
		},
		ResourceProfile: ResourceProfile{
			Compute: ComputeProfile{
				CPUIntensive: false,
				Architecture: "ARM",
			},
			Network: NetworkProfile{
				BandwidthIntensive: false,
				LatencySensitive:   false,
			},
		},
		NetworkFunctions: []string{"amf", "smf", "upf"},
	}
}

// initializeKPIs initializes performance KPI specifications
func (kb *TelecomKnowledgeBase) initializeKPIs() {
	kb.PerformanceKPIs["registration_success_rate"] = &KPISpec{
		Name:        "Registration Success Rate",
		Type:        "gauge",
		Unit:        "percentage",
		Description: "Percentage of successful UE registrations",
		Category:    "reliability",
		Thresholds: Thresholds{
			Critical: 95.0,
			Warning:  98.0,
			Target:   99.5,
		},
		SLA: SLASpec{
			Availability: 99.9,
			Performance:  99.0,
		},
	}

	kb.PerformanceKPIs["session_establishment_latency"] = &KPISpec{
		Name:        "Session Establishment Latency",
		Type:        "histogram",
		Unit:        "milliseconds",
		Description: "Time to establish PDU session",
		Category:    "performance",
		Thresholds: Thresholds{
			Critical: 1000.0,
			Warning:  500.0,
			Target:   200.0,
		},
	}

	kb.PerformanceKPIs["throughput"] = &KPISpec{
		Name:        "User Plane Throughput",
		Type:        "gauge",
		Unit:        "mbps",
		Description: "Data throughput through user plane",
		Category:    "performance",
		Thresholds: Thresholds{
			Target: 1000.0,
		},
	}
}

// initializeDeploymentPatterns initializes deployment patterns
func (kb *TelecomKnowledgeBase) initializeDeploymentPatterns() {
	kb.DeploymentTypes["high-availability"] = &DeploymentPattern{
		Name:        "high-availability",
		Description: "High availability deployment with geographic redundancy",
		UseCase:     []string{"production", "critical-services", "carrier-grade"},
		Architecture: DeploymentArchitecture{
			Type:         "multi-region",
			Redundancy:   "active-active",
			DataPlane:    "distributed",
			ControlPlane: "distributed",
			ServiceMesh:  true,
		},
		Scaling: ScalingPattern{
			Horizontal: true,
			Vertical:   false,
			Predictive: true,
			Reactive:   true,
		},
		Resilience: ResiliencePattern{
			CircuitBreaker: true,
			Retry:          true,
			Timeout:        true,
			Bulkhead:       true,
			HealthCheck:    true,
		},
		Performance: PerformancePattern{
			Caching:           true,
			LoadBalancing:     true,
			ConnectionPooling: true,
			Compression:       true,
		},
	}

	kb.DeploymentTypes["edge-optimized"] = &DeploymentPattern{
		Name:        "edge-optimized",
		Description: "Edge deployment optimized for low latency",
		UseCase:     []string{"urllc", "edge-computing", "industrial"},
		Architecture: DeploymentArchitecture{
			Type:         "single-zone",
			Redundancy:   "active-passive",
			DataPlane:    "distributed",
			ControlPlane: "centralized",
		},
	}
}

// GetNetworkFunction retrieves network function specification by name
func (kb *TelecomKnowledgeBase) GetNetworkFunction(name string) (*NetworkFunctionSpec, bool) {
	nf, exists := kb.NetworkFunctions[strings.ToLower(name)]
	return nf, exists
}

// GetInterface retrieves interface specification by name
func (kb *TelecomKnowledgeBase) GetInterface(name string) (*InterfaceSpec, bool) {
	iface, exists := kb.Interfaces[strings.ToLower(name)]
	return iface, exists
}

// GetQosProfile retrieves QoS profile by name
func (kb *TelecomKnowledgeBase) GetQosProfile(name string) (*QosProfile, bool) {
	qos, exists := kb.QosProfiles[strings.ToLower(name)]
	return qos, exists
}

// GetSliceType retrieves slice type specification by name
func (kb *TelecomKnowledgeBase) GetSliceType(name string) (*SliceTypeSpec, bool) {
	slice, exists := kb.SliceTypes[strings.ToLower(name)]
	return slice, exists
}

// ListNetworkFunctions returns all available network function names
func (kb *TelecomKnowledgeBase) ListNetworkFunctions() []string {
	var names []string
	for name := range kb.NetworkFunctions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetNetworkFunctionsByType returns network functions of a specific type
func (kb *TelecomKnowledgeBase) GetNetworkFunctionsByType(nfType string) []*NetworkFunctionSpec {
	var functions []*NetworkFunctionSpec
	for _, nf := range kb.NetworkFunctions {
		if nf.Type == nfType {
			functions = append(functions, nf)
		}
	}
	return functions
}

// GetInterfacesForFunction returns all interfaces used by a network function
func (kb *TelecomKnowledgeBase) GetInterfacesForFunction(functionName string) []*InterfaceSpec {
	nf, exists := kb.GetNetworkFunction(functionName)
	if !exists {
		return nil
	}

	var interfaces []*InterfaceSpec
	for _, ifaceName := range nf.Interfaces {
		if iface, exists := kb.GetInterface(ifaceName); exists {
			interfaces = append(interfaces, iface)
		}
	}
	return interfaces
}

// GetDependenciesForFunction returns all dependencies for a network function
func (kb *TelecomKnowledgeBase) GetDependenciesForFunction(functionName string) ([]*NetworkFunctionSpec, []*NetworkFunctionSpec) {
	nf, exists := kb.GetNetworkFunction(functionName)
	if !exists {
		return nil, nil
	}

	var required []*NetworkFunctionSpec
	var optional []*NetworkFunctionSpec

	for _, depName := range nf.Dependencies {
		if dep, exists := kb.GetNetworkFunction(depName); exists {
			required = append(required, dep)
		}
	}

	for _, depName := range nf.OptionalDeps {
		if dep, exists := kb.GetNetworkFunction(depName); exists {
			optional = append(optional, dep)
		}
	}

	return required, optional
}

// ValidateSliceConfiguration validates a network slice configuration
func (kb *TelecomKnowledgeBase) ValidateSliceConfiguration(sliceType, qosProfile string, functions []string) []string {
	var issues []string

	// Check if slice type exists
	slice, exists := kb.GetSliceType(sliceType)
	if !exists {
		issues = append(issues, fmt.Sprintf("Unknown slice type: %s", sliceType))
		return issues
	}

	// Validate QoS profile
	if qosProfile != "" {
		if _, exists := kb.GetQosProfile(qosProfile); !exists {
			issues = append(issues, fmt.Sprintf("Unknown QoS profile: %s", qosProfile))
		}
	}

	// Validate network functions
	for _, funcName := range functions {
		if _, exists := kb.GetNetworkFunction(funcName); !exists {
			issues = append(issues, fmt.Sprintf("Unknown network function: %s", funcName))
		}
	}

	// Check if all required functions are present
	requiredFunctions := slice.NetworkFunctions
	for _, required := range requiredFunctions {
		found := false
		for _, provided := range functions {
			if strings.EqualFold(required, provided) {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("Missing required network function for %s slice: %s", sliceType, required))
		}
	}

	return issues
}

// GetPerformanceBaseline returns performance baseline for a network function
func (kb *TelecomKnowledgeBase) GetPerformanceBaseline(functionName string) (*PerformanceBaseline, bool) {
	nf, exists := kb.GetNetworkFunction(functionName)
	if !exists {
		return nil, false
	}
	return &nf.Performance, true
}

// GetScalingParameters returns scaling parameters for a network function
func (kb *TelecomKnowledgeBase) GetScalingParameters(functionName string) (*ScalingParameters, bool) {
	nf, exists := kb.GetNetworkFunction(functionName)
	if !exists {
		return nil, false
	}
	return &nf.Scaling, true
}

// GetDeploymentPattern returns deployment pattern specification
func (kb *TelecomKnowledgeBase) GetDeploymentPattern(patternName string) (*DeploymentPattern, bool) {
	pattern, exists := kb.DeploymentTypes[strings.ToLower(patternName)]
	return pattern, exists
}

// EstimateResourceRequirements estimates total resource requirements for a set of functions
func (kb *TelecomKnowledgeBase) EstimateResourceRequirements(functions []string, replicas map[string]int) (ResourceRequirements, error) {
	var total ResourceRequirements
	var totalCPU, totalMemory, totalStorage float64

	for _, funcName := range functions {
		nf, exists := kb.GetNetworkFunction(funcName)
		if !exists {
			return total, fmt.Errorf("unknown network function: %s", funcName)
		}

		replicaCount := 1
		if r, ok := replicas[funcName]; ok {
			replicaCount = r
		}

		// Parse and sum CPU (assuming cores)
		var cpu float64
		fmt.Sscanf(nf.Resources.MaxCPU, "%f", &cpu)
		totalCPU += cpu * float64(replicaCount)

		// Parse and sum Memory (assuming Gi)
		var memory float64
		fmt.Sscanf(nf.Resources.MaxMemory, "%fGi", &memory)
		totalMemory += memory * float64(replicaCount)

		// Parse and sum Storage (assuming Gi)
		var storage float64
		fmt.Sscanf(nf.Resources.Storage, "%fGi", &storage)
		totalStorage += storage * float64(replicaCount)
	}

	total.MaxCPU = fmt.Sprintf("%.0f", totalCPU)
	total.MaxMemory = fmt.Sprintf("%.0fGi", totalMemory)
	total.Storage = fmt.Sprintf("%.0fGi", totalStorage)

	return total, nil
}

// GetCompatibleSliceTypes returns slice types compatible with given requirements
func (kb *TelecomKnowledgeBase) GetCompatibleSliceTypes(maxLatency float64, minThroughput int, minReliability float64) []*SliceTypeSpec {
	var compatible []*SliceTypeSpec

	for _, slice := range kb.SliceTypes {
		latencyOK := slice.Requirements.Latency.UserPlane <= maxLatency

		var throughputMin int
		fmt.Sscanf(slice.Requirements.Throughput.Min, "%d", &throughputMin)
		throughputOK := throughputMin >= minThroughput

		reliabilityOK := slice.Requirements.Reliability.Availability >= minReliability

		if latencyOK && throughputOK && reliabilityOK {
			compatible = append(compatible, slice)
		}
	}

	return compatible
}

// GetRecommendedDeploymentConfig returns recommended deployment configuration
func (kb *TelecomKnowledgeBase) GetRecommendedDeploymentConfig(functionName, environment string) (*DeploymentConfig, error) {
	nf, exists := kb.GetNetworkFunction(functionName)
	if !exists {
		return nil, fmt.Errorf("unknown network function: %s", functionName)
	}

	config, exists := nf.DeploymentPatterns[environment]
	if !exists {
		// Return production config as default
		if prodConfig, exists := nf.DeploymentPatterns["production"]; exists {
			return &prodConfig, nil
		}
		return nil, fmt.Errorf("no deployment configuration found for %s in %s environment", functionName, environment)
	}

	return &config, nil
}

// IsInitialized returns whether the knowledge base has been initialized
func (kb *TelecomKnowledgeBase) IsInitialized() bool {
	return kb.initialized
}

// GetVersion returns the knowledge base version
func (kb *TelecomKnowledgeBase) GetVersion() string {
	return "1.0.0-3GPP-R17"
}

// GetLastUpdated returns the last update time (placeholder)
func (kb *TelecomKnowledgeBase) GetLastUpdated() time.Time {
	return time.Now() // In production, this would be from metadata
}
