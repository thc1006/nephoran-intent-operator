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
	"math"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// FiveGCoreOptimizer optimizes 5G Core network function configurations.

type FiveGCoreOptimizer struct {
	*BaseFunctionImpl
}

// NewFiveGCoreOptimizer creates a new 5G Core optimizer.

func NewFiveGCoreOptimizer() *FiveGCoreOptimizer {
	metadata := &FunctionMetadata{
		Name: "5g-core-optimizer",

		Version: "v1.0.0",

		Description: "Optimizes 5G Core network function configurations for performance and resource utilization",

		Type: FunctionTypeMutator,

		Categories: []string{"5g", "core", "optimization", "performance"},

		Keywords: []string{"5g", "optimization", "performance", "resource", "scaling"},

		ResourceTypes: []ResourceTypeSupport{
			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "AMF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "SMF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "UPF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "NSSF", Operations: []string{"read", "update"}},
		},

		Telecom: &TelecomProfile{
			NetworkFunctionTypes: []string{"AMF", "SMF", "UPF", "NSSF"},

			FiveGCapabilities: []string{"performance-optimization", "auto-scaling", "resource-planning"},
		},

		Author: "Nephoran Intent Operator",

		License: "Apache-2.0",
	}

	schema := &FunctionSchema{
		Type: "object",

		Properties: map[string]*SchemaProperty{
			"optimizationTarget": {
				Type: "string",

				Description: "Target for optimization",

				Enum: []interface{}{"performance", "resources", "cost", "balanced"},

				Default: "balanced",
			},

			"expectedLoad": {
				Type: "object",

				Description: "Expected system load characteristics",

				Properties: map[string]*SchemaProperty{
					"peakUsers": {
						Type: "integer",

						Description: "Peak concurrent users",

						Minimum: floatPtr(1000),

						Maximum: floatPtr(10000000),
					},

					"averageUsers": {
						Type: "integer",

						Description: "Average concurrent users",

						Minimum: floatPtr(100),
					},

					"sessionRate": {
						Type: "number",

						Description: "Sessions per second",

						Minimum: floatPtr(10),
					},
				},
			},

			"resourceConstraints": {
				Type: "object",

				Description: "Resource constraints",

				Properties: map[string]*SchemaProperty{
					"maxCPU": {Type: "string", Description: "Maximum CPU per instance"},

					"maxMemory": {Type: "string", Description: "Maximum memory per instance"},

					"maxInstances": {Type: "integer", Description: "Maximum number of instances"},
				},
			},
		},
	}

	base := NewBaseFunctionImpl(metadata, schema)

	return &FiveGCoreOptimizer{BaseFunctionImpl: base}
}

// Execute optimizes 5G Core configurations.

func (o *FiveGCoreOptimizer) Execute(ctx context.Context, input *ResourceList) (*ResourceList, error) {
	LogInfo(ctx, "Starting 5G Core optimization")

	// Parse configuration.
	var functionConfigMap map[string]interface{}
	if input.FunctionConfig != nil {
		if err := json.Unmarshal(input.FunctionConfig, &functionConfigMap); err != nil {
			functionConfigMap = make(map[string]interface{})
		}
	} else {
		functionConfigMap = make(map[string]interface{})
	}

	config := o.parseOptimizationConfig(functionConfigMap)

	// Create output resource list.

	output := &ResourceList{
		Items: make([]porch.KRMResource, len(input.Items)),

		FunctionConfig: input.FunctionConfig,

		Results: []*porch.FunctionResult{},

		Context: input.Context,
	}

	// Copy input resources.

	copy(output.Items, input.Items)

	// Optimize each network function type.

	o.optimizeAMFs(ctx, output, config)

	o.optimizeSMFs(ctx, output, config)

	o.optimizeUPFs(ctx, output, config)

	o.optimizeNSSFs(ctx, output, config)

	// Cross-function optimization.

	o.optimizeInterFunctionConnections(ctx, output, config)

	LogInfo(ctx, "5G Core optimization completed", "results", len(output.Results))

	return output, nil
}

