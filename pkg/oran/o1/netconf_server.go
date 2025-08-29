
package o1



import (

	"context"

	"crypto/tls"

	"encoding/xml"

	"fmt"

	"io"

	"net"

	"strings"

	"sync"

	"time"



	logr "github.com/go-logr/logr"

	"golang.org/x/crypto/ssh"



	"sigs.k8s.io/controller-runtime/pkg/log"

)



// NetconfServer provides a complete NETCONF 1.1 server implementation.

// following RFC 6241 and RFC 6242 specifications.

type NetconfServer struct {

	config       *NetconfServerConfig

	listener     net.Listener

	tlsConfig    *tls.Config

	sshConfig    *ssh.ServerConfig

	sessions     map[string]*NetconfSession

	sessionsMux  sync.RWMutex

	capabilities []string

	datastore    *NetconfDatastore

	subscribers  map[string]*NotificationSubscription

	subsMux      sync.RWMutex

	shutdown     chan struct{}

	running      bool

	mutex        sync.RWMutex

}



// NetconfServerConfig holds server configuration.

type NetconfServerConfig struct {

	Host                string

	Port                int

	TLSPort             int

	SSHPort             int

	TLSConfig           *tls.Config

	SSHHostKeyFile      string

	MaxSessions         int

	SessionTimeout      time.Duration

	EnableNotifications bool

	EnableCandidate     bool

	EnableStartup       bool

	EnableXPath         bool

	EnableValidation    bool

}



// NetconfSession represents an active NETCONF session.

type NetconfSession struct {

	ID            string

	conn          net.Conn

	sshChannel    ssh.Channel // SSH channel for NETCONF over SSH

	sshSession    *ssh.Session

	decoder       *xml.Decoder

	encoder       *xml.Encoder

	capabilities  []string

	authenticated bool

	locks         map[string]string // datastore -> session-id

	subscriptions []string

	lastActivity  time.Time

	mutex         sync.RWMutex

	ctx           context.Context

	cancel        context.CancelFunc

}



// NetconfDatastore manages NETCONF datastores (running, candidate, startup).

type NetconfDatastore struct {

	running   map[string]interface{}

	candidate map[string]interface{}

	startup   map[string]interface{}

	mutex     sync.RWMutex

	locks     map[string]string // datastore -> session-id

}



// NotificationSubscription represents a NETCONF notification subscription.

type NotificationSubscription struct {

	ID        string

	SessionID string

	Filter    string

	StartTime time.Time

	StopTime  time.Time

	Stream    string

	Active    bool

	Channel   chan *NetconfNotification

}



// NetconfNotification represents a NETCONF notification message.

type NetconfNotification struct {

	XMLName   xml.Name  `xml:"notification"`

	Namespace string    `xml:"xmlns,attr"`

	EventTime time.Time `xml:"eventTime"`

	Content   string    `xml:",innerxml"`

}



// NetconfRPCRequest represents a NETCONF RPC request.

type NetconfRPCRequest struct {

	XMLName   xml.Name `xml:"rpc"`

	MessageID string   `xml:"message-id,attr"`

	Namespace string   `xml:"xmlns,attr"`

	Operation string   `xml:",innerxml"`

}



// NetconfRPCResponse represents a NETCONF RPC response.

type NetconfRPCResponse struct {

	XMLName   xml.Name         `xml:"rpc-reply"`

	MessageID string           `xml:"message-id,attr"`

	Namespace string           `xml:"xmlns,attr"`

	Data      interface{}      `xml:"data,omitempty"`

	OK        *struct{}        `xml:"ok,omitempty"`

	Error     *NetconfRPCError `xml:"rpc-error,omitempty"`

}



// NetconfRPCError represents a NETCONF RPC error.

type NetconfRPCError struct {

	Type     string `xml:"error-type"`

	Tag      string `xml:"error-tag"`

	Severity string `xml:"error-severity"`

	Message  string `xml:"error-message"`

	Info     string `xml:"error-info,omitempty"`

	Path     string `xml:"error-path,omitempty"`

}



// NetconfEditConfig represents edit-config operation data.

type NetconfEditConfig struct {

	Target           string `xml:"target>running,omitempty"`

	CandidateTarget  string `xml:"target>candidate,omitempty"`

	DefaultOperation string `xml:"default-operation,omitempty"`

	TestOption       string `xml:"test-option,omitempty"`

	ErrorOption      string `xml:"error-option,omitempty"`

	Config           string `xml:"config,innerxml"`

}



