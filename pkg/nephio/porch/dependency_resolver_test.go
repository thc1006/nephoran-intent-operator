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

package porch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyResolver_5GCoreScenario(t *testing.T) {
	// Test 5G Core function dependencies
	tests := []struct {
		name              string
		rootPackage       *PackageReference
		expectedDeps      []string
		expectedOrder     []string
		expectedConflicts int
	}{
		{
			name: "AMF with UDM dependency",
			rootPackage: &PackageReference{
				Name:      "amf",
				Namespace: "5g-core",
			},
			expectedDeps:      []string{"udm", "ausf", "nrf"},
			expectedOrder:     []string{"nrf", "udm", "ausf", "amf"},
			expectedConflicts: 0,
		},
		{
			name: "SMF with UPF dependency",
			rootPackage: &PackageReference{
				Name:      "smf",
				Namespace: "5g-core",
			},
			expectedDeps:      []string{"upf", "pcf", "nrf"},
			expectedOrder:     []string{"nrf", "pcf", "upf", "smf"},
			expectedConflicts: 0,
		},
		{
			name: "NSSF with network slicing",
			rootPackage: &PackageReference{
				Name:      "nssf",
				Namespace: "5g-core",
			},
			expectedDeps:      []string{"nrf", "slice-manager"},
			expectedOrder:     []string{"nrf", "slice-manager", "nssf"},
			expectedConflicts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and resolver
			client := createMockClient()
			config := &DependencyResolverConfig{
				EnableCaching:     true,
				MaxGraphDepth:     10,
				MaxResolutionTime: 5 * time.Minute,
			}
			resolver, err := NewDependencyResolver(client, config)
			require.NoError(t, err)
			defer resolver.Close()

			ctx := context.Background()

			// Build dependency graph
			graph, err := resolver.BuildDependencyGraph(ctx,
				[]*PackageReference{tt.rootPackage},
				&GraphBuildOptions{
					MaxDepth:          5,
					ResolveTransitive: true,
					UseCache:          false,
				})
			require.NoError(t, err)
			assert.NotNil(t, graph)

			// Verify dependencies are discovered
			for _, expectedDep := range tt.expectedDeps {
				found := false
				for nodeID := range graph.Nodes {
					if contains(nodeID, expectedDep) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected dependency %s not found", expectedDep)
			}

			// Check for circular dependencies
			circularResult, err := resolver.DetectCircularDependencies(ctx, graph)
			require.NoError(t, err)
			assert.False(t, circularResult.HasCircularDependencies,
				"Unexpected circular dependencies found")

			// Resolve dependencies
			result, err := resolver.ResolveDependencies(ctx, tt.rootPackage, nil)
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.Equal(t, tt.expectedConflicts, len(result.Conflicts))
		})
	}
}

