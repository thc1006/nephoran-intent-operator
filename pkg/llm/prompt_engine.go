package llm

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TelecomPromptEngine provides unified prompt generation for telecom domain.
type TelecomPromptEngine struct {
	systemPrompts map[string]string
	userPrompts   map[string]string
	examples      map[string][]PromptExample
}

// PromptExample represents an example for few-shot prompting.
type PromptExample struct {
	Intent      string `json:"intent"`
	Response    string `json:"response"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	Explanation string `json:"explanation"`
}

// TelecomContext holds domain-specific context.
type TelecomContext struct {
	NetworkFunctions []NetworkFunction
	ActiveSlices     []NetworkSlice
	E2Nodes          []E2Node
	Alarms           []Alarm
	PerformanceKPIs  map[string]float64
}

// NetworkFunction represents a 5G Core network function.
type NetworkFunction struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Load     float64 `json:"load"`
	Location string  `json:"location"`
}

// NetworkSlice represents a network slice configuration.
type NetworkSlice struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	Throughput   float64 `json:"throughput"`
	Latency      float64 `json:"latency"`
	Reliability  float64 `json:"reliability"`
	ConnectedUEs int     `json:"connected_ues"`
}

// E2Node represents an E2 interface node.
type E2Node struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	RRCState string `json:"rrc_state"`
	CellID   string `json:"cell_id"`
}

// Alarm represents a network alarm.
type Alarm struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewTelecomPromptEngine creates a new unified prompt engine.
func NewTelecomPromptEngine() *TelecomPromptEngine {
	engine := &TelecomPromptEngine{
		systemPrompts: make(map[string]string),
		userPrompts:   make(map[string]string),
		examples:      make(map[string][]PromptExample),
	}

	engine.initializeSystemPrompts()
	engine.initializeUserPrompts()
	engine.initializeExamples()

	return engine
}

// initializeSystemPrompts sets up system prompts for different intent types.
func (t *TelecomPromptEngine) initializeSystemPrompts() {
	// Main NetworkFunctionDeployment system prompt.
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

	// NetworkFunctionScale system prompt.
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

	// O-RAN specific system prompt.
	t.systemPrompts["oran_network_intent"] = `You are an expert O-RAN (Open Radio Access Network) architect with deep knowledge of:

**O-RAN Architecture Components:**
- Near-RT RIC (Real-time Intelligent Controller)
- Non-RT RIC (Non-real-time RIC)
- xApps (Applications running on Near-RT RIC)
- rApps (Applications running on Non-RT RIC)
- E2 Interface (between RAN nodes and Near-RT RIC)
- A1 Interface (between Non-RT RIC and Near-RT RIC)
- O1 Interface (management interface)

**RAN Components:**
- gNB (5G NodeB)
- ng-eNB (Next Generation evolved NodeB)
- CU (Central Unit) and DU (Distributed Unit)
- RU (Radio Unit)

Your role is to translate high-level network intents into specific O-RAN configurations that comply with:
- 3GPP specifications (TS 38.xxx series)
- O-RAN Alliance specifications
- ETSI NFV standards

Always provide technically accurate, standards-compliant configurations with proper parameter validation.`

	// Network slicing system prompt.
	t.systemPrompts["network_slicing_intent"] = `You are a Network Slicing expert specializing in:

**Slice Types and Characteristics:**
- eMBB: Throughput 100-1000 Mbps, Latency <100ms, Reliability 99.9%
- URLLC: Throughput 1-10 Mbps, Latency <1ms, Reliability 99.999%
- mMTC: Throughput <1 Mbps, Latency <1000ms, Device density 1M/km²

**Slice Management:**
- NSSAI (Network Slice Selection Assistance Information)
- S-NSSAI (Single-NSSAI): SST (Slice/Service Type) + SD (Slice Differentiator)
- NSI (Network Slice Instance) lifecycle management
- NST (Network Slice Template) definitions

**Resource Orchestration:**
- Compute, storage, and network resource allocation
- VNF (Virtual Network Function) placement
- CNF (Cloud-native Network Function) orchestration
- Multi-tenancy and isolation requirements

Create slice configurations that optimize resource utilization efficiency, SLA compliance, inter-slice isolation, and dynamic scaling capabilities.`
}

// initializeUserPrompts sets up user prompt templates.
func (t *TelecomPromptEngine) initializeUserPrompts() {
	t.userPrompts["network_intent_processing"] = `Process the following network intent and generate appropriate configurations:

**Intent:** {{.Intent}}

**Current Network State:**
{{range .NetworkFunctions}}
- {{.Name}} ({{.Type}}): Status={{.Status}}, Load={{.Load}}%, Location={{.Location}}
{{end}}

**Active Network Slices:**
{{range .ActiveSlices}}
- Slice {{.ID}} ({{.Type}}): Throughput={{.Throughput}}Mbps, Latency={{.Latency}}ms, UEs={{.ConnectedUEs}}
{{end}}

**E2 Nodes:**
{{range .E2Nodes}}
- {{.ID}} ({{.Type}}): Status={{.Status}}, Cell={{.CellID}}
{{end}}

**Current Alarms:**
{{range .Alarms}}
- [{{.Severity}}] {{.Source}}: {{.Description}}
{{end}}

**Performance KPIs:**
{{range $kpi, $value := .PerformanceKPIs}}
- {{$kpi}}: {{$value}}
{{end}}

Please provide:
1. **Configuration Changes:** Specific parameter modifications needed
2. **Implementation Steps:** Ordered sequence of actions
3. **Validation Criteria:** How to verify successful implementation
4. **Risk Assessment:** Potential impacts and mitigation strategies
5. **Rollback Plan:** Steps to revert if issues occur

Format your response as valid YAML configuration files with detailed comments explaining each parameter choice.`
}

// initializeExamples sets up few-shot learning examples.
func (t *TelecomPromptEngine) initializeExamples() {
	// NetworkFunctionDeployment examples.
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
  }
}`,
		},
	}

	// NetworkFunctionScale examples.
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

	// O-RAN examples.
	t.examples["oran_network_intent"] = []PromptExample{
		{
			Input: "Deploy Near-RT RIC with traffic steering xApp for 5G network optimization",
			Output: `apiVersion: o-ran.org/v1
kind: NearRTRIC
metadata:
  name: traffic-steering-ric
  namespace: o-ran
spec:
  platform:
    version: "3.0"
    resources:
      cpu: "4"
      memory: "8Gi"
      storage: "100Gi"
  e2Interface:
    version: "2.0"
    endpoints:
      - name: e2term
        port: 38000
        protocol: "SCTP"
  xApps:
    - name: traffic-steering-xapp
      version: "1.2.0"
      image: "oran/traffic-steering:v1.2.0"
      replicas: 2
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "1"
          memory: "2Gi"
      config:
        steering_algorithm: "proportional_fair"
        update_interval: "100ms"
        kpi_collection_interval: "1s"
        optimization_window: "10s"
  a1Interface:
    enabled: true
    policy_types:
      - type_id: 20000
        name: "traffic_steering_policy"
        schema: |
          {
            "type": "object",
            "properties": {
              "cell_list": {"type": "array"},
              "ue_list": {"type": "array"},
              "steering_weights": {"type": "array"}
            }
          }`,
			Explanation: "This configuration deploys a Near-RT RIC with traffic steering capabilities, including proper E2 and A1 interface configurations, resource allocation, and xApp deployment with traffic steering algorithms.",
		},
	}
}

