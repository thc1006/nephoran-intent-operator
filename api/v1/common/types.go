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

// Package common provides shared types and utilities for the Nephoran Intent Operator API.
// This package includes common deployment strategies, health check configurations,
// secret references, and health status types used across the operator components.
package common

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SharedDeploymentStrategy consolidates deployment strategies across different resource types.

type SharedDeploymentStrategy struct {
	Type string `json:"type"`

	Rolling *RollingUpdateConfig `json:"rolling,omitempty"`

	Recreate bool `json:"recreate,omitempty"`
}

// RollingUpdateConfig defines rolling update configuration.

type RollingUpdateConfig struct {
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`

	MaxSurge *int32 `json:"maxSurge,omitempty"`
}

// SharedHealthCheckConfig consolidates health check configurations.

type SharedHealthCheckConfig struct {
	Enabled bool `json:"enabled"`

	Path string `json:"path,omitempty"`

	Port int32 `json:"port,omitempty"`

	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`

	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	FailureThreshold int32 `json:"failureThreshold,omitempty"`
}

// SharedSecretReference consolidates secret references across different resource types.

type SharedSecretReference struct {
	Name string `json:"name"`

	Namespace string `json:"namespace,omitempty"`

	Key string `json:"key,omitempty"`
}

// SharedHealthStatus consolidates health status definitions.

type SharedHealthStatus struct {
	Phase string `json:"phase"`

	Message string `json:"message,omitempty"`

	Reason string `json:"reason,omitempty"`

	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
}
