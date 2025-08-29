// Package conductor provides the core reconciliation and watch functionality.

// for the Nephoran Intent Operator's intent processing and lifecycle management.


package conductor



import (

	"context"

	"encoding/json"

	"fmt"

	"os"

	"os/exec"

	"path/filepath"

	"regexp"

	"strconv"

	"strings"

	"time"



	"github.com/go-logr/logr"



	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"



	"k8s.io/apimachinery/pkg/runtime"



	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/log"

)



// WatchReconciler reconciles a NetworkIntent object for conductor-watch.

type WatchReconciler struct {

	client.Client

	Scheme    *runtime.Scheme

	Logger    logr.Logger // Injected logger

	PorchPath string      // Path to porch CLI binary

	PorchMode string      // "apply" or "dry-run"

	OutputDir string      // Directory for output files

	DryRun    bool        // Skip actual porch execution

}



//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/status,verbs=get;update;patch

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/finalizers,verbs=update



// Reconcile is part of the main kubernetes reconciliation loop which aims to.

// move the current state of the cluster closer to the desired state.

func (r *WatchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Use injected logger or fallback to context logger.

	logger := r.Logger

	if logger.GetSink() == nil {

		logger = log.FromContext(ctx)

	}



	// Enhance logger with request context.

	logger = logger.WithValues(

		"reconcile.namespace", req.Namespace,

		"reconcile.name", req.Name,

		"reconcile.namespacedName", req.NamespacedName.String(),

	)



	// Log reconciliation start.

	logger.Info("Starting NetworkIntent reconciliation")



	// Fetch the NetworkIntent instance.

	var networkIntent nephoranv1.NetworkIntent

	if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {

		if client.IgnoreNotFound(err) != nil {

			logger.Error(err, "Failed to get NetworkIntent")

			return ctrl.Result{}, err

		}

		// Object was deleted.

		logger.Info("NetworkIntent not found, likely deleted - reconciliation complete")

		return ctrl.Result{}, nil

	}



	// Enhance logger with object metadata.

	logger = logger.WithValues(

		"object.generation", networkIntent.Generation,

		"object.resourceVersion", networkIntent.ResourceVersion,

		"object.uid", networkIntent.UID,

		"object.creationTimestamp", networkIntent.CreationTimestamp.Format(time.RFC3339),

	)



	// Log the NetworkIntent details.

	logger.Info("NetworkIntent found and processing",

		"generation", networkIntent.Generation,

		"resourceVersion", networkIntent.ResourceVersion,

	)



	// Log and validate the spec.

	intentSummary := "<empty>"

	if networkIntent.Spec.Intent != "" {

		intentSummary = networkIntent.Spec.Intent

		if len(intentSummary) > 100 {

			intentSummary = intentSummary[:97] + "..."

		}

		logger.Info("Processing NetworkIntent spec",

			"intentSummary", intentSummary,

			"intentLength", len(networkIntent.Spec.Intent),

		)

	} else {

		logger.Info("NetworkIntent spec is empty, skipping processing")

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	}



	// Log status if present.

	if networkIntent.Status.Phase != "" {

		logger.Info("Current NetworkIntent status",

			"phase", networkIntent.Status.Phase,

			"observedGeneration", networkIntent.Status.ObservedGeneration,

			"lastMessage", networkIntent.Status.LastMessage,

		)

	}



	// Parse intent and generate JSON.

	logger.Info("Parsing NetworkIntent spec to JSON")

	intentData, err := r.parseIntentToJSON(&networkIntent)

	if err != nil {

		logger.Error(err, "Failed to parse intent to JSON",

			"intent", networkIntent.Spec.Intent,

		)

		// Requeue after 30 seconds on parse error.

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	}



	logger.Info("Successfully generated intent JSON",

		"intentType", intentData["intent_type"],

		"target", intentData["target"],

		"replicas", intentData["replicas"],

		"correlationId", intentData["correlation_id"],

	)



	// Write intent JSON to file.

	logger.Info("Writing intent JSON to file")

	intentFile, err := r.writeIntentJSON(req.NamespacedName.String(), intentData)

	if err != nil {

		logger.Error(err, "Failed to write intent JSON file")

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	}



	logger.Info("Successfully written intent JSON file",

		"file", intentFile,

		"outputDir", r.OutputDir,

	)



	// Execute porch CLI (conditional on dry-run flag).

	if r.DryRun {

		logger.Info("Dry-run mode: skipping porch execution",

			"intentFile", intentFile,

			"porchPath", r.PorchPath,

			"porchMode", r.PorchMode,

		)

	} else {

		logger.Info("Executing porch CLI",

			"porchPath", r.PorchPath,

			"porchMode", r.PorchMode,

			"intentFile", intentFile,

		)

		if err := r.executePorch(logger, intentFile, req.NamespacedName.String()); err != nil {

			logger.Error(err, "Failed to execute porch CLI",

				"intentFile", intentFile,

				"porchPath", r.PorchPath,

				"mode", r.PorchMode,

			)

			// Requeue after 1 minute on porch error.

			return ctrl.Result{RequeueAfter: time.Minute}, nil

		}

		logger.Info("Successfully executed porch CLI")

	}



	logger.Info("NetworkIntent reconciliation completed successfully",

		"porchMode", r.PorchMode,

		"dryRun", r.DryRun,

		"processingTime", time.Since(time.Now().Add(-time.Since(time.Now()))),

	)



	return ctrl.Result{}, nil

}



// parseIntentToJSON converts NetworkIntent spec to intent JSON.

func (r *WatchReconciler) parseIntentToJSON(ni *nephoranv1.NetworkIntent) (map[string]interface{}, error) {

	intent := ni.Spec.Intent



	// Regex patterns for different intent formats.

	patterns := []struct {

		regex *regexp.Regexp

		name  string

	}{

		{regexp.MustCompile(`scale\s+(deployment|app|service)\s+(\S+)\s+to\s+(\d+)\s+replicas?`), "scale_to"},

		{regexp.MustCompile(`scale\s+(\S+)\s+to\s+(\d+)\s+replicas?`), "scale_simple"},

		{regexp.MustCompile(`scale\s+(deployment|app|service)\s+(\S+)\s+replicas?:\s*(\d+)`), "scale_colon"},

		{regexp.MustCompile(`scale\s+(\S+)\s+(\d+)\s+instances?`), "scale_instances"},

	}



	var target string

	var replicas int



	for _, p := range patterns {

		matches := p.regex.FindStringSubmatch(intent)

		if len(matches) > 0 {

			switch p.name {

			case "scale_to":

				target = matches[2]

				replicas, _ = strconv.Atoi(matches[3])

			case "scale_simple":

				target = matches[1]

				replicas, _ = strconv.Atoi(matches[2])

			case "scale_colon":

				target = matches[2]

				replicas, _ = strconv.Atoi(matches[3])

			case "scale_instances":

				target = matches[1]

				replicas, _ = strconv.Atoi(matches[2])

			}

			break

		}

	}



	if target == "" || replicas == 0 {

		return nil, fmt.Errorf("unable to parse intent: %s", intent)

	}



	// Validate replicas.

	if replicas > 100 {

		return nil, fmt.Errorf("replicas count too high: %d (max 100)", replicas)

	}



	// Create intent JSON matching docs/contracts/intent.schema.json.

	return map[string]interface{}{

		"intent_type":    "scaling",

		"target":         target,

		"namespace":      ni.Namespace,

		"replicas":       replicas,

		"source":         "conductor-watch",

		"correlation_id": fmt.Sprintf("%s-%s-%d", ni.Name, ni.Namespace, time.Now().Unix()),

		"reason":         fmt.Sprintf("Generated from NetworkIntent %s/%s", ni.Namespace, ni.Name),

	}, nil

}



// writeIntentJSON writes the intent data to a JSON file.

func (r *WatchReconciler) writeIntentJSON(name string, data map[string]interface{}) (string, error) {

	// Ensure output directory exists.

	if err := os.MkdirAll(r.OutputDir, 0o755); err != nil {

		return "", fmt.Errorf("failed to create output directory: %w", err)

	}



	// Generate filename with timestamp.

	timestamp := time.Now().Format("20060102T150405")

	filename := fmt.Sprintf("intent-%s-%s.json",

		strings.ReplaceAll(name, "/", "-"),

		timestamp,

	)

	filepath := filepath.Join(r.OutputDir, filename)



	// Marshal to JSON.

	jsonData, err := json.MarshalIndent(data, "", "  ")

	if err != nil {

		return "", fmt.Errorf("failed to marshal JSON: %w", err)

	}



	// Write file.

	if err := os.WriteFile(filepath, jsonData, 0o640); err != nil {

		return "", fmt.Errorf("failed to write file: %w", err)

	}



	return filepath, nil

}



// executePorch executes the porch CLI with the intent file.

func (r *WatchReconciler) executePorch(logger logr.Logger, intentFile, name string) error {

	// Build command arguments.

	args := []string{

		"generate",

		"--intent", intentFile,

		"--mode", r.PorchMode,

		"--output", r.OutputDir,

	}



	logger.Info("Preparing porch command",

		"command", r.PorchPath,

		"args", strings.Join(args, " "),

		"workingDir", r.OutputDir,

	)



	// Create command.

	cmd := exec.Command(r.PorchPath, args...)

	cmd.Dir = r.OutputDir



	// Set environment.

	cmd.Env = os.Environ()



	// Execute command.

	logger.Info("Executing porch command")

	output, err := cmd.CombinedOutput()

	if err != nil {

		logger.Error(err, "Porch execution failed",

			"output", string(output),

			"exitCode", cmd.ProcessState.ExitCode(),

		)

		return fmt.Errorf("porch execution failed: %w, output: %s", err, string(output))

	}



	logger.Info("Porch execution completed successfully",

		"output", string(output),

	)



	return nil

}



// SetupWithManager sets up the controller with the Manager.

func (r *WatchReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).

		For(&nephoranv1.NetworkIntent{}).

		Complete(r)

}