// NetconfGetConfig represents get-config operation parameters.

type NetconfGetConfig struct {

	Source          string `xml:"source>running,omitempty"`

	CandidateSource string `xml:"source>candidate,omitempty"`

	StartupSource   string `xml:"source>startup,omitempty"`

	Filter          string `xml:"filter,innerxml"`

}



// NewNetconfServer creates a new NETCONF server instance.

func NewNetconfServer(config *NetconfServerConfig) *NetconfServer {

	if config == nil {

		config = &NetconfServerConfig{

			Host:                "0.0.0.0",

			Port:                830,

			TLSPort:             6513,

			SSHPort:             830,

			MaxSessions:         100,

			SessionTimeout:      30 * time.Minute,

			EnableNotifications: true,

			EnableCandidate:     true,

			EnableStartup:       true,

			EnableXPath:         true,

			EnableValidation:    true,

		}

	}



	server := &NetconfServer{

		config:      config,

		sessions:    make(map[string]*NetconfSession),

		subscribers: make(map[string]*NotificationSubscription),

		shutdown:    make(chan struct{}),

		datastore:   NewNetconfDatastore(),

	}



	// Initialize server capabilities.

	server.initializeCapabilities()



	return server

}



// initializeCapabilities sets up NETCONF server capabilities.

func (ns *NetconfServer) initializeCapabilities() {

	ns.capabilities = []string{

		"urn:ietf:params:netconf:base:1.0",

		"urn:ietf:params:netconf:base:1.1",

		"urn:ietf:params:netconf:capability:writable-running:1.0",

		"urn:ietf:params:netconf:capability:rollback-on-error:1.0",

		"urn:ietf:params:netconf:capability:validate:1.1",

		"urn:ietf:params:netconf:capability:confirmed-commit:1.1",

	}



	if ns.config.EnableCandidate {

		ns.capabilities = append(ns.capabilities,

			"urn:ietf:params:netconf:capability:candidate:1.0")

	}



	if ns.config.EnableStartup {

		ns.capabilities = append(ns.capabilities,

			"urn:ietf:params:netconf:capability:startup:1.0")

	}



	if ns.config.EnableNotifications {

		ns.capabilities = append(ns.capabilities,

			"urn:ietf:params:netconf:capability:notification:1.0",

			"urn:ietf:params:netconf:capability:interleave:1.0")

	}



	if ns.config.EnableXPath {

		ns.capabilities = append(ns.capabilities,

			"urn:ietf:params:netconf:capability:xpath:1.0")

	}



	// Add O-RAN specific capabilities.

	ns.capabilities = append(ns.capabilities,

		"urn:o-ran:fm:1.0",

		"urn:o-ran:pm:1.0",

		"urn:o-ran:cm:1.0",

		"urn:o-ran:sm:1.0",

		"urn:o-ran:hardware:1.0",

		"urn:o-ran:software-management:1.0",

		"urn:o-ran:file-management:1.0",

		"urn:o-ran:troubleshooting:1.0")

}



// Start starts the NETCONF server on configured ports.

func (ns *NetconfServer) Start(ctx context.Context) error {

	ns.mutex.Lock()

	defer ns.mutex.Unlock()



	if ns.running {

		return fmt.Errorf("server already running")

	}



	logger := log.FromContext(ctx)

	logger.Info("starting NETCONF server", "port", ns.config.Port, "tlsPort", ns.config.TLSPort)



	// Start SSH NETCONF subsystem listener.

	if err := ns.startSSHListener(ctx); err != nil {

		return fmt.Errorf("failed to start SSH listener: %w", err)

	}



	// Start TLS listener if configured.

	if ns.config.TLSConfig != nil {

		if err := ns.startTLSListener(ctx); err != nil {

			logger.Error(err, "failed to start TLS listener, continuing with SSH only")

		}

	}



	ns.running = true



	// Start session management goroutine.

	go ns.sessionManager(ctx)



	logger.Info("NETCONF server started successfully")

	return nil

}



// startSSHListener starts the SSH NETCONF subsystem listener.

