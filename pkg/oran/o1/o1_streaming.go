package o1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// StreamingService provides real-time streaming capabilities for O1 interface
type StreamingService struct {
	logger             *zap.Logger
	upgrader           websocket.Upgrader
	connections        map[string]*StreamConnection
	connectionsMutex   sync.RWMutex
	subscriptions      map[string]*StreamSubscription
	subscriptionsMutex sync.RWMutex
	metrics            *StreamingMetrics
	config             *StreamingConfig
	eventBus           *EventBus
	authManager        *StreamingAuthManager
	rateLimiter        *StreamingRateLimiter
}

// StreamingConfig holds configuration for streaming service
type StreamingConfig struct {
	MaxConnections          int           `yaml:"max_connections" json:"max_connections"`
	ConnectionTimeout       time.Duration `yaml:"connection_timeout" json:"connection_timeout"`
	HeartbeatInterval       time.Duration `yaml:"heartbeat_interval" json:"heartbeat_interval"`
	MaxSubscriptionsPerConn int           `yaml:"max_subscriptions_per_conn" json:"max_subscriptions_per_conn"`
	BufferSize              int           `yaml:"buffer_size" json:"buffer_size"`
	CompressionEnabled      bool          `yaml:"compression_enabled" json:"compression_enabled"`
	EnableAuth              bool          `yaml:"enable_auth" json:"enable_auth"`
	RateLimitPerSecond      int           `yaml:"rate_limit_per_second" json:"rate_limit_per_second"`
}

// StreamConnection represents a WebSocket connection
type StreamConnection struct {
	ID               string
	Conn             *websocket.Conn
	Send             chan []byte
	Subscriptions    map[string]*StreamSubscription
	LastActivity     time.Time
	ClientInfo       *ClientInfo
	AuthContext      *AuthContext
	RateLimiter      *ConnectionRateLimiter
	CompressionLevel int
	mutex            sync.RWMutex
	closed           bool
}

// StreamSubscription represents a subscription to data streams
type StreamSubscription struct {
	ID                   string
	Type                 StreamType
	Filter               *StreamFilter
	Connection           *StreamConnection
	CreatedAt            time.Time
	LastMessage          time.Time
	MessageCount         int64
	Active               bool
	QoSLevel             QoSLevel
	BufferSize           int
	BackpressureStrategy BackpressureStrategy
}

// StreamType defines types of data streams
type StreamType string

const (
	StreamTypeAlarms        StreamType = "alarms"
	StreamTypePerformance   StreamType = "performance"
	StreamTypeConfiguration StreamType = "configuration"
	StreamTypeEvents        StreamType = "events"
	StreamTypeLogs          StreamType = "logs"
	StreamTypeTopology      StreamType = "topology"
	StreamTypeStatus        StreamType = "status"
)

// QoSLevel defines quality of service levels
type QoSLevel int

const (
	QoSBestEffort QoSLevel = iota
	QoSReliable
	QoSGuaranteed
)

// BackpressureStrategy defines how to handle backpressure
type BackpressureStrategy int

const (
	BackpressureDrop BackpressureStrategy = iota
	BackpressureBuffer
	BackpressureThrottle
)

// StreamFilter defines filtering criteria for streams
type StreamFilter struct {
	SourceFilter     []string          `json:"source_filter,omitempty"`
	SeverityFilter   []string          `json:"severity_filter,omitempty"`
	TypeFilter       []string          `json:"type_filter,omitempty"`
	TimeWindow       *TimeWindow       `json:"time_window,omitempty"`
	AttributeFilters map[string]string `json:"attribute_filters,omitempty"`
	XPathFilter      string            `json:"xpath_filter,omitempty"`
	RegexFilter      string            `json:"regex_filter,omitempty"`
}

// TimeWindow defines time-based filtering
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ClientInfo holds client information
type ClientInfo struct {
	UserAgent    string
	RemoteAddr   string
	ClientID     string
	Organization string
	Capabilities []string
}

// AuthContext holds authentication context
type AuthContext struct {
	UserID      string
	Roles       []string
	Permissions []string
	Token       string
	ExpiresAt   time.Time
}

