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

package blueprints

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

const (
	// O-RAN Blueprint Categories
	BlueprintCategoryNearRTRIC = "near-rt-ric"
	BlueprintCategoryNonRTRIC  = "non-rt-ric"
	BlueprintCategoryORAN_DU   = "oran-du"
	BlueprintCategoryORAN_CU   = "oran-cu"
	BlueprintCategoryXApp      = "xapp"
	BlueprintCategoryRApp      = "rapp"
	BlueprintCategorySMO       = "smo"
	BlueprintCategory5GCore    = "5g-core"

	// Blueprint lifecycle states
	BlueprintStatePending    = "Pending"
	BlueprintStateGenerating = "Generating"
	BlueprintStateReady      = "Ready"
	BlueprintStateFailed     = "Failed"
	BlueprintStateDeploying  = "Deploying"
	BlueprintStateDeployed   = "Deployed"

	// O-RAN Interface Types
	InterfaceA1            = "a1"
	InterfaceO1            = "o1"
	InterfaceO2            = "o2"
	InterfaceE2            = "e2"
	InterfaceOpenFronthaul = "open-fronthaul"
)

// ORANBlueprintManager manages O-RAN compliant blueprint packages
type ORANBlueprintManager struct {
	client      client.Client
	porchClient porch.PorchClient
	logger      *zap.Logger
	metrics     *BlueprintMetrics

	// Component managers
	oranCatalog    *ORANBlueprintCatalog
	fiveGCatalog   *FiveGCoreCatalog
	renderEngine   *BlueprintRenderingEngine
	configGen      *NetworkFunctionConfigGenerator
	validator      *ORANValidator
	templateEngine *TemplateEngine

	// Operation management
	operationQueue chan *BlueprintOperation
	workerpool     *sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc

	// Configuration
	config *BlueprintConfig
}

// ORANBlueprintCatalog manages O-RAN specific blueprint templates
type ORANBlueprintCatalog struct {
	NearRTRIC  map[string]*BlueprintTemplate
	NonRTRIC   map[string]*BlueprintTemplate
	ORAN_DU    map[string]*BlueprintTemplate
	ORAN_CU    map[string]*BlueprintTemplate
	xApps      map[string]*BlueprintTemplate
	rApps      map[string]*BlueprintTemplate
	SMO        map[string]*BlueprintTemplate
	interfaces map[string]*InterfaceTemplate
	mutex      sync.RWMutex
}

// FiveGCoreCatalog manages 5G Core network function blueprints
type FiveGCoreCatalog struct {
	AMF   *AmfBlueprintTemplate
	SMF   *SmfBlueprintTemplate
	UPF   *UpfBlueprintTemplate
	NSSF  *NssfBlueprintTemplate
	UDM   *UdmBlueprintTemplate
	AUSF  *AusfBlueprintTemplate
	PCF   *PcfBlueprintTemplate
	NRF   *NrfBlueprintTemplate
	UDR   *UdrBlueprintTemplate
	NWDAF *NwdafBlueprintTemplate
	mutex sync.RWMutex
}

// BlueprintTemplate represents a complete blueprint template
type BlueprintTemplate struct {
	ID            string             `json:"id" yaml:"id"`
	Name          string             `json:"name" yaml:"name"`
	Description   string             `json:"description" yaml:"description"`
	Version       string             `json:"version" yaml:"version"`
	Category      string             `json:"category" yaml:"category"`
	ComponentType v1.NetworkTargetComponent `json:"componentType" yaml:"componentType"`
	IntentTypes   []v1.IntentType    `json:"intentTypes" yaml:"intentTypes"`

	// O-RAN Compliance
	ORANCompliant bool     `json:"oranCompliant" yaml:"oranCompliant"`
	Interfaces    []string `json:"interfaces" yaml:"interfaces"`
	ServiceModels []string `json:"serviceModels,omitempty" yaml:"serviceModels,omitempty"`

	// Templates and configurations
	HelmChart    *HelmChartTemplate    `json:"helmChart,omitempty" yaml:"helmChart,omitempty"`
	KRMResources []KRMResourceTemplate `json:"krmResources" yaml:"krmResources"`
	ConfigMaps   []ConfigMapTemplate   `json:"configMaps,omitempty" yaml:"configMaps,omitempty"`
	Secrets      []SecretTemplate      `json:"secrets,omitempty" yaml:"secrets,omitempty"`

	// Network Function specific
	NetworkConfig *NetworkFunctionConfig `json:"networkConfig,omitempty" yaml:"networkConfig,omitempty"`
	QoSProfile    *QoSProfileConfig      `json:"qosProfile,omitempty" yaml:"qosProfile,omitempty"`
	ScalingPolicy *ScalingPolicyConfig   `json:"scalingPolicy,omitempty" yaml:"scalingPolicy,omitempty"`

	// Dependencies and constraints
	Dependencies []BlueprintDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Constraints  []BlueprintConstraint `json:"constraints,omitempty" yaml:"constraints,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" yaml:"updatedAt"`
	Author    string    `json:"author" yaml:"author"`
	License   string    `json:"license" yaml:"license"`
}

