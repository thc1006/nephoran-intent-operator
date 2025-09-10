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

// Package v1 provides API types for Cloud Native Network Function (CNF) deployment
// in the Nephoran Intent Operator. This package defines custom resources and
// specifications for managing the lifecycle, scaling, and configuration of
// telecommunications network functions in Kubernetes environments, supporting
// O-RAN and 5G network orchestration requirements.
package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CNFType defines the type of Cloud Native Function.
// +kubebuilder:validation:Enum=Core5G;ORAN;Edge;Custom

type CNFType string

const (

	// CNF5GCore represents 5G Core network functions.

	CNF5GCore CNFType = "5G-Core"

	// CNFORAN represents O-RAN network functions.

	CNFORAN CNFType = "O-RAN"

	// CNFEdge represents Edge computing network functions.

	CNFEdge CNFType = "Edge"

	// CNFCustom represents custom telecommunications functions.

	CNFCustom CNFType = "Custom"
)

// CNFFunction defines specific CNF function types.

// +kubebuilder:validation:Enum=AMF;SMF;UPF;NRF;AUSF;UDM;PCF;NSSF;NEF;SMSF;BSF;UDR;UDSF;CHF;N3IWF;TNGF;TWIF;NWDAF;SCP;SEPP;O-DU;O-CU-CP;O-CU-UP;Near-RT-RIC;Non-RT-RIC;O-eNB;SMO;rApp;xApp;O-FH;O-M-P;UE-Simulator;Traffic-Generator

type CNFFunction string

const (

	// 5G Core Functions.

	CNFFunctionAMF CNFFunction = "AMF"

	// CNFFunctionSMF holds cnffunctionsmf value.

	CNFFunctionSMF CNFFunction = "SMF"

	// CNFFunctionUPF holds cnffunctionupf value.

	CNFFunctionUPF CNFFunction = "UPF"

	// CNFFunctionNRF holds cnffunctionnrf value.

	CNFFunctionNRF CNFFunction = "NRF"

	// CNFFunctionAUSF holds cnffunctionausf value.

	CNFFunctionAUSF CNFFunction = "AUSF"

	// CNFFunctionUDM holds cnffunctionudm value.

	CNFFunctionUDM CNFFunction = "UDM"

	// CNFFunctionPCF holds cnffunctionpcf value.

	CNFFunctionPCF CNFFunction = "PCF"

	// CNFFunctionNSSF holds cnffunctionnssf value.

	CNFFunctionNSSF CNFFunction = "NSSF"

	// CNFFunctionNEF holds cnffunctionnef value.

	CNFFunctionNEF CNFFunction = "NEF"

	// CNFFunctionSMSF holds cnffunctionsmsf value.

	CNFFunctionSMSF CNFFunction = "SMSF"

	// CNFFunctionBSF holds cnffunctionbsf value.

	CNFFunctionBSF CNFFunction = "BSF"

	// CNFFunctionUDR holds cnffunctionudr value.

	CNFFunctionUDR CNFFunction = "UDR"

	// CNFFunctionUDSF holds cnffunctionudsf value.

	CNFFunctionUDSF CNFFunction = "UDSF"

	// CNFFunctionCHF holds cnffunctionchf value.

	CNFFunctionCHF CNFFunction = "CHF"

	// CNFFunctionN3IWF holds cnffunctionn3iwf value.

	CNFFunctionN3IWF CNFFunction = "N3IWF"

	// CNFFunctionTNGF holds cnffunctiontngf value.

	CNFFunctionTNGF CNFFunction = "TNGF"

	// CNFFunctionTWIF holds cnffunctiontwif value.

	CNFFunctionTWIF CNFFunction = "TWIF"

	// CNFFunctionNWDAF holds cnffunctionnwdaf value.

	CNFFunctionNWDAF CNFFunction = "NWDAF"

	// CNFFunctionSCP holds cnffunctionscp value.

	CNFFunctionSCP CNFFunction = "SCP"

	// CNFFunctionSEPP holds cnffunctionsepp value.

	CNFFunctionSEPP CNFFunction = "SEPP"

	// O-RAN Functions.

	CNFFunctionODU CNFFunction = "O-DU"

	// CNFFunctionOCUCP holds cnffunctionocucp value.

	CNFFunctionOCUCP CNFFunction = "O-CU-CP"

	// CNFFunctionOCUUP holds cnffunctionocuup value.

	CNFFunctionOCUUP CNFFunction = "O-CU-UP"

	// CNFFunctionNearRTRIC holds cnffunctionnearrtric value.

	CNFFunctionNearRTRIC CNFFunction = "Near-RT-RIC"

	// CNFFunctionNonRTRIC holds cnffunctionnonrtric value.

	CNFFunctionNonRTRIC CNFFunction = "Non-RT-RIC"

	// CNFFunctionOENB holds cnffunctionoenb value.

	CNFFunctionOENB CNFFunction = "O-eNB"

	// CNFFunctionSMO holds cnffunctionsmo value.

	CNFFunctionSMO CNFFunction = "SMO"

	// CNFFunctionRApp holds cnffunctionrapp value.

	CNFFunctionRApp CNFFunction = "rApp"

	// CNFFunctionXApp holds cnffunctionxapp value.

	CNFFunctionXApp CNFFunction = "xApp"

	// CNFFunctionOFH holds cnffunctionofh value.

	CNFFunctionOFH CNFFunction = "O-FH"

	// CNFFunctionOMP holds cnffunctionomp value.

	CNFFunctionOMP CNFFunction = "O-M-P"

	// Testing and Support Functions.

	CNFFunctionUESimulator CNFFunction = "UE-Simulator"

	// CNFFunctionTrafficGenerator holds cnffunctiontrafficgenerator value.

	CNFFunctionTrafficGenerator CNFFunction = "Traffic-Generator"
)

