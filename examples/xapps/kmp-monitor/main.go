
package main



import (

	"context"

	"encoding/json"

	"fmt"

	"log"

	"net/http"

	"os"

	"os/signal"

	"syscall"

	"time"



	"github.com/nephio-project/nephoran-intent-operator/pkg/config"

	"github.com/nephio-project/nephoran-intent-operator/pkg/oran/e2"

	"github.com/nephio-project/nephoran-intent-operator/pkg/oran/e2/service_models"

)



// KMPMonitorXApp demonstrates a complete KMP monitoring xApp.

type KMPMonitorXApp struct {

	sdk        *e2.XAppSDK

	kmpModel   *service_models.KPMServiceModel

	metrics    map[string]*CellMetrics

	httpServer *http.Server

}



// CellMetrics stores aggregated metrics for a cell.

type CellMetrics struct {

	CellID            string             `json:"cell_id"`

	LastUpdate        time.Time          `json:"last_update"`

	ActiveUEs         int                `json:"active_ues"`

	ConnectionSuccess float64            `json:"connection_success_rate"`

	PRBUtilizationDL  float64            `json:"prb_utilization_dl"`

	PRBUtilizationUL  float64            `json:"prb_utilization_ul"`

	ThroughputDL      float64            `json:"throughput_dl_mbps"`

	ThroughputUL      float64            `json:"throughput_ul_mbps"`

	Measurements      map[string]float64 `json:"raw_measurements"`

}



func main() {

	// Configuration from environment variables.

	xappConfig := &e2.XAppConfig{

		XAppName:        config.GetEnvOrDefault("XAPP_NAME", "kmp-monitor-xapp"),

		XAppVersion:     config.GetEnvOrDefault("XAPP_VERSION", "1.0.0"),

		XAppDescription: "KMP Monitoring and Analytics xApp",

		E2NodeID:        config.GetEnvOrDefault("E2_NODE_ID", "gnb-001"),

		NearRTRICURL:    config.GetEnvOrDefault("NEAR_RT_RIC_URL", "http://near-rt-ric:8080"),

		ServiceModels:   []string{"KPM"},

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



	// Create xApp instance.

	xapp, err := NewKMPMonitorXApp(xappConfig)

	if err != nil {

		log.Fatalf("Failed to create xApp: %v", err)

	}



	// Setup signal handling for graceful shutdown.

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()



	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)



	// Start xApp.

	if err := xapp.Start(ctx); err != nil {

		log.Fatalf("Failed to start xApp: %v", err)

	}



	log.Printf("KMP Monitor xApp started successfully")



	// Wait for shutdown signal.

	<-sigChan

	log.Println("Shutdown signal received, stopping xApp...")



	// Graceful shutdown.

	if err := xapp.Stop(ctx); err != nil {

		log.Printf("Error during shutdown: %v", err)

	}

}



// NewKMPMonitorXApp creates a new KMP monitoring xApp instance.

func NewKMPMonitorXApp(config *e2.XAppConfig) (*KMPMonitorXApp, error) {

	// Create E2Manager.

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



	// Create xApp SDK.

	sdk, err := e2.NewXAppSDK(config, e2Manager)

	if err != nil {

		return nil, fmt.Errorf("failed to create xApp SDK: %w", err)

	}



	xapp := &KMPMonitorXApp{

		sdk:      sdk,

		kmpModel: service_models.NewKPMServiceModel(),

		metrics:  make(map[string]*CellMetrics),

	}



	// Register indication handler.

	sdk.RegisterIndicationHandler("default", xapp.handleKMPIndication)



	// Setup HTTP server for metrics API.

	xapp.setupHTTPServer()



	return xapp, nil

}



// Start initializes and starts the xApp.

