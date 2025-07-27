package llm

import (
	"fmt"
	"regexp"
	"strings"
)

// TelecomPromptEngine handles telecom-specific prompt engineering
type TelecomPromptEngine struct {
	systemPrompts map[string]string
	examples      map[string][]PromptExample
}

type PromptExample struct {
	Intent   string `json:"intent"`
	Response string `json:"response"`
}

// NewTelecomPromptEngine creates a new telecom prompt engineering system
func NewTelecomPromptEngine() *TelecomPromptEngine {
	engine := &TelecomPromptEngine{
		systemPrompts: make(map[string]string),
		examples:      make(map[string][]PromptExample),
	}

	engine.initializeSystemPrompts()
	engine.initializeExamples()

	return engine
}

func (t *TelecomPromptEngine) initializeSystemPrompts() {
	t.systemPrompts["NetworkFunctionDeployment"] = `You are an expert telecommunications network engineer with deep knowledge of:

**5G Core Network Functions:**
- AMF (Access and Mobility Management Function): Handles registration, authentication, mobility management
- SMF (Session Management Function): Manages PDU sessions, IP address allocation
- UPF (User Plane Function): Packet routing and forwarding, QoS enforcement
- NSSF (Network Slice Selection Function): Selects network slice instances
- PCF (Policy Control Function): Unified policy framework, QoS control
- UDM (Unified Data Management): Subscription management, authentication credentials
- UDR (Unified Data Repository): Stores subscription data, policy data
- AUSF (Authentication Server Function): Authentication services
- NRF (Network Repository Function): Service discovery and registration

**O-RAN Architecture:**
- O-DU (O-RAN Distributed Unit): Lower-layer radio processing, real-time functions
- O-CU (O-RAN Central Unit): Higher-layer radio processing, non-real-time functions
- Near-RT RIC (Near Real-Time RAN Intelligent Controller): Sub-second control loops, xApps
- Non-RT RIC (Non Real-Time RIC): Longer-term optimization, rApps
- O1 Interface: Management plane for FCAPS (Fault, Configuration, Accounting, Performance, Security)
- A1 Interface: Policy and enrichment information exchange
- E2 Interface: Near real-time control and monitoring

**Network Slicing and QoS:**
- eMBB (Enhanced Mobile Broadband): High data rates, dense urban areas
- URLLC (Ultra-Reliable Low-Latency Communication): Critical applications, industrial IoT
- mMTC (Massive Machine-Type Communication): IoT devices, sensors
- QoS Flow Identifier (QFI): Service data flow identification
- Service and Session Continuity (SSC): Modes 1, 2, 3 for different mobility patterns

**Edge Computing and MEC:**
- Multi-access Edge Computing deployment patterns
- Edge application lifecycle management
- Ultra-low latency service placement
- Local data processing and caching

**Kubernetes and Cloud-Native:**
- CNF (Cloud-Native Network Functions) deployment patterns
- Helm charts for telecom workloads
- Network policies and service mesh integration
- Persistent storage for stateful network functions

Your task is to translate natural language network operations requests into structured JSON objects for Kubernetes deployment.

**Output Requirements:**
1. MUST return valid JSON matching the specified schema
2. Use realistic telecom-specific naming conventions
3. Include appropriate resource allocations based on network function type
4. Generate meaningful O1 configuration and A1 policies when applicable
5. Consider high availability and scaling requirements

**Schema for NetworkFunctionDeployment:**
{
  "type": "NetworkFunctionDeployment",
  "name": "descriptive-nf-name-following-3gpp-naming",
  "namespace": "target-namespace",
  "spec": {
    "replicas": integer,
    "image": "container-registry/nf-image:version",
    "resources": {
      "requests": {"cpu": "cpu-in-millicores", "memory": "memory-in-Mi-or-Gi"},
      "limits": {"cpu": "cpu-in-millicores", "memory": "memory-in-Mi-or-Gi"}
    },
    "ports": [{"containerPort": port_number, "protocol": "TCP|UDP|SCTP"}],
    "env": [{"name": "ENV_VAR_NAME", "value": "value"}],
    "serviceType": "ClusterIP|NodePort|LoadBalancer",
    "annotations": {"key": "value"}
  },
  "o1_config": "XML configuration for FCAPS operations following O1 interface standards",
  "a1_policy": {
    "policy_type_id": "policy-identifier",
    "policy_data": {
      "scope": "network-slice-id or cell-id",
      "qos_parameters": {},
      "resource_allocation": {}
    }
  },
  "network_slice": {
    "sst": "slice-service-type",
    "sd": "slice-differentiator",
    "plmn_id": "public-land-mobile-network-id"
  }
}`

	t.systemPrompts["NetworkFunctionScale"] = `You are an expert telecommunications network engineer with deep knowledge of 5G Core, O-RAN, and cloud-native network functions.

Your task is to translate natural language scaling requests into structured JSON objects for Kubernetes horizontal/vertical scaling.

**Understanding Network Function Scaling:**
- Horizontal Scaling: Increase/decrease replica count for stateless NFs (AMF, SMF, PCF)
- Vertical Scaling: Increase/decrease CPU/memory for stateful NFs (UPF, UDR)
- Auto-scaling: Configure HPA based on metrics (CPU, memory, custom metrics)

**Scaling Considerations:**
- UPF: Typically requires vertical scaling due to packet processing requirements
- AMF/SMF: Can benefit from horizontal scaling for high registration/session load
- Near-RT RIC: May need both horizontal and vertical scaling based on xApp load

**Schema for NetworkFunctionScale:**
{
  "type": "NetworkFunctionScale",
  "name": "existing-nf-name",
  "namespace": "target-namespace",
  "scaling": {
    "horizontal": {
      "replicas": integer,
      "min_replicas": integer,
      "max_replicas": integer,
      "target_cpu_utilization": percentage
    },
    "vertical": {
      "cpu": "cpu-in-millicores",
      "memory": "memory-in-Mi-or-Gi"
    }
  }
}`
}

