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

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CNFDeploymentIntent defines CNF deployment specifications derived from NetworkIntent
type CNFDeploymentIntent struct {
	// CNF type classification
	// +optional
	CNFType CNFType `json:"cnfType,omitempty"`

	// Specific CNF function
	// +optional
	Function CNFFunction `json:"function,omitempty"`

	// Deployment strategy preference
	// +optional
	// +kubebuilder:validation:Enum=Helm;Operator;Direct;GitOps
	DeploymentStrategy DeploymentStrategy `json:"deploymentStrategy,omitempty"`

	// Number of replicas
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Replicas *int32 `json:"replicas,omitempty"`

	// Resource requirements
	// +optional
	Resources *CNFResourceIntent `json:"resources,omitempty"`

	// Auto-scaling preferences
	// +optional
	AutoScaling *AutoScalingIntent `json:"autoScaling,omitempty"`

	// Service mesh integration preference
	// +optional
	ServiceMesh *ServiceMeshIntent `json:"serviceMesh,omitempty"`

	// Monitoring preferences
	// +optional
	Monitoring *MonitoringIntent `json:"monitoring,omitempty"`

	// Security requirements
	// +optional
	Security *SecurityIntent `json:"security,omitempty"`

	// High availability requirements
	// +optional
	HighAvailability *HighAvailabilityIntent `json:"highAvailability,omitempty"`

	// Performance requirements
	// +optional
	Performance *PerformanceIntent `json:"performance,omitempty"`

	// Network slice requirements
	// +optional
	NetworkSlicing *NetworkSlicingIntent `json:"networkSlicing,omitempty"`
}

// CNFResourceIntent defines resource requirements from intent processing
type CNFResourceIntent struct {
	// CPU resource requirements
	// +optional
	CPU *resource.Quantity `json:"cpu,omitempty"`

	// Memory resource requirements
	// +optional
	Memory *resource.Quantity `json:"memory,omitempty"`

	// Storage resource requirements
	// +optional
	Storage *resource.Quantity `json:"storage,omitempty"`

	// GPU requirements
	// +optional
	GPU *int32 `json:"gpu,omitempty"`

	// DPDK requirements
	// +optional
	DPDK *DPDKIntent `json:"dpdk,omitempty"`

	// Hugepages requirements
	// +optional
	Hugepages map[string]resource.Quantity `json:"hugepages,omitempty"`

	// Performance tier (low, medium, high, extreme)
	// +optional
	// +kubebuilder:validation:Enum=low;medium;high;extreme
	PerformanceTier string `json:"performanceTier,omitempty"`
}

// DPDKIntent defines DPDK requirements from intent
type DPDKIntent struct {
	// Enable DPDK
	Enabled bool `json:"enabled"`

	// Number of cores
	// +optional
	Cores *int32 `json:"cores,omitempty"`

	// Memory in MB
	// +optional
	Memory *int32 `json:"memory,omitempty"`

	// Driver preference
	// +optional
	Driver string `json:"driver,omitempty"`
}

// AutoScalingIntent defines auto-scaling preferences from intent
type AutoScalingIntent struct {
	// Enable auto-scaling
	Enabled bool `json:"enabled"`

	// Minimum replicas
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// Maximum replicas
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// Target CPU utilization
	// +optional
	TargetCPUUtilization *int32 `json:"targetCpuUtilization,omitempty"`

	// Target memory utilization
	// +optional
	TargetMemoryUtilization *int32 `json:"targetMemoryUtilization,omitempty"`

	// Custom metrics
	// +optional
	CustomMetrics []string `json:"customMetrics,omitempty"`

	// Scaling policy (aggressive, moderate, conservative)
	// +optional
	// +kubebuilder:validation:Enum=aggressive;moderate;conservative
	ScalingPolicy string `json:"scalingPolicy,omitempty"`
}

// ServiceMeshIntent defines service mesh preferences from intent
type ServiceMeshIntent struct {
	// Enable service mesh
	Enabled bool `json:"enabled"`

	// Preferred service mesh type
	// +optional
	// +kubebuilder:validation:Enum=istio;linkerd;consul
	Type string `json:"type,omitempty"`

	// mTLS requirements
	// +optional
	MTLS *MTLSIntent `json:"mtls,omitempty"`

	// Traffic management preferences
	// +optional
	TrafficManagement []string `json:"trafficManagement,omitempty"`
}