// InterfaceTemplate represents O-RAN interface configuration templates
type InterfaceTemplate struct {
	InterfaceType string            `json:"interfaceType" yaml:"interfaceType"`
	Version       string            `json:"version" yaml:"version"`
	Endpoints     []EndpointConfig  `json:"endpoints" yaml:"endpoints"`
	Protocol      string            `json:"protocol" yaml:"protocol"`
	Security      *SecurityConfig   `json:"security,omitempty" yaml:"security,omitempty"`
	QoS           *QoSConfig        `json:"qos,omitempty" yaml:"qos,omitempty"`
	Monitoring    *MonitoringConfig `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
}

// Network Function Blueprint Templates

// AmfBlueprintTemplate represents AMF-specific blueprint template
type AmfBlueprintTemplate struct {
	*BlueprintTemplate
	SessionManagement    *SessionManagementConfig  `json:"sessionManagement" yaml:"sessionManagement"`
	MobilityManagement   *MobilityManagementConfig `json:"mobilityManagement" yaml:"mobilityManagement"`
	AuthenticationConfig *AuthenticationConfig     `json:"authentication" yaml:"authentication"`
	PLMNConfiguration    []PLMNConfig              `json:"plmnConfig" yaml:"plmnConfig"`
}

// SmfBlueprintTemplate represents SMF-specific blueprint template
type SmfBlueprintTemplate struct {
	*BlueprintTemplate
	PDUSessionManagement *PDUSessionConfig `json:"pduSessionManagement" yaml:"pduSessionManagement"`
	QoSFlowManagement    *QoSFlowConfig    `json:"qosFlowManagement" yaml:"qosFlowManagement"`
	ChargingConfig       *ChargingConfig   `json:"chargingConfig" yaml:"chargingConfig"`
	PolicyEnforcement    *PolicyConfig     `json:"policyEnforcement" yaml:"policyEnforcement"`
}

// UpfBlueprintTemplate represents UPF-specific blueprint template
type UpfBlueprintTemplate struct {
	*BlueprintTemplate
	DataPlaneConfig         *DataPlaneConfig      `json:"dataPlaneConfig" yaml:"dataPlaneConfig"`
	TrafficRouting          *TrafficRoutingConfig `json:"trafficRouting" yaml:"trafficRouting"`
	EdgeDeployment          *EdgeDeploymentConfig `json:"edgeDeployment,omitempty" yaml:"edgeDeployment,omitempty"`
	PerformanceOptimization *PerformanceConfig    `json:"performanceOptimization" yaml:"performanceOptimization"`
}

// Additional 5G Core blueprint templates would follow similar patterns...
type NssfBlueprintTemplate struct {
	*BlueprintTemplate
	SliceSelection *SliceSelectionConfig `json:"sliceSelection" yaml:"sliceSelection"`
	NSIManagement  *NSIManagementConfig  `json:"nsiManagement" yaml:"nsiManagement"`
}

type UdmBlueprintTemplate struct {
	*BlueprintTemplate
	SubscriberData     *SubscriberDataConfig     `json:"subscriberData" yaml:"subscriberData"`
	IdentityManagement *IdentityManagementConfig `json:"identityManagement" yaml:"identityManagement"`
}

type AusfBlueprintTemplate struct {
	*BlueprintTemplate
	AuthenticationServer *AuthenticationServerConfig `json:"authenticationServer" yaml:"authenticationServer"`
	KeyManagement        *KeyManagementConfig        `json:"keyManagement" yaml:"keyManagement"`
}

type PcfBlueprintTemplate struct {
	*BlueprintTemplate
	PolicyControl *PolicyControlConfig `json:"policyControl" yaml:"policyControl"`
	ChargingRules *ChargingRulesConfig `json:"chargingRules" yaml:"chargingRules"`
}

type NrfBlueprintTemplate struct {
	*BlueprintTemplate
	ServiceRegistry  *ServiceRegistryConfig  `json:"serviceRegistry" yaml:"serviceRegistry"`
	ServiceDiscovery *ServiceDiscoveryConfig `json:"serviceDiscovery" yaml:"serviceDiscovery"`
}

type UdrBlueprintTemplate struct {
	*BlueprintTemplate
	DataRepository  *DataRepositoryConfig  `json:"dataRepository" yaml:"dataRepository"`
	DataConsistency *DataConsistencyConfig `json:"dataConsistency" yaml:"dataConsistency"`
}

type NwdafBlueprintTemplate struct {
	*BlueprintTemplate
	AnalyticsEngine *AnalyticsEngineConfig `json:"analyticsEngine" yaml:"analyticsEngine"`
	MLPipeline      *MLPipelineConfig      `json:"mlPipeline" yaml:"mlPipeline"`
}

// Configuration structures for network functions

type NetworkFunctionConfig struct {
	Interfaces           []InterfaceConfig    `json:"interfaces" yaml:"interfaces"`
	ServiceBindings      []ServiceBinding     `json:"serviceBindings" yaml:"serviceBindings"`
	ResourceRequirements ResourceRequirements `json:"resourceRequirements" yaml:"resourceRequirements"`
}

type QoSProfileConfig struct {
	QCI               int     `json:"qci" yaml:"qci"`
	Priority          int     `json:"priority" yaml:"priority"`
	PacketDelayBudget int     `json:"packetDelayBudget" yaml:"packetDelayBudget"`
	PacketErrorRate   float64 `json:"packetErrorRate" yaml:"packetErrorRate"`
	MaxBitRate        string  `json:"maxBitRate" yaml:"maxBitRate"`
	GuaranteedBitRate string  `json:"guaranteedBitRate" yaml:"guaranteedBitRate"`
}

type ScalingPolicyConfig struct {
	MinReplicas     int32        `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas     int32        `json:"maxReplicas" yaml:"maxReplicas"`
	TargetCPU       int32        `json:"targetCPU" yaml:"targetCPU"`
	TargetMemory    int32        `json:"targetMemory" yaml:"targetMemory"`
	ScaleUpPolicy   *ScalePolicy `json:"scaleUpPolicy" yaml:"scaleUpPolicy"`
	ScaleDownPolicy *ScalePolicy `json:"scaleDownPolicy" yaml:"scaleDownPolicy"`
}

type ScalePolicy struct {
	StabilizationWindow int32 `json:"stabilizationWindow" yaml:"stabilizationWindow"`
	PercentagePods      int32 `json:"percentagePods" yaml:"percentagePods"`
	Pods                int32 `json:"pods" yaml:"pods"`
}

// Template structures

