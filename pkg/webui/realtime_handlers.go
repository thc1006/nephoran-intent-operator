/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package webui

import (
	
	"encoding/json"
"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
)

// StreamFilter represents filters for real-time streams.

type StreamFilter struct {
	EventTypes []string `json:"event_types,omitempty"` // intent, package, cluster, deployment

	Status []string `json:"status,omitempty"` // pending, processing, completed, failed

	Priority []string `json:"priority,omitempty"` // low, medium, high, critical

	Components []string `json:"components,omitempty"` // AMF, SMF, UPF, etc.

	Clusters []string `json:"clusters,omitempty"` // cluster names/IDs

	Namespaces []string `json:"namespaces,omitempty"` // kubernetes namespaces

	Labels map[string]string `json:"labels,omitempty"` // label selectors

	Since *time.Time `json:"since,omitempty"` // events since timestamp
}

// StreamMessage represents a generic real-time stream message.

type StreamMessage struct {
	Type string `json:"type"` // intent_update, package_update, cluster_update, system_event

	ID string `json:"id"` // unique message ID

	Timestamp time.Time `json:"timestamp"`

	Source string `json:"source"` // source component/service

	Data interface{} `json:"data"` // actual event data

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// WebSocketMessage represents a WebSocket-specific message with additional controls.

type WebSocketMessage struct {
	StreamMessage

	Action string `json:"action,omitempty"` // subscribe, unsubscribe, ping, pong

	Filters *StreamFilter `json:"filters,omitempty"` // subscription filters

	RequestID string `json:"request_id,omitempty"` // for request/response correlation
}

// SystemEvent represents system-wide events.

type SystemEvent struct {
	Level string `json:"level"` // info, warning, error, critical

	Component string `json:"component"` // which component generated the event

	Event string `json:"event"` // event type

	Message string `json:"message"` // human-readable message

	Details json.RawMessage `json:"details,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// MetricsUpdate represents real-time metrics updates.

type MetricsUpdate struct {
	MetricName string `json:"metric_name"`

	Value float64 `json:"value"`

	Labels map[string]string `json:"labels,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Unit string `json:"unit,omitempty"`
}

// setupRealtimeRoutes sets up real-time streaming API routes.

func (s *NephoranAPIServer) setupRealtimeRoutes(router *mux.Router) {
	realtime := router.PathPrefix("/realtime").Subrouter()

	// WebSocket endpoints.

	realtime.HandleFunc("/ws", s.handleWebSocket).Methods("GET")

	realtime.HandleFunc("/ws/intents", s.handleIntentWebSocket).Methods("GET")

	realtime.HandleFunc("/ws/packages", s.handlePackageWebSocket).Methods("GET")

	realtime.HandleFunc("/ws/clusters", s.handleClusterWebSocket).Methods("GET")

	realtime.HandleFunc("/ws/metrics", s.handleMetricsWebSocket).Methods("GET")

	// Server-Sent Events (SSE) endpoints.

	realtime.HandleFunc("/events", s.handleServerSentEvents).Methods("GET")

	realtime.HandleFunc("/events/intents", s.handleIntentEvents).Methods("GET")

	realtime.HandleFunc("/events/packages", s.handlePackageEvents).Methods("GET")

	realtime.HandleFunc("/events/clusters", s.handleClusterEvents).Methods("GET")

	realtime.HandleFunc("/events/system", s.handleSystemEvents).Methods("GET")

	realtime.HandleFunc("/events/metrics", s.handleMetricsEvents).Methods("GET")

	// Stream management endpoints.

	realtime.HandleFunc("/streams", s.listActiveStreams).Methods("GET")

	realtime.HandleFunc("/streams/{id}", s.getStreamInfo).Methods("GET")

	realtime.HandleFunc("/streams/{id}/close", s.closeStream).Methods("DELETE")
}

// handleWebSocket handles the main WebSocket endpoint for all event types.

func (s *NephoranAPIServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocketConnection(w, r, "all")
}

// handleIntentWebSocket handles WebSocket connections for intent events only.

func (s *NephoranAPIServer) handleIntentWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocketConnection(w, r, "intents")
}

// handlePackageWebSocket handles WebSocket connections for package events only.

func (s *NephoranAPIServer) handlePackageWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocketConnection(w, r, "packages")
}

// handleClusterWebSocket handles WebSocket connections for cluster events only.

func (s *NephoranAPIServer) handleClusterWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocketConnection(w, r, "clusters")
}

// handleMetricsWebSocket handles WebSocket connections for metrics events only.

