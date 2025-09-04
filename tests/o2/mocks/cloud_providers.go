// Package mocks provides mock implementations for O2 interface cloud provider testing.

// This module implements comprehensive mock cloud providers for testing O2 interface functionality.

package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

// CloudProvider represents a generic cloud provider interface for O2 testing.

type CloudProvider interface {
	CreateCluster(ctx context.Context, config ClusterConfig) (*ClusterInfo, error)
	DeleteCluster(ctx context.Context, clusterID string) error
	GetCluster(ctx context.Context, clusterID string) (*ClusterInfo, error)
	ListClusters(ctx context.Context) ([]*ClusterInfo, error)
	ScaleCluster(ctx context.Context, clusterID string, nodeCount int) error
}

// ClusterConfig represents the configuration for creating a cluster.

type ClusterConfig struct {
	Name          string            `json:"name"`
	Provider      string            `json:"provider"`
	Region        string            `json:"region"`
	NodeCount     int               `json:"node_count"`
	NodeType      string            `json:"node_type"`
	K8sVersion    string            `json:"k8s_version"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	NetworkConfig NetworkConfig     `json:"network_config"`
	SecurityConfig SecurityConfig   `json:"security_config"`
}

// ClusterInfo represents information about a cluster.

type ClusterInfo struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Region     string                 `json:"region"`
	Status     ClusterStatus          `json:"status"`
	NodeCount  int                    `json:"node_count"`
	Nodes      []NodeInfo             `json:"nodes"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Endpoint   string                 `json:"endpoint"`
	Config     map[string]interface{} `json:"config"`
}

// NetworkConfig represents network configuration for a cluster.

type NetworkConfig struct {
	VPCId           string   `json:"vpc_id"`
	SubnetIds       []string `json:"subnet_ids"`
	ServiceCIDR     string   `json:"service_cidr"`
	PodCIDR         string   `json:"pod_cidr"`
	LoadBalancerIPs []string `json:"load_balancer_ips"`
}

// SecurityConfig represents security configuration for a cluster.

type SecurityConfig struct {
	EnableRBAC          bool     `json:"enable_rbac"`
	EnablePodSecurity   bool     `json:"enable_pod_security"`
	AllowedCIDRs        []string `json:"allowed_cidrs"`
	EncryptionAtRest    bool     `json:"encryption_at_rest"`
	EncryptionInTransit bool     `json:"encryption_in_transit"`
}

// NodeInfo represents information about a cluster node.

type NodeInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Status       NodeStatus        `json:"status"`
	InstanceType string            `json:"instance_type"`
	Zone         string            `json:"zone"`
	PrivateIP    string            `json:"private_ip"`
	PublicIP     string            `json:"public_ip"`
	Labels       map[string]string `json:"labels"`
	Taints       []NodeTaint       `json:"taints"`
	CreatedAt    time.Time         `json:"created_at"`
}

// NodeTaint represents a Kubernetes node taint.

type NodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// ClusterStatus represents the status of a cluster.

type ClusterStatus string

const (
	ClusterStatusCreating ClusterStatus = "creating"
	ClusterStatusActive   ClusterStatus = "active"
	ClusterStatusUpdating ClusterStatus = "updating"
	ClusterStatusDeleting ClusterStatus = "deleting"
	ClusterStatusError    ClusterStatus = "error"
)

// NodeStatus represents the status of a node.

type NodeStatus string

const (
	NodeStatusPending      NodeStatus = "pending"
	NodeStatusRunning      NodeStatus = "running"
	NodeStatusTerminating  NodeStatus = "terminating"
	NodeStatusTerminated   NodeStatus = "terminated"
	NodeStatusError        NodeStatus = "error"
)

// MockAWSProvider implements a mock AWS cloud provider.

type MockAWSProvider struct {
	mock.Mock
	clusters map[string]*ClusterInfo
	mu       sync.RWMutex
}

// NewMockAWSProvider creates a new mock AWS provider.

func NewMockAWSProvider() *MockAWSProvider {
	return &MockAWSProvider{
		clusters: make(map[string]*ClusterInfo),
	}
}

// CreateCluster creates a new cluster in AWS.