type HelmChartTemplate struct {
	Name         string                 `json:"name" yaml:"name"`
	Version      string                 `json:"version" yaml:"version"`
	Repository   string                 `json:"repository" yaml:"repository"`
	Values       map[string]interface{} `json:"values" yaml:"values"`
	Dependencies []HelmDependency       `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

type KRMResourceTemplate struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   map[string]interface{} `json:"metadata" yaml:"metadata"`
	Spec       map[string]interface{} `json:"spec" yaml:"spec"`
	Template   string                 `json:"template,omitempty" yaml:"template,omitempty"`
}

type ConfigMapTemplate struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Data      map[string]string `json:"data" yaml:"data"`
	Template  string            `json:"template,omitempty" yaml:"template,omitempty"`
}

type SecretTemplate struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Type      string            `json:"type" yaml:"type"`
	Data      map[string]string `json:"data" yaml:"data"`
	Template  string            `json:"template,omitempty" yaml:"template,omitempty"`
}

type BlueprintDependency struct {
	ComponentType v1.NetworkTargetComponent `json:"componentType" yaml:"componentType"`
	Version       string             `json:"version" yaml:"version"`
	Required      bool               `json:"required" yaml:"required"`
	Interface     string             `json:"interface,omitempty" yaml:"interface,omitempty"`
}

type BlueprintConstraint struct {
	Type        string      `json:"type" yaml:"type"`
	Condition   string      `json:"condition" yaml:"condition"`
	Value       interface{} `json:"value" yaml:"value"`
	Description string      `json:"description" yaml:"description"`
}

// Configuration specific to network functions

type SessionManagementConfig struct {
	MaxSessions      int  `json:"maxSessions" yaml:"maxSessions"`
	SessionTimeout   int  `json:"sessionTimeout" yaml:"sessionTimeout"`
	HandoverSupport  bool `json:"handoverSupport" yaml:"handoverSupport"`
	EmergencyService bool `json:"emergencyService" yaml:"emergencyService"`
}

type MobilityManagementConfig struct {
	TrackingAreaList   []int  `json:"trackingAreaList" yaml:"trackingAreaList"`
	ReregistrationTime int    `json:"reregistrationTime" yaml:"reregistrationTime"`
	PagingPolicy       string `json:"pagingPolicy" yaml:"pagingPolicy"`
}

type AuthenticationConfig struct {
	SupportedAlgorithms []string `json:"supportedAlgorithms" yaml:"supportedAlgorithms"`
	AKAVersion          string   `json:"akaVersion" yaml:"akaVersion"`
	PrivacySupport      bool     `json:"privacySupport" yaml:"privacySupport"`
}

type PLMNConfig struct {
	MCC          string      `json:"mcc" yaml:"mcc"`
	MNC          string      `json:"mnc" yaml:"mnc"`
	TAC          int         `json:"tac" yaml:"tac"`
	SliceSupport []SliceInfo `json:"sliceSupport" yaml:"sliceSupport"`
}

type SliceInfo struct {
	SST int    `json:"sst" yaml:"sst"`
	SD  string `json:"sd,omitempty" yaml:"sd,omitempty"`
}

type PDUSessionConfig struct {
	MaxPDUSessions      int      `json:"maxPDUSessions" yaml:"maxPDUSessions"`
	DefaultQoS          QoSInfo  `json:"defaultQoS" yaml:"defaultQoS"`
	SessionAmbr         string   `json:"sessionAmbr" yaml:"sessionAmbr"`
	AllowedSessionTypes []string `json:"allowedSessionTypes" yaml:"allowedSessionTypes"`
}

type QoSFlowConfig struct {
	MaxQoSFlows        int       `json:"maxQoSFlows" yaml:"maxQoSFlows"`
	SupportedQoS       []QoSInfo `json:"supportedQoS" yaml:"supportedQoS"`
	FlowControlEnabled bool      `json:"flowControlEnabled" yaml:"flowControlEnabled"`
}

type QoSInfo struct {
	QFI               int     `json:"qfi" yaml:"qfi"`
	FiveQI            int     `json:"fiveQI" yaml:"fiveQI"`
	Priority          int     `json:"priority" yaml:"priority"`
	PacketDelayBudget int     `json:"packetDelayBudget" yaml:"packetDelayBudget"`
	PacketErrorRate   float64 `json:"packetErrorRate" yaml:"packetErrorRate"`
}

type ChargingConfig struct {
	ChargingMode      string `json:"chargingMode" yaml:"chargingMode"`
	RatingGroups      []int  `json:"ratingGroups" yaml:"ratingGroups"`
	MeasurementMethod string `json:"measurementMethod" yaml:"measurementMethod"`
	ReportingInterval int    `json:"reportingInterval" yaml:"reportingInterval"`
}

type PolicyConfig struct {
	PolicyDecisionPoint string             `json:"policyDecisionPoint" yaml:"policyDecisionPoint"`
	PolicyRules         []PolicyRule       `json:"policyRules" yaml:"policyRules"`
	EnforcementPoints   []EnforcementPoint `json:"enforcementPoints" yaml:"enforcementPoints"`
}

type PolicyRule struct {
	ID         string                 `json:"id" yaml:"id"`
	Condition  string                 `json:"condition" yaml:"condition"`
	Action     string                 `json:"action" yaml:"action"`
	Priority   int                    `json:"priority" yaml:"priority"`
	Parameters map[string]interface{} `json:"parameters" yaml:"parameters"`
}

type EnforcementPoint struct {
	ComponentType v1.NetworkTargetComponent `json:"componentType" yaml:"componentType"`
	Interface     string             `json:"interface" yaml:"interface"`
	Method        string             `json:"method" yaml:"method"`
}

type DataPlaneConfig struct {
	InterfaceType       string       `json:"interfaceType" yaml:"interfaceType"`
	EncapsulationMode   string       `json:"encapsulationMode" yaml:"encapsulationMode"`
	SupportedProtocols  []string     `json:"supportedProtocols" yaml:"supportedProtocols"`
	BufferConfiguration BufferConfig `json:"bufferConfiguration" yaml:"bufferConfiguration"`
}

type BufferConfig struct {
	Size         string         `json:"size" yaml:"size"`
	Thresholds   map[string]int `json:"thresholds" yaml:"thresholds"`
	DropPolicies []DropPolicy   `json:"dropPolicies" yaml:"dropPolicies"`
}

type DropPolicy struct {
	Priority  int    `json:"priority" yaml:"priority"`
	Threshold int    `json:"threshold" yaml:"threshold"`
	Action    string `json:"action" yaml:"action"`
}

type TrafficRoutingConfig struct {
	RoutingRules    []RoutingRule         `json:"routingRules" yaml:"routingRules"`
	LoadBalancing   LoadBalancingConfig   `json:"loadBalancing" yaml:"loadBalancing"`
	TrafficSteering TrafficSteeringConfig `json:"trafficSteering" yaml:"trafficSteering"`
}

type RoutingRule struct {
	ID          string `json:"id" yaml:"id"`
	Source      string `json:"source" yaml:"source"`
	Destination string `json:"destination" yaml:"destination"`
	Protocol    string `json:"protocol" yaml:"protocol"`
	Action      string `json:"action" yaml:"action"`
	Priority    int    `json:"priority" yaml:"priority"`
}

type LoadBalancingConfig struct {
	Algorithm       string            `json:"algorithm" yaml:"algorithm"`
	HealthCheck     HealthCheckConfig `json:"healthCheck" yaml:"healthCheck"`
	SessionAffinity bool              `json:"sessionAffinity" yaml:"sessionAffinity"`
	Weights         map[string]int    `json:"weights" yaml:"weights"`
}

type HealthCheckConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Interval int    `json:"interval" yaml:"interval"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Path     string `json:"path" yaml:"path"`
}

