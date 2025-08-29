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

package dependencies

import (
	"time"
)

// Basic types that are missing from other files

// UsageData represents usage data for packages.
type UsageData struct {
	Package   *PackageReference `json:"package"`
	Usage     int64             `json:"usage"`
	Timestamp time.Time         `json:"timestamp"`
}

// Additional basic types needed for compilation that are NOT already defined elsewhere
type GraphNode struct{
	PackageRef *PackageReference `json:"packageRef"`
}
type GraphMetrics struct{}
type DependencyGraph struct{
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
type CostTrend string
type HealthGrade string
type OptimizationStrategy string
type HealthTrend string
type PackageRegistryConfig struct{}
type UpdateStrategy string
type UpdateType string
type GraphEdge struct{}
type RolloutStatus string
type UpdateStep struct{}
type DependencyUpdate struct{}