package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// Additional types for resource planning
type InterfaceConfiguration struct {
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Protocol  string         `json:"protocol"`
	Endpoints []EndpointSpec `json:"endpoints"`
	Security  SecuritySpec   `json:"security"`
	QoS       QoSSpec        `json:"qos"`
}

type EndpointSpec struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Path     string `json:"path,omitempty"`
}

type SecuritySpec struct {
	Authentication string `json:"authentication"`
	Encryption     string `json:"encryption"`
}

type QoSSpec struct {
	Bandwidth  string  `json:"bandwidth"`
	Latency    string  `json:"latency"`
	Jitter     string  `json:"jitter"`
	PacketLoss float64 `json:"packet_loss"`
}

type SecurityPolicy struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type HealthCheckSpec struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Port     int    `json:"port"`
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
	Retries  int    `json:"retries"`
}

type MonitoringSpec struct {
	Enabled    bool     `json:"enabled"`
	Metrics    []string `json:"metrics"`
	Alerts     []string `json:"alerts"`
	Dashboards []string `json:"dashboards"`
}

// ResourcePlanner handles resource planning and manifest generation phases
type ResourcePlanner struct {
	*NetworkIntentReconciler
}

// Ensure ResourcePlanner implements the required interfaces
var _ ResourcePlannerInterface = (*ResourcePlanner)(nil)
var _ PhaseProcessor = (*ResourcePlanner)(nil)

// NewResourcePlanner creates a new resource planner
func NewResourcePlanner(r *NetworkIntentReconciler) *ResourcePlanner {
	return &ResourcePlanner{
		NetworkIntentReconciler: r,
	}
}