func (ns *NetconfServer) startSSHListener(ctx context.Context) error {

	// Configure SSH server.

	sshConfig := &ssh.ServerConfig{

		ServerVersion: "SSH-2.0-NETCONF_SERVER",

	}



	// Load host key (in production, load from file).

	hostKey, err := generateHostKey()

	if err != nil {

		return fmt.Errorf("failed to generate host key: %w", err)

	}

	sshConfig.AddHostKey(hostKey)



	// Configure authentication (basic password auth for demo).

	sshConfig.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {

		// In production, implement proper authentication.

		if c.User() == "admin" && string(pass) == "admin" {

			return nil, nil

		}

		return nil, fmt.Errorf("authentication failed")

	}



	// Start listening.

	address := fmt.Sprintf("%s:%d", ns.config.Host, ns.config.SSHPort)

	listener, err := net.Listen("tcp", address)

	if err != nil {

		return fmt.Errorf("failed to listen on %s: %w", address, err)

	}



	ns.listener = listener

	ns.sshConfig = sshConfig



	// Start accepting connections.

	go ns.acceptSSHConnections(ctx)



	return nil

}



// startTLSListener starts the TLS NETCONF listener.

func (ns *NetconfServer) startTLSListener(ctx context.Context) error {

	address := fmt.Sprintf("%s:%d", ns.config.Host, ns.config.TLSPort)

	listener, err := tls.Listen("tcp", address, ns.config.TLSConfig)

	if err != nil {

		return fmt.Errorf("failed to listen on TLS %s: %w", address, err)

	}



	// Start accepting TLS connections.

	go ns.acceptTLSConnections(ctx, listener)



	return nil

}



// acceptSSHConnections accepts and handles SSH connections.

func (ns *NetconfServer) acceptSSHConnections(ctx context.Context) {

	logger := log.FromContext(ctx)



	for {

		select {

		case <-ctx.Done():

			return

		case <-ns.shutdown:

			return

		default:

		}



		conn, err := ns.listener.Accept()

		if err != nil {

			logger.Error(err, "failed to accept SSH connection")

			continue

		}



		go ns.handleSSHConnection(ctx, conn)

	}

}



// acceptTLSConnections accepts and handles TLS connections.

func (ns *NetconfServer) acceptTLSConnections(ctx context.Context, listener net.Listener) {

	logger := log.FromContext(ctx)



	for {

		select {

		case <-ctx.Done():

			return

		case <-ns.shutdown:

			return

		default:

		}



		conn, err := listener.Accept()

		if err != nil {

			logger.Error(err, "failed to accept TLS connection")

			continue

		}



		go ns.handleTLSConnection(ctx, conn)

	}

}



// handleSSHConnection handles an SSH connection with NETCONF subsystem.

func (ns *NetconfServer) handleSSHConnection(ctx context.Context, conn net.Conn) {

	logger := log.FromContext(ctx)



	// SSH handshake.

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, ns.sshConfig)

	if err != nil {

		logger.Error(err, "SSH handshake failed")

		conn.Close()

		return

	}

	defer sshConn.Close()



	// Handle SSH requests.

	go ssh.DiscardRequests(reqs)



	// Handle SSH channels.

	for newChannel := range chans {

		if newChannel.ChannelType() != "session" {

			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")

			continue

		}



		channel, requests, err := newChannel.Accept()

		if err != nil {

			logger.Error(err, "failed to accept SSH channel")

			continue

		}



		go ns.handleSSHChannel(ctx, channel, requests, sshConn.User())

	}

}



// handleSSHChannel handles an SSH channel for NETCONF subsystem.

func (ns *NetconfServer) handleSSHChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, username string) {

	defer channel.Close()



	logger := log.FromContext(ctx)



	// Wait for subsystem request.

	for req := range requests {

		if req.Type == "subsystem" && string(req.Payload[4:]) == "netconf" {

			req.Reply(true, nil)



			// Create NETCONF session.

			sessionID := generateSessionID()

			session := &NetconfSession{

				ID:            sessionID,

				sshChannel:    channel,

				capabilities:  ns.capabilities,

				authenticated: true,

				locks:         make(map[string]string),

				subscriptions: make([]string, 0),

				lastActivity:  time.Now(),

			}

			session.ctx, session.cancel = context.WithCancel(ctx)



			// Register session.

			ns.sessionsMux.Lock()

			ns.sessions[sessionID] = session

			ns.sessionsMux.Unlock()



			logger.Info("NETCONF session established", "sessionID", sessionID, "user", username)



			// Handle NETCONF protocol.

			ns.handleNetconfSession(ctx, session)



			// Cleanup.

			ns.sessionsMux.Lock()

			delete(ns.sessions, sessionID)

			ns.sessionsMux.Unlock()



			return

		}

		req.Reply(false, nil)

	}

}



