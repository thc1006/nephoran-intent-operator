
package testutil



import (

	"context"

	"fmt"

	"sync"

	"time"



	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"

)



// FakeE2Manager implements E2ManagerInterface for testing.

type FakeE2Manager struct {

	mutex                    sync.RWMutex

	nodes                    map[string]*e2.E2Node

	connections              map[string]string // nodeID -> endpoint

	provisionCallCount       int

	registrationCallCount    int

	connectionCallCount      int

	deregistrationCallCount  int

	listCallCount            int

	lastProvisionedSpec      nephoranv1.E2NodeSetSpec

	shouldFailProvision      bool

	shouldFailConnection     bool

	shouldFailRegistration   bool

	shouldFailDeregistration bool

	shouldFailList           bool

}



// NewFakeE2Manager creates a new fake E2Manager for testing.

func NewFakeE2Manager() *FakeE2Manager {

	return &FakeE2Manager{

		nodes:       make(map[string]*e2.E2Node),

		connections: make(map[string]string),

	}

}



// ProvisionNode performs provisionnode operation.

func (f *FakeE2Manager) ProvisionNode(ctx context.Context, spec nephoranv1.E2NodeSetSpec) error {

	f.mutex.Lock()

	defer f.mutex.Unlock()



	f.provisionCallCount++

	f.lastProvisionedSpec = spec



	if f.shouldFailProvision {

		return fmt.Errorf("fake provision failure")

	}

	return nil

}



// SetupE2Connection performs setupe2connection operation.

func (f *FakeE2Manager) SetupE2Connection(nodeID string, endpoint string) error {

	f.mutex.Lock()

	defer f.mutex.Unlock()



	f.connectionCallCount++

	if f.shouldFailConnection {

		return fmt.Errorf("fake connection failure for node %s", nodeID)

	}



	f.connections[nodeID] = endpoint

	return nil

}



// RegisterE2Node performs registere2node operation.

func (f *FakeE2Manager) RegisterE2Node(ctx context.Context, nodeID string, ranFunctions []e2.RanFunction) error {

	f.mutex.Lock()

	defer f.mutex.Unlock()



	f.registrationCallCount++

	if f.shouldFailRegistration {

		return fmt.Errorf("fake registration failure for node %s", nodeID)

	}



	node := &e2.E2Node{

		NodeID:       nodeID,

		RanFunctions: make([]*e2.E2NodeFunction, len(ranFunctions)),

		ConnectionStatus: e2.E2ConnectionStatus{

			State: "CONNECTED",

		},

		HealthStatus: e2.NodeHealth{

			NodeID:    nodeID,

			Status:    "HEALTHY",

			LastCheck: time.Now(),

			Functions: make(map[int]*e2.FunctionHealth),

		},

		LastSeen: time.Now(),

	}



	// Convert RanFunction to E2NodeFunction.

	for i, rf := range ranFunctions {

		node.RanFunctions[i] = &e2.E2NodeFunction{

			FunctionID:          rf.FunctionID,

			FunctionDefinition:  rf.FunctionDefinition,

			FunctionRevision:    rf.FunctionRevision,

			FunctionOID:         rf.FunctionOID,

			FunctionDescription: rf.FunctionDescription,

			ServiceModel:        rf.ServiceModel,

			Status: e2.E2NodeFunctionStatus{

				State:         "ACTIVE",

				LastHeartbeat: time.Now(),

			},

		}

	}



	f.nodes[nodeID] = node

	return nil

}



// DeregisterE2Node performs deregistere2node operation.

func (f *FakeE2Manager) DeregisterE2Node(ctx context.Context, nodeID string) error {

	f.mutex.Lock()

	defer f.mutex.Unlock()



	f.deregistrationCallCount++

	if f.shouldFailDeregistration {

		return fmt.Errorf("fake deregistration failure for node %s", nodeID)

	}



	delete(f.nodes, nodeID)

	delete(f.connections, nodeID)

	return nil

}



// ListE2Nodes performs liste2nodes operation.

func (f *FakeE2Manager) ListE2Nodes(ctx context.Context) ([]*e2.E2Node, error) {

	f.mutex.RLock()

	defer f.mutex.RUnlock()



	f.listCallCount++

	if f.shouldFailList {

		return nil, fmt.Errorf("fake list failure")

	}



	nodes := make([]*e2.E2Node, 0, len(f.nodes))

	for _, node := range f.nodes {

		nodes = append(nodes, node)

	}

	return nodes, nil

}