type TrafficSteeringConfig struct {
	Enabled  bool           `json:"enabled" yaml:"enabled"`
	Rules    []SteeringRule `json:"rules" yaml:"rules"`
	Fallback string         `json:"fallback" yaml:"fallback"`
}

type SteeringRule struct {
	Criteria    string `json:"criteria" yaml:"criteria"`
	Destination string `json:"destination" yaml:"destination"`
	Weight      int    `json:"weight" yaml:"weight"`
}

type EdgeDeploymentConfig struct {
	Enabled       bool               `json:"enabled" yaml:"enabled"`
	EdgeLocations []EdgeLocation     `json:"edgeLocations" yaml:"edgeLocations"`
	Optimization  OptimizationConfig `json:"optimization" yaml:"optimization"`
}

type EdgeLocation struct {
	Name      string         `json:"name" yaml:"name"`
	Region    string         `json:"region" yaml:"region"`
	Resources ResourceLimits `json:"resources" yaml:"resources"`
	Latency   LatencyConfig  `json:"latency" yaml:"latency"`
}

type OptimizationConfig struct {
	CPU     bool `json:"cpu" yaml:"cpu"`
	Memory  bool `json:"memory" yaml:"memory"`
	Network bool `json:"network" yaml:"network"`
	Storage bool `json:"storage" yaml:"storage"`
}

type LatencyConfig struct {
	Target int `json:"target" yaml:"target"`
	Max    int `json:"max" yaml:"max"`
	Jitter int `json:"jitter" yaml:"jitter"`
}

type PerformanceConfig struct {
	Throughput    ThroughputConfig     `json:"throughput" yaml:"throughput"`
	Latency       LatencyOptimization  `json:"latency" yaml:"latency"`
	ResourceUsage ResourceOptimization `json:"resourceUsage" yaml:"resourceUsage"`
}

type ThroughputConfig struct {
	Target    string `json:"target" yaml:"target"`
	Burst     string `json:"burst" yaml:"burst"`
	Sustained string `json:"sustained" yaml:"sustained"`
}

type LatencyOptimization struct {
	Processing   int  `json:"processing" yaml:"processing"`
	Queuing      int  `json:"queuing" yaml:"queuing"`
	Forwarding   int  `json:"forwarding" yaml:"forwarding"`
	Optimization bool `json:"optimization" yaml:"optimization"`
}

type ResourceOptimization struct {
	CPUAffinity      bool  `json:"cpuAffinity" yaml:"cpuAffinity"`
	NUMATopology     bool  `json:"numaTopology" yaml:"numaTopology"`
	HugePagesEnabled bool  `json:"hugePagesEnabled" yaml:"hugePagesEnabled"`
	IsolatedCores    []int `json:"isolatedCores" yaml:"isolatedCores"`
}

// Additional configuration types for other network functions
type SliceSelectionConfig struct {
	SelectionRules []SliceSelectionRule `json:"selectionRules" yaml:"selectionRules"`
	DefaultSlice   SliceInfo            `json:"defaultSlice" yaml:"defaultSlice"`
	LoadBalancing  bool                 `json:"loadBalancing" yaml:"loadBalancing"`
}

type SliceSelectionRule struct {
	Criteria    string    `json:"criteria" yaml:"criteria"`
	TargetSlice SliceInfo `json:"targetSlice" yaml:"targetSlice"`
	Priority    int       `json:"priority" yaml:"priority"`
}

type NSIManagementConfig struct {
	MaxNSI          int             `json:"maxNSI" yaml:"maxNSI"`
	LifecyclePolicy LifecyclePolicy `json:"lifecyclePolicy" yaml:"lifecyclePolicy"`
	ResourceSharing bool            `json:"resourceSharing" yaml:"resourceSharing"`
}

type LifecyclePolicy struct {
	InstantiationTimeout int `json:"instantiationTimeout" yaml:"instantiationTimeout"`
	TerminationTimeout   int `json:"terminationTimeout" yaml:"terminationTimeout"`
	ModificationTimeout  int `json:"modificationTimeout" yaml:"modificationTimeout"`
}

type SubscriberDataConfig struct {
	DataConsistency   string         `json:"dataConsistency" yaml:"dataConsistency"`
	ReplicationFactor int            `json:"replicationFactor" yaml:"replicationFactor"`
	BackupStrategy    BackupStrategy `json:"backupStrategy" yaml:"backupStrategy"`
}

type BackupStrategy struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Frequency string `json:"frequency" yaml:"frequency"`
	Retention string `json:"retention" yaml:"retention"`
	Storage   string `json:"storage" yaml:"storage"`
}

type IdentityManagementConfig struct {
	IdentityTypes     []string         `json:"identityTypes" yaml:"identityTypes"`
	PrivacyProtection bool             `json:"privacyProtection" yaml:"privacyProtection"`
	Encryption        EncryptionConfig `json:"encryption" yaml:"encryption"`
}