func (t *TelecomPromptEngine) initializeExamples() {
	// NetworkFunctionDeployment examples
	t.examples["NetworkFunctionDeployment"] = []PromptExample{
		{
			Intent: "Deploy a UPF network function with 3 replicas for high availability in the core network",
			Response: `{
  "type": "NetworkFunctionDeployment",
  "name": "upf-core-network",
  "namespace": "5g-core",
  "spec": {
    "replicas": 3,
    "image": "registry.5g.local/upf:v2.1.0",
    "resources": {
      "requests": {"cpu": "2000m", "memory": "4Gi"},
      "limits": {"cpu": "4000m", "memory": "8Gi"}
    },
    "ports": [
      {"containerPort": 8805, "protocol": "UDP"},
      {"containerPort": 2152, "protocol": "UDP"}
    ],
    "env": [
      {"name": "UPF_MODE", "value": "core"},
      {"name": "DNN_LIST", "value": "internet,ims"},
      {"name": "METRICS_ENABLED", "value": "true"}
    ],
    "serviceType": "LoadBalancer",
    "annotations": {
      "5g.nephoran.com/network-function": "upf",
      "5g.nephoran.com/slice-support": "embb,urllc"
    }
  },
  "o1_config": "<?xml version=\"1.0\"?><config xmlns=\"urn:o-ran:nf:1.0\"><upf><interfaces><n3>192.168.1.0/24</n3><n4>192.168.2.0/24</n4><n6>192.168.3.0/24</n6></interfaces><qos><profiles><profile id=\"1\" name=\"embb\"><max_bitrate>1000000000</max_bitrate></profile></profiles></qos></upf></config>",
  "a1_policy": {
    "policy_type_id": "upf-qos-policy-v1",
    "policy_data": {
      "scope": "nsi-core-001",
      "qos_parameters": {
        "guaranteed_bitrate": "100Mbps",
        "maximum_bitrate": "1Gbps",
        "packet_delay_budget": "10ms"
      },
      "resource_allocation": {
        "cpu_allocation": "70%",
        "memory_allocation": "6Gi"
      }
    }
  },
  "network_slice": {
    "sst": "1",
    "sd": "000001",
    "plmn_id": "00101"
  }
}`,
		},
		{
			Intent: "Set up Near-RT RIC with xApp support for intelligent traffic management",
			Response: `{
  "type": "NetworkFunctionDeployment",
  "name": "near-rt-ric-xapp-platform",
  "namespace": "o-ran",
  "spec": {
    "replicas": 2,
    "image": "registry.oran.local/near-rt-ric:v3.0.0",
    "resources": {
      "requests": {"cpu": "1000m", "memory": "2Gi"},
      "limits": {"cpu": "2000m", "memory": "4Gi"}
    },
    "ports": [
      {"containerPort": 8080, "protocol": "TCP"},
      {"containerPort": 36421, "protocol": "SCTP"},
      {"containerPort": 36422, "protocol": "SCTP"}
    ],
    "env": [
      {"name": "RIC_MODE", "value": "near-rt"},
      {"name": "XAPP_REGISTRY", "value": "http://xapp-registry:8080"},
      {"name": "E2_INTERFACE_ENABLED", "value": "true"},
      {"name": "A1_INTERFACE_ENABLED", "value": "true"}
    ],
    "serviceType": "ClusterIP",
    "annotations": {
      "o-ran.nephoran.com/component": "near-rt-ric",
      "o-ran.nephoran.com/interfaces": "e2,a1"
    }
  },
  "o1_config": "<?xml version=\"1.0\"?><config xmlns=\"urn:o-ran:ric:1.0\"><near-rt-ric><e2-interface><sctp-port>36421</sctp-port><supported-functions>RC,KPM,E2SM</supported-functions></e2-interface><a1-interface><policy-types>QoS,Mobility,Interference</policy-types></a1-interface><xapp-manager><deployment-timeout>300</deployment-timeout><health-check-interval>30</health-check-interval></xapp-manager></near-rt-ric></config>",
  "a1_policy": {
    "policy_type_id": "ric-traffic-steering-v1",
    "policy_data": {
      "scope": "cell-cluster-001",
      "qos_parameters": {
        "optimization_target": "throughput",
        "interference_threshold": "-90dBm"
      },
      "resource_allocation": {
        "xapp_cpu_quota": "50%",
        "xapp_memory_quota": "1Gi"
      }
    }
  }
}`,
		},
		{
			Intent: "Configure edge computing node with GPU acceleration for video processing",
			Response: `{
  "type": "NetworkFunctionDeployment",
  "name": "mec-video-processing-gpu",
  "namespace": "edge-apps",
  "spec": {
    "replicas": 1,
    "image": "registry.edge.local/video-processor:v1.5.0",
    "resources": {
      "requests": {"cpu": "500m", "memory": "1Gi", "nvidia.com/gpu": "1"},
      "limits": {"cpu": "2000m", "memory": "4Gi", "nvidia.com/gpu": "1"}
    },
    "ports": [
      {"containerPort": 8888, "protocol": "TCP"},
      {"containerPort": 1935, "protocol": "TCP"}
    ],
    "env": [
      {"name": "GPU_ENABLED", "value": "true"},
      {"name": "CUDA_VERSION", "value": "12.0"},
      {"name": "VIDEO_CODEC", "value": "h264,h265"},
      {"name": "MAX_CONCURRENT_STREAMS", "value": "16"}
    ],
    "serviceType": "NodePort",
    "annotations": {
      "mec.nephoran.com/service-type": "video-processing",
      "mec.nephoran.com/gpu-required": "true",
      "mec.nephoran.com/latency-requirement": "ultra-low"
    }
  },
  "o1_config": "<?xml version=\"1.0\"?><config xmlns=\"urn:mec:app:1.0\"><video-processor><gpu-acceleration enabled=\"true\"><memory-allocation>2Gi</memory-allocation><compute-capability>7.5</compute-capability></gpu-acceleration><streaming><max-resolution>4K</max-resolution><target-latency>5ms</target-latency></streaming></video-processor></config>",
  "a1_policy": {
    "policy_type_id": "mec-resource-allocation-v1",
    "policy_data": {
      "scope": "edge-zone-001",
      "qos_parameters": {
        "max_processing_latency": "5ms",
        "guaranteed_throughput": "500Mbps"
      },
      "resource_allocation": {
        "gpu_memory": "8Gi",
        "bandwidth_allocation": "1Gbps"
      }
    }
  }
}`,
		},
	}

	// NetworkFunctionScale examples
	t.examples["NetworkFunctionScale"] = []PromptExample{
		{
			Intent: "Scale AMF instances to 5 replicas to handle increased signaling load",
			Response: `{
  "type": "NetworkFunctionScale",
  "name": "amf-core",
  "namespace": "5g-core",
  "scaling": {
    "horizontal": {
      "replicas": 5,
      "min_replicas": 3,
      "max_replicas": 10,
      "target_cpu_utilization": 70
    }
  }
}`,
		},
		{
			Intent: "Increase UPF resources to 4 CPU cores and 8GB memory for high throughput",
			Response: `{
  "type": "NetworkFunctionScale",
  "name": "upf-edge",
  "namespace": "5g-core",
  "scaling": {
    "vertical": {
      "cpu": "4000m",
      "memory": "8Gi"
    }
  }
}`,
		},
	}
}