// PlanResources implements Phase 2: Resource planning with telecom knowledge
func (p *ResourcePlanner) PlanResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "resource-planning")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseResourcePlanning

	// Get telecom knowledge base
	kb := p.deps.GetTelecomKnowledgeBase()
	if kb == nil {
		err := fmt.Errorf("telecom knowledge base not available")
		// Set Ready condition to indicate configuration issue
		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "TelecomKnowledgeBaseNotAvailable", "Telecom knowledge base is not properly configured")
		return ctrl.Result{}, err
	}

	// Parse extracted parameters from LLM phase
	var llmParams map[string]interface{}
	if len(networkIntent.Spec.Parameters.Raw) > 0 {
		if err := json.Unmarshal(networkIntent.Spec.Parameters.Raw, &llmParams); err != nil {
			logger.Error(err, "failed to parse LLM parameters")
			// Set Ready condition to indicate parameter parsing issue
			p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "LLMParametersParsingFailed", fmt.Sprintf("Failed to parse LLM parameters: %v", err))
			return ctrl.Result{RequeueAfter: p.config.RetryDelay}, err
		}
	}

	// Create resource plan
	resourcePlan := &ResourcePlan{
		NetworkFunctions: []PlannedNetworkFunction{},
		Interfaces:       []InterfaceConfiguration{},
		SecurityPolicies: []SecurityPolicy{},
	}

	// Extract network functions from LLM parameters
	if nfList, ok := llmParams["network_functions"].([]interface{}); ok {
		for _, nfInterface := range nfList {
			if nfName, ok := nfInterface.(string); ok {
				if nfSpec, exists := kb.GetNetworkFunction(nfName); exists {
					plannedNF := p.planNetworkFunction(nfSpec, llmParams, processingCtx.TelecomContext)
					resourcePlan.NetworkFunctions = append(resourcePlan.NetworkFunctions, plannedNF)
				}
			}
		}
	}

	// Plan slice configuration if specified
	if sliceConfig, ok := llmParams["slice_configuration"].(map[string]interface{}); ok {
		sliceType := ""
		if st, ok := sliceConfig["slice_type"].(string); ok {
			sliceType = st
		}

		if sliceSpec, exists := kb.GetSliceType(sliceType); exists {
			resourcePlan.SliceConfiguration = &SliceConfiguration{
				SliceType:  sliceType,
				SST:        sliceSpec.SST,
				QoSProfile: sliceSpec.QosProfile,
				Isolation:  "standard",
				Parameters: make(map[string]interface{}),
			}

			// Copy slice requirements to parameters
			resourcePlan.SliceConfiguration.Parameters["latency_requirement"] = sliceSpec.Requirements.Latency.UserPlane
			resourcePlan.SliceConfiguration.Parameters["throughput_min"] = sliceSpec.Requirements.Throughput.Min
			resourcePlan.SliceConfiguration.Parameters["reliability"] = sliceSpec.Requirements.Reliability.Availability
		}
	}

	// Plan interfaces based on network functions
	interfaceMap := make(map[string]bool)
	for _, nf := range resourcePlan.NetworkFunctions {
		for _, ifaceName := range nf.Interfaces {
			if !interfaceMap[ifaceName] {
				if ifaceSpec, exists := kb.GetInterface(ifaceName); exists {
					interfaceConfig := InterfaceConfiguration{
						Name:      ifaceSpec.Name,
						Type:      ifaceSpec.Type,
						Protocol:  ifaceSpec.Protocol,
						Endpoints: []EndpointSpec{},
						Security: SecuritySpec{
							Authentication: ifaceSpec.SecurityModel.Authentication,
							Encryption:     ifaceSpec.SecurityModel.Encryption,
						},
						QoS: QoSSpec{
							Bandwidth:  ifaceSpec.QosRequirements.Bandwidth,
							Latency:    ifaceSpec.QosRequirements.Latency,
							Jitter:     ifaceSpec.QosRequirements.Jitter,
							PacketLoss: ifaceSpec.QosRequirements.PacketLoss,
						},
					}

					// Add endpoints from interface spec
					for _, endpoint := range ifaceSpec.Endpoints {
						interfaceConfig.Endpoints = append(interfaceConfig.Endpoints, EndpointSpec{
							Name:     endpoint.Name,
							Port:     endpoint.Port,
							Protocol: endpoint.Protocol,
							Path:     endpoint.Path,
						})
					}

					resourcePlan.Interfaces = append(resourcePlan.Interfaces, interfaceConfig)
					interfaceMap[ifaceName] = true
				}
			}
		}
	}

	// Calculate total resource requirements
	totalResources := ResourceRequirements{
		CPU:     "0",
		Memory:  "0Gi",
		Storage: "0Gi",
	}

	var totalCPU, totalMemory, totalStorage float64
	for _, nf := range resourcePlan.NetworkFunctions {
		// Parse CPU
		var cpu float64
		fmt.Sscanf(nf.Resources.CPU, "%f", &cpu)
		totalCPU += cpu * float64(nf.Replicas)

		// Parse Memory
		var memory float64
		fmt.Sscanf(nf.Resources.Memory, "%fGi", &memory)
		totalMemory += memory * float64(nf.Replicas)

		// Parse Storage
		var storage float64
		fmt.Sscanf(nf.Resources.Storage, "%fGi", &storage)
		totalStorage += storage * float64(nf.Replicas)
	}

	totalResources.CPU = fmt.Sprintf("%.1f", totalCPU)
	totalResources.Memory = fmt.Sprintf("%.1fGi", totalMemory)
	totalResources.Storage = fmt.Sprintf("%.1fGi", totalStorage)
	resourcePlan.ResourceRequirements = totalResources

	// Determine deployment pattern
	deploymentPattern := "production"
	if pattern, ok := processingCtx.TelecomContext["deployment_pattern"].(string); ok {
		deploymentPattern = pattern
	}
	resourcePlan.DeploymentPattern = deploymentPattern

	// Estimate cost (simplified calculation)
	resourcePlan.EstimatedCost = p.calculateEstimatedCost(totalCPU, totalMemory, totalStorage)

	// Store resource plan in processing context
	processingCtx.ResourcePlan = resourcePlan

	// Update NetworkIntent status
	planData, err := json.Marshal(resourcePlan)
	if err != nil {
		return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal resource plan: %w", err)
	}

	// Store resource plan in status
	networkIntent.Status.ResourcePlan = string(planData)

	condition := metav1.Condition{
		Type:               "ResourcesPlanned",
		Status:             metav1.ConditionTrue,
		Reason:             "ResourcePlanningSucceeded",
		Message:            fmt.Sprintf("Resource plan created with %d network functions, estimated cost $%.2f", len(resourcePlan.NetworkFunctions), resourcePlan.EstimatedCost),
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := p.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	planningDuration := time.Since(startTime)
	processingCtx.Metrics["resource_planning_duration"] = planningDuration.Seconds()
	processingCtx.Metrics["planned_network_functions"] = float64(len(resourcePlan.NetworkFunctions))
	processingCtx.Metrics["estimated_cost"] = resourcePlan.EstimatedCost

	p.recordEvent(networkIntent, "Normal", "ResourcePlanningSucceeded",
		fmt.Sprintf("Created resource plan with %d NFs", len(resourcePlan.NetworkFunctions)))
	logger.Info("Resource planning phase completed successfully",
		"duration", planningDuration,
		"network_functions", len(resourcePlan.NetworkFunctions),
		"estimated_cost", resourcePlan.EstimatedCost)

	return ctrl.Result{}, nil
}

