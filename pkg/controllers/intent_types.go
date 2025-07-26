package controllers

// NetworkFunctionDeploymentSpec defines the desired state of a network function deployment.
type NetworkFunctionDeploymentSpec struct {
	Replicas  int                 `json:"replicas"`
	Image     string              `json:"image"`
	Resources ResourceRequirements `json:"resources"`
}

// ResourceRequirements defines the resource requests and limits for a container.
type ResourceRequirements struct {
	Requests ResourceList `json:"requests"`
	Limits   ResourceList `json:"limits"`
}

// ResourceList defines the CPU and memory resources.
type ResourceList struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// NetworkFunctionDeploymentIntent defines the structured parameters for a network function deployment intent.
type NetworkFunctionDeploymentIntent struct {
	Type      string                      `json:"type"`
	Name      string                      `json:"name"`
	Namespace string                      `json:"namespace"`
	Spec      NetworkFunctionDeploymentSpec `json:"spec"`
}