func (s *NephoranAPIServer) handleMetricsWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocketConnection(w, r, "metrics")
}

// handleWebSocketConnection handles the core WebSocket connection logic.

func (s *NephoranAPIServer) handleWebSocketConnection(w http.ResponseWriter, r *http.Request, eventType string) {
	// Check if we've reached connection limit.

	s.connectionsMutex.RLock()

	currentConnections := len(s.wsConnections)

	s.connectionsMutex.RUnlock()

	if currentConnections >= s.config.MaxWSConnections {

		s.writeErrorResponse(w, http.StatusTooManyRequests, "connection_limit_exceeded",

			"Maximum WebSocket connections reached")

		return

	}

	// Get user authentication context.

	userID := auth.GetUserID(r.Context())

	if userID == "" {
		userID = "anonymous"
	}

	// Upgrade connection to WebSocket.

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {

		s.logger.Error(err, "Failed to upgrade WebSocket connection")

		return

	}

	// Generate connection ID.

	connectionID := uuid.New().String()

	// Parse query parameters for initial filters.

	filters := s.parseStreamFilters(r)

	// Create WebSocket connection object.

	wsConn := &WebSocketConnection{
		ID: connectionID,

		UserID: userID,

		Connection: conn,

		Send: make(chan []byte, 256),

		Filters: json.RawMessage("{}"),

		LastSeen: time.Now(),
	}

	// Register the connection.

	s.connectionsMutex.Lock()

	s.wsConnections[connectionID] = wsConn

	s.connectionsMutex.Unlock()

	s.logger.Info("WebSocket connection established",

		"connection_id", connectionID,

		"user_id", userID,

		"event_type", eventType,

		"total_connections", len(s.wsConnections))

	// Send welcome message.

	welcomeMsg := WebSocketMessage{
		StreamMessage: StreamMessage{
			Type: "connection_established",

			ID: uuid.New().String(),

			Timestamp: time.Now(),

			Source: "nephoran-api-server",

			Data: json.RawMessage("{}"),
		},

		Action: "welcome",
	}

	select {

	case wsConn.Send <- mustMarshal(welcomeMsg):

	default:

		close(wsConn.Send)

		s.removeWebSocketConnection(connectionID)

		return

	}

	// Start goroutines for reading and writing.

	go s.webSocketReader(wsConn)

	go s.webSocketWriter(wsConn)
}

// webSocketReader handles incoming WebSocket messages.

func (s *NephoranAPIServer) webSocketReader(conn *WebSocketConnection) {
	defer func() {
		s.removeWebSocketConnection(conn.ID)

		conn.Connection.Close()
	}()

	// Set read deadline.

	conn.Connection.SetReadDeadline(time.Now().Add(s.config.WSTimeout))

	conn.Connection.SetPongHandler(func(string) error {
		conn.Connection.SetReadDeadline(time.Now().Add(s.config.WSTimeout))

		conn.LastSeen = time.Now()

		return nil
	})

	for {

		var msg WebSocketMessage

		err := conn.Connection.ReadJSON(&msg)
		if err != nil {

			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error(err, "WebSocket read error", "connection_id", conn.ID)
			}

			break

		}

		conn.LastSeen = time.Now()

		s.handleWebSocketMessage(conn, &msg)

	}
}

// webSocketWriter handles outgoing WebSocket messages.

func (s *NephoranAPIServer) webSocketWriter(conn *WebSocketConnection) {
	ticker := time.NewTicker(s.config.PingInterval)

	defer func() {
		ticker.Stop()

		conn.Connection.Close()
	}()

	for {
		select {

		case message, ok := <-conn.Send:

			conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {

				conn.Connection.WriteMessage(websocket.CloseMessage, []byte{})

				return

			}

			if err := conn.Connection.WriteMessage(websocket.TextMessage, message); err != nil {

				s.logger.Error(err, "WebSocket write error", "connection_id", conn.ID)

				return

			}

		case <-ticker.C:

			conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if err := conn.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		}
	}
}

// handleWebSocketMessage processes incoming WebSocket messages.

func (s *NephoranAPIServer) handleWebSocketMessage(conn *WebSocketConnection, msg *WebSocketMessage) {
	switch msg.Action {

	case "subscribe":

		s.handleWebSocketSubscribe(conn, msg)

	case "unsubscribe":

		s.handleWebSocketUnsubscribe(conn, msg)

	case "ping":

		s.handleWebSocketPing(conn, msg)

	case "get_status":

		s.handleWebSocketGetStatus(conn, msg)

	default:

		s.sendWebSocketError(conn, msg.RequestID, "unknown_action",

			fmt.Sprintf("Unknown action: %s", msg.Action))

	}
}

