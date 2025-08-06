package security

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NetworkPolicyManager implements zero-trust networking for O-RAN components
type NetworkPolicyManager struct {
	client    client.Client
	namespace string
}

// NewNetworkPolicyManager creates a new network policy manager
func NewNetworkPolicyManager(client client.Client, namespace string) *NetworkPolicyManager {
	return &NetworkPolicyManager{
		client:    client,
		namespace: namespace,
	}
}

// PolicyType defines the type of network policy
type PolicyType string

const (
	PolicyTypeDenyAll          PolicyType = "deny-all"
	PolicyTypeAllowIngress     PolicyType = "allow-ingress"
	PolicyTypeAllowEgress      PolicyType = "allow-egress"
	PolicyTypeComponentSpecific PolicyType = "component-specific"
	PolicyTypeORANInterface    PolicyType = "oran-interface"
)

// CreateDefaultDenyAllPolicy creates a default deny-all network policy (zero-trust baseline)
func (m *NetworkPolicyManager) CreateDefaultDenyAllPolicy(ctx context.Context) error {
	logger := log.FromContext(ctx)
	
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-all",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "network-policy",
				"security.nephoran.io/type":   string(PolicyTypeDenyAll),
				"security.nephoran.io/ztna":   "enabled",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // Selects all pods in namespace
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			// Empty ingress and egress rules mean deny all
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
			Egress:  []networkingv1.NetworkPolicyEgressRule{},
		},
	}
	
	if err := m.client.Create(ctx, policy); err != nil {
		logger.Error(err, "Failed to create default deny-all policy")
		return fmt.Errorf("failed to create default deny-all policy: %w", err)
	}
	
	logger.Info("Created default deny-all network policy")
	return nil
}

// CreateControllerNetworkPolicy creates network policy for the operator controller
func (m *NetworkPolicyManager) CreateControllerNetworkPolicy(ctx context.Context) error {
	logger := log.FromContext(ctx)
	
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nephoran-controller-policy",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "controller",
				"security.nephoran.io/type":   string(PolicyTypeComponentSpecific),
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":      "nephoran-intent-operator",
					"app.kubernetes.io/component": "controller",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					// Allow webhook traffic from API server
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "kube-system",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 9443},
						},
					},
				},
				{
					// Allow metrics scraping from Prometheus
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": "prometheus",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Allow DNS resolution
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "kube-system",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"k8s-app": "kube-dns",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolUDP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
						},
					},
				},
				{
					// Allow Kubernetes API access
					To: []networkingv1.NetworkPolicyPeer{
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR: "10.0.0.0/8", // Adjust for your cluster
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
						},
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
						},
					},
				},
				{
					// Allow LLM service access
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name":      "nephoran-intent-operator",
									"app.kubernetes.io/component": "llm-processor",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8081},
						},
					},
				},
			},
		},
	}
	
	if err := m.client.Create(ctx, policy); err != nil {
		logger.Error(err, "Failed to create controller network policy")
		return fmt.Errorf("failed to create controller network policy: %w", err)
	}
	
	logger.Info("Created controller network policy")
	return nil
}

// CreateLLMServiceNetworkPolicy creates network policy for LLM service
func (m *NetworkPolicyManager) CreateLLMServiceNetworkPolicy(ctx context.Context) error {
	logger := log.FromContext(ctx)
	
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nephoran-llm-policy",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "llm-processor",
				"security.nephoran.io/type":   string(PolicyTypeComponentSpecific),
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":      "nephoran-intent-operator",
					"app.kubernetes.io/component": "llm-processor",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					// Allow traffic from controller
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name":      "nephoran-intent-operator",
									"app.kubernetes.io/component": "controller",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8081},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Allow DNS
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "kube-system",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"k8s-app": "kube-dns",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolUDP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
						},
					},
				},
				{
					// Allow OpenAI API access (HTTPS)
					To: []networkingv1.NetworkPolicyPeer{
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR: "0.0.0.0/0", // External access for OpenAI
								Except: []string{
									"10.0.0.0/8",
									"172.16.0.0/12",
									"192.168.0.0/16",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
						},
					},
				},
				{
					// Allow Weaviate vector database access
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": "weaviate",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
						},
					},
				},
			},
		},
	}
	
	if err := m.client.Create(ctx, policy); err != nil {
		logger.Error(err, "Failed to create LLM service network policy")
		return fmt.Errorf("failed to create LLM service network policy: %w", err)
	}
	
	logger.Info("Created LLM service network policy")
	return nil
}