func (x *KMPMonitorXApp) Start(ctx context.Context) error {

	// Start xApp SDK.

	if err := x.sdk.Start(ctx); err != nil {

		return fmt.Errorf("failed to start SDK: %w", err)

	}



	// Start HTTP server.

	go func() {

		log.Printf("Starting HTTP server on :8080")

		if err := x.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {

			log.Printf("HTTP server error: %v", err)

		}

	}()



	// Create KMP subscriptions for multiple cells.

	cells := []string{"cell-001", "cell-002", "cell-003"}

	if err := x.createKMPSubscriptions(ctx, cells); err != nil {

		return fmt.Errorf("failed to create subscriptions: %w", err)

	}



	return nil

}



// Stop gracefully shuts down the xApp.

func (x *KMPMonitorXApp) Stop(ctx context.Context) error {

	// Stop HTTP server.

	if x.httpServer != nil {

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

		defer cancel()

		if err := x.httpServer.Shutdown(ctx); err != nil {

			log.Printf("HTTP server shutdown error: %v", err)

		}

	}



	// Stop xApp SDK.

	return x.sdk.Stop(ctx)

}



// createKMPSubscriptions creates KMP subscriptions for specified cells.

func (x *KMPMonitorXApp) createKMPSubscriptions(ctx context.Context, cells []string) error {

	measurements := []string{

		"RRC.ConnEstabAtt",

		"RRC.ConnEstabSucc",

		"RRC.ConnMean",

		"RRC.ConnMax",

		"DRB.PdcpSduVolumeDL",

		"DRB.PdcpSduVolumeUL",

		"DRB.UEThpDl",

		"DRB.UEThpUl",

		"RRU.PrbTotDl",

		"RRU.PrbTotUl",

		"RRU.PrbUsedDl",

		"RRU.PrbUsedUl",

	}



	for _, cellID := range cells {

		// Create subscription using KMP service model.

		subscription, err := x.kmpModel.CreateKPMSubscription(x.sdk.GetConfig().E2NodeID, cellID, measurements)

		if err != nil {

			return fmt.Errorf("failed to create KMP subscription for cell %s: %w", cellID, err)

		}



		// Subscribe through SDK.

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



		log.Printf("Created KMP subscription for cell %s", cellID)



		// Initialize metrics for this cell.

		x.metrics[cellID] = &CellMetrics{

			CellID:       cellID,

			LastUpdate:   time.Now(),

			Measurements: make(map[string]float64),

		}

	}



	return nil

}



// handleKMPIndication processes incoming KMP indication messages.

func (x *KMPMonitorXApp) handleKMPIndication(ctx context.Context, indication *e2.RICIndication) error {

	// Parse indication using KMP service model.

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



	// Update metrics.

	x.updateCellMetrics(kmpReport)



	// Log metrics (for demo purposes).

	log.Printf("KMP Report - Cell: %s, UEs: %d, Measurements: %d",

		kmpReport.CellID, kmpReport.UECount, len(kmpReport.Measurements))



	return nil

}



// updateCellMetrics updates the stored metrics for a cell.

func (x *KMPMonitorXApp) updateCellMetrics(report *service_models.KPMReport) {

	cellMetrics, exists := x.metrics[report.CellID]

	if !exists {

		cellMetrics = &CellMetrics{

			CellID:       report.CellID,

			Measurements: make(map[string]float64),

		}

		x.metrics[report.CellID] = cellMetrics

	}



	// Update basic info.

	cellMetrics.LastUpdate = report.Timestamp

	cellMetrics.ActiveUEs = report.UECount



	// Store raw measurements.

	for name, value := range report.Measurements {

		cellMetrics.Measurements[name] = value

	}



	// Calculate derived metrics.

	x.calculateDerivedMetrics(cellMetrics)

}



// calculateDerivedMetrics computes useful derived metrics from raw measurements.