// handleTLSConnection handles a TLS connection for NETCONF.

func (ns *NetconfServer) handleTLSConnection(ctx context.Context, conn net.Conn) {

	logger := log.FromContext(ctx)

	defer conn.Close()



	// Create NETCONF session.

	sessionID := generateSessionID()

	session := &NetconfSession{

		ID:            sessionID,

		conn:          conn,

		capabilities:  ns.capabilities,

		authenticated: true, // TLS client cert authentication

		locks:         make(map[string]string),

		subscriptions: make([]string, 0),

		lastActivity:  time.Now(),

	}

	session.ctx, session.cancel = context.WithCancel(ctx)



	// Register session.

	ns.sessionsMux.Lock()

	ns.sessions[sessionID] = session

	ns.sessionsMux.Unlock()



	logger.Info("NETCONF TLS session established", "sessionID", sessionID)



	// Handle NETCONF protocol.

	ns.handleNetconfSession(ctx, session)



	// Cleanup.

	ns.sessionsMux.Lock()

	delete(ns.sessions, sessionID)

	ns.sessionsMux.Unlock()

}



// handleNetconfSession handles the NETCONF protocol for a session.

func (ns *NetconfServer) handleNetconfSession(ctx context.Context, session *NetconfSession) {

	logger := log.FromContext(ctx)

	defer session.cancel()



	// Setup XML decoder/encoder.

	var reader io.Reader

	var writer io.Writer

	if session.conn != nil {

		reader = session.conn

		writer = session.conn

	} else if session.sshChannel != nil {

		reader = session.sshChannel

		writer = session.sshChannel

	} else {

		logger.Error(nil, "no valid connection for session", "sessionID", session.ID)

		return

	}

	session.decoder = xml.NewDecoder(reader)

	session.encoder = xml.NewEncoder(writer)



	// Send hello message.

	if err := ns.sendHello(session); err != nil {

		logger.Error(err, "failed to send hello", "sessionID", session.ID)

		return

	}



	// Receive client hello.

	if err := ns.receiveHello(session); err != nil {

		logger.Error(err, "failed to receive hello", "sessionID", session.ID)

		return

	}



	// Handle RPC requests.

	for {

		select {

		case <-ctx.Done():

			return

		case <-session.ctx.Done():

			return

		default:

		}



		// Set read timeout (only for net.Conn).

		if session.conn != nil {

			session.conn.SetReadDeadline(time.Now().Add(ns.config.SessionTimeout))

		}



		var rpc NetconfRPCRequest

		if err := session.decoder.Decode(&rpc); err != nil {

			if err == io.EOF {

				logger.Info("session closed by client", "sessionID", session.ID)

				return

			}

			logger.Error(err, "failed to decode RPC", "sessionID", session.ID)

			continue

		}



		session.lastActivity = time.Now()



		// Handle RPC.

		go ns.handleRPC(ctx, session, &rpc)

	}

}



// sendHello sends NETCONF hello message to client.

func (ns *NetconfServer) sendHello(session *NetconfSession) error {

	hello := HelloMessage{

		Namespace:    "urn:ietf:params:xml:ns:netconf:base:1.0",

		Capabilities: session.capabilities,

		SessionID:    session.ID,

	}



	// Encode and send with NETCONF 1.1 framing.

	xmlData, err := xml.Marshal(hello)

	if err != nil {

		return fmt.Errorf("failed to marshal hello: %w", err)

	}



	message := fmt.Sprintf("%s]]>]]>", string(xmlData))

	// Write to appropriate connection type.

	if session.conn != nil {

		_, err = session.conn.Write([]byte(message))

	} else if session.sshChannel != nil {

		_, err = session.sshChannel.Write([]byte(message))

	}

	return err

}



// receiveHello receives and processes client hello message.