// planNetworkFunction creates a planned network function from the knowledge base spec
func (p *ResourcePlanner) planNetworkFunction(nfSpec *telecom.NetworkFunctionSpec, llmParams map[string]interface{}, telecomContext map[string]interface{}) PlannedNetworkFunction {
	// Determine deployment configuration
	deploymentType := "production"
	if dt, ok := llmParams["deployment_type"].(string); ok {
		deploymentType = dt
	}

	deployConfig, exists := nfSpec.DeploymentPatterns[deploymentType]
	if !exists {
		// Fallback to production if specified type doesn't exist
		if prodConfig, prodExists := nfSpec.DeploymentPatterns["production"]; prodExists {
			deployConfig = prodConfig
		} else {
			// Create minimal config
			deployConfig = telecom.DeploymentConfig{
				Replicas:        1,
				ResourceProfile: "medium",
			}
		}
	}

	// Adjust replicas based on scaling requirements
	replicas := deployConfig.Replicas
	if scalingReq, ok := llmParams["scaling_requirements"].(map[string]interface{}); ok {
		if minReplicas, ok := scalingReq["min_replicas"].(float64); ok {
			replicas = int(minReplicas)
		}
	}

	// Build configuration parameters
	configuration := make(map[string]interface{})
	for paramName, param := range nfSpec.Configuration {
		configuration[paramName] = param.Default
	}

	// Override with any specific configuration from LLM
	if nfConfig, ok := llmParams["configuration"].(map[string]interface{}); ok {
		for key, value := range nfConfig {
			configuration[key] = value
		}
	}

	// Build health checks
	var healthChecks []HealthCheckSpec
	for _, hc := range nfSpec.HealthChecks {
		healthChecks = append(healthChecks, HealthCheckSpec{
			Type:     hc.Type,
			Path:     hc.Path,
			Port:     hc.Port,
			Interval: hc.Interval,
			Timeout:  hc.Timeout,
			Retries:  hc.Retries,
		})
	}

	// Build monitoring spec
	monitoring := MonitoringSpec{
		Enabled:    true,
		Metrics:    []string{},
		Alerts:     []string{},
		Dashboards: []string{},
	}

	for _, metric := range nfSpec.MonitoringMetrics {
		monitoring.Metrics = append(monitoring.Metrics, metric.Name)
		for _, alert := range metric.Alerts {
			monitoring.Alerts = append(monitoring.Alerts, alert.Name)
		}
	}

	return PlannedNetworkFunction{
		Name:     strings.ToLower(nfSpec.Name),
		Type:     nfSpec.Type,
		Version:  nfSpec.Version,
		Replicas: replicas,
		Resources: ResourceRequirements{
			CPU:         nfSpec.Resources.MaxCPU,
			Memory:      nfSpec.Resources.MaxMemory,
			Storage:     nfSpec.Resources.Storage,
			NetworkBW:   nfSpec.Resources.NetworkBW,
			Accelerator: nfSpec.Resources.Accelerator,
		},
		Configuration: configuration,
		Dependencies:  nfSpec.Dependencies,
		Interfaces:    nfSpec.Interfaces,
		HealthChecks:  healthChecks,
		Monitoring:    monitoring,
	}
}

// calculateEstimatedCost calculates estimated monthly cost for resources
func (p *ResourcePlanner) calculateEstimatedCost(cpu, memory, storage float64) float64 {
	// Simplified cost calculation (in USD per month)
	cpuCostPerCore := 50.0  // $50 per CPU core per month
	memoryCostPerGi := 10.0 // $10 per Gi memory per month
	storageCostPerGi := 2.0 // $2 per Gi storage per month

	totalCost := (cpu * cpuCostPerCore) + (memory * memoryCostPerGi) + (storage * storageCostPerGi)
	return totalCost
}

