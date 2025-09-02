package multicluster

// Stub types for missing external dependencies
// These are temporary stubs to allow compilation while the actual dependencies are not available

// WorkloadCluster stub for nephiov1alpha1.WorkloadCluster
type WorkloadCluster struct {
	// Stub implementation - in production this would be from nephio API
}

// Package stubs to replace missing imports
var porchv1alpha1 = struct {
	// Type placeholder - not used directly but allows dot notation in stubs
}{}

var nephiov1alpha1 = struct {
	DeploymentStatusSucceeded string
}{
	DeploymentStatusSucceeded: "Succeeded",
}