func (x *KMPMonitorXApp) calculateDerivedMetrics(metrics *CellMetrics) {

	// Connection success rate.

	if connAtt, ok := metrics.Measurements["RRC.ConnEstabAtt"]; ok && connAtt > 0 {

		if connSucc, ok := metrics.Measurements["RRC.ConnEstabSucc"]; ok {

			metrics.ConnectionSuccess = (connSucc / connAtt) * 100

		}

	}



	// PRB utilization.

	if prbUsedDl, ok := metrics.Measurements["RRU.PrbUsedDl"]; ok {

		if prbTotDl, ok := metrics.Measurements["RRU.PrbTotDl"]; ok && prbTotDl > 0 {

			metrics.PRBUtilizationDL = (prbUsedDl / prbTotDl) * 100

		}

	}



	if prbUsedUl, ok := metrics.Measurements["RRU.PrbUsedUl"]; ok {

		if prbTotUl, ok := metrics.Measurements["RRU.PrbTotUl"]; ok && prbTotUl > 0 {

			metrics.PRBUtilizationUL = (prbUsedUl / prbTotUl) * 100

		}

	}



	// Throughput in Mbps (assuming volume is in bytes and measured over 1 second).

	if volDl, ok := metrics.Measurements["DRB.PdcpSduVolumeDL"]; ok {

		metrics.ThroughputDL = (volDl * 8) / (1024 * 1024) // Convert bytes to Mbps

	}



	if volUl, ok := metrics.Measurements["DRB.PdcpSduVolumeUL"]; ok {

		metrics.ThroughputUL = (volUl * 8) / (1024 * 1024) // Convert bytes to Mbps

	}

}



// setupHTTPServer configures the HTTP server for metrics API.

func (x *KMPMonitorXApp) setupHTTPServer() {

	mux := http.NewServeMux()



	// Health check endpoint.

	mux.HandleFunc("/health", x.handleHealth)



	// Metrics endpoints.

	mux.HandleFunc("/metrics", x.handleMetrics)

	mux.HandleFunc("/metrics/", x.handleCellMetrics)



	// xApp info endpoint.

	mux.HandleFunc("/info", x.handleInfo)



	x.httpServer = &http.Server{

		Addr:    ":8080",

		Handler: mux,

	}

}



// handleHealth provides health check information.

func (x *KMPMonitorXApp) handleHealth(w http.ResponseWriter, r *http.Request) {

	state := x.sdk.GetState()

	metrics := x.sdk.GetMetrics()



	health := map[string]interface{}{

		"status":    string(state),

		"timestamp": time.Now().UTC(),

		"sdk_metrics": map[string]interface{}{

			"subscriptions_active":  metrics.SubscriptionsActive,

			"indications_received":  metrics.IndicationsReceived,

			"control_requests_sent": metrics.ControlRequestsSent,

			"error_count":           metrics.ErrorCount,

			"throughput_per_second": metrics.ThroughputPerSecond,

		},

	}



	w.Header().Set("Content-Type", "application/json")

	if state != e2.XAppStateRunning {

		w.WriteHeader(http.StatusServiceUnavailable)

	}



	json.NewEncoder(w).Encode(health)

}



// handleMetrics returns metrics for all cells.

func (x *KMPMonitorXApp) handleMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(x.metrics)

}



// handleCellMetrics returns metrics for a specific cell.

func (x *KMPMonitorXApp) handleCellMetrics(w http.ResponseWriter, r *http.Request) {

	cellID := r.URL.Path[len("/metrics/"):]

	if cellID == "" {

		http.Error(w, "Cell ID is required", http.StatusBadRequest)

		return

	}



	metrics, exists := x.metrics[cellID]

	if !exists {

		http.Error(w, "Cell not found", http.StatusNotFound)

		return

	}



	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(metrics)

}



// handleInfo returns xApp information.

func (x *KMPMonitorXApp) handleInfo(w http.ResponseWriter, r *http.Request) {

	config := x.sdk.GetConfig()

	info := map[string]interface{}{

		"name":           config.XAppName,

		"version":        config.XAppVersion,

		"description":    config.XAppDescription,

		"e2_node_id":     config.E2NodeID,

		"ric_url":        config.NearRTRICURL,

		"service_models": config.ServiceModels,

		"state":          string(x.sdk.GetState()),

		"cells":          len(x.metrics),

	}



	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(info)

}



// Note: Helper functions have been moved to pkg/config/env_helpers.go.