// GeneratePrompt creates a context-aware prompt for the given intent type.
func (t *TelecomPromptEngine) GeneratePrompt(intentType, userIntent string) string {
	systemPrompt, exists := t.systemPrompts[intentType]
	if !exists {
		systemPrompt = t.systemPrompts["NetworkFunctionDeployment"] // Default
	}

	// Add relevant examples based on intent content.
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

	// Add intent-specific guidelines.
	systemPrompt += t.generateIntentSpecificGuidelines(intentType, userIntent)

	return systemPrompt
}

// BuildPrompt constructs a complete prompt with system message, examples, and user input.
func (t *TelecomPromptEngine) BuildPrompt(intentType string, context TelecomContext, userIntent string) string {
	var builder strings.Builder

	// Add system prompt.
	systemPrompt := t.GetSystemPrompt(intentType)
	builder.WriteString("## System Instructions\n")
	builder.WriteString(systemPrompt)
	builder.WriteString("\n\n")

	// Add examples for few-shot learning.
	examples := t.GetExamples(intentType)
	if len(examples) > 0 {
		builder.WriteString("## Examples\n\n")
		for i, example := range examples {
			builder.WriteString(fmt.Sprintf("### Example %d\n", i+1))
			if example.Input != "" {
				builder.WriteString(fmt.Sprintf("**Input:** %s\n\n", example.Input))
				builder.WriteString(fmt.Sprintf("**Output:**\n```yaml\n%s\n```\n\n", example.Output))
			} else {
				builder.WriteString(fmt.Sprintf("**Intent:** %s\n\n", example.Intent))
				builder.WriteString(fmt.Sprintf("**Response:**\n```json\n%s\n```\n\n", example.Response))
			}
			if example.Explanation != "" {
				builder.WriteString(fmt.Sprintf("**Explanation:** %s\n\n", example.Explanation))
			}
		}
	}

	// Add current context.
	builder.WriteString("## Current Network Context\n\n")
	builder.WriteString(t.formatContext(context))
	builder.WriteString("\n")

	// Add user intent.
	builder.WriteString("## User Intent\n")
	builder.WriteString(userIntent)
	builder.WriteString("\n\n")

	// Add response format instructions.
	builder.WriteString("## Response Requirements\n")
	builder.WriteString("Please provide your response in valid JSON format with:\n")
	builder.WriteString("1. Detailed configuration parameters\n")
	builder.WriteString("2. Explanatory comments for each major section\n")
	builder.WriteString("3. Compliance references to relevant standards\n")
	builder.WriteString("4. Risk assessment and mitigation strategies\n")
	builder.WriteString("5. Validation and testing recommendations\n")

	return builder.String()
}