type EncryptionConfig struct {
	Algorithm      string `json:"algorithm" yaml:"algorithm"`
	KeyLength      int    `json:"keyLength" yaml:"keyLength"`
	KeyRotation    bool   `json:"keyRotation" yaml:"keyRotation"`
	RotationPeriod int    `json:"rotationPeriod" yaml:"rotationPeriod"`
}

type AuthenticationServerConfig struct {
	SupportedMethods  []string              `json:"supportedMethods" yaml:"supportedMethods"`
	CertificateConfig CertificateConfig     `json:"certificateConfig" yaml:"certificateConfig"`
	TokenManagement   TokenManagementConfig `json:"tokenManagement" yaml:"tokenManagement"`
}

type CertificateConfig struct {
	CertificateAuthority string `json:"certificateAuthority" yaml:"certificateAuthority"`
	ValidityPeriod       int    `json:"validityPeriod" yaml:"validityPeriod"`
	RevocationMethod     string `json:"revocationMethod" yaml:"revocationMethod"`
}

type TokenManagementConfig struct {
	TokenLifetime   int  `json:"tokenLifetime" yaml:"tokenLifetime"`
	RefreshEnabled  bool `json:"refreshEnabled" yaml:"refreshEnabled"`
	RefreshLifetime int  `json:"refreshLifetime" yaml:"refreshLifetime"`
}

type KeyManagementConfig struct {
	KeyDerivationFunction string            `json:"keyDerivationFunction" yaml:"keyDerivationFunction"`
	KeyStorage            KeyStorageConfig  `json:"keyStorage" yaml:"keyStorage"`
	KeyRotationPolicy     KeyRotationPolicy `json:"keyRotationPolicy" yaml:"keyRotationPolicy"`
}

type KeyStorageConfig struct {
	StorageType   string `json:"storageType" yaml:"storageType"`
	EncryptionKey string `json:"encryptionKey" yaml:"encryptionKey"`
	AccessControl string `json:"accessControl" yaml:"accessControl"`
}

type KeyRotationPolicy struct {
	AutoRotation   bool `json:"autoRotation" yaml:"autoRotation"`
	RotationPeriod int  `json:"rotationPeriod" yaml:"rotationPeriod"`
	BackupKeys     int  `json:"backupKeys" yaml:"backupKeys"`
}

type PolicyControlConfig struct {
	PolicyStore     PolicyStoreConfig    `json:"policyStore" yaml:"policyStore"`
	DecisionEngine  DecisionEngineConfig `json:"decisionEngine" yaml:"decisionEngine"`
	EnforcementMode string               `json:"enforcementMode" yaml:"enforcementMode"`
}

type PolicyStoreConfig struct {
	StorageType  string `json:"storageType" yaml:"storageType"`
	Consistency  string `json:"consistency" yaml:"consistency"`
	CacheEnabled bool   `json:"cacheEnabled" yaml:"cacheEnabled"`
	CacheTTL     int    `json:"cacheTTL" yaml:"cacheTTL"`
}

type DecisionEngineConfig struct {
	Algorithm          string `json:"algorithm" yaml:"algorithm"`
	ConcurrentRequests int    `json:"concurrentRequests" yaml:"concurrentRequests"`
	TimeoutMs          int    `json:"timeoutMs" yaml:"timeoutMs"`
}

type ChargingRulesConfig struct {
	RuleEngine        RuleEngineConfig `json:"ruleEngine" yaml:"ruleEngine"`
	MeteringEnabled   bool             `json:"meteringEnabled" yaml:"meteringEnabled"`
	ReportingInterval int              `json:"reportingInterval" yaml:"reportingInterval"`
}

type RuleEngineConfig struct {
	MaxRules       int    `json:"maxRules" yaml:"maxRules"`
	EvaluationMode string `json:"evaluationMode" yaml:"evaluationMode"`
	CacheSize      int    `json:"cacheSize" yaml:"cacheSize"`
}

type ServiceRegistryConfig struct {
	MaxServices       int                  `json:"maxServices" yaml:"maxServices"`
	HeartbeatInterval int                  `json:"heartbeatInterval" yaml:"heartbeatInterval"`
	ServiceTimeout    int                  `json:"serviceTimeout" yaml:"serviceTimeout"`
	LoadBalancing     ServiceLoadBalancing `json:"loadBalancing" yaml:"loadBalancing"`
}

type ServiceLoadBalancing struct {
	Enabled     bool   `json:"enabled" yaml:"enabled"`
	Algorithm   string `json:"algorithm" yaml:"algorithm"`
	HealthCheck bool   `json:"healthCheck" yaml:"healthCheck"`
}

type ServiceDiscoveryConfig struct {
	DiscoveryMethods []string           `json:"discoveryMethods" yaml:"discoveryMethods"`
	CacheConfig      ServiceCacheConfig `json:"cacheConfig" yaml:"cacheConfig"`
	FilterEnabled    bool               `json:"filterEnabled" yaml:"filterEnabled"`
}

type ServiceCacheConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
	TTL     int  `json:"ttl" yaml:"ttl"`
	MaxSize int  `json:"maxSize" yaml:"maxSize"`
	Preload bool `json:"preload" yaml:"preload"`
}

type DataRepositoryConfig struct {
	StorageEngine     StorageEngineConfig `json:"storageEngine" yaml:"storageEngine"`
	ReplicationConfig ReplicationConfig   `json:"replicationConfig" yaml:"replicationConfig"`
	IndexingStrategy  IndexingConfig      `json:"indexingStrategy" yaml:"indexingStrategy"`
}

type StorageEngineConfig struct {
	EngineType      string `json:"engineType" yaml:"engineType"`
	ConnectionPool  int    `json:"connectionPool" yaml:"connectionPool"`
	QueryTimeout    int    `json:"queryTimeout" yaml:"queryTimeout"`
	TransactionMode string `json:"transactionMode" yaml:"transactionMode"`
}