// CNFDeploymentStrategy defines how the CNF should be deployed.

// +kubebuilder:validation:Enum=Helm;Operator;Direct;GitOps

type CNFDeploymentStrategy string

const (

	// CNFDeploymentStrategyHelm holds cnfdeploymentstrategyhelm value.

	CNFDeploymentStrategyHelm CNFDeploymentStrategy = "Helm"

	// CNFDeploymentStrategyOperator holds cnfdeploymentstrategyoperator value.

	CNFDeploymentStrategyOperator CNFDeploymentStrategy = "Operator"

	// CNFDeploymentStrategyDirect holds cnfdeploymentstrategydirect value.

	CNFDeploymentStrategyDirect CNFDeploymentStrategy = "Direct"

	// CNFDeploymentStrategyGitOps holds cnfdeploymentstrategygitops value.

	CNFDeploymentStrategyGitOps CNFDeploymentStrategy = "GitOps"
)

// CNFResources defines resource requirements for CNF.

type CNFResources struct {
	// CPU resource requirements.

	CPU resource.Quantity `json:"cpu"`

	// Memory resource requirements.

	Memory resource.Quantity `json:"memory"`

	// Storage resource requirements.

	// +optional

	Storage *resource.Quantity `json:"storage,omitempty"`

	// Maximum CPU resource limit.

	// +optional

	MaxCPU *resource.Quantity `json:"maxCpu,omitempty"`

	// Maximum Memory resource limit.

	// +optional

	MaxMemory *resource.Quantity `json:"maxMemory,omitempty"`

	// GPU resource requirements.

	// +optional

	GPU *int32 `json:"gpu,omitempty"`

	// Hugepages requirements for high-performance networking.

	// +optional

	Hugepages map[string]resource.Quantity `json:"hugepages,omitempty"`

	// DPDK requirements for packet processing.

	// +optional

	DPDK *DPDKConfig `json:"dpdk,omitempty"`
}

// DPDKConfig defines DPDK-specific configuration.

type DPDKConfig struct {
	// Enable DPDK support.

	Enabled bool `json:"enabled"`

	// Number of DPDK cores.

	// +optional

	Cores *int32 `json:"cores,omitempty"`

	// DPDK memory in MB.

	// +optional

	Memory *int32 `json:"memory,omitempty"`

	// DPDK driver.

	// +optional

	Driver string `json:"driver,omitempty"`
}

// HelmConfig defines Helm chart configuration.

