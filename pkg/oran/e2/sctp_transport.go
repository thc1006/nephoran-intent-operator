package e2

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// SCTPTransport provides SCTP transport for E2AP messages
// Following O-RAN.WG3.E2AP specifications for SCTP usage
type SCTPTransport struct {
	config          *SCTPConfig
	listener        net.Listener
	connections     map[string]*SCTPConnection
	messageHandler  E2APMessageHandler
	codec           *ASN1Codec
	running         atomic.Bool
	metrics         *SCTPMetrics
	connectionPool  *SCTPConnectionPool
	mutex           sync.RWMutex
	stopChan        chan struct{}
}

// SCTPConfig contains SCTP transport configuration
type SCTPConfig struct {
	// Network settings
	ListenAddress   string
	ListenPort      int
	MaxConnections  int
	MaxStreams      int
	
	// SCTP parameters
	InitTimeout     time.Duration
	HeartbeatInterval time.Duration
	MaxRetransmits  int
	RTO             time.Duration // Retransmission timeout
	
	// Connection settings
	KeepAlive       bool
	KeepAliveInterval time.Duration
	ConnectionTimeout time.Duration
	
	// Buffer settings
	SendBufferSize  int
	RecvBufferSize  int
	MaxMessageSize  int
	
	// Security settings
	EnableTLS       bool
	TLSConfig       *TLSConfig
}

// SCTPConnection represents an SCTP association
type SCTPConnection struct {
	conn            net.Conn
	nodeID          string
	streams         map[uint16]*SCTPStream
	state           SCTPConnectionState
	metrics         *SCTPConnectionMetrics
	lastActivity    time.Time
	pendingMessages map[uint32]*PendingMessage
	sequenceNumber  uint32
	mutex           sync.RWMutex
}

// SCTPStream represents an SCTP stream within an association
type SCTPStream struct {
	streamID        uint16
	streamType      SCTPStreamType
	sequenceNumber  uint32
	inboundQueue    chan *E2APMessage
	outboundQueue   chan *E2APMessage
	state           SCTPStreamState
	metrics         *SCTPStreamMetrics
}

// SCTPConnectionState represents the state of an SCTP connection
type SCTPConnectionState int

const (
	SCTPStateClosed SCTPConnectionState = iota
	SCTPStateConnecting
	SCTPStateEstablished
	SCTPStateShutdownPending
	SCTPStateShutdownSent
	SCTPStateShutdownReceived
	SCTPStateShutdownAckSent
)

// SCTPStreamType defines the type of SCTP stream
type SCTPStreamType int

const (
	SCTPStreamTypeControl SCTPStreamType = iota
	SCTPStreamTypeData
	SCTPStreamTypeManagement
)

// SCTPStreamState represents the state of an SCTP stream
type SCTPStreamState int

const (
	SCTPStreamStateClosed SCTPStreamState = iota
	SCTPStreamStateOpen
	SCTPStreamStateClosing
)

// SCTPMetrics contains transport-level metrics
type SCTPMetrics struct {
	ConnectionsTotal       atomic.Int64
	ConnectionsActive      atomic.Int64
	MessagesSent           atomic.Int64
	MessagesReceived       atomic.Int64
	BytesSent              atomic.Int64
	BytesReceived          atomic.Int64
	Errors                 atomic.Int64
	Retransmissions        atomic.Int64
	HeartbeatsSent         atomic.Int64
	HeartbeatsReceived     atomic.Int64
}

// SCTPConnectionMetrics contains per-connection metrics
type SCTPConnectionMetrics struct {
	MessagesSent       int64
	MessagesReceived   int64
	BytesSent          int64
	BytesReceived      int64
	Errors             int64
	Retransmissions    int64
	RTT                time.Duration
	LastActivity       time.Time
}

// SCTPStreamMetrics contains per-stream metrics
type SCTPStreamMetrics struct {
	MessagesSent     int64
	MessagesReceived int64
	QueueDepth       int
	DroppedMessages  int64
}

// E2APMessageHandler handles received E2AP messages
type E2APMessageHandler func(nodeID string, message *E2APMessage) error

