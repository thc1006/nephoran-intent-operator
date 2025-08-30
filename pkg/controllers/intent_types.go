package controllers

import (
	"encoding/json"
)

// StructuredIntent is the general structure returned by the LLM processor.

type StructuredIntent struct {
	Type string `json:"type"`

	Data json.RawMessage `json:"data"`
}

// NetworkFunctionDeploymentIntent represents the detailed intent for deploying a new network function.

type NetworkFunctionDeploymentIntent struct {
	Type string `json:"type"`

	Name string `json:"name"`

	Namespace string `json:"namespace"`

	Spec DeploymentSpec `json:"spec"`

	O1Config string `json:"o1_config"`

	A1Policy A1Policy `json:"a1_policy"`
}

// DeploymentSpec defines the specifications for the Kubernetes Deployment.

type DeploymentSpec struct {
	Replicas int `json:"replicas"`

	Image string `json:"image"`

	Resources map[string]interface{} `json:"resources"`
}

// A1Policy defines the structure for an A1 policy.

type A1Policy struct {
	PolicyTypeID string `json:"policy_type_id"`

	PolicyData map[string]interface{} `json:"policy_data"`
}

// NetworkFunctionScaleIntent represents the detailed intent for scaling an existing network function.

type NetworkFunctionScaleIntent struct {
	Type string `json:"type"`

	Name string `json:"name"`

	Namespace string `json:"namespace"`

	Replicas int `json:"replicas"`
}