type HelmConfig struct {
	// Chart repository URL.

	// +kubebuilder:validation:Pattern=`^https?://*`

	Repository string `json:"repository"`

	// Chart name.

	ChartName string `json:"chartName"`

	// Chart version.

	ChartVersion string `json:"chartVersion"`

	// Values override the default chart values.

	// +optional

	// +kubebuilder:pruning:PreserveUnknownFields

	Values runtime.RawExtension `json:"values,omitempty"`

	// Release name for the Helm deployment.

	// +optional

	ReleaseName string `json:"releaseName,omitempty"`
}

// OperatorConfig defines operator-based deployment configuration.

type OperatorConfig struct {
	// Operator name.

	Name string `json:"name"`

	// Operator namespace.

	Namespace string `json:"namespace"`

	// Custom resource definition for the operator.

	// +kubebuilder:pruning:PreserveUnknownFields

	CustomResource runtime.RawExtension `json:"customResource"`
}

// ServiceMeshConfig defines service mesh integration.

type ServiceMeshConfig struct {
	// Enable service mesh integration.

	Enabled bool `json:"enabled"`

	// Service mesh type (Istio, Linkerd, Consul Connect).

	// +optional

	// +kubebuilder:validation:Enum=Istio;Linkerd;Consul

	Type string `json:"type,omitempty"`

	// mTLS configuration.

	// +optional

	MTLS *MTLSConfig `json:"mtls,omitempty"`

	// Traffic policies.

	// +optional

	TrafficPolicies []TrafficPolicy `json:"trafficPolicies,omitempty"`
}

// MTLSConfig defines mutual TLS configuration.

type MTLSConfig struct {
	// Enable mTLS.

	Enabled bool `json:"enabled"`

	// mTLS mode (strict, permissive).

	// +optional

	// +kubebuilder:validation:Enum=strict;permissive

	Mode string `json:"mode,omitempty"`
}

// TrafficPolicy defines traffic routing policies.

type TrafficPolicy struct {
	// Policy name.

	Name string `json:"name"`

	// Source service.

	Source string `json:"source"`

	// Destination service.

	Destination string `json:"destination"`

	// Load balancing configuration.

	// +optional

	LoadBalancing *LoadBalancingConfig `json:"loadBalancing,omitempty"`
}

// LoadBalancingConfig defines load balancing configuration.

type LoadBalancingConfig struct {
	// Load balancing algorithm.

	// +kubebuilder:validation:Enum=round_robin;least_conn;random;ip_hash

	Algorithm string `json:"algorithm"`

	// Health check configuration.

	// +optional

	HealthCheck *CNFHealthCheckConfig `json:"healthCheck,omitempty"`
}

// CNFHealthCheckConfig defines health check configuration.

type CNFHealthCheckConfig struct {
	// Path for HTTP health checks.

	Path string `json:"path"`

	// Port for health checks.

	Port int32 `json:"port"`

	// Check interval in seconds.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=300

	Interval int32 `json:"interval"`

	// Timeout in seconds.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=60

	Timeout int32 `json:"timeout"`

	// Number of retries.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=10

	Retries int32 `json:"retries"`
}

// AutoScaling defines auto-scaling configuration.

type AutoScaling struct {
	// Enable auto-scaling.

	Enabled bool `json:"enabled"`

	// Minimum number of replicas.

	// +kubebuilder:validation:Minimum=1

	MinReplicas int32 `json:"minReplicas"`

	// Maximum number of replicas.

	// +kubebuilder:validation:Minimum=1

	MaxReplicas int32 `json:"maxReplicas"`

	// CPU utilization threshold for scaling.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=100

	CPUUtilization *int32 `json:"cpuUtilization,omitempty"`

	// Memory utilization threshold for scaling.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=100

	MemoryUtilization *int32 `json:"memoryUtilization,omitempty"`

	// Custom metrics for scaling.

	// +optional

	CustomMetrics []CustomMetric `json:"customMetrics,omitempty"`
}

// CustomMetric defines custom metrics for auto-scaling.

type CustomMetric struct {
	// Metric name.

	Name string `json:"name"`

	// Metric type (pods, object, external).

	// +kubebuilder:validation:Enum=pods;object;external

	Type string `json:"type"`

	// Target value.

	TargetValue string `json:"targetValue"`
}

// CNFDeploymentSpec defines the desired state of CNFDeployment.