func (m *MockAWSProvider) CreateCluster(ctx context.Context, config ClusterConfig) (*ClusterInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, config)

	// If mock expectations are set, return mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*ClusterInfo), args.Error(1)
	}

	// Default behavior
	clusterID := fmt.Sprintf("aws-cluster-%d", time.Now().Unix())
	
	cluster := &ClusterInfo{
		ID:        clusterID,
		Name:      config.Name,
		Provider:  "aws",
		Region:    config.Region,
		Status:    ClusterStatusCreating,
		NodeCount: config.NodeCount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Endpoint:  fmt.Sprintf("https://%s.eks.%s.amazonaws.com", clusterID, config.Region),
		Config: map[string]interface{}{
			"eksVersion":     config.K8sVersion,
			"nodeGroupName":  fmt.Sprintf("%s-nodegroup", config.Name),
			"instanceType":   config.NodeType,
			"amiType":        "AL2_x86_64",
			"diskSize":       20,
			"scalingConfig": map[string]int{
				"minSize":     1,
				"maxSize":     config.NodeCount * 2,
				"desiredSize": config.NodeCount,
			},
		},
		Nodes: make([]NodeInfo, 0),
	}

	// Simulate nodes
	for i := 0; i < config.NodeCount; i++ {
		node := NodeInfo{
			ID:           fmt.Sprintf("aws-node-%s-%d", clusterID, i),
			Name:         fmt.Sprintf("%s-node-%d", config.Name, i),
			Status:       NodeStatusPending,
			InstanceType: config.NodeType,
			Zone:         fmt.Sprintf("%s%c", config.Region, 'a'+i%3),
			PrivateIP:    fmt.Sprintf("10.0.%d.%d", i/254, i%254),
			Labels: map[string]string{
				"kubernetes.io/os":             "linux",
				"kubernetes.io/arch":           "amd64",
				"node.kubernetes.io/instance-type": config.NodeType,
				"topology.kubernetes.io/zone":  fmt.Sprintf("%s%c", config.Region, 'a'+i%3),
			},
			CreatedAt: time.Now(),
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}

	m.clusters[clusterID] = cluster

	// Simulate async cluster creation
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.mu.Lock()
		if cluster, exists := m.clusters[clusterID]; exists {
			cluster.Status = ClusterStatusActive
			cluster.UpdatedAt = time.Now()
			for i := range cluster.Nodes {
				cluster.Nodes[i].Status = NodeStatusRunning
				cluster.Nodes[i].PublicIP = fmt.Sprintf("54.%d.%d.%d", 
					i/65536, (i/256)%256, i%256)
			}
		}
		m.mu.Unlock()
	}()

	return cluster, nil
}

// DeleteCluster deletes a cluster from AWS.

func (m *MockAWSProvider) DeleteCluster(ctx context.Context, clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clusterID)

	// If mock expectations are set, return mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	// Default behavior
	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	cluster.Status = ClusterStatusDeleting
	cluster.UpdatedAt = time.Now()

	// Simulate async deletion
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		delete(m.clusters, clusterID)
		m.mu.Unlock()
	}()

	return nil
}

// GetCluster retrieves cluster information from AWS.

func (m *MockAWSProvider) GetCluster(ctx context.Context, clusterID string) (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, clusterID)

	// If mock expectations are set, return mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*ClusterInfo), args.Error(1)
	}

	// Default behavior
	cluster, exists := m.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	return cluster, nil
}

// ListClusters lists all clusters in AWS.

func (m *MockAWSProvider) ListClusters(ctx context.Context) ([]*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx)

	// If mock expectations are set, return mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).([]*ClusterInfo), args.Error(1)
	}

	// Default behavior
	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// ScaleCluster scales a cluster in AWS.