func TestDependencyResolver_ORANScenario(t *testing.T) {
	// Test O-RAN component dependencies
	tests := []struct {
		name         string
		rootPackage  *PackageReference
		expectedDeps []string
	}{
		{
			name: "Near-RT RIC with E2 nodes",
			rootPackage: &PackageReference{
				Name:      "near-rt-ric",
				Namespace: "oran",
			},
			expectedDeps: []string{"e2-termination", "a1-mediator", "subscription-manager"},
		},
		{
			name: "xApp with Near-RT RIC dependency",
			rootPackage: &PackageReference{
				Name:      "xapp-traffic-steering",
				Namespace: "oran",
			},
			expectedDeps: []string{"near-rt-ric-platform", "ric-sdk"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createMockClient()
			resolver, err := NewDependencyResolver(client, nil)
			require.NoError(t, err)
			defer resolver.Close()

			ctx := context.Background()

			// Build dependency graph
			graph, err := resolver.BuildDependencyGraph(ctx,
				[]*PackageReference{tt.rootPackage},
				&GraphBuildOptions{
					MaxDepth:          5,
					ResolveTransitive: true,
				})
			require.NoError(t, err)

			// Verify O-RAN specific dependencies
			for _, expectedDep := range tt.expectedDeps {
				found := false
				for nodeID := range graph.Nodes {
					if contains(nodeID, expectedDep) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected O-RAN dependency %s not found", expectedDep)
			}
		})
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	// Test circular dependency detection and resolution
	client := createMockClient()
	resolver, err := NewDependencyResolver(client, nil)
	require.NoError(t, err)
	defer resolver.Close()

	ctx := context.Background()

	// Create a graph with circular dependencies
	graph := &DependencyGraph{
		ID:    "test-circular",
		Nodes: make(map[string]*DependencyNode),
		Edges: []*DependencyEdge{},
	}

	// Create circular dependency: A -> B -> C -> A
	nodeA := &DependencyNode{
		ID:         "package-a",
		PackageRef: &PackageReference{Name: "a"},
	}
	nodeB := &DependencyNode{
		ID:         "package-b",
		PackageRef: &PackageReference{Name: "b"},
	}
	nodeC := &DependencyNode{
		ID:         "package-c",
		PackageRef: &PackageReference{Name: "c"},
	}

	graph.Nodes["package-a"] = nodeA
	graph.Nodes["package-b"] = nodeB
	graph.Nodes["package-c"] = nodeC

	// Create edges forming a cycle
	edgeAB := &DependencyEdge{From: nodeA, To: nodeB}
	edgeBC := &DependencyEdge{From: nodeB, To: nodeC}
	edgeCA := &DependencyEdge{From: nodeC, To: nodeA}

	nodeA.Dependencies = []*DependencyEdge{edgeAB}
	nodeB.Dependencies = []*DependencyEdge{edgeBC}
	nodeC.Dependencies = []*DependencyEdge{edgeCA}

	graph.Edges = append(graph.Edges, edgeAB, edgeBC, edgeCA)

	// Detect circular dependencies
	circularResult, err := resolver.DetectCircularDependencies(ctx, graph)
	require.NoError(t, err)
	assert.True(t, circularResult.HasCircularDependencies)
	assert.GreaterOrEqual(t, len(circularResult.Cycles), 1)
	assert.Equal(t, 3, len(circularResult.AffectedPackages))

	// Verify breaking suggestions are provided
	assert.NotEmpty(t, circularResult.BreakingSuggestions)
}

func TestDependencyResolver_VersionConstraints(t *testing.T) {
	// Test version constraint solving
	client := createMockClient()
	resolver, err := NewDependencyResolver(client, nil)
	require.NoError(t, err)
	defer resolver.Close()

	ctx := context.Background()

	// Create version requirements
	requirements := []*VersionRequirement{
		{
			PackageRef: &PackageReference{Name: "amf"},
			Constraints: []*VersionConstraint{
				{Operator: ConstraintOperatorGreaterEquals, Version: "1.0.0"},
				{Operator: ConstraintOperatorLessThan, Version: "2.0.0"},
			},
		},
		{
			PackageRef: &PackageReference{Name: "smf"},
			Constraints: []*VersionConstraint{
				{Operator: ConstraintOperatorEquals, Version: "1.5.0"},
			},
		},
	}

	// Solve version constraints
	solution, err := resolver.SolveVersionConstraints(ctx, requirements)
	require.NoError(t, err)
	assert.True(t, solution.Success)
	assert.NotNil(t, solution.Solutions)
	assert.Equal(t, 0, len(solution.Conflicts))
}

func TestDependencyResolver_TransitiveDependencies(t *testing.T) {
	// Test transitive dependency resolution
	client := createMockClient()
	resolver, err := NewDependencyResolver(client, nil)
	require.NoError(t, err)
	defer resolver.Close()

	ctx := context.Background()

	pkgRef := &PackageReference{
		Name:      "amf",
		Namespace: "5g-core",
	}

	// Resolve transitive dependencies
	transitive, err := resolver.ResolveTransitiveDependencies(ctx, pkgRef, 3)
	require.NoError(t, err)
	assert.NotNil(t, transitive)
	assert.GreaterOrEqual(t, transitive.TotalDependencies, 3)
	assert.LessOrEqual(t, transitive.MaxDepth, 3)
}

func TestDependencyResolver_UpdatePropagation(t *testing.T) {
	// Test update propagation through dependency graph
	client := createMockClient()
	resolver, err := NewDependencyResolver(client, nil)
	require.NoError(t, err)
	defer resolver.Close()

	ctx := context.Background()

	updatedPackage := &PackageReference{
		Name:      "nrf",
		Namespace: "5g-core",
	}

	// Propagate updates
	propagationResult, err := resolver.PropagateUpdates(ctx,
		updatedPackage,
		PropagationStrategyEager)
	require.NoError(t, err)
	assert.True(t, propagationResult.Success)
	assert.NotEmpty(t, propagationResult.UpdatedPackages)

	// Verify propagation plan
	assert.NotNil(t, propagationResult.PropagationPlan)
	assert.GreaterOrEqual(t, len(propagationResult.PropagationPlan.Steps), 1)
}

func TestContextAwareSelector_5GCoreContext(t *testing.T) {
	// Test context-aware dependency selection for 5G Core
	selector := NewContextAwareDependencySelector()
	ctx := context.Background()

	pkg := &PackageReference{
		Name:      "amf",
		Namespace: "5g-core",
	}

	targetClusters := []*WorkloadCluster{
		{
			Name:   "core-cluster",
			Region: "us-west",
			Type:   ClusterTypeCore,
		},
		{
			Name:   "edge-cluster",
			Region: "us-west",
			Type:   ClusterTypeEdge,
		},
	}

	opts := &ContextSelectionOptions{
		OptimizeForLatency: true,
		RequireHA:          true,
	}

	// Select dependencies based on context
	contextualDeps, err := selector.SelectDependencies(ctx, pkg, targetClusters, opts)
	require.NoError(t, err)
	assert.NotNil(t, contextualDeps)
	assert.NotEmpty(t, contextualDeps.PrimaryDependencies)
	assert.NotEmpty(t, contextualDeps.DeploymentOrder)
	assert.Greater(t, contextualDeps.ContextScore, 0.0)
}

func TestSATSolver_ComplexConstraints(t *testing.T) {
	// Test SAT solver with complex version constraints
	config := &SATSolverConfig{
		MaxDecisions:   1000,
		MaxConflicts:   100,
		Timeout:        1 * time.Minute,
		EnableLearning: true,
		EnableVSIDS:    true,
	}

	solver := NewSATSolver(config)
	defer solver.Close()

	ctx := context.Background()

	// Create complex requirements
	requirements := []*VersionRequirement{
		{
			PackageRef: &PackageReference{Name: "package-a"},
			Constraints: []*VersionConstraint{
				{Operator: ConstraintOperatorGreaterEquals, Version: "1.0.0"},
				{Operator: ConstraintOperatorLessThan, Version: "2.0.0"},
			},
		},
		{
			PackageRef: &PackageReference{Name: "package-b"},
			Constraints: []*VersionConstraint{
				{Operator: ConstraintOperatorGreaterEquals, Version: "2.0.0"},
			},
		},
		{
			PackageRef: &PackageReference{Name: "package-c"},
			Constraints: []*VersionConstraint{
				{Operator: ConstraintOperatorTilde, Version: "1.5.0"},
			},
		},
	}

	// Solve constraints
	solution, err := solver.Solve(ctx, requirements)
	require.NoError(t, err)
	assert.True(t, solution.Success)
	assert.NotNil(t, solution.Solutions)
	assert.GreaterOrEqual(t, len(solution.Solutions), 3)
}

func TestGraphAnalyzer_ComplexGraph(t *testing.T) {
	// Test graph analysis on complex dependency graph
	analyzer := NewDependencyGraphAnalyzer()
	ctx := context.Background()

	// Create a complex graph
	graph := createComplexTestGraph()

	opts := &AnalysisOptions{
		UseCache:         false,
		ParallelAnalysis: true,
		MaxDepth:         10,
	}

	// Analyze the graph
	analysis, err := analyzer.AnalyzeGraph(ctx, graph, opts)
	require.NoError(t, err)
	assert.NotNil(t, analysis)

	// Verify analysis results
	assert.NotEmpty(t, analysis.TopologicalOrder)
	assert.NotEmpty(t, analysis.CentralityScores)
	assert.NotNil(t, analysis.GraphMetrics)
	assert.GreaterOrEqual(t, analysis.GraphMetrics.NodeCount, 5)

	// Check for anomalies
	if len(analysis.Anomalies) > 0 {
		for _, anomaly := range analysis.Anomalies {
			t.Logf("Anomaly detected: %s - %s", anomaly.Type, anomaly.Description)
		}
	}

	// Check optimization hints
	if len(analysis.OptimizationHints) > 0 {
		for _, hint := range analysis.OptimizationHints {
			t.Logf("Optimization hint: %s - %s", hint.Type, hint.Description)
		}
	}
}

// Helper functions

func createMockClient() *Client {
	// Create a mock client for testing
	// In production, this would be a real Porch client
	return &Client{}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

func createComplexTestGraph() *DependencyGraph {
	// Create a complex test graph with multiple nodes and edges
	graph := &DependencyGraph{
		ID:    "test-complex",
		Nodes: make(map[string]*DependencyNode),
		Edges: []*DependencyEdge{},
	}

	// Create nodes
	nodes := []string{"amf", "smf", "upf", "nrf", "udm", "ausf", "pcf"}
	for _, name := range nodes {
		node := &DependencyNode{
			ID:         "5g-core/" + name,
			PackageRef: &PackageReference{Name: name, Namespace: "5g-core"},
		}
		graph.Nodes[node.ID] = node
	}

	// Create edges (simplified 5G Core dependencies)
	edges := []struct{ from, to string }{
		{"amf", "udm"},
		{"amf", "ausf"},
		{"amf", "nrf"},
		{"smf", "upf"},
		{"smf", "pcf"},
		{"smf", "nrf"},
		{"udm", "nrf"},
		{"ausf", "nrf"},
		{"pcf", "nrf"},
	}

	for _, e := range edges {
		fromNode := graph.Nodes["5g-core/"+e.from]
		toNode := graph.Nodes["5g-core/"+e.to]
		edge := &DependencyEdge{
			From: fromNode,
			To:   toNode,
			Type: DependencyTypeRuntime,
		}
		fromNode.Dependencies = append(fromNode.Dependencies, edge)
		toNode.RequiredBy = append(toNode.RequiredBy, edge)
		graph.Edges = append(graph.Edges, edge)
	}

	graph.NodeCount = len(graph.Nodes)
	graph.EdgeCount = len(graph.Edges)

	return graph
}
