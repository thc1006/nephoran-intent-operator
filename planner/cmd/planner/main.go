package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/thc1006/nephoran-intent-operator/internal/planner"
	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

// Config represents the main application configuration
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

// YAMLConfig represents the YAML file structure
type YAMLConfig struct {
	Planner      PlannerConfig      `yaml:"planner"`
	ScalingRules ScalingRulesConfig `yaml:"scaling_rules"`
	Logging      LoggingConfig      `yaml:"logging"`
}

// httpClient is a reusable HTTP client optimized for polling scenarios
// with connection pooling and proper timeouts
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		// Connection pooling settings optimized for repeated requests
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		
		// Timeouts for different phases of the request
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		
		// Disable compression for KMP metrics (typically small JSON)
		DisableCompression: true,
		
		// Force HTTP/1.1 for better connection reuse with most metrics endpoints
		ForceAttemptHTTP2: false,
	},
}

// PlannerConfig represents the planner section of the YAML
type PlannerConfig struct {
	MetricsURL      string `yaml:"metrics_url"`
	EventsURL       string `yaml:"events_url"`
	OutputDir       string `yaml:"output_dir"`
	IntentEndpoint  string `yaml:"intent_endpoint"`
	PollingInterval string `yaml:"polling_interval"`
	SimMode         bool   `yaml:"sim_mode"`
	SimDataFile     string `yaml:"sim_data_file"`
	StateFile       string `yaml:"state_file"`
}

// ScalingRulesConfig represents the scaling_rules section
type ScalingRulesConfig struct {
	CooldownDuration  string                 `yaml:"cooldown_duration"`
	MinReplicas       int                    `yaml:"min_replicas"`
	MaxReplicas       int                    `yaml:"max_replicas"`
	EvaluationWindow  string                 `yaml:"evaluation_window"`
	Thresholds        ScalingThresholdsConfig `yaml:"thresholds"`
}

// ScalingThresholdsConfig represents the thresholds section
type ScalingThresholdsConfig struct {
	Latency        ThresholdConfig `yaml:"latency"`
	PRBUtilization ThresholdConfig `yaml:"prb_utilization"`
}

// ThresholdConfig represents individual threshold configuration
type ThresholdConfig struct {
	ScaleOut float64 `yaml:"scale_out"`
	ScaleIn  float64 `yaml:"scale_in"`
}

// LoggingConfig represents the logging section
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "planner/config/config.yaml", "Configuration file path")
	flag.Parse()

	cfg := &Config{
		MetricsURL:      "http://localhost:9090/metrics/kmp",
		EventsURL:       "http://localhost:9091/events/ves",
		OutputDir:       "./handoff",
		IntentEndpoint:  "http://localhost:8080/intent",
		PollingInterval: 30 * time.Second,
		SimMode:         false,
		SimDataFile:     "examples/planner/kmp-sample.json",
		StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
	}

	var yamlConfig *YAMLConfig
	if configFile != "" && fileExists(configFile) {
		if err := loadConfig(configFile, cfg); err != nil {
			log.Printf("Warning: Failed to load config file %s: %v", configFile, err)
			log.Printf("Using default configuration")
		} else {
			// Also load the full YAML config for rule engine configuration
			if data, err := os.ReadFile(configFile); err == nil {
				var fullConfig YAMLConfig
				if err := yaml.Unmarshal(data, &fullConfig); err == nil {
					yamlConfig = &fullConfig
				}
			}
		}
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

	// Create rule engine configuration
	ruleConfig := createRuleEngineConfig(cfg, yamlConfig)
	engine := rules.NewRuleEngine(ruleConfig)

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
	var kmpData rules.KPMData
	var err error

	if cfg.SimMode {
		kmpData, err = loadSimData(cfg.SimDataFile)
	} else if cfg.MetricsDir != "" {
		kmpData, err = loadLatestMetricsFromDir(cfg.MetricsDir)
	} else {
		kmpData, err = fetchKPMMetrics(cfg.MetricsURL)
	}

	if err != nil {
		log.Printf("Error fetching metrics: %v", err)
		return
	}

	decision := engine.Evaluate(kmpData)
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
	resp, err := httpClient.Get(url)
	if err != nil {
		return rules.KPMData{}, fmt.Errorf("failed to fetch metrics from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rules.KPMData{}, fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return rules.KPMData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var data rules.KPMData
	if err := json.Unmarshal(body, &data); err != nil {
		return rules.KPMData{}, fmt.Errorf("failed to unmarshal KMP data: %w", err)
	}

	return data, nil
}

func loadSimData(file string) (rules.KPMData, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return rules.KPMData{}, err
	}

	var kmpData rules.KPMData
	if err := json.Unmarshal(data, &kmpData); err != nil {
		return rules.KPMData{}, err
	}

	return kmpData, nil
}

func loadLatestMetricsFromDir(dir string) (rules.KPMData, error) {
	// Read all JSON files from metrics directory
	files, err := filepath.Glob(filepath.Join(dir, "kmp-*.json"))
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

// writeIntent writes scaling intent to a secure file with O-RAN compliant permissions.
// Intent files contain sensitive O-RAN network management data and must be protected
// from unauthorized access according to O-RAN WG11 security specifications.
func writeIntent(outputDir string, intent *planner.Intent) error {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal intent data: %w", err)
	}

	filename := fmt.Sprintf("intent-%d.json", time.Now().Unix())
	path := filepath.Join(outputDir, filename)

	// SECURITY: Use 0600 permissions to ensure only the owner can read/write intent files.
	// This prevents unauthorized access to sensitive O-RAN network management data.
	// O-RAN WG11 security requirements mandate protection of operational data.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write intent file with secure permissions: %w", err)
	}

	log.Printf("Intent written securely to %s (permissions: 0600)", path)
	return nil
}

