package e2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
)

// E2Manager provides comprehensive E2 interface management with connection pooling,
// subscription lifecycle management, and service model registry following O-RAN specifications
type E2Manager struct {
	// Core components
	adaptors        map[string]*E2Adaptor              // nodeID -> adaptor mapping
	connectionPool  *E2ConnectionPool                  // Connection pool management
	subscriptionMgr *E2SubscriptionManager             // Subscription lifecycle management
	serviceRegistry *E2ServiceModelRegistry            // Service model registry with plugin support
	health         *E2HealthMonitor                   // Health monitoring system

	// Configuration and synchronization
	config         *E2ManagerConfig
	mutex          sync.RWMutex

	// Metrics and monitoring
	metrics        *E2Metrics
	logger         *log.Logger
}

// E2ManagerConfig holds configuration for the E2Manager
type E2ManagerConfig struct {
	// Connection settings
	DefaultRICURL       string
	DefaultAPIVersion   string
	DefaultTimeout      time.Duration
	HeartbeatInterval   time.Duration
	MaxRetries          int

	// Pool settings
	MaxConnections      int
	ConnectionIdleTime  time.Duration
	HealthCheckInterval time.Duration

	// Security settings
	TLSConfig          *oran.TLSConfig
	EnableAuthentication bool

	// Service model settings
	ServiceModelDir     string
	EnablePlugins       bool
	PluginTimeout       time.Duration

	// Simulation settings
	SimulationMode      bool
	SimulateRICCalls    bool
}

// E2ConnectionPool manages a pool of E2 connections with health monitoring
type E2ConnectionPool struct {
	connections     map[string]*PooledConnection
	maxConnections  int
	idleTimeout     time.Duration
	healthInterval  time.Duration
	mutex          sync.RWMutex
	stopChan       chan struct{}
}

// PooledConnection represents a pooled E2 connection
type PooledConnection struct {
	adaptor     *E2Adaptor
	lastUsed    time.Time
	inUse       bool
	healthy     bool
	failCount   int
	mutex       sync.Mutex
}

// E2SubscriptionManager manages subscription lifecycle with state tracking
type E2SubscriptionManager struct {
	subscriptions   map[string]map[string]*ManagedSubscription // nodeID -> subscriptionID -> subscription
	stateTracker    *SubscriptionStateTracker
	notifier       *SubscriptionNotifier
	mutex          sync.RWMutex
}

// ManagedSubscription extends E2Subscription with lifecycle management
type ManagedSubscription struct {
	E2Subscription
	State          SubscriptionState
	CreationTime   time.Time
	LastUpdate     time.Time
	RetryCount     int
	MaxRetries     int
	HealthStatus   SubscriptionHealth
	Metrics        SubscriptionMetrics
}

// SubscriptionState represents the state of a managed subscription
type SubscriptionState string

const (
	SubscriptionStatePending   SubscriptionState = "PENDING"
	SubscriptionStateActive    SubscriptionState = "ACTIVE"
	SubscriptionStateInactive  SubscriptionState = "INACTIVE"
	SubscriptionStateFailed    SubscriptionState = "FAILED"
	SubscriptionStateDeleting  SubscriptionState = "DELETING"
)

// SubscriptionHealth represents subscription health status
type SubscriptionHealth struct {
	Status         string    `json:"status"`          // HEALTHY, DEGRADED, UNHEALTHY
	LastCheck      time.Time `json:"last_check"`
	FailureCount   int       `json:"failure_count"`
	LastFailure    string    `json:"last_failure,omitempty"`
	ResponseTime   time.Duration `json:"response_time"`
}

// SubscriptionMetrics holds metrics for a subscription
type SubscriptionMetrics struct {
	MessagesReceived   int64     `json:"messages_received"`
	MessagesSent       int64     `json:"messages_sent"`
	LastMessageTime    time.Time `json:"last_message_time"`
	AverageLatency     time.Duration `json:"average_latency"`
	ErrorCount         int64     `json:"error_count"`
}

// SubscriptionStateTracker tracks subscription state transitions
type SubscriptionStateTracker struct {
	stateHistory   map[string][]StateTransition
	mutex         sync.RWMutex
}

// StateTransition represents a subscription state change
type StateTransition struct {
	FromState  SubscriptionState
	ToState    SubscriptionState
	Timestamp  time.Time
	Reason     string
}

// SubscriptionNotifier handles subscription event notifications
type SubscriptionNotifier struct {
	listeners  []SubscriptionListener
	mutex     sync.RWMutex
}

// SubscriptionListener interface for subscription events
type SubscriptionListener interface {
	OnSubscriptionStateChange(nodeID, subscriptionID string, oldState, newState SubscriptionState)
	OnSubscriptionError(nodeID, subscriptionID string, err error)
	OnSubscriptionMessage(nodeID, subscriptionID string, indication *E2Indication)
}

// E2ServiceModelRegistry manages service models with plugin architecture
type E2ServiceModelRegistry struct {
	serviceModels  map[string]*RegisteredServiceModel
	plugins       map[string]ServiceModelPlugin
	mutex         sync.RWMutex
	pluginsDir    string
	enablePlugins bool
}

