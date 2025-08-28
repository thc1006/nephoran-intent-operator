package nephiomulticluster

import "time"

// Stub types for missing external dependencies
// These are temporary stubs to allow compilation while the actual dependencies are not available

// PackageRevision stub for porchv1alpha1.PackageRevision
type PackageRevision struct {
	// Stub implementation - in production this would be from porch API
}

// WorkloadCluster stub for nephiov1alpha1.WorkloadCluster  
type WorkloadCluster struct {
	// Stub implementation - in production this would be from nephio API
}

// MultiClusterDeploymentStatus stub for nephiov1alpha1.MultiClusterDeploymentStatus
type MultiClusterDeploymentStatus struct {
	Clusters map[string]ClusterDeploymentStatus
}

// ClusterDeploymentStatus stub for nephiov1alpha1.ClusterDeploymentStatus
type ClusterDeploymentStatus struct {
	ClusterName string
	Status      string 
	Timestamp   time.Time
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