// NewSCTPTransport creates a new SCTP transport
func NewSCTPTransport(config *SCTPConfig, codec *ASN1Codec) *SCTPTransport {
	return &SCTPTransport{
		config:         config,
		connections:    make(map[string]*SCTPConnection),
		codec:          codec,
		metrics:        &SCTPMetrics{},
		connectionPool: NewSCTPConnectionPool(config.MaxConnections),
		stopChan:       make(chan struct{}),
	}
}

// Start starts the SCTP transport listener
func (t *SCTPTransport) Start(ctx context.Context) error {
	if !t.running.CompareAndSwap(false, true) {
		return fmt.Errorf("transport already running")
	}
	
	// Create listener
	// Note: In production, use a real SCTP library like github.com/pion/sctp
	// This implementation uses TCP as a fallback for demonstration
	addr := fmt.Sprintf("%s:%d", t.config.ListenAddress, t.config.ListenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.running.Store(false)
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	
	t.listener = listener
	
	// Start accept loop
	go t.acceptLoop(ctx)
	
	// Start maintenance tasks
	go t.maintenanceLoop(ctx)
	
	return nil
}

// Stop stops the SCTP transport
func (t *SCTPTransport) Stop() error {
	if !t.running.CompareAndSwap(true, false) {
		return fmt.Errorf("transport not running")
	}
	
	close(t.stopChan)
	
	// Close listener
	if t.listener != nil {
		t.listener.Close()
	}
	
	// Close all connections
	t.mutex.Lock()
	for nodeID, conn := range t.connections {
		t.closeConnection(nodeID, conn)
	}
	t.connections = make(map[string]*SCTPConnection)
	t.mutex.Unlock()
	
	return nil
}

// Connect establishes an SCTP connection to a remote E2 node
func (t *SCTPTransport) Connect(nodeID string, address string) error {
	if !t.running.Load() {
		return fmt.Errorf("transport not running")
	}
	
	// Check if already connected
	t.mutex.RLock()
	if _, exists := t.connections[nodeID]; exists {
		t.mutex.RUnlock()
		return fmt.Errorf("already connected to node %s", nodeID)
	}
	t.mutex.RUnlock()
	
	// Establish connection
	// Note: In production, use SCTP-specific connection establishment
	conn, err := net.DialTimeout("tcp", address, t.config.ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	
	// Create SCTP connection
	sctpConn := &SCTPConnection{
		conn:            conn,
		nodeID:          nodeID,
		streams:         make(map[uint16]*SCTPStream),
		state:           SCTPStateEstablished,
		metrics:         &SCTPConnectionMetrics{},
		lastActivity:    time.Now(),
		pendingMessages: make(map[uint32]*PendingMessage),
		sequenceNumber:  0,
	}
	
	// Initialize streams
	t.initializeStreams(sctpConn)
	
	// Store connection
	t.mutex.Lock()
	t.connections[nodeID] = sctpConn
	t.mutex.Unlock()
	
	// Start connection handlers
	go t.handleConnection(sctpConn)
	
	// Update metrics
	t.metrics.ConnectionsTotal.Add(1)
	t.metrics.ConnectionsActive.Add(1)
	
	return nil
}

// SendMessage sends an E2AP message over SCTP
func (t *SCTPTransport) SendMessage(nodeID string, message *E2APMessage) error {
	if !t.running.Load() {
		return fmt.Errorf("transport not running")
	}
	
	// Get connection
	t.mutex.RLock()
	conn, exists := t.connections[nodeID]
	t.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("no connection to node %s", nodeID)
	}
	
	// Encode message
	data, err := t.codec.EncodeE2APMessage(message)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}
	
	// Select appropriate stream based on message type
	streamID := t.selectStream(message.MessageType)
	
	// Send message
	if err := t.sendOnStream(conn, streamID, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	// Update metrics
	t.metrics.MessagesSent.Add(1)
	t.metrics.BytesSent.Add(int64(len(data)))
	conn.metrics.MessagesSent++
	conn.metrics.BytesSent += int64(len(data))
	
	return nil
}

// SetMessageHandler sets the message handler for received E2AP messages
func (t *SCTPTransport) SetMessageHandler(handler E2APMessageHandler) {
	t.messageHandler = handler
}

// acceptLoop accepts incoming SCTP connections
func (t *SCTPTransport) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopChan:
			return
		default:
			// Accept connection with timeout
			if tcpListener, ok := t.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			}
			
			conn, err := t.listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if !t.running.Load() {
					return
				}
				continue
			}
			
			// Handle new connection
			go t.handleIncomingConnection(conn)
		}
	}
}

