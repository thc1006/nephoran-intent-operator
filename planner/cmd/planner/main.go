package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/planner"
	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

type Config struct {
	MetricsURL      string        `json:"metrics_url"`
	EventsURL       string        `json:"events_url"`
	OutputDir       string        `json:"output_dir"`
	IntentEndpoint  string        `json:"intent_endpoint"`
	PollingInterval time.Duration `json:"polling_interval"`
	SimMode         bool          `json:"sim_mode"`
	SimDataFile     string        `json:"sim_data_file"`
	StateFile       string        `json:"state_file"`
	MetricsDir      string        `json:"metrics_dir"`
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "planner/config/config.yaml", "Configuration file path")
	flag.Parse()

	cfg := &Config{
		MetricsURL:      "http://localhost:9090/metrics/kpm",
		EventsURL:       "http://localhost:9091/events/ves",
		OutputDir:       "./handoff",
		IntentEndpoint:  "http://localhost:8080/intent",
		PollingInterval: 30 * time.Second,
		SimMode:         false,
		SimDataFile:     "examples/planner/kpm-sample.json",
		StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
	}

	if configFile != "" && fileExists(configFile) {
		loadConfig(configFile, cfg)
	}

	if envURL := os.Getenv("PLANNER_METRICS_URL"); envURL != "" {
		cfg.MetricsURL = envURL
	}
	if envDir := os.Getenv("PLANNER_OUTPUT_DIR"); envDir != "" {
		cfg.OutputDir = envDir
	}
	if envSim := os.Getenv("PLANNER_SIM_MODE"); envSim == "true" {
		cfg.SimMode = true
	}
	if envMetricsDir := os.Getenv("PLANNER_METRICS_DIR"); envMetricsDir != "" {
		cfg.MetricsDir = envMetricsDir
	}

	log.Printf("Starting Nephoran Closed-Loop Planner")
	log.Printf("Config: MetricsURL=%s, OutputDir=%s, PollingInterval=%v, SimMode=%v",
		cfg.MetricsURL, cfg.OutputDir, cfg.PollingInterval, cfg.SimMode)

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	engine := rules.NewRuleEngine(rules.Config{
		StateFile:            cfg.StateFile,
		CooldownDuration:     60 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     90 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		runPlannerLoop(ctx, cfg, engine)
	}()

	<-sigChan
	log.Println("Shutting down planner...")
	cancel()
	wg.Wait()
	log.Println("Planner stopped")
}

func runPlannerLoop(ctx context.Context, cfg *Config, engine *rules.RuleEngine) {
	ticker := time.NewTicker(cfg.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processMetrics(cfg, engine)
		}
	}
}

func processMetrics(cfg *Config, engine *rules.RuleEngine) {
	var kpmData rules.KPMData
	var err error

	if cfg.SimMode {
		kpmData, err = loadSimData(cfg.SimDataFile)
	} else if cfg.MetricsDir != "" {
		kpmData, err = loadLatestMetricsFromDir(cfg.MetricsDir)
	} else {
		kpmData, err = fetchKPMMetrics(cfg.MetricsURL)
	}

	if err != nil {
		log.Printf("Error fetching metrics: %v", err)
		return
	}

	decision := engine.Evaluate(kpmData)
	if decision == nil {
		log.Println("No scaling decision made")
		return
	}

	log.Printf("Scaling decision: %s, Reason: %s, Target replicas: %d",
		decision.Action, decision.Reason, decision.TargetReplicas)

	intent := &planner.Intent{
		IntentType:    "scaling",
		Target:        decision.Target,
		Namespace:     decision.Namespace,
		Replicas:      decision.TargetReplicas,
		Reason:        decision.Reason,
		Source:        "planner",
		CorrelationID: fmt.Sprintf("planner-%d", time.Now().Unix()),
	}

	if err := writeIntent(cfg.OutputDir, intent); err != nil {
		log.Printf("Error writing intent: %v", err)
	}
}

func fetchKPMMetrics(url string) (rules.KPMData, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return rules.KPMData{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return rules.KPMData{}, err
	}

	var data rules.KPMData
	if err := json.Unmarshal(body, &data); err != nil {
		return rules.KPMData{}, err
	}

	return data, nil
}

func loadSimData(file string) (rules.KPMData, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return rules.KPMData{}, err
	}

	var kpmData rules.KPMData
	if err := json.Unmarshal(data, &kpmData); err != nil {
		return rules.KPMData{}, err
	}

	return kpmData, nil
}

func loadLatestMetricsFromDir(dir string) (rules.KPMData, error) {
	// Read all JSON files from metrics directory
	files, err := filepath.Glob(filepath.Join(dir, "kpm-*.json"))
	if err != nil {
		return rules.KPMData{}, fmt.Errorf("failed to list metrics files: %w", err)
	}

	if len(files) == 0 {
		return rules.KPMData{}, fmt.Errorf("no metrics files found in %s", dir)
	}

	// Get the most recent file (but we'll read them all for history)
	var latestFile string
	var latestTime time.Time
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = file
		}
	}

	if latestFile == "" {
		return rules.KPMData{}, fmt.Errorf("no valid metrics files found")
	}

	// Read the latest file for current data
	log.Printf("Reading latest metrics from: %s", latestFile)
	latestData, err := loadSimData(latestFile)
	if err != nil {
		return rules.KPMData{}, err
	}

	// Note: In a real system, we'd accumulate history differently
	// For now, return the latest data point
	return latestData, nil
}

func writeIntent(outputDir string, intent *planner.Intent) error {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("intent-%d.json", time.Now().Unix())
	path := filepath.Join(outputDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	log.Printf("Intent written to %s", path)
	return nil
}

func loadConfig(file string, cfg *Config) error {
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