// StreamingMetrics holds Prometheus metrics for streaming
type StreamingMetrics struct {
	ActiveConnections  prometheus.Gauge
	TotalConnections   prometheus.Counter
	MessagesStreamed   prometheus.CounterVec
	StreamingLatency   prometheus.HistogramVec
	SubscriptionCount  prometheus.GaugeVec
	ConnectionDuration prometheus.HistogramVec
	ErrorCount         prometheus.CounterVec
	BackpressureEvents prometheus.CounterVec
}

// EventBus handles event distribution
type EventBus struct {
	subscribers map[StreamType][]chan interface{}
	mutex       sync.RWMutex
	logger      *zap.Logger
}

// StreamingAuthManager handles authentication for streaming
type StreamingAuthManager struct {
	tokenValidator *TokenValidator
	permissionMgr  *PermissionManager
	logger         *zap.Logger
}

// StreamingRateLimiter handles rate limiting
type StreamingRateLimiter struct {
	globalLimiter map[string]*RateLimit
	mutex         sync.RWMutex
}

// ConnectionRateLimiter handles per-connection rate limiting
type ConnectionRateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewStreamingService creates a new streaming service
func NewStreamingService(config *StreamingConfig, logger *zap.Logger) *StreamingService {
	metrics := &StreamingMetrics{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "o1_streaming_active_connections",
			Help: "Number of active streaming connections",
		}),
		TotalConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "o1_streaming_total_connections",
			Help: "Total number of streaming connections",
		}),
		MessagesStreamed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "o1_streaming_messages_total",
			Help: "Total number of messages streamed",
		}, []string{"stream_type", "connection_id"}),
		StreamingLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "o1_streaming_latency_seconds",
			Help: "Streaming latency in seconds",
		}, []string{"stream_type"}),
		SubscriptionCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "o1_streaming_subscriptions",
			Help: "Number of active subscriptions",
		}, []string{"stream_type"}),
		ConnectionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "o1_streaming_connection_duration_seconds",
			Help: "Connection duration in seconds",
		}, []string{"client_type"}),
		ErrorCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "o1_streaming_errors_total",
			Help: "Total number of streaming errors",
		}, []string{"error_type"}),
		BackpressureEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "o1_streaming_backpressure_events_total",
			Help: "Total number of backpressure events",
		}, []string{"strategy", "stream_type"}),
	}

	// Register metrics
	prometheus.MustRegister(
		metrics.ActiveConnections,
		metrics.TotalConnections,
		metrics.MessagesStreamed,
		metrics.StreamingLatency,
		metrics.SubscriptionCount,
		metrics.ConnectionDuration,
		metrics.ErrorCount,
		metrics.BackpressureEvents,
	)

	upgrader := websocket.Upgrader{
		ReadBufferSize:  config.BufferSize,
		WriteBufferSize: config.BufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true // Configure properly in production
		},
		EnableCompression: config.CompressionEnabled,
	}

	eventBus := &EventBus{
		subscribers: make(map[StreamType][]chan interface{}),
		logger:      logger,
	}

	authManager := &StreamingAuthManager{
		tokenValidator: NewTokenValidator(),
		permissionMgr:  NewPermissionManager(),
		logger:         logger,
	}

	rateLimiter := &StreamingRateLimiter{
		globalLimiter: make(map[string]*RateLimit),
	}

	return &StreamingService{
		logger:        logger,
		upgrader:      upgrader,
		connections:   make(map[string]*StreamConnection),
		subscriptions: make(map[string]*StreamSubscription),
		metrics:       metrics,
		config:        config,
		eventBus:      eventBus,
		authManager:   authManager,
		rateLimiter:   rateLimiter,
	}
}