// GenerateManifests implements Phase 3: Deployment manifest generation
func (p *ResourcePlanner) GenerateManifests(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "manifest-generation")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseManifestGeneration

	if processingCtx.ResourcePlan == nil {
		err := fmt.Errorf("resource plan not available for manifest generation")
		// Set Ready condition to indicate missing resource plan
		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "ResourcePlanNotAvailable", "Resource plan is not available for manifest generation")
		return ctrl.Result{}, err
	}

	manifestMap := make(map[string]string)

	// Generate manifests for each network function
	for _, nf := range processingCtx.ResourcePlan.NetworkFunctions {
		// Generate Deployment manifest
		deployment := p.generateDeploymentManifest(networkIntent, &nf)
		deploymentYaml, err := yaml.Marshal(deployment)
		if err != nil {
			return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal deployment manifest: %w", err)
		}
		manifestMap[fmt.Sprintf("deployment-%s.yaml", nf.Name)] = string(deploymentYaml)

		// Generate Service manifest
		service := p.generateServiceManifest(networkIntent, &nf)
		serviceYaml, err := yaml.Marshal(service)
		if err != nil {
			return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal service manifest: %w", err)
		}
		manifestMap[fmt.Sprintf("service-%s.yaml", nf.Name)] = string(serviceYaml)

		// Generate ConfigMap if needed
		if len(nf.Configuration) > 0 {
			configMap := p.generateConfigMapManifest(networkIntent, &nf)
			configMapYaml, err := yaml.Marshal(configMap)
			if err != nil {
				return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal configmap manifest: %w", err)
			}
			manifestMap[fmt.Sprintf("configmap-%s.yaml", nf.Name)] = string(configMapYaml)
		}

		// Generate NetworkPolicy for security
		networkPolicy := p.generateNetworkPolicyManifest(networkIntent, &nf)
		networkPolicyYaml, err := yaml.Marshal(networkPolicy)
		if err != nil {
			return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal network policy manifest: %w", err)
		}
		manifestMap[fmt.Sprintf("networkpolicy-%s.yaml", nf.Name)] = string(networkPolicyYaml)
	}

	// Generate slice configuration if specified
	if processingCtx.ResourcePlan.SliceConfiguration != nil {
		sliceConfigMap := p.generateSliceConfigMap(networkIntent, processingCtx.ResourcePlan.SliceConfiguration)
		sliceConfigMapYaml, err := yaml.Marshal(sliceConfigMap)
		if err != nil {
			return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal slice configmap manifest: %w", err)
		}
		manifestMap["slice-configuration.yaml"] = string(sliceConfigMapYaml)
	}

	// Store manifests in processing context
	processingCtx.Manifests = manifestMap

	// Update NetworkIntent status
	condition := metav1.Condition{
		Type:               "ManifestsGenerated",
		Status:             metav1.ConditionTrue,
		Reason:             "ManifestGenerationSucceeded",
		Message:            fmt.Sprintf("Generated %d deployment manifests for network functions", len(manifestMap)),
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := p.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	generationDuration := time.Since(startTime)
	processingCtx.Metrics["manifest_generation_duration"] = generationDuration.Seconds()
	processingCtx.Metrics["generated_manifests_count"] = float64(len(manifestMap))

	p.recordEvent(networkIntent, "Normal", "ManifestGenerationSucceeded",
		fmt.Sprintf("Generated %d manifests", len(manifestMap)))
	logger.Info("Manifest generation phase completed successfully",
		"duration", generationDuration,
		"manifest_count", len(manifestMap))

	return ctrl.Result{}, nil
}

