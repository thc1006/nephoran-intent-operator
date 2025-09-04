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

package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// NetworkSliceConfigFunction implements network slice configuration for 5G networks.

type NetworkSliceConfigFunction struct {
	tracer trace.Tracer
}

// NetworkSliceConfig defines the configuration structure for network slicing.

type NetworkSliceConfig struct {
	// Slice identification.

	SliceID string `json:"sliceId" yaml:"sliceId"`

	SliceName string `json:"sliceName" yaml:"sliceName"`

	SliceType string `json:"sliceType" yaml:"sliceType"` // eMBB, URLLC, mMTC

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Service Level Agreement (SLA) parameters.

	SLA *SLAConfiguration `json:"sla" yaml:"sla"`

	// Quality of Service (QoS) parameters.

	QoS *QoSConfiguration `json:"qos,omitempty" yaml:"qos,omitempty"`

	// Resource allocation.

	Resources *SliceResourceAllocation `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Isolation requirements.

	Isolation *SliceIsolationConfig `json:"isolation,omitempty" yaml:"isolation,omitempty"`

	// Geographic and coverage requirements.

	Coverage *SliceCoverageConfig `json:"coverage,omitempty" yaml:"coverage,omitempty"`

	// Security requirements.

	Security *SliceSecurityConfig `json:"security,omitempty" yaml:"security,omitempty"`

	// Lifecycle management.

	Lifecycle *SliceLifecycleConfig `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`

	// Monitoring and observability.

	Monitoring *SliceMonitoringConfig `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`

	// Network function configurations.

	NetworkFunctions []*NetworkFunctionConfig `json:"networkFunctions,omitempty" yaml:"networkFunctions,omitempty"`
}

// SLAConfiguration defines service level agreement parameters.

type SLAConfiguration struct {
	// Latency requirements.

	Latency *LatencyRequirements `json:"latency,omitempty" yaml:"latency,omitempty"`

	// Throughput requirements.

	Throughput *ThroughputRequirements `json:"throughput,omitempty" yaml:"throughput,omitempty"`

	// Availability requirements.

	Availability *AvailabilityRequirements `json:"availability,omitempty" yaml:"availability,omitempty"`

	// Reliability requirements.

	Reliability *ReliabilityRequirements `json:"reliability,omitempty" yaml:"reliability,omitempty"`

	// Scalability requirements.

	Scalability *ScalabilityRequirements `json:"scalability,omitempty" yaml:"scalability,omitempty"`
}

// LatencyRequirements defines latency SLA parameters.

type LatencyRequirements struct {
	MaxLatency string `json:"maxLatency" yaml:"maxLatency"` // e.g., "1ms", "10ms"

	TypicalLatency string `json:"typicalLatency,omitempty" yaml:"typicalLatency,omitempty"`

	JitterTolerance string `json:"jitterTolerance,omitempty" yaml:"jitterTolerance,omitempty"`

	Percentile float64 `json:"percentile,omitempty" yaml:"percentile,omitempty"`
}

// ThroughputRequirements defines throughput SLA parameters.

type ThroughputRequirements struct {
	MinDownlink string `json:"minDownlink,omitempty" yaml:"minDownlink,omitempty"` // e.g., "100Mbps", "1Gbps"

	MinUplink string `json:"minUplink,omitempty" yaml:"minUplink,omitempty"`

	MaxDownlink string `json:"maxDownlink,omitempty" yaml:"maxDownlink,omitempty"`

	MaxUplink string `json:"maxUplink,omitempty" yaml:"maxUplink,omitempty"`

	UserDensity int32 `json:"userDensity,omitempty" yaml:"userDensity,omitempty"` // users per km²

	DeviceDensity int32 `json:"deviceDensity,omitempty" yaml:"deviceDensity,omitempty"` // devices per km²
}

// AvailabilityRequirements defines availability SLA parameters.

type AvailabilityRequirements struct {
	Target float64 `json:"target" yaml:"target"` // e.g., 0.999, 0.9999

	ServiceLevel string `json:"serviceLevel,omitempty" yaml:"serviceLevel,omitempty"`

	Downtime string `json:"downtime,omitempty" yaml:"downtime,omitempty"` // allowed downtime per month

	MTBF string `json:"mtbf,omitempty" yaml:"mtbf,omitempty"` // Mean Time Between Failures

	MTTR string `json:"mttr,omitempty" yaml:"mttr,omitempty"` // Mean Time To Repair
}

// ReliabilityRequirements defines reliability SLA parameters.

type ReliabilityRequirements struct {
	SuccessRate float64 `json:"successRate" yaml:"successRate"` // e.g., 0.999

	ErrorRate float64 `json:"errorRate,omitempty" yaml:"errorRate,omitempty"`

	PacketLoss float64 `json:"packetLoss,omitempty" yaml:"packetLoss,omitempty"` // maximum acceptable packet loss

	Redundancy string `json:"redundancy,omitempty" yaml:"redundancy,omitempty"` // redundancy level
}

// ScalabilityRequirements defines scalability SLA parameters.

type ScalabilityRequirements struct {
	MinUsers int32 `json:"minUsers,omitempty" yaml:"minUsers,omitempty"`

	MaxUsers int32 `json:"maxUsers,omitempty" yaml:"maxUsers,omitempty"`

	AutoScaling bool `json:"autoScaling,omitempty" yaml:"autoScaling,omitempty"`

	ScaleUpTime string `json:"scaleUpTime,omitempty" yaml:"scaleUpTime,omitempty"` // time to scale up

	ScaleDownTime string `json:"scaleDownTime,omitempty" yaml:"scaleDownTime,omitempty"` // time to scale down
}

// QoSConfiguration defines quality of service parameters.

type QoSConfiguration struct {
	FiveQI int32 `json:"5qi,omitempty" yaml:"5qi,omitempty"`

	PriorityLevel int32 `json:"priorityLevel,omitempty" yaml:"priorityLevel,omitempty"`

	PacketDelayBudget string `json:"packetDelayBudget,omitempty" yaml:"packetDelayBudget,omitempty"`

	PacketErrorRate string `json:"packetErrorRate,omitempty" yaml:"packetErrorRate,omitempty"`

	MaxDataBurst string `json:"maxDataBurst,omitempty" yaml:"maxDataBurst,omitempty"`

	GFBR string `json:"gfbr,omitempty" yaml:"gfbr,omitempty"` // Guaranteed Flow Bit Rate

	MFBR string `json:"mfbr,omitempty" yaml:"mfbr,omitempty"` // Maximum Flow Bit Rate

	QoSClass string `json:"qosClass,omitempty" yaml:"qosClass,omitempty"`
}

// SliceResourceAllocation defines resource allocation for the slice.

type SliceResourceAllocation struct {
	// Compute resources.

	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"` // e.g., "2000m", "4"

	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"` // e.g., "4Gi", "8Gi"

	Storage string `json:"storage,omitempty" yaml:"storage,omitempty"` // e.g., "10Gi", "100Gi"

	// Network resources.

	Bandwidth string `json:"bandwidth,omitempty" yaml:"bandwidth,omitempty"` // e.g., "1Gbps", "10Gbps"

	Connections int32 `json:"connections,omitempty" yaml:"connections,omitempty"`

	// Radio resources.

	SpectrumAllocation *SpectrumAllocation `json:"spectrumAllocation,omitempty" yaml:"spectrumAllocation,omitempty"`

	// Edge resources.

	EdgeNodes []string `json:"edgeNodes,omitempty" yaml:"edgeNodes,omitempty"`

	EdgeZones []string `json:"edgeZones,omitempty" yaml:"edgeZones,omitempty"`

	// Resource pools.

	ResourcePools []ResourcePoolAllocation `json:"resourcePools,omitempty" yaml:"resourcePools,omitempty"`
}