// NetworkSliceOptimizer optimizes network slice configurations.

type NetworkSliceOptimizer struct {
	*BaseFunctionImpl
}

// NewNetworkSliceOptimizer creates a new network slice optimizer.

func NewNetworkSliceOptimizer() *NetworkSliceOptimizer {
	metadata := &FunctionMetadata{
		Name: "network-slice-optimizer",

		Version: "v1.0.0",

		Description: "Optimizes network slice configurations for different service types",

		Type: FunctionTypeMutator,

		Categories: []string{"network-slicing", "optimization", "5g"},

		Keywords: []string{"network-slice", "embb", "urllc", "mmtc", "qos", "sla"},

		ResourceTypes: []ResourceTypeSupport{
			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "NetworkSlice", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "AMF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "SMF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "UPF", Operations: []string{"read", "update"}},
		},

		Telecom: &TelecomProfile{
			NetworkSliceSupport: true,

			FiveGCapabilities: []string{"eMBB", "URLLC", "mMTC", "network-slicing"},
		},

		Author: "Nephoran Intent Operator",

		License: "Apache-2.0",
	}

	schema := &FunctionSchema{
		Type: "object",

		Properties: map[string]*SchemaProperty{
			"sliceTypes": {
				Type: "array",

				Description: "Types of network slices to optimize",

				Items: &SchemaProperty{
					Type: "string",

					Enum: []interface{}{"eMBB", "URLLC", "mMTC"},
				},
			},

			"isolationLevel": {
				Type: "string",

				Description: "Isolation level for network slices",

				Enum: []interface{}{"none", "logical", "physical"},

				Default: "logical",
			},
		},
	}

	base := NewBaseFunctionImpl(metadata, schema)

	return &NetworkSliceOptimizer{BaseFunctionImpl: base}
}

// Execute optimizes network slice configurations.

func (o *NetworkSliceOptimizer) Execute(ctx context.Context, input *ResourceList) (*ResourceList, error) {
	LogInfo(ctx, "Starting network slice optimization")

	output := &ResourceList{
		Items: make([]porch.KRMResource, len(input.Items)),

		FunctionConfig: input.FunctionConfig,

		Results: []*porch.FunctionResult{},

		Context: input.Context,
	}

	copy(output.Items, input.Items)

	// Find network slices.

	networkSlices := FindResourcesByGVK(output.Items, schema.GroupVersionKind{
		Group: "workload.nephio.org",

		Version: "v1alpha1",

		Kind: "NetworkSlice",
	})

	// Optimize each network slice.

	for _, slice := range networkSlices {

		sliceIndex := o.findResourceIndex(output.Items, &slice)

		if sliceIndex >= 0 {

			results := o.optimizeNetworkSlice(ctx, &output.Items[sliceIndex])

			output.Results = append(output.Results, results...)

		}

	}

	LogInfo(ctx, "Network slice optimization completed",

		"slices", len(networkSlices),

		"results", len(output.Results))

	return output, nil
}

// MultiVendorNormalizer normalizes multi-vendor configurations.

type MultiVendorNormalizer struct {
	*BaseFunctionImpl
}

// NewMultiVendorNormalizer creates a new multi-vendor normalizer.