// CreateORANInterfacePolicy creates network policies for O-RAN interfaces
func (m *NetworkPolicyManager) CreateORANInterfacePolicy(ctx context.Context, interfaceType string) error {
	logger := log.FromContext(ctx)
	
	var policy *networkingv1.NetworkPolicy
	
	switch interfaceType {
	case "A1":
		policy = m.createA1InterfacePolicy()
	case "O1":
		policy = m.createO1InterfacePolicy()
	case "O2":
		policy = m.createO2InterfacePolicy()
	case "E2":
		policy = m.createE2InterfacePolicy()
	default:
		return fmt.Errorf("unknown O-RAN interface type: %s", interfaceType)
	}
	
	if err := m.client.Create(ctx, policy); err != nil {
		logger.Error(err, "Failed to create O-RAN interface policy", "interface", interfaceType)
		return fmt.Errorf("failed to create O-RAN interface policy %s: %w", interfaceType, err)
	}
	
	logger.Info("Created O-RAN interface network policy", "interface", interfaceType)
	return nil
}

// createA1InterfacePolicy creates network policy for A1 interface (Non-RT RIC to Near-RT RIC)
func (m *NetworkPolicyManager) createA1InterfacePolicy() *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oran-a1-interface-policy",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "oran-a1",
				"security.nephoran.io/type":   string(PolicyTypeORANInterface),
				"oran.alliance/interface":     "A1",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"oran.alliance/component": "near-rt-ric",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"oran.alliance/component": "non-rt-ric",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8443}, // A1 interface port
						},
					},
				},
			},
		},
	}
}

// createO1InterfacePolicy creates network policy for O1 interface (FCAPS management)
func (m *NetworkPolicyManager) createO1InterfacePolicy() *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oran-o1-interface-policy",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "oran-o1",
				"security.nephoran.io/type":   string(PolicyTypeORANInterface),
				"oran.alliance/interface":     "O1",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"oran.alliance/managed": "true",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"oran.alliance/component": "smo",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 830}, // NETCONF
						},
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6513}, // NETCONF over TLS
						},
					},
				},
			},
		},
	}
}

// createO2InterfacePolicy creates network policy for O2 interface (Cloud infrastructure)
func (m *NetworkPolicyManager) createO2InterfacePolicy() *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oran-o2-interface-policy",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "oran-o2",
				"security.nephoran.io/type":   string(PolicyTypeORANInterface),
				"oran.alliance/interface":     "O2",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"oran.alliance/component": "o-cloud",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"oran.alliance/component": "smo",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443}, // O2 REST API
						},
					},
				},
			},
		},
	}
}

// createE2InterfacePolicy creates network policy for E2 interface (Near-RT RIC to E2 nodes)
func (m *NetworkPolicyManager) createE2InterfacePolicy() *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oran-e2-interface-policy",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "oran-e2",
				"security.nephoran.io/type":   string(PolicyTypeORANInterface),
				"oran.alliance/interface":     "E2",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"oran.alliance/e2-node": "true",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"oran.alliance/component": "near-rt-ric",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolSCTP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 36421}, // E2AP SCTP
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"oran.alliance/component": "near-rt-ric",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolSCTP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 36421},
						},
					},
				},
			},
		},
	}
}

// CreateExternalAccessPolicy creates policy for external service access
func (m *NetworkPolicyManager) CreateExternalAccessPolicy(ctx context.Context, serviceName string, allowedCIDRs []string) error {
	logger := log.FromContext(ctx)
	
	// Build IPBlock peers
	var peers []networkingv1.NetworkPolicyPeer
	for _, cidr := range allowedCIDRs {
		peers = append(peers, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: cidr,
			},
		})
	}
	
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-external-access", serviceName),
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": serviceName,
				"security.nephoran.io/type":   "external-access",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": serviceName,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: peers,
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
						},
					},
				},
			},
		},
	}
	
	if err := m.client.Create(ctx, policy); err != nil {
		logger.Error(err, "Failed to create external access policy", "service", serviceName)
		return fmt.Errorf("failed to create external access policy for %s: %w", serviceName, err)
	}
	
	logger.Info("Created external access policy", "service", serviceName)
	return nil
}