type CNFDeploymentSpec struct {
	// CNF type classification.

	// +kubebuilder:validation:Required

	CNFType CNFType `json:"cnfType"`

	// Specific CNF function.

	// +kubebuilder:validation:Required

	Function CNFFunction `json:"function"`

	// Deployment strategy.

	// +kubebuilder:validation:Required

	// +kubebuilder:default="Helm"

	DeploymentStrategy CNFDeploymentStrategy `json:"deploymentStrategy"`

	// Number of replicas.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=100

	// +kubebuilder:default=1

	Replicas int32 `json:"replicas"`

	// Resource requirements.

	// +kubebuilder:validation:Required

	Resources CNFResources `json:"resources"`

	// Helm configuration (if using Helm strategy).

	// +optional

	Helm *HelmConfig `json:"helm,omitempty"`

	// Operator configuration (if using Operator strategy).

	// +optional

	Operator *OperatorConfig `json:"operator,omitempty"`

	// Service mesh integration.

	// +optional

	ServiceMesh *ServiceMeshConfig `json:"serviceMesh,omitempty"`

	// Auto-scaling configuration.

	// +optional

	AutoScaling *AutoScaling `json:"autoScaling,omitempty"`

	// Network slice identifier.

	// +optional

	// +kubebuilder:validation:Pattern=`^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$`

	NetworkSlice string `json:"networkSlice,omitempty"`

	// Target cluster for deployment.

	// +optional

	TargetCluster string `json:"targetCluster,omitempty"`

	// Target namespace for deployment.

	// +optional

	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`

	// +kubebuilder:validation:MaxLength=63

	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Configuration parameters.

	// +optional

	// +kubebuilder:pruning:PreserveUnknownFields

	Configuration runtime.RawExtension `json:"configuration,omitempty"`

	// Security policies.

	// +optional

	SecurityPolicies []string `json:"securityPolicies,omitempty"`

	// Monitoring configuration.

	// +optional

	Monitoring *MonitoringConfig `json:"monitoring,omitempty"`

	// Backup configuration.

	// +optional

	Backup *BackupConfig `json:"backup,omitempty"`
}

// MonitoringConfig defines monitoring configuration.

type MonitoringConfig struct {
	// Enable monitoring.

	Enabled bool `json:"enabled"`

	// Prometheus scraping configuration.

	// +optional

	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`

	// Custom metrics to collect.

	// +optional

	CustomMetrics []string `json:"customMetrics,omitempty"`

	// Alerting rules.

	// +optional

	AlertingRules []string `json:"alertingRules,omitempty"`
}

// PrometheusConfig defines Prometheus configuration.

type PrometheusConfig struct {
	// Enable Prometheus scraping.

	Enabled bool `json:"enabled"`

	// Scraping path.

	// +optional

	// +kubebuilder:default="/metrics"

	Path string `json:"path,omitempty"`

	// Scraping port.

	// +optional

	// +kubebuilder:default=9090

	Port int32 `json:"port,omitempty"`

	// Scraping interval.

	// +optional

	// +kubebuilder:default="30s"

	Interval string `json:"interval,omitempty"`
}

// BackupConfig defines backup configuration.

type BackupConfig struct {
	// Enable backup.

	Enabled bool `json:"enabled"`

	// Backup schedule (cron format).

	// +optional

	Schedule string `json:"schedule,omitempty"`

	// Retention policy in days.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=365

	RetentionDays *int32 `json:"retentionDays,omitempty"`

	// Backup storage location.

	// +optional

	StorageLocation string `json:"storageLocation,omitempty"`
}

// CNFDeploymentStatus defines the observed state of CNFDeployment.

