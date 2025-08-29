package chaos

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InjectionType defines the type of failure injection.
type InjectionType string

const (
	// InjectionTypeNetworkLatency holds injectiontypenetworklatency value.
	InjectionTypeNetworkLatency InjectionType = "network-latency"
	// InjectionTypeNetworkLoss holds injectiontypenetworkloss value.
	InjectionTypeNetworkLoss InjectionType = "network-loss"
	// InjectionTypeNetworkPartition holds injectiontypenetworkpartition value.
	InjectionTypeNetworkPartition InjectionType = "network-partition"
	// InjectionTypePodKill holds injectiontypepodkill value.
	InjectionTypePodKill InjectionType = "pod-kill"
	// InjectionTypePodEviction holds injectiontypepodeviction value.
	InjectionTypePodEviction InjectionType = "pod-eviction"
	// InjectionTypeCPUStress holds injectiontypecpustress value.
	InjectionTypeCPUStress InjectionType = "cpu-stress"
	// InjectionTypeMemoryStress holds injectiontypememorystress value.
	InjectionTypeMemoryStress InjectionType = "memory-stress"
	// InjectionTypeDiskIOStress holds injectiontypediskiostress value.
	InjectionTypeDiskIOStress InjectionType = "disk-io-stress"
	// InjectionTypeProcessKill holds injectiontypeprocesskill value.
	InjectionTypeProcessKill InjectionType = "process-kill"
	// InjectionTypeHTTPFault holds injectiontypehttpfault value.
	InjectionTypeHTTPFault InjectionType = "http-fault"
	// InjectionTypeDNSFault holds injectiontypednsfault value.
	InjectionTypeDNSFault InjectionType = "dns-fault"
	// InjectionTypeTimeSkew holds injectiontypetimeskew value.
	InjectionTypeTimeSkew InjectionType = "time-skew"
)

// InjectionResult contains the result of a failure injection.
type InjectionResult struct {
	InjectionID   string
	Type          InjectionType
	Target        string
	StartTime     time.Time
	Status        string
	AffectedPods  []string
	AffectedNodes []string
	Metadata      map[string]interface{}
}

// ActiveInjection tracks an active failure injection.
type ActiveInjection struct {
	ID         string
	Type       InjectionType
	Experiment *Experiment
	Target     InjectionTarget
	StartTime  time.Time
	StopFunc   func() error
	Metadata   map[string]interface{}
}

// InjectionTarget specifies what to target for injection.
type InjectionTarget struct {
	Namespace string
	Pods      []corev1.Pod
	Services  []corev1.Service
	Nodes     []corev1.Node
}

// FailureInjector handles controlled failure injection.
type FailureInjector struct {
	client           client.Client
	kubeClient       kubernetes.Interface
	logger           *zap.Logger
	activeInjections sync.Map
	tcImage          string // Traffic control sidecar image
}

// NewFailureInjector creates a new failure injector.
func NewFailureInjector(client client.Client, kubeClient kubernetes.Interface, logger *zap.Logger) *FailureInjector {
	return &FailureInjector{
		client:     client,
		kubeClient: kubeClient,
		logger:     logger,
		tcImage:    "gaiadocker/iproute2:latest",
	}
}