// GeneratePrompt creates a context-aware prompt for the given intent type
func (t *TelecomPromptEngine) GeneratePrompt(intentType, userIntent string) string {
	systemPrompt, exists := t.systemPrompts[intentType]
	if !exists {
		systemPrompt = t.systemPrompts["NetworkFunctionDeployment"] // Default
	}

	// Add relevant examples based on intent content
	examples := t.selectRelevantExamples(intentType, userIntent)
	if len(examples) > 0 {
		systemPrompt += "\n\n**Examples:**\n"
		for i, example := range examples {
			if i >= 2 { // Limit to 2 examples to avoid token limits
				break
			}
			systemPrompt += fmt.Sprintf("\nExample %d:\nIntent: %s\nResponse: %s\n", i+1, example.Intent, example.Response)
		}
	}

	// Add intent-specific guidelines
	systemPrompt += t.generateIntentSpecificGuidelines(intentType, userIntent)

	return systemPrompt
}

// selectRelevantExamples picks the most relevant examples based on intent content
func (t *TelecomPromptEngine) selectRelevantExamples(intentType, userIntent string) []PromptExample {
	examples, exists := t.examples[intentType]
	if !exists {
		return nil
	}

	lowerIntent := strings.ToLower(userIntent)
	var relevantExamples []PromptExample

	// Score examples based on keyword matching
	type scoredExample struct {
		example PromptExample
		score   int
	}

	var scoredExamples []scoredExample

	for _, example := range examples {
		score := 0
		lowerExampleIntent := strings.ToLower(example.Intent)

		// Define scoring keywords
		keywords := map[string]int{
			"upf":          10,
			"amf":          10,
			"smf":          10,
			"ric":          10,
			"near-rt":      8,
			"edge":         8,
			"mec":          8,
			"gpu":          6,
			"video":        6,
			"scale":        8,
			"replicas":     8,
			"high":         4,
			"availability": 4,
			"processing":   4,
			"core":         5,
			"network":      3,
		}

		for keyword, weight := range keywords {
			if strings.Contains(lowerIntent, keyword) && strings.Contains(lowerExampleIntent, keyword) {
				score += weight
			}
		}

		if score > 0 {
			scoredExamples = append(scoredExamples, scoredExample{example, score})
		}
	}

	// Sort by score and return top examples
	for i := 0; i < len(scoredExamples)-1; i++ {
		for j := i + 1; j < len(scoredExamples); j++ {
			if scoredExamples[j].score > scoredExamples[i].score {
				scoredExamples[i], scoredExamples[j] = scoredExamples[j], scoredExamples[i]
			}
		}
	}

	for _, scored := range scoredExamples {
		relevantExamples = append(relevantExamples, scored.example)
	}

	return relevantExamples
}

