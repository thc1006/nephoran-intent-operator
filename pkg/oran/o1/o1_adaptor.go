package o1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// O1AdaptorStatus represents the status of O1 adaptor connections
type O1AdaptorStatus struct {
	ConnectedClients int                 `json:"connected_clients"`
	TotalClients     int                 `json:"total_clients"`
	LastError        string              `json:"last_error,omitempty"`
	LastUpdate       time.Time           `json:"last_update"`
	ClientStatuses   []NetconfClientInfo `json:"client_statuses"`
}

// NetconfClientInfo provides information about NETCONF client connections
type NetconfClientInfo struct {
	ClientID      string    `json:"client_id"`
	Endpoint      string    `json:"endpoint"`
	Connected     bool      `json:"connected"`
	LastActivity  time.Time `json:"last_activity"`
	Capabilities  []string  `json:"capabilities,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// TLSConfigOptions represents TLS configuration options for O1 connections
type TLSConfigOptions struct {
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	CAFile             string `yaml:"ca_file" json:"ca_file"`
	CertFile           string `yaml:"cert_file" json:"cert_file"`
	KeyFile            string `yaml:"key_file" json:"key_file"`
}

// AdaptorO1Config holds configuration for the O1 adaptor
type AdaptorO1Config struct {
	DefaultPort        int                `yaml:"default_port" json:"default_port"`
	ConnectTimeout     time.Duration      `yaml:"connect_timeout" json:"connect_timeout"`
	RequestTimeout     time.Duration      `yaml:"request_timeout" json:"request_timeout"`
	MaxRetries         int                `yaml:"max_retries" json:"max_retries"`
	RetryInterval      time.Duration      `yaml:"retry_interval" json:"retry_interval"`
	TLSEnabled         bool               `yaml:"tls_enabled" json:"tls_enabled"`
	TLSConfig          *TLSConfigOptions  `yaml:"tls_config" json:"tls_config"`
	YANGModelDir       string             `yaml:"yang_model_dir" json:"yang_model_dir"`
	EnableNotifications bool              `yaml:"enable_notifications" json:"enable_notifications"`
	BufferSize         int                `yaml:"buffer_size" json:"buffer_size"`
	MetricsEnabled     bool               `yaml:"metrics_enabled" json:"metrics_enabled"`
}

// AdaptorClientInfo holds client information for adaptor management
type AdaptorClientInfo struct {
	ID           string
	Endpoint     string
	LastActivity time.Time
	AuthConfig   *AuthConfig
}

// O1Adaptor implements the O1 interface for network element management
type O1Adaptor struct {
	clients          map[string]*NetconfClient
	clientInfo       map[string]*AdaptorClientInfo  // Additional client metadata
	clientsMux       sync.RWMutex
	config           *AdaptorO1Config
	yangRegistry     *YANGModelRegistry
	subscriptions    map[string][]EventCallback
	subsMux          sync.RWMutex
	metricCollectors map[string]*MetricCollector
	metricsMux       sync.RWMutex
	kubeClient       client.Client
}

// MetricCollector represents a metric collector for O1 performance data
type MetricCollector struct {
	ClientID      string
	MetricTypes   []string
	LastCollected time.Time
	Enabled       bool
}

// O1Response represents a generic O1 response structure
type O1Response struct {
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NetconfOperation represents a NETCONF operation being performed
type NetconfOperation struct {
	ID          string
	SessionID   string
	Operation   string // get, get-config, edit-config, etc.
	Target      string // running, candidate, startup
	XPath       string
	StartTime   time.Time
	Status      string // pending, success, error
	Response    interface{}
	Error       string
}

// NewO1Adaptor creates a new O1 adaptor with default configuration
func NewO1Adaptor(config *AdaptorO1Config, kubeClient client.Client) *O1Adaptor {
	if config == nil {
		config = &AdaptorO1Config{
			DefaultPort:    830, // NETCONF port
			ConnectTimeout: 30 * time.Second,
			RequestTimeout: 60 * time.Second,
			MaxRetries:     3,
			RetryInterval:  5 * time.Second,
		}
	}

	return &O1Adaptor{
		clients:          make(map[string]*NetconfClient),
		clientInfo:       make(map[string]*AdaptorClientInfo),
		config:           config,
		yangRegistry:     NewYANGModelRegistry(),
		subscriptions:    make(map[string][]EventCallback),
		metricCollectors: make(map[string]*MetricCollector),
		kubeClient:       kubeClient,
	}
}

// GetClientStatus returns the status of all connected clients
func (a *O1Adaptor) GetClientStatus() *O1AdaptorStatus {
	a.clientsMux.RLock()
	defer a.clientsMux.RUnlock()
	
	var connectedClients int
	clientStatuses := make([]NetconfClientInfo, 0, len(a.clients))
	
	for clientID, client := range a.clients {
		info := a.clientInfo[clientID]
		connected := client.IsConnected()
		
		if connected {
			connectedClients++
		}
		
		status := NetconfClientInfo{
			ClientID:     clientID,
			Connected:    connected,
			Capabilities: client.GetCapabilities(),
			SessionID:    client.GetSessionID(),
		}
		
		if info != nil {
			status.Endpoint = info.Endpoint
			status.LastActivity = info.LastActivity
		}
		
		clientStatuses = append(clientStatuses, status)
	}
	
	return &O1AdaptorStatus{
		ConnectedClients: connectedClients,
		TotalClients:     len(a.clients),
		LastUpdate:       time.Now(),
		ClientStatuses:   clientStatuses,
	}
}

// AddClient adds a new NETCONF client to the adaptor
func (a *O1Adaptor) AddClient(clientID string, authConfig *AuthConfig) error {
	a.clientsMux.Lock()
	defer a.clientsMux.Unlock()
	
	if _, exists := a.clients[clientID]; exists {
		return fmt.Errorf("client %s already exists", clientID)
	}
	
	netconfConfig := &NetconfConfig{
		Port:    a.config.DefaultPort,
		Timeout: a.config.ConnectTimeout,
	}
	
	client := NewNetconfClient(netconfConfig)
	
	// Apply TLS configuration from O1Config if available
	if a.config.TLSConfig != nil {
		// SECURITY FIX: Never skip TLS verification - always validate certificates
		// If custom CA is needed, use proper certificate loading instead
		if a.config.TLSConfig != nil && a.config.TLSConfig.InsecureSkipVerify {
			log.Log.Error(nil, "SECURITY VIOLATION: TLS verification cannot be disabled for O1 interface",
				"config", fmt.Sprintf("%+v", a.config))
			return fmt.Errorf("security violation: TLS verification is mandatory for O1 interface")
		}

		// Load CA certificate if specified
		if a.config.TLSConfig.CAFile != "" {
			// TODO: Implement proper CA certificate loading from file or Kubernetes secret
			// This should load the CA certificate and add it to the certificate pool
			log.Log.Info("CA file specified but not implemented yet", "caFile", a.config.TLSConfig.CAFile)
		}
	}
	
	a.clients[clientID] = client
	a.clientInfo[clientID] = &AdaptorClientInfo{
		ID:         clientID,
		AuthConfig: authConfig,
	}
	
	return nil
}

// RemoveClient removes a NETCONF client from the adaptor
func (a *O1Adaptor) RemoveClient(clientID string) error {
	a.clientsMux.Lock()
	defer a.clientsMux.Unlock()
	
	client, exists := a.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	if client.IsConnected() {
		client.Close()
	}
	
	delete(a.clients, clientID)
	delete(a.clientInfo, clientID)
	return nil
}

// ConnectClient establishes connection to a specific NETCONF client
func (a *O1Adaptor) ConnectClient(clientID, host string, port int) error {
	a.clientsMux.Lock()
	defer a.clientsMux.Unlock()
	
	client, exists := a.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	info := a.clientInfo[clientID]
	if info == nil {
		return fmt.Errorf("client info for %s not found", clientID)
	}
	
	if client.IsConnected() {
		return fmt.Errorf("client %s is already connected", clientID)
	}
	
	logger := log.Log.WithValues("clientID", clientID, "host", host, "port", port)
	var lastErr error
	
	endpoint := fmt.Sprintf("%s:%d", host, port)
	info.Endpoint = endpoint
	
	// Retry logic for establishing connection
	for attempt := 1; attempt <= a.config.MaxRetries; attempt++ {
		logger.Info("attempting to connect", "attempt", attempt, "maxRetries", a.config.MaxRetries)
		
		if err := client.Connect(endpoint, info.AuthConfig); err != nil {
			lastErr = err
			logger.Info("connection attempt failed", "attempt", attempt, "error", err)
			if attempt < a.config.MaxRetries {
				time.Sleep(a.config.RetryInterval)
				continue
			}
		} else {
			lastErr = nil
			break
		}
	}
	
	if lastErr != nil {
		return fmt.Errorf("failed to connect to client %s after %d attempts: %v", clientID, a.config.MaxRetries, lastErr)
	}
	
	info.LastActivity = time.Now()
	logger.Info("successfully connected to client")
	return nil
}

// DisconnectClient disconnects a specific NETCONF client
func (a *O1Adaptor) DisconnectClient(clientID string) error {
	a.clientsMux.Lock()
	defer a.clientsMux.Unlock()
	
	client, exists := a.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	return client.Close()
}

// GetConfiguration retrieves configuration from a specific client
func (a *O1Adaptor) GetConfiguration(clientID, xpath string) (*O1Response, error) {
	client, exists := a.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}
	
	if !client.IsConnected() {
		return nil, fmt.Errorf("client %s is not connected", clientID)
	}
	
	// Update last activity
	if info := a.clientInfo[clientID]; info != nil {
		info.LastActivity = time.Now()
	}
	
	// Placeholder implementation for NETCONF get-config operation
	response := &O1Response{
		Status:    "success",
		Message:   "Configuration retrieved successfully",
		Data:      map[string]interface{}{"xpath": xpath, "config": "mock_config_data"},
		Timestamp: time.Now(),
	}
	
	return response, nil
}

// SetConfiguration sets configuration on a specific client
func (a *O1Adaptor) SetConfiguration(clientID, xpath string, config interface{}) (*O1Response, error) {
	client, exists := a.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}
	
	if !client.IsConnected() {
		return nil, fmt.Errorf("client %s is not connected", clientID)
	}
	
	// Validate configuration against YANG models
	if err := a.yangRegistry.ValidateConfig(context.Background(), config, xpath); err != nil {
		return &O1Response{
			Status:    "error",
			Error:     fmt.Sprintf("configuration validation failed: %v", err),
			Timestamp: time.Now(),
		}, nil
	}
	
	// Update last activity
	if info := a.clientInfo[clientID]; info != nil {
		info.LastActivity = time.Now()
	}
	
	// Placeholder implementation for NETCONF edit-config operation
	response := &O1Response{
		Status:    "success",
		Message:   "Configuration set successfully",
		Data:      map[string]interface{}{"xpath": xpath, "applied": true},
		Timestamp: time.Now(),
	}
	
	return response, nil
}

// SubscribeToNotifications subscribes to notifications from a specific client
func (a *O1Adaptor) SubscribeToNotifications(clientID string, callback EventCallback) error {
	a.subsMux.Lock()
	defer a.subsMux.Unlock()
	
	if _, exists := a.clients[clientID]; !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	if a.subscriptions[clientID] == nil {
		a.subscriptions[clientID] = make([]EventCallback, 0)
	}
	
	a.subscriptions[clientID] = append(a.subscriptions[clientID], callback)
	return nil
}

// UnsubscribeFromNotifications removes notification subscriptions for a client
func (a *O1Adaptor) UnsubscribeFromNotifications(clientID string) error {
	a.subsMux.Lock()
	defer a.subsMux.Unlock()
	
	delete(a.subscriptions, clientID)
	return nil
}

// EnableMetricCollection enables metric collection for a specific client
func (a *O1Adaptor) EnableMetricCollection(clientID string, metricTypes []string) error {
	a.metricsMux.Lock()
	defer a.metricsMux.Unlock()
	
	if _, exists := a.clients[clientID]; !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	collector := &MetricCollector{
		ClientID:    clientID,
		MetricTypes: metricTypes,
		Enabled:     true,
	}
	
	a.metricCollectors[clientID] = collector
	return nil
}

// DisableMetricCollection disables metric collection for a specific client
func (a *O1Adaptor) DisableMetricCollection(clientID string) error {
	a.metricsMux.Lock()
	defer a.metricsMux.Unlock()
	
	if collector, exists := a.metricCollectors[clientID]; exists {
		collector.Enabled = false
	}
	
	return nil
}

// CollectMetrics collects performance metrics from all enabled collectors
func (a *O1Adaptor) CollectMetrics(ctx context.Context) (map[string]interface{}, error) {
	a.metricsMux.RLock()
	defer a.metricsMux.RUnlock()
	
	metrics := make(map[string]interface{})
	
	for clientID, collector := range a.metricCollectors {
		if !collector.Enabled {
			continue
		}
		
		client, exists := a.clients[clientID]
		if !exists || !client.IsConnected() {
			continue
		}
		
		// Placeholder implementation for metric collection
		clientMetrics := map[string]interface{}{
			"client_id":      clientID,
			"last_collected": collector.LastCollected,
			"metrics":        []string{}, // Would contain actual metric data
		}
		
		metrics[clientID] = clientMetrics
		collector.LastCollected = time.Now()
	}
	
	return metrics, nil
}

// Shutdown gracefully shuts down the O1 adaptor
func (a *O1Adaptor) Shutdown(ctx context.Context) error {
	logger := log.Log.WithValues("component", "O1Adaptor")
	logger.Info("shutting down O1 adaptor")
	
	a.clientsMux.Lock()
	defer a.clientsMux.Unlock()
	
	for clientID, client := range a.clients {
		if client.IsConnected() {
			logger.Info("disconnecting client", "clientID", clientID)
			if err := client.Close(); err != nil {
				logger.Error(err, "failed to disconnect client", "clientID", clientID)
			}
		}
	}
	
	return nil
}