# Prometheus Metrics 實現指南

**當前狀態**: go.mod 已更新，準備添加 metrics 代碼

---

## 📋 已完成

✅ **Task #60**: A1 Policy 清理（252 → 6 policies）
✅ `go.mod`: 添加 `prometheus/client_golang v1.18.0` 依賴
✅ 創建任務進度文檔

---

## 🔄 下一步實現（Task #61）

### 1. 更新 main.go - 添加 Metrics 定義

在 `main.go` 頂部添加：

```go
import (
	// ... existing imports ...
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
```

### 2. 添加 Metrics HTTP 服務器

在 `main()` 函數中啟動 metrics 服務器：

```go
func main() {
	// ... existing code ...

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Metrics server listening on :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// ... existing code ...
}
```

### 3. 記錄 Metrics - pollAndExecutePolicies()

```go
func (x *ScalingXApp) pollAndExecutePolicies(ctx context.Context) error {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("GET").Observe(time.Since(start).Seconds())
	}()

	// Get all policies
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/100/policies", x.a1URL)
	resp, err := http.Get(url)
	if err != nil {
		a1Requests.WithLabelValues("GET", "error").Inc()
		return fmt.Errorf("failed to get policies: %v", err)
	}
	defer resp.Body.Close()

	a1Requests.WithLabelValues("GET", strconv.Itoa(resp.StatusCode)).Inc()

	// ... existing code ...

	activePolicies.Set(float64(len(policyIDs)))
	lastPollTimestamp.SetToCurrentTime()

	// ... existing code ...
}
```

### 4. 記錄 Metrics - scaleDeployment()

```go
func (x *ScalingXApp) scaleDeployment(ctx context.Context, spec ScalingSpec) error {
	start := time.Now()
	defer func() {
		scalingDuration.WithLabelValues(spec.Namespace, spec.Target).Observe(time.Since(start).Seconds())
	}()

	// ... existing scaling logic ...

	if err != nil {
		policiesProcessed.WithLabelValues(spec.Namespace, spec.Target, "failed").Inc()
		return err
	}

	policiesProcessed.WithLabelValues(spec.Namespace, spec.Target, "success").Inc()
	return nil
}
```

### 5. 更新 deployment.yaml

添加 metrics 端口和 annotations：

```yaml
spec:
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "2112"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: scaling-xapp
        ports:
        - name: metrics
          containerPort: 2112
          protocol: TCP
```

### 6. 創建 Service 更新

在 `deployment.yaml` 的 Service 部分添加 metrics 端口：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: service-ricxapp-scaling-metrics
  namespace: ricxapp
spec:
  selector:
    app: ricxapp-scaling
  ports:
  - name: metrics
    port: 2112
    targetPort: 2112
    protocol: TCP
```

### 7. 創建 ServiceMonitor (可選)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: scaling-xapp
  namespace: ricxapp
  labels:
    app: ricxapp-scaling
spec:
  selector:
    matchLabels:
      app: ricxapp-scaling
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

---

## 🔧 構建和部署步驟

```bash
cd deployments/xapps/scaling-xapp

# 1. 下載依賴
go mod tidy

# 2. 構建新映像
buildah bud -t scaling-xapp:v1.1-metrics .

# 3. 導出到 containerd
buildah push localhost/scaling-xapp:v1.1-metrics oci-archive:/tmp/scaling-xapp-metrics.tar:scaling-xapp:v1.1-metrics
sudo ctr -n k8s.io images import /tmp/scaling-xapp-metrics.tar
sudo ctr -n k8s.io images tag scaling-xapp:v1.1-metrics docker.io/library/scaling-xapp:latest

# 4. 更新 deployment
kubectl apply -f deployment.yaml
kubectl delete pod -n ricxapp -l app=ricxapp-scaling

# 5. 驗證 metrics
kubectl port-forward -n ricxapp deployment/ricxapp-scaling 2112:2112 &
curl http://localhost:2112/metrics | grep scaling_xapp
```

---

## 📊 驗證 Metrics

預期看到的 metrics：

```
# HELP scaling_xapp_policies_processed_total Total number of policies processed
# TYPE scaling_xapp_policies_processed_total counter
scaling_xapp_policies_processed_total{deployment="nf-sim",namespace="ran-a",result="success"} 15

# HELP scaling_xapp_active_policies Number of active policies
# TYPE scaling_xapp_active_policies gauge
scaling_xapp_active_policies 6

# HELP scaling_xapp_a1_request_duration_seconds A1 API request duration distribution
# TYPE scaling_xapp_a1_request_duration_seconds histogram
scaling_xapp_a1_request_duration_seconds_bucket{method="GET",le="0.005"} 0
scaling_xapp_a1_request_duration_seconds_bucket{method="GET",le="0.01"} 5
...
```

---

## 🎯 Task #62 準備

Task #61 完成後，實現 Policy Status Reporting：

1. 添加 `reportPolicyStatus()` 函數
2. 在 `scaleDeployment()` 成功/失敗時調用
3. 測試 A1 Mediator 接收狀態報告

---

**預計完成時間**: 1-2 小時（包括測試）