type CNFDeploymentStatus struct {
	// Conditions represent the latest available observations of the deployment's state.

	// +optional

	// +listType=atomic
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current deployment phase.

	// +optional

	// +kubebuilder:validation:Enum=Pending;Deploying;Running;Scaling;Upgrading;Failed;Terminating

	Phase string `json:"phase,omitempty"`

	// Ready replicas count.

	// +optional

	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Available replicas count.

	// +optional

	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Deployment start time.

	// +optional

	DeploymentStartTime *metav1.Time `json:"deploymentStartTime,omitempty"`

	// Last updated time.

	// +optional

	LastUpdatedTime *metav1.Time `json:"lastUpdatedTime,omitempty"`

	// Deployed Helm release name.

	// +optional

	HelmRelease string `json:"helmRelease,omitempty"`

	// Resource utilization metrics.

	// +optional

	ResourceUtilization map[string]float64 `json:"resourceUtilization,omitempty"`

	// Health status.

	// +optional

	Health *CNFHealthStatus `json:"health,omitempty"`

	// Service endpoints.

	// +optional

	ServiceEndpoints []ServiceEndpoint `json:"serviceEndpoints,omitempty"`

	// Observed generation.

	// +optional

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// CNFHealthStatus defines the health status.

type CNFHealthStatus struct {
	// Overall health status.

	// +kubebuilder:validation:Enum=Healthy;Degraded;Unhealthy;Unknown

	Status string `json:"status"`

	// Last health check time.

	// +optional

	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// Health check details.

	// +optional

	Details map[string]string `json:"details,omitempty"`
}

// ServiceEndpoint defines service endpoint information.

type ServiceEndpoint struct {
	// Service name.

	Name string `json:"name"`

	// Service type.

	Type string `json:"type"`

	// Cluster IP.

	// +optional

	ClusterIP string `json:"clusterIP,omitempty"`

	// External IP (for LoadBalancer services).

	// +optional

	ExternalIP string `json:"externalIP,omitempty"`

	// Ports.

	Ports []ServicePort `json:"ports"`
}

// ServicePort defines service port information.

type ServicePort struct {
	// Port name.

	Name string `json:"name"`

	// Port number.

	Port int32 `json:"port"`

	// Target port.

	TargetPort string `json:"targetPort"`

	// Protocol.

	Protocol string `json:"protocol"`
}

//+kubebuilder:object:root=true

//+kubebuilder:subresource:status

//+kubebuilder:subresource:scale:specpath=spec.replicas,statuspath=status.readyReplicas

//+kubebuilder:printcolumn:name="CNF Type",type=string,JSONPath=`.spec.cnfType`

//+kubebuilder:printcolumn:name="Function",type=string,JSONPath=`.spec.function`

//+kubebuilder:printcolumn:name="Strategy",type=string,JSONPath=`.spec.deploymentStrategy`

//+kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`

//+kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`

//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

//+kubebuilder:resource:shortName=cnfd;cnf

//+kubebuilder:storageversion

// CNFDeployment is the Schema for the cnfdeployments API.

type CNFDeployment struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CNFDeploymentSpec `json:"spec,omitempty"`

	Status CNFDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CNFDeploymentList contains a list of CNFDeployment.

type CNFDeploymentList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CNFDeployment `json:"items"`
}

// ValidateCNFDeployment validates the CNF deployment specification.