type ReplicationConfig struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	ReplicaCount int    `json:"replicaCount" yaml:"replicaCount"`
	SyncMode     string `json:"syncMode" yaml:"syncMode"`
	FailoverTime int    `json:"failoverTime" yaml:"failoverTime"`
}

type IndexingConfig struct {
	PrimaryIndex     bool     `json:"primaryIndex" yaml:"primaryIndex"`
	SecondaryIndexes []string `json:"secondaryIndexes" yaml:"secondaryIndexes"`
	FullTextSearch   bool     `json:"fullTextSearch" yaml:"fullTextSearch"`
}

type DataConsistencyConfig struct {
	ConsistencyLevel   string           `json:"consistencyLevel" yaml:"consistencyLevel"`
	ConflictResolution string           `json:"conflictResolution" yaml:"conflictResolution"`
	ValidationRules    []ValidationRule `json:"validationRules" yaml:"validationRules"`
}

type ValidationRule struct {
	Field       string   `json:"field" yaml:"field"`
	Type        string   `json:"type" yaml:"type"`
	Required    bool     `json:"required" yaml:"required"`
	Constraints []string `json:"constraints" yaml:"constraints"`
}

type AnalyticsEngineConfig struct {
	ProcessingMode      string                 `json:"processingMode" yaml:"processingMode"`
	DataSources         []DataSourceConfig     `json:"dataSources" yaml:"dataSources"`
	AnalyticsModels     []AnalyticsModelConfig `json:"analyticsModels" yaml:"analyticsModels"`
	OutputConfiguration OutputConfig           `json:"outputConfiguration" yaml:"outputConfiguration"`
}

type DataSourceConfig struct {
	Name           string            `json:"name" yaml:"name"`
	Type           string            `json:"type" yaml:"type"`
	ConnectionInfo map[string]string `json:"connectionInfo" yaml:"connectionInfo"`
	SamplingRate   float64           `json:"samplingRate" yaml:"samplingRate"`
}

type AnalyticsModelConfig struct {
	ModelID        string                 `json:"modelId" yaml:"modelId"`
	ModelType      string                 `json:"modelType" yaml:"modelType"`
	TrainingData   string                 `json:"trainingData" yaml:"trainingData"`
	Parameters     map[string]interface{} `json:"parameters" yaml:"parameters"`
	UpdateInterval int                    `json:"updateInterval" yaml:"updateInterval"`
}

type OutputConfig struct {
	Destinations []OutputDestination `json:"destinations" yaml:"destinations"`
	Format       string              `json:"format" yaml:"format"`
	Aggregation  AggregationConfig   `json:"aggregation" yaml:"aggregation"`
}

type OutputDestination struct {
	Name        string            `json:"name" yaml:"name"`
	Type        string            `json:"type" yaml:"type"`
	Endpoint    string            `json:"endpoint" yaml:"endpoint"`
	Credentials map[string]string `json:"credentials" yaml:"credentials"`
}

type AggregationConfig struct {
	Enabled   bool     `json:"enabled" yaml:"enabled"`
	Window    int      `json:"window" yaml:"window"`
	Functions []string `json:"functions" yaml:"functions"`
	Grouping  []string `json:"grouping" yaml:"grouping"`
}

type MLPipelineConfig struct {
	PipelineStages   []PipelineStage  `json:"pipelineStages" yaml:"pipelineStages"`
	ModelManagement  ModelManagement  `json:"modelManagement" yaml:"modelManagement"`
	TrainingSchedule TrainingSchedule `json:"trainingSchedule" yaml:"trainingSchedule"`
}

type PipelineStage struct {
	Name          string                 `json:"name" yaml:"name"`
	Type          string                 `json:"type" yaml:"type"`
	Configuration map[string]interface{} `json:"configuration" yaml:"configuration"`
	Dependencies  []string               `json:"dependencies" yaml:"dependencies"`
}

type ModelManagement struct {
	Versioning        bool   `json:"versioning" yaml:"versioning"`
	ModelRegistry     string `json:"modelRegistry" yaml:"modelRegistry"`
	A_BTestingEnabled bool   `json:"abTestingEnabled" yaml:"abTestingEnabled"`
	RollbackEnabled   bool   `json:"rollbackEnabled" yaml:"rollbackEnabled"`
}

type TrainingSchedule struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	Frequency  string `json:"frequency" yaml:"frequency"`
	DataWindow int    `json:"dataWindow" yaml:"dataWindow"`
	AutoDeploy bool   `json:"autoDeploy" yaml:"autoDeploy"`
}

// Common configuration types used across network functions

type InterfaceConfig struct {
	Name       string            `json:"name" yaml:"name"`
	Type       string            `json:"type" yaml:"type"`
	Protocol   string            `json:"protocol" yaml:"protocol"`
	Address    string            `json:"address" yaml:"address"`
	Port       int               `json:"port" yaml:"port"`
	TLS        *TLSConfig        `json:"tls,omitempty" yaml:"tls,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type TLSConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	CertFile   string `json:"certFile" yaml:"certFile"`
	KeyFile    string `json:"keyFile" yaml:"keyFile"`
	CAFile     string `json:"caFile" yaml:"caFile"`
	SkipVerify bool   `json:"skipVerify" yaml:"skipVerify"`
}

type ServiceBinding struct {
	Service      string              `json:"service" yaml:"service"`
	Interface    string              `json:"interface" yaml:"interface"`
	Protocol     string              `json:"protocol" yaml:"protocol"`
	LoadBalancer *LoadBalancerConfig `json:"loadBalancer,omitempty" yaml:"loadBalancer,omitempty"`
}

type LoadBalancerConfig struct {
	Type        string             `json:"type" yaml:"type"`
	Algorithm   string             `json:"algorithm" yaml:"algorithm"`
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
}

type ResourceRequirements struct {
	CPU              string           `json:"cpu" yaml:"cpu"`
	Memory           string           `json:"memory" yaml:"memory"`
	Storage          string           `json:"storage" yaml:"storage"`
	EphemeralStorage string           `json:"ephemeralStorage,omitempty" yaml:"ephemeralStorage,omitempty"`
	Limits           ResourceLimits   `json:"limits" yaml:"limits"`
	Requests         ResourceRequests `json:"requests" yaml:"requests"`
}

type ResourceLimits struct {
	CPU     string `json:"cpu" yaml:"cpu"`
	Memory  string `json:"memory" yaml:"memory"`
	Storage string `json:"storage" yaml:"storage"`
}

type ResourceRequests struct {
	CPU     string `json:"cpu" yaml:"cpu"`
	Memory  string `json:"memory" yaml:"memory"`
	Storage string `json:"storage" yaml:"storage"`
}

type EndpointConfig struct {
	Name     string            `json:"name" yaml:"name"`
	Address  string            `json:"address" yaml:"address"`
	Port     int               `json:"port" yaml:"port"`
	Protocol string            `json:"protocol" yaml:"protocol"`
	Path     string            `json:"path,omitempty" yaml:"path,omitempty"`
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

type SecurityConfig struct {
	Authentication *AuthConfig       `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	Authorization  *AuthzConfig      `json:"authorization,omitempty" yaml:"authorization,omitempty"`
	Encryption     *EncryptionConfig `json:"encryption,omitempty" yaml:"encryption,omitempty"`
}