// SpectrumAllocation defines radio spectrum allocation.

type SpectrumAllocation struct {
	FrequencyBands []FrequencyBand `json:"frequencyBands,omitempty" yaml:"frequencyBands,omitempty"`

	Bandwidth string `json:"bandwidth,omitempty" yaml:"bandwidth,omitempty"`

	TxPower string `json:"txPower,omitempty" yaml:"txPower,omitempty"`

	AntennaConfig string `json:"antennaConfig,omitempty" yaml:"antennaConfig,omitempty"`
}

// FrequencyBand defines frequency band allocation.

type FrequencyBand struct {
	Band string `json:"band" yaml:"band"` // e.g., "n78", "n1"

	StartFreq string `json:"startFreq,omitempty" yaml:"startFreq,omitempty"`

	EndFreq string `json:"endFreq,omitempty" yaml:"endFreq,omitempty"`

	Bandwidth string `json:"bandwidth,omitempty" yaml:"bandwidth,omitempty"`

	Technology string `json:"technology,omitempty" yaml:"technology,omitempty"` // 4G, 5G
}

// ResourcePoolAllocation defines allocation from resource pools.

type ResourcePoolAllocation struct {
	PoolName string `json:"poolName" yaml:"poolName"`

	PoolType string `json:"poolType" yaml:"poolType"` // compute, network, storage

	Allocation string `json:"allocation" yaml:"allocation"` // amount or percentage

	Priority int32 `json:"priority,omitempty" yaml:"priority,omitempty"`

	Constraints map[string]string `json:"constraints,omitempty" yaml:"constraints,omitempty"`
}

// SliceIsolationConfig defines isolation requirements.

type SliceIsolationConfig struct {
	Level string `json:"level" yaml:"level"` // physical, logical, none

	Mechanisms []string `json:"mechanisms,omitempty" yaml:"mechanisms,omitempty"`

	Encryption bool `json:"encryption,omitempty" yaml:"encryption,omitempty"`

	VLANs []string `json:"vlans,omitempty" yaml:"vlans,omitempty"`

	VxLANs []string `json:"vxlans,omitempty" yaml:"vxlans,omitempty"`

	NetworkPolicies []string `json:"networkPolicies,omitempty" yaml:"networkPolicies,omitempty"`

	ResourceQuotas []string `json:"resourceQuotas,omitempty" yaml:"resourceQuotas,omitempty"`
}

// SliceCoverageConfig defines coverage requirements.

type SliceCoverageConfig struct {
	GeographicAreas []GeographicArea `json:"geographicAreas,omitempty" yaml:"geographicAreas,omitempty"`

	CoverageType string `json:"coverageType,omitempty" yaml:"coverageType,omitempty"` // indoor, outdoor, both

	Mobility *MobilityConfig `json:"mobility,omitempty" yaml:"mobility,omitempty"`

	Handover *HandoverConfig `json:"handover,omitempty" yaml:"handover,omitempty"`
}