// HandleWebSocketConnection handles new WebSocket connections
func (s *StreamingService) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	// Check connection limit
	s.connectionsMutex.RLock()
	if len(s.connections) >= s.config.MaxConnections {
		s.connectionsMutex.RUnlock()
		http.Error(w, "Connection limit exceeded", http.StatusTooManyRequests)
		return
	}
	s.connectionsMutex.RUnlock()

	// Authenticate if enabled
	var authContext *AuthContext
	if s.config.EnableAuth {
		var err error
		authContext, err = s.authManager.authenticateRequest(r)
		if err != nil {
			s.logger.Error("Authentication failed", zap.Error(err))
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Create stream connection
	streamConn := &StreamConnection{
		ID:            generateConnectionID(),
		Conn:          conn,
		Send:          make(chan []byte, s.config.BufferSize),
		Subscriptions: make(map[string]*StreamSubscription),
		LastActivity:  time.Now(),
		ClientInfo: &ClientInfo{
			UserAgent:  r.UserAgent(),
			RemoteAddr: r.RemoteAddr,
			ClientID:   r.Header.Get("X-Client-ID"),
		},
		AuthContext: authContext,
		RateLimiter: NewConnectionRateLimiter(s.config.RateLimitPerSecond),
	}

	// Register connection
	s.connectionsMutex.Lock()
	s.connections[streamConn.ID] = streamConn
	s.connectionsMutex.Unlock()

	s.metrics.ActiveConnections.Inc()
	s.metrics.TotalConnections.Inc()

	s.logger.Info("New streaming connection established",
		zap.String("connection_id", streamConn.ID),
		zap.String("remote_addr", streamConn.ClientInfo.RemoteAddr))

	// Start connection handlers
	go s.handleConnectionRead(streamConn)
	go s.handleConnectionWrite(streamConn)
	go s.handleConnectionHeartbeat(streamConn)
}

// handleConnectionRead handles reading messages from WebSocket
func (s *StreamingService) handleConnectionRead(conn *StreamConnection) {
	defer s.closeConnection(conn)

	conn.Conn.SetReadDeadline(time.Now().Add(s.config.ConnectionTimeout))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(s.config.ConnectionTimeout))
		conn.LastActivity = time.Now()
		return nil
	})

	for {
		messageType, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		if messageType == websocket.TextMessage {
			s.handleStreamingMessage(conn, message)
		}

		conn.LastActivity = time.Now()
	}
}

// handleConnectionWrite handles writing messages to WebSocket
func (s *StreamingService) handleConnectionWrite(conn *StreamConnection) {
	defer conn.Conn.Close()

	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.logger.Error("Failed to write message", zap.Error(err))
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleConnectionHeartbeat manages connection heartbeat
func (s *StreamingService) handleConnectionHeartbeat(conn *StreamConnection) {
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(conn.LastActivity) > s.config.ConnectionTimeout {
				s.logger.Info("Connection timeout", zap.String("connection_id", conn.ID))
				s.closeConnection(conn)
				return
			}
		}
	}
}

// handleStreamingMessage processes streaming messages
func (s *StreamingService) handleStreamingMessage(conn *StreamConnection, message []byte) {
	var request StreamingRequest
	if err := json.Unmarshal(message, &request); err != nil {
		s.sendError(conn, "Invalid message format", err)
		return
	}

	switch request.Action {
	case "subscribe":
		s.handleSubscribe(conn, &request)
	case "unsubscribe":
		s.handleUnsubscribe(conn, &request)
	case "list_subscriptions":
		s.handleListSubscriptions(conn)
	case "get_status":
		s.handleGetStatus(conn)
	default:
		s.sendError(conn, "Unknown action", fmt.Errorf("action: %s", request.Action))
	}
}

// StreamingRequest represents a client request
type StreamingRequest struct {
	Action         string        `json:"action"`
	StreamType     StreamType    `json:"stream_type,omitempty"`
	Filter         *StreamFilter `json:"filter,omitempty"`
	SubscriptionID string        `json:"subscription_id,omitempty"`
	QoSLevel       QoSLevel      `json:"qos_level,omitempty"`
	BufferSize     int           `json:"buffer_size,omitempty"`
}

