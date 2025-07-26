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