func NewMultiVendorNormalizer() *MultiVendorNormalizer {
	metadata := &FunctionMetadata{
		Name: "multi-vendor-normalizer",

		Version: "v1.0.0",

		Description: "Normalizes configurations across different vendor implementations",

		Type: FunctionTypeMutator,

		Categories: []string{"multi-vendor", "normalization", "interoperability"},

		Keywords: []string{"vendor", "interoperability", "normalization", "standards"},

		ResourceTypes: []ResourceTypeSupport{
			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "AMF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "SMF", Operations: []string{"read", "update"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "UPF", Operations: []string{"read", "update"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "ODU", Operations: []string{"read", "update"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "OCU", Operations: []string{"read", "update"}},
		},

		Telecom: &TelecomProfile{
			Standards: []StandardCompliance{
				{Name: "3GPP", Version: "R17", Required: true},

				{Name: "O-RAN", Version: "R1.0", Required: true},
			},
		},

		Author: "Nephoran Intent Operator",

		License: "Apache-2.0",
	}

	schema := &FunctionSchema{
		Type: "object",

		Properties: map[string]*SchemaProperty{
			"targetVendor": {
				Type: "string",

				Description: "Target vendor for normalization",

				Enum: []interface{}{"generic", "ericsson", "nokia", "huawei", "samsung"},

				Default: "generic",
			},

			"normalizeInterfaces": {
				Type: "boolean",

				Description: "Normalize interface configurations",

				Default: true,
			},

			"normalizeParameters": {
				Type: "boolean",

				Description: "Normalize vendor-specific parameters",

				Default: true,
			},
		},
	}

	base := NewBaseFunctionImpl(metadata, schema)

	return &MultiVendorNormalizer{BaseFunctionImpl: base}
}

// Execute normalizes multi-vendor configurations.

func (n *MultiVendorNormalizer) Execute(ctx context.Context, input *ResourceList) (*ResourceList, error) {
	LogInfo(ctx, "Starting multi-vendor normalization")

	var functionConfigMap map[string]interface{}
	if input.FunctionConfig != nil {
		if err := json.Unmarshal(input.FunctionConfig, &functionConfigMap); err != nil {
			functionConfigMap = make(map[string]interface{})
		}
	} else {
		functionConfigMap = make(map[string]interface{})
	}

	config := n.parseNormalizationConfig(functionConfigMap)

	output := &ResourceList{
		Items: make([]porch.KRMResource, len(input.Items)),

		FunctionConfig: input.FunctionConfig,

		Results: []*porch.FunctionResult{},

		Context: input.Context,
	}

	copy(output.Items, input.Items)

	// Normalize each resource.

	for i := range output.Items {

		results := n.normalizeResource(ctx, &output.Items[i], config)

		output.Results = append(output.Results, results...)

	}

	LogInfo(ctx, "Multi-vendor normalization completed", "results", len(output.Results))

	return output, nil
}

// Helper methods for FiveGCoreOptimizer.

func (o *FiveGCoreOptimizer) parseOptimizationConfig(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return map[string]interface{}{
			"peakUsers":    100000,
			"averageUsers": 50000,
			"sessionRate":  1000,
		}
	}

	return config
}

func (o *FiveGCoreOptimizer) optimizeAMFs(ctx context.Context, output *ResourceList, config map[string]interface{}) {
	amfs := FindResourcesByGVK(output.Items, schema.GroupVersionKind{
		Group: "workload.nephio.org",

		Version: "v1alpha1",

		Kind: "AMF",
	})

	for _, amf := range amfs {

		amfIndex := o.findResourceIndex(output.Items, &amf)

		if amfIndex >= 0 {

			results := o.optimizeAMF(ctx, &output.Items[amfIndex], config)

			output.Results = append(output.Results, results...)

		}

	}
}