// handleSubscribe handles subscription requests
func (s *StreamingService) handleSubscribe(conn *StreamConnection, request *StreamingRequest) {
	// Check subscription limit
	if len(conn.Subscriptions) >= s.config.MaxSubscriptionsPerConn {
		s.sendError(conn, "Subscription limit exceeded", nil)
		return
	}

	// Check permissions
	if conn.AuthContext != nil {
		if !s.authManager.hasPermission(conn.AuthContext, string(request.StreamType), "subscribe") {
			s.sendError(conn, "Permission denied", nil)
			return
		}
	}

	// Create subscription
	subscription := &StreamSubscription{
		ID:                   generateSubscriptionID(),
		Type:                 request.StreamType,
		Filter:               request.Filter,
		Connection:           conn,
		CreatedAt:            time.Now(),
		Active:               true,
		QoSLevel:             request.QoSLevel,
		BufferSize:           request.BufferSize,
		BackpressureStrategy: BackpressureBuffer, // Default
	}

	// Register subscription
	conn.mutex.Lock()
	conn.Subscriptions[subscription.ID] = subscription
	conn.mutex.Unlock()

	s.subscriptionsMutex.Lock()
	s.subscriptions[subscription.ID] = subscription
	s.subscriptionsMutex.Unlock()

	// Subscribe to event bus
	s.eventBus.subscribe(subscription)

	s.metrics.SubscriptionCount.WithLabelValues(string(request.StreamType)).Inc()

	// Send confirmation
	response := map[string]interface{}{
		"type":            "subscription_created",
		"subscription_id": subscription.ID,
		"stream_type":     subscription.Type,
		"status":          "active",
	}
	s.sendMessage(conn, response)

	s.logger.Info("Subscription created",
		zap.String("connection_id", conn.ID),
		zap.String("subscription_id", subscription.ID),
		zap.String("stream_type", string(request.StreamType)))
}

// handleUnsubscribe handles unsubscription requests
func (s *StreamingService) handleUnsubscribe(conn *StreamConnection, request *StreamingRequest) {
	conn.mutex.Lock()
	subscription, exists := conn.Subscriptions[request.SubscriptionID]
	if exists {
		delete(conn.Subscriptions, request.SubscriptionID)
	}
	conn.mutex.Unlock()

	if !exists {
		s.sendError(conn, "Subscription not found", nil)
		return
	}

	s.subscriptionsMutex.Lock()
	delete(s.subscriptions, request.SubscriptionID)
	s.subscriptionsMutex.Unlock()

	// Unsubscribe from event bus
	s.eventBus.unsubscribe(subscription)

	s.metrics.SubscriptionCount.WithLabelValues(string(subscription.Type)).Dec()

	// Send confirmation
	response := map[string]interface{}{
		"type":            "subscription_removed",
		"subscription_id": request.SubscriptionID,
		"status":          "inactive",
	}
	s.sendMessage(conn, response)

	s.logger.Info("Subscription removed",
		zap.String("connection_id", conn.ID),
		zap.String("subscription_id", request.SubscriptionID))
}

// StreamData publishes data to subscribers
func (s *StreamingService) StreamData(streamType StreamType, data interface{}) {
	s.eventBus.publish(streamType, data)
}

// StreamAlarm publishes alarm data
func (s *StreamingService) StreamAlarm(alarm *AlarmData) {
	s.StreamData(StreamTypeAlarms, alarm)
}

// StreamPerformanceData publishes performance data
func (s *StreamingService) StreamPerformanceData(data *PerformanceData) {
	s.StreamData(StreamTypePerformance, data)
}

// StreamConfigurationChange publishes configuration changes
func (s *StreamingService) StreamConfigurationChange(change *ConfigurationChange) {
	s.StreamData(StreamTypeConfiguration, change)
}