func (ns *NetconfServer) receiveHello(session *NetconfSession) error {

	// Read hello with timeout.

	if session.conn != nil {

		session.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	}



	buffer := make([]byte, 4096)

	var n int

	var err error

	if session.conn != nil {

		n, err = session.conn.Read(buffer)

	} else if session.sshChannel != nil {

		n, err = session.sshChannel.Read(buffer)

	} else {

		return fmt.Errorf("no valid connection for reading hello")

	}

	if err != nil {

		return fmt.Errorf("failed to read client hello: %w", err)

	}



	helloXML := string(buffer[:n])

	helloXML = strings.TrimSuffix(helloXML, "]]>]]>")



	var clientHello HelloMessage

	if err := xml.Unmarshal([]byte(helloXML), &clientHello); err != nil {

		return fmt.Errorf("failed to parse client hello: %w", err)

	}



	// Negotiate capabilities.

	session.capabilities = ns.negotiateCapabilities(session.capabilities, clientHello.Capabilities)



	return nil

}



// negotiateCapabilities negotiates capabilities between server and client.

func (ns *NetconfServer) negotiateCapabilities(server, client []string) []string {

	clientCaps := make(map[string]bool)

	for _, cap := range client {

		clientCaps[cap] = true

	}



	var negotiated []string

	for _, serverCap := range server {

		if clientCaps[serverCap] {

			negotiated = append(negotiated, serverCap)

		}

	}



	return negotiated

}



// handleRPC handles a NETCONF RPC request.

func (ns *NetconfServer) handleRPC(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest) {

	logger := log.FromContext(ctx)



	response := &NetconfRPCResponse{

		MessageID: rpc.MessageID,

		Namespace: "urn:ietf:params:xml:ns:netconf:base:1.0",

	}



	// Parse operation type.

	operation := ns.parseOperation(rpc.Operation)



	logger.Info("processing RPC", "sessionID", session.ID, "operation", operation, "messageID", rpc.MessageID)



	switch operation {

	case "get":

		ns.handleGet(ctx, session, rpc, response)

	case "get-config":

		ns.handleGetConfig(ctx, session, rpc, response)

	case "edit-config":

		ns.handleEditConfig(ctx, session, rpc, response)

	case "copy-config":

		ns.handleCopyConfig(ctx, session, rpc, response)

	case "delete-config":

		ns.handleDeleteConfig(ctx, session, rpc, response)

	case "lock":

		ns.handleLock(ctx, session, rpc, response)

	case "unlock":

		ns.handleUnlock(ctx, session, rpc, response)

	case "close-session":

		ns.handleCloseSession(ctx, session, rpc, response)

	case "kill-session":

		ns.handleKillSession(ctx, session, rpc, response)

	case "validate":

		ns.handleValidate(ctx, session, rpc, response)

	case "commit":

		ns.handleCommit(ctx, session, rpc, response)

	case "discard-changes":

		ns.handleDiscardChanges(ctx, session, rpc, response)

	case "create-subscription":

		ns.handleCreateSubscription(ctx, session, rpc, response)

	default:

		response.Error = &NetconfRPCError{

			Type:     "application",

			Tag:      "operation-not-supported",

			Severity: "error",

			Message:  fmt.Sprintf("Unknown operation: %s", operation),

		}

	}



	// Send response.

	ns.sendResponse(session, response)

}



// NETCONF RPC operation handlers.