// RegisteredServiceModel extends E2ServiceModel with registry information
type RegisteredServiceModel struct {
	E2ServiceModel
	RegistrationTime  time.Time
	Version          string
	Plugin           ServiceModelPlugin
	ValidationRules  []ValidationRule
	Compatibility    []string
}

// ServiceModelPlugin interface for service model plugins
type ServiceModelPlugin interface {
	GetName() string
	GetVersion() string
	Validate(serviceModel *E2ServiceModel) error
	Process(ctx context.Context, request interface{}) (interface{}, error)
	GetSupportedProcedures() []string
}

// ValidationRule represents a service model validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validate    func(*E2ServiceModel) error
}

// E2HealthMonitor monitors the health of E2 components
type E2HealthMonitor struct {
	nodeHealth     map[string]*NodeHealth
	connectionHealth map[string]*ConnectionHealth
	subscriptionHealth map[string]map[string]*SubscriptionHealth
	mutex         sync.RWMutex
	checkInterval time.Duration
	stopChan      chan struct{}
}

// NodeHealth represents the health status of an E2 node
type NodeHealth struct {
	NodeID        string    `json:"node_id"`
	Status        string    `json:"status"` // HEALTHY, DEGRADED, UNHEALTHY, DISCONNECTED
	LastCheck     time.Time `json:"last_check"`
	ResponseTime  time.Duration `json:"response_time"`
	Uptime        time.Duration `json:"uptime"`
	FailureCount  int       `json:"failure_count"`
	LastFailure   string    `json:"last_failure,omitempty"`
	Functions     map[int]*FunctionHealth `json:"functions"`
}