// InjectFailure injects a failure based on experiment configuration.
func (f *FailureInjector) InjectFailure(ctx context.Context, experiment *Experiment) (*InjectionResult, error) {
	f.logger.Info("Injecting failure",
		zap.String("type", string(experiment.Type)),
		zap.String("target", experiment.Target.Namespace))

	// Get target resources.
	target, err := f.getInjectionTarget(ctx, experiment)
	if err != nil {
		return nil, fmt.Errorf("failed to get injection target: %w", err)
	}

	// Create injection result.
	result := &InjectionResult{
		InjectionID: fmt.Sprintf("inj-%s-%d", experiment.ID, time.Now().Unix()),
		Type:        f.mapExperimentTypeToInjection(experiment.Type),
		Target:      experiment.Target.Namespace,
		StartTime:   time.Now(),
		Status:      "active",
		Metadata:    make(map[string]interface{}),
	}

	// Inject based on experiment type.
	switch experiment.Type {
	case ExperimentTypeNetwork:
		err = f.injectNetworkChaos(ctx, experiment, target, result)
	case ExperimentTypePod:
		err = f.injectPodChaos(ctx, experiment, target, result)
	case ExperimentTypeResource:
		err = f.injectResourceChaos(ctx, experiment, target, result)
	case ExperimentTypeDatabase:
		err = f.injectDatabaseChaos(ctx, experiment, target, result)
	case ExperimentTypeExternal:
		err = f.injectExternalServiceChaos(ctx, experiment, target, result)
	case ExperimentTypeLoad:
		err = f.injectLoadChaos(ctx, experiment, target, result)
	case ExperimentTypeDependency:
		err = f.injectDependencyChaos(ctx, experiment, target, result)
	case ExperimentTypeComposite:
		err = f.injectCompositeChaos(ctx, experiment, target, result)
	default:
		err = fmt.Errorf("unsupported experiment type: %s", experiment.Type)
	}

	if err != nil {
		result.Status = "failed"
		return result, err
	}

	// Register active injection.
	f.registerActiveInjection(result.InjectionID, experiment, target)

	return result, nil
}

// getInjectionTarget identifies target resources for injection.
func (f *FailureInjector) getInjectionTarget(ctx context.Context, experiment *Experiment) (*InjectionTarget, error) {
	target := &InjectionTarget{
		Namespace: experiment.Target.Namespace,
	}

	// Get pods matching label selector.
	if len(experiment.Target.LabelSelector) > 0 {
		podList := &corev1.PodList{}
		labelSelector := labels.SelectorFromSet(experiment.Target.LabelSelector)
		if err := f.client.List(ctx, podList,
			client.InNamespace(experiment.Target.Namespace),
			client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
			return nil, fmt.Errorf("failed to list pods: %w", err)
		}

		// Apply blast radius limits.
		maxPods := experiment.BlastRadius.MaxPods
		if maxPods > 0 && len(podList.Items) > maxPods {
			// Randomly select pods up to the limit.
			rand.Shuffle(len(podList.Items), func(i, j int) {
				podList.Items[i], podList.Items[j] = podList.Items[j], podList.Items[i]
			})
			target.Pods = podList.Items[:maxPods]
		} else {
			target.Pods = podList.Items
		}
	}

	// Get specific pods if listed.
	for _, podName := range experiment.Target.Pods {
		pod := &corev1.Pod{}
		if err := f.client.Get(ctx, client.ObjectKey{
			Namespace: experiment.Target.Namespace,
			Name:      podName,
		}, pod); err == nil {
			target.Pods = append(target.Pods, *pod)
		}
	}

	// Get services.
	for _, svcName := range experiment.Target.Services {
		svc := &corev1.Service{}
		if err := f.client.Get(ctx, client.ObjectKey{
			Namespace: experiment.Target.Namespace,
			Name:      svcName,
		}, svc); err == nil {
			target.Services = append(target.Services, *svc)
		}
	}

	// Get nodes if specified.
	if len(experiment.Target.Nodes) > 0 {
		for _, nodeName := range experiment.Target.Nodes {
			node := &corev1.Node{}
			if err := f.client.Get(ctx, client.ObjectKey{Name: nodeName}, node); err == nil {
				target.Nodes = append(target.Nodes, *node)
			}
		}
	}

	return target, nil
}