// generateIntentSpecificGuidelines adds specific guidelines based on the intent content
func (t *TelecomPromptEngine) generateIntentSpecificGuidelines(intentType, userIntent string) string {
	guidelines := "\n\n**Specific Guidelines for this Intent:**\n"
	lowerIntent := strings.ToLower(userIntent)

	// Network function specific guidelines
	if strings.Contains(lowerIntent, "upf") {
		guidelines += "- UPF requires higher CPU/memory for packet processing\n"
		guidelines += "- Consider DPDK or SR-IOV for high-performance networking\n"
		guidelines += "- Include N3, N4, N6, N9 interface configurations\n"
	}

	if strings.Contains(lowerIntent, "amf") {
		guidelines += "- AMF is typically stateless and can scale horizontally\n"
		guidelines += "- Include N1, N2, N8, N11, N12, N14, N15 interface configurations\n"
		guidelines += "- Consider session continuity for mobile users\n"
	}

	if strings.Contains(lowerIntent, "ric") || strings.Contains(lowerIntent, "near-rt") {
		guidelines += "- Near-RT RIC requires E2 and A1 interface support\n"
		guidelines += "- Include xApp deployment capabilities\n"
		guidelines += "- Consider real-time processing requirements (<10ms)\n"
	}

	if strings.Contains(lowerIntent, "edge") || strings.Contains(lowerIntent, "mec") {
		guidelines += "- Edge applications require low-latency networking\n"
		guidelines += "- Consider local data processing and caching\n"
		guidelines += "- Include edge-specific resource constraints\n"
	}

	// Scaling specific guidelines
	if strings.Contains(lowerIntent, "scale") {
		guidelines += "- Consider both horizontal and vertical scaling options\n"
		guidelines += "- Include auto-scaling parameters when appropriate\n"
		guidelines += "- Account for stateful vs stateless scaling patterns\n"
	}

	// High availability guidelines
	if strings.Contains(lowerIntent, "high availability") || strings.Contains(lowerIntent, "ha") {
		guidelines += "- Use anti-affinity rules for pod distribution\n"
		guidelines += "- Set appropriate replica counts (minimum 3 for HA)\n"
		guidelines += "- Include readiness and liveness probes\n"
	}

	// Resource allocation guidelines
	if regexp.MustCompile(`\d+\s*(cpu|core|memory|ram|gb|gi)`).MatchString(lowerIntent) {
		guidelines += "- Extract specific resource requirements from the intent\n"
		guidelines += "- Set appropriate requests and limits\n"
		guidelines += "- Consider resource over-subscription ratios\n"
	}

	return guidelines
}