func (o *FiveGCoreOptimizer) optimizeAMF(ctx context.Context, amf *porch.KRMResource, config map[string]interface{}) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(amf)

	// Get expected load.

	expectedLoad, ok := config["expectedLoad"].(map[string]interface{})

	if !ok {
		return results
	}

	peakUsers, _ := expectedLoad["peakUsers"].(int)

	if peakUsers == 0 {
		peakUsers = 100000
	}

	// Calculate optimal resource allocation.

	cpuCores := o.calculateAMFCPU(peakUsers)

	memory := o.calculateAMFMemory(peakUsers)

	replicas := o.calculateAMFReplicas(peakUsers)

	// Update resource specifications.

	if err := SetSpecField(amf, "resources.requests.cpu", fmt.Sprintf("%dm", cpuCores)); err == nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("AMF %s: Set CPU request to %dm", name, cpuCores),
		))
	}

	if err := SetSpecField(amf, "resources.requests.memory", fmt.Sprintf("%dMi", memory)); err == nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("AMF %s: Set memory request to %dMi", name, memory),
		))
	}

	// Set resource limits (20% higher than requests).

	cpuLimit := int(float64(cpuCores) * 1.2)

	memoryLimit := int(float64(memory) * 1.2)

	SetSpecField(amf, "resources.limits.cpu", fmt.Sprintf("%dm", cpuLimit))

	SetSpecField(amf, "resources.limits.memory", fmt.Sprintf("%dMi", memoryLimit))

	// Update replica count.

	if err := SetSpecField(amf, "replicas", replicas); err == nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("AMF %s: Set replica count to %d", name, replicas),
		))
	}

	// Optimize connection pool sizes.

	maxConnections := o.calculateAMFConnections(peakUsers)

	if err := SetSpecField(amf, "connectionPool.maxConnections", maxConnections); err == nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("AMF %s: Set max connections to %d", name, maxConnections),
		))
	}

	return results
}

func (o *FiveGCoreOptimizer) optimizeSMFs(ctx context.Context, output *ResourceList, config map[string]interface{}) {
	smfs := FindResourcesByGVK(output.Items, schema.GroupVersionKind{
		Group: "workload.nephio.org",

		Version: "v1alpha1",

		Kind: "SMF",
	})

	for _, smf := range smfs {

		smfIndex := o.findResourceIndex(output.Items, &smf)

		if smfIndex >= 0 {

			results := o.optimizeSMF(ctx, &output.Items[smfIndex], config)

			output.Results = append(output.Results, results...)

		}

	}
}

func (o *FiveGCoreOptimizer) optimizeSMF(ctx context.Context, smf *porch.KRMResource, config map[string]interface{}) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(smf)

	expectedLoad, ok := config["expectedLoad"].(map[string]interface{})

	if !ok {
		return results
	}

	sessionRate, _ := expectedLoad["sessionRate"].(float64)

	if sessionRate == 0 {
		sessionRate = 1000
	}

	// Calculate SMF resources based on session rate.

	cpuCores := o.calculateSMFCPU(sessionRate)

	memory := o.calculateSMFMemory(sessionRate)

	// Update resources.

	SetSpecField(smf, "resources.requests.cpu", fmt.Sprintf("%dm", cpuCores))

	SetSpecField(smf, "resources.requests.memory", fmt.Sprintf("%dMi", memory))

	// Configure PDU session pool.

	maxSessions := int(sessionRate * 3600) // Sessions per hour

	if err := SetSpecField(smf, "pduSessions.maxConcurrentSessions", maxSessions); err == nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("SMF %s: Set max concurrent sessions to %d", name, maxSessions),
		))
	}

	// Configure PFCP settings.

	pfcpTimeout := o.calculatePFCPTimeout(sessionRate)

	SetSpecField(smf, "pfcp.sessionTimeout", fmt.Sprintf("%ds", pfcpTimeout))

	results = append(results, CreateInfo(

		fmt.Sprintf("SMF %s optimized for %0.0f sessions/sec", name, sessionRate),
	))

	return results
}

func (o *FiveGCoreOptimizer) optimizeUPFs(ctx context.Context, output *ResourceList, config map[string]interface{}) {
	upfs := FindResourcesByGVK(output.Items, schema.GroupVersionKind{
		Group: "workload.nephio.org",

		Version: "v1alpha1",

		Kind: "UPF",
	})

	for _, upf := range upfs {

		upfIndex := o.findResourceIndex(output.Items, &upf)

		if upfIndex >= 0 {

			results := o.optimizeUPF(ctx, &output.Items[upfIndex], config)

			output.Results = append(output.Results, results...)

		}

	}
}