// handleIncomingConnection handles a new incoming connection
func (t *SCTPTransport) handleIncomingConnection(conn net.Conn) {
	// Perform SCTP handshake and E2 setup
	// For now, we'll use a simplified approach
	
	// Read initial E2 Setup Request to identify the node
	message, nodeID, err := t.readInitialMessage(conn)
	if err != nil {
		conn.Close()
		return
	}
	
	// Check if we can accept this connection
	t.mutex.Lock()
	if len(t.connections) >= t.config.MaxConnections {
		t.mutex.Unlock()
		conn.Close()
		return
	}
	
	// Create SCTP connection
	sctpConn := &SCTPConnection{
		conn:            conn,
		nodeID:          nodeID,
		streams:         make(map[uint16]*SCTPStream),
		state:           SCTPStateEstablished,
		metrics:         &SCTPConnectionMetrics{},
		lastActivity:    time.Now(),
		pendingMessages: make(map[uint32]*PendingMessage),
		sequenceNumber:  0,
	}
	
	// Initialize streams
	t.initializeStreams(sctpConn)
	
	// Store connection
	t.connections[nodeID] = sctpConn
	t.mutex.Unlock()
	
	// Process the initial message
	if t.messageHandler != nil {
		t.messageHandler(nodeID, message)
	}
	
	// Start connection handler
	go t.handleConnection(sctpConn)
	
	// Update metrics
	t.metrics.ConnectionsTotal.Add(1)
	t.metrics.ConnectionsActive.Add(1)
}

// handleConnection handles an established SCTP connection
func (t *SCTPTransport) handleConnection(conn *SCTPConnection) {
	defer func() {
		t.mutex.Lock()
		delete(t.connections, conn.nodeID)
		t.mutex.Unlock()
		
		conn.conn.Close()
		t.metrics.ConnectionsActive.Add(-1)
	}()
	
	// Create read buffer
	buffer := make([]byte, t.config.MaxMessageSize)
	
	for {
		// Set read deadline
		conn.conn.SetReadDeadline(time.Now().Add(t.config.HeartbeatInterval))
		
		// Read message length (4 bytes)
		lengthBytes := buffer[:4]
		if _, err := io.ReadFull(conn.conn, lengthBytes); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Send heartbeat
				t.sendHeartbeat(conn)
				continue
			}
			return
		}
		
		// Parse message length
		messageLength := binary.BigEndian.Uint32(lengthBytes)
		if messageLength > uint32(t.config.MaxMessageSize) {
			return
		}
		
		// Read message data
		messageData := buffer[:messageLength]
		if _, err := io.ReadFull(conn.conn, messageData); err != nil {
			return
		}
		
		// Decode E2AP message
		message, err := t.codec.DecodeE2APMessage(messageData)
		if err != nil {
			t.metrics.Errors.Add(1)
			continue
		}
		
		// Update metrics
		t.metrics.MessagesReceived.Add(1)
		t.metrics.BytesReceived.Add(int64(messageLength))
		conn.metrics.MessagesReceived++
		conn.metrics.BytesReceived += int64(messageLength)
		conn.lastActivity = time.Now()
		
		// Handle message
		if t.messageHandler != nil {
			if err := t.messageHandler(conn.nodeID, message); err != nil {
				t.metrics.Errors.Add(1)
			}
		}
	}
}