// GeographicArea defines a geographic coverage area.

type GeographicArea struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"` // country, region, city, custom

	Coordinates []Coordinate `json:"coordinates,omitempty" yaml:"coordinates,omitempty"`

	Radius string `json:"radius,omitempty" yaml:"radius,omitempty"`

	Priority int32 `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// Coordinate defines a geographic coordinate.

type Coordinate struct {
	Latitude float64 `json:"latitude" yaml:"latitude"`

	Longitude float64 `json:"longitude" yaml:"longitude"`

	Altitude float64 `json:"altitude,omitempty" yaml:"altitude,omitempty"`
}

// MobilityConfig defines mobility requirements.

type MobilityConfig struct {
	MobilityLevel string `json:"mobilityLevel" yaml:"mobilityLevel"` // stationary, pedestrian, vehicular, high-speed

	MaxSpeed string `json:"maxSpeed,omitempty" yaml:"maxSpeed,omitempty"`

	HandoverTime string `json:"handoverTime,omitempty" yaml:"handoverTime,omitempty"`

	SupportedMobility []string `json:"supportedMobility,omitempty" yaml:"supportedMobility,omitempty"`
}

// HandoverConfig defines handover requirements.

type HandoverConfig struct {
	HandoverType string `json:"handoverType,omitempty" yaml:"handoverType,omitempty"` // intra-cell, inter-cell, inter-system

	MaxHandoverTime string `json:"maxHandoverTime,omitempty" yaml:"maxHandoverTime,omitempty"`

	MaxPacketLoss string `json:"maxPacketLoss,omitempty" yaml:"maxPacketLoss,omitempty"`

	HandoverTriggers []string `json:"handoverTriggers,omitempty" yaml:"handoverTriggers,omitempty"`
}

// SliceSecurityConfig defines security requirements.

type SliceSecurityConfig struct {
	EncryptionLevel string `json:"encryptionLevel,omitempty" yaml:"encryptionLevel,omitempty"`

	AuthMechanisms []string `json:"authMechanisms,omitempty" yaml:"authMechanisms,omitempty"`

	AccessControl *AccessControlConfig `json:"accessControl,omitempty" yaml:"accessControl,omitempty"`

	IntegrityChecks bool `json:"integrityChecks,omitempty" yaml:"integrityChecks,omitempty"`

	KeyManagement *KeyManagementConfig `json:"keyManagement,omitempty" yaml:"keyManagement,omitempty"`

	ThreatDetection *ThreatDetectionConfig `json:"threatDetection,omitempty" yaml:"threatDetection,omitempty"`
}

// AccessControlConfig defines access control requirements.

type AccessControlConfig struct {
	Model string `json:"model,omitempty" yaml:"model,omitempty"` // RBAC, ABAC, MAC

	Policies []string `json:"policies,omitempty" yaml:"policies,omitempty"`

	Whitelist []string `json:"whitelist,omitempty" yaml:"whitelist,omitempty"`

	Blacklist []string `json:"blacklist,omitempty" yaml:"blacklist,omitempty"`

	MFA bool `json:"mfa,omitempty" yaml:"mfa,omitempty"`
}

// KeyManagementConfig defines key management requirements.

type KeyManagementConfig struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	RotationPeriod string `json:"rotationPeriod,omitempty" yaml:"rotationPeriod,omitempty"`

	KeySize int32 `json:"keySize,omitempty" yaml:"keySize,omitempty"`

	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
}

// ThreatDetectionConfig defines threat detection requirements.

type ThreatDetectionConfig struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	DetectionTypes []string `json:"detectionTypes,omitempty" yaml:"detectionTypes,omitempty"`

	ResponseActions []string `json:"responseActions,omitempty" yaml:"responseActions,omitempty"`

	AlertingLevel string `json:"alertingLevel,omitempty" yaml:"alertingLevel,omitempty"`
}

// SliceLifecycleConfig defines lifecycle management.

type SliceLifecycleConfig struct {
	AutoProvisioning bool `json:"autoProvisioning,omitempty" yaml:"autoProvisioning,omitempty"`

	ProvisioningTime string `json:"provisioningTime,omitempty" yaml:"provisioningTime,omitempty"`

	DecommissionTime string `json:"decommissionTime,omitempty" yaml:"decommissionTime,omitempty"`

	BackupStrategy *BackupStrategy `json:"backupStrategy,omitempty" yaml:"backupStrategy,omitempty"`

	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty" yaml:"updateStrategy,omitempty"`
}

// BackupStrategy defines backup requirements.

type BackupStrategy struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	BackupInterval string `json:"backupInterval,omitempty" yaml:"backupInterval,omitempty"`

	RetentionPeriod string `json:"retentionPeriod,omitempty" yaml:"retentionPeriod,omitempty"`

	BackupLocation string `json:"backupLocation,omitempty" yaml:"backupLocation,omitempty"`
}

// UpdateStrategy defines update strategy.