// MTLSIntent defines mTLS preferences
type MTLSIntent struct {
	// Enable mTLS
	Enabled bool `json:"enabled"`

	// mTLS mode preference
	// +optional
	// +kubebuilder:validation:Enum=strict;permissive
	Mode string `json:"mode,omitempty"`
}

// MonitoringIntent defines monitoring preferences from intent
type MonitoringIntent struct {
	// Enable monitoring
	Enabled bool `json:"enabled"`

	// Metrics collection preferences
	// +optional
	Metrics []string `json:"metrics,omitempty"`

	// Alerting preferences
	// +optional
	Alerts []string `json:"alerts,omitempty"`

	// Dashboard requirements
	// +optional
	Dashboards []string `json:"dashboards,omitempty"`

	// Logging level preference
	// +optional
	// +kubebuilder:validation:Enum=debug;info;warn;error
	LogLevel string `json:"logLevel,omitempty"`

	// Tracing requirements
	// +optional
	TracingEnabled bool `json:"tracingEnabled,omitempty"`
}

// SecurityIntent defines security requirements from intent
type SecurityIntent struct {
	// Security level (basic, standard, high, strict)
	// +optional
	// +kubebuilder:validation:Enum=basic;standard;high;strict
	Level string `json:"level,omitempty"`

	// Encryption requirements
	// +optional
	Encryption []string `json:"encryption,omitempty"`

	// Authentication requirements
	// +optional
	Authentication []string `json:"authentication,omitempty"`

	// Network policies
	// +optional
	NetworkPolicies []string `json:"networkPolicies,omitempty"`

	// Pod security standards
	// +optional
	// +kubebuilder:validation:Enum=privileged;baseline;restricted
	PodSecurityStandard string `json:"podSecurityStandard,omitempty"`

	// RBAC requirements
	// +optional
	RBACEnabled bool `json:"rbacEnabled,omitempty"`
}

// HighAvailabilityIntent defines high availability requirements
type HighAvailabilityIntent struct {
	// Enable high availability
	Enabled bool `json:"enabled"`

	// Availability level (99.9%, 99.95%, 99.99%)
	// +optional
	// +kubebuilder:validation:Pattern=`^99\.(9|95|99)%$`
	AvailabilityLevel string `json:"availabilityLevel,omitempty"`

	// Multi-zone deployment preference
	// +optional
	MultiZone bool `json:"multiZone,omitempty"`

	// Anti-affinity requirements
	// +optional
	AntiAffinity bool `json:"antiAffinity,omitempty"`

	// Backup requirements
	// +optional
	BackupEnabled bool `json:"backupEnabled,omitempty"`

	// Disaster recovery requirements
	// +optional
	DisasterRecovery bool `json:"disasterRecovery,omitempty"`
}

// PerformanceIntent defines performance requirements from intent
type PerformanceIntent struct {
	// Latency requirements (in milliseconds)
	// +optional
	LatencyRequirement *int32 `json:"latencyRequirement,omitempty"`

	// Throughput requirements (requests per second)
	// +optional
	ThroughputRequirement *int32 `json:"throughputRequirement,omitempty"`

	// Bandwidth requirements
	// +optional
	BandwidthRequirement string `json:"bandwidthRequirement,omitempty"`

	// Packet loss tolerance
	// +optional
	PacketLossTolerance *float64 `json:"packetLossTolerance,omitempty"`

	// Jitter tolerance (in milliseconds)
	// +optional
	JitterTolerance *int32 `json:"jitterTolerance,omitempty"`

	// Performance tier
	// +optional
	// +kubebuilder:validation:Enum=basic;standard;premium;ultra
	Tier string `json:"tier,omitempty"`

	// QoS requirements
	// +optional
	QoSClass string `json:"qosClass,omitempty"`
}

// NetworkSlicingIntent defines network slicing requirements
type NetworkSlicingIntent struct {
	// Enable network slicing
	Enabled bool `json:"enabled"`

	// Slice type (eMBB, URLLC, mMTC)
	// +optional
	// +kubebuilder:validation:Enum=eMBB;URLLC;mMTC;custom
	SliceType string `json:"sliceType,omitempty"`

	// SST (Slice/Service Type)
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=255
	SST *int32 `json:"sst,omitempty"`

	// SD (Slice Differentiator)
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9A-Fa-f]{6}$`
	SD string `json:"sd,omitempty"`

	// Isolation level (logical, physical, complete)
	// +optional
	// +kubebuilder:validation:Enum=logical;physical;complete
	IsolationLevel string `json:"isolationLevel,omitempty"`

	// Dedicated resources
	// +optional
	DedicatedResources bool `json:"dedicatedResources,omitempty"`

	// SLA requirements
	// +optional
	SLARequirements map[string]string `json:"slaRequirements,omitempty"`
}

// CNFIntentProcessingResult represents the result of processing CNF-related intents
type CNFIntentProcessingResult struct {
	// Detected CNF functions from the intent
	DetectedFunctions []CNFFunction `json:"detectedFunctions,omitempty"`

	// Recommended deployment strategy
	RecommendedStrategy DeploymentStrategy `json:"recommendedStrategy,omitempty"`

	// Generated CNF deployment specifications
	CNFDeployments []CNFDeploymentIntent `json:"cnfDeployments,omitempty"`

	// Resource estimation
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	EstimatedResources runtime.RawExtension `json:"estimatedResources,omitempty"`

	// Cost estimation
	EstimatedCost float64 `json:"estimatedCost,omitempty"`

	// Deployment timeline estimation (in minutes)
	EstimatedDeploymentTime int32 `json:"estimatedDeploymentTime,omitempty"`

	// Confidence score (0.0 - 1.0)
	ConfidenceScore float64 `json:"confidenceScore,omitempty"`

	// Processing warnings
	Warnings []string `json:"warnings,omitempty"`

	// Processing errors
	Errors []string `json:"errors,omitempty"`

	// Additional context from LLM processing
	// +kubebuilder:pruning:PreserveUnknownFields
	LLMContext runtime.RawExtension `json:"llmContext,omitempty"`
}

// GetObjectKind implements runtime.Object interface
func (c *CNFIntentProcessingResult) GetObjectKind() schema.ObjectKind {
	return c
}

// GetObjectKind implements schema.ObjectKind interface
func (c *CNFIntentProcessingResult) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	// This is a data structure, not a Kubernetes resource, so no-op is appropriate
}

// GroupVersionKind implements schema.ObjectKind interface
func (c *CNFIntentProcessingResult) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{}
}

// DeepCopyObject implements runtime.Object interface
func (c *CNFIntentProcessingResult) DeepCopyObject() runtime.Object {
	if c == nil {
		return nil
	}

	out := new(CNFIntentProcessingResult)
	c.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out
func (c *CNFIntentProcessingResult) DeepCopyInto(out *CNFIntentProcessingResult) {
	*out = *c

	if c.DetectedFunctions != nil {
		in, out := &c.DetectedFunctions, &out.DetectedFunctions
		*out = make([]CNFFunction, len(*in))
		copy(*out, *in)
	}

	if c.CNFDeployments != nil {
		in, out := &c.CNFDeployments, &out.CNFDeployments
		*out = make([]CNFDeploymentIntent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}

	if c.EstimatedResources.Raw != nil {
		c.EstimatedResources.DeepCopyInto(&out.EstimatedResources)
	}

	if c.Warnings != nil {
		in, out := &c.Warnings, &out.Warnings
		*out = make([]string, len(*in))
		copy(*out, *in)
	}

	if c.Errors != nil {
		in, out := &c.Errors, &out.Errors
		*out = make([]string, len(*in))
		copy(*out, *in)
	}

	if c.LLMContext.Raw != nil {
		c.LLMContext.DeepCopyInto(&out.LLMContext)
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out
func (c *CNFDeploymentIntent) DeepCopyInto(out *CNFDeploymentIntent) {
	*out = *c

	if c.Replicas != nil {
		in, out := &c.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}

	if c.Resources != nil {
		in, out := &c.Resources, &out.Resources
		*out = new(CNFResourceIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.AutoScaling != nil {
		in, out := &c.AutoScaling, &out.AutoScaling
		*out = new(AutoScalingIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.ServiceMesh != nil {
		in, out := &c.ServiceMesh, &out.ServiceMesh
		*out = new(ServiceMeshIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.Monitoring != nil {
		in, out := &c.Monitoring, &out.Monitoring
		*out = new(MonitoringIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.Security != nil {
		in, out := &c.Security, &out.Security
		*out = new(SecurityIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.HighAvailability != nil {
		in, out := &c.HighAvailability, &out.HighAvailability
		*out = new(HighAvailabilityIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.Performance != nil {
		in, out := &c.Performance, &out.Performance
		*out = new(PerformanceIntent)
		(*in).DeepCopyInto(*out)
	}

	if c.NetworkSlicing != nil {
		in, out := &c.NetworkSlicing, &out.NetworkSlicing
		*out = new(NetworkSlicingIntent)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto methods for supporting types
func (c *CNFResourceIntent) DeepCopyInto(out *CNFResourceIntent) {
	*out = *c
	if c.CPU != nil {
		in, out := &c.CPU, &out.CPU
		x := (*in).DeepCopy()
		*out = &x
	}
	if c.Memory != nil {
		in, out := &c.Memory, &out.Memory
		x := (*in).DeepCopy()
		*out = &x
	}
	if c.Storage != nil {
		in, out := &c.Storage, &out.Storage
		x := (*in).DeepCopy()
		*out = &x
	}
	if c.GPU != nil {
		in, out := &c.GPU, &out.GPU
		*out = new(int32)
		**out = **in
	}
	if c.DPDK != nil {
		in, out := &c.DPDK, &out.DPDK
		*out = new(DPDKIntent)
		(*in).DeepCopyInto(*out)
	}
	if c.Hugepages != nil {
		in, out := &c.Hugepages, &out.Hugepages
		*out = make(map[string]resource.Quantity, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
}

func (c *DPDKIntent) DeepCopyInto(out *DPDKIntent) {
	*out = *c
	if c.Cores != nil {
		in, out := &c.Cores, &out.Cores
		*out = new(int32)
		**out = **in
	}
	if c.Memory != nil {
		in, out := &c.Memory, &out.Memory
		*out = new(int32)
		**out = **in
	}
}

func (c *AutoScalingIntent) DeepCopyInto(out *AutoScalingIntent) {
	*out = *c
	if c.MinReplicas != nil {
		in, out := &c.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if c.MaxReplicas != nil {
		in, out := &c.MaxReplicas, &out.MaxReplicas
		*out = new(int32)
		**out = **in
	}
	if c.TargetCPUUtilization != nil {
		in, out := &c.TargetCPUUtilization, &out.TargetCPUUtilization
		*out = new(int32)
		**out = **in
	}
	if c.TargetMemoryUtilization != nil {
		in, out := &c.TargetMemoryUtilization, &out.TargetMemoryUtilization
		*out = new(int32)
		**out = **in
	}
	if c.CustomMetrics != nil {
		in, out := &c.CustomMetrics, &out.CustomMetrics
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

func (c *ServiceMeshIntent) DeepCopyInto(out *ServiceMeshIntent) {
	*out = *c
	if c.MTLS != nil {
		in, out := &c.MTLS, &out.MTLS
		*out = new(MTLSIntent)
		(*in).DeepCopyInto(*out)
	}
	if c.TrafficManagement != nil {
		in, out := &c.TrafficManagement, &out.TrafficManagement
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

func (c *MTLSIntent) DeepCopyInto(out *MTLSIntent) {
	*out = *c
}

func (c *MonitoringIntent) DeepCopyInto(out *MonitoringIntent) {
	*out = *c
	if c.Metrics != nil {
		in, out := &c.Metrics, &out.Metrics
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if c.Alerts != nil {
		in, out := &c.Alerts, &out.Alerts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if c.Dashboards != nil {
		in, out := &c.Dashboards, &out.Dashboards
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

func (c *SecurityIntent) DeepCopyInto(out *SecurityIntent) {
	*out = *c
	if c.Encryption != nil {
		in, out := &c.Encryption, &out.Encryption
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if c.Authentication != nil {
		in, out := &c.Authentication, &out.Authentication
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if c.NetworkPolicies != nil {
		in, out := &c.NetworkPolicies, &out.NetworkPolicies
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

func (c *HighAvailabilityIntent) DeepCopyInto(out *HighAvailabilityIntent) {
	*out = *c
}

func (c *PerformanceIntent) DeepCopyInto(out *PerformanceIntent) {
	*out = *c
	if c.LatencyRequirement != nil {
		in, out := &c.LatencyRequirement, &out.LatencyRequirement
		*out = new(int32)
		**out = **in
	}
	if c.ThroughputRequirement != nil {
		in, out := &c.ThroughputRequirement, &out.ThroughputRequirement
		*out = new(int32)
		**out = **in
	}
	if c.PacketLossTolerance != nil {
		in, out := &c.PacketLossTolerance, &out.PacketLossTolerance
		*out = new(float64)
		**out = **in
	}
	if c.JitterTolerance != nil {
		in, out := &c.JitterTolerance, &out.JitterTolerance
		*out = new(int32)
		**out = **in
	}
}

func (c *NetworkSlicingIntent) DeepCopyInto(out *NetworkSlicingIntent) {
	*out = *c
	if c.SST != nil {
		in, out := &c.SST, &out.SST
		*out = new(int32)
		**out = **in
	}
	if c.SLARequirements != nil {
		in, out := &c.SLARequirements, &out.SLARequirements
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// CNFTopologyIntent defines network topology and connectivity requirements
type CNFTopologyIntent struct {
	// Network function connectivity requirements
	Connectivity []CNFConnectivityRequirement `json:"connectivity,omitempty"`

	// Service function chaining requirements
	ServiceChaining []ServiceChainRequirement `json:"serviceChaining,omitempty"`

	// Load balancing requirements
	LoadBalancing *LoadBalancingRequirement `json:"loadBalancing,omitempty"`

	// Multi-cluster deployment requirements
	MultiCluster *MultiClusterRequirement `json:"multiCluster,omitempty"`

	// Edge deployment preferences
	EdgeDeployment *EdgeDeploymentRequirement `json:"edgeDeployment,omitempty"`
}

// CNFConnectivityRequirement defines connectivity between CNFs
type CNFConnectivityRequirement struct {
	// Source CNF function
	Source CNFFunction `json:"source"`

	// Destination CNF function
	Destination CNFFunction `json:"destination"`

	// Interface type (N1, N2, N3, N4, SBI, etc.)
	InterfaceType string `json:"interfaceType"`

	// Protocol requirements
	Protocol []string `json:"protocol"`

	// Bandwidth requirements
	Bandwidth string `json:"bandwidth,omitempty"`

	// Latency requirements
	Latency *int32 `json:"latency,omitempty"`

	// Security requirements
	Security []string `json:"security,omitempty"`
}

// ServiceChainRequirement defines service function chaining
type ServiceChainRequirement struct {
	// Chain name
	Name string `json:"name"`

	// Ordered list of CNF functions in the chain
	Functions []CNFFunction `json:"functions"`

	// Traffic selection criteria
	TrafficSelector map[string]string `json:"trafficSelector,omitempty"`

	// Chain performance requirements
	Performance *ChainPerformanceRequirement `json:"performance,omitempty"`
}

// ChainPerformanceRequirement defines performance for service chains
type ChainPerformanceRequirement struct {
	// End-to-end latency requirement
	EndToEndLatency *int32 `json:"endToEndLatency,omitempty"`

	// Throughput requirement
	Throughput *int32 `json:"throughput,omitempty"`

	// Availability requirement
	Availability string `json:"availability,omitempty"`
}

// LoadBalancingRequirement defines load balancing requirements
type LoadBalancingRequirement struct {
	// Enable load balancing
	Enabled bool `json:"enabled"`

	// Load balancing algorithm
	Algorithm string `json:"algorithm,omitempty"`

	// Session affinity
	SessionAffinity bool `json:"sessionAffinity,omitempty"`

	// Health check configuration
	HealthCheck *HealthCheckRequirement `json:"healthCheck,omitempty"`
}

// HealthCheckRequirement defines health check requirements
type HealthCheckRequirement struct {
	// Health check type (HTTP, TCP, gRPC)
	Type string `json:"type"`

	// Check interval
	Interval *int32 `json:"interval,omitempty"`

	// Timeout
	Timeout *int32 `json:"timeout,omitempty"`

	// Retry count
	Retries *int32 `json:"retries,omitempty"`
}

// MultiClusterRequirement defines multi-cluster deployment requirements
type MultiClusterRequirement struct {
	// Enable multi-cluster deployment
	Enabled bool `json:"enabled"`

	// Target clusters
	TargetClusters []string `json:"targetClusters,omitempty"`

	// Cluster selection strategy (round-robin, resource-based, geo-based)
	SelectionStrategy string `json:"selectionStrategy,omitempty"`

	// Inter-cluster communication requirements
	InterClusterCommunication bool `json:"interClusterCommunication,omitempty"`

	// Failover requirements
	Failover bool `json:"failover,omitempty"`
}

// EdgeDeploymentRequirement defines edge deployment requirements
type EdgeDeploymentRequirement struct {
	// Enable edge deployment
	Enabled bool `json:"enabled"`

	// Edge locations
	EdgeLocations []string `json:"edgeLocations,omitempty"`

	// Latency requirements for edge placement
	LatencyRequirement *int32 `json:"latencyRequirement,omitempty"`

	// Resource constraints for edge deployment
	ResourceConstraints *CNFResourceIntent `json:"resourceConstraints,omitempty"`

	// Connectivity to central functions
	CentralConnectivity bool `json:"centralConnectivity,omitempty"`
}