// initializeStreams initializes SCTP streams for a connection
func (t *SCTPTransport) initializeStreams(conn *SCTPConnection) {
	// Create control stream (stream 0)
	conn.streams[0] = &SCTPStream{
		streamID:       0,
		streamType:     SCTPStreamTypeControl,
		sequenceNumber: 0,
		inboundQueue:   make(chan *E2APMessage, 100),
		outboundQueue:  make(chan *E2APMessage, 100),
		state:          SCTPStreamStateOpen,
		metrics:        &SCTPStreamMetrics{},
	}
	
	// Create data streams (streams 1-3)
	for i := uint16(1); i <= 3; i++ {
		conn.streams[i] = &SCTPStream{
			streamID:       i,
			streamType:     SCTPStreamTypeData,
			sequenceNumber: 0,
			inboundQueue:   make(chan *E2APMessage, 100),
			outboundQueue:  make(chan *E2APMessage, 100),
			state:          SCTPStreamStateOpen,
			metrics:        &SCTPStreamMetrics{},
		}
	}
	
	// Create management stream (stream 4)
	conn.streams[4] = &SCTPStream{
		streamID:       4,
		streamType:     SCTPStreamTypeManagement,
		sequenceNumber: 0,
		inboundQueue:   make(chan *E2APMessage, 100),
		outboundQueue:  make(chan *E2APMessage, 100),
		state:          SCTPStreamStateOpen,
		metrics:        &SCTPStreamMetrics{},
	}
}

// selectStream selects the appropriate SCTP stream for a message type
func (t *SCTPTransport) selectStream(messageType E2APMessageType) uint16 {
	switch messageType {
	case E2APMessageTypeSetupRequest, E2APMessageTypeSetupResponse, E2APMessageTypeSetupFailure:
		return 0 // Control stream
	case E2APMessageTypeRICSubscriptionRequest, E2APMessageTypeRICSubscriptionResponse,
		 E2APMessageTypeRICSubscriptionFailure, E2APMessageTypeRICSubscriptionDeleteRequest,
		 E2APMessageTypeRICSubscriptionDeleteResponse, E2APMessageTypeRICSubscriptionDeleteFailure:
		return 1 // Subscription stream
	case E2APMessageTypeRICIndication:
		return 2 // Indication stream
	case E2APMessageTypeRICControlRequest, E2APMessageTypeRICControlAcknowledge, E2APMessageTypeRICControlFailure:
		return 3 // Control stream
	default:
		return 4 // Management stream
	}
}

// sendOnStream sends data on a specific SCTP stream
func (t *SCTPTransport) sendOnStream(conn *SCTPConnection, streamID uint16, data []byte) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	
	// Check connection state
	if conn.state != SCTPStateEstablished {
		return fmt.Errorf("connection not established")
	}
	
	// Get stream
	stream, exists := conn.streams[streamID]
	if !exists {
		return fmt.Errorf("stream %d does not exist", streamID)
	}
	
	// Check stream state
	if stream.state != SCTPStreamStateOpen {
		return fmt.Errorf("stream %d is not open", streamID)
	}
	
	// Create SCTP data chunk with stream information
	// In a real SCTP implementation, this would include proper SCTP headers
	chunk := &bytes.Buffer{}
	
	// Write message length (4 bytes)
	if err := binary.Write(chunk, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	
	// Write message data
	if _, err := chunk.Write(data); err != nil {
		return err
	}
	
	// Send over connection
	if _, err := conn.conn.Write(chunk.Bytes()); err != nil {
		return err
	}
	
	// Update stream metrics
	stream.metrics.MessagesSent++
	
	return nil
}

// sendHeartbeat sends an SCTP heartbeat
func (t *SCTPTransport) sendHeartbeat(conn *SCTPConnection) {
	// In a real SCTP implementation, this would send an SCTP HEARTBEAT chunk
	// For now, we'll send a simple keepalive message
	
	heartbeat := []byte{0x00, 0x00, 0x00, 0x04, 'H', 'B', 'A', 'T'}
	conn.conn.Write(heartbeat)
	
	t.metrics.HeartbeatsSent.Add(1)
}