// injectNetworkChaos injects network-related failures.
func (f *FailureInjector) injectNetworkChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	params := experiment.Parameters

	switch params["latency"] {
	case "":
		// Packet loss injection.
		if lossStr, exists := params["loss"]; exists {
			return f.injectPacketLoss(ctx, target, lossStr, result)
		}
		// Network partition.
		if _, exists := params["target_namespace"]; exists {
			return f.injectNetworkPartition(ctx, target, params, result)
		}
	default:
		// Latency injection.
		return f.injectNetworkLatency(ctx, target, params, result)
	}

	return fmt.Errorf("no network chaos parameters specified")
}

// injectNetworkLatency injects network latency using tc (traffic control).
func (f *FailureInjector) injectNetworkLatency(ctx context.Context, target *InjectionTarget, params map[string]string, result *InjectionResult) error {
	latency := params["latency"]
	jitter := params["jitter"]
	correlation := params["correlation"]

	for _, pod := range target.Pods {
		// Skip if pod is not running.
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		// Execute tc commands to add latency.
		commands := []string{
			// Add root qdisc.
			fmt.Sprintf("tc qdisc add dev eth0 root handle 1: prio"),
			// Add netem qdisc with latency.
			fmt.Sprintf("tc qdisc add dev eth0 parent 1:3 handle 30: netem delay %s %s %s%%",
				latency, jitter, correlation),
			// Add filter to apply to all traffic.
			fmt.Sprintf("tc filter add dev eth0 protocol ip parent 1:0 prio 3 u32 match ip dst 0.0.0.0/0 flowid 1:3"),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject latency in pod",
					zap.String("pod", pod.Name),
					zap.Error(err))
				// Continue with other pods.
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["latency"] = latency
	result.Metadata["jitter"] = jitter

	return nil
}

// injectPacketLoss injects packet loss using tc.
func (f *FailureInjector) injectPacketLoss(ctx context.Context, target *InjectionTarget, lossPercent string, result *InjectionResult) error {
	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		commands := []string{
			"tc qdisc add dev eth0 root handle 1: prio",
			fmt.Sprintf("tc qdisc add dev eth0 parent 1:3 handle 30: netem loss %s%%", lossPercent),
			"tc filter add dev eth0 protocol ip parent 1:0 prio 3 u32 match ip dst 0.0.0.0/0 flowid 1:3",
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject packet loss in pod",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["packet_loss"] = lossPercent
	return nil
}

// injectNetworkPartition creates network partition between namespaces.
func (f *FailureInjector) injectNetworkPartition(ctx context.Context, target *InjectionTarget, params map[string]string, result *InjectionResult) error {
	targetNamespace := params["target_namespace"]
	direction := params["direction"] // in, out, both

	// Create NetworkPolicy to block traffic.
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("chaos-partition-%s", result.InjectionID),
			Namespace: target.Namespace,
			Labels: map[string]string{
				"chaos.injection": "network-partition",
				"chaos.id":        result.InjectionID,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{},
			},
			PolicyTypes: []networkingv1.PolicyType{},
		},
	}

	// Configure policy based on direction.
	switch direction {
	case "in":
		networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
		networkPolicy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
			{
				From: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "name",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{targetNamespace},
								},
							},
						},
					},
				},
			},
		}
	case "out":
		networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
		networkPolicy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "name",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{targetNamespace},
								},
							},
						},
					},
				},
			},
		}
	case "both":
		networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes,
			networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress)
	}

	// Create the network policy.
	if err := f.client.Create(ctx, networkPolicy); err != nil {
		return fmt.Errorf("failed to create network policy: %w", err)
	}

	result.Metadata["partition_type"] = "network-policy"
	result.Metadata["target_namespace"] = targetNamespace
	result.Metadata["direction"] = direction

	// Store cleanup function.
	f.storeCleanupFunc(result.InjectionID, func() error {
		return f.client.Delete(context.Background(), networkPolicy)
	})

	return nil
}