func (m *MockAWSProvider) ScaleCluster(ctx context.Context, clusterID string, nodeCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clusterID, nodeCount)

	// If mock expectations are set, return mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	// Default behavior
	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	if cluster.Status != ClusterStatusActive {
		return fmt.Errorf("cluster %s is not in active state", clusterID)
	}

	currentCount := len(cluster.Nodes)
	
	if nodeCount > currentCount {
		// Scale up - add nodes
		for i := currentCount; i < nodeCount; i++ {
			node := NodeInfo{
				ID:           fmt.Sprintf("aws-node-%s-%d", clusterID, i),
				Name:         fmt.Sprintf("%s-node-%d", cluster.Name, i),
				Status:       NodeStatusPending,
				InstanceType: cluster.Nodes[0].InstanceType,
				Zone:         fmt.Sprintf("%s%c", cluster.Region, 'a'+i%3),
				PrivateIP:    fmt.Sprintf("10.0.%d.%d", i/254, i%254),
				Labels: map[string]string{
					"kubernetes.io/os":             "linux",
					"kubernetes.io/arch":           "amd64",
					"node.kubernetes.io/instance-type": cluster.Nodes[0].InstanceType,
					"topology.kubernetes.io/zone":  fmt.Sprintf("%s%c", cluster.Region, 'a'+i%3),
				},
				CreatedAt: time.Now(),
			}
			cluster.Nodes = append(cluster.Nodes, node)
		}
	} else if nodeCount < currentCount {
		// Scale down - remove nodes
		cluster.Nodes = cluster.Nodes[:nodeCount]
	}

	cluster.NodeCount = nodeCount
	cluster.Status = ClusterStatusUpdating
	cluster.UpdatedAt = time.Now()

	// Simulate async scaling
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		if cluster, exists := m.clusters[clusterID]; exists {
			cluster.Status = ClusterStatusActive
			cluster.UpdatedAt = time.Now()
			// Update node statuses
			for i := range cluster.Nodes {
				cluster.Nodes[i].Status = NodeStatusRunning
				if cluster.Nodes[i].PublicIP == "" {
					cluster.Nodes[i].PublicIP = fmt.Sprintf("54.%d.%d.%d", 
						i/65536, (i/256)%256, i%256)
				}
			}
		}
		m.mu.Unlock()
	}()

	return nil
}

// MockAzureProvider implements a mock Azure cloud provider.

type MockAzureProvider struct {
	mock.Mock
	clusters map[string]*ClusterInfo
	mu       sync.RWMutex
}

// NewMockAzureProvider creates a new mock Azure provider.

func NewMockAzureProvider() *MockAzureProvider {
	return &MockAzureProvider{
		clusters: make(map[string]*ClusterInfo),
	}
}

// CreateCluster creates a new cluster in Azure.

func (m *MockAzureProvider) CreateCluster(ctx context.Context, config ClusterConfig) (*ClusterInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, config)

	// If mock expectations are set, return mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*ClusterInfo), args.Error(1)
	}

	// Default behavior - similar to AWS but with Azure-specific details
	clusterID := fmt.Sprintf("azure-cluster-%d", time.Now().Unix())
	
	cluster := &ClusterInfo{
		ID:        clusterID,
		Name:      config.Name,
		Provider:  "azure",
		Region:    config.Region,
		Status:    ClusterStatusCreating,
		NodeCount: config.NodeCount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Endpoint:  fmt.Sprintf("https://%s.%s.azmk8s.io:443", config.Name, config.Region),
		Config: map[string]interface{}{
			"kubernetesVersion": config.K8sVersion,
			"agentPoolName":     fmt.Sprintf("%snodes", config.Name[:3]),
			"vmSize":            config.NodeType,
			"osDiskSizeGB":      30,
			"count":             config.NodeCount,
			"maxPods":           110,
			"osType":            "Linux",
		},
		Nodes: make([]NodeInfo, 0),
	}

	// Create nodes with Azure-specific naming
	for i := 0; i < config.NodeCount; i++ {
		node := NodeInfo{
			ID:           fmt.Sprintf("aks-%s-%d", clusterID, i),
			Name:         fmt.Sprintf("aks-%s-vmss_%d", config.Name, i),
			Status:       NodeStatusPending,
			InstanceType: config.NodeType,
			Zone:         fmt.Sprintf("%s-%d", config.Region, (i%3)+1),
			PrivateIP:    fmt.Sprintf("10.240.%d.%d", i/254, i%254+4),
			Labels: map[string]string{
				"kubernetes.io/os":              "linux",
				"kubernetes.io/arch":            "amd64",
				"node.kubernetes.io/instance-type": config.NodeType,
				"topology.kubernetes.io/zone":   fmt.Sprintf("%s-%d", config.Region, (i%3)+1),
				"agentpool":                     fmt.Sprintf("%snodes", config.Name[:3]),
			},
			CreatedAt: time.Now(),
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}

	m.clusters[clusterID] = cluster

	// Simulate async cluster creation
	go func() {
		time.Sleep(120 * time.Millisecond)
		m.mu.Lock()
		if cluster, exists := m.clusters[clusterID]; exists {
			cluster.Status = ClusterStatusActive
			cluster.UpdatedAt = time.Now()
			for i := range cluster.Nodes {
				cluster.Nodes[i].Status = NodeStatusRunning
				cluster.Nodes[i].PublicIP = fmt.Sprintf("20.%d.%d.%d", 
					i/65536, (i/256)%256, i%256)
			}
		}
		m.mu.Unlock()
	}()

	return cluster, nil
}