// readInitialMessage reads and decodes the initial E2AP message from a connection
func (t *SCTPTransport) readInitialMessage(conn net.Conn) (*E2APMessage, string, error) {
	// Set timeout for initial message
	conn.SetReadDeadline(time.Now().Add(t.config.InitTimeout))
	
	// Read message length
	lengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBytes); err != nil {
		return nil, "", err
	}
	
	messageLength := binary.BigEndian.Uint32(lengthBytes)
	if messageLength > uint32(t.config.MaxMessageSize) {
		return nil, "", fmt.Errorf("message too large: %d bytes", messageLength)
	}
	
	// Read message data
	messageData := make([]byte, messageLength)
	if _, err := io.ReadFull(conn, messageData); err != nil {
		return nil, "", err
	}
	
	// Decode E2AP message
	message, err := t.codec.DecodeE2APMessage(messageData)
	if err != nil {
		return nil, "", err
	}
	
	// Extract node ID from E2 Setup Request
	if message.MessageType != E2APMessageTypeSetupRequest {
		return nil, "", fmt.Errorf("expected E2 Setup Request, got %v", message.MessageType)
	}
	
	setupReq, ok := message.Payload.(*E2SetupRequest)
	if !ok {
		return nil, "", fmt.Errorf("invalid E2 Setup Request payload")
	}
	
	nodeID := fmt.Sprintf("%s-%s", setupReq.GlobalE2NodeID.PLMNIdentity.MCC, setupReq.GlobalE2NodeID.NodeID)
	
	return message, nodeID, nil
}

// maintenanceLoop performs periodic maintenance tasks
func (t *SCTPTransport) maintenanceLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.performMaintenance()
		}
	}
}

// performMaintenance performs maintenance tasks
func (t *SCTPTransport) performMaintenance() {
	t.mutex.RLock()
	connections := make([]*SCTPConnection, 0, len(t.connections))
	for _, conn := range t.connections {
		connections = append(connections, conn)
	}
	t.mutex.RUnlock()
	
	now := time.Now()
	for _, conn := range connections {
		// Check for idle connections
		if now.Sub(conn.lastActivity) > t.config.ConnectionTimeout {
			t.mutex.Lock()
			t.closeConnection(conn.nodeID, conn)
			delete(t.connections, conn.nodeID)
			t.mutex.Unlock()
		}
	}
}

// closeConnection closes an SCTP connection
func (t *SCTPTransport) closeConnection(nodeID string, conn *SCTPConnection) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	
	if conn.state == SCTPStateClosed {
		return
	}
	
	conn.state = SCTPStateClosed
	
	// Close all streams
	for _, stream := range conn.streams {
		stream.state = SCTPStreamStateClosed
		close(stream.inboundQueue)
		close(stream.outboundQueue)
	}
	
	// Close the underlying connection
	conn.conn.Close()
}

// GetMetrics returns transport metrics
func (t *SCTPTransport) GetMetrics() *SCTPMetrics {
	return t.metrics
}

// GetConnectionMetrics returns metrics for a specific connection
func (t *SCTPTransport) GetConnectionMetrics(nodeID string) (*SCTPConnectionMetrics, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	conn, exists := t.connections[nodeID]
	if !exists {
		return nil, fmt.Errorf("no connection to node %s", nodeID)
	}
	
	return conn.metrics, nil
}

// SCTPConnectionPool manages a pool of SCTP connections
type SCTPConnectionPool struct {
	maxConnections int
	connections    map[string]*SCTPConnection
	mutex          sync.RWMutex
}

// NewSCTPConnectionPool creates a new connection pool
func NewSCTPConnectionPool(maxConnections int) *SCTPConnectionPool {
	return &SCTPConnectionPool{
		maxConnections: maxConnections,
		connections:    make(map[string]*SCTPConnection),
	}
}

// Get gets a connection from the pool
func (p *SCTPConnectionPool) Get(nodeID string) (*SCTPConnection, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	conn, exists := p.connections[nodeID]
	return conn, exists
}

// Put puts a connection into the pool
func (p *SCTPConnectionPool) Put(nodeID string, conn *SCTPConnection) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if len(p.connections) >= p.maxConnections {
		return errors.New("connection pool is full")
	}
	
	p.connections[nodeID] = conn
	return nil
}

// Remove removes a connection from the pool
func (p *SCTPConnectionPool) Remove(nodeID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	delete(p.connections, nodeID)
}

// Size returns the current size of the pool
func (p *SCTPConnectionPool) Size() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	return len(p.connections)
}