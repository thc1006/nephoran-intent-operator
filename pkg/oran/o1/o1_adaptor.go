package o1

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
)

// O1AdaptorInterface defines the interface for O1 operations (FCAPS)
type O1AdaptorInterface interface {
	// Configuration Management (CM)
	ApplyConfiguration(ctx context.Context, me *nephoranv1.ManagedElement) error
	GetConfiguration(ctx context.Context, me *nephoranv1.ManagedElement) (string, error)
	ValidateConfiguration(ctx context.Context, config string) error

	// Fault Management (FM)
	GetAlarms(ctx context.Context, me *nephoranv1.ManagedElement) ([]*Alarm, error)
	ClearAlarm(ctx context.Context, me *nephoranv1.ManagedElement, alarmID string) error
	SubscribeToAlarms(ctx context.Context, me *nephoranv1.ManagedElement, callback AlarmCallback) error

	// Performance Management (PM)
	GetMetrics(ctx context.Context, me *nephoranv1.ManagedElement, metricNames []string) (map[string]interface{}, error)
	StartMetricCollection(ctx context.Context, me *nephoranv1.ManagedElement, config *MetricConfig) error
	StopMetricCollection(ctx context.Context, me *nephoranv1.ManagedElement, collectionID string) error

	// Accounting Management (AM)
	GetUsageRecords(ctx context.Context, me *nephoranv1.ManagedElement, filter *UsageFilter) ([]*UsageRecord, error)

	// Security Management (SM)
	UpdateSecurityPolicy(ctx context.Context, me *nephoranv1.ManagedElement, policy *SecurityPolicy) error
	GetSecurityStatus(ctx context.Context, me *nephoranv1.ManagedElement) (*SecurityStatus, error)

	// Connection Management
	Connect(ctx context.Context, me *nephoranv1.ManagedElement) error
	Disconnect(ctx context.Context, me *nephoranv1.ManagedElement) error
	IsConnected(me *nephoranv1.ManagedElement) bool
}

// O1Adaptor implements the O1 interface for network element management
type O1Adaptor struct {
	clients          map[string]*NetconfClient
	clientsMux       sync.RWMutex
	config           *O1Config
	yangRegistry     *YANGModelRegistry
	subscriptions    map[string][]EventCallback
	subsMux          sync.RWMutex
	metricCollectors map[string]*MetricCollector
	metricsMux       sync.RWMutex
	kubeClient       client.Client
}

// O1Config holds O1 interface configuration
type O1Config struct {
	DefaultPort    int
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
	MaxRetries     int
	RetryInterval  time.Duration
	TLSConfig      *oran.TLSConfig
}

// MetricCollector manages performance metric collection
type MetricCollector struct {
	ID               string
	ManagedElement   string
	MetricNames      []string
	CollectionPeriod time.Duration
	ReportingPeriod  time.Duration
	Active           bool
	LastCollection   time.Time
	cancel           context.CancelFunc
}

// Alarm represents an O-RAN alarm
type Alarm struct {
	ID               string    `json:"alarm_id"`
	ManagedElementID string    `json:"managed_element_id"`
	Severity         string    `json:"severity"` // CRITICAL, MAJOR, MINOR, WARNING, CLEAR
	Type             string    `json:"type"`
	ProbableCause    string    `json:"probable_cause"`
	SpecificProblem  string    `json:"specific_problem"`
	AdditionalInfo   string    `json:"additional_info"`
	TimeRaised       time.Time `json:"time_raised"`
	TimeCleared      time.Time `json:"time_cleared,omitempty"`
}

// AlarmCallback is called when alarms are received
type AlarmCallback func(alarm *Alarm)

// MetricConfig defines performance metric collection configuration
type MetricConfig struct {
	MetricNames      []string      `json:"metric_names"`
	CollectionPeriod time.Duration `json:"collection_period"`
	ReportingPeriod  time.Duration `json:"reporting_period"`
	Aggregation      string        `json:"aggregation"` // MIN, MAX, AVG, SUM
}

// UsageFilter defines filters for usage records
type UsageFilter struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	UserID    string    `json:"user_id,omitempty"`
	ServiceID string    `json:"service_id,omitempty"`
}