func (o *FiveGCoreOptimizer) optimizeUPF(ctx context.Context, upf *porch.KRMResource, config map[string]interface{}) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(upf)

	// UPF optimization focuses on throughput and packet processing.

	expectedLoad, ok := config["expectedLoad"].(map[string]interface{})

	if !ok {
		return results
	}

	peakUsers, _ := expectedLoad["peakUsers"].(int)

	if peakUsers == 0 {
		peakUsers = 100000
	}

	// Calculate throughput requirements (assuming average 10 Mbps per user).

	totalThroughputGbps := float64(peakUsers) * 10 / 1000 // Convert to Gbps

	// Calculate UPF resources.

	cpuCores := o.calculateUPFCPU(totalThroughputGbps)

	memory := o.calculateUPFMemory(totalThroughputGbps)

	// Update resources.

	SetSpecField(upf, "resources.requests.cpu", fmt.Sprintf("%dm", cpuCores))

	SetSpecField(upf, "resources.requests.memory", fmt.Sprintf("%dMi", memory))

	// Configure data plane.

	workerThreads := maxInt(2, cpuCores/1000) // One thread per CPU core

	if err := SetSpecField(upf, "dataPlane.workerThreads", workerThreads); err == nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("UPF %s: Set worker threads to %d", name, workerThreads),
		))
	}

	// Configure packet buffer.

	packetBufferSize := o.calculatePacketBufferSize(totalThroughputGbps)

	SetSpecField(upf, "dataPlane.packetBufferSize", fmt.Sprintf("%dMB", packetBufferSize))

	results = append(results, CreateInfo(

		fmt.Sprintf("UPF %s optimized for %.1f Gbps throughput", name, totalThroughputGbps),
	))

	return results
}

func (o *FiveGCoreOptimizer) optimizeNSSFs(ctx context.Context, output *ResourceList, config map[string]interface{}) {
	nssfs := FindResourcesByGVK(output.Items, schema.GroupVersionKind{
		Group: "workload.nephio.org",

		Version: "v1alpha1",

		Kind: "NSSF",
	})

	for _, nssf := range nssfs {

		nssfIndex := o.findResourceIndex(output.Items, &nssf)

		if nssfIndex >= 0 {

			results := o.optimizeNSSF(ctx, &output.Items[nssfIndex], config)

			output.Results = append(output.Results, results...)

		}

	}
}

func (o *FiveGCoreOptimizer) optimizeNSSF(ctx context.Context, nssf *porch.KRMResource, config map[string]interface{}) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(nssf)

	// NSSF is typically lightweight, focus on availability.

	SetSpecField(nssf, "resources.requests.cpu", "100m")

	SetSpecField(nssf, "resources.requests.memory", "256Mi")

	SetSpecField(nssf, "replicas", 2) // Always run at least 2 replicas for HA

	results = append(results, CreateInfo(

		fmt.Sprintf("NSSF %s configured for high availability", name),
	))

	return results
}

func (o *FiveGCoreOptimizer) optimizeInterFunctionConnections(ctx context.Context, output *ResourceList, config map[string]interface{}) {
	// This would optimize connections between different NFs.

	// For example, configuring load balancer affinities, connection pooling, etc.

	output.Results = append(output.Results, CreateInfo(

		"Inter-function connection optimization completed",
	))
}

// Resource calculation methods.

func (o *FiveGCoreOptimizer) calculateAMFCPU(peakUsers int) int {
	// Base CPU requirement plus scaling factor.

	baseCPU := 500 // 500m base

	scalingFactor := float64(peakUsers) / 10000 // 100m per 10k users

	return baseCPU + int(scalingFactor*100)
}

func (o *FiveGCoreOptimizer) calculateAMFMemory(peakUsers int) int {
	// Base memory plus per-user memory.

	baseMemory := 512 // 512Mi base

	perUserMemory := float64(peakUsers) * 0.001 // 1KB per user

	return baseMemory + int(perUserMemory)
}