// selectRelevantExamples picks the most relevant examples based on intent content.
func (t *TelecomPromptEngine) selectRelevantExamples(intentType, userIntent string) []PromptExample {
	examples, exists := t.examples[intentType]
	if !exists {
		return nil
	}

	lowerIntent := strings.ToLower(userIntent)
	var relevantExamples []PromptExample

	// Score examples based on keyword matching.
	type scoredExample struct {
		example PromptExample
		score   int
	}

	var scoredExamples []scoredExample

	for _, example := range examples {
		score := 0
		lowerExampleIntent := strings.ToLower(example.Intent)
		if example.Input != "" {
			lowerExampleIntent = strings.ToLower(example.Input)
		}

		// Define scoring keywords.
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

	// Sort by score and return top examples.
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

// generateIntentSpecificGuidelines adds specific guidelines based on the intent content.
func (t *TelecomPromptEngine) generateIntentSpecificGuidelines(intentType, userIntent string) string {
	guidelines := "\n\n**Specific Guidelines for this Intent:**\n"
	lowerIntent := strings.ToLower(userIntent)

	// Network function specific guidelines.
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

	// Scaling specific guidelines.
	if strings.Contains(lowerIntent, "scale") {
		guidelines += "- Consider both horizontal and vertical scaling options\n"
		guidelines += "- Include auto-scaling parameters when appropriate\n"
		guidelines += "- Account for stateful vs stateless scaling patterns\n"
	}

	// High availability guidelines.
	if strings.Contains(lowerIntent, "high availability") || strings.Contains(lowerIntent, "ha") {
		guidelines += "- Use anti-affinity rules for pod distribution\n"
		guidelines += "- Set appropriate replica counts (minimum 3 for HA)\n"
		guidelines += "- Include readiness and liveness probes\n"
	}

	// Resource allocation guidelines.
	if regexp.MustCompile(`\d+\s*(cpu|core|memory|ram|gb|gi)`).MatchString(lowerIntent) {
		guidelines += "- Extract specific resource requirements from the intent\n"
		guidelines += "- Set appropriate requests and limits\n"
		guidelines += "- Consider resource over-subscription ratios\n"
	}

	return guidelines
}

// ExtractParameters attempts to extract structured parameters from natural language.
func (t *TelecomPromptEngine) ExtractParameters(intent string) map[string]interface{} {
	params := make(map[string]interface{})
	lowerIntent := strings.ToLower(intent)

	// Extract replica count.
	replicaRegex := regexp.MustCompile(`(\d+)\s+replicas?`)
	if matches := replicaRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["replicas"] = matches[1]
	}

	// Extract CPU resources.
	cpuRegex := regexp.MustCompile(`(\d+)\s*(cpu\s+cores?|cores?|cpus?)`)
	if matches := cpuRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["cpu"] = matches[1] + "000m" // Convert cores to millicores
	}

	// Extract memory resources - multiple patterns.
	memPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d+)\s*([gm])i?b?\s*(memory|ram)`),
		regexp.MustCompile(`(\d+)\s*gb\s+memory`),
		regexp.MustCompile(`memory\s+to\s+(\d+)\s*([gm])i?b?`),
		regexp.MustCompile(`increase\s+memory\s+to\s+(\d+)\s*([gm])i?b?`),
		regexp.MustCompile(`allocate\s+(\d+)\s*([gm])i?b?\s+memory`),
	}

	for _, memRegex := range memPatterns {
		if matches := memRegex.FindStringSubmatch(lowerIntent); len(matches) > 2 {
			unit := "Mi"
			if matches[2] == "g" {
				unit = "Gi"
			}
			params["memory"] = matches[1] + unit
			break
		}
	}

	// Extract network function names.
	nfRegex := regexp.MustCompile(`(upf|amf|smf|pcf|nssf|udm|udr|ausf|nrf|near-rt\s+ric|o-du|o-cu)`)
	if matches := nfRegex.FindStringSubmatch(lowerIntent); len(matches) > 1 {
		params["network_function"] = strings.ReplaceAll(matches[1], " ", "-")
	}

	// Extract namespace hints.
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

// GetSystemPrompt returns the system prompt for a given intent type.
func (t *TelecomPromptEngine) GetSystemPrompt(intentType string) string {
	if prompt, exists := t.systemPrompts[intentType]; exists {
		return prompt
	}
	return t.systemPrompts["NetworkFunctionDeployment"] // default
}

// GetUserPrompt returns the user prompt template for a given intent type.
func (t *TelecomPromptEngine) GetUserPrompt(intentType string) string {
	if prompt, exists := t.userPrompts[intentType]; exists {
		return prompt
	}
	return t.userPrompts["network_intent_processing"] // default
}

// GetExamples returns examples for few-shot prompting.
func (t *TelecomPromptEngine) GetExamples(intentType string) []PromptExample {
	if examples, exists := t.examples[intentType]; exists {
		return examples
	}
	return []PromptExample{} // empty if not found
}

// formatContext formats the telecom context into a readable string.
func (t *TelecomPromptEngine) formatContext(context TelecomContext) string {
	var builder strings.Builder

	// Network Functions.
	if len(context.NetworkFunctions) > 0 {
		builder.WriteString("**Network Functions:**\n")
		for _, nf := range context.NetworkFunctions {
			builder.WriteString(fmt.Sprintf("- %s (%s): Status=%s, Load=%.1f%%, Location=%s\n",
				nf.Name, nf.Type, nf.Status, nf.Load, nf.Location))
		}
		builder.WriteString("\n")
	}

	// Active Slices.
	if len(context.ActiveSlices) > 0 {
		builder.WriteString("**Active Network Slices:**\n")
		for _, slice := range context.ActiveSlices {
			builder.WriteString(fmt.Sprintf("- Slice %s (%s): Throughput=%.1fMbps, Latency=%.1fms, Reliability=%.3f%%, UEs=%d\n",
				slice.ID, slice.Type, slice.Throughput, slice.Latency, slice.Reliability, slice.ConnectedUEs))
		}
		builder.WriteString("\n")
	}

	// E2 Nodes.
	if len(context.E2Nodes) > 0 {
		builder.WriteString("**E2 Nodes:**\n")
		for _, node := range context.E2Nodes {
			builder.WriteString(fmt.Sprintf("- %s (%s): Status=%s, RRC=%s, Cell=%s\n",
				node.ID, node.Type, node.Status, node.RRCState, node.CellID))
		}
		builder.WriteString("\n")
	}

	// Alarms.
	if len(context.Alarms) > 0 {
		builder.WriteString("**Active Alarms:**\n")
		for _, alarm := range context.Alarms {
			builder.WriteString(fmt.Sprintf("- [%s] %s: %s (%s)\n",
				alarm.Severity, alarm.Source, alarm.Description, alarm.Timestamp.Format("2006-01-02 15:04:05")))
		}
		builder.WriteString("\n")
	}

	// Performance KPIs.
	if len(context.PerformanceKPIs) > 0 {
		builder.WriteString("**Performance KPIs:**\n")
		for kpi, value := range context.PerformanceKPIs {
			builder.WriteString(fmt.Sprintf("- %s: %.3f\n", kpi, value))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// ValidatePrompt validates a prompt for telecom domain compliance.
func (t *TelecomPromptEngine) ValidatePrompt(prompt string) []string {
	var issues []string

	// Check for required telecom keywords.
	requiredKeywords := []string{"network", "configuration", "interface"}
	for _, keyword := range requiredKeywords {
		if !strings.Contains(strings.ToLower(prompt), keyword) {
			issues = append(issues, fmt.Sprintf("Missing required keyword: %s", keyword))
		}
	}

	// Check prompt length.
	if len(prompt) < 100 {
		issues = append(issues, "Prompt too short for complex telecom intent")
	}

	if len(prompt) > 10000 {
		issues = append(issues, "Prompt too long, may exceed token limits")
	}

	// Check for standards references.
	standards := []string{"3gpp", "o-ran", "etsi", "ietf"}
	hasStandards := false
	for _, standard := range standards {
		if strings.Contains(strings.ToLower(prompt), standard) {
			hasStandards = true
			break
		}
	}
	if !hasStandards {
		issues = append(issues, "Consider including relevant standards references")
	}

	return issues
}