// generateDeploymentManifest generates a Kubernetes Deployment manifest for a network function
func (p *ResourcePlanner) generateDeploymentManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *appsv1.Deployment {
	labels := map[string]string{
		"app":                           nf.Name,
		"networkintent":                 networkIntent.Name,
		"networkfunction":               nf.Type,
		"nephoran.com/managed":          "true",
		"nephoran.com/network-intent":   networkIntent.Name,
		"nephoran.com/network-function": nf.Type,
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nf.Name,
			Namespace: networkIntent.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(nf.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":             nf.Name,
					"networkfunction": nf.Type,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  nf.Name,
							Image: fmt.Sprintf("%s:%s", strings.ToLower(nf.Type), nf.Version),
							Ports: p.generateContainerPorts(nf),
							Env:   p.generateEnvironmentVariables(nf),
							Resources: corev1.ResourceRequirements{
								Limits:   parseQuantity(nf.Resources.CPU + "," + nf.Resources.Memory),
								Requests: parseQuantity(nf.Resources.CPU + "," + nf.Resources.Memory),
							},
							LivenessProbe:  p.generateLivenessProbe(nf),
							ReadinessProbe: p.generateReadinessProbe(nf),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/config",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-config", nf.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

// generateServiceManifest generates a Kubernetes Service manifest for a network function
func (p *ResourcePlanner) generateServiceManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *corev1.Service {
	labels := map[string]string{
		"app":                           nf.Name,
		"networkintent":                 networkIntent.Name,
		"networkfunction":               nf.Type,
		"nephoran.com/managed":          "true",
		"nephoran.com/network-intent":   networkIntent.Name,
		"nephoran.com/network-function": nf.Type,
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service", nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":             nf.Name,
				"networkfunction": nf.Type,
			},
			Ports: p.generateServicePorts(nf),
			Type:  corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

// generateConfigMapManifest generates a ConfigMap for network function configuration
func (p *ResourcePlanner) generateConfigMapManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *corev1.ConfigMap {
	labels := map[string]string{
		"app":                           nf.Name,
		"networkintent":                 networkIntent.Name,
		"networkfunction":               nf.Type,
		"nephoran.com/managed":          "true",
		"nephoran.com/network-intent":   networkIntent.Name,
		"nephoran.com/network-function": nf.Type,
	}

	// Convert configuration to string values
	data := make(map[string]string)
	for key, value := range nf.Configuration {
		if strValue, ok := value.(string); ok {
			data[key] = strValue
		} else {
			jsonValue, _ := json.Marshal(value)
			data[key] = string(jsonValue)
		}
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
		},
		Data: data,
	}

	return configMap
}

// generateNetworkPolicyManifest generates a NetworkPolicy for security
func (p *ResourcePlanner) generateNetworkPolicyManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *networkingv1.NetworkPolicy {
	labels := map[string]string{
		"app":                           nf.Name,
		"networkintent":                 networkIntent.Name,
		"networkfunction":               nf.Type,
		"nephoran.com/managed":          "true",
		"nephoran.com/network-intent":   networkIntent.Name,
		"nephoran.com/network-function": nf.Type,
	}

	networkPolicy := &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-netpol", nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":             nf.Name,
					"networkfunction": nf.Type,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: p.generateIngressRules(nf),
			Egress:  p.generateEgressRules(nf),
		},
	}

	return networkPolicy
}

// generateSliceConfigMap generates a ConfigMap for network slice configuration
func (p *ResourcePlanner) generateSliceConfigMap(networkIntent *nephoranv1.NetworkIntent, sliceConfig *SliceConfiguration) *corev1.ConfigMap {
	labels := map[string]string{
		"networkintent":               networkIntent.Name,
		"slice-type":                  sliceConfig.SliceType,
		"nephoran.com/managed":        "true",
		"nephoran.com/network-intent": networkIntent.Name,
		"nephoran.com/slice-config":   "true",
	}

	// Convert slice configuration to YAML
	sliceYaml, _ := yaml.Marshal(sliceConfig)

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-slice-config", networkIntent.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"slice-config.yaml": string(sliceYaml),
			"sst":               fmt.Sprintf("%d", sliceConfig.SST),
			"qos-profile":       sliceConfig.QoSProfile,
			"isolation":         sliceConfig.Isolation,
		},
	}

	return configMap
}

// Helper functions for manifest generation
func (p *ResourcePlanner) generateContainerPorts(nf *PlannedNetworkFunction) []corev1.ContainerPort {
	var ports []corev1.ContainerPort

	// Default ports based on network function type
	switch strings.ToLower(nf.Type) {
	case "amf":
		ports = append(ports, corev1.ContainerPort{Name: "n2", ContainerPort: 8080, Protocol: corev1.ProtocolTCP})
		ports = append(ports, corev1.ContainerPort{Name: "n11", ContainerPort: 8081, Protocol: corev1.ProtocolTCP})
	case "smf":
		ports = append(ports, corev1.ContainerPort{Name: "n4", ContainerPort: 8080, Protocol: corev1.ProtocolTCP})
		ports = append(ports, corev1.ContainerPort{Name: "n7", ContainerPort: 8081, Protocol: corev1.ProtocolTCP})
	case "upf":
		ports = append(ports, corev1.ContainerPort{Name: "n3", ContainerPort: 8080, Protocol: corev1.ProtocolTCP})
		ports = append(ports, corev1.ContainerPort{Name: "n6", ContainerPort: 8081, Protocol: corev1.ProtocolTCP})
	default:
		ports = append(ports, corev1.ContainerPort{Name: "api", ContainerPort: 8080, Protocol: corev1.ProtocolTCP})
	}

	// Add management port
	ports = append(ports, corev1.ContainerPort{Name: "mgmt", ContainerPort: 9090, Protocol: corev1.ProtocolTCP})

	return ports
}