func (cnf *CNFDeployment) ValidateCNFDeployment() error {
	// Preallocate slice with expected capacity for performance.

	errors := make([]string, 0, 4)

	// Validate function compatibility with CNF type.
	if err := cnf.validateFunctionCompatibility(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate deployment strategy configuration.
	if err := cnf.validateDeploymentStrategy(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate resource requirements.
	if err := cnf.validateResources(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate auto-scaling configuration.
	if err := cnf.validateAutoScaling(); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateFunctionCompatibility ensures CNF function is compatible with CNF type.
func (cnf *CNFDeployment) validateFunctionCompatibility() error {
	compatibilityMap := map[CNFType][]CNFFunction{
		CNF5GCore: {
			CNFFunctionAMF, CNFFunctionSMF, CNFFunctionUPF, CNFFunctionNRF,

			CNFFunctionAUSF, CNFFunctionUDM, CNFFunctionPCF, CNFFunctionNSSF,

			CNFFunctionNEF, CNFFunctionSMSF, CNFFunctionBSF, CNFFunctionUDR,

			CNFFunctionUDSF, CNFFunctionCHF, CNFFunctionN3IWF, CNFFunctionTNGF,

			CNFFunctionTWIF, CNFFunctionNWDAF, CNFFunctionSCP, CNFFunctionSEPP,
		},

		CNFORAN: {
			CNFFunctionODU, CNFFunctionOCUCP, CNFFunctionOCUUP,

			CNFFunctionNearRTRIC, CNFFunctionNonRTRIC, CNFFunctionOENB,

			CNFFunctionSMO, CNFFunctionRApp, CNFFunctionXApp,

			CNFFunctionOFH, CNFFunctionOMP,
		},

		CNFEdge: {
			CNFFunctionUESimulator, CNFFunctionTrafficGenerator,
		},

		CNFCustom: {}, // Custom type allows any function
	}

	if cnf.Spec.CNFType != CNFCustom {
		compatible, exists := compatibilityMap[cnf.Spec.CNFType]

		if !exists {
			return fmt.Errorf("unknown CNF type: %s", cnf.Spec.CNFType)
		}

		for _, validFunction := range compatible {
			if cnf.Spec.Function == validFunction {
				return nil
			}
		}
		return fmt.Errorf("function %s is not compatible with CNF type %s", cnf.Spec.Function, cnf.Spec.CNFType)
	}

	return nil
}

// validateDeploymentStrategy ensures deployment strategy configuration is valid.
func (cnf *CNFDeployment) validateDeploymentStrategy() error {
	switch cnf.Spec.DeploymentStrategy {

	case CNFDeploymentStrategyHelm:
		if cnf.Spec.Helm == nil {
			return fmt.Errorf("helm configuration is required for Helm deployment strategy")
		}

		if cnf.Spec.Helm.Repository == "" || cnf.Spec.Helm.ChartName == "" {
			return fmt.Errorf("helm repository and chartName are required")
		}

	case CNFDeploymentStrategyOperator:
		if cnf.Spec.Operator == nil {
			return fmt.Errorf("operator configuration is required for Operator deployment strategy")
		}

		if cnf.Spec.Operator.Name == "" || cnf.Spec.Operator.Namespace == "" {
			return fmt.Errorf("operator name and namespace are required")
		}

	}

	return nil
}

// validateResources ensures resource requirements are properly specified.

func (cnf *CNFDeployment) validateResources() error {
	// Validate max resources are greater than or equal to min resources.

	if cnf.Spec.Resources.MaxCPU != nil {
		if cnf.Spec.Resources.MaxCPU.Cmp(cnf.Spec.Resources.CPU) < 0 {
			return fmt.Errorf("maxCpu must be greater than or equal to cpu")
		}
	}

	if cnf.Spec.Resources.MaxMemory != nil {
		if cnf.Spec.Resources.MaxMemory.Cmp(cnf.Spec.Resources.Memory) < 0 {
			return fmt.Errorf("maxMemory must be greater than or equal to memory")
		}
	}

	// Validate minimum resource requirements for specific functions.

	minRequirements := map[CNFFunction]struct {
		CPU int64 // milliCPU

		Memory int64 // bytes
	}{
		CNFFunctionUPF: {CPU: 2000, Memory: 4000000000}, // 2 CPU, ~4Gi memory

		CNFFunctionAMF: {CPU: 1000, Memory: 2000000000}, // 1 CPU, ~2Gi memory

		CNFFunctionSMF: {CPU: 1000, Memory: 2000000000}, // 1 CPU, ~2Gi memory

	}

	if req, exists := minRequirements[cnf.Spec.Function]; exists {

		if cnf.Spec.Resources.CPU.MilliValue() < req.CPU {
			return fmt.Errorf("function %s requires minimum %dm CPU", cnf.Spec.Function, req.CPU)
		}

		if cnf.Spec.Resources.Memory.Value() < req.Memory {
			return fmt.Errorf("function %s requires minimum %d bytes memory", cnf.Spec.Function, req.Memory)
		}

	}

	return nil
}

// validateAutoScaling ensures auto-scaling configuration is valid.

func (cnf *CNFDeployment) validateAutoScaling() error {
	if cnf.Spec.AutoScaling == nil || !cnf.Spec.AutoScaling.Enabled {
		return nil
	}

	as := cnf.Spec.AutoScaling

	// Validate replica counts.

	if as.MaxReplicas <= as.MinReplicas {
		return fmt.Errorf("maxReplicas must be greater than minReplicas")
	}

	if cnf.Spec.Replicas < as.MinReplicas || cnf.Spec.Replicas > as.MaxReplicas {
		return fmt.Errorf("replicas must be between minReplicas and maxReplicas")
	}

	// Ensure at least one scaling metric is configured.

	if as.CPUUtilization == nil && as.MemoryUtilization == nil && len(as.CustomMetrics) == 0 {
		return fmt.Errorf("at least one scaling metric must be configured")
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&CNFDeployment{}, &CNFDeploymentList{})
}