func (o *FiveGCoreOptimizer) calculateAMFReplicas(peakUsers int) int {
	// Each AMF instance can handle ~50k users.

	replicas := int(math.Ceil(float64(peakUsers) / 50000))

	return maxInt(2, replicas) // Minimum 2 for HA
}

func (o *FiveGCoreOptimizer) calculateAMFConnections(peakUsers int) int {
	// Assume 10% of users are actively connected.

	return int(float64(peakUsers) * 0.1)
}

func (o *FiveGCoreOptimizer) calculateSMFCPU(sessionRate float64) int {
	// CPU scales with session establishment rate.

	baseCPU := 300

	sessionCPU := sessionRate * 0.5 // 0.5m per session/sec

	return baseCPU + int(sessionCPU)
}

func (o *FiveGCoreOptimizer) calculateSMFMemory(sessionRate float64) int {
	baseMemory := 256

	sessionMemory := sessionRate * 0.1 // 0.1Mi per session/sec

	return baseMemory + int(sessionMemory)
}

func (o *FiveGCoreOptimizer) calculatePFCPTimeout(sessionRate float64) int {
	// Higher session rates need shorter timeouts.

	if sessionRate > 1000 {
		return 30
	} else if sessionRate > 100 {
		return 60
	}

	return 120
}

func (o *FiveGCoreOptimizer) calculateUPFCPU(throughputGbps float64) int {
	// UPF is CPU intensive for packet processing.

	baseCPU := 1000 // 1 CPU base

	throughputCPU := throughputGbps * 500 // 500m per Gbps

	return baseCPU + int(throughputCPU)
}

func (o *FiveGCoreOptimizer) calculateUPFMemory(throughputGbps float64) int {
	baseMemory := 1024 // 1Gi base

	throughputMemory := throughputGbps * 200 // 200Mi per Gbps

	return baseMemory + int(throughputMemory)
}

func (o *FiveGCoreOptimizer) calculatePacketBufferSize(throughputGbps float64) int {
	// Buffer size based on throughput.

	baseBuffer := 128 // 128MB base

	throughputBuffer := throughputGbps * 64 // 64MB per Gbps

	return baseBuffer + int(throughputBuffer)
}

func (o *FiveGCoreOptimizer) findResourceIndex(resources []porch.KRMResource, target *porch.KRMResource) int {
	targetName, _ := GetResourceName(target)

	for i, resource := range resources {
		if resource.Kind == target.Kind && resource.APIVersion == target.APIVersion {
			if name, _ := GetResourceName(&resource); name == targetName {
				return i
			}
		}
	}

	return -1
}

// Network slice optimization methods.

func (o *NetworkSliceOptimizer) optimizeNetworkSlice(ctx context.Context, slice *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(slice)

	// Get slice type.

	sliceType, err := GetSpecField(slice, "sliceType")
	if err != nil {

		results = append(results, CreateWarning(

			fmt.Sprintf("Network slice %s has no slice type defined", name),
		))

		return results

	}

	sliceTypeStr, ok := sliceType.(string)

	if !ok {
		return results
	}

	switch strings.ToUpper(sliceTypeStr) {

	case "EMBB":

		results = append(results, o.optimizeEMBBSlice(ctx, slice)...)

	case "URLLC":

		results = append(results, o.optimizeURLLCSlice(ctx, slice)...)

	case "MMTC":

		results = append(results, o.optimizeMTCSlice(ctx, slice)...)

	default:

		results = append(results, CreateWarning(

			fmt.Sprintf("Unknown slice type %s for slice %s", sliceTypeStr, name),
		))

	}

	return results
}

