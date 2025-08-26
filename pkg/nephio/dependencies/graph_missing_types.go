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
	"context"
)

// Missing types referenced in graph.go

// GraphUpdate represents an update operation on a dependency graph
type GraphUpdate struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"` // "add_node", "remove_node", "add_edge", "remove_edge", "update_node", "update_edge"
	Operation string      `json:"operation"`
	NodeID    string      `json:"nodeId,omitempty"`
	EdgeID    string      `json:"edgeId,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GraphVisitor defines the interface for graph traversal visitors
type GraphVisitor interface {
	VisitNode(ctx context.Context, node *GraphNode) error
	VisitEdge(ctx context.Context, edge *GraphEdge) error
	PreVisit(ctx context.Context) error
	PostVisit(ctx context.Context) error
}

// VersionStrategy defines version handling strategies
type VersionStrategy string

const (
	VersionStrategyLatest    VersionStrategy = "latest"
	VersionStrategyStable    VersionStrategy = "stable"
	VersionStrategyExact     VersionStrategy = "exact"
	VersionStrategyMinimum   VersionStrategy = "minimum"
	VersionStrategyRange     VersionStrategy = "range"
	VersionStrategyPreferred VersionStrategy = "preferred"
)

// PackageFilter defines filtering criteria for packages
type PackageFilter struct {
	Names         []string          `json:"names,omitempty"`
	Versions      []string          `json:"versions,omitempty"`
	Types         []string          `json:"types,omitempty"`
	Scopes        []DependencyScope `json:"scopes,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Repositories  []string          `json:"repositories,omitempty"`
	Environments  []string          `json:"environments,omitempty"`
	Include       bool              `json:"include"` // true = include matching, false = exclude matching
	CaseSensitive bool              `json:"caseSensitive,omitempty"`
}

// CycleImpact and CycleBreakingOption are moved here since they're already defined elsewhere
// but being referenced in graph.go. We'll need to remove duplicates.

// Additional graph-related types that may be missing
type GraphPath struct {
	ID       string   `json:"id"`
	Nodes    []string `json:"nodes"`
	Edges    []string `json:"edges"`
	Length   int      `json:"length"`
	Weight   float64  `json:"weight"`
	Cost     float64  `json:"cost,omitempty"`
	Duration *float64 `json:"duration,omitempty"`
}

type GraphManagerConfig struct {
	MaxNodes             int     `json:"maxNodes,omitempty"`
	MaxEdges             int     `json:"maxEdges,omitempty"`
	MaxDepth             int     `json:"maxDepth,omitempty"`
	EnableCaching        bool    `json:"enableCaching"`
	EnableVisualization  bool    `json:"enableVisualization"`
	EnableOptimization   bool    `json:"enableOptimization"`
	ConcurrencyLevel     int     `json:"concurrencyLevel,omitempty"`
	AnalysisTimeout      int     `json:"analysisTimeout,omitempty"` // seconds
}

func (c *GraphManagerConfig) Validate() error {
	// Add validation logic here
	return nil
}

func DefaultGraphManagerConfig() *GraphManagerConfig {
	return &GraphManagerConfig{
		MaxNodes:            10000,
		MaxEdges:            50000,
		MaxDepth:            100,
		EnableCaching:       true,
		EnableVisualization: true,
		EnableOptimization:  true,
		ConcurrencyLevel:    4,
		AnalysisTimeout:     300,
	}
}

// Additional types that might be needed
type GraphManagerMetrics struct{}
type GraphStorage interface{}
type GraphAnalyzer struct{}
type GraphVisualizer struct{}
type GraphOptimizer struct{}
type GraphAlgorithm interface{}
type GraphWorkerPool struct{}

type GraphFilter struct {
	Names       []string `json:"names,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CreatedFrom *string  `json:"createdFrom,omitempty"`
	CreatedTo   *string  `json:"createdTo,omitempty"`
	NodeCount   *int     `json:"nodeCount,omitempty"`
	HasCycles   *bool    `json:"hasCycles,omitempty"`
}

type GraphMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	NodeCount   int    `json:"nodeCount"`
	EdgeCount   int    `json:"edgeCount"`
	HasCycles   bool   `json:"hasCycles"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type GraphValidation struct {
	Valid       bool     `json:"valid"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	ValidatedAt string   `json:"validatedAt"`
}

type GraphManagerHealth struct {
	Status    string            `json:"status"`
	Uptime    string            `json:"uptime"`
	Metrics   *GraphManagerMetrics `json:"metrics"`
	Issues    []string          `json:"issues,omitempty"`
	CheckedAt string            `json:"checkedAt"`
}

type OptimizationOptions struct {
	Strategy     OptimizationStrategy `json:"strategy"`
	MaxIterations int                 `json:"maxIterations,omitempty"`
	Tolerance     float64             `json:"tolerance,omitempty"`
	PreservePaths []string            `json:"preservePaths,omitempty"`
	Constraints   map[string]interface{} `json:"constraints,omitempty"`
}

type GraphTransformer interface {
	Transform(ctx context.Context, graph *DependencyGraph) (*DependencyGraph, error)
	Name() string
	Description() string
}

// Simple graph visitor implementation
type simpleGraphVisitor struct {
	onNode func(ctx context.Context, node *GraphNode) error
	onEdge func(ctx context.Context, edge *GraphEdge) error
	onPre  func(ctx context.Context) error
	onPost func(ctx context.Context) error
}

func NewSimpleGraphVisitor() *simpleGraphVisitor {
	return &simpleGraphVisitor{
		onNode: func(ctx context.Context, node *GraphNode) error { return nil },
		onEdge: func(ctx context.Context, edge *GraphEdge) error { return nil },
		onPre:  func(ctx context.Context) error { return nil },
		onPost: func(ctx context.Context) error { return nil },
	}
}

func (v *simpleGraphVisitor) VisitNode(ctx context.Context, node *GraphNode) error {
	return v.onNode(ctx, node)
}

func (v *simpleGraphVisitor) VisitEdge(ctx context.Context, edge *GraphEdge) error {
	return v.onEdge(ctx, edge)
}

func (v *simpleGraphVisitor) PreVisit(ctx context.Context) error {
	return v.onPre(ctx)
}

func (v *simpleGraphVisitor) PostVisit(ctx context.Context) error {
	return v.onPost(ctx)
}

func (v *simpleGraphVisitor) OnNode(fn func(ctx context.Context, node *GraphNode) error) *simpleGraphVisitor {
	v.onNode = fn
	return v
}

func (v *simpleGraphVisitor) OnEdge(fn func(ctx context.Context, edge *GraphEdge) error) *simpleGraphVisitor {
	v.onEdge = fn
	return v
}

func (v *simpleGraphVisitor) OnPre(fn func(ctx context.Context) error) *simpleGraphVisitor {
	v.onPre = fn
	return v
}

func (v *simpleGraphVisitor) OnPost(fn func(ctx context.Context) error) *simpleGraphVisitor {
	v.onPost = fn
	return v
}

// Remove the duplicate Vulnerability definition if it exists in this file
// (it should be defined in validator.go or missing_types.go)