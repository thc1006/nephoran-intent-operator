package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Intent represents the intent JSON structure for extracting correlation_id
type Intent struct {
	CorrelationID string `json:"correlation_id,omitempty"`
}

// initLogger initializes structured logging with proper configuration
func initLogger() logr.Logger {
	config := zap.Config{
		Level:    zap.NewAtomicLevelAt(zap.InfoLevel),
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Use console encoding for development if not in production
	if os.Getenv("ENVIRONMENT") != "production" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapLogger, err := config.Build()
	if err != nil {
		// Fallback to basic logger if configuration fails
		zapLogger, _ = zap.NewDevelopment()
	}
	
	return zapr.NewLogger(zapLogger).WithName("conductor")
}

func main() {
	var (
		handoffDir string
		outDir     string
	)

	flag.StringVar(&handoffDir, "watch", "handoff", "Directory to watch for intent-*.json files")
	flag.StringVar(&outDir, "out", "", "Output directory for scaling patches (defaults to examples/packages/scaling)")
	flag.Parse()

	// Initialize structured logger
	logger := initLogger()

	// Set default output directory if not specified
	if outDir == "" {
		outDir = os.Getenv("CONDUCTOR_OUT_DIR")
		if outDir == "" {
			outDir = "examples/packages/scaling"
		}
	}

	logger.Info("Starting conductor file watcher", "watchDir", handoffDir, "outDir", outDir)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Failed to create file system watcher")
		os.Exit(1)
	}
	defer func() { 
		if err := watcher.Close(); err != nil {
			logger.Error(err, "Failed to close file system watcher")
		}
	}()

	// Add handoff directory to watcher
	if err := watcher.Add(handoffDir); err != nil {
		logger.Error(err, "Failed to add directory to watcher", "directory", handoffDir)
		os.Exit(1)
	}

	// Start event processing
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Event processing stopped due to context cancellation")
				return
			case event, ok := <-watcher.Events:
				if !ok {
					logger.Info("Event channel closed, stopping event processing")
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					if isIntentFile(event.Name) {
						logger.Info("New intent file detected", "file", event.Name)
						// Add small delay to ensure file is fully written
						time.Sleep(100 * time.Millisecond)
						processIntentFile(logger, event.Name, outDir)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					logger.Info("Error channel closed, stopping error processing")
					return
				}
				logger.Error(err, "File system watcher error")
			}
		}
	}()

	logger.Info("Conductor is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received, stopping conductor...")
	cancel()

	// Give goroutines time to finish
	time.Sleep(500 * time.Millisecond)
	logger.Info("Conductor stopped successfully")
}

// isIntentFile checks if the file matches the pattern intent-*.json
func isIntentFile(path string) bool {
	filename := filepath.Base(path)
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}

// processIntentFile triggers porch-publisher for the given intent file
func processIntentFile(logger logr.Logger, intentPath, outDir string) {
	// Try to extract correlation_id from the intent file
	correlationID := extractCorrelationID(logger, intentPath)
	contextLogger := logger.WithValues("intentFile", intentPath, "correlationID", correlationID)
	
	if correlationID != "" {
		contextLogger.Info("Processing intent with correlation ID")
	} else {
		contextLogger.Info("Processing intent without correlation ID")
	}

	// Build the command to run porch-publisher
	cmd := exec.Command("go", "run", "./cmd/porch-publisher", "-intent", intentPath, "-out", outDir)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		contextLogger.Error(err, "Failed to run porch-publisher", "output", string(output))
		return
	}

	contextLogger.Info("Successfully processed intent file", "output", strings.TrimSpace(string(output)))
}

// extractCorrelationID reads the intent file and extracts correlation_id if present
func extractCorrelationID(logger logr.Logger, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error(err, "Could not read intent file for correlation_id extraction", "file", path)
		return ""
	}

	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		logger.Error(err, "Could not parse intent JSON for correlation_id extraction", "file", path)
		return ""
	}

	return intent.CorrelationID
}