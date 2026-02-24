package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Prometheus metrics
var (
	// Counters
	policiesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scaling_xapp_policies_processed_total",
			Help: "Total number of policies processed",
		},
		[]string{"namespace", "deployment", "result"},
	)

	a1Requests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scaling_xapp_a1_requests_total",
			Help: "Total number of A1 API requests",
		},
		[]string{"method", "status_code"},
	)

	// Gauges
	activePolicies = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "scaling_xapp_active_policies",
			Help: "Number of active policies",
		},
	)

	lastPollTimestamp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "scaling_xapp_last_poll_timestamp",
			Help: "Timestamp of last successful poll",
		},
	)

	// Histograms
	a1RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scaling_xapp_a1_request_duration_seconds",
			Help:    "A1 API request duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	scalingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scaling_xapp_scaling_duration_seconds",
			Help:    "Scaling operation duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"namespace", "deployment"},
	)
)

// A1Policy represents the policy received from A1 Mediator
type A1Policy struct {
	PolicyID   string                 `json:"policy_id"`
	PolicyType int                    `json:"policy_type_id"`
	RicID      string                 `json:"ric_id"`
	Service    string                 `json:"service_id"`
	JSON       map[string]interface{} `json:"json"`
}

// A1PolicyData represents the actual A1 policy structure
type A1PolicyData struct {
	QoSObjectives struct {
		Replicas int32 `json:"replicas"`
	} `json:"qosObjectives"`
	Scope struct {
		IntentType string `json:"intentType"`
		Target     string `json:"target"`
		Namespace  string `json:"namespace"`
	} `json:"scope"`
}

// ScalingSpec from the policy JSON
type ScalingSpec struct {
	IntentType string `json:"intentType"`
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	Replicas   int32  `json:"replicas"`
	Source     string `json:"source"`
}

type ScalingXApp struct {
	k8sClient    *kubernetes.Clientset
	a1URL        string
	pollInterval time.Duration
}

func NewScalingXApp() (*ScalingXApp, error) {
	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	a1URL := os.Getenv("A1_MEDIATOR_URL")
	if a1URL == "" {
		a1URL = "http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000"
	}

	pollInterval := 30 * time.Second
	if interval := os.Getenv("POLL_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			pollInterval = d
		}
	}

	return &ScalingXApp{
		k8sClient:    clientset,
		a1URL:        a1URL,
		pollInterval: pollInterval,
	}, nil
}

func (x *ScalingXApp) Start(ctx context.Context) error {
	log.Printf("Starting Scaling xApp")
	log.Printf("A1 Mediator URL: %s", x.a1URL)
	log.Printf("Poll Interval: %v", x.pollInterval)

	ticker := time.NewTicker(x.pollInterval)
	defer ticker.Stop()

	// Initial poll
	if err := x.pollAndExecutePolicies(ctx); err != nil {
		log.Printf("Error during initial poll: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutting down Scaling xApp")
			return ctx.Err()
		case <-ticker.C:
			if err := x.pollAndExecutePolicies(ctx); err != nil {
				log.Printf("Error polling policies: %v", err)
			}
		}
	}
}

func (x *ScalingXApp) pollAndExecutePolicies(ctx context.Context) error {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("GET_POLICIES").Observe(time.Since(start).Seconds())
	}()

	// Get all policies of type 100 (scaling)
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/100/policies", x.a1URL)

	resp, err := http.Get(url)
	if err != nil {
		a1Requests.WithLabelValues("GET", "error").Inc()
		return fmt.Errorf("failed to get policies: %v", err)
	}
	defer resp.Body.Close()

	a1Requests.WithLabelValues("GET", strconv.Itoa(resp.StatusCode)).Inc()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	var policyIDs []string
	if err := json.Unmarshal(body, &policyIDs); err != nil {
		return fmt.Errorf("failed to unmarshal policy IDs: %v", err)
	}

	log.Printf("Found %d scaling policies", len(policyIDs))

	// Update metrics
	activePolicies.Set(float64(len(policyIDs)))
	lastPollTimestamp.SetToCurrentTime()

	// Process each policy
	for _, policyID := range policyIDs {
		if err := x.executePolicy(ctx, policyID); err != nil {
			log.Printf("Error executing policy %s: %v", policyID, err)
		}
	}

	return nil
}

func (x *ScalingXApp) executePolicy(ctx context.Context, policyID string) error {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("GET_POLICY").Observe(time.Since(start).Seconds())
	}()

	// Get policy details
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/100/policies/%s", x.a1URL, policyID)

	resp, err := http.Get(url)
	if err != nil {
		a1Requests.WithLabelValues("GET", "error").Inc()
		return fmt.Errorf("failed to get policy: %v", err)
	}
	defer resp.Body.Close()

	a1Requests.WithLabelValues("GET", strconv.Itoa(resp.StatusCode)).Inc()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	// Parse the nested A1 policy structure
	var policyData A1PolicyData
	if err := json.Unmarshal(body, &policyData); err != nil {
		return fmt.Errorf("failed to unmarshal policy: %v", err)
	}

	// Convert to ScalingSpec
	spec := ScalingSpec{
		IntentType: policyData.Scope.IntentType,
		Target:     policyData.Scope.Target,
		Namespace:  policyData.Scope.Namespace,
		Replicas:   policyData.QoSObjectives.Replicas,
	}

	log.Printf("Executing scaling policy: %s (target=%s, namespace=%s, replicas=%d)",
		policyID, spec.Target, spec.Namespace, spec.Replicas)

	// Execute the scaling action
	return x.scaleDeployment(ctx, spec)
}

func (x *ScalingXApp) scaleDeployment(ctx context.Context, spec ScalingSpec) error {
	start := time.Now()
	defer func() {
		scalingDuration.WithLabelValues(spec.Namespace, spec.Target).Observe(time.Since(start).Seconds())
	}()

	// Get the deployment
	deployment, err := x.k8sClient.AppsV1().Deployments(spec.Namespace).Get(
		ctx, spec.Target, metav1.GetOptions{})
	if err != nil {
		policiesProcessed.WithLabelValues(spec.Namespace, spec.Target, "failed").Inc()
		return fmt.Errorf("failed to get deployment: %v", err)
	}

	currentReplicas := *deployment.Spec.Replicas
	if currentReplicas == spec.Replicas {
		log.Printf("Deployment %s/%s already at desired replicas (%d)",
			spec.Namespace, spec.Target, spec.Replicas)
		policiesProcessed.WithLabelValues(spec.Namespace, spec.Target, "already_scaled").Inc()
		return nil
	}

	// Update replicas
	deployment.Spec.Replicas = &spec.Replicas

	_, err = x.k8sClient.AppsV1().Deployments(spec.Namespace).Update(
		ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		policiesProcessed.WithLabelValues(spec.Namespace, spec.Target, "failed").Inc()
		return fmt.Errorf("failed to update deployment: %v", err)
	}

	log.Printf("✅ Successfully scaled %s/%s: %d → %d replicas",
		spec.Namespace, spec.Target, currentReplicas, spec.Replicas)

	policiesProcessed.WithLabelValues(spec.Namespace, spec.Target, "success").Inc()
	return nil
}

func main() {
	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Printf("Metrics server listening on :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	xapp, err := NewScalingXApp()
	if err != nil {
		log.Fatalf("Failed to create Scaling xApp: %v", err)
	}

	ctx := context.Background()
	if err := xapp.Start(ctx); err != nil {
		log.Fatalf("Scaling xApp stopped: %v", err)
	}
}