// Test helper methods.



// Reset performs reset operation.

func (f *FakeE2Manager) Reset() {

	f.mutex.Lock()

	defer f.mutex.Unlock()



	f.nodes = make(map[string]*e2.E2Node)

	f.connections = make(map[string]string)

	f.shouldFailProvision = false

	f.shouldFailConnection = false

	f.shouldFailRegistration = false

	f.shouldFailDeregistration = false

	f.shouldFailList = false

	f.provisionCallCount = 0

	f.registrationCallCount = 0

	f.connectionCallCount = 0

	f.deregistrationCallCount = 0

	f.listCallCount = 0

}



// SetShouldFailProvision performs setshouldfailprovision operation.

func (f *FakeE2Manager) SetShouldFailProvision(fail bool) {

	f.mutex.Lock()

	defer f.mutex.Unlock()

	f.shouldFailProvision = fail

}



// SetShouldFailConnection performs setshouldfailconnection operation.

func (f *FakeE2Manager) SetShouldFailConnection(fail bool) {

	f.mutex.Lock()

	defer f.mutex.Unlock()

	f.shouldFailConnection = fail

}



// SetShouldFailRegistration performs setshouldfailregistration operation.

func (f *FakeE2Manager) SetShouldFailRegistration(fail bool) {

	f.mutex.Lock()

	defer f.mutex.Unlock()

	f.shouldFailRegistration = fail

}



// SetShouldFailDeregistration performs setshouldfailderegistration operation.

func (f *FakeE2Manager) SetShouldFailDeregistration(fail bool) {

	f.mutex.Lock()

	defer f.mutex.Unlock()

	f.shouldFailDeregistration = fail

}



// SetShouldFailList performs setshouldfaillist operation.

func (f *FakeE2Manager) SetShouldFailList(fail bool) {

	f.mutex.Lock()

	defer f.mutex.Unlock()

	f.shouldFailList = fail

}



// Getters for test verification.



// GetProvisionCallCount performs getprovisioncallcount operation.

func (f *FakeE2Manager) GetProvisionCallCount() int {

	f.mutex.RLock()

	defer f.mutex.RUnlock()

	return f.provisionCallCount

}



// GetRegistrationCallCount performs getregistrationcallcount operation.

func (f *FakeE2Manager) GetRegistrationCallCount() int {

	f.mutex.RLock()

	defer f.mutex.RUnlock()

	return f.registrationCallCount

}



// GetConnectionCallCount performs getconnectioncallcount operation.

func (f *FakeE2Manager) GetConnectionCallCount() int {

	f.mutex.RLock()

	defer f.mutex.RUnlock()

	return f.connectionCallCount

}



// GetDeregistrationCallCount performs getderegistrationcallcount operation.

func (f *FakeE2Manager) GetDeregistrationCallCount() int {

	f.mutex.RLock()

	defer f.mutex.RUnlock()

	return f.deregistrationCallCount

}



// GetListCallCount performs getlistcallcount operation.

func (f *FakeE2Manager) GetListCallCount() int {

	f.mutex.RLock()

	defer f.mutex.RUnlock()

	return f.listCallCount

}



// GetLastProvisionedSpec performs getlastprovisionedspec operation.

func (f *FakeE2Manager) GetLastProvisionedSpec() nephoranv1.E2NodeSetSpec {

	f.mutex.RLock()

	defer f.mutex.RUnlock()

	return f.lastProvisionedSpec

}



// GetNodes performs getnodes operation.

func (f *FakeE2Manager) GetNodes() map[string]*e2.E2Node {

	f.mutex.RLock()

	defer f.mutex.RUnlock()



	// Return a copy to avoid race conditions.

	nodes := make(map[string]*e2.E2Node)

	for k, v := range f.nodes {

		nodes[k] = v

	}

	return nodes

}



// GetConnections performs getconnections operation.

func (f *FakeE2Manager) GetConnections() map[string]string {

	f.mutex.RLock()

	defer f.mutex.RUnlock()



	// Return a copy to avoid race conditions.

	connections := make(map[string]string)

	for k, v := range f.connections {

		connections[k] = v

	}

	return connections

}



// AddNode adds a node directly to the fake manager for testing scenarios.

func (f *FakeE2Manager) AddNode(nodeID string, node *e2.E2Node) {

	f.mutex.Lock()

	defer f.mutex.Unlock()

	f.nodes[nodeID] = node

}

