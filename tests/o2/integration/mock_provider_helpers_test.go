package o2_integration_tests_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/thc1006/nephoran-intent-operator/tests/o2/mocks"
)

// inMemoryClusterStore provides an in-memory cluster store for mock provider state.
type inMemoryClusterStore struct {
	mu       sync.RWMutex
	clusters map[string]*mocks.ClusterInfo
}

func newInMemoryClusterStore() *inMemoryClusterStore {
	return &inMemoryClusterStore{
		clusters: make(map[string]*mocks.ClusterInfo),
	}
}

func (s *inMemoryClusterStore) store(c *mocks.ClusterInfo) {
	s.mu.Lock()
	s.clusters[c.ID] = c
	s.mu.Unlock()
}

func (s *inMemoryClusterStore) get(id string) (*mocks.ClusterInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[id]
	return c, ok
}

func (s *inMemoryClusterStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.clusters[id]
	if ok {
		delete(s.clusters, id)
	}
	return ok
}

func (s *inMemoryClusterStore) list() []*mocks.ClusterInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*mocks.ClusterInfo, 0, len(s.clusters))
	for _, c := range s.clusters {
		result = append(result, c)
	}
	return result
}

// configureMockAWSProvider sets up testify mock expectations for the AWS provider.
// This is required because MockAWSProvider.CreateCluster calls m.Called() unconditionally,
// which panics without expectations.
//
// The key pattern here: capture the *mock.Call pointer and update ReturnArguments on
// the expectation object directly inside Run. This is the only reliable way to implement
// dynamic return values in testify, because MethodCalled reads call.ReturnArguments from
// the expectation pointer AFTER RunFn executes.
func configureMockAWSProvider(p *mocks.MockAWSProvider) {
	store := newInMemoryClusterStore()
	counter := 0

	var createClusterCall *mock.Call
	createClusterCall = p.On("CreateCluster", mock.Anything, mock.AnythingOfType("mocks.ClusterConfig")).
		Maybe().
		Return((*mocks.ClusterInfo)(nil), nil).
		Run(func(args mock.Arguments) {
			config := args.Get(1).(mocks.ClusterConfig)
			counter++
			cluster := &mocks.ClusterInfo{
				ID:        fmt.Sprintf("aws-cluster-%d", counter),
				Name:      config.Name,
				Provider:  "aws",
				Region:    config.Region,
				Status:    mocks.ClusterStatusActive,
				NodeCount: config.NodeCount,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Endpoint:  fmt.Sprintf("https://aws-cluster-%d.eks.us-east-1.amazonaws.com", counter),
				Config:    map[string]interface{}{"version": config.K8sVersion},
				Nodes:     []mocks.NodeInfo{},
			}
			store.store(cluster)
			createClusterCall.ReturnArguments = mock.Arguments{cluster, nil}
		})

	var getClusterCall *mock.Call
	getClusterCall = p.On("GetCluster", mock.Anything, mock.AnythingOfType("string")).
		Maybe().
		Return((*mocks.ClusterInfo)(nil), nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			cluster, ok := store.get(id)
			if !ok {
				getClusterCall.ReturnArguments = mock.Arguments{
					(*mocks.ClusterInfo)(nil),
					fmt.Errorf("cluster %s not found", id),
				}
				return
			}
			getClusterCall.ReturnArguments = mock.Arguments{cluster, nil}
		})

	p.On("ScaleCluster", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int")).
		Maybe().
		Return(nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			nodeCount := args.Get(2).(int)
			if cluster, ok := store.get(id); ok {
				cluster.NodeCount = nodeCount
				cluster.UpdatedAt = time.Now()
				store.store(cluster)
			}
		})

	var listClustersCall *mock.Call
	listClustersCall = p.On("ListClusters", mock.Anything).
		Maybe().
		Return([]*mocks.ClusterInfo{}, nil).
		Run(func(args mock.Arguments) {
			clusters := store.list()
			if len(clusters) == 0 {
				clusters = []*mocks.ClusterInfo{}
			}
			listClustersCall.ReturnArguments = mock.Arguments{clusters, nil}
		})

	var deleteClusterCall *mock.Call
	deleteClusterCall = p.On("DeleteCluster", mock.Anything, mock.AnythingOfType("string")).
		Maybe().
		Return(nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			if !store.delete(id) {
				deleteClusterCall.ReturnArguments = mock.Arguments{
					fmt.Errorf("cluster %s not found", id),
				}
			} else {
				deleteClusterCall.ReturnArguments = mock.Arguments{nil}
			}
		})
}

