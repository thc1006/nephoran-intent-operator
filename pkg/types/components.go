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

package coretypes

import (
	"github.com/thc1006/nephoran-intent-operator/pkg/contracts"
	"time"
)

// ComponentStatus is an alias to the contracts package for backward compatibility
type ComponentStatus = contracts.ComponentStatus

// SystemHealth represents the overall health of the system
type SystemHealth struct {
	OverallStatus  string                      `json:"overallStatus"`
	Healthy        bool                        `json:"healthy"`
	Components     map[string]*ComponentStatus `json:"components"`
	LastUpdate     time.Time                   `json:"lastUpdate"`
	ActiveIntents  int                         `json:"activeIntents"`
	ProcessingRate float64                     `json:"processingRate"`
	ErrorRate      float64                     `json:"errorRate"`
	ResourceUsage  ResourceUsage               `json:"resourceUsage"`
}

// ResourceUsage represents resource utilization
type ResourceUsage struct {
	CPUPercent        float64 `json:"cpuPercent"`
	MemoryPercent     float64 `json:"memoryPercent"`
	DiskPercent       float64 `json:"diskPercent"`
	NetworkInMBps     float64 `json:"networkInMBps"`
	NetworkOutMBps    float64 `json:"networkOutMBps"`
	ActiveConnections int     `json:"activeConnections"`
}