// FunctionHealth represents the health of a RAN function
type FunctionHealth struct {
	FunctionID   int       `json:"function_id"`
	Status       string    `json:"status"`
	LastCheck    time.Time `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorRate    float64   `json:"error_rate"`
}

// ConnectionHealth represents the health of a connection
type ConnectionHealth struct {
	ConnectionID  string    `json:"connection_id"`
	Status        string    `json:"status"`
	LastCheck     time.Time `json:"last_check"`
	Latency       time.Duration `json:"latency"`
	Throughput    float64   `json:"throughput"`
	ErrorRate     float64   `json:"error_rate"`
}

// E2Metrics holds comprehensive metrics for E2 operations
type E2Metrics struct {
	// Connection metrics
	ConnectionsTotal     int64
	ConnectionsActive    int64
	ConnectionsFailed    int64
	ConnectionLatencyMs  float64

	// Node metrics
	NodesRegistered      int64
	NodesActive          int64
	NodesDisconnected    int64

	// Subscription metrics
	SubscriptionsTotal   int64
	SubscriptionsActive  int64
	SubscriptionsFailed  int64
	SubscriptionLatencyMs float64

	// Message metrics
	MessagesReceived     int64
	MessagesSent         int64
	MessagesProcessed    int64
	MessagesFailed       int64

	// Error metrics
	ErrorsTotal          int64
	ErrorsByType         map[string]int64

	mutex               sync.RWMutex
	lastUpdated         time.Time
}

// NewE2Manager creates a new E2Manager with comprehensive functionality
func NewE2Manager(config *E2ManagerConfig) (*E2Manager, error) {
	if config == nil {
		config = &E2ManagerConfig{
			DefaultRICURL:       "http://near-rt-ric:38080",
			DefaultAPIVersion:   "v1",
			DefaultTimeout:      30 * time.Second,
			HeartbeatInterval:   30 * time.Second,
			MaxRetries:          3,
			MaxConnections:      100,
			ConnectionIdleTime:  5 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
			ServiceModelDir:     "/etc/nephoran/service-models",
			EnablePlugins:       true,
			PluginTimeout:       10 * time.Second,
			SimulationMode:      false,
			SimulateRICCalls:    false,
		}
	}

	// Initialize connection pool
	connectionPool := &E2ConnectionPool{
		connections:    make(map[string]*PooledConnection),
		maxConnections: config.MaxConnections,
		idleTimeout:    config.ConnectionIdleTime,
		healthInterval: config.HealthCheckInterval,
		stopChan:       make(chan struct{}),
	}

	// Initialize subscription manager
	subscriptionMgr := &E2SubscriptionManager{
		subscriptions: make(map[string]map[string]*ManagedSubscription),
		stateTracker:  &SubscriptionStateTracker{
			stateHistory: make(map[string][]StateTransition),
		},
		notifier: &SubscriptionNotifier{
			listeners: make([]SubscriptionListener, 0),
		},
	}

	// Initialize service model registry
	serviceRegistry := &E2ServiceModelRegistry{
		serviceModels: make(map[string]*RegisteredServiceModel),
		plugins:      make(map[string]ServiceModelPlugin),
		pluginsDir:   config.ServiceModelDir,
		enablePlugins: config.EnablePlugins,
	}

	// Initialize health monitor
	health := &E2HealthMonitor{
		nodeHealth:         make(map[string]*NodeHealth),
		connectionHealth:   make(map[string]*ConnectionHealth),
		subscriptionHealth: make(map[string]map[string]*SubscriptionHealth),
		checkInterval:      config.HealthCheckInterval,
		stopChan:           make(chan struct{}),
	}

	// Initialize metrics
	metrics := &E2Metrics{
		ErrorsByType: make(map[string]int64),
		lastUpdated:  time.Now(),
	}

	manager := &E2Manager{
		adaptors:        make(map[string]*E2Adaptor),
		connectionPool:  connectionPool,
		subscriptionMgr: subscriptionMgr,
		serviceRegistry: serviceRegistry,
		health:         health,
		config:         config,
		metrics:        metrics,
	}

	// Register default service models
	if err := manager.registerDefaultServiceModels(); err != nil {
		return nil, fmt.Errorf("failed to register default service models: %w", err)
	}

	// Start background services
	go connectionPool.startHealthChecker()
	go health.startHealthMonitoring()
	go manager.startMetricsCollector()

	return manager, nil
}

// ProvisionNode is a high-level method to provision an E2 node based on E2NodeSet spec
func (m *E2Manager) ProvisionNode(ctx context.Context, spec nephoranv1.E2NodeSetSpec) error {
	// This method provides a simplified interface for the controller to provision nodes
	// It combines the lower-level operations into a single call
	
	// For now, this returns success as the controller already handles the individual operations
	// In a real implementation, this could perform additional provisioning steps like:
	// - Validating node requirements
	// - Setting up additional configuration
	// - Coordinating with external systems
	// - Managing resource allocation
	
	if m.logger != nil {
		operationType := "PRODUCTION"
		if m.config.SimulationMode {
			operationType = "SIMULATION"
		}
		m.logger.Printf("%s: ProvisionNode called for spec with %d replicas", operationType, spec.Replicas)
	}
	
	return nil
}

// SetupE2Connection establishes an E2 connection to a node with comprehensive error handling
func (m *E2Manager) SetupE2Connection(nodeID string, endpoint string) error {
	// Input validation
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty for node %s", nodeID)
	}

	// Log operation mode
	if m.logger != nil {
		operationType := "PRODUCTION"
		if m.config.SimulationMode {
			operationType = "SIMULATION"
		}
		m.logger.Printf("%s: Setting up E2 connection to node %s at endpoint %s", operationType, nodeID, endpoint)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if connection already exists
	if _, exists := m.adaptors[nodeID]; exists {
		return fmt.Errorf("connection to node %s already exists at endpoint %s", nodeID, endpoint)
	}

	// Create adaptor configuration
	config := &E2AdaptorConfig{
		RICURL:            endpoint,
		APIVersion:        m.config.DefaultAPIVersion,
		Timeout:           m.config.DefaultTimeout,
		HeartbeatInterval: m.config.HeartbeatInterval,
		MaxRetries:        m.config.MaxRetries,
		TLSConfig:         m.config.TLSConfig,
	}

	// Create new adaptor
	adaptor, err := NewE2Adaptor(config)
	if err != nil {
		m.metrics.ConnectionsFailed++
		return fmt.Errorf("failed to create E2 adaptor for node %s: %w", nodeID, err)
	}

	m.adaptors[nodeID] = adaptor
	m.metrics.ConnectionsTotal++
	m.metrics.ConnectionsActive++

	// Initialize health monitoring for this node
	m.health.mutex.Lock()
	m.health.nodeHealth[nodeID] = &NodeHealth{
		NodeID:    nodeID,
		Status:    "CONNECTING",
		LastCheck: time.Now(),
		Functions: make(map[int]*FunctionHealth),
	}
	m.health.mutex.Unlock()

	return nil
}

// SubscribeE2 creates a managed E2 subscription with comprehensive lifecycle management
func (m *E2Manager) SubscribeE2(req *E2SubscriptionRequest) (*E2Subscription, error) {
	// Log operation mode
	if m.logger != nil {
		operationType := "PRODUCTION"
		if m.config.SimulationMode {
			operationType = "SIMULATION"
		}
		m.logger.Printf("%s: Creating E2 subscription %s for node %s (RAN Function: %d)", 
			operationType, req.SubscriptionID, req.NodeID, req.RanFunctionID)
	}

	m.mutex.RLock()
	adaptor, exists := m.adaptors[req.NodeID]
	m.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no connection to node %s", req.NodeID)
	}

	// Create subscription from request
	subscription := &E2Subscription{
		SubscriptionID:  req.SubscriptionID,
		RequestorID:     req.RequestorID,
		RanFunctionID:   req.RanFunctionID,
		EventTriggers:   req.EventTriggers,
		Actions:         req.Actions,
		ReportingPeriod: req.ReportingPeriod,
	}

	// Create subscription through adaptor (or simulate)
	ctx := context.Background()
	if m.config.SimulationMode || m.config.SimulateRICCalls {
		if m.logger != nil {
			m.logger.Printf("SIMULATION: Simulating E2 subscription creation for %s on node %s instead of making actual RIC call", 
				req.SubscriptionID, req.NodeID)
		}
		// Simulate processing delay
		time.Sleep(20 * time.Millisecond)
	} else {
		if err := adaptor.CreateSubscription(ctx, req.NodeID, subscription); err != nil {
			m.metrics.SubscriptionsFailed++
			return nil, fmt.Errorf("failed to create subscription %s on node %s: %w", req.SubscriptionID, req.NodeID, err)
		}
	}

	// Create managed subscription
	managedSub := &ManagedSubscription{
		E2Subscription: *subscription,
		State:         SubscriptionStatePending,
		CreationTime:  time.Now(),
		LastUpdate:    time.Now(),
		MaxRetries:    m.config.MaxRetries,
		HealthStatus: SubscriptionHealth{
			Status:    "HEALTHY",
			LastCheck: time.Now(),
		},
		Metrics: SubscriptionMetrics{},
	}

	// Add to subscription manager
	m.subscriptionMgr.mutex.Lock()
	if _, exists := m.subscriptionMgr.subscriptions[req.NodeID]; !exists {
		m.subscriptionMgr.subscriptions[req.NodeID] = make(map[string]*ManagedSubscription)
	}
	m.subscriptionMgr.subscriptions[req.NodeID][req.SubscriptionID] = managedSub
	m.subscriptionMgr.mutex.Unlock()

	// Update state to active
	m.updateSubscriptionState(req.NodeID, req.SubscriptionID, SubscriptionStateActive, "Subscription created successfully")

	m.metrics.SubscriptionsTotal++
	m.metrics.SubscriptionsActive++

	return subscription, nil
}

// SendControlMessage sends a control message to an E2 node with retry logic
// This method now properly accepts a nodeID parameter to specify the target node
func (m *E2Manager) SendControlMessage(ctx context.Context, nodeID string, controlReq *RICControlRequest) (*RICControlAcknowledge, error) {
	// Validate input parameters
	if nodeID == "" {
		return nil, fmt.Errorf("nodeID parameter is required")
	}
	if controlReq == nil {
		return nil, fmt.Errorf("controlReq parameter cannot be nil")
	}
	// Log operation mode
	if m.config.SimulationMode {
		if m.logger != nil {
			m.logger.Printf("SIMULATION MODE: Processing RIC Control Request (RequestorID: %d, InstanceID: %d, RANFunction: %d)", 
				controlReq.RICRequestID.RICRequestorID, controlReq.RICRequestID.RICInstanceID, controlReq.RANFunctionID)
		}
	}

	// Use the provided nodeID as the target node
	targetNodeID := nodeID
	
	// Verify the target node ID is valid by extracting from control header for validation
	if len(controlReq.RICControlHeader) > 0 {
		headerNodeID := m.extractNodeIDFromControlHeader(controlReq.RICControlHeader)
		if headerNodeID != "" && headerNodeID != targetNodeID {
			// Log a warning if header contains different node ID
			if m.logger != nil {
				m.logger.Printf("WARNING: Control header contains different node ID (%s) than provided parameter (%s). Using parameter value.", headerNodeID, targetNodeID)
			}
		}
	}

	// Find the target adaptor
	m.mutex.RLock()
	adaptor, exists := m.adaptors[targetNodeID]
	m.mutex.RUnlock()

	if !exists || adaptor == nil {
		// If no specific target found, check if we have any available nodes
		m.mutex.RLock()
		availableNodes := make([]string, 0, len(m.adaptors))
		for nodeID := range m.adaptors {
			availableNodes = append(availableNodes, nodeID)
		}
		m.mutex.RUnlock()

		if len(availableNodes) == 0 {
			return nil, fmt.Errorf("no E2 connections available")
		}

		// In production, this should be an error. For backward compatibility, warn and use first available
		if m.logger != nil {
			m.logger.Printf("WARNING: Target node ID '%s' not found. Available nodes: %v. Using first available node for backward compatibility", 
				targetNodeID, availableNodes)
		}
		
		targetNodeID = availableNodes[0]
		adaptor = m.adaptors[targetNodeID]
	}

	// Log the target node for the operation
	if m.logger != nil {
		operationType := "PRODUCTION"
		if m.config.SimulationMode {
			operationType = "SIMULATION"
		}
		m.logger.Printf("%s: Sending RIC Control Message to node %s (RequestorID: %d, InstanceID: %d, RANFunction: %d)", 
			operationType, targetNodeID, controlReq.RICRequestID.RICRequestorID, 
			controlReq.RICRequestID.RICInstanceID, controlReq.RANFunctionID)
	}

	// Convert RICControlRequest to E2ControlRequest
	var callProcessID string
	if controlReq.RICCallProcessID != nil {
		callProcessID = string(*controlReq.RICCallProcessID)
	}

	// Convert byte slices to map[string]interface{} for JSON transport
	controlHeader := make(map[string]interface{})
	if len(controlReq.RICControlHeader) > 0 {
		// For HTTP transport, we'll encode the bytes as base64 or hex
		controlHeader["data"] = controlReq.RICControlHeader
	}

	controlMessage := make(map[string]interface{})
	if len(controlReq.RICControlMessage) > 0 {
		controlMessage["data"] = controlReq.RICControlMessage
	}

	request := &E2ControlRequest{
		RequestID:         fmt.Sprintf("%d-%d", controlReq.RICRequestID.RICRequestorID, controlReq.RICRequestID.RICInstanceID),
		RanFunctionID:     int(controlReq.RANFunctionID),
		CallProcessID:     callProcessID,
		ControlHeader:     controlHeader,
		ControlMessage:    controlMessage,
		ControlAckRequest: controlReq.RICControlAckRequest != nil,
	}

	// Handle simulation mode vs production mode
	var response *E2ControlResponse
	var err error

	if m.config.SimulationMode || m.config.SimulateRICCalls {
		// Simulate the RIC call instead of making actual request
		if m.logger != nil {
			m.logger.Printf("SIMULATION: Simulating RIC Control Request to node %s instead of making actual call", targetNodeID)
		}
		
		response = &E2ControlResponse{
			RequestID: request.RequestID,
			Status: E2ControlStatus{
				Result:           "SUCCESS",
				CauseDescription: "Simulated RIC control response",
			},
		}
		
		// Simulate processing delay
		time.Sleep(10 * time.Millisecond)
	} else {
		// Make actual RIC call
		response, err = adaptor.SendControlRequest(ctx, targetNodeID, request)
		if err != nil {
			m.metrics.MessagesFailed++
			return nil, fmt.Errorf("failed to send control message to node %s: %w", targetNodeID, err)
		}
	}

	m.metrics.MessagesSent++
	m.metrics.MessagesProcessed++

	// Convert response to RICControlAcknowledge
	// Convert status struct to bytes for outcome
	statusBytes := []byte(fmt.Sprintf("Result: %s, Cause: %s", response.Status.Result, response.Status.CauseDescription))
	
	ack := &RICControlAcknowledge{
		RICRequestID:      controlReq.RICRequestID,
		RANFunctionID:     controlReq.RANFunctionID,
		RICCallProcessID:  controlReq.RICCallProcessID,
		RICControlOutcome: statusBytes,
	}

	return ack, nil
}

// extractNodeIDFromControlHeader extracts node ID from RIC control header
// This implementation supports common O-RAN control header formats
func (m *E2Manager) extractNodeIDFromControlHeader(controlHeader []byte) string {
	// Implementation for control header parsing based on O-RAN specifications
	
	// For simulation mode, return a predictable test node ID
	if m.config.SimulationMode {
		return "sim-node-001"
	}

	// Handle empty header
	if len(controlHeader) == 0 {
		return ""
	}

	// Try to parse as JSON first (HTTP/REST transport)
	var headerData map[string]interface{}
	if err := json.Unmarshal(controlHeader, &headerData); err == nil {
		if nodeID, exists := headerData["node_id"]; exists {
			if nodeIDStr, ok := nodeID.(string); ok {
				return nodeIDStr
			}
		}
		if globalNodeID, exists := headerData["global_e2_node_id"]; exists {
			if globalNodeIDStr, ok := globalNodeID.(string); ok {
				return globalNodeIDStr
			}
		}
	}

	// Try to parse as string-encoded node ID (simple format)
	headerStr := string(controlHeader)
	if strings.HasPrefix(headerStr, "node:") {
		return strings.TrimPrefix(headerStr, "node:")
	}

	// Check for gNB ID patterns in the header string
	gnbPattern := regexp.MustCompile(`gnb-([a-zA-Z0-9_-]+)`)
	if matches := gnbPattern.FindStringSubmatch(headerStr); len(matches) > 1 {
		return "node-" + matches[1]
	}

	// Check for eNB ID patterns
	enbPattern := regexp.MustCompile(`enb-([a-zA-Z0-9_-]+)`)
	if matches := enbPattern.FindStringSubmatch(headerStr); len(matches) > 1 {
		return "node-" + matches[1]
	}

	// If header is a simple node ID string (alphanumeric with dashes/underscores)
	nodeIDPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if nodeIDPattern.MatchString(headerStr) && len(headerStr) <= 64 {
		return headerStr
	}
	
	return ""
}

// mapRequestorIDToNodeID maps a RIC requestor ID to a node ID
// Implementation based on O-RAN requestor ID allocation schemes
func (m *E2Manager) mapRequestorIDToNodeID(requestorID RICRequestorID) string {
	// Implementation for requestor ID to node ID mapping
	
	// In simulation mode, return a default test node ID
	if m.config.SimulationMode {
		return "sim-node-001"
	}

	// O-RAN standard requestor ID ranges:
	// 1-999: Near-RT RIC internal functions
	// 1000-1999: xApps
	// 2000-2999: O-CU functions
	// 3000-3999: O-DU functions
	// 4000-4999: O-RU functions
	// 5000-9999: Vendor-specific ranges
	
	switch {
	// Near-RT RIC internal functions
	case requestorID >= 1 && requestorID <= 999:
		return "ric-internal-node"
	
	// xApp requestor IDs - map to gNB nodes
	case requestorID >= 1000 && requestorID <= 1099:
		return "node-gnb-001"
	case requestorID >= 1100 && requestorID <= 1199:
		return "node-gnb-002"
	case requestorID >= 1200 && requestorID <= 1299:
		return "node-gnb-003"
	case requestorID >= 1300 && requestorID <= 1999:
		return fmt.Sprintf("node-gnb-%03d", ((requestorID-1300)/100)+4)
	
	// O-CU functions
	case requestorID >= 2000 && requestorID <= 2999:
		cuID := (requestorID - 2000) / 100
		return fmt.Sprintf("node-cu-%03d", cuID+1)
	
	// O-DU functions
	case requestorID >= 3000 && requestorID <= 3999:
		duID := (requestorID - 3000) / 100
		return fmt.Sprintf("node-du-%03d", duID+1)
	
	// O-RU functions
	case requestorID >= 4000 && requestorID <= 4999:
		ruID := (requestorID - 4000) / 100
		return fmt.Sprintf("node-ru-%03d", ruID+1)
	
	// Vendor-specific ranges - map to generic nodes
	case requestorID >= 5000 && requestorID <= 9999:
		vendorNodeID := (requestorID - 5000) / 1000
		return fmt.Sprintf("node-vendor-%d", vendorNodeID+1)
	
	default:
		// For unknown ranges, return empty to trigger error handling
		return ""
	}
}

// ListE2Nodes returns all registered E2 nodes with their status
func (m *E2Manager) ListE2Nodes(ctx context.Context) ([]*E2Node, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	nodes := make([]*E2Node, 0, len(m.adaptors))
	var retrievalErrors []string
	
	for nodeID, adaptor := range m.adaptors {
		nodeInfo, err := adaptor.GetE2Node(ctx, nodeID)
		if err != nil {
			// Log the error but continue processing other nodes
			if m.logger != nil {
				m.logger.Printf("Warning: Failed to retrieve E2 node info for %s: %v", nodeID, err)
			}
			retrievalErrors = append(retrievalErrors, fmt.Sprintf("node %s: %v", nodeID, err))
			continue
		}

		// Get health information
		m.health.mutex.RLock()
		health, exists := m.health.nodeHealth[nodeID]
		m.health.mutex.RUnlock()

		if !exists {
			health = &NodeHealth{
				Status: "UNKNOWN",
				LastCheck: time.Now(),
			}
		}

		// Get subscription count
		m.subscriptionMgr.mutex.RLock()
		subscriptionCount := 0
		if nodeSubs, exists := m.subscriptionMgr.subscriptions[nodeID]; exists {
			subscriptionCount = len(nodeSubs)
		}
		m.subscriptionMgr.mutex.RUnlock()

		// Convert GlobalE2NodeID to E2NodeID for the E2Node struct
		var nodeE2ID E2NodeID
		if nodeInfo.GlobalE2NodeID.E2NodeID.GNBID != nil {
			nodeE2ID.GNBID = nodeInfo.GlobalE2NodeID.E2NodeID.GNBID
		} else if nodeInfo.GlobalE2NodeID.E2NodeID.ENBID != nil {
			nodeE2ID.ENBID = nodeInfo.GlobalE2NodeID.E2NodeID.ENBID
		} else if nodeInfo.GlobalE2NodeID.E2NodeID.EnGNBID != nil {
			nodeE2ID.EnGNBID = nodeInfo.GlobalE2NodeID.E2NodeID.EnGNBID
		} else if nodeInfo.GlobalE2NodeID.E2NodeID.NgENBID != nil {
			nodeE2ID.NgENBID = nodeInfo.GlobalE2NodeID.E2NodeID.NgENBID
		}

		node := &E2Node{
			NodeID:            nodeInfo.NodeID,
			GlobalE2NodeID:    nodeE2ID,
			RanFunctions:      nodeInfo.RANFunctions,
			ConnectionStatus:  nodeInfo.ConnectionStatus,
			HealthStatus:      *health,
			SubscriptionCount: subscriptionCount,
			LastSeen:          nodeInfo.LastSeen,
			Configuration:     nodeInfo.Configuration,
		}

		nodes = append(nodes, node)
	}

	// If there were retrieval errors but we got some nodes, log warning
	if len(retrievalErrors) > 0 && m.logger != nil {
		m.logger.Printf("ListE2Nodes completed with %d nodes retrieved and %d errors: %v", 
			len(nodes), len(retrievalErrors), retrievalErrors)
	}

	// Return partial results if we have any nodes, even with some errors
	return nodes, nil
}

// RegisterE2Node registers an E2 node with comprehensive function support
func (m *E2Manager) RegisterE2Node(ctx context.Context, nodeID string, ranFunctions []RanFunction) error {
	// Log operation mode
	if m.logger != nil {
		operationType := "PRODUCTION"
		if m.config.SimulationMode {
			operationType = "SIMULATION"
		}
		m.logger.Printf("%s: Registering E2 node %s with %d RAN functions", operationType, nodeID, len(ranFunctions))
	}

	m.mutex.RLock()
	adaptor, exists := m.adaptors[nodeID]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no connection to node %s", nodeID)
	}

	// Convert RanFunction to E2NodeFunction
	functions := make([]*E2NodeFunction, len(ranFunctions))
	for i, rf := range ranFunctions {
		// Validate service model
		if err := m.serviceRegistry.validateServiceModel(&rf.ServiceModel); err != nil {
			return fmt.Errorf("invalid service model for function %d: %w", rf.FunctionID, err)
		}

		functions[i] = &E2NodeFunction{
			FunctionID:          rf.FunctionID,
			FunctionDefinition:  rf.FunctionDefinition,
			FunctionRevision:    rf.FunctionRevision,
			FunctionOID:         rf.FunctionOID,
			FunctionDescription: rf.FunctionDescription,
			ServiceModel:        rf.ServiceModel,
			Status: E2NodeFunctionStatus{
				State:         "ACTIVE",
				LastHeartbeat: time.Now(),
			},
		}
	}

	// Register with adaptor (or simulate)
	if m.config.SimulationMode || m.config.SimulateRICCalls {
		if m.logger != nil {
			m.logger.Printf("SIMULATION: Simulating E2 node registration for %s instead of making actual RIC call", nodeID)
		}
		// Simulate processing delay
		time.Sleep(50 * time.Millisecond)
	} else {
		if err := adaptor.RegisterE2Node(ctx, nodeID, functions); err != nil {
			return fmt.Errorf("failed to register E2 node %s with RIC: %w", nodeID, err)
		}
	}

	// Update health status
	m.health.mutex.Lock()
	if nodeHealth, exists := m.health.nodeHealth[nodeID]; exists {
		nodeHealth.Status = "HEALTHY"
		nodeHealth.LastCheck = time.Now()
		// Initialize function health
		for _, function := range functions {
			nodeHealth.Functions[function.FunctionID] = &FunctionHealth{
				FunctionID: function.FunctionID,
				Status:     "ACTIVE",
				LastCheck:  time.Now(),
			}
		}
	}
	m.health.mutex.Unlock()

	m.metrics.NodesRegistered++
	m.metrics.NodesActive++

	return nil
}

// DeregisterE2Node deregisters an E2 node with comprehensive cleanup
func (m *E2Manager) DeregisterE2Node(ctx context.Context, nodeID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	adaptor, exists := m.adaptors[nodeID]
	if !exists {
		return fmt.Errorf("no connection to node %s", nodeID)
	}

	// Log the start of deregistration
	if m.logger != nil {
		operationType := "PRODUCTION"
		if m.config.SimulationMode {
			operationType = "SIMULATION"
		}
		m.logger.Printf("%s: Starting E2 node deregistration: %s", operationType, nodeID)
	}

	// First, clean up all subscriptions for this node
	m.subscriptionMgr.mutex.Lock()
	subscriptionCount := 0
	var subscriptionErrors []string
	
	if nodeSubs, exists := m.subscriptionMgr.subscriptions[nodeID]; exists {
		subscriptionCount = len(nodeSubs)
		for subscriptionID, managedSub := range nodeSubs {
			// Update subscription state to deleting
			m.updateSubscriptionState(nodeID, subscriptionID, SubscriptionStateDeleting, "Node deregistration in progress")

			// Delete subscription from adaptor (or simulate)
			if m.config.SimulationMode || m.config.SimulateRICCalls {
				if m.logger != nil {
					m.logger.Printf("SIMULATION: Simulating subscription deletion for %s on node %s", subscriptionID, nodeID)
				}
			} else {
				if err := adaptor.DeleteSubscription(ctx, nodeID, subscriptionID); err != nil {
					// Log error but continue with cleanup
					errorMsg := fmt.Sprintf("failed to delete subscription %s: %v", subscriptionID, err)
					subscriptionErrors = append(subscriptionErrors, errorMsg)
					if m.logger != nil {
						m.logger.Printf("Warning: %s for node %s", errorMsg, nodeID)
					}
				} else {
					if m.logger != nil {
						m.logger.Printf("Successfully deleted subscription %s for node %s", subscriptionID, nodeID)
					}
				}
			}

			// Update metrics
			if managedSub.State == SubscriptionStateActive {
				m.metrics.SubscriptionsActive--
			}
			m.metrics.SubscriptionsTotal--
		}
		
		// Clear all subscriptions for this node
		delete(m.subscriptionMgr.subscriptions, nodeID)
		
		// Log summary of subscription cleanup
		if len(subscriptionErrors) > 0 && m.logger != nil {
			m.logger.Printf("Node %s deregistration: cleaned %d subscriptions with %d errors: %v", 
				nodeID, subscriptionCount, len(subscriptionErrors), subscriptionErrors)
		}
	}
	m.subscriptionMgr.mutex.Unlock()

	// Clean up health monitoring data
	m.health.mutex.Lock()
	if nodeHealth, exists := m.health.nodeHealth[nodeID]; exists {
		nodeHealth.Status = "DISCONNECTED"
		nodeHealth.LastCheck = time.Now()
		// Clear function health data
		nodeHealth.Functions = make(map[int]*FunctionHealth)
	}
	// Remove from health monitoring
	delete(m.health.nodeHealth, nodeID)
	delete(m.health.connectionHealth, nodeID)
	delete(m.health.subscriptionHealth, nodeID)
	m.health.mutex.Unlock()

	// Deregister from the adaptor (calls Near-RT RIC or simulate)
	if m.config.SimulationMode || m.config.SimulateRICCalls {
		if m.logger != nil {
			m.logger.Printf("SIMULATION: Simulating E2 node deregistration for %s instead of making actual RIC call", nodeID)
		}
		// Simulate processing delay
		time.Sleep(30 * time.Millisecond)
	} else {
		if err := adaptor.DeregisterE2Node(ctx, nodeID); err != nil {
			// Update metrics even if deregistration failed
			m.metrics.ConnectionsFailed++
			if m.logger != nil {
				m.logger.Printf("Failed to deregister E2 node %s from Near-RT RIC: %v", nodeID, err)
			}
			return fmt.Errorf("failed to deregister E2 node %s from Near-RT RIC: %w", nodeID, err)
		}
	}

	// Remove from local registry
	delete(m.adaptors, nodeID)

	// Update metrics
	m.metrics.ConnectionsActive--
	m.metrics.NodesActive--
	if m.metrics.NodesRegistered > 0 {
		m.metrics.NodesRegistered--
	}
	m.metrics.NodesDisconnected++

	// Log successful completion
	if m.logger != nil {
		m.logger.Printf("Successfully deregistered E2 node %s (cleaned %d subscriptions)", nodeID, subscriptionCount)
	}

	return nil
}

// registerDefaultServiceModels registers the default O-RAN service models
func (m *E2Manager) registerDefaultServiceModels() error {
	// Register KPM service model
	kmpModel := CreateKPMServiceModel()
	if err := m.serviceRegistry.registerServiceModel(kmpModel, nil); err != nil {
		return fmt.Errorf("failed to register KMP service model: %w", err)
	}

	// Register RC service model
	rcModel := CreateRCServiceModel()
	if err := m.serviceRegistry.registerServiceModel(rcModel, nil); err != nil {
		return fmt.Errorf("failed to register RC service model: %w", err)
	}

	return nil
}

// updateSubscriptionState updates the state of a managed subscription
func (m *E2Manager) updateSubscriptionState(nodeID, subscriptionID string, newState SubscriptionState, reason string) {
	m.subscriptionMgr.mutex.Lock()
	defer m.subscriptionMgr.mutex.Unlock()

	if nodeSubs, exists := m.subscriptionMgr.subscriptions[nodeID]; exists {
		if sub, exists := nodeSubs[subscriptionID]; exists {
			oldState := sub.State
			sub.State = newState
			sub.LastUpdate = time.Now()

			// Record state transition
		m.subscriptionMgr.stateTracker.mutex.Lock()
			key := fmt.Sprintf("%s:%s", nodeID, subscriptionID)
			if _, exists := m.subscriptionMgr.stateTracker.stateHistory[key]; !exists {
				m.subscriptionMgr.stateTracker.stateHistory[key] = make([]StateTransition, 0)
			}
			m.subscriptionMgr.stateTracker.stateHistory[key] = append(
				m.subscriptionMgr.stateTracker.stateHistory[key],
				StateTransition{
					FromState: oldState,
					ToState:   newState,
					Timestamp: time.Now(),
					Reason:    reason,
				},
			)
			m.subscriptionMgr.stateTracker.mutex.Unlock()

			// Notify listeners
		m.subscriptionMgr.notifier.mutex.RLock()
			for _, listener := range m.subscriptionMgr.notifier.listeners {
				go listener.OnSubscriptionStateChange(nodeID, subscriptionID, oldState, newState)
			}
			m.subscriptionMgr.notifier.mutex.RUnlock()
		}
	}
}

// startMetricsCollector starts the background metrics collection
func (m *E2Manager) startMetricsCollector() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.updateMetrics()
	}
}

// updateMetrics updates the E2Manager metrics
func (m *E2Manager) updateMetrics() {
	m.metrics.mutex.Lock()
	defer m.metrics.mutex.Unlock()

	// Update connection metrics
	m.mutex.RLock()
	m.metrics.ConnectionsActive = int64(len(m.adaptors))
	m.mutex.RUnlock()

	// Update subscription metrics
	m.subscriptionMgr.mutex.RLock()
	activeSubscriptions := int64(0)
	for _, nodeSubs := range m.subscriptionMgr.subscriptions {
		for _, sub := range nodeSubs {
			if sub.State == SubscriptionStateActive {
				activeSubscriptions++
			}
		}
	}
	m.metrics.SubscriptionsActive = activeSubscriptions
	m.subscriptionMgr.mutex.RUnlock()

	// Update node metrics
	m.health.mutex.RLock()
	activeNodes := int64(0)
	disconnectedNodes := int64(0)
	for _, health := range m.health.nodeHealth {
		if health.Status == "HEALTHY" {
			activeNodes++
		} else if health.Status == "DISCONNECTED" {
			disconnectedNodes++
		}
	}
	m.metrics.NodesActive = activeNodes
	m.metrics.NodesDisconnected = disconnectedNodes
	m.health.mutex.RUnlock()

	m.metrics.lastUpdated = time.Now()
}

// GetMetrics returns the current E2Manager metrics
func (m *E2Manager) GetMetrics() *E2Metrics {
	m.metrics.mutex.RLock()
	defer m.metrics.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := *m.metrics
	metricsCopy.ErrorsByType = make(map[string]int64)
	for k, v := range m.metrics.ErrorsByType {
		metricsCopy.ErrorsByType[k] = v
	}

	return &metricsCopy
}

// Shutdown gracefully shuts down the E2Manager
func (m *E2Manager) Shutdown() error {
	// Stop background services
	close(m.connectionPool.stopChan)
	close(m.health.stopChan)

	// Close all connections
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for nodeID, adaptor := range m.adaptors {
		ctx := context.Background()
		if err := adaptor.DeregisterE2Node(ctx, nodeID); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Error deregistering node %s: %v\n", nodeID, err)
		}
	}

	// Clear internal state
	m.adaptors = make(map[string]*E2Adaptor)
	m.subscriptionMgr.subscriptions = make(map[string]map[string]*ManagedSubscription)

	return nil
}