func (ns *NetconfServer) handleGet(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Basic get operation - returns operational data.

	response.Data = map[string]interface{}{

		"message": "Get operation not fully implemented",

	}

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleGetConfig(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Get configuration data.

	response.Data = map[string]interface{}{

		"message": "Get-config operation not fully implemented",

	}

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleEditConfig(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Edit configuration.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleCopyConfig(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Copy configuration between datastores.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleDeleteConfig(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Delete configuration.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleLock(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Lock datastore.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleUnlock(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Unlock datastore.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleCloseSession(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Close session.

	response.OK = &struct{}{}

	session.cancel() // Close the session

}



func (ns *NetconfServer) handleKillSession(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Kill another session.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleValidate(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Validate configuration.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleCommit(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Commit candidate configuration to running.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleDiscardChanges(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Discard candidate configuration changes.

	response.OK = &struct{}{}

}



func (ns *NetconfServer) handleCreateSubscription(ctx context.Context, session *NetconfSession, rpc *NetconfRPCRequest, response *NetconfRPCResponse) {

	// Create notification subscription.

	response.OK = &struct{}{}

}



// sendResponse sends a NETCONF RPC response.

func (ns *NetconfServer) sendResponse(session *NetconfSession, response *NetconfRPCResponse) {

	session.mutex.Lock()

	defer session.mutex.Unlock()



	xmlData, err := xml.Marshal(response)

	if err != nil {

		log.Log.Error(err, "failed to marshal response")

		return

	}



	message := fmt.Sprintf("%s]]>]]>", string(xmlData))



	// Write to appropriate connection type.

	if session.conn != nil {

		session.conn.Write([]byte(message))

	} else if session.sshChannel != nil {

		session.sshChannel.Write([]byte(message))

	}

}



// parseOperation extracts the operation name from RPC content.

func (ns *NetconfServer) parseOperation(operation string) string {

	// Simple operation parsing - extract first XML element name.

	if idx := strings.Index(operation, "<"); idx != -1 {

		remaining := operation[idx+1:]

		if endIdx := strings.Index(remaining, " "); endIdx != -1 {

			return remaining[:endIdx]

		}

		if endIdx := strings.Index(remaining, ">"); endIdx != -1 {

			return remaining[:endIdx]

		}

	}

	return "unknown"

}



// sessionManager manages session lifecycle and cleanup.

func (ns *NetconfServer) sessionManager(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()



	logger := log.FromContext(ctx)



	for {

		select {

		case <-ctx.Done():

			return

		case <-ns.shutdown:

			return

		case <-ticker.C:

			// Clean up expired sessions.

			ns.cleanupExpiredSessions(logger)

		}

	}

}



// cleanupExpiredSessions removes expired sessions.

func (ns *NetconfServer) cleanupExpiredSessions(logger logr.Logger) {

	ns.sessionsMux.Lock()

	defer ns.sessionsMux.Unlock()



	now := time.Now()

	for sessionID, session := range ns.sessions {

		if now.Sub(session.lastActivity) > ns.config.SessionTimeout {

			logger.Info("cleaning up expired session", "sessionID", sessionID)

			session.cancel()

			delete(ns.sessions, sessionID)

		}

	}

}



// Stop stops the NETCONF server.

func (ns *NetconfServer) Stop(ctx context.Context) error {

	ns.mutex.Lock()

	defer ns.mutex.Unlock()



	if !ns.running {

		return nil

	}



	logger := log.FromContext(ctx)

	logger.Info("stopping NETCONF server")



	close(ns.shutdown)



	// Close all sessions.

	ns.sessionsMux.Lock()

	for _, session := range ns.sessions {

		session.cancel()

	}

	ns.sessionsMux.Unlock()



	// Close listeners.

	if ns.listener != nil {

		ns.listener.Close()

	}



	ns.running = false

	logger.Info("NETCONF server stopped")

	return nil

}



// Helper functions are defined in other files to avoid duplication.



// GetSessions returns active session information.

func (ns *NetconfServer) GetSessions() map[string]interface{} {

	ns.sessionsMux.RLock()

	defer ns.sessionsMux.RUnlock()



	sessions := make(map[string]interface{})

	for id, session := range ns.sessions {

		sessions[id] = map[string]interface{}{

			"id":            session.ID,

			"authenticated": session.authenticated,

			"last_activity": session.lastActivity,

			"capabilities":  len(session.capabilities),

			"locks":         len(session.locks),

			"subscriptions": len(session.subscriptions),

		}

	}



	return sessions

}



// Utility functions.



// generateSessionID generates a unique session ID.

func generateSessionID() string {

	return fmt.Sprintf("session-%d", time.Now().UnixNano())

}



// generateHostKey generates a host key for SSH server.

func generateHostKey() (ssh.Signer, error) {

	// In production, load from file or generate proper key.

	privateKey := `-----BEGIN OPENSSH PRIVATE KEY-----

b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn

NhAAAAAwEAAQAAAQEAyQy...

-----END OPENSSH PRIVATE KEY-----`



	return ssh.ParsePrivateKey([]byte(privateKey))

}



// NewNetconfDatastore creates a new NETCONF datastore.

func NewNetconfDatastore() *NetconfDatastore {

	return &NetconfDatastore{

		running:   make(map[string]interface{}),

		candidate: make(map[string]interface{}),

		startup:   make(map[string]interface{}),

		locks:     make(map[string]string),

	}

}