type UpdateStrategy struct {
	Strategy string `json:"strategy,omitempty" yaml:"strategy,omitempty"` // rolling, blue-green, canary

	MaxUnavailable string `json:"maxUnavailable,omitempty" yaml:"maxUnavailable,omitempty"`

	MaxSurge string `json:"maxSurge,omitempty" yaml:"maxSurge,omitempty"`

	RollbackTimeout string `json:"rollbackTimeout,omitempty" yaml:"rollbackTimeout,omitempty"`
}

// SliceMonitoringConfig defines monitoring requirements.

type SliceMonitoringConfig struct {
	KPIs []KPIConfig `json:"kpis,omitempty" yaml:"kpis,omitempty"`

	Metrics []MetricConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	Alerting *AlertingConfig `json:"alerting,omitempty" yaml:"alerting,omitempty"`

	Dashboards []string `json:"dashboards,omitempty" yaml:"dashboards,omitempty"`

	LoggingLevel string `json:"loggingLevel,omitempty" yaml:"loggingLevel,omitempty"`

	TracingEnabled bool `json:"tracingEnabled,omitempty" yaml:"tracingEnabled,omitempty"`
}

// KPIConfig defines KPI monitoring.

type KPIConfig struct {
	Name string `json:"name" yaml:"name"`

	Threshold float64 `json:"threshold,omitempty" yaml:"threshold,omitempty"`

	Unit string `json:"unit,omitempty" yaml:"unit,omitempty"`

	Aggregation string `json:"aggregation,omitempty" yaml:"aggregation,omitempty"`
}

// MetricConfig defines custom metrics.

type MetricConfig struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"`

	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// AlertingConfig defines alerting configuration.

type AlertingConfig struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	AlertRules []AlertRule `json:"alertRules,omitempty" yaml:"alertRules,omitempty"`

	NotificationChannels []NotificationChannel `json:"notificationChannels,omitempty" yaml:"notificationChannels,omitempty"`
}

// AlertRule defines an alerting rule.

type AlertRule struct {
	Name string `json:"name" yaml:"name"`

	Condition string `json:"condition" yaml:"condition"`

	Severity string `json:"severity,omitempty" yaml:"severity,omitempty"`

	Duration string `json:"duration,omitempty" yaml:"duration,omitempty"`

	Action string `json:"action,omitempty" yaml:"action,omitempty"`
}

// NotificationChannel defines notification channel.

type NotificationChannel struct {
	Type string `json:"type" yaml:"type"`

	Target string `json:"target" yaml:"target"`

	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`

	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

// NetworkFunctionConfig defines network function configuration.

type NetworkFunctionConfig struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"` // AMF, SMF, UPF, PCF, UDM, etc.

	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

	Resources *ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`

	Replicas int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	Affinity *AffinityConfig `json:"affinity,omitempty" yaml:"affinity,omitempty"`
}

// ResourceRequirements defines resource requirements for network functions.

type ResourceRequirements struct {
	Requests map[string]string `json:"requests,omitempty" yaml:"requests,omitempty"`

	Limits map[string]string `json:"limits,omitempty" yaml:"limits,omitempty"`
}

// AffinityConfig defines affinity configuration.

type AffinityConfig struct {
	NodeAffinity *NodeAffinityConfig `json:"nodeAffinity,omitempty" yaml:"nodeAffinity,omitempty"`

	PodAffinity *PodAffinityConfig `json:"podAffinity,omitempty" yaml:"podAffinity,omitempty"`

	PodAntiAffinity *PodAntiAffinityConfig `json:"podAntiAffinity,omitempty" yaml:"podAntiAffinity,omitempty"`
}

// NodeAffinityConfig defines node affinity.

type NodeAffinityConfig struct {
	RequiredDuringSchedulingIgnoredDuringExecution []NodeSelectorTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelectorTerm defines node selector terms.

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty" yaml:"matchExpressions,omitempty"`

	MatchFields []NodeSelectorRequirement `json:"matchFields,omitempty" yaml:"matchFields,omitempty"`
}

// NodeSelectorRequirement defines node selector requirements.

type NodeSelectorRequirement struct {
	Key string `json:"key" yaml:"key"`

	Operator string `json:"operator" yaml:"operator"`

	Values []string `json:"values,omitempty" yaml:"values,omitempty"`
}

// PreferredSchedulingTerm defines preferred scheduling terms.

type PreferredSchedulingTerm struct {
	Weight int32 `json:"weight" yaml:"weight"`

	Preference NodeSelectorTerm `json:"preference" yaml:"preference"`
}

// PodAffinityConfig defines pod affinity.