func (p *ResourcePlanner) generateEnvironmentVariables(nf *PlannedNetworkFunction) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	// Add common environment variables
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NF_TYPE",
		Value: nf.Type,
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NF_NAME",
		Value: nf.Name,
	})
	envVars = append(envVars, corev1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
	envVars = append(envVars, corev1.EnvVar{
		Name: "POD_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})

	return envVars
}

func (p *ResourcePlanner) generateLivenessProbe(nf *PlannedNetworkFunction) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.FromString("mgmt"),
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    3,
	}
}

func (p *ResourcePlanner) generateReadinessProbe(nf *PlannedNetworkFunction) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ready",
				Port: intstr.FromString("mgmt"),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		FailureThreshold:    3,
	}
}

func (p *ResourcePlanner) generateServicePorts(nf *PlannedNetworkFunction) []corev1.ServicePort {
	var ports []corev1.ServicePort

	// Generate ports based on container ports
	containerPorts := p.generateContainerPorts(nf)
	for _, cp := range containerPorts {
		ports = append(ports, corev1.ServicePort{
			Name:       cp.Name,
			Port:       cp.ContainerPort,
			TargetPort: intstr.FromString(cp.Name),
			Protocol:   cp.Protocol,
		})
	}

	return ports
}

func (p *ResourcePlanner) generateIngressRules(nf *PlannedNetworkFunction) []networkingv1.NetworkPolicyIngressRule {
	var rules []networkingv1.NetworkPolicyIngressRule

	// Allow traffic from same namespace
	rules = append(rules, networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": nf.Name, // This should match the namespace name
					},
				},
			},
		},
	})

	return rules
}

func (p *ResourcePlanner) generateEgressRules(nf *PlannedNetworkFunction) []networkingv1.NetworkPolicyEgressRule {
	var rules []networkingv1.NetworkPolicyEgressRule

	// Allow all egress (can be restricted based on requirements)
	rules = append(rules, networkingv1.NetworkPolicyEgressRule{})

	return rules
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

// Interface implementation methods

// ProcessPhase implements PhaseProcessor interface
func (p *ResourcePlanner) ProcessPhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	// This processor handles both resource planning and manifest generation
	// Check which phase needs to be executed based on conditions
	if !isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") {
		return p.PlanResources(ctx, networkIntent, processingCtx)
	} else if !isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") {
		return p.GenerateManifests(ctx, networkIntent, processingCtx)
	}
	return ctrl.Result{}, nil
}

// GetPhaseName implements PhaseProcessor interface
func (p *ResourcePlanner) GetPhaseName() ProcessingPhase {
	return PhaseResourcePlanning
}

// IsPhaseComplete implements PhaseProcessor interface
func (p *ResourcePlanner) IsPhaseComplete(networkIntent *nephoranv1.NetworkIntent) bool {
	return isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") &&
		isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated")
}

// PlanNetworkFunction implements ResourcePlannerInterface (expose existing method)
func (p *ResourcePlanner) PlanNetworkFunction(nfName string, llmParams map[string]interface{}, telecomContext map[string]interface{}) (*PlannedNetworkFunction, error) {
	kb := p.deps.GetTelecomKnowledgeBase()
	if kb == nil {
		return nil, fmt.Errorf("telecom knowledge base not available")
	}

	nfSpec, exists := kb.GetNetworkFunction(nfName)
	if !exists {
		return nil, fmt.Errorf("network function spec not found: %s", nfName)
	}

	plannedNF := p.planNetworkFunction(nfSpec, llmParams, telecomContext)
	return &plannedNF, nil
}

// CalculateEstimatedCost implements ResourcePlannerInterface
func (p *ResourcePlanner) CalculateEstimatedCost(plan *ResourcePlan) (float64, error) {
	if plan == nil {
		return 0, fmt.Errorf("resource plan cannot be nil")
	}

	var totalCPU, totalMemory, totalStorage float64

	for _, nf := range plan.NetworkFunctions {
		// Parse CPU
		var cpu float64
		fmt.Sscanf(nf.Resources.CPU, "%f", &cpu)
		totalCPU += cpu * float64(nf.Replicas)

		// Parse Memory
		var memory float64
		fmt.Sscanf(nf.Resources.Memory, "%fGi", &memory)
		totalMemory += memory * float64(nf.Replicas)

		// Parse Storage
		var storage float64
		fmt.Sscanf(nf.Resources.Storage, "%fGi", &storage)
		totalStorage += storage * float64(nf.Replicas)
	}

	return p.calculateEstimatedCost(totalCPU, totalMemory, totalStorage), nil
}