// ValidateNetworkPolicies validates that network policies are effective
func (m *NetworkPolicyManager) ValidateNetworkPolicies(ctx context.Context) (*NetworkPolicyValidationReport, error) {
	logger := log.FromContext(ctx)
	
	report := &NetworkPolicyValidationReport{
		Timestamp: metav1.Now(),
		Namespace: m.namespace,
		Issues:    []string{},
		Warnings:  []string{},
	}
	
	// List all network policies
	policies := &networkingv1.NetworkPolicyList{}
	if err := m.client.List(ctx, policies, client.InNamespace(m.namespace)); err != nil {
		return nil, fmt.Errorf("failed to list network policies: %w", err)
	}
	
	report.PolicyCount = len(policies.Items)
	
	// Check for default deny-all policy
	hasDenyAll := false
	for _, policy := range policies.Items {
		if policy.Name == "default-deny-all" {
			hasDenyAll = true
			break
		}
	}
	
	if !hasDenyAll {
		report.Issues = append(report.Issues, "Missing default deny-all policy (zero-trust baseline)")
	}
	
	// List all pods
	pods := &corev1.PodList{}
	if err := m.client.List(ctx, pods, client.InNamespace(m.namespace)); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	
	// Check pod coverage
	coveredPods := make(map[string]bool)
	for _, policy := range policies.Items {
		for _, pod := range pods.Items {
			if m.isPodSelected(pod, policy.Spec.PodSelector) {
				coveredPods[pod.Name] = true
			}
		}
	}
	
	report.CoveredPods = len(coveredPods)
	report.TotalPods = len(pods.Items)
	
	if report.CoveredPods < report.TotalPods {
		uncoveredCount := report.TotalPods - report.CoveredPods
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d pods not covered by network policies", uncoveredCount))
	}
	
	// Validate individual policies
	for _, policy := range policies.Items {
		// Check for overly permissive rules
		for _, ingress := range policy.Spec.Ingress {
			if len(ingress.From) == 0 {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Policy %s has ingress rule with no source restrictions", policy.Name))
			}
		}
		
		for _, egress := range policy.Spec.Egress {
			if len(egress.To) == 0 {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Policy %s has egress rule with no destination restrictions", policy.Name))
			}
			
			// Check for overly broad CIDR blocks
			for _, peer := range egress.To {
				if peer.IPBlock != nil && peer.IPBlock.CIDR == "0.0.0.0/0" {
					hasExceptions := len(peer.IPBlock.Except) > 0
					if !hasExceptions {
						report.Warnings = append(report.Warnings, fmt.Sprintf("Policy %s allows unrestricted egress to internet", policy.Name))
					}
				}
			}
		}
	}
	
	report.Compliant = len(report.Issues) == 0
	logger.Info("Network policy validation completed", 
		"compliant", report.Compliant,
		"policies", report.PolicyCount,
		"coverage", fmt.Sprintf("%d/%d", report.CoveredPods, report.TotalPods))
	
	return report, nil
}

// NetworkPolicyValidationReport contains network policy validation results
type NetworkPolicyValidationReport struct {
	Timestamp    metav1.Time
	Namespace    string
	Compliant    bool
	PolicyCount  int
	CoveredPods  int
	TotalPods    int
	Issues       []string
	Warnings     []string
}

// isPodSelected checks if a pod matches a label selector
func (m *NetworkPolicyManager) isPodSelected(pod corev1.Pod, selector metav1.LabelSelector) bool {
	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		// Empty selector matches all pods
		return true
	}
	
	// Check MatchLabels
	for key, value := range selector.MatchLabels {
		if podValue, exists := pod.Labels[key]; !exists || podValue != value {
			return false
		}
	}
	
	// Check MatchExpressions (simplified)
	for _, expr := range selector.MatchExpressions {
		podValue, exists := pod.Labels[expr.Key]
		
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			if !exists || !contains(expr.Values, podValue) {
				return false
			}
		case metav1.LabelSelectorOpNotIn:
			if exists && contains(expr.Values, podValue) {
				return false
			}
		case metav1.LabelSelectorOpExists:
			if !exists {
				return false
			}
		case metav1.LabelSelectorOpDoesNotExist:
			if exists {
				return false
			}
		}
	}
	
	return true
}

// EnforceZeroTrustNetworking applies comprehensive zero-trust networking
func (m *NetworkPolicyManager) EnforceZeroTrustNetworking(ctx context.Context) error {
	logger := log.FromContext(ctx)
	
	// Create default deny-all policy
	if err := m.CreateDefaultDenyAllPolicy(ctx); err != nil {
		return fmt.Errorf("failed to create deny-all policy: %w", err)
	}
	
	// Create component-specific policies
	if err := m.CreateControllerNetworkPolicy(ctx); err != nil {
		return fmt.Errorf("failed to create controller policy: %w", err)
	}
	
	if err := m.CreateLLMServiceNetworkPolicy(ctx); err != nil {
		return fmt.Errorf("failed to create LLM service policy: %w", err)
	}
	
	// Create O-RAN interface policies
	interfaces := []string{"A1", "O1", "O2", "E2"}
	for _, iface := range interfaces {
		if err := m.CreateORANInterfacePolicy(ctx, iface); err != nil {
			logger.Error(err, "Failed to create O-RAN interface policy", "interface", iface)
			// Continue with other interfaces
		}
	}
	
	logger.Info("Zero-trust networking enforced")
	return nil
}