// configureMockAzureProvider sets up testify mock expectations for the Azure provider.
func configureMockAzureProvider(p *mocks.MockAzureProvider) {
	store := newInMemoryClusterStore()
	counter := 0

	var createClusterCall *mock.Call
	createClusterCall = p.On("CreateCluster", mock.Anything, mock.AnythingOfType("mocks.ClusterConfig")).
		Maybe().
		Return((*mocks.ClusterInfo)(nil), nil).
		Run(func(args mock.Arguments) {
			config := args.Get(1).(mocks.ClusterConfig)
			counter++
			cluster := &mocks.ClusterInfo{
				ID:        fmt.Sprintf("azure-cluster-%d", counter),
				Name:      config.Name,
				Provider:  "azure",
				Region:    config.Region,
				Status:    mocks.ClusterStatusActive,
				NodeCount: config.NodeCount,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Endpoint:  fmt.Sprintf("https://azure-cluster-%d.aks.eastus.azmk8s.io", counter),
				Config:    map[string]interface{}{"version": config.K8sVersion},
				Nodes:     []mocks.NodeInfo{},
			}
			store.store(cluster)
			createClusterCall.ReturnArguments = mock.Arguments{cluster, nil}
		})

	var getClusterCall *mock.Call
	getClusterCall = p.On("GetCluster", mock.Anything, mock.AnythingOfType("string")).
		Maybe().
		Return((*mocks.ClusterInfo)(nil), nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			cluster, ok := store.get(id)
			if !ok {
				getClusterCall.ReturnArguments = mock.Arguments{
					(*mocks.ClusterInfo)(nil),
					fmt.Errorf("cluster %s not found", id),
				}
				return
			}
			getClusterCall.ReturnArguments = mock.Arguments{cluster, nil}
		})

	p.On("ScaleCluster", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int")).
		Maybe().
		Return(nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			nodeCount := args.Get(2).(int)
			if cluster, ok := store.get(id); ok {
				cluster.NodeCount = nodeCount
				cluster.UpdatedAt = time.Now()
				store.store(cluster)
			}
		})

	var listClustersCall *mock.Call
	listClustersCall = p.On("ListClusters", mock.Anything).
		Maybe().
		Return([]*mocks.ClusterInfo{}, nil).
		Run(func(args mock.Arguments) {
			clusters := store.list()
			if len(clusters) == 0 {
				clusters = []*mocks.ClusterInfo{}
			}
			listClustersCall.ReturnArguments = mock.Arguments{clusters, nil}
		})

	var deleteClusterCall *mock.Call
	deleteClusterCall = p.On("DeleteCluster", mock.Anything, mock.AnythingOfType("string")).
		Maybe().
		Return(nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			if !store.delete(id) {
				deleteClusterCall.ReturnArguments = mock.Arguments{
					fmt.Errorf("cluster %s not found", id),
				}
			} else {
				deleteClusterCall.ReturnArguments = mock.Arguments{nil}
			}
		})
}

// configureMockGCPProvider sets up testify mock expectations for the GCP provider.
func configureMockGCPProvider(p *mocks.MockGCPProvider) {
	store := newInMemoryClusterStore()
	counter := 0

	var createClusterCall *mock.Call
	createClusterCall = p.On("CreateCluster", mock.Anything, mock.AnythingOfType("mocks.ClusterConfig")).
		Maybe().
		Return((*mocks.ClusterInfo)(nil), nil).
		Run(func(args mock.Arguments) {
			config := args.Get(1).(mocks.ClusterConfig)
			counter++
			cluster := &mocks.ClusterInfo{
				ID:        fmt.Sprintf("gcp-cluster-%d", counter),
				Name:      config.Name,
				Provider:  "gcp",
				Region:    config.Region,
				Status:    mocks.ClusterStatusActive,
				NodeCount: config.NodeCount,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Endpoint:  fmt.Sprintf("https://gcp-cluster-%d.us-central1.gke.gke.io", counter),
				Config:    map[string]interface{}{"version": config.K8sVersion},
				Nodes:     []mocks.NodeInfo{},
			}
			store.store(cluster)
			createClusterCall.ReturnArguments = mock.Arguments{cluster, nil}
		})

	var getClusterCall *mock.Call
	getClusterCall = p.On("GetCluster", mock.Anything, mock.AnythingOfType("string")).
		Maybe().
		Return((*mocks.ClusterInfo)(nil), nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			cluster, ok := store.get(id)
			if !ok {
				getClusterCall.ReturnArguments = mock.Arguments{
					(*mocks.ClusterInfo)(nil),
					fmt.Errorf("cluster %s not found", id),
				}
				return
			}
			getClusterCall.ReturnArguments = mock.Arguments{cluster, nil}
		})

	p.On("ScaleCluster", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int")).
		Maybe().
		Return(nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			nodeCount := args.Get(2).(int)
			if cluster, ok := store.get(id); ok {
				cluster.NodeCount = nodeCount
				cluster.UpdatedAt = time.Now()
				store.store(cluster)
			}
		})

	var listClustersCall *mock.Call
	listClustersCall = p.On("ListClusters", mock.Anything).
		Maybe().
		Return([]*mocks.ClusterInfo{}, nil).
		Run(func(args mock.Arguments) {
			clusters := store.list()
			if len(clusters) == 0 {
				clusters = []*mocks.ClusterInfo{}
			}
			listClustersCall.ReturnArguments = mock.Arguments{clusters, nil}
		})

	var deleteClusterCall *mock.Call
	deleteClusterCall = p.On("DeleteCluster", mock.Anything, mock.AnythingOfType("string")).
		Maybe().
		Return(nil).
		Run(func(args mock.Arguments) {
			id := args.Get(1).(string)
			if !store.delete(id) {
				deleteClusterCall.ReturnArguments = mock.Arguments{
					fmt.Errorf("cluster %s not found", id),
				}
			} else {
				deleteClusterCall.ReturnArguments = mock.Arguments{nil}
			}
		})
}