// injectPodChaos injects pod-related failures.
func (f *FailureInjector) injectPodChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	mode := experiment.Parameters["mode"]

	switch mode {
	case "random":
		return f.killRandomPods(ctx, target, result)
	case "fixed":
		count, _ := strconv.Atoi(experiment.Parameters["count"])
		return f.killFixedPods(ctx, target, count, result)
	case "percentage":
		percent, _ := strconv.ParseFloat(experiment.Parameters["percent"], 64)
		return f.killPercentagePods(ctx, target, percent, result)
	default:
		return f.killRandomPods(ctx, target, result)
	}
}

// killRandomPods randomly kills pods.
func (f *FailureInjector) killRandomPods(ctx context.Context, target *InjectionTarget, result *InjectionResult) error {
	if len(target.Pods) == 0 {
		return fmt.Errorf("no pods to kill")
	}

	// Select a random pod.
	pod := target.Pods[rand.Intn(len(target.Pods))]

	// Delete the pod.
	if err := f.client.Delete(ctx, &pod); err != nil {
		return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
	}

	result.AffectedPods = append(result.AffectedPods, pod.Name)
	f.logger.Info("Killed pod", zap.String("pod", pod.Name))

	return nil
}

// killFixedPods kills a fixed number of pods.
func (f *FailureInjector) killFixedPods(ctx context.Context, target *InjectionTarget, count int, result *InjectionResult) error {
	if count > len(target.Pods) {
		count = len(target.Pods)
	}

	// Randomly select pods to kill.
	rand.Shuffle(len(target.Pods), func(i, j int) {
		target.Pods[i], target.Pods[j] = target.Pods[j], target.Pods[i]
	})

	for i := 0; i < count; i++ {
		pod := target.Pods[i]
		if err := f.client.Delete(ctx, &pod); err != nil {
			f.logger.Warn("Failed to delete pod",
				zap.String("pod", pod.Name),
				zap.Error(err))
			continue
		}
		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	return nil
}

// killPercentagePods kills a percentage of pods.
func (f *FailureInjector) killPercentagePods(ctx context.Context, target *InjectionTarget, percent float64, result *InjectionResult) error {
	// Safe integer conversion with bounds checking
	countFloat := float64(len(target.Pods)) * percent / 100
	count := safeFloatToInt(countFloat)
	if count == 0 && len(target.Pods) > 0 {
		count = 1
	}
	return f.killFixedPods(ctx, target, count, result)
}

// injectResourceChaos injects resource-related stress.
func (f *FailureInjector) injectResourceChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	if cpuPercent, exists := experiment.Parameters["cpu_percent"]; exists {
		return f.injectCPUStress(ctx, target, cpuPercent, result)
	}

	if memoryMB, exists := experiment.Parameters["memory_mb"]; exists {
		return f.injectMemoryStress(ctx, target, memoryMB, result)
	}

	if ioWorkers, exists := experiment.Parameters["io_workers"]; exists {
		return f.injectDiskIOStress(ctx, target, ioWorkers, result)
	}

	return fmt.Errorf("no resource chaos parameters specified")
}