// AlarmData represents alarm data for streaming
type AlarmData struct {
	ID          string                 `json:"id"`
	Source      string                 `json:"source"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Attributes  map[string]interface{} `json:"attributes"`
}

// PerformanceData represents performance data for streaming
type PerformanceData struct {
	Source     string                 `json:"source"`
	MetricName string                 `json:"metric_name"`
	Value      float64                `json:"value"`
	Unit       string                 `json:"unit"`
	Timestamp  time.Time              `json:"timestamp"`
	Labels     map[string]string      `json:"labels"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ConfigurationChange represents configuration changes
type ConfigurationChange struct {
	Source     string                 `json:"source"`
	ChangeType string                 `json:"change_type"`
	Path       string                 `json:"path"`
	OldValue   interface{}            `json:"old_value"`
	NewValue   interface{}            `json:"new_value"`
	Timestamp  time.Time              `json:"timestamp"`
	User       string                 `json:"user"`
	Attributes map[string]interface{} `json:"attributes"`
}

// closeConnection closes a streaming connection
func (s *StreamingService) closeConnection(conn *StreamConnection) {
	conn.mutex.Lock()
	if conn.closed {
		conn.mutex.Unlock()
		return
	}
	conn.closed = true
	conn.mutex.Unlock()

	// Remove from connections
	s.connectionsMutex.Lock()
	delete(s.connections, conn.ID)
	s.connectionsMutex.Unlock()

	// Remove all subscriptions
	for _, subscription := range conn.Subscriptions {
		s.subscriptionsMutex.Lock()
		delete(s.subscriptions, subscription.ID)
		s.subscriptionsMutex.Unlock()

		s.eventBus.unsubscribe(subscription)
		s.metrics.SubscriptionCount.WithLabelValues(string(subscription.Type)).Dec()
	}

	close(conn.Send)
	conn.Conn.Close()

	s.metrics.ActiveConnections.Dec()

	s.logger.Info("Connection closed",
		zap.String("connection_id", conn.ID))
}

// sendMessage sends a message to a connection
func (s *StreamingService) sendMessage(conn *StreamConnection, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}

	select {
	case conn.Send <- data:
	default:
		s.logger.Warn("Connection send buffer full, closing connection",
			zap.String("connection_id", conn.ID))
		s.closeConnection(conn)
	}
}

// sendError sends an error message to a connection
func (s *StreamingService) sendError(conn *StreamConnection, message string, err error) {
	errorMsg := map[string]interface{}{
		"type":    "error",
		"message": message,
	}
	if err != nil {
		errorMsg["details"] = err.Error()
	}
	s.sendMessage(conn, errorMsg)
}

// Start starts the streaming service
func (s *StreamingService) Start(ctx context.Context) error {
	s.logger.Info("Starting O1 streaming service")

	// Start cleanup routine
	go s.cleanup(ctx)

	return nil
}

// Stop stops the streaming service
func (s *StreamingService) Stop() error {
	s.logger.Info("Stopping O1 streaming service")

	// Close all connections
	s.connectionsMutex.Lock()
	for _, conn := range s.connections {
		s.closeConnection(conn)
	}
	s.connectionsMutex.Unlock()

	return nil
}

// cleanup performs periodic cleanup
func (s *StreamingService) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpiredConnections()
		}
	}
}

// cleanupExpiredConnections removes expired connections
func (s *StreamingService) cleanupExpiredConnections() {
	s.connectionsMutex.Lock()
	defer s.connectionsMutex.Unlock()

	for id, conn := range s.connections {
		if time.Since(conn.LastActivity) > s.config.ConnectionTimeout {
			delete(s.connections, id)
			s.closeConnection(conn)
		}
	}
}

// Helper functions
func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// Event bus implementation
func (eb *EventBus) subscribe(subscription *StreamSubscription) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if _, exists := eb.subscribers[subscription.Type]; !exists {
		eb.subscribers[subscription.Type] = make([]chan interface{}, 0)
	}

	// Create channel for subscription
	ch := make(chan interface{}, subscription.BufferSize)
	eb.subscribers[subscription.Type] = append(eb.subscribers[subscription.Type], ch)

	// Start subscription handler
	go eb.handleSubscription(subscription, ch)
}

func (eb *EventBus) unsubscribe(subscription *StreamSubscription) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if channels, exists := eb.subscribers[subscription.Type]; exists {
		// Remove channel (simplified implementation)
		for i, ch := range channels {
			close(ch)
			eb.subscribers[subscription.Type] = append(channels[:i], channels[i+1:]...)
			break
		}
	}
}

func (eb *EventBus) publish(streamType StreamType, data interface{}) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if channels, exists := eb.subscribers[streamType]; exists {
		for _, ch := range channels {
			select {
			case ch <- data:
			default:
				// Channel full, skip
			}
		}
	}
}