func (o *NetworkSliceOptimizer) optimizeEMBBSlice(ctx context.Context, slice *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(slice)

	// eMBB optimizations focus on throughput.

	SetSpecField(slice, "qos.5qi", 9) // Non-GBR for eMBB

	SetSpecField(slice, "qos.priorityLevel", 8) // Lower priority

	SetSpecField(slice, "sla.throughput.maxDownlink", "1Gbps")

	SetSpecField(slice, "sla.latency.maxLatency", "50ms")

	results = append(results, CreateInfo(

		fmt.Sprintf("eMBB slice %s optimized for high throughput", name),
	))

	return results
}

func (o *NetworkSliceOptimizer) optimizeURLLCSlice(ctx context.Context, slice *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(slice)

	// URLLC optimizations focus on low latency and reliability.

	SetSpecField(slice, "qos.5qi", 1) // GBR for URLLC

	SetSpecField(slice, "qos.priorityLevel", 1) // Highest priority

	SetSpecField(slice, "sla.latency.maxLatency", "1ms")

	SetSpecField(slice, "sla.reliability.successRate", 0.999999) // 99.9999% reliability

	SetSpecField(slice, "isolation.level", "physical") // Physical isolation for URLLC

	results = append(results, CreateInfo(

		fmt.Sprintf("URLLC slice %s optimized for ultra-low latency", name),
	))

	return results
}

func (o *NetworkSliceOptimizer) optimizeMTCSlice(ctx context.Context, slice *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(slice)

	// mMTC optimizations focus on massive connectivity.

	SetSpecField(slice, "qos.5qi", 8) // Non-GBR, delay tolerant

	SetSpecField(slice, "qos.priorityLevel", 9) // Lowest priority

	SetSpecField(slice, "sla.latency.maxLatency", "10s")

	SetSpecField(slice, "resources.connections", 1000000) // Support massive connections

	results = append(results, CreateInfo(

		fmt.Sprintf("mMTC slice %s optimized for massive connectivity", name),
	))

	return results
}

func (o *NetworkSliceOptimizer) findResourceIndex(resources []porch.KRMResource, target *porch.KRMResource) int {
	targetName, _ := GetResourceName(target)

	for i, resource := range resources {
		if resource.Kind == target.Kind && resource.APIVersion == target.APIVersion {
			if name, _ := GetResourceName(&resource); name == targetName {
				return i
			}
		}
	}

	return -1
}

// Multi-vendor normalization methods.

func (n *MultiVendorNormalizer) parseNormalizationConfig(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return make(map[string]interface{})
	}

	return config
}

func (n *MultiVendorNormalizer) normalizeResource(ctx context.Context, resource *porch.KRMResource, config map[string]interface{}) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Detect current vendor.

	vendor := n.detectVendor(resource)

	targetVendor, ok := config["targetVendor"].(string)

	if !ok {
		targetVendor = "generic"
	}

	if vendor != "" && vendor != targetVendor {
		results = append(results, n.normalizeVendorSpecific(ctx, resource, vendor, targetVendor)...)
	}

	// Normalize interface configurations.

	if normalizeInterfaces, ok := config["normalizeInterfaces"].(bool); ok && normalizeInterfaces {
		results = append(results, n.normalizeInterfaces(ctx, resource)...)
	}

	// Add normalization metadata.

	SetResourceAnnotation(resource, "nephoran.io/original-vendor", vendor)

	SetResourceAnnotation(resource, "nephoran.io/normalized-to", targetVendor)

	if len(results) > 0 {
		results = append(results, CreateInfo(

			fmt.Sprintf("Resource %s normalized from %s to %s", name, vendor, targetVendor),
		))
	}

	return results
}

func (n *MultiVendorNormalizer) detectVendor(resource *porch.KRMResource) string {
	// Check annotations for vendor information.

	if vendor, exists := GetResourceAnnotation(resource, "vendor"); exists {
		return strings.ToLower(vendor)
	}

	// Check labels.

	if vendor, exists := GetResourceLabel(resource, "vendor"); exists {
		return strings.ToLower(vendor)
	}

	// Check spec fields for vendor-specific patterns.
	if resource.Spec != nil {
		spec := resource.Spec
		// Ericsson patterns.
		if _, exists := spec["ericsson"]; exists {
			return "ericsson"
		}

		// Nokia patterns.
		if _, exists := spec["nokia"]; exists {
			return "nokia"
		}

		// Huawei patterns.
		if _, exists := spec["huawei"]; exists {
			return "huawei"
		}
	}

	return "unknown"
}