// ExtractParameters attempts to extract structured parameters from natural language
func (t *TelecomPromptEngine) ExtractParameters(intent string) map[string]interface{} {
	params := make(map[string]interface{})
	lowerIntent := strings.ToLower(intent)

	// Extract replica count
	replicaRegex := regexp.MustCompile(`(\d+)\s+replicas?`)
	if matches := replicaRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["replicas"] = matches[1]
	}

	// Extract CPU resources
	cpuRegex := regexp.MustCompile(`(\d+)\s*(cpu\s+cores?|cores?|cpus?)`)
	if matches := cpuRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["cpu"] = matches[1] + "000m" // Convert cores to millicores
	}

	// Extract memory resources
	memRegex := regexp.MustCompile(`(\d+)\s*([gm])i?b?\s*(memory|ram)`)
	if matches := memRegex.FindStringSubmatch(lowerIntent); len(matches) > 2 {
		unit := "Mi"
		if matches[2] == "g" {
			unit = "Gi"
		}
		params["memory"] = matches[1] + unit
	}

	// Also try alternative memory patterns
	memRegex2 := regexp.MustCompile(`(\d+)\s*gb\s+memory`)
	if matches := memRegex2.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["memory"] = matches[1] + "Gi"
	}

	// Handle "set memory to XMB" pattern
	memRegex3 := regexp.MustCompile(`memory\s+to\s+(\d+)\s*([gm])i?b?`)
	if matches := memRegex3.FindStringSubmatch(lowerIntent); len(matches) > 2 {
		unit := "Mi"
		if matches[2] == "g" {
			unit = "Gi"
		}
		params["memory"] = matches[1] + unit
	}

	// Handle "increase memory to XGB" pattern
	memRegex4 := regexp.MustCompile(`increase\s+memory\s+to\s+(\d+)\s*([gm])i?b?`)
	if matches := memRegex4.FindStringSubmatch(lowerIntent); len(matches) > 2 {
		unit := "Mi"
		if matches[2] == "g" {
			unit = "Gi"
		}
		params["memory"] = matches[1] + unit
	}

	// Handle "allocate XGB memory" pattern
	memRegex5 := regexp.MustCompile(`allocate\s+(\d+)\s*([gm])i?b?\s+memory`)
	if matches := memRegex5.FindStringSubmatch(lowerIntent); len(matches) > 2 {
		unit := "Mi"
		if matches[2] == "g" {
			unit = "Gi"
		}
		params["memory"] = matches[1] + unit
	}

	// Extract network function names
	nfRegex := regexp.MustCompile(`(upf|amf|smf|pcf|nssf|udm|udr|ausf|nrf|near-rt\s+ric|o-du|o-cu)`)
	if matches := nfRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["network_function"] = strings.ReplaceAll(matches[1], " ", "-")
	}

	// Extract namespace hints
	nsRegex := regexp.MustCompile(`(5g-core|o-ran|edge|edge-apps|mec|core|radio)`)
	if matches := nsRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		switch matches[1] {
		case "5g-core", "core":
			params["namespace"] = "5g-core"
		case "o-ran", "radio":
			params["namespace"] = "o-ran"
		case "edge", "edge-apps", "mec":
			params["namespace"] = "edge-apps"
		}
	}

	return params
}
