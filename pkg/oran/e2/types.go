package e2

import (
	"context"
	"fmt"
	"time"
)

// Additional types and interfaces for E2 implementation

// E2SubscriptionRequest represents a request to create an E2 subscription
type E2SubscriptionRequest struct {
	NodeID          string            `json:"node_id"`
	SubscriptionID  string            `json:"subscription_id"`
	RequestorID     string            `json:"requestor_id"`
	RanFunctionID   int               `json:"ran_function_id"`
	EventTriggers   []E2EventTrigger  `json:"event_triggers"`
	Actions         []E2Action        `json:"actions"`
	ReportingPeriod time.Duration     `json:"reporting_period"`
}

// E2ControlMessage represents a control message to be sent to an E2 node
type E2ControlMessage struct {
	NodeID            string                 `json:"node_id"`
	RequestID         string                 `json:"request_id"`
	RanFunctionID     int                    `json:"ran_function_id"`
	CallProcessID     string                 `json:"call_process_id,omitempty"`
	ControlHeader     map[string]interface{} `json:"control_header"`
	ControlMessage    map[string]interface{} `json:"control_message"`
	ControlAckRequest bool                   `json:"control_ack_request"`
	Response          *E2ControlResponse     `json:"response,omitempty"`
}

// E2Node represents an E2 node with enhanced information
type E2Node struct {
	NodeID            string              `json:"node_id"`
	GlobalE2NodeID    E2NodeID            `json:"global_e2_node_id"`
	RanFunctions      []*E2NodeFunction   `json:"ran_functions"`
	ConnectionStatus  E2ConnectionStatus  `json:"connection_status"`
	HealthStatus      NodeHealth          `json:"health_status"`
	SubscriptionCount int                 `json:"subscription_count"`
	LastSeen          time.Time           `json:"last_seen"`
	Configuration     map[string]interface{} `json:"configuration,omitempty"`
}

// RanFunction represents a RAN function for registration
type RanFunction struct {
	FunctionID          int              `json:"function_id"`
	FunctionDefinition  string           `json:"function_definition"`
	FunctionRevision    int              `json:"function_revision"`
	FunctionOID         string           `json:"function_oid"`
	FunctionDescription string           `json:"function_description"`
	ServiceModel        E2ServiceModel   `json:"service_model"`
}

// Connection pool methods for E2ConnectionPool

// getConnection gets a connection from the pool
func (p *E2ConnectionPool) getConnection(nodeID string) (*PooledConnection, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	conn, exists := p.connections[nodeID]
	if !exists {
		return nil, fmt.Errorf("no connection for node %s", nodeID)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.inUse {
		return nil, fmt.Errorf("connection for node %s is in use", nodeID)
	}

	if !conn.healthy {
		return nil, fmt.Errorf("connection for node %s is unhealthy", nodeID)
	}

	conn.inUse = true
	conn.lastUsed = time.Now()

	return conn, nil
}

// releaseConnection releases a connection back to the pool
func (p *E2ConnectionPool) releaseConnection(nodeID string) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	conn, exists := p.connections[nodeID]
	if !exists {
		return
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.inUse = false
	conn.lastUsed = time.Now()
}

// addConnection adds a new connection to the pool
func (p *E2ConnectionPool) addConnection(nodeID string, adaptor *E2Adaptor) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.connections) >= p.maxConnections {
		return fmt.Errorf("connection pool is full")
	}

	p.connections[nodeID] = &PooledConnection{
		adaptor:  adaptor,
		lastUsed: time.Now(),
		inUse:    false,
		healthy:  true,
		failCount: 0,
	}

	return nil
}

// removeConnection removes a connection from the pool
func (p *E2ConnectionPool) removeConnection(nodeID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.connections, nodeID)
}

// startHealthChecker starts the background health checker for the connection pool
func (p *E2ConnectionPool) startHealthChecker() {
	ticker := time.NewTicker(p.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.checkConnections()
		case <-p.stopChan:
			return
		}
	}
}

// checkConnections performs health checks on all connections
func (p *E2ConnectionPool) checkConnections() {
	p.mutex.RLock()
	connections := make(map[string]*PooledConnection)
	for k, v := range p.connections {
		connections[k] = v
	}
	p.mutex.RUnlock()

	for nodeID, conn := range connections {
		go p.checkConnection(nodeID, conn)
	}
}

// checkConnection performs a health check on a single connection
func (p *E2ConnectionPool) checkConnection(nodeID string, conn *PooledConnection) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// Skip if connection is in use
	if conn.inUse {
		return
	}

	// Check if connection is idle for too long
	if time.Since(conn.lastUsed) > p.idleTimeout {
		// Mark as unhealthy but don't remove yet
		conn.healthy = false
		return
	}

	// Perform actual health check (simplified for HTTP transport)
	ctx := context.Background()
	nodes, err := conn.adaptor.ListE2Nodes(ctx)
	if err != nil {
		conn.failCount++
		if conn.failCount > 3 {
			conn.healthy = false
		}
	} else {
		conn.failCount = 0
		conn.healthy = true
		// Check if our node is in the list
		found := false
		for _, node := range nodes {
			if node.NodeID == nodeID {
				found = true
				break
			}
		}
		if !found {
			conn.healthy = false
		}
	}
}

// Health monitor methods for E2HealthMonitor

// startHealthMonitoring starts the background health monitoring
func (h *E2HealthMonitor) startHealthMonitoring() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.performHealthChecks()
		case <-h.stopChan:
			return
		}
	}
}

// performHealthChecks performs health checks on all monitored components
func (h *E2HealthMonitor) performHealthChecks() {
	h.mutex.RLock()
	nodes := make(map[string]*NodeHealth)
	for k, v := range h.nodeHealth {
		nodes[k] = v
	}
	h.mutex.RUnlock()

	for nodeID, health := range nodes {
		go h.checkNodeHealth(nodeID, health)
	}
}

// checkNodeHealth performs a health check on a single node
func (h *E2HealthMonitor) checkNodeHealth(nodeID string, health *NodeHealth) {
	start := time.Now()

	// Perform health check (simplified)
	// In a real implementation, this would ping the node or check its status
	err := h.pingNode(nodeID)

	health.LastCheck = time.Now()
	health.ResponseTime = time.Since(start)

	if err != nil {
		health.FailureCount++
		health.LastFailure = err.Error()
		
		// Update status based on failure count
		if health.FailureCount >= 3 {
			health.Status = "UNHEALTHY"
		} else if health.FailureCount >= 1 {
			health.Status = "DEGRADED"
		}
	} else {
		health.FailureCount = 0
		health.LastFailure = ""
		health.Status = "HEALTHY"
	}
}

// pingNode performs a simple ping to check node connectivity
func (h *E2HealthMonitor) pingNode(nodeID string) error {
	// Simplified ping implementation
	// In a real implementation, this would use the actual node endpoint
	return nil // Assume healthy for now
}

// updateNodeHealth updates the health status of a node
func (h *E2HealthMonitor) updateNodeHealth(nodeID string, status string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if health, exists := h.nodeHealth[nodeID]; exists {
		health.Status = status
		health.LastCheck = time.Now()
	}
}

// getNodeHealth returns the health status of a node
func (h *E2HealthMonitor) getNodeHealth(nodeID string) (*NodeHealth, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	health, exists := h.nodeHealth[nodeID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	healthCopy := *health
	return &healthCopy, true
}

// addNodeHealth adds health monitoring for a new node
func (h *E2HealthMonitor) addNodeHealth(nodeID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.nodeHealth[nodeID] = &NodeHealth{
		NodeID:    nodeID,
		Status:    "UNKNOWN",
		LastCheck: time.Now(),
		Functions: make(map[int]*FunctionHealth),
	}
}

// removeNodeHealth removes health monitoring for a node
func (h *E2HealthMonitor) removeNodeHealth(nodeID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.nodeHealth, nodeID)
	delete(h.subscriptionHealth, nodeID)
}

// Subscription listener implementation

// AddSubscriptionListener adds a subscription event listener
func (n *SubscriptionNotifier) AddSubscriptionListener(listener SubscriptionListener) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.listeners = append(n.listeners, listener)
}

// RemoveSubscriptionListener removes a subscription event listener
func (n *SubscriptionNotifier) RemoveSubscriptionListener(listener SubscriptionListener) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for i, l := range n.listeners {
		if l == listener {
			n.listeners = append(n.listeners[:i], n.listeners[i+1:]...)
			break
		}
	}
}

// notifyStateChange notifies all listeners of a state change
func (n *SubscriptionNotifier) notifyStateChange(nodeID, subscriptionID string, oldState, newState SubscriptionState) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for _, listener := range n.listeners {
		go listener.OnSubscriptionStateChange(nodeID, subscriptionID, oldState, newState)
	}
}

// notifyError notifies all listeners of an error
func (n *SubscriptionNotifier) notifyError(nodeID, subscriptionID string, err error) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for _, listener := range n.listeners {
		go listener.OnSubscriptionError(nodeID, subscriptionID, err)
	}
}

// notifyMessage notifies all listeners of a message
func (n *SubscriptionNotifier) notifyMessage(nodeID, subscriptionID string, indication *E2Indication) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for _, listener := range n.listeners {
		go listener.OnSubscriptionMessage(nodeID, subscriptionID, indication)
	}
}
