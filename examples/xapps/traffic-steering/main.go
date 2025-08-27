package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2/service_models"
)

// TrafficSteeringXApp demonstrates intelligent traffic steering using RC service model
type TrafficSteeringXApp struct {
	sdk           *e2.XAppSDK
	rcModel       *service_models.RCServiceModel
	kmpModel      *service_models.KPMServiceModel
	cellLoadInfo  map[string]*CellLoadInfo
	steeringRules *SteeringRules
	httpServer    *http.Server
}

// CellLoadInfo stores load information for traffic steering decisions
type CellLoadInfo struct {
	CellID            string             `json:"cell_id"`
	LastUpdate        time.Time          `json:"last_update"`
	PRBUtilizationDL  float64            `json:"prb_utilization_dl"`
	PRBUtilizationUL  float64            `json:"prb_utilization_ul"`
	ActiveUEs         int                `json:"active_ues"`
	ThroughputDL      float64            `json:"throughput_dl_mbps"`
	ThroughputUL      float64            `json:"throughput_ul_mbps"`
	ConnectionSuccess float64            `json:"connection_success_rate"`
	LoadLevel         LoadLevel          `json:"load_level"`
	NeighborCells     []string           `json:"neighbor_cells"`
	ConnectedUEs      map[string]*UEInfo `json:"connected_ues"`
}

// UEInfo stores information about connected UEs
type UEInfo struct {
	UEID         string    `json:"ue_id"`
	RSRP         float64   `json:"rsrp_dbm"`
	RSRQ         float64   `json:"rsrq_db"`
	CQI          int       `json:"cqi"`
	ThroughputDL float64   `json:"throughput_dl_kbps"`
	ThroughputUL float64   `json:"throughput_ul_kbps"`
	Priority     int       `json:"priority"`
	LastUpdate   time.Time `json:"last_update"`
}

// LoadLevel represents the load level of a cell
type LoadLevel string

const (
	LoadLevelLow      LoadLevel = "LOW"      // < 30%
	LoadLevelModerate LoadLevel = "MODERATE" // 30-70%
	LoadLevelHigh     LoadLevel = "HIGH"     // 70-85%
	LoadLevelCritical LoadLevel = "CRITICAL" // > 85%
)

// SteeringRules defines traffic steering thresholds and policies
type SteeringRules struct {
	HighLoadThreshold     float64       `json:"high_load_threshold"`     // 75%
	CriticalLoadThreshold float64       `json:"critical_load_threshold"` // 85%
	LowLoadThreshold      float64       `json:"low_load_threshold"`      // 30%
	MinRSRPThreshold      float64       `json:"min_rsrp_threshold"`      // -110 dBm
	MaxSteerUEs           int           `json:"max_steer_ues"`           // 5 UEs per cycle
	SteeringInterval      time.Duration `json:"steering_interval"`       // 30 seconds
	HysteresisMargin      float64       `json:"hysteresis_margin"`       // 5%
}

// SteeringDecision represents a traffic steering decision
type SteeringDecision struct {
	UEID           string    `json:"ue_id"`
	SourceCellID   string    `json:"source_cell_id"`
	TargetCellID   string    `json:"target_cell_id"`
	Reason         string    `json:"reason"`
	LoadDifference float64   `json:"load_difference"`
	Timestamp      time.Time `json:"timestamp"`
	Status         string    `json:"status"`
}