type AuthConfig struct {
	Type       string            `json:"type" yaml:"type"`
	Parameters map[string]string `json:"parameters" yaml:"parameters"`
}

type AuthzConfig struct {
	Type     string   `json:"type" yaml:"type"`
	Policies []string `json:"policies" yaml:"policies"`
	Roles    []string `json:"roles" yaml:"roles"`
}

type QoSConfig struct {
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	Profile    string            `json:"profile" yaml:"profile"`
	Parameters map[string]string `json:"parameters" yaml:"parameters"`
}

type MonitoringConfig struct {
	Enabled  bool           `json:"enabled" yaml:"enabled"`
	Metrics  MetricsConfig  `json:"metrics" yaml:"metrics"`
	Logging  LoggingConfig  `json:"logging" yaml:"logging"`
	Tracing  TracingConfig  `json:"tracing" yaml:"tracing"`
	Alerting AlertingConfig `json:"alerting" yaml:"alerting"`
}

type MetricsConfig struct {
	Enabled  bool     `json:"enabled" yaml:"enabled"`
	Endpoint string   `json:"endpoint" yaml:"endpoint"`
	Interval int      `json:"interval" yaml:"interval"`
	Metrics  []string `json:"metrics" yaml:"metrics"`
}

type LoggingConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Level   string `json:"level" yaml:"level"`
	Format  string `json:"format" yaml:"format"`
	Output  string `json:"output" yaml:"output"`
}

type TracingConfig struct {
	Enabled    bool    `json:"enabled" yaml:"enabled"`
	Endpoint   string  `json:"endpoint" yaml:"endpoint"`
	SampleRate float64 `json:"sampleRate" yaml:"sampleRate"`
}

type AlertingConfig struct {
	Enabled  bool           `json:"enabled" yaml:"enabled"`
	Rules    []AlertRule    `json:"rules" yaml:"rules"`
	Channels []AlertChannel `json:"channels" yaml:"channels"`
}

type AlertRule struct {
	Name      string            `json:"name" yaml:"name"`
	Condition string            `json:"condition" yaml:"condition"`
	Threshold float64           `json:"threshold" yaml:"threshold"`
	Duration  int               `json:"duration" yaml:"duration"`
	Severity  string            `json:"severity" yaml:"severity"`
	Labels    map[string]string `json:"labels" yaml:"labels"`
}

type AlertChannel struct {
	Name   string            `json:"name" yaml:"name"`
	Type   string            `json:"type" yaml:"type"`
	Config map[string]string `json:"config" yaml:"config"`
}

type HelmDependency struct {
	Name       string `json:"name" yaml:"name"`
	Version    string `json:"version" yaml:"version"`
	Repository string `json:"repository" yaml:"repository"`
	Optional   bool   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// BlueprintMetrics contains Prometheus metrics for blueprint operations
type BlueprintMetrics struct {
	BlueprintGenerations    prometheus.Counter
	BlueprintErrors         prometheus.Counter
	BlueprintDuration       prometheus.Histogram
	TemplateCacheHits       prometheus.Counter
	TemplateCacheMisses     prometheus.Counter
	ValidationDuration      prometheus.Histogram
	RenderingDuration       prometheus.Histogram
	ORANCompliantBlueprints prometheus.Counter
}

// NewBlueprintMetrics creates new blueprint metrics
func NewBlueprintMetrics() *BlueprintMetrics {
	return &BlueprintMetrics{
		BlueprintGenerations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_generations_total",
			Help: "Total number of blueprint generations",
		}),
		BlueprintErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_errors_total",
			Help: "Total number of blueprint generation errors",
		}),
		BlueprintDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_blueprint_duration_seconds",
			Help:    "Duration of blueprint generation operations",
			Buckets: prometheus.DefBuckets,
		}),
		TemplateCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_template_cache_hits_total",
			Help: "Total number of template cache hits",
		}),
		TemplateCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_template_cache_misses_total",
			Help: "Total number of template cache misses",
		}),
		ValidationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_blueprint_validation_duration_seconds",
			Help:    "Duration of blueprint validation operations",
			Buckets: prometheus.DefBuckets,
		}),
		RenderingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_blueprint_rendering_duration_seconds",
			Help:    "Duration of blueprint rendering operations",
			Buckets: prometheus.DefBuckets,
		}),
		ORANCompliantBlueprints: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_oran_compliant_blueprints_total",
			Help: "Total number of O-RAN compliant blueprints generated",
		}),
	}
}

// BlueprintConfig contains configuration for the blueprint manager
type BlueprintConfig struct {
	PorchEndpoint        string
	TemplateRepository   string
	CacheTTL             time.Duration
	MaxConcurrentOps     int
	EnableValidation     bool
	EnableORANCompliance bool
	DefaultNamespace     string
}