// DeleteCluster deletes a cluster from Azure.

func (m *MockAzureProvider) DeleteCluster(ctx context.Context, clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clusterID)

	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	cluster.Status = ClusterStatusDeleting
	cluster.UpdatedAt = time.Now()

	go func() {
		time.Sleep(60 * time.Millisecond)
		m.mu.Lock()
		delete(m.clusters, clusterID)
		m.mu.Unlock()
	}()

	return nil
}

// GetCluster retrieves cluster information from Azure.

func (m *MockAzureProvider) GetCluster(ctx context.Context, clusterID string) (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, clusterID)

	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*ClusterInfo), args.Error(1)
	}

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	return cluster, nil
}

// ListClusters lists all clusters in Azure.

func (m *MockAzureProvider) ListClusters(ctx context.Context) ([]*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx)

	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).([]*ClusterInfo), args.Error(1)
	}

	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// ScaleCluster scales a cluster in Azure.

func (m *MockAzureProvider) ScaleCluster(ctx context.Context, clusterID string, nodeCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clusterID, nodeCount)

	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	if cluster.Status != ClusterStatusActive {
		return fmt.Errorf("cluster %s is not in active state", clusterID)
	}

	// Similar scaling logic as AWS
	currentCount := len(cluster.Nodes)
	
	if nodeCount > currentCount {
		for i := currentCount; i < nodeCount; i++ {
			node := NodeInfo{
				ID:           fmt.Sprintf("aks-%s-%d", clusterID, i),
				Name:         fmt.Sprintf("aks-%s-vmss_%d", cluster.Name, i),
				Status:       NodeStatusPending,
				InstanceType: cluster.Nodes[0].InstanceType,
				Zone:         fmt.Sprintf("%s-%d", cluster.Region, (i%3)+1),
				PrivateIP:    fmt.Sprintf("10.240.%d.%d", i/254, i%254+4),
				Labels: cluster.Nodes[0].Labels, // Copy labels from first node
				CreatedAt: time.Now(),
			}
			cluster.Nodes = append(cluster.Nodes, node)
		}
	} else if nodeCount < currentCount {
		cluster.Nodes = cluster.Nodes[:nodeCount]
	}

	cluster.NodeCount = nodeCount
	cluster.Status = ClusterStatusUpdating
	cluster.UpdatedAt = time.Now()

	go func() {
		time.Sleep(70 * time.Millisecond)
		m.mu.Lock()
		if cluster, exists := m.clusters[clusterID]; exists {
			cluster.Status = ClusterStatusActive
			cluster.UpdatedAt = time.Now()
			for i := range cluster.Nodes {
				cluster.Nodes[i].Status = NodeStatusRunning
				if cluster.Nodes[i].PublicIP == "" {
					cluster.Nodes[i].PublicIP = fmt.Sprintf("20.%d.%d.%d", 
						i/65536, (i/256)%256, i%256)
				}
			}
		}
		m.mu.Unlock()
	}()

	return nil
}

// MockGCPProvider implements a mock Google Cloud Platform provider.

type MockGCPProvider struct {
	mock.Mock
	clusters map[string]*ClusterInfo
	mu       sync.RWMutex
}

// NewMockGCPProvider creates a new mock GCP provider.

func NewMockGCPProvider() *MockGCPProvider {
	return &MockGCPProvider{
		clusters: make(map[string]*ClusterInfo),
	}
}

// CreateCluster creates a new cluster in GCP.