func main() {
	// Configuration from environment variables
	xappConfig := &e2.XAppConfig{
		XAppName:        config.GetEnvOrDefault("XAPP_NAME", "traffic-steering-xapp"),
		XAppVersion:     config.GetEnvOrDefault("XAPP_VERSION", "1.0.0"),
		XAppDescription: "Intelligent Traffic Steering xApp",
		E2NodeID:        config.GetEnvOrDefault("E2_NODE_ID", "gnb-001"),
		NearRTRICURL:    config.GetEnvOrDefault("NEAR_RT_RIC_URL", "http://near-rt-ric:8080"),
		ServiceModels:   []string{"KPM", "RC"},
		ResourceLimits: &e2.XAppResourceLimits{
			MaxMemoryMB:      2048,
			MaxCPUCores:      2.0,
			MaxSubscriptions: 20,
			RequestTimeout:   30 * time.Second,
		},
		HealthCheck: &e2.XAppHealthConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			FailureThreshold: 3,
			HealthEndpoint:   "/health",
		},
	}

	// Create xApp instance
	xapp, err := NewTrafficSteeringXApp(xappConfig)
	if err != nil {
		log.Fatalf("Failed to create xApp: %v", err)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start xApp
	if err := xapp.Start(ctx); err != nil {
		log.Fatalf("Failed to start xApp: %v", err)
	}

	log.Printf("Traffic Steering xApp started successfully")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping xApp...")

	// Graceful shutdown
	if err := xapp.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

// NewTrafficSteeringXApp creates a new traffic steering xApp instance
func NewTrafficSteeringXApp(config *e2.XAppConfig) (*TrafficSteeringXApp, error) {
	// Create E2Manager
	e2Manager, err := e2.NewE2Manager(&e2.E2ManagerConfig{
		DefaultRICURL:     config.NearRTRICURL,
		DefaultAPIVersion: "v1",
		DefaultTimeout:    30 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		MaxRetries:        3,
		SimulationMode:    false,
		SimulateRICCalls:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create E2Manager: %w", err)
	}

	// Create xApp SDK
	sdk, err := e2.NewXAppSDK(config, e2Manager)
	if err != nil {
		return nil, fmt.Errorf("failed to create xApp SDK: %w", err)
	}

	xapp := &TrafficSteeringXApp{
		sdk:          sdk,
		rcModel:      service_models.NewRCServiceModel(),
		kmpModel:     service_models.NewKPMServiceModel(),
		cellLoadInfo: make(map[string]*CellLoadInfo),
		steeringRules: &SteeringRules{
			HighLoadThreshold:     75.0,
			CriticalLoadThreshold: 85.0,
			LowLoadThreshold:      30.0,
			MinRSRPThreshold:      -110.0,
			MaxSteerUEs:           5,
			SteeringInterval:      30 * time.Second,
			HysteresisMargin:      5.0,
		},
	}

	// Register indication handler
	sdk.RegisterIndicationHandler("default", xapp.handleLoadIndication)

	// Setup HTTP server
	xapp.setupHTTPServer()

	return xapp, nil
}

// Start initializes and starts the xApp
func (x *TrafficSteeringXApp) Start(ctx context.Context) error {
	// Start xApp SDK
	if err := x.sdk.Start(ctx); err != nil {
		return fmt.Errorf("failed to start SDK: %w", err)
	}

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on :8080")
		if err := x.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Initialize cell topology and create subscriptions
	if err := x.initializeCellTopology(ctx); err != nil {
		return fmt.Errorf("failed to initialize cell topology: %w", err)
	}

	// Start traffic steering loop
	go x.trafficSteeringLoop(ctx)

	return nil
}

// Stop gracefully shuts down the xApp
func (x *TrafficSteeringXApp) Stop(ctx context.Context) error {
	// Stop HTTP server
	if x.httpServer != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := x.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop xApp SDK
	return x.sdk.Stop(ctx)
}

// initializeCellTopology sets up cell information and neighbor relationships
func (x *TrafficSteeringXApp) initializeCellTopology(ctx context.Context) error {
	// Define cell topology (in real deployment, this would come from configuration or O1 interface)
	cellTopology := map[string][]string{
		"cell-001": {"cell-002", "cell-003"},
		"cell-002": {"cell-001", "cell-003", "cell-004"},
		"cell-003": {"cell-001", "cell-002", "cell-004"},
		"cell-004": {"cell-002", "cell-003"},
	}

	// Initialize cell load info
	for cellID, neighbors := range cellTopology {
		x.cellLoadInfo[cellID] = &CellLoadInfo{
			CellID:        cellID,
			LastUpdate:    time.Now(),
			LoadLevel:     LoadLevelModerate,
			NeighborCells: neighbors,
			ConnectedUEs:  make(map[string]*UEInfo),
		}
	}

	// Create KMP subscriptions for load monitoring
	for cellID := range cellTopology {
		measurements := []string{
			"RRC.ConnEstabAtt",
			"RRC.ConnEstabSucc",
			"RRU.PrbUsedDl",
			"RRU.PrbUsedUl",
			"RRU.PrbTotDl",
			"RRU.PrbTotUl",
			"DRB.PdcpSduVolumeDL",
			"DRB.PdcpSduVolumeUL",
		}

		subscription, err := x.kmpModel.CreateKPMSubscription(x.sdk.GetConfig().E2NodeID, cellID, measurements)
		if err != nil {
			return fmt.Errorf("failed to create KMP subscription for cell %s: %w", cellID, err)
		}

		_, err = x.sdk.Subscribe(ctx, &e2.E2SubscriptionRequest{
			SubscriptionID:  subscription.SubscriptionID,
			RequestorID:     x.sdk.GetConfig().XAppName,
			NodeID:          x.sdk.GetConfig().E2NodeID,
			RanFunctionID:   subscription.RanFunctionID,
			EventTriggers:   subscription.EventTriggers,
			Actions:         subscription.Actions,
			ReportingPeriod: subscription.ReportingPeriod,
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe for cell %s: %w", cellID, err)
		}

		log.Printf("Created load monitoring subscription for cell %s", cellID)
	}

	return nil
}

// handleLoadIndication processes incoming load indication messages
func (x *TrafficSteeringXApp) handleLoadIndication(ctx context.Context, indication *e2.RICIndication) error {
	// Parse indication using KMP service model
	report, err := x.kmpModel.ParseIndication(
		indication.RICIndicationHeader,
		indication.RICIndicationMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to parse KMP indication: %w", err)
	}

	kmpReport, ok := report.(*service_models.KPMReport)
	if !ok {
		return fmt.Errorf("unexpected report type")
	}

	// Update cell load information
	x.updateCellLoadInfo(kmpReport)

	return nil
}

// updateCellLoadInfo updates the load information for a cell
func (x *TrafficSteeringXApp) updateCellLoadInfo(report *service_models.KPMReport) {
	cellInfo, exists := x.cellLoadInfo[report.CellID]
	if !exists {
		log.Printf("Received report for unknown cell: %s", report.CellID)
		return
	}

	cellInfo.LastUpdate = report.Timestamp
	cellInfo.ActiveUEs = report.UECount

	// Calculate PRB utilization
	if prbUsedDl, ok := report.Measurements["RRU.PrbUsedDl"]; ok {
		if prbTotDl, ok := report.Measurements["RRU.PrbTotDl"]; ok && prbTotDl > 0 {
			cellInfo.PRBUtilizationDL = (prbUsedDl / prbTotDl) * 100
		}
	}

	if prbUsedUl, ok := report.Measurements["RRU.PrbUsedUl"]; ok {
		if prbTotUl, ok := report.Measurements["RRU.PrbTotUl"]; ok && prbTotUl > 0 {
			cellInfo.PRBUtilizationUL = (prbUsedUl / prbTotUl) * 100
		}
	}

	// Calculate throughput
	if volDl, ok := report.Measurements["DRB.PdcpSduVolumeDL"]; ok {
		cellInfo.ThroughputDL = (volDl * 8) / (1024 * 1024) // Convert to Mbps
	}

	if volUl, ok := report.Measurements["DRB.PdcpSduVolumeUL"]; ok {
		cellInfo.ThroughputUL = (volUl * 8) / (1024 * 1024) // Convert to Mbps
	}

	// Calculate connection success rate
	if connAtt, ok := report.Measurements["RRC.ConnEstabAtt"]; ok && connAtt > 0 {
		if connSucc, ok := report.Measurements["RRC.ConnEstabSucc"]; ok {
			cellInfo.ConnectionSuccess = (connSucc / connAtt) * 100
		}
	}

	// Determine load level based on PRB utilization
	avgPRBUtil := (cellInfo.PRBUtilizationDL + cellInfo.PRBUtilizationUL) / 2
	if avgPRBUtil >= x.steeringRules.CriticalLoadThreshold {
		cellInfo.LoadLevel = LoadLevelCritical
	} else if avgPRBUtil >= x.steeringRules.HighLoadThreshold {
		cellInfo.LoadLevel = LoadLevelHigh
	} else if avgPRBUtil <= x.steeringRules.LowLoadThreshold {
		cellInfo.LoadLevel = LoadLevelLow
	} else {
		cellInfo.LoadLevel = LoadLevelModerate
	}

	log.Printf("Updated load info for cell %s: PRB DL=%.1f%%, UL=%.1f%%, Level=%s",
		report.CellID, cellInfo.PRBUtilizationDL, cellInfo.PRBUtilizationUL, cellInfo.LoadLevel)
}

// trafficSteeringLoop runs the main traffic steering algorithm
func (x *TrafficSteeringXApp) trafficSteeringLoop(ctx context.Context) {
	ticker := time.NewTicker(x.steeringRules.SteeringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			x.evaluateAndSteer(ctx)
		}
	}
}

// evaluateAndSteer evaluates current network state and performs traffic steering if needed
func (x *TrafficSteeringXApp) evaluateAndSteer(ctx context.Context) {
	// Find overloaded and underloaded cells
	overloadedCells := x.findCellsByLoadLevel(LoadLevelHigh, LoadLevelCritical)
	underloadedCells := x.findCellsByLoadLevel(LoadLevelLow, LoadLevelModerate)

	if len(overloadedCells) == 0 {
		log.Printf("No overloaded cells found, no steering needed")
		return
	}

	if len(underloadedCells) == 0 {
		log.Printf("No underloaded cells available for steering")
		return
	}

	log.Printf("Found %d overloaded cells and %d underloaded cells",
		len(overloadedCells), len(underloadedCells))

	// Sort cells by load for optimal steering decisions
	sort.Slice(overloadedCells, func(i, j int) bool {
		return (overloadedCells[i].PRBUtilizationDL + overloadedCells[i].PRBUtilizationUL) >
			(overloadedCells[j].PRBUtilizationDL + overloadedCells[j].PRBUtilizationUL)
	})

	sort.Slice(underloadedCells, func(i, j int) bool {
		return (underloadedCells[i].PRBUtilizationDL + underloadedCells[i].PRBUtilizationUL) <
			(underloadedCells[j].PRBUtilizationDL + underloadedCells[j].PRBUtilizationUL)
	})

	// Perform traffic steering
	steeringDecisions := make([]*SteeringDecision, 0)

	for _, overloadedCell := range overloadedCells {
		// Find best target cell for this overloaded cell
		targetCell := x.findBestTargetCell(overloadedCell, underloadedCells)
		if targetCell == nil {
			continue
		}

		// Select UEs to steer
		candidateUEs := x.selectUEsForSteering(overloadedCell, targetCell)

		// Limit number of UEs to steer per cycle
		maxSteer := x.steeringRules.MaxSteerUEs
		if len(candidateUEs) > maxSteer {
			candidateUEs = candidateUEs[:maxSteer]
		}

		// Create steering decisions and execute
		for _, ueInfo := range candidateUEs {
			decision := &SteeringDecision{
				UEID:         ueInfo.UEID,
				SourceCellID: overloadedCell.CellID,
				TargetCellID: targetCell.CellID,
				Reason: fmt.Sprintf("Load balancing: source=%.1f%%, target=%.1f%%",
					(overloadedCell.PRBUtilizationDL+overloadedCell.PRBUtilizationUL)/2,
					(targetCell.PRBUtilizationDL+targetCell.PRBUtilizationUL)/2),
				LoadDifference: math.Abs((overloadedCell.PRBUtilizationDL+overloadedCell.PRBUtilizationUL)-
					(targetCell.PRBUtilizationDL+targetCell.PRBUtilizationUL)) / 2,
				Timestamp: time.Now(),
				Status:    "PENDING",
			}

			// Execute traffic steering
			if err := x.executeTrafficSteering(ctx, decision); err != nil {
				log.Printf("Failed to steer UE %s: %v", ueInfo.UEID, err)
				decision.Status = "FAILED"
			} else {
				decision.Status = "SUCCESS"
				log.Printf("Successfully steered UE %s from cell %s to cell %s",
					ueInfo.UEID, overloadedCell.CellID, targetCell.CellID)
			}

			steeringDecisions = append(steeringDecisions, decision)
		}
	}

	if len(steeringDecisions) > 0 {
		log.Printf("Completed traffic steering cycle: %d decisions made", len(steeringDecisions))
	}
}

// findCellsByLoadLevel returns cells matching the specified load levels
func (x *TrafficSteeringXApp) findCellsByLoadLevel(levels ...LoadLevel) []*CellLoadInfo {
	var cells []*CellLoadInfo

	for _, cellInfo := range x.cellLoadInfo {
		for _, level := range levels {
			if cellInfo.LoadLevel == level {
				cells = append(cells, cellInfo)
				break
			}
		}
	}

	return cells
}

// findBestTargetCell finds the best target cell for steering from a source cell
func (x *TrafficSteeringXApp) findBestTargetCell(sourceCell *CellLoadInfo, candidateCells []*CellLoadInfo) *CellLoadInfo {
	var bestTarget *CellLoadInfo
	bestScore := -1.0

	sourceLoad := (sourceCell.PRBUtilizationDL + sourceCell.PRBUtilizationUL) / 2

	for _, candidate := range candidateCells {
		// Check if candidate is a neighbor of source cell
		isNeighbor := false
		for _, neighborID := range sourceCell.NeighborCells {
			if neighborID == candidate.CellID {
				isNeighbor = true
				break
			}
		}

		if !isNeighbor {
			continue // Only consider neighbor cells
		}

		candidateLoad := (candidate.PRBUtilizationDL + candidate.PRBUtilizationUL) / 2

		// Calculate load difference (higher is better for steering)
		loadDifference := sourceLoad - candidateLoad

		// Require minimum load difference to avoid ping-pong effects
		if loadDifference < x.steeringRules.HysteresisMargin {
			continue
		}

		// Score based on load difference and connection success rate
		score := loadDifference * (candidate.ConnectionSuccess / 100.0)

		if score > bestScore {
			bestScore = score
			bestTarget = candidate
		}
	}

	return bestTarget
}

// selectUEsForSteering selects UEs that are good candidates for traffic steering
func (x *TrafficSteeringXApp) selectUEsForSteering(sourceCell, targetCell *CellLoadInfo) []*UEInfo {
	var candidates []*UEInfo

	// In a real implementation, this would query actual UE information
	// For demo purposes, we simulate some UEs
	for i := 0; i < sourceCell.ActiveUEs && i < 10; i++ {
		ue := &UEInfo{
			UEID:         fmt.Sprintf("ue-%s-%d", sourceCell.CellID, i+1),
			RSRP:         -80.0 + float64(i)*2, // Simulate varying signal quality
			RSRQ:         -10.0 + float64(i)*0.5,
			CQI:          10 + i%5,
			ThroughputDL: 1000.0 + float64(i)*100,
			ThroughputUL: 500.0 + float64(i)*50,
			Priority:     1, // Normal priority
			LastUpdate:   time.Now(),
		}

		// Select UEs with good signal quality (can be steered without quality degradation)
		if ue.RSRP > x.steeringRules.MinRSRPThreshold && ue.CQI >= 7 {
			candidates = append(candidates, ue)
		}
	}

	// Sort by priority and signal quality (steer lower priority UEs first)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority < candidates[j].Priority
		}
		return candidates[i].RSRP > candidates[j].RSRP
	})

	return candidates
}