func (n *MultiVendorNormalizer) normalizeVendorSpecific(ctx context.Context, resource *porch.KRMResource, fromVendor, toVendor string) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	// This would contain vendor-specific normalization logic.

	// For example, converting Ericsson-specific parameters to Nokia format.

	switch fromVendor {

	case "ericsson":

		results = append(results, n.normalizeFromEricsson(ctx, resource, toVendor)...)

	case "nokia":

		results = append(results, n.normalizeFromNokia(ctx, resource, toVendor)...)

	case "huawei":

		results = append(results, n.normalizeFromHuawei(ctx, resource, toVendor)...)

	}

	return results
}

func (n *MultiVendorNormalizer) normalizeFromEricsson(ctx context.Context, resource *porch.KRMResource, toVendor string) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	// Example: Convert Ericsson-specific configuration format.

	if ericssonConfig, err := GetSpecField(resource, "ericsson"); err == nil && ericssonConfig != nil {
		// Convert to generic format.

		if genericConfig := n.convertEricssonToGeneric(ericssonConfig); genericConfig != nil {

			SetSpecField(resource, "config", genericConfig)

			// Remove vendor-specific section.

			if resource.Spec != nil {
				spec := resource.Spec
				delete(spec, "ericsson")
				resource.Spec = spec
			}

			results = append(results, CreateInfo("Converted Ericsson-specific configuration"))

		}
	}

	return results
}

func (n *MultiVendorNormalizer) normalizeFromNokia(ctx context.Context, resource *porch.KRMResource, toVendor string) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	// Similar normalization for Nokia.

	return results
}

func (n *MultiVendorNormalizer) normalizeFromHuawei(ctx context.Context, resource *porch.KRMResource, toVendor string) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	// Similar normalization for Huawei.

	return results
}

func (n *MultiVendorNormalizer) normalizeInterfaces(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	// Normalize interface naming and parameters.

	if interfaces, err := GetSpecField(resource, "interfaces"); err == nil && interfaces != nil {
		if interfaceMap, ok := interfaces.(map[string]interface{}); ok {

			normalizedInterfaces := n.normalizeInterfaceMap(interfaceMap)

			SetSpecField(resource, "interfaces", normalizedInterfaces)

			results = append(results, CreateInfo("Normalized interface configurations"))

		}
	}

	return results
}

func (n *MultiVendorNormalizer) convertEricssonToGeneric(ericssonConfig interface{}) map[string]interface{} {
	// This would contain actual conversion logic.

	// For now, return a placeholder.

	return make(map[string]interface{})
}

func (n *MultiVendorNormalizer) normalizeInterfaceMap(interfaces map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for key, value := range interfaces {

		// Normalize interface names to standard format.

		normalizedKey := n.normalizeInterfaceName(key)

		normalized[normalizedKey] = value

	}

	return normalized
}

func (n *MultiVendorNormalizer) normalizeInterfaceName(interfaceName string) string {
	// Normalize interface names to standard format.

	lower := strings.ToLower(interfaceName)

	// Map vendor-specific names to standard names.

	nameMapping := map[string]string{
		"n1-n2": "n1_n2",

		"n1/n2": "n1_n2",

		"sbi-if": "sbi",

		"sbi_intf": "sbi",

		"pfcp-if": "pfcp",

		"pfcp_intf": "pfcp",
	}

	if normalized, exists := nameMapping[lower]; exists {
		return normalized
	}

	return lower
}

// Utility functions.

func floatPtr(f float64) *float64 {
	return &f
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