// NewORANBlueprintManager creates a new O-RAN blueprint manager
func NewORANBlueprintManager(
	client client.Client,
	porchClient porch.PorchClient,
	config *BlueprintConfig,
	logger *zap.Logger,
) (*ORANBlueprintManager, error) {
	if config == nil {
		config = &BlueprintConfig{
			PorchEndpoint:        "http://porch-server.porch-system.svc.cluster.local:9080",
			TemplateRepository:   "https://github.com/nephio-project/catalog.git",
			CacheTTL:             15 * time.Minute,
			MaxConcurrentOps:     10,
			EnableValidation:     true,
			EnableORANCompliance: true,
			DefaultNamespace:     "default",
		}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &ORANBlueprintManager{
		client:         client,
		porchClient:    porchClient,
		logger:         logger,
		metrics:        NewBlueprintMetrics(),
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		workerpool:     &sync.WaitGroup{},
		operationQueue: make(chan *BlueprintOperation, config.MaxConcurrentOps*2),
	}

	// Initialize component managers
	if err := manager.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Start workers
	manager.startWorkers()

	logger.Info("O-RAN Blueprint Manager initialized successfully",
		zap.String("porch_endpoint", config.PorchEndpoint),
		zap.Bool("oran_compliance", config.EnableORANCompliance),
		zap.Int("max_concurrent_ops", config.MaxConcurrentOps))

	return manager, nil
}

// initializeComponents initializes all component managers
func (obm *ORANBlueprintManager) initializeComponents() error {
	var err error

	// Initialize O-RAN catalog
	obm.oranCatalog, err = NewORANBlueprintCatalog(obm.config, obm.logger.Named("oran-catalog"))
	if err != nil {
		return fmt.Errorf("failed to initialize O-RAN catalog: %w", err)
	}

	// Initialize 5G Core catalog
	obm.fiveGCatalog, err = NewFiveGCoreCatalog(obm.config, obm.logger.Named("5g-catalog"))
	if err != nil {
		return fmt.Errorf("failed to initialize 5G Core catalog: %w", err)
	}

	// Initialize rendering engine
	obm.renderEngine, err = NewBlueprintRenderingEngine(obm.config, obm.logger.Named("renderer"))
	if err != nil {
		return fmt.Errorf("failed to initialize rendering engine: %w", err)
	}

	// Initialize network function config generator
	obm.configGen, err = NewNetworkFunctionConfigGenerator(obm.config, obm.logger.Named("config-gen"))
	if err != nil {
		return fmt.Errorf("failed to initialize config generator: %w", err)
	}

	// Initialize O-RAN validator if enabled
	if obm.config.EnableORANCompliance {
		obm.validator, err = NewORANValidator(obm.config, obm.logger.Named("validator"))
		if err != nil {
			return fmt.Errorf("failed to initialize O-RAN validator: %w", err)
		}
	}

	// Initialize template engine
	obm.templateEngine, err = NewTemplateEngine(obm.config, obm.logger.Named("template-engine"))
	if err != nil {
		return fmt.Errorf("failed to initialize template engine: %w", err)
	}

	return nil
}

// startWorkers starts background worker goroutines
func (obm *ORANBlueprintManager) startWorkers() {
	// Start blueprint processing workers
	for i := 0; i < obm.config.MaxConcurrentOps; i++ {
		obm.workerpool.Add(1)
		go obm.blueprintWorker()
	}
}

// CreateORANBlueprint creates an O-RAN compliant blueprint from NetworkIntent
func (obm *ORANBlueprintManager) CreateORANBlueprint(ctx context.Context, intent *v1.NetworkIntent) (*porch.PackageRevision, error) {
	startTime := time.Now()
	obm.metrics.BlueprintGenerations.Inc()

	defer func() {
		duration := time.Since(startTime)
		obm.metrics.BlueprintDuration.Observe(duration.Seconds())
	}()

	obm.logger.Info("Creating O-RAN blueprint",
		zap.String("intent_name", intent.Name),
		zap.String("intent_type", string(intent.Spec.IntentType)),
		zap.Any("target_components", intent.Spec.TargetComponents))

	// Step 1: Select appropriate blueprint templates
	templates, err := obm.selectBlueprintTemplates(ctx, intent)
	if err != nil {
		obm.metrics.BlueprintErrors.Inc()
		return nil, fmt.Errorf("failed to select blueprint templates: %w", err)
	}

	// Step 2: Render blueprint with NetworkIntent parameters
	renderedBlueprint, err := obm.renderEngine.RenderORANBlueprint(ctx, &BlueprintRequest{
		Intent:    intent,
		Templates: templates,
		Metadata:  obm.buildBlueprintMetadata(intent),
	})
	if err != nil {
		obm.metrics.BlueprintErrors.Inc()
		return nil, fmt.Errorf("failed to render blueprint: %w", err)
	}

	// Step 3: Generate network function configurations
	nfConfigs, err := obm.configGen.GenerateConfigurations(ctx, intent, templates)
	if err != nil {
		obm.metrics.BlueprintErrors.Inc()
		return nil, fmt.Errorf("failed to generate network function configurations: %w", err)
	}

	// Step 4: Validate O-RAN compliance if enabled
	if obm.validator != nil {
		if err := obm.validator.ValidateORANCompliance(ctx, renderedBlueprint, nfConfigs); err != nil {
			obm.metrics.BlueprintErrors.Inc()
			return nil, fmt.Errorf("O-RAN compliance validation failed: %w", err)
		}
	}

	// Step 5: Create PackageRevision through Porch
	packageRevision, err := obm.createPackageRevision(ctx, intent, renderedBlueprint, nfConfigs)
	if err != nil {
		obm.metrics.BlueprintErrors.Inc()
		return nil, fmt.Errorf("failed to create package revision: %w", err)
	}

	if renderedBlueprint.ORANCompliant {
		obm.metrics.ORANCompliantBlueprints.Inc()
	}

	obm.logger.Info("O-RAN blueprint created successfully",
		zap.String("intent_name", intent.Name),
		zap.String("package_name", packageRevision.Name),
		zap.Bool("oran_compliant", renderedBlueprint.ORANCompliant),
		zap.Duration("duration", time.Since(startTime)))

	return packageRevision, nil
}
