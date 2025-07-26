package controllers

import "k8s.io/apimachinery/pkg/runtime"

// ... (NetworkFunctionDeployment structs remain the same) ...

// NetworkFunctionScaleIntent defines the structured parameters for scaling a network function.
type NetworkFunctionScaleIntent struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int    `json:"replicas"`
}

// GenericIntent is used to unmarshal the 'type' field to determine the intent.
type GenericIntent struct {
	Type string `json:"type"`
}