// loadConfig reads and parses the YAML configuration file
func loadConfig(file string, cfg *Config) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", file, err)
	}

	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Apply planner configuration
	if err := applyPlannerConfig(&yamlConfig.Planner, cfg); err != nil {
		return fmt.Errorf("failed to apply planner config: %w", err)
	}

	log.Printf("Successfully loaded configuration from %s", file)
	return nil
}

// applyPlannerConfig applies the planner section to the main Config struct
func applyPlannerConfig(plannerCfg *PlannerConfig, cfg *Config) error {
	if plannerCfg.MetricsURL != "" {
		cfg.MetricsURL = plannerCfg.MetricsURL
	}
	if plannerCfg.EventsURL != "" {
		cfg.EventsURL = plannerCfg.EventsURL
	}
	if plannerCfg.OutputDir != "" {
		cfg.OutputDir = plannerCfg.OutputDir
	}
	if plannerCfg.IntentEndpoint != "" {
		cfg.IntentEndpoint = plannerCfg.IntentEndpoint
	}
	if plannerCfg.SimDataFile != "" {
		cfg.SimDataFile = plannerCfg.SimDataFile
	}
	if plannerCfg.StateFile != "" {
		cfg.StateFile = plannerCfg.StateFile
	}
	
	// Set SimMode
	cfg.SimMode = plannerCfg.SimMode

	// Parse polling interval
	if plannerCfg.PollingInterval != "" {
		duration, err := time.ParseDuration(plannerCfg.PollingInterval)
		if err != nil {
			return fmt.Errorf("invalid polling_interval format '%s': %w", plannerCfg.PollingInterval, err)
		}
		cfg.PollingInterval = duration
	}

	// Handle state file path - if not absolute, use temp dir
	if cfg.StateFile != "" && !filepath.IsAbs(cfg.StateFile) {
		cfg.StateFile = filepath.Join(os.TempDir(), cfg.StateFile)
	}

	return nil
}

// createRuleEngineConfig creates the rule engine configuration from YAML config or defaults
func createRuleEngineConfig(cfg *Config, yamlConfig *YAMLConfig) rules.Config {
	// Default values
	ruleConfig := rules.Config{
		StateFile:            cfg.StateFile,
		CooldownDuration:     60 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     90 * time.Second,
	}

	// Apply YAML configuration if available
	if yamlConfig != nil {
		scalingRules := &yamlConfig.ScalingRules
		
		// Parse durations
		if scalingRules.CooldownDuration != "" {
			if duration, err := time.ParseDuration(scalingRules.CooldownDuration); err == nil {
				ruleConfig.CooldownDuration = duration
			} else {
				log.Printf("Warning: Invalid cooldown_duration '%s', using default", scalingRules.CooldownDuration)
			}
		}
		
		if scalingRules.EvaluationWindow != "" {
			if duration, err := time.ParseDuration(scalingRules.EvaluationWindow); err == nil {
				ruleConfig.EvaluationWindow = duration
			} else {
				log.Printf("Warning: Invalid evaluation_window '%s', using default", scalingRules.EvaluationWindow)
			}
		}

		// Apply replica limits
		if scalingRules.MinReplicas > 0 {
			ruleConfig.MinReplicas = scalingRules.MinReplicas
		}
		if scalingRules.MaxReplicas > 0 {
			ruleConfig.MaxReplicas = scalingRules.MaxReplicas
		}

		// Apply thresholds
		thresholds := &scalingRules.Thresholds
		if thresholds.Latency.ScaleOut > 0 {
			ruleConfig.LatencyThresholdHigh = thresholds.Latency.ScaleOut
		}
		if thresholds.Latency.ScaleIn > 0 {
			ruleConfig.LatencyThresholdLow = thresholds.Latency.ScaleIn
		}
		if thresholds.PRBUtilization.ScaleOut > 0 {
			ruleConfig.PRBThresholdHigh = thresholds.PRBUtilization.ScaleOut
		}
		if thresholds.PRBUtilization.ScaleIn > 0 {
			ruleConfig.PRBThresholdLow = thresholds.PRBUtilization.ScaleIn
		}
	}

	return ruleConfig
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}