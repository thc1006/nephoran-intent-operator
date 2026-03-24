package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ── Prometheus Metrics ──────────────────────────────────────────────

var (
	policiesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scaling_xapp_policies_processed_total",
			Help: "Total number of policies processed",
		},
		[]string{"policy_type", "namespace", "deployment", "result"},
	)

	a1Requests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scaling_xapp_a1_requests_total",
			Help: "Total number of A1 API requests",
		},
		[]string{"method", "policy_type", "status_code"},
	)

	policyStatusReports = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scaling_xapp_policy_status_reports_total",
			Help: "Total number of policy status reports sent",
		},
		[]string{"policy_type", "enforce_status", "result"},
	)

	activePolicies = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scaling_xapp_active_policies",
			Help: "Number of active policies by type",
		},
		[]string{"policy_type"},
	)

	lastPollTimestamp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "scaling_xapp_last_poll_timestamp",
			Help: "Timestamp of last successful poll",
		},
	)

	a1RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scaling_xapp_a1_request_duration_seconds",
			Help:    "A1 API request duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "policy_type"},
	)

	scalingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scaling_xapp_scaling_duration_seconds",
			Help:    "Scaling operation duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"namespace", "deployment"},
	)

	// TS-specific metrics (low-cardinality labels only)
	tsPoliciesApplied = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scaling_xapp_ts_policies_applied_total",
			Help: "Total number of TS policies applied",
		},
		[]string{"result"},
	)

	tsActivePolicies = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "scaling_xapp_ts_active_policies",
			Help: "Number of active TS policies being enforced",
		},
	)

	tsLastThreshold = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "scaling_xapp_ts_last_threshold",
			Help: "Most recently applied TS threshold value",
		},
	)
)

// ── Data Types ──────────────────────────────────────────────────────

// A1PolicyData represents scaling policy (PolicyType 100)
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

// TSPolicyData represents Traffic Steering policy (PolicyType 20008)
type TSPolicyData struct {
	Threshold int `json:"threshold"`
}

// ScalingSpec from the scaling policy JSON
type ScalingSpec struct {
	IntentType string `json:"intentType"`
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	Replicas   int32  `json:"replicas"`
}

// PolicyStatus represents the status report for Redis storage
type PolicyStatus struct {
	EnforcementStatus string `json:"enforcement_status"`
	EnforcementReason string `json:"enforcement_reason"`
}

// ── ScalingXApp ─────────────────────────────────────────────────────

type ScalingXApp struct {
	k8sClient    *kubernetes.Clientset
	httpClient   *http.Client
	a1URL        string
	pollInterval time.Duration
	redisClient  *redis.Client

	// Dedup: track applied TS thresholds to avoid redundant writes
	tsCache   map[string]int // policyID → last applied threshold
	tsCacheMu sync.Mutex
}

func NewScalingXApp() (*ScalingXApp, error) {
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

	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		redisAddr = "a1-status-store.ricplt:6379"
	}

	redisClient := InitRedisClient(RedisConfig{
		Address: redisAddr,
	})

	return &ScalingXApp{
		k8sClient: clientset,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		a1URL:        a1URL,
		pollInterval: pollInterval,
		redisClient:  redisClient,
		tsCache:      make(map[string]int),
	}, nil
}

func (x *ScalingXApp) Start(ctx context.Context) error {
	log.Printf("Starting Scaling xApp (with TS support)")
	log.Printf("A1 Mediator URL: %s", x.a1URL)
	log.Printf("Poll Interval: %v", x.pollInterval)
	log.Printf("Policy Types: 100 (scaling), 20008 (traffic steering)")

	ticker := time.NewTicker(x.pollInterval)
	defer ticker.Stop()

	x.pollAll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutting down Scaling xApp")
			return ctx.Err()
		case <-ticker.C:
			x.pollAll(ctx)
		}
	}
}

func (x *ScalingXApp) pollAll(ctx context.Context) {
	if err := x.pollScalingPolicies(ctx); err != nil {
		log.Printf("Error polling scaling policies: %v", err)
	}
	if err := x.pollTSPolicies(ctx); err != nil {
		log.Printf("Error polling TS policies: %v", err)
	}
	lastPollTimestamp.SetToCurrentTime()
}

// ── PolicyType 100: Scaling ─────────────────────────────────────────

func (x *ScalingXApp) pollScalingPolicies(ctx context.Context) error {
	policyIDs, err := x.listPolicies("100")
	if err != nil {
		return err
	}

	activePolicies.WithLabelValues("100").Set(float64(len(policyIDs)))

	if len(policyIDs) > 0 {
		log.Printf("Found %d scaling policies", len(policyIDs))
	}

	for _, policyID := range policyIDs {
		if err := x.executeScalingPolicy(ctx, policyID); err != nil {
			log.Printf("Error executing scaling policy %s: %v", policyID, err)
		}
	}
	return nil
}