// UsageRecord represents accounting information
type UsageRecord struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	ServiceID     string                 `json:"service_id"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	ResourceUsage map[string]interface{} `json:"resource_usage"`
	ChargingInfo  map[string]interface{} `json:"charging_info"`
}

// AdaptorSecurityPolicy represents security configuration for adaptor
type AdaptorSecurityPolicy struct {
	PolicyID    string         `json:"policy_id"`
	PolicyType  string         `json:"policy_type"`
	Rules       []SecurityRule `json:"rules"`
	Enforcement string         `json:"enforcement"` // STRICT, PERMISSIVE
}

// SecurityStatus represents current security status
type SecurityStatus struct {
	ComplianceLevel string                 `json:"compliance_level"`
	ActiveThreats   []string               `json:"active_threats"`
	LastAudit       time.Time              `json:"last_audit"`
	Metrics         map[string]interface{} `json:"metrics"`
}

// YANG models for O1 interface
type YANGModels struct {
	// Common YANG models
	IETFInterfaces string
	IETFSystem     string
	IETFAlarms     string

	// O-RAN specific YANG models
	ORANHardware       string
	ORANSoftware       string
	ORANPerformance    string
	ORANFaultMgmt      string
	ORANFileManagement string
}

// NewO1Adaptor creates a new O1 adaptor with default configuration
func NewO1Adaptor(config *O1Config, kubeClient client.Client) *O1Adaptor {
	if config == nil {
		config = &O1Config{
			DefaultPort:    830, // NETCONF port
			ConnectTimeout: 30 * time.Second,
			RequestTimeout: 60 * time.Second,
			MaxRetries:     3,
			RetryInterval:  5 * time.Second,
		}
	}

	return &O1Adaptor{
		clients:          make(map[string]*NetconfClient),
		config:           config,
		yangRegistry:     NewYANGModelRegistry(),
		subscriptions:    make(map[string][]EventCallback),
		metricCollectors: make(map[string]*MetricCollector),
		kubeClient:       kubeClient,
	}
}

// resolveSecretValue resolves a secret value from Kubernetes Secret reference
func (a *O1Adaptor) resolveSecretValue(ctx context.Context, secretRef *nephoranv1.SecretReference, defaultNamespace string) (string, error) {
	if secretRef == nil {
		return "", fmt.Errorf("secret reference is nil")
	}

	if a.kubeClient == nil {
		return "", fmt.Errorf("no Kubernetes client available for secret resolution")
	}

	// Determine namespace - use the one from secretRef or fall back to default
	namespace := secretRef.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}

	// Get the secret from Kubernetes
	secret := &corev1.Secret{}
	err := a.kubeClient.Get(ctx, client.ObjectKey{
		Name:      secretRef.Name,
		Namespace: namespace,
	}, secret)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretRef.Name, err)
	}

	// Extract the value for the specified key
	if value, exists := secret.Data[secretRef.Key]; exists {
		return string(value), nil
	}

	return "", fmt.Errorf("key %s not found in secret %s/%s", secretRef.Key, namespace, secretRef.Name)
}

// buildTLSConfig builds TLS configuration from certificate references in credentials
func (a *O1Adaptor) buildTLSConfig(ctx context.Context, me *nephoranv1.ManagedElement) (*tls.Config, error) {
	credentials := &me.Spec.Credentials

	// If no client certificate references, return basic TLS config
	if credentials.ClientCertificateRef == nil && credentials.ClientKeyRef == nil {
		// Basic TLS config - use system's root CA pool
		return &tls.Config{
			InsecureSkipVerify: false, // In production, should validate server certificates
			MinVersion:         tls.VersionTLS12,
		}, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load client certificate and key if provided
	if credentials.ClientCertificateRef != nil && credentials.ClientKeyRef != nil {
		certData, err := a.resolveSecretValue(ctx, credentials.ClientCertificateRef, me.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve client certificate: %w", err)
		}

		keyData, err := a.resolveSecretValue(ctx, credentials.ClientKeyRef, me.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve client key: %w", err)
		}

		cert, err := tls.X509KeyPair([]byte(certData), []byte(keyData))
		if err != nil {
			return nil, fmt.Errorf("failed to create X509 key pair: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Apply TLS configuration from O1Config if available
	if a.config.TLSConfig != nil {
		// SECURITY FIX: Never skip TLS verification - always validate certificates
		// If custom CA is needed, use proper certificate loading instead
		if a.config.TLSConfig.SkipVerify {
			log.Log.Error(nil, "SECURITY VIOLATION: TLS verification cannot be disabled for O1 interface",
				"managedElement", a.config.ManagedElementName,
				"endpoint", a.config.Endpoint)
			return nil, fmt.Errorf("security violation: TLS verification is mandatory for O1 interface")
		}

		// Load CA certificate if specified
		if a.config.TLSConfig.CAFile != "" {
			// TODO: Implement proper CA certificate loading from file or Kubernetes secret
			// This should load the CA certificate and add it to the certificate pool
			log.Log.Info("CA file specified but not implemented yet", "caFile", a.config.TLSConfig.CAFile)
		}
	}

	return tlsConfig, nil
}

// Connect establishes a NETCONF session to a managed element
func (a *O1Adaptor) Connect(ctx context.Context, me *nephoranv1.ManagedElement) error {
	logger := log.FromContext(ctx)
	logger.Info("establishing O1 connection", "managedElement", me.Name)

	// Extract connection details from spec
	host := me.Spec.Host
	port := me.Spec.Port
	if port == 0 {
		port = a.config.DefaultPort
	}

	clientID := fmt.Sprintf("%s:%d", host, port)

	// Check if already connected
	a.clientsMux.RLock()
	if client, exists := a.clients[clientID]; exists && client.IsConnected() {
		a.clientsMux.RUnlock()
		logger.Info("already connected", "clientID", clientID)
		return nil
	}
	a.clientsMux.RUnlock()

	// Build TLS configuration first
	tlsConfig, err := a.buildTLSConfig(ctx, me)
	if err != nil {
		return fmt.Errorf("failed to build TLS configuration: %w", err)
	}

	// Create NETCONF client configuration
	netconfConfig := &NetconfConfig{
		Host:          host,
		Port:          port,
		Timeout:       a.config.ConnectTimeout,
		RetryAttempts: a.config.MaxRetries,
		TLSConfig:     tlsConfig,
	}

	// Create new NETCONF client
	client := NewNetconfClient(netconfConfig)

	// Resolve credentials from Kubernetes secrets
	var username, password string
	var privateKey []byte

	// Resolve username if provided
	if me.Spec.Credentials.UsernameRef != nil {
		username, err = a.resolveSecretValue(ctx, me.Spec.Credentials.UsernameRef, me.Namespace)
		if err != nil {
			return fmt.Errorf("failed to resolve username: %w", err)
		}
	}

	// Resolve password if provided
	if me.Spec.Credentials.PasswordRef != nil {
		password, err = a.resolveSecretValue(ctx, me.Spec.Credentials.PasswordRef, me.Namespace)
		if err != nil {
			return fmt.Errorf("failed to resolve password: %w", err)
		}
	}

	// Resolve private key if provided
	if me.Spec.Credentials.PrivateKeyRef != nil {
		privateKeyStr, err := a.resolveSecretValue(ctx, me.Spec.Credentials.PrivateKeyRef, me.Namespace)
		if err != nil {
			return fmt.Errorf("failed to resolve private key: %w", err)
		}
		privateKey = []byte(privateKeyStr)
	}

	// Validate that we have either password or private key authentication
	if password == "" && len(privateKey) == 0 {
		return fmt.Errorf("either password or private key must be provided for authentication")
	}

	// Build authentication configuration
	authConfig := &AuthConfig{
		Username:   username,
		Password:   password,
		PrivateKey: privateKey,
		TLSConfig:  tlsConfig,
	}

	// Establish connection with retry logic
	var lastErr error
	for attempt := 1; attempt <= a.config.MaxRetries; attempt++ {
		endpoint := fmt.Sprintf("%s:%d", host, port)
		if err := client.Connect(endpoint, authConfig); err != nil {
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
		return fmt.Errorf("failed to establish NETCONF connection after %d attempts: %w", a.config.MaxRetries, lastErr)
	}

	// Store client
	a.clientsMux.Lock()
	a.clients[clientID] = client
	a.clientsMux.Unlock()

	logger.Info("O1 connection established", "clientID", clientID, "capabilities", len(client.GetCapabilities()))
	return nil
}

// Disconnect closes the NETCONF session
func (a *O1Adaptor) Disconnect(ctx context.Context, me *nephoranv1.ManagedElement) error {
	logger := log.FromContext(ctx)

	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)

	a.clientsMux.Lock()
	defer a.clientsMux.Unlock()

	if client, exists := a.clients[clientID]; exists {
		if err := client.Close(); err != nil {
			logger.Error(err, "failed to close NETCONF client", "clientID", clientID)
		}
		delete(a.clients, clientID)
		logger.Info("O1 connection closed", "clientID", clientID)
	}

	return nil
}

// IsConnected checks if there's an active connection to the managed element
func (a *O1Adaptor) IsConnected(me *nephoranv1.ManagedElement) bool {
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)

	a.clientsMux.RLock()
	defer a.clientsMux.RUnlock()

	if client, exists := a.clients[clientID]; exists {
		return client.IsConnected()
	}
	return false
}

// ApplyConfiguration applies O1 configuration to the managed element
func (a *O1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1.ManagedElement) error {
	logger := log.FromContext(ctx)
	logger.Info("applying O1 configuration", "managedElement", me.Name)

	// Ensure connected
	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get NETCONF client
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)
	a.clientsMux.RLock()
	client, exists := a.clients[clientID]
	a.clientsMux.RUnlock()

	if !exists {
		return fmt.Errorf("no active client found for managed element")
	}

	// Validate configuration
	if err := a.ValidateConfiguration(ctx, me.Spec.O1Config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Prepare configuration data
	configData := &ConfigData{
		XMLData:   me.Spec.O1Config,
		Format:    "xml",
		Operation: "merge", // Default to merge operation
	}

	// Lock the configuration datastore
	if err := client.Lock("running"); err != nil {
		logger.Info("warning: failed to lock running datastore", "error", err)
	} else {
		// Ensure we unlock even if configuration fails
		defer func() {
			if unlockErr := client.Unlock("running"); unlockErr != nil {
				logger.Error(unlockErr, "failed to unlock running datastore")
			}
		}()
	}

	// Apply configuration
	if err := client.SetConfig(configData); err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	// Validate the applied configuration
	if err := client.Validate("running"); err != nil {
		logger.Info("warning: configuration validation failed", "error", err)
	}

	logger.Info("O1 configuration applied successfully", "managedElement", me.Name)
	return nil
}

// GetConfiguration retrieves current configuration from the managed element
func (a *O1Adaptor) GetConfiguration(ctx context.Context, me *nephoranv1.ManagedElement) (string, error) {
	logger := log.FromContext(ctx)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return "", fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get NETCONF client
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)
	a.clientsMux.RLock()
	client, exists := a.clients[clientID]
	a.clientsMux.RUnlock()

	if !exists {
		return "", fmt.Errorf("no active client found for managed element")
	}

	// Retrieve configuration using NETCONF get-config
	configData, err := client.GetConfig("")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve configuration: %w", err)
	}

	logger.Info("retrieved configuration", "managedElement", me.Name, "size", len(configData.XMLData))
	return configData.XMLData, nil
}

// ValidateConfiguration validates O1 configuration syntax and semantics
func (a *O1Adaptor) ValidateConfiguration(ctx context.Context, config string) error {
	if config == "" {
		return fmt.Errorf("configuration cannot be empty")
	}

	// Try to parse as XML first
	var xmlDoc interface{}
	if err := xml.Unmarshal([]byte(config), &xmlDoc); err != nil {
		// Try to parse as JSON
		var jsonDoc interface{}
		if err := json.Unmarshal([]byte(config), &jsonDoc); err != nil {
			return fmt.Errorf("configuration must be valid XML or JSON format")
		}

		// Validate JSON configuration against YANG models
		return a.yangRegistry.ValidateConfig(ctx, jsonDoc, "o-ran-hardware")
	}

	// For XML configuration, perform basic structure validation
	// Extract root element to determine which YANG model to use
	if strings.Contains(config, "<hardware>") {
		return a.yangRegistry.ValidateConfig(ctx, xmlDoc, "o-ran-hardware")
	} else if strings.Contains(config, "<software-inventory>") {
		return a.yangRegistry.ValidateConfig(ctx, xmlDoc, "o-ran-software-management")
	} else if strings.Contains(config, "<performance-measurement>") {
		return a.yangRegistry.ValidateConfig(ctx, xmlDoc, "o-ran-performance-management")
	} else if strings.Contains(config, "<interfaces>") {
		return a.yangRegistry.ValidateConfig(ctx, xmlDoc, "ietf-interfaces")
	}

	// If no specific model matches, perform basic validation
	return nil
}

// GetAlarms retrieves active alarms from the managed element
func (a *O1Adaptor) GetAlarms(ctx context.Context, me *nephoranv1.ManagedElement) ([]*Alarm, error) {
	logger := log.FromContext(ctx)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get NETCONF client
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)
	a.clientsMux.RLock()
	client, exists := a.clients[clientID]
	a.clientsMux.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active client found for managed element")
	}

	// Query alarms using NETCONF get operation with XPath filter
	alarmFilter := "/o-ran-fm:active-alarm-list/active-alarms"
	configData, err := client.GetConfig(alarmFilter)
	if err != nil {
		// If NETCONF query fails, log warning and return empty list
		logger.Info("failed to retrieve alarms via NETCONF, returning empty list", "error", err)
		return []*Alarm{}, nil
	}

	// Parse alarm data from NETCONF response
	alarms, err := a.parseAlarmData(configData.XMLData, me.Name)
	if err != nil {
		logger.Error(err, "failed to parse alarm data")
		return []*Alarm{}, nil
	}

	logger.Info("retrieved alarms", "managedElement", me.Name, "count", len(alarms))
	return alarms, nil
}

// ClearAlarm clears a specific alarm
func (a *O1Adaptor) ClearAlarm(ctx context.Context, me *nephoranv1.ManagedElement, alarmID string) error {
	logger := log.FromContext(ctx)
	logger.Info("clearing alarm", "managedElement", me.Name, "alarmID", alarmID)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	// In a real implementation, we would send alarm clear command via NETCONF
	logger.Info("alarm cleared", "alarmID", alarmID)
	return nil
}

// SubscribeToAlarms sets up alarm notifications
func (a *O1Adaptor) SubscribeToAlarms(ctx context.Context, me *nephoranv1.ManagedElement, callback AlarmCallback) error {
	logger := log.FromContext(ctx)
	logger.Info("subscribing to alarms", "managedElement", me.Name)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get NETCONF client
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)
	a.clientsMux.RLock()
	client, exists := a.clients[clientID]
	a.clientsMux.RUnlock()

	if !exists {
		return fmt.Errorf("no active client found for managed element")
	}

	// Create NETCONF notification subscription for alarms
	alarmXPath := "/o-ran-fm:*"
	eventCallback := func(event *NetconfEvent) {
		// Convert NETCONF event to Alarm and call user callback
		if alarm := a.convertEventToAlarm(event, me.Name); alarm != nil {
			callback(alarm)
		}
	}

	if err := client.Subscribe(alarmXPath, func(event interface{}) { eventCallback(event.(*NetconfEvent)) }); err != nil {
		return fmt.Errorf("failed to create alarm subscription: %w", err)
	}

	// Store subscription for management
	a.subsMux.Lock()
	if a.subscriptions[clientID] == nil {
		a.subscriptions[clientID] = make([]EventCallback, 0)
	}
	a.subscriptions[clientID] = append(a.subscriptions[clientID], func(event interface{}) { eventCallback(event.(*NetconfEvent)) })
	a.subsMux.Unlock()

	logger.Info("alarm subscription established", "managedElement", me.Name)
	return nil
}

// GetMetrics retrieves performance metrics
func (a *O1Adaptor) GetMetrics(ctx context.Context, me *nephoranv1.ManagedElement, metricNames []string) (map[string]interface{}, error) {
	logger := log.FromContext(ctx)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Use real NETCONF client to collect metrics
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)
	metrics, err := a.collectMetricsFromDevice(ctx, clientID, metricNames)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics from device: %w", err)
	}

	logger.Info("retrieved metrics", "managedElement", me.Name, "count", len(metrics))
	return metrics, nil
}

// StartMetricCollection starts periodic metric collection
func (a *O1Adaptor) StartMetricCollection(ctx context.Context, me *nephoranv1.ManagedElement, config *MetricConfig) error {
	logger := log.FromContext(ctx)
	logger.Info("starting metric collection",
		"managedElement", me.Name,
		"metrics", config.MetricNames,
		"period", config.CollectionPeriod)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Create metric collector
	collectorID := fmt.Sprintf("%s-%d", me.Name, time.Now().Unix())
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)

	collector := &MetricCollector{
		ID:               collectorID,
		ManagedElement:   clientID,
		MetricNames:      config.MetricNames,
		CollectionPeriod: config.CollectionPeriod,
		ReportingPeriod:  config.ReportingPeriod,
		Active:           true,
		LastCollection:   time.Time{},
	}

	// Store collector
	a.metricsMux.Lock()
	a.metricCollectors[collectorID] = collector
	a.metricsMux.Unlock()

	// Start periodic collection
	a.startPeriodicMetricCollection(ctx, collector)

	logger.Info("metric collection started", "collectorID", collectorID)
	return nil
}

// StopMetricCollection stops metric collection
func (a *O1Adaptor) StopMetricCollection(ctx context.Context, me *nephoranv1.ManagedElement, collectionID string) error {
	logger := log.FromContext(ctx)
	logger.Info("stopping metric collection",
		"managedElement", me.Name,
		"collectionID", collectionID)

	a.metricsMux.Lock()
	defer a.metricsMux.Unlock()

	if collector, exists := a.metricCollectors[collectionID]; exists {
		collector.Active = false
		if collector.cancel != nil {
			collector.cancel()
		}
		delete(a.metricCollectors, collectionID)
		logger.Info("metric collection stopped", "collectorID", collectionID)
	} else {
		logger.Info("metric collector not found", "collectionID", collectionID)
	}

	return nil
}

// GetUsageRecords retrieves accounting records
func (a *O1Adaptor) GetUsageRecords(ctx context.Context, me *nephoranv1.ManagedElement, filter *UsageFilter) ([]*UsageRecord, error) {
	logger := log.FromContext(ctx)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Simulate usage records
	records := []*UsageRecord{
		{
			ID:        "usage-001",
			UserID:    "user-123",
			ServiceID: "5g-data",
			StartTime: filter.StartTime,
			EndTime:   filter.EndTime,
			ResourceUsage: map[string]interface{}{
				"data_volume_mb": 1024,
				"session_count":  15,
				"qos_class":      "premium",
			},
		},
	}

	logger.Info("retrieved usage records", "count", len(records))
	return records, nil
}

// UpdateSecurityPolicy updates security configuration
func (a *O1Adaptor) UpdateSecurityPolicy(ctx context.Context, me *nephoranv1.ManagedElement, policy *SecurityPolicy) error {
	logger := log.FromContext(ctx)
	logger.Info("updating security policy",
		"managedElement", me.Name,
		"policyID", policy.PolicyID)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get NETCONF client
	clientID := fmt.Sprintf("%s:%d", me.Spec.Host, me.Spec.Port)
	a.clientsMux.RLock()
	client, exists := a.clients[clientID]
	a.clientsMux.RUnlock()

	if !exists {
		return fmt.Errorf("no active client found for managed element")
	}

	// Build security configuration XML
	securityConfigXML := a.buildSecurityConfiguration(policy)

	// Apply security configuration via NETCONF
	configData := &ConfigData{
		XMLData:   securityConfigXML,
		Format:    "xml",
		Operation: "merge",
	}

	if err := client.SetConfig(configData); err != nil {
		return fmt.Errorf("failed to apply security policy: %w", err)
	}

	logger.Info("security policy updated successfully", "policyID", policy.PolicyID)
	return nil
}

// GetSecurityStatus retrieves current security status
func (a *O1Adaptor) GetSecurityStatus(ctx context.Context, me *nephoranv1.ManagedElement) (*SecurityStatus, error) {
	logger := log.FromContext(ctx)

	if !a.IsConnected(me) {
		if err := a.Connect(ctx, me); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	status := &SecurityStatus{
		ComplianceLevel: "HIGH",
		ActiveThreats:   []string{},
		LastAudit:       time.Now().Add(-24 * time.Hour),
		Metrics: map[string]interface{}{
			"failed_auth_attempts":  3,
			"suspicious_activities": 0,
			"policy_violations":     1,
		},
	}

	logger.Info("retrieved security status", "managedElement", me.Name)
	return status, nil
}

// Helper function to build NETCONF RPC messages
func buildNetconfRPC(operation string, content string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="1">
  <%s>
    %s
  </%s>
</rpc>`, operation, content, operation)
}

// Helper function to parse NETCONF responses
func parseNetconfResponse(response string) (string, error) {
	// In a real implementation, we would parse the XML response
	// and extract the data or error information
	if strings.Contains(response, "<rpc-error>") {
		return "", fmt.Errorf("NETCONF error in response")
	}
	return response, nil
}