type PodAffinityConfig struct {
	RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAntiAffinityConfig defines pod anti-affinity.

type PodAntiAffinityConfig struct {
	RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinityTerm defines pod affinity terms.

type PodAffinityTerm struct {
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`

	Namespaces []string `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`

	TopologyKey string `json:"topologyKey" yaml:"topologyKey"`
}

// WeightedPodAffinityTerm defines weighted pod affinity terms.

type WeightedPodAffinityTerm struct {
	Weight int32 `json:"weight" yaml:"weight"`

	PodAffinityTerm PodAffinityTerm `json:"podAffinityTerm" yaml:"podAffinityTerm"`
}

// NewNetworkSliceConfigFunction creates a new network slice configuration function.

func NewNetworkSliceConfigFunction() *NetworkSliceConfigFunction {
	return &NetworkSliceConfigFunction{
		tracer: otel.Tracer("network-slice-config-function"),
	}
}

// Execute implements the KRM function for network slice configuration.

func (f *NetworkSliceConfigFunction) Execute(ctx context.Context, resources []porch.KRMResource, config map[string]interface{}) ([]porch.KRMResource, []*porch.FunctionResult, error) {
	ctx, span := f.tracer.Start(ctx, "network-slice-config-execute")

	defer span.End()

	logger := log.FromContext(ctx).WithName("network-slice-config")

	span.SetAttributes(

		attribute.Int("input.resources", len(resources)),
	)

	// Parse configuration.

	sliceConfig, err := f.parseConfig(config)
	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "config parsing failed")

		return nil, nil, fmt.Errorf("failed to parse network slice configuration: %w", err)

	}

	span.SetAttributes(

		attribute.String("slice.id", sliceConfig.SliceID),

		attribute.String("slice.type", sliceConfig.SliceType),
	)

	logger.Info("Configuring network slice",

		"sliceId", sliceConfig.SliceID,

		"sliceType", sliceConfig.SliceType,

		"resources", len(resources),
	)

	// Process resources.

	processedResources, results, err := f.processResources(ctx, resources, sliceConfig)
	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "resource processing failed")

		return nil, results, fmt.Errorf("failed to process network slice resources: %w", err)

	}

	// Generate additional resources if needed.

	additionalResources, additionalResults, err := f.generateAdditionalResources(ctx, sliceConfig)

	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "additional resource generation failed")

		logger.Error(err, "Failed to generate additional resources")

		// Continue with what we have.

	} else {

		processedResources = append(processedResources, additionalResources...)

		results = append(results, additionalResults...)

	}

	span.SetAttributes(

		attribute.Int("output.resources", len(processedResources)),

		attribute.Int("output.results", len(results)),
	)

	span.SetStatus(codes.Ok, "network slice configuration completed")

	logger.Info("Network slice configuration completed successfully",

		"sliceId", sliceConfig.SliceID,

		"processedResources", len(processedResources),
	)

	return processedResources, results, nil
}

// parseConfig parses the function configuration into NetworkSliceConfig.

func (f *NetworkSliceConfigFunction) parseConfig(config map[string]interface{}) (*NetworkSliceConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Convert to JSON and back to struct for type safety.

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var sliceConfig NetworkSliceConfig

	if err := json.Unmarshal(configJSON, &sliceConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields.

	if sliceConfig.SliceID == "" {
		return nil, fmt.Errorf("sliceId is required")
	}

	if sliceConfig.SliceType == "" {
		return nil, fmt.Errorf("sliceType is required")
	}

	// Validate slice type.

	validSliceTypes := []string{"eMBB", "URLLC", "mMTC", "V2X", "IoT"}

	validType := false

	for _, validSliceType := range validSliceTypes {
		if sliceConfig.SliceType == validSliceType {

			validType = true

			break

		}
	}

	if !validType {
		return nil, fmt.Errorf("invalid sliceType: %s, must be one of: %v", sliceConfig.SliceType, validSliceTypes)
	}

	return &sliceConfig, nil
}

// processResources processes existing resources based on slice configuration.

func (f *NetworkSliceConfigFunction) processResources(ctx context.Context, resources []porch.KRMResource, config *NetworkSliceConfig) ([]porch.KRMResource, []*porch.FunctionResult, error) {
	var processedResources []porch.KRMResource

	var results []*porch.FunctionResult

	for i, resource := range resources {
		// Process based on resource type.

		switch {

		case resource.Kind == "NetworkIntent":

			processed, result := f.processNetworkIntent(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {
				results = append(results, result)
			}

		case resource.Kind == "Deployment":

			processed, result := f.processDeployment(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {
				results = append(results, result)
			}

		case resource.Kind == "Service":

			processed, result := f.processService(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {
				results = append(results, result)
			}

		case resource.Kind == "ConfigMap":

			processed, result := f.processConfigMap(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {
				results = append(results, result)
			}

		default:

			// Pass through unmodified.

			processedResources = append(processedResources, resource)

			results = append(results, &porch.FunctionResult{
				Message: fmt.Sprintf("Resource %d passed through unchanged", i),

				Severity: "info",
			})

		}
	}

	return processedResources, results, nil
}

// processNetworkIntent configures NetworkIntent resources for network slicing.

func (f *NetworkSliceConfigFunction) processNetworkIntent(resource porch.KRMResource, config *NetworkSliceConfig) (porch.KRMResource, *porch.FunctionResult) {
	// Add network slice configuration to NetworkIntent.

	var spec map[string]interface{}
	if resource.Spec != nil {
		if err := json.Unmarshal(resource.Spec, &spec); err != nil {
			spec = make(map[string]interface{})
		}
	} else {
		spec = make(map[string]interface{})
	}

	// Add network slice specification.
	spec["networkSlice"] = map[string]interface{}{}

	// Marshal back to JSON
	marshaled, err := json.Marshal(spec)
	if err != nil {
		return resource, &porch.FunctionResult{
			Severity: string(porch.ValidationSeverityError),
			Message:  fmt.Sprintf("Failed to marshal spec: %v", err),
		}
	}
	resource.Spec = json.RawMessage(marshaled)

	// Add labels for slice identification.
	var metadata map[string]interface{}
	if resource.Metadata != nil {
		if err := json.Unmarshal(resource.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	if metadata["labels"] == nil {
		metadata["labels"] = make(map[string]interface{})
	}

	labels := metadata["labels"].(map[string]interface{})
	labels["nephoran.com/network-slice-id"] = config.SliceID
	labels["nephoran.com/network-slice-type"] = config.SliceType

	// Marshal back to JSON
	metadataMarshaled, err := json.Marshal(metadata)
	if err != nil {
		return resource, &porch.FunctionResult{
			Severity: string(porch.ValidationSeverityError),
			Message:  fmt.Sprintf("Failed to marshal metadata: %v", err),
		}
	}
	resource.Metadata = json.RawMessage(metadataMarshaled)

	return resource, &porch.FunctionResult{
		Message: fmt.Sprintf("Configured NetworkIntent for slice %s", config.SliceID),

		Severity: "info",
	}
}

// processDeployment configures Deployment resources for network slicing.

func (f *NetworkSliceConfigFunction) processDeployment(resource porch.KRMResource, config *NetworkSliceConfig) (porch.KRMResource, *porch.FunctionResult) {
	// Add resource requirements based on slice configuration.

	if config.Resources != nil {
		f.applyResourceRequirements(&resource, config.Resources)
	}

	// Add slice-specific labels and annotations.

	f.addSliceLabelsAndAnnotations(&resource, config)

	// Configure network functions if specified.

	if len(config.NetworkFunctions) > 0 {
		f.configureNetworkFunctions(&resource, config.NetworkFunctions)
	}

	return resource, &porch.FunctionResult{
		Message: fmt.Sprintf("Configured Deployment for slice %s", config.SliceID),

		Severity: "info",
	}
}

// processService configures Service resources for network slicing.

func (f *NetworkSliceConfigFunction) processService(resource porch.KRMResource, config *NetworkSliceConfig) (porch.KRMResource, *porch.FunctionResult) {
	// Add slice-specific labels and annotations.

	f.addSliceLabelsAndAnnotations(&resource, config)

	// Configure QoS annotations if specified.

	if config.QoS != nil {
		f.addQoSAnnotations(&resource, config.QoS)
	}

	return resource, &porch.FunctionResult{
		Message: fmt.Sprintf("Configured Service for slice %s", config.SliceID),

		Severity: "info",
	}
}

// processConfigMap configures ConfigMap resources for network slicing.

func (f *NetworkSliceConfigFunction) processConfigMap(resource porch.KRMResource, config *NetworkSliceConfig) (porch.KRMResource, *porch.FunctionResult) {
	// Add slice configuration to ConfigMap data.
	var data map[string]interface{}
	if resource.Data != nil {
		if err := json.Unmarshal(resource.Data, &data); err != nil {
			data = make(map[string]interface{})
		}
	} else {
		data = make(map[string]interface{})
	}

	// Add slice configuration.
	sliceConfigData, _ := json.Marshal(config)
	data["slice-config.json"] = string(sliceConfigData)

	// Add specific configuration based on slice type.
	switch config.SliceType {
	case "eMBB":
		data["slice-type"] = "enhanced-mobile-broadband"
		data["bandwidth-priority"] = "high"

	case "URLLC":
		data["slice-type"] = "ultra-reliable-low-latency"
		data["latency-priority"] = "ultra-low"

	case "mMTC":
		data["slice-type"] = "massive-machine-type-communications"
		data["density-optimization"] = "high"

	}

	// Add SLA configuration if present.
	if config.SLA != nil {
		slaData, _ := json.Marshal(config.SLA)
		data["sla-config.json"] = string(slaData)
	}

	// Marshal back to JSON
	marshaled, err := json.Marshal(data)
	if err != nil {
		return resource, &porch.FunctionResult{
			Severity: string(porch.ValidationSeverityError),
			Message:  fmt.Sprintf("Failed to marshal data: %v", err),
		}
	}
	resource.Data = json.RawMessage(marshaled)

	return resource, &porch.FunctionResult{
		Message:  fmt.Sprintf("Configured ConfigMap for slice %s", config.SliceID),
		Severity: string(porch.ValidationSeverityInfo),
	}
}

// generateAdditionalResources generates additional resources needed for network slicing.

func (f *NetworkSliceConfigFunction) generateAdditionalResources(ctx context.Context, config *NetworkSliceConfig) ([]porch.KRMResource, []*porch.FunctionResult, error) {
	var resources []porch.KRMResource

	var results []*porch.FunctionResult

	// Generate ResourceQuota for isolation.

	if config.Isolation != nil && config.Resources != nil {

		quota := f.generateResourceQuota(config)

		resources = append(resources, quota)

		results = append(results, &porch.FunctionResult{
			Message: fmt.Sprintf("Generated ResourceQuota for slice %s", config.SliceID),

			Severity: "info",
		})

	}

	// Generate NetworkPolicy for network isolation.

	if config.Isolation != nil && len(config.Isolation.NetworkPolicies) > 0 {

		netPol := f.generateNetworkPolicy(config)

		resources = append(resources, netPol)

		results = append(results, &porch.FunctionResult{
			Message: fmt.Sprintf("Generated NetworkPolicy for slice %s", config.SliceID),

			Severity: "info",
		})

	}

	// Generate ServiceMonitor for monitoring.

	if config.Monitoring != nil && len(config.Monitoring.KPIs) > 0 {

		monitor := f.generateServiceMonitor(config)

		resources = append(resources, monitor)

		results = append(results, &porch.FunctionResult{
			Message: fmt.Sprintf("Generated ServiceMonitor for slice %s", config.SliceID),

			Severity: "info",
		})

	}

	// Generate PodDisruptionBudget for availability.

	if config.SLA != nil && config.SLA.Availability != nil {

		pdb := f.generatePodDisruptionBudget(config)

		resources = append(resources, pdb)

		results = append(results, &porch.FunctionResult{
			Message: fmt.Sprintf("Generated PodDisruptionBudget for slice %s", config.SliceID),

			Severity: "info",
		})

	}

	return resources, results, nil
}

// Helper methods for resource configuration.

func (f *NetworkSliceConfigFunction) applyResourceRequirements(resource *porch.KRMResource, resourceConfig *SliceResourceAllocation) {
	var spec map[string]interface{}
	if resource.Spec != nil {
		if err := json.Unmarshal(resource.Spec, &spec); err != nil {
			spec = make(map[string]interface{})
		}
	} else {
		spec = make(map[string]interface{})
	}

	// Apply to container resources in deployment.
	if template, ok := spec["template"].(map[string]interface{}); ok {
		if spec, ok := template["spec"].(map[string]interface{}); ok {
			if containers, ok := spec["containers"].([]interface{}); ok {
				for _, container := range containers {
					if containerMap, ok := container.(map[string]interface{}); ok {

						if containerMap["resources"] == nil {
							containerMap["resources"] = make(map[string]interface{})
						}

						resourcesMap := containerMap["resources"].(map[string]interface{})

						// Set requests.

						if requests, ok := resourcesMap["requests"].(map[string]interface{}); ok || resourceConfig.CPU != "" || resourceConfig.Memory != "" {

							if requests == nil {

								requests = make(map[string]interface{})

								resourcesMap["requests"] = requests

							}

							if resourceConfig.CPU != "" {
								requests["cpu"] = resourceConfig.CPU
							}

							if resourceConfig.Memory != "" {
								requests["memory"] = resourceConfig.Memory
							}

						}

						// Set limits (typically higher than requests).

						if limits, ok := resourcesMap["limits"].(map[string]interface{}); ok || resourceConfig.CPU != "" || resourceConfig.Memory != "" {

							if limits == nil {

								limits = make(map[string]interface{})

								resourcesMap["limits"] = limits

							}

							if resourceConfig.CPU != "" {
								// Set limits to 2x requests for CPU.

								if cpu, err := k8sresource.ParseQuantity(resourceConfig.CPU); err == nil {

									cpuDouble := cpu.DeepCopy()

									cpuDouble.Add(cpu) // Double it

									limits["cpu"] = cpuDouble.String()

								} else {
									limits["cpu"] = resourceConfig.CPU
								}
							}

							if resourceConfig.Memory != "" {
								// Set limits to 1.5x requests for memory.

								if mem, err := k8sresource.ParseQuantity(resourceConfig.Memory); err == nil {

									memCopy := mem.DeepCopy()

									half := *k8sresource.NewQuantity(mem.Value()/2, k8sresource.DecimalSI)

									memCopy.Add(half) // 1.5x

									limits["memory"] = memCopy.String()

								} else {
									limits["memory"] = resourceConfig.Memory
								}
							}

						}

					}
				}
			}
		}
	}

	// Marshal back to JSON
	marshaled, err := json.Marshal(spec)
	if err != nil {
		return // Silent fail for now
	}
	resource.Spec = json.RawMessage(marshaled)
}

func (f *NetworkSliceConfigFunction) addSliceLabelsAndAnnotations(resource *porch.KRMResource, config *NetworkSliceConfig) {
	// Add labels.
	var metadata map[string]interface{}
	if resource.Metadata != nil {
		if err := json.Unmarshal(resource.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	if metadata["labels"] == nil {
		metadata["labels"] = make(map[string]interface{})
	}

	labels := metadata["labels"].(map[string]interface{})

	labels["nephoran.com/network-slice-id"] = config.SliceID

	labels["nephoran.com/network-slice-type"] = config.SliceType

	labels["nephoran.com/managed-by"] = "network-slice-config"

	// Add annotations.

	if metadata["annotations"] == nil {
		metadata["annotations"] = make(map[string]interface{})
	}

	annotations := metadata["annotations"].(map[string]interface{})

	annotations["nephoran.com/slice-config-timestamp"] = time.Now().UTC().Format(time.RFC3339)

	if config.Description != "" {
		annotations["nephoran.com/slice-description"] = config.Description
	}

	// Add SLA annotations.
	if config.SLA != nil {
		if config.SLA.Latency != nil {
			annotations["nephoran.com/max-latency"] = config.SLA.Latency.MaxLatency
		}

		if config.SLA.Availability != nil {
			annotations["nephoran.com/availability-target"] = fmt.Sprintf("%.4f", config.SLA.Availability.Target)
		}
	}

	// Marshal back to JSON
	metadataMarshaled, err := json.Marshal(metadata)
	if err != nil {
		return // Silent fail for now
	}
	resource.Metadata = json.RawMessage(metadataMarshaled)
}

func (f *NetworkSliceConfigFunction) addQoSAnnotations(resource *porch.KRMResource, qos *QoSConfiguration) {
	var metadata map[string]interface{}
	if resource.Metadata != nil {
		if err := json.Unmarshal(resource.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	if metadata["annotations"] == nil {
		metadata["annotations"] = make(map[string]interface{})
	}

	annotations := metadata["annotations"].(map[string]interface{})

	if qos.FiveQI != 0 {
		annotations["nephoran.com/5qi"] = strconv.Itoa(int(qos.FiveQI))
	}

	if qos.PriorityLevel != 0 {
		annotations["nephoran.com/priority-level"] = strconv.Itoa(int(qos.PriorityLevel))
	}

	if qos.PacketDelayBudget != "" {
		annotations["nephoran.com/packet-delay-budget"] = qos.PacketDelayBudget
	}

	if qos.QoSClass != "" {
		annotations["nephoran.com/qos-class"] = qos.QoSClass
	}

	// Marshal back to JSON
	metadataMarshaled, err := json.Marshal(metadata)
	if err != nil {
		return // Silent fail for now
	}
	resource.Metadata = json.RawMessage(metadataMarshaled)
}

func (f *NetworkSliceConfigFunction) configureNetworkFunctions(resource *porch.KRMResource, nfConfigs []*NetworkFunctionConfig) {
	// This would configure specific network functions based on the slice requirements.

	// For now, add metadata to indicate which NFs should be deployed.
	var metadata map[string]interface{}
	if resource.Metadata != nil {
		if err := json.Unmarshal(resource.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	if metadata["annotations"] == nil {
		metadata["annotations"] = make(map[string]interface{})
	}

	annotations := metadata["annotations"].(map[string]interface{})

	var nfTypes []string

	for _, nfConfig := range nfConfigs {
		nfTypes = append(nfTypes, nfConfig.Type)
	}

	annotations["nephoran.com/required-network-functions"] = strings.Join(nfTypes, ",")

	// Marshal back to JSON
	metadataMarshaled, err := json.Marshal(metadata)
	if err != nil {
		return // Silent fail for now
	}
	resource.Metadata = json.RawMessage(metadataMarshaled)
}

// Resource generation methods.

func (f *NetworkSliceConfigFunction) generateResourceQuota(config *NetworkSliceConfig) porch.KRMResource {
	return porch.KRMResource{
		APIVersion: "v1",

		Kind: "ResourceQuota",

		Metadata: json.RawMessage(`{
			"name": "` + config.SliceID + `-quota",
			"labels": {
				"nephoran.com/network-slice-id": "` + config.SliceID + `"
			}
		}`),

		Spec: json.RawMessage(`{
			"hard": {
				"requests.cpu": "` + config.Resources.CPU + `",
				"requests.memory": "` + config.Resources.Memory + `",
				"limits.cpu": "` + config.Resources.CPU + `",
				"limits.memory": "` + config.Resources.Memory + `"
			}
		}`),
	}
}

func (f *NetworkSliceConfigFunction) generateNetworkPolicy(config *NetworkSliceConfig) porch.KRMResource {
	return porch.KRMResource{
		APIVersion: "networking.k8s.io/v1",

		Kind: "NetworkPolicy",

		Metadata: json.RawMessage(`{
			"name": "` + config.SliceID + `-netpol",
			"labels": {
				"nephoran.com/network-slice-id": "` + config.SliceID + `"
			}
		}`),

		Spec: json.RawMessage(`{
			"podSelector": {
				"matchLabels": {
					"nephoran.com/network-slice-id": "` + config.SliceID + `"
				}
			},
			"policyTypes": ["Ingress", "Egress"],
			"ingress": [{}],
			"egress": [{}]
		}`),
	}
}

func (f *NetworkSliceConfigFunction) generateServiceMonitor(config *NetworkSliceConfig) porch.KRMResource {
	return porch.KRMResource{
		APIVersion: "monitoring.coreos.com/v1",

		Kind: "ServiceMonitor",

		Metadata: json.RawMessage(`{
			"name": "` + config.SliceID + `-monitor",
			"labels": {
				"nephoran.com/network-slice-id": "` + config.SliceID + `"
			}
		}`),

		Spec: json.RawMessage(`{
			"selector": {
				"matchLabels": {
					"nephoran.com/network-slice-id": "` + config.SliceID + `"
				}
			},
			"endpoints": [{
				"port": "metrics"
			}]
		}`),
	}
}

func (f *NetworkSliceConfigFunction) generatePodDisruptionBudget(config *NetworkSliceConfig) porch.KRMResource {
	// Calculate minimum available based on availability target.

	minAvailable := "50%"

	if config.SLA.Availability.Target > 0.999 {
		minAvailable = "80%"
	} else if config.SLA.Availability.Target > 0.99 {
		minAvailable = "66%"
	}

	return porch.KRMResource{
		APIVersion: "policy/v1",

		Kind: "PodDisruptionBudget",

		Metadata: json.RawMessage(`{
			"name": "` + config.SliceID + `-pdb",
			"labels": {
				"nephoran.com/network-slice-id": "` + config.SliceID + `"
			}
		}`),

		Spec: json.RawMessage(`{
			"selector": {
				"matchLabels": {
					"nephoran.com/network-slice-id": "` + config.SliceID + `"
				}
			},
			"minAvailable": "` + minAvailable + `"
		}`),
	}
}