func (eb *EventBus) handleSubscription(subscription *StreamSubscription, ch chan interface{}) {
	for data := range ch {
		if !subscription.Active {
			break
		}

		// Apply filters
		if subscription.Filter != nil && !eb.matchesFilter(data, subscription.Filter) {
			continue
		}

		// Send to connection
		message := map[string]interface{}{
			"type":            "data",
			"stream_type":     subscription.Type,
			"subscription_id": subscription.ID,
			"data":            data,
			"timestamp":       time.Now(),
		}

		// Handle backpressure
		subscription.Connection.RateLimiter.WaitIfNeeded()

		subscription.Connection.Send <- func() []byte {
			data, _ := json.Marshal(message)
			return data
		}()

		subscription.MessageCount++
		subscription.LastMessage = time.Now()
	}
}

func (eb *EventBus) matchesFilter(data interface{}, filter *StreamFilter) bool {
	// Simplified filter implementation
	// In production, implement comprehensive filtering based on all filter criteria
	return true
}

// Rate limiter implementations
type StreamingRateLimit struct {
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	mutex      sync.Mutex
}

func NewConnectionRateLimiter(ratePerSecond int) *ConnectionRateLimiter {
	return &ConnectionRateLimiter{
		tokens:     ratePerSecond,
		maxTokens:  ratePerSecond,
		refillRate: ratePerSecond,
		lastRefill: time.Now(),
	}
}

func (crl *ConnectionRateLimiter) WaitIfNeeded() {
	crl.mutex.Lock()
	defer crl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(crl.lastRefill)

	// Refill tokens
	if elapsed > time.Second {
		tokensToAdd := int(elapsed.Seconds()) * crl.refillRate
		crl.tokens = minInt(crl.maxTokens, crl.tokens+tokensToAdd)
		crl.lastRefill = now
	}

	// Wait if no tokens available
	if crl.tokens <= 0 {
		time.Sleep(time.Second / time.Duration(crl.refillRate))
		crl.tokens = 1
	} else {
		crl.tokens--
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Auth manager implementations (simplified)
type TokenValidator struct{}
type PermissionManager struct{}

func NewTokenValidator() *TokenValidator {
	return &TokenValidator{}
}

func NewPermissionManager() *PermissionManager {
	return &PermissionManager{}
}

func (sam *StreamingAuthManager) authenticateRequest(r *http.Request) (*AuthContext, error) {
	// Implement OAuth2/JWT authentication
	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	// Validate token and return auth context
	return &AuthContext{
		UserID:      "user123",
		Roles:       []string{"operator"},
		Permissions: []string{"stream:alarms", "stream:performance"},
		Token:       token,
		ExpiresAt:   time.Now().Add(time.Hour),
	}, nil
}

func (sam *StreamingAuthManager) hasPermission(authCtx *AuthContext, resource, action string) bool {
	permission := fmt.Sprintf("%s:%s", action, resource)
	for _, p := range authCtx.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Additional helper methods
func (s *StreamingService) handleListSubscriptions(conn *StreamConnection) {
	conn.mutex.RLock()
	subscriptions := make([]map[string]interface{}, 0, len(conn.Subscriptions))
	for _, sub := range conn.Subscriptions {
		subscriptions = append(subscriptions, map[string]interface{}{
			"id":            sub.ID,
			"stream_type":   sub.Type,
			"created_at":    sub.CreatedAt,
			"active":        sub.Active,
			"message_count": sub.MessageCount,
		})
	}
	conn.mutex.RUnlock()

	response := map[string]interface{}{
		"type":          "subscription_list",
		"subscriptions": subscriptions,
	}
	s.sendMessage(conn, response)
}

func (s *StreamingService) handleGetStatus(conn *StreamConnection) {
	s.connectionsMutex.RLock()
	totalConnections := len(s.connections)
	s.connectionsMutex.RUnlock()

	s.subscriptionsMutex.RLock()
	totalSubscriptions := len(s.subscriptions)
	s.subscriptionsMutex.RUnlock()

	response := map[string]interface{}{
		"type":                "status",
		"connection_id":       conn.ID,
		"total_connections":   totalConnections,
		"total_subscriptions": totalSubscriptions,
		"server_time":         time.Now(),
		"uptime":              time.Since(conn.LastActivity),
	}
	s.sendMessage(conn, response)
}