func (m *MockGCPProvider) CreateCluster(ctx context.Context, config ClusterConfig) (*ClusterInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, config)

	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*ClusterInfo), args.Error(1)
	}

	clusterID := fmt.Sprintf("gcp-cluster-%d", time.Now().Unix())
	
	cluster := &ClusterInfo{
		ID:        clusterID,
		Name:      config.Name,
		Provider:  "gcp",
		Region:    config.Region,
		Status:    ClusterStatusCreating,
		NodeCount: config.NodeCount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Endpoint:  fmt.Sprintf("https://container.googleapis.com/v1/projects/project/zones/%s/clusters/%s", config.Region, config.Name),
		Config: map[string]interface{}{
			"masterVersion":       config.K8sVersion,
			"nodePoolName":        fmt.Sprintf("%s-default-pool", config.Name),
			"machineType":         config.NodeType,
			"diskSizeGb":          100,
			"initialNodeCount":    config.NodeCount,
			"imageType":           "COS_CONTAINERD",
			"diskType":            "pd-standard",
			"preemptible":         false,
		},
		Nodes: make([]NodeInfo, 0),
	}

	// Create nodes with GCP-specific naming
	for i := 0; i < config.NodeCount; i++ {
		node := NodeInfo{
			ID:           fmt.Sprintf("gke-%s-default-pool-%x", config.Name, time.Now().UnixNano()+int64(i)),
			Name:         fmt.Sprintf("gke-%s-default-pool-%d", config.Name, i),
			Status:       NodeStatusPending,
			InstanceType: config.NodeType,
			Zone:         fmt.Sprintf("%s-%c", config.Region, 'a'+i%3),
			PrivateIP:    fmt.Sprintf("10.128.%d.%d", i/254, i%254+2),
			Labels: map[string]string{
				"kubernetes.io/os":               "linux",
				"kubernetes.io/arch":             "amd64",
				"node.kubernetes.io/instance-type": config.NodeType,
				"topology.kubernetes.io/zone":    fmt.Sprintf("%s-%c", config.Region, 'a'+i%3),
				"cloud.google.com/gke-nodepool":  fmt.Sprintf("%s-default-pool", config.Name),
			},
			CreatedAt: time.Now(),
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}

	m.clusters[clusterID] = cluster

	// Simulate async cluster creation
	go func() {
		time.Sleep(150 * time.Millisecond)
		m.mu.Lock()
		if cluster, exists := m.clusters[clusterID]; exists {
			cluster.Status = ClusterStatusActive
			cluster.UpdatedAt = time.Now()
			for i := range cluster.Nodes {
				cluster.Nodes[i].Status = NodeStatusRunning
				cluster.Nodes[i].PublicIP = fmt.Sprintf("35.%d.%d.%d", 
					i/65536, (i/256)%256, i%256)
			}
		}
		m.mu.Unlock()
	}()

	return cluster, nil
}

// DeleteCluster deletes a cluster from GCP.

func (m *MockGCPProvider) DeleteCluster(ctx context.Context, clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clusterID)

	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	cluster.Status = ClusterStatusDeleting
	cluster.UpdatedAt = time.Now()

	go func() {
		time.Sleep(80 * time.Millisecond)
		m.mu.Lock()
		delete(m.clusters, clusterID)
		m.mu.Unlock()
	}()

	return nil
}

// GetCluster retrieves cluster information from GCP.

func (m *MockGCPProvider) GetCluster(ctx context.Context, clusterID string) (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, clusterID)

	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*ClusterInfo), args.Error(1)
	}

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	return cluster, nil
}

// ListClusters lists all clusters in GCP.

func (m *MockGCPProvider) ListClusters(ctx context.Context) ([]*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx)

	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).([]*ClusterInfo), args.Error(1)
	}

	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// ScaleCluster scales a cluster in GCP.