// handleWebSocketSubscribe handles subscription requests.

func (s *NephoranAPIServer) handleWebSocketSubscribe(conn *WebSocketConnection, msg *WebSocketMessage) {
	if msg.Filters != nil {

		// Update connection filters.

		conn.Filters["filters"] = msg.Filters

		s.logger.Info("WebSocket subscription updated",

			"connection_id", conn.ID,

			"filters", msg.Filters)

	}

	// Send subscription confirmation.

	response := WebSocketMessage{
		StreamMessage: StreamMessage{
			Type: "subscription_confirmed",

			ID: uuid.New().String(),

			Timestamp: time.Now(),

			Source: "nephoran-api-server",

			Data: json.RawMessage("{}"),
		},

		Action: "subscribe_response",

		RequestID: msg.RequestID,
	}

	select {

	case conn.Send <- mustMarshal(response):

	default:

		s.logger.V(1).Info("Failed to send subscription confirmation", "connection_id", conn.ID)

	}
}

// handleWebSocketUnsubscribe handles unsubscription requests.

func (s *NephoranAPIServer) handleWebSocketUnsubscribe(conn *WebSocketConnection, msg *WebSocketMessage) {
	// Reset filters to default.

	conn.Filters = json.RawMessage("{}")

	// Send unsubscription confirmation.

	response := WebSocketMessage{
		StreamMessage: StreamMessage{
			Type: "unsubscription_confirmed",

			ID: uuid.New().String(),

			Timestamp: time.Now(),

			Source: "nephoran-api-server",

			Data: json.RawMessage("{}"),
		},

		Action: "unsubscribe_response",

		RequestID: msg.RequestID,
	}

	select {

	case conn.Send <- mustMarshal(response):

	default:

		s.logger.V(1).Info("Failed to send unsubscription confirmation", "connection_id", conn.ID)

	}
}

// handleWebSocketPing handles ping requests.

func (s *NephoranAPIServer) handleWebSocketPing(conn *WebSocketConnection, msg *WebSocketMessage) {
	response := WebSocketMessage{
		StreamMessage: StreamMessage{
			Type: "pong",

			ID: uuid.New().String(),

			Timestamp: time.Now(),

			Source: "nephoran-api-server",
		},

		Action: "pong",

		RequestID: msg.RequestID,
	}

	select {

	case conn.Send <- mustMarshal(response):

	default:

		s.logger.V(1).Info("Failed to send pong response", "connection_id", conn.ID)

	}
}

// handleWebSocketGetStatus handles status requests.

func (s *NephoranAPIServer) handleWebSocketGetStatus(conn *WebSocketConnection, msg *WebSocketMessage) {
	s.connectionsMutex.RLock()

	totalConnections := len(s.wsConnections)

	sseConnections := len(s.sseConnections)

	s.connectionsMutex.RUnlock()

	response := WebSocketMessage{
		StreamMessage: StreamMessage{
			Type: "status",

			ID: uuid.New().String(),

			Timestamp: time.Now(),

			Source: "nephoran-api-server",

			Data: json.RawMessage("{}"),
		},

		Action: "status_response",

		RequestID: msg.RequestID,
	}

	select {

	case conn.Send <- mustMarshal(response):

	default:

		s.logger.V(1).Info("Failed to send status response", "connection_id", conn.ID)

	}
}

// sendWebSocketError sends an error message through WebSocket.

func (s *NephoranAPIServer) sendWebSocketError(conn *WebSocketConnection, requestID, code, message string) {
	errorMsg := WebSocketMessage{
		StreamMessage: StreamMessage{
			Type: "error",

			ID: uuid.New().String(),

			Timestamp: time.Now(),

			Source: "nephoran-api-server",

			Data: json.RawMessage("{}"),
		},

		Action: "error",

		RequestID: requestID,
	}

	select {

	case conn.Send <- mustMarshal(errorMsg):

	default:

		s.logger.V(1).Info("Failed to send error message", "connection_id", conn.ID)

	}
}

// Server-Sent Events handlers.

// handleServerSentEvents handles the main SSE endpoint for all event types.

func (s *NephoranAPIServer) handleServerSentEvents(w http.ResponseWriter, r *http.Request) {
	s.handleSSEConnection(w, r, "all")
}

// handleIntentEvents handles SSE connections for intent events only.

func (s *NephoranAPIServer) handleIntentEvents(w http.ResponseWriter, r *http.Request) {
	s.handleSSEConnection(w, r, "intents")
}

// handlePackageEvents handles SSE connections for package events only.

func (s *NephoranAPIServer) handlePackageEvents(w http.ResponseWriter, r *http.Request) {
	s.handleSSEConnection(w, r, "packages")
}

// handleClusterEvents handles SSE connections for cluster events only.

func (s *NephoranAPIServer) handleClusterEvents(w http.ResponseWriter, r *http.Request) {
	s.handleSSEConnection(w, r, "clusters")
}

// handleSystemEvents handles SSE connections for system events only.

func (s *NephoranAPIServer) handleSystemEvents(w http.ResponseWriter, r *http.Request) {
	s.handleSSEConnection(w, r, "system")
}

// handleMetricsEvents handles SSE connections for metrics events only.

func (s *NephoranAPIServer) handleMetricsEvents(w http.ResponseWriter, r *http.Request) {
	s.handleSSEConnection(w, r, "metrics")
}

// handleSSEConnection handles the core SSE connection logic.

func (s *NephoranAPIServer) handleSSEConnection(w http.ResponseWriter, r *http.Request, eventType string) {
	// Set SSE headers.

	w.Header().Set("Content-Type", "text/event-stream")

	w.Header().Set("Cache-Control", "no-cache")

	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get flusher.

	flusher, ok := w.(http.Flusher)

	if !ok {

		s.writeErrorResponse(w, http.StatusInternalServerError, "sse_not_supported",

			"Server-Sent Events not supported")

		return

	}

	// Get user authentication context.

	userID := auth.GetUserID(r.Context())

	if userID == "" {
		userID = "anonymous"
	}

	// Generate connection ID.

	connectionID := uuid.New().String()

	// Parse query parameters for initial filters.

	filters := s.parseStreamFilters(r)

	// Create SSE connection object.

	sseConn := &SSEConnection{
		ID: connectionID,

		UserID: userID,

		Writer: w,

		Flusher: flusher,

		Filters: json.RawMessage("{}"),

		LastSeen: time.Now(),
	}

	// Register the connection.

	s.connectionsMutex.Lock()

	s.sseConnections[connectionID] = sseConn

	s.connectionsMutex.Unlock()

	s.logger.Info("SSE connection established",

		"connection_id", connectionID,

		"user_id", userID,

		"event_type", eventType,

		"total_connections", len(s.sseConnections))

	// Send welcome message.

	welcomeMsg := StreamMessage{
		Type: "connection_established",

		ID: uuid.New().String(),

		Timestamp: time.Now(),

		Source: "nephoran-api-server",

		Data: json.RawMessage("{}"),
	}

	s.sendSSEMessage(sseConn, &welcomeMsg)

	// Handle client disconnect.

	notify := w.(http.CloseNotifier).CloseNotify()

	// Keep connection alive.

	ticker := time.NewTicker(30 * time.Second)

	defer func() {
		ticker.Stop()

		s.removeSSEConnection(connectionID)

		s.logger.Info("SSE connection closed", "connection_id", connectionID)
	}()

	for {
		select {

		case <-notify:

			return

		case <-r.Context().Done():

			return

		case <-ticker.C:

			// Send keep-alive ping.

			fmt.Fprintf(w, "event: ping\ndata: %s\n\n", mustMarshalString(json.RawMessage("{}")))

			flusher.Flush()

			sseConn.LastSeen = time.Now()

		}
	}
}

// sendSSEMessage sends a message through SSE connection.

func (s *NephoranAPIServer) sendSSEMessage(conn *SSEConnection, msg *StreamMessage) {
	if conn.Flusher == nil {
		return
	}

	fmt.Fprintf(conn.Writer, "event: %s\ndata: %s\n\n", msg.Type, mustMarshalString(msg))

	conn.Flusher.Flush()

	conn.LastSeen = time.Now()
}

// Stream management endpoints.

// listActiveStreams handles GET /api/v1/realtime/streams.

func (s *NephoranAPIServer) listActiveStreams(w http.ResponseWriter, r *http.Request) {
	// Check admin permissions.

	if s.authMiddleware != nil && !auth.HasPermission(r.Context(), auth.PermissionViewMetrics) {

		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",

			"View metrics permission required for stream management")

		return

	}

	s.connectionsMutex.RLock()

	defer s.connectionsMutex.RUnlock()

	streams := make([]map[string]interface{}, 0)

	// Add WebSocket connections.

	for id, conn := range s.wsConnections {
		streams = append(streams, json.RawMessage("{}"))
	}

	// Add SSE connections.

	for id, conn := range s.sseConnections {
		streams = append(streams, json.RawMessage("{}"))
	}

	s.writeJSONResponse(w, http.StatusOK, json.RawMessage("{}"))
}

// Helper functions.

func (s *NephoranAPIServer) parseStreamFilters(r *http.Request) *StreamFilter {
	filters := &StreamFilter{
		Labels: make(map[string]string),
	}

	query := r.URL.Query()

	// Parse event types.

	if eventTypes := query.Get("event_types"); eventTypes != "" {
		filters.EventTypes = strings.Split(eventTypes, ",")
	}

	// Parse status filters.

	if statuses := query.Get("status"); statuses != "" {
		filters.Status = strings.Split(statuses, ",")
	}

	// Parse priority filters.

	if priorities := query.Get("priority"); priorities != "" {
		filters.Priority = strings.Split(priorities, ",")
	}

	// Parse component filters.

	if components := query.Get("components"); components != "" {
		filters.Components = strings.Split(components, ",")
	}

	// Parse cluster filters.

	if clusters := query.Get("clusters"); clusters != "" {
		filters.Clusters = strings.Split(clusters, ",")
	}

	// Parse namespace filters.

	if namespaces := query.Get("namespaces"); namespaces != "" {
		filters.Namespaces = strings.Split(namespaces, ",")
	}

	// Parse since timestamp.

	if since := query.Get("since"); since != "" {
		if timestamp, err := time.Parse(time.RFC3339, since); err == nil {
			filters.Since = &timestamp
		}
	}

	// Parse label filters (format: label.key=value).

	for key, values := range query {
		if strings.HasPrefix(key, "label.") {

			labelKey := strings.TrimPrefix(key, "label.")

			if len(values) > 0 {
				filters.Labels[labelKey] = values[0]
			}

		}
	}

	return filters
}

func (s *NephoranAPIServer) removeWebSocketConnection(connectionID string) {
	s.connectionsMutex.Lock()

	defer s.connectionsMutex.Unlock()

	if conn, exists := s.wsConnections[connectionID]; exists {

		close(conn.Send)

		delete(s.wsConnections, connectionID)

		s.logger.Info("WebSocket connection removed", "connection_id", connectionID)

	}
}

func (s *NephoranAPIServer) removeSSEConnection(connectionID string) {
	s.connectionsMutex.Lock()

	defer s.connectionsMutex.Unlock()

	if _, exists := s.sseConnections[connectionID]; exists {

		delete(s.sseConnections, connectionID)

		s.logger.Info("SSE connection removed", "connection_id", connectionID)

	}
}

// Broadcast event to all appropriate connections based on filters.

func (s *NephoranAPIServer) broadcastEvent(eventType string, data interface{}) {
	message := &StreamMessage{
		Type: eventType,

		ID: uuid.New().String(),

		Timestamp: time.Now(),

		Source: "nephoran-api-server",

		Data: data,
	}

	// Broadcast to WebSocket connections.

	s.connectionsMutex.RLock()

	for _, conn := range s.wsConnections {
		if s.shouldSendToConnection(conn.Filters, eventType, data) {
			select {

			case conn.Send <- mustMarshal(message):

			default:

				close(conn.Send)

			}
		}
	}

	// Broadcast to SSE connections.

	for _, conn := range s.sseConnections {
		if s.shouldSendToConnection(conn.Filters, eventType, data) {
			s.sendSSEMessage(conn, message)
		}
	}

	s.connectionsMutex.RUnlock()
}

func (s *NephoranAPIServer) shouldSendToConnection(filters map[string]interface{}, eventType string, data interface{}) bool {
	// Basic event type filtering.

	if connEventType, exists := filters["event_type"]; exists {
		if eventTypeStr, ok := connEventType.(string); ok {
			if eventTypeStr != "all" && !strings.Contains(eventType, eventTypeStr) {
				return false
			}
		}
	}

	// Additional filter logic would be implemented here based on the actual data structure.

	return true
}

// getStreamInfo handles GET /api/v1/realtime/streams/{id}.

func (s *NephoranAPIServer) getStreamInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	streamID := vars["id"]

	// TODO: Implement actual stream info retrieval.

	streamInfo := json.RawMessage("{}")

	s.writeJSONResponse(w, http.StatusOK, streamInfo)
}

// closeStream handles DELETE /api/v1/realtime/streams/{id}/close.

func (s *NephoranAPIServer) closeStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	streamID := vars["id"]

	// TODO: Implement actual stream closing logic.

	s.logger.Info("Closing stream", "streamID", streamID)

	response := json.RawMessage("{}")

	s.writeJSONResponse(w, http.StatusOK, response)
}