// injectCPUStress injects CPU stress using stress-ng.
func (f *FailureInjector) injectCPUStress(ctx context.Context, target *InjectionTarget, cpuPercent string, result *InjectionResult) error {
	workers := "2"
	timeout := "60s"

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		// Install stress-ng if not present and run CPU stress.
		commands := []string{
			"apt-get update && apt-get install -y stress-ng || yum install -y stress-ng",
			fmt.Sprintf("stress-ng --cpu %s --cpu-load %s --timeout %s &", workers, cpuPercent, timeout),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject CPU stress in pod",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["cpu_stress"] = cpuPercent
	return nil
}

// injectMemoryStress injects memory stress.
func (f *FailureInjector) injectMemoryStress(ctx context.Context, target *InjectionTarget, memoryMB string, result *InjectionResult) error {
	workers := "1"
	timeout := "60s"

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		commands := []string{
			"apt-get update && apt-get install -y stress-ng || yum install -y stress-ng",
			fmt.Sprintf("stress-ng --vm %s --vm-bytes %sM --timeout %s &", workers, memoryMB, timeout),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject memory stress in pod",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["memory_stress"] = memoryMB
	return nil
}

// injectDiskIOStress injects disk I/O stress.
func (f *FailureInjector) injectDiskIOStress(ctx context.Context, target *InjectionTarget, ioWorkers string, result *InjectionResult) error {
	timeout := "60s"

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		commands := []string{
			"apt-get update && apt-get install -y stress-ng || yum install -y stress-ng",
			fmt.Sprintf("stress-ng --io %s --timeout %s &", ioWorkers, timeout),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject I/O stress in pod",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["io_stress"] = ioWorkers
	return nil
}

// injectDatabaseChaos injects database-related failures.
func (f *FailureInjector) injectDatabaseChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	failureType := experiment.Parameters["failure_type"]

	switch failureType {
	case "connection_drop":
		return f.injectDatabaseConnectionDrop(ctx, target, experiment.Parameters, result)
	case "slow_query":
		return f.injectDatabaseSlowQuery(ctx, target, experiment.Parameters, result)
	default:
		return fmt.Errorf("unsupported database failure type: %s", failureType)
	}
}

// injectDatabaseConnectionDrop simulates database connection drops.
func (f *FailureInjector) injectDatabaseConnectionDrop(ctx context.Context, target *InjectionTarget, params map[string]string, result *InjectionResult) error {
	dbPort := params["port"]
	failureRate := params["failure_rate"]

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		// Use iptables to randomly drop database connections.
		commands := []string{
			fmt.Sprintf("iptables -A OUTPUT -p tcp --dport %s -m statistic --mode random --probability 0.%s -j DROP",
				dbPort, failureRate),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject database connection drop",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["db_failure_type"] = "connection_drop"
	result.Metadata["failure_rate"] = failureRate

	// Store cleanup function.
	f.storeCleanupFunc(result.InjectionID, func() error {
		for _, pod := range target.Pods {
			cmd := fmt.Sprintf("iptables -D OUTPUT -p tcp --dport %s -m statistic --mode random --probability 0.%s -j DROP",
				dbPort, failureRate)
			f.execInPod(context.Background(), &pod, cmd)
		}
		return nil
	})

	return nil
}

// injectDatabaseSlowQuery simulates slow database queries.
func (f *FailureInjector) injectDatabaseSlowQuery(ctx context.Context, target *InjectionTarget, params map[string]string, result *InjectionResult) error {
	delayMS := params["query_delay_ms"]

	// This would typically be implemented using a database proxy or middleware.
	// For demonstration, we'll use tc to add latency to database port.
	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		commands := []string{
			"tc qdisc add dev eth0 root handle 1: prio",
			fmt.Sprintf("tc qdisc add dev eth0 parent 1:3 handle 30: netem delay %sms", delayMS),
			fmt.Sprintf("tc filter add dev eth0 protocol ip parent 1:0 prio 3 u32 match ip dport 5432 0xffff flowid 1:3"),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject database slow query",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["query_delay"] = delayMS
	return nil
}

// injectExternalServiceChaos injects external service failures.
func (f *FailureInjector) injectExternalServiceChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	apiEndpoint := experiment.Parameters["api_endpoint"]
	failureType := experiment.Parameters["failure_type"]

	switch failureType {
	case "timeout":
		return f.injectAPITimeout(ctx, target, apiEndpoint, experiment.Parameters, result)
	case "error":
		return f.injectAPIError(ctx, target, apiEndpoint, experiment.Parameters, result)
	default:
		return fmt.Errorf("unsupported external service failure type: %s", failureType)
	}
}

// injectAPITimeout simulates API timeouts.
func (f *FailureInjector) injectAPITimeout(ctx context.Context, target *InjectionTarget, endpoint string, params map[string]string, result *InjectionResult) error {
	timeoutMS := params["timeout_ms"]

	// Map endpoint to IP/domain.
	endpointMap := map[string]string{
		"openai":   "api.openai.com",
		"weaviate": "weaviate.svc.cluster.local",
	}

	domain, exists := endpointMap[endpoint]
	if !exists {
		domain = endpoint
	}

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		// Add latency to simulate timeout.
		commands := []string{
			"tc qdisc add dev eth0 root handle 1: prio",
			fmt.Sprintf("tc qdisc add dev eth0 parent 1:3 handle 30: netem delay %sms", timeoutMS),
			fmt.Sprintf("tc filter add dev eth0 protocol ip parent 1:0 prio 3 u32 match ip dst $(dig +short %s | head -1) flowid 1:3", domain),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject API timeout",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["api_endpoint"] = endpoint
	result.Metadata["timeout"] = timeoutMS

	return nil
}

// injectAPIError simulates API errors.
func (f *FailureInjector) injectAPIError(ctx context.Context, target *InjectionTarget, endpoint string, params map[string]string, result *InjectionResult) error {
	errorRate := params["failure_rate"]

	// This would typically be implemented using a service mesh or proxy.
	// For demonstration, we'll drop packets to simulate errors.
	endpointMap := map[string]string{
		"openai":   "api.openai.com",
		"weaviate": "weaviate.svc.cluster.local",
	}

	domain, exists := endpointMap[endpoint]
	if !exists {
		domain = endpoint
	}

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		commands := []string{
			fmt.Sprintf("iptables -A OUTPUT -d $(dig +short %s | head -1) -m statistic --mode random --probability 0.%s -j DROP",
				domain, errorRate),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject API error",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["api_endpoint"] = endpoint
	result.Metadata["error_rate"] = errorRate

	return nil
}

// injectLoadChaos injects high load.
func (f *FailureInjector) injectLoadChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	intentsPerMinute := experiment.Parameters["intents_per_minute"]

	// Create a job to generate load.
	job := f.createLoadGeneratorJob(experiment.Target.Namespace, intentsPerMinute, experiment.Duration)

	if err := f.client.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create load generator job: %w", err)
	}

	result.Metadata["load_generator"] = job.Name
	result.Metadata["intents_per_minute"] = intentsPerMinute

	// Store cleanup function.
	f.storeCleanupFunc(result.InjectionID, func() error {
		return f.client.Delete(context.Background(), job)
	})

	return nil
}

// createLoadGeneratorJob creates a job to generate load.
func (f *FailureInjector) createLoadGeneratorJob(namespace string, intentsPerMinute string, duration time.Duration) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "load-generator-",
			Namespace:    namespace,
			Labels: map[string]string{
				"chaos.injection": "load-generator",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "load-generator",
					Image: "curlimages/curl:latest",
					Command: []string{
						"/bin/sh",
						"-c",
						fmt.Sprintf(`
							rate=$((%s / 60))
							duration=%d
							end=$(($(date +%%s) + duration))
							while [ $(date +%%s) -lt $end ]; do
								curl -X POST http://intent-controller:8080/intents \
									-H "Content-Type: application/json" \
									-d '{"intent": "Deploy test AMF"}'
								sleep $((1 / rate))
							done
						`, intentsPerMinute, int(duration.Seconds())),
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
		},
	}
}

// injectDependencyChaos injects dependency failures.
func (f *FailureInjector) injectDependencyChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	// Implement based on specific dependency type.
	if strings.Contains(experiment.Name, "kubernetes-api") {
		return f.injectKubernetesAPIFailure(ctx, target, experiment.Parameters, result)
	}

	if strings.Contains(experiment.Name, "dns") {
		return f.injectDNSFailure(ctx, target, experiment.Parameters, result)
	}

	return fmt.Errorf("unsupported dependency type")
}

// injectKubernetesAPIFailure simulates Kubernetes API failures.
func (f *FailureInjector) injectKubernetesAPIFailure(ctx context.Context, target *InjectionTarget, params map[string]string, result *InjectionResult) error {
	// Add network latency to Kubernetes API server.
	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		commands := []string{
			"tc qdisc add dev eth0 root handle 1: prio",
			fmt.Sprintf("tc qdisc add dev eth0 parent 1:3 handle 30: netem delay %sms", params["delay_ms"]),
			"tc filter add dev eth0 protocol ip parent 1:0 prio 3 u32 match ip dport 443 0xffff flowid 1:3",
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject Kubernetes API failure",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	return nil
}

// injectDNSFailure simulates DNS failures.
func (f *FailureInjector) injectDNSFailure(ctx context.Context, target *InjectionTarget, params map[string]string, result *InjectionResult) error {
	failureRate := params["failure_rate"]

	for _, pod := range target.Pods {
		if pod.Status.Phase != corev1.PodPhase("Running") {
			continue
		}

		// Drop DNS packets randomly.
		commands := []string{
			fmt.Sprintf("iptables -A OUTPUT -p udp --dport 53 -m statistic --mode random --probability 0.%s -j DROP", failureRate),
		}

		for _, cmd := range commands {
			if err := f.execInPod(ctx, &pod, cmd); err != nil {
				f.logger.Warn("Failed to inject DNS failure",
					zap.String("pod", pod.Name),
					zap.Error(err))
			}
		}

		result.AffectedPods = append(result.AffectedPods, pod.Name)
	}

	result.Metadata["dns_failure_rate"] = failureRate

	// Store cleanup function.
	f.storeCleanupFunc(result.InjectionID, func() error {
		for _, pod := range target.Pods {
			cmd := fmt.Sprintf("iptables -D OUTPUT -p udp --dport 53 -m statistic --mode random --probability 0.%s -j DROP", failureRate)
			f.execInPod(context.Background(), &pod, cmd)
		}
		return nil
	})

	return nil
}

// injectCompositeChaos injects multiple failures simultaneously.
func (f *FailureInjector) injectCompositeChaos(ctx context.Context, experiment *Experiment, target *InjectionTarget, result *InjectionResult) error {
	scenarios := strings.Split(experiment.Parameters["scenarios"], ",")
	staggerStart := experiment.Parameters["stagger_start"] == "true"
	staggerDelay, _ := time.ParseDuration(experiment.Parameters["stagger_delay"])

	var wg sync.WaitGroup
	errors := make([]error, 0)
	var errorsMu sync.Mutex

	for i, scenario := range scenarios {
		wg.Add(1)
		go func(index int, scenarioName string) {
			defer wg.Done()

			if staggerStart && index > 0 {
				time.Sleep(time.Duration(index) * staggerDelay)
			}

			// Create sub-experiment for each scenario.
			subExp := *experiment
			subExp.ID = fmt.Sprintf("%s-%s", experiment.ID, scenarioName)

			// Inject based on scenario type.
			var err error
			switch scenarioName {
			case "network-latency":
				subExp.Type = ExperimentTypeNetwork
				subExp.Parameters = map[string]string{
					"latency": "100ms",
					"jitter":  "10ms",
				}
			case "pod-failure":
				subExp.Type = ExperimentTypePod
				subExp.Parameters = map[string]string{
					"mode": "random",
				}
			case "cpu-stress":
				subExp.Type = ExperimentTypeResource
				subExp.Parameters = map[string]string{
					"cpu_percent": "80",
				}
			}

			subResult, err := f.InjectFailure(ctx, &subExp)
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
			} else {
				result.AffectedPods = append(result.AffectedPods, subResult.AffectedPods...)
			}
		}(i, strings.TrimSpace(scenario))
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("some composite injections failed: %v", errors)
	}

	result.Metadata["scenarios"] = scenarios
	return nil
}

// execInPod executes a command in a pod.
func (f *FailureInjector) execInPod(ctx context.Context, pod *corev1.Pod, command string) error {
	req := f.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   []string{"/bin/sh", "-c", command},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, metav1.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(nil, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// StopFailure stops a specific failure injection.
func (f *FailureInjector) StopFailure(ctx context.Context, injectionID string) error {
	f.logger.Info("Stopping failure injection", zap.String("id", injectionID))

	if injection, exists := f.activeInjections.Load(injectionID); exists {
		if activeInj, ok := injection.(*ActiveInjection); ok {
			if activeInj.StopFunc != nil {
				if err := activeInj.StopFunc(); err != nil {
					return fmt.Errorf("failed to stop injection: %w", err)
				}
			}
			f.activeInjections.Delete(injectionID)
		}
	}

	return nil
}

// StopAllFailures stops all active failure injections for an experiment.
func (f *FailureInjector) StopAllFailures(ctx context.Context, experimentID string) error {
	f.logger.Info("Stopping all failures for experiment", zap.String("experiment", experimentID))

	var errors []error
	f.activeInjections.Range(func(key, value interface{}) bool {
		if activeInj, ok := value.(*ActiveInjection); ok {
			if activeInj.Experiment != nil && activeInj.Experiment.ID == experimentID {
				if err := f.StopFailure(ctx, key.(string)); err != nil {
					errors = append(errors, err)
				}
			}
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some injections: %v", errors)
	}

	return nil
}

// registerActiveInjection registers an active injection.
func (f *FailureInjector) registerActiveInjection(id string, experiment *Experiment, target *InjectionTarget) {
	f.activeInjections.Store(id, &ActiveInjection{
		ID:         id,
		Type:       f.mapExperimentTypeToInjection(experiment.Type),
		Experiment: experiment,
		Target: InjectionTarget{
			Namespace: target.Namespace,
			Pods:      target.Pods,
			Services:  target.Services,
			Nodes:     target.Nodes,
		},
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	})
}

// storeCleanupFunc stores a cleanup function for an injection.
func (f *FailureInjector) storeCleanupFunc(injectionID string, cleanupFunc func() error) {
	if injection, exists := f.activeInjections.Load(injectionID); exists {
		if activeInj, ok := injection.(*ActiveInjection); ok {
			activeInj.StopFunc = cleanupFunc
		}
	}
}

// mapExperimentTypeToInjection maps experiment type to injection type.
func (f *FailureInjector) mapExperimentTypeToInjection(expType ExperimentType) InjectionType {
	switch expType {
	case ExperimentTypeNetwork:
		return InjectionTypeNetworkLatency
	case ExperimentTypePod:
		return InjectionTypePodKill
	case ExperimentTypeResource:
		return InjectionTypeCPUStress
	case ExperimentTypeDatabase:
		return InjectionTypeHTTPFault
	case ExperimentTypeExternal:
		return InjectionTypeHTTPFault
	case ExperimentTypeDependency:
		return InjectionTypeDNSFault
	default:
		return InjectionTypeNetworkLatency
	}
}

// GetActiveInjections returns all active injections.
func (f *FailureInjector) GetActiveInjections() []*ActiveInjection {
	var injections []*ActiveInjection
	f.activeInjections.Range(func(key, value interface{}) bool {
		if inj, ok := value.(*ActiveInjection); ok {
			injections = append(injections, inj)
		}
		return true
	})
	return injections
}

// CleanupAllInjections cleans up all active injections.
func (f *FailureInjector) CleanupAllInjections(ctx context.Context) error {
	f.logger.Info("Cleaning up all active injections")

	var errors []error
	f.activeInjections.Range(func(key, value interface{}) bool {
		if err := f.StopFailure(ctx, key.(string)); err != nil {
			errors = append(errors, err)
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to cleanup some injections: %v", errors)
	}

	return nil
}