func (m *MockGCPProvider) ScaleCluster(ctx context.Context, clusterID string, nodeCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clusterID, nodeCount)

	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	if cluster.Status != ClusterStatusActive {
		return fmt.Errorf("cluster %s is not in active state", clusterID)
	}

	currentCount := len(cluster.Nodes)
	
	if nodeCount > currentCount {
		for i := currentCount; i < nodeCount; i++ {
			node := NodeInfo{
				ID:           fmt.Sprintf("gke-%s-default-pool-%x", cluster.Name, time.Now().UnixNano()+int64(i)),
				Name:         fmt.Sprintf("gke-%s-default-pool-%d", cluster.Name, i),
				Status:       NodeStatusPending,
				InstanceType: cluster.Nodes[0].InstanceType,
				Zone:         fmt.Sprintf("%s-%c", cluster.Region, 'a'+i%3),
				PrivateIP:    fmt.Sprintf("10.128.%d.%d", i/254, i%254+2),
				Labels:       cluster.Nodes[0].Labels, // Copy labels from first node
				CreatedAt:    time.Now(),
			}
			cluster.Nodes = append(cluster.Nodes, node)
		}
	} else if nodeCount < currentCount {
		cluster.Nodes = cluster.Nodes[:nodeCount]
	}

	cluster.NodeCount = nodeCount
	cluster.Status = ClusterStatusUpdating
	cluster.UpdatedAt = time.Now()

	go func() {
		time.Sleep(90 * time.Millisecond)
		m.mu.Lock()
		if cluster, exists := m.clusters[clusterID]; exists {
			cluster.Status = ClusterStatusActive
			cluster.UpdatedAt = time.Now()
			for i := range cluster.Nodes {
				cluster.Nodes[i].Status = NodeStatusRunning
				if cluster.Nodes[i].PublicIP == "" {
					cluster.Nodes[i].PublicIP = fmt.Sprintf("35.%d.%d.%d", 
						i/65536, (i/256)%256, i%256)
				}
			}
		}
		m.mu.Unlock()
	}()

	return nil
}

// CloudProviderFactory creates cloud provider instances for testing.

type CloudProviderFactory struct{}

// NewCloudProviderFactory creates a new cloud provider factory.

func NewCloudProviderFactory() *CloudProviderFactory {
	return &CloudProviderFactory{}
}

// CreateProvider creates a cloud provider based on the provider type.

func (f *CloudProviderFactory) CreateProvider(providerType string) (CloudProvider, error) {
	switch providerType {
	case "aws":
		return NewMockAWSProvider(), nil
	case "azure":
		return NewMockAzureProvider(), nil
	case "gcp":
		return NewMockGCPProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", providerType)
	}
}

// MultiCloudManager manages multiple cloud providers for testing.

type MultiCloudManager struct {
	providers map[string]CloudProvider
	mu        sync.RWMutex
}

// NewMultiCloudManager creates a new multi-cloud manager.

func NewMultiCloudManager() *MultiCloudManager {
	return &MultiCloudManager{
		providers: make(map[string]CloudProvider),
	}
}

// AddProvider adds a cloud provider to the manager.

func (m *MultiCloudManager) AddProvider(name string, provider CloudProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = provider
}

// GetProvider gets a cloud provider by name.

func (m *MultiCloudManager) GetProvider(name string) (CloudProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	
	return provider, nil
}

// ListProviders lists all available providers.

func (m *MultiCloudManager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	
	return names
}

// CreateClusterMultiRegion creates clusters across multiple regions.

func (m *MultiCloudManager) CreateClusterMultiRegion(ctx context.Context, configs map[string]ClusterConfig) (map[string]*ClusterInfo, error) {
	results := make(map[string]*ClusterInfo)
	errors := make(map[string]error)
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for providerName, config := range configs {
		wg.Add(1)
		go func(name string, cfg ClusterConfig) {
			defer wg.Done()
			
			provider, err := m.GetProvider(name)
			if err != nil {
				mu.Lock()
				errors[name] = err
				mu.Unlock()
				return
			}
			
			cluster, err := provider.CreateCluster(ctx, cfg)
			mu.Lock()
			if err != nil {
				errors[name] = err
			} else {
				results[name] = cluster
			}
			mu.Unlock()
		}(providerName, config)
	}
	
	wg.Wait()
	
	if len(errors) > 0 {
		// Return partial results and first error
		for _, err := range errors {
			return results, err
		}
	}
	
	return results, nil
}

// Helper function to convert cluster info to JSON for debugging.

func (c *ClusterInfo) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// Helper function to validate cluster configuration.

func (c *ClusterConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}
	
	if c.Provider == "" {
		return fmt.Errorf("cluster provider cannot be empty")
	}
	
	if c.NodeCount <= 0 {
		return fmt.Errorf("node count must be greater than 0")
	}
	
	if c.NodeType == "" {
		return fmt.Errorf("node type cannot be empty")
	}
	
	return nil
}