func (x *ScalingXApp) executeScalingPolicy(ctx context.Context, policyID string) error {
	body, err := x.getPolicy("100", policyID)
	if err != nil {
		return err
	}

	var policyData A1PolicyData
	if err := json.Unmarshal(body, &policyData); err != nil {
		return fmt.Errorf("failed to unmarshal scaling policy: %v", err)
	}

	spec := ScalingSpec{
		IntentType: policyData.Scope.IntentType,
		Target:     policyData.Scope.Target,
		Namespace:  policyData.Scope.Namespace,
		Replicas:   policyData.QoSObjectives.Replicas,
	}

	log.Printf("Executing scaling policy: %s (target=%s/%s, replicas=%d)",
		policyID, spec.Namespace, spec.Target, spec.Replicas)

	err = x.scaleDeployment(ctx, spec)

	if err != nil {
		reason := fmt.Sprintf("Failed to scale %s/%s: %v", spec.Namespace, spec.Target, err)
		x.reportPolicyStatus(100, policyID, false, reason)
		return err
	}

	reason := fmt.Sprintf("Scaled %s/%s to %d replicas", spec.Namespace, spec.Target, spec.Replicas)
	x.reportPolicyStatus(100, policyID, true, reason)
	return nil
}

func (x *ScalingXApp) scaleDeployment(ctx context.Context, spec ScalingSpec) error {
	start := time.Now()
	defer func() {
		scalingDuration.WithLabelValues(spec.Namespace, spec.Target).Observe(time.Since(start).Seconds())
	}()

	deployment, err := x.k8sClient.AppsV1().Deployments(spec.Namespace).Get(
		ctx, spec.Target, metav1.GetOptions{})
	if err != nil {
		policiesProcessed.WithLabelValues("100", spec.Namespace, spec.Target, "failed").Inc()
		return fmt.Errorf("failed to get deployment: %v", err)
	}

	var currentReplicas int32 = 1
	if deployment.Spec.Replicas != nil {
		currentReplicas = *deployment.Spec.Replicas
	}
	if currentReplicas == spec.Replicas {
		log.Printf("Deployment %s/%s already at desired replicas (%d)",
			spec.Namespace, spec.Target, spec.Replicas)
		policiesProcessed.WithLabelValues("100", spec.Namespace, spec.Target, "already_scaled").Inc()
		return nil
	}

	deployment.Spec.Replicas = &spec.Replicas
	_, err = x.k8sClient.AppsV1().Deployments(spec.Namespace).Update(
		ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		policiesProcessed.WithLabelValues("100", spec.Namespace, spec.Target, "failed").Inc()
		return fmt.Errorf("failed to update deployment: %v", err)
	}

	log.Printf("Scaled %s/%s: %d -> %d replicas",
		spec.Namespace, spec.Target, currentReplicas, spec.Replicas)
	policiesProcessed.WithLabelValues("100", spec.Namespace, spec.Target, "success").Inc()
	return nil
}

// ── PolicyType 20008: Traffic Steering ──────────────────────────────

func (x *ScalingXApp) pollTSPolicies(ctx context.Context) error {
	policyIDs, err := x.listPolicies("20008")
	if err != nil {
		return err
	}

	activePolicies.WithLabelValues("20008").Set(float64(len(policyIDs)))
	tsActivePolicies.Set(float64(len(policyIDs)))

	if len(policyIDs) > 0 {
		log.Printf("Found %d TS policies", len(policyIDs))
	}

	for _, policyID := range policyIDs {
		if err := x.executeTSPolicy(ctx, policyID); err != nil {
			log.Printf("Error executing TS policy %s: %v", policyID, err)
		}
	}

	// Evict deleted policies from dedup cache
	active := make(map[string]struct{}, len(policyIDs))
	for _, id := range policyIDs {
		active[id] = struct{}{}
	}
	x.tsCacheMu.Lock()
	for id := range x.tsCache {
		if _, ok := active[id]; !ok {
			delete(x.tsCache, id)
		}
	}
	x.tsCacheMu.Unlock()

	return nil
}

func (x *ScalingXApp) executeTSPolicy(ctx context.Context, policyID string) error {
	body, err := x.getPolicy("20008", policyID)
	if err != nil {
		return err
	}

	var tsData TSPolicyData
	if err := json.Unmarshal(body, &tsData); err != nil {
		return fmt.Errorf("failed to unmarshal TS policy: %v", err)
	}

	threshold := tsData.Threshold
	if threshold < 0 || threshold > 100 {
		reason := fmt.Sprintf("Invalid threshold %d (must be 0-100)", threshold)
		x.reportPolicyStatus(20008, policyID, false, reason)
		tsPoliciesApplied.WithLabelValues("invalid").Inc()
		return fmt.Errorf("%s", reason)
	}

	// Dedup: skip if threshold unchanged since last poll
	x.tsCacheMu.Lock()
	prev, seen := x.tsCache[policyID]
	if seen && prev == threshold {
		x.tsCacheMu.Unlock()
		return nil // already applied, skip redundant write
	}
	x.tsCache[policyID] = threshold
	x.tsCacheMu.Unlock()

	log.Printf("Applying TS policy: %s (threshold=%d%%)", policyID, threshold)

	tsLastThreshold.Set(float64(threshold))

	// Store TS policy data in Redis for observability
	tsStatus := map[string]interface{}{
		"threshold":  threshold,
		"applied_at": time.Now().UTC().Format(time.RFC3339),
		"policy_id":  policyID,
	}
	tsJSON, err := json.Marshal(tsStatus)
	if err != nil {
		log.Printf("Warning: failed to marshal TS status: %v", err)
		return nil
	}

	redisKey := fmt.Sprintf("ts:policy:%s", policyID)
	rCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := x.redisClient.Set(rCtx, redisKey, string(tsJSON), 24*time.Hour).Err(); err != nil {
		log.Printf("Warning: failed to write TS data to Redis: %v", err)
	}

	// Report ENFORCED to A1 Mediator (via Redis status key)
	reason := fmt.Sprintf("TS threshold %d%% applied for policy %s", threshold, policyID)
	x.reportPolicyStatus(20008, policyID, true, reason)
	tsPoliciesApplied.WithLabelValues("success").Inc()

	policiesProcessed.WithLabelValues("20008", "ricxapp", "ricxapp-scaling", "success").Inc()
	return nil
}

// ── A1 REST Helpers ─────────────────────────────────────────────────

func (x *ScalingXApp) listPolicies(policyTypeID string) ([]string, error) {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("LIST", policyTypeID).Observe(time.Since(start).Seconds())
	}()

	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%s/policies", x.a1URL, policyTypeID)
	resp, err := x.httpClient.Get(url)
	if err != nil {
		a1Requests.WithLabelValues("GET", policyTypeID, "error").Inc()
		return nil, fmt.Errorf("failed to list policies for type %s: %v", policyTypeID, err)
	}
	defer resp.Body.Close()

	a1Requests.WithLabelValues("GET", policyTypeID, strconv.Itoa(resp.StatusCode)).Inc()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d listing policies for type %s", resp.StatusCode, policyTypeID)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var policyIDs []string
	if err := json.Unmarshal(body, &policyIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy IDs: %v", err)
	}
	return policyIDs, nil
}

func (x *ScalingXApp) getPolicy(policyTypeID, policyID string) ([]byte, error) {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("GET", policyTypeID).Observe(time.Since(start).Seconds())
	}()

	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%s/policies/%s", x.a1URL, policyTypeID, policyID)
	resp, err := x.httpClient.Get(url)
	if err != nil {
		a1Requests.WithLabelValues("GET", policyTypeID, "error").Inc()
		return nil, fmt.Errorf("failed to get policy %s: %v", policyID, err)
	}
	defer resp.Body.Close()

	a1Requests.WithLabelValues("GET", policyTypeID, strconv.Itoa(resp.StatusCode)).Inc()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for policy %s", resp.StatusCode, policyID)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
}

// ── Status Reporting ────────────────────────────────────────────────

func (x *ScalingXApp) reportPolicyStatus(policyTypeID int, policyID string, enforced bool, reason string) {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("REDIS_SET", strconv.Itoa(policyTypeID)).Observe(time.Since(start).Seconds())
	}()

	status := PolicyStatus{
		EnforcementStatus: "NOT_ENFORCED",
		EnforcementReason: reason,
	}
	if enforced {
		status.EnforcementStatus = "ENFORCED"
	}

	statusJSON, err := json.Marshal(status)
	if err != nil {
		log.Printf("Failed to marshal policy status for %s: %v", policyID, err)
		policyStatusReports.WithLabelValues(strconv.Itoa(policyTypeID), status.EnforcementStatus, "marshal_error").Inc()
		return
	}

	redisKey := formatRedisKey(policyTypeID, policyID)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = x.redisClient.Set(ctx, redisKey, string(statusJSON), 24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to write policy status to Redis for %s: %v", policyID, err)
		policyStatusReports.WithLabelValues(strconv.Itoa(policyTypeID), status.EnforcementStatus, "redis_error").Inc()
		return
	}

	log.Printf("Policy status: %s -> %s (key: %s)", policyID, status.EnforcementStatus, redisKey)
	policyStatusReports.WithLabelValues(strconv.Itoa(policyTypeID), status.EnforcementStatus, "success").Inc()
}

// ── Main ────────────────────────────────────────────────────────────

func main() {
	xapp, err := NewScalingXApp()
	if err != nil {
		log.Fatalf("Failed to create Scaling xApp: %v", err)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			if err := xapp.checkRedisHealth(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Redis unhealthy: " + err.Error()))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Printf("Metrics server listening on :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := xapp.Start(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("Scaling xApp stopped: %v", err)
	}
	log.Printf("Scaling xApp shut down gracefully")
}