// executeTrafficSteering sends RC control message to steer traffic
func (x *TrafficSteeringXApp) executeTrafficSteering(ctx context.Context, decision *SteeringDecision) error {
	// Create traffic steering control request using RC service model
	e2ControlReq, err := x.rcModel.CreateTrafficSteeringControl(decision.UEID, decision.TargetCellID)
	if err != nil {
		return fmt.Errorf("failed to create control request: %w", err)
	}

	// Convert E2ControlRequest to RICControlRequest for SDK
	ricControlReq, err := x.convertE2ToRICControlRequest(e2ControlReq)
	if err != nil {
		return fmt.Errorf("failed to convert control request: %w", err)
	}

	// Send control message through SDK to the configured E2 node
	ack, err := x.sdk.SendControlMessage(ctx, x.sdk.GetConfig().E2NodeID, ricControlReq)
	if err != nil {
		return fmt.Errorf("failed to send control message: %w", err)
	}

	log.Printf("Traffic steering control sent for UE %s, received ack: %+v",
		decision.UEID, ack)

	return nil
}

// setupHTTPServer configures the HTTP server for monitoring and control
func (x *TrafficSteeringXApp) setupHTTPServer() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", x.handleHealth)

	// Cell load information endpoints
	mux.HandleFunc("/cells", x.handleCells)
	mux.HandleFunc("/cells/", x.handleCell)

	// Steering rules endpoints
	mux.HandleFunc("/rules", x.handleRules)

	// xApp info endpoint
	mux.HandleFunc("/info", x.handleInfo)

	x.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}

// handleHealth provides health check information
func (x *TrafficSteeringXApp) handleHealth(w http.ResponseWriter, r *http.Request) {
	state := x.sdk.GetState()
	metrics := x.sdk.GetMetrics()

	health := map[string]interface{}{
		"status":    string(state),
		"timestamp": time.Now().UTC(),
		"cells":     len(x.cellLoadInfo),
		"sdk_metrics": map[string]interface{}{
			"subscriptions_active":  metrics.SubscriptionsActive,
			"indications_received":  metrics.IndicationsReceived,
			"control_requests_sent": metrics.ControlRequestsSent,
			"error_count":           metrics.ErrorCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if state != e2.XAppStateRunning {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}

// handleCells returns load information for all cells
func (x *TrafficSteeringXApp) handleCells(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(x.cellLoadInfo)
}

// handleCell returns load information for a specific cell
func (x *TrafficSteeringXApp) handleCell(w http.ResponseWriter, r *http.Request) {
	cellID := r.URL.Path[len("/cells/"):]
	if cellID == "" {
		http.Error(w, "Cell ID is required", http.StatusBadRequest)
		return
	}

	cellInfo, exists := x.cellLoadInfo[cellID]
	if !exists {
		http.Error(w, "Cell not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cellInfo)
}

// handleRules returns current steering rules
func (x *TrafficSteeringXApp) handleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(x.steeringRules)
}

// handleInfo returns xApp information
func (x *TrafficSteeringXApp) handleInfo(w http.ResponseWriter, r *http.Request) {
	config := x.sdk.GetConfig()
	info := map[string]interface{}{
		"name":           config.XAppName,
		"version":        config.XAppVersion,
		"description":    config.XAppDescription,
		"e2_node_id":     config.E2NodeID,
		"ric_url":        config.NearRTRICURL,
		"service_models": config.ServiceModels,
		"state":          string(x.sdk.GetState()),
		"cells":          len(x.cellLoadInfo),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// convertE2ToRICControlRequest converts E2ControlRequest to RICControlRequest
func (x *TrafficSteeringXApp) convertE2ToRICControlRequest(e2Req *e2.E2ControlRequest) (*e2.RICControlRequest, error) {
	// Extract header bytes
	var headerBytes []byte
	if headerData, ok := e2Req.ControlHeader["data"]; ok {
		if bytes, ok := headerData.([]byte); ok {
			headerBytes = bytes
		} else {
			return nil, fmt.Errorf("control header data is not []byte")
		}
	}

	// Extract message bytes
	var messageBytes []byte
	if messageData, ok := e2Req.ControlMessage["data"]; ok {
		if bytes, ok := messageData.([]byte); ok {
			messageBytes = bytes
		} else {
			return nil, fmt.Errorf("control message data is not []byte")
		}
	}

	// Parse request ID to get requestor and instance IDs
	requestorID := uint32(1000) // Default for xApp
	instanceID := uint32(1)     // Default instance

	// Convert call process ID
	var callProcessID *e2.RICCallProcessID
	if e2Req.CallProcessID != "" {
		cpID := e2.RICCallProcessID(e2Req.CallProcessID)
		callProcessID = &cpID
	}

	// Convert ACK request
	var ackRequest *e2.RICControlAckRequest
	if e2Req.ControlAckRequest {
		ack := e2.RICControlAckRequest(1) // TRUE
		ackRequest = &ack
	}

	ricReq := &e2.RICControlRequest{
		RICRequestID: e2.RICRequestID{
			RICRequestorID: e2.RICRequestorID(requestorID),
			RICInstanceID:  e2.RICInstanceID(instanceID),
		},
		RANFunctionID:        e2.RANFunctionID(e2Req.RanFunctionID),
		RICCallProcessID:     callProcessID,
		RICControlHeader:     headerBytes,
		RICControlMessage:    messageBytes,
		RICControlAckRequest: ackRequest,
	}

	return ricReq, nil
}

// Note: Helper functions have been moved to pkg/config/env_helpers.go
