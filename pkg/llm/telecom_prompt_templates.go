package llm

import (
	"fmt"
	"strings"
	"time"
)

// TelecomPromptTemplates provides specialized prompts for telecom domain
type TelecomPromptTemplates struct {
	systemPrompts map[string]string
	userPrompts   map[string]string
	examples      map[string][]PromptExample
}

// PromptExample represents an example for few-shot prompting
type PromptExample struct {
	Input       string
	Output      string
	Explanation string
}

// TelecomContext holds domain-specific context
type TelecomContext struct {
	NetworkFunctions []NetworkFunction
	ActiveSlices     []NetworkSlice
	E2Nodes          []E2Node
	Alarms           []Alarm
	PerformanceKPIs  map[string]float64
}

// NetworkFunction represents a 5G Core network function
type NetworkFunction struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"` // AMF, SMF, UPF, etc.
	Status   string  `json:"status"`
	Load     float64 `json:"load"`
	Location string  `json:"location"`
}

// NetworkSlice represents a network slice configuration
type NetworkSlice struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"` // eMBB, URLLC, mMTC
	Status       string  `json:"status"`
	Throughput   float64 `json:"throughput"`
	Latency      float64 `json:"latency"`
	Reliability  float64 `json:"reliability"`
	ConnectedUEs int     `json:"connected_ues"`
}

// E2Node represents an E2 interface node
type E2Node struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // gNB, ng-eNB
	Status   string `json:"status"`
	RRCState string `json:"rrc_state"`
	CellID   string `json:"cell_id"`
}

// Alarm represents a network alarm
type Alarm struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewTelecomPromptTemplates creates a new instance with pre-configured templates
func NewTelecomPromptTemplates() *TelecomPromptTemplates {
	templates := &TelecomPromptTemplates{
		systemPrompts: make(map[string]string),
		userPrompts:   make(map[string]string),
		examples:      make(map[string][]PromptExample),
	}

	templates.initializeSystemPrompts()
	templates.initializeUserPrompts()
	templates.initializeExamples()

	return templates
}

// initializeSystemPrompts sets up system prompts for different intent types
func (t *TelecomPromptTemplates) initializeSystemPrompts() {
	// O-RAN Network Intent System Prompt
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

**5G Core Network Functions:**
- AMF (Access and Mobility Management Function)
- SMF (Session Management Function)
- UPF (User Plane Function)
- PCF (Policy Control Function)
- UDM (Unified Data Management)
- AUSF (Authentication Server Function)
- NSSF (Network Slice Selection Function)

**Network Slicing:**
- eMBB (Enhanced Mobile Broadband): High throughput, moderate latency
- URLLC (Ultra-Reliable Low Latency Communications): Ultra-low latency, high reliability
- mMTC (Massive Machine Type Communications): High device density, low power

Your role is to translate high-level network intents into specific O-RAN configurations that comply with:
- 3GPP specifications (TS 38.xxx series)
- O-RAN Alliance specifications
- ETSI NFV standards

Always provide technically accurate, standards-compliant configurations with proper parameter validation.`

	// 5G Core Network Intent System Prompt
	t.systemPrompts["5gc_network_intent"] = `You are a 5G Core Network specialist with expertise in:

**Service-Based Architecture (SBA):**
- Service registration and discovery
- HTTP/2-based communication
- OAuth 2.0 security framework
- Service mesh integration

**Network Functions:**
- AMF: Access and mobility management, registration, authentication
- SMF: Session management, IP allocation, policy enforcement
- UPF: User plane processing, packet forwarding, QoS enforcement
- PCF: Policy control, charging rules, QoS policies
- UDM: User data management, subscription data
- AUSF: Authentication services, security anchoring
- NSSF: Network slice selection, slice instance management

**Protocols and Interfaces:**
- N1: UE ↔ AMF (NAS signaling)
- N2: gNB ↔ AMF (NGAP)
- N3: gNB ↔ UPF (GTP-U)
- N4: SMF ↔ UPF (PFCP)
- N6: UPF ↔ Data Network
- Namf, Nsmf, Npcf, etc. (Service-based interfaces)

Generate configurations that ensure:
- Proper service mesh integration
- Scalable microservices architecture  
- Security policy compliance
- Performance optimization
- Standards compliance (3GPP TS 23.501, TS 23.502)`

	// Network Slicing Intent System Prompt
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

**QoS Framework:**
- 5QI (5G QoS Identifier) mapping
- GBR (Guaranteed Bit Rate) and Non-GBR flows
- Packet delay budget and error rates
- Priority handling and preemption

Create slice configurations that optimize:
- Resource utilization efficiency
- SLA compliance and monitoring
- Inter-slice isolation
- Dynamic scaling capabilities`

	// RAN Optimization Intent System Prompt
	t.systemPrompts["ran_optimization_intent"] = `You are a RAN optimization specialist with expertise in:

**Coverage Optimization:**
- Cell planning and site selection
- Antenna tilt and azimuth optimization
- Power control algorithms (OLPC, CLPC)
- Handover parameter tuning
- Inter-RAT mobility optimization

**Capacity Management:**
- Load balancing across cells
- Carrier aggregation configuration
- MIMO configuration optimization
- Resource block scheduling
- Admission control policies

**Quality Optimization:**
- KPI monitoring and thresholds:
  - RSRP (Reference Signal Received Power)
  - RSRQ (Reference Signal Received Quality)
  - SINR (Signal-to-Interference-plus-Noise Ratio)
  - Block Error Rate (BLER)
  - Throughput and latency metrics

**Self-Organizing Networks (SON):**
- Self-configuration of new base stations
- Self-optimization of parameters
- Self-healing for fault recovery
- Automatic neighbor relation (ANR)

**ML/AI Integration:**
- Traffic prediction models
- Anomaly detection algorithms
- Predictive maintenance
- Intelligent resource allocation

Generate optimization strategies that:
- Improve network KPIs systematically
- Minimize interference and maximize throughput
- Ensure seamless mobility experience
- Comply with regulatory requirements`
}

// initializeUserPrompts sets up user prompt templates
func (t *TelecomPromptTemplates) initializeUserPrompts() {
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

	t.userPrompts["slice_configuration"] = `Configure a network slice with the following requirements:

**Slice Requirements:**
- Type: {{.SliceType}}
- Service Level Agreement:
  - Throughput: {{.RequiredThroughput}}
  - Latency: {{.MaxLatency}}
  - Reliability: {{.MinReliability}}
  - Coverage Area: {{.CoverageArea}}
  - Expected UEs: {{.ExpectedUEs}}

**Resource Constraints:**
- Available Compute: {{.AvailableCompute}}
- Available Bandwidth: {{.AvailableBandwidth}}
- Network Functions: {{.AvailableFunctions}}

Generate:
1. **Slice Template (NST):** Complete slice definition
2. **Resource Allocation:** Compute, network, and storage requirements
3. **VNF/CNF Placement:** Optimal placement strategy
4. **QoS Configuration:** 5QI mappings and flow configurations
5. **Monitoring Setup:** KPI definitions and thresholds
6. **Scaling Policies:** Auto-scaling rules and triggers

Ensure compliance with 3GPP TS 23.501 and O-RAN specifications.`

	t.userPrompts["ran_optimization"] = `Optimize RAN performance for the following scenario:

**Optimization Objective:** {{.Objective}}

**Current Performance:**
{{range $kpi, $value := .CurrentKPIs}}
- {{$kpi}}: {{$value}}
{{end}}

**Target Performance:**
{{range $kpi, $value := .TargetKPIs}}
- {{$kpi}}: {{$value}}
{{end}}

**Network Topology:**
{{range .Sites}}
- Site {{.ID}}: Location={{.Location}}, Sectors={{.Sectors}}, Technology={{.Technology}}
{{end}}

**Constraints:**
- Budget: {{.Budget}}
- Timeline: {{.Timeline}}
- Regulatory Limits: {{.RegulatoryLimits}}

Provide:
1. **Parameter Optimization:** Specific parameter changes with justification
2. **Algorithm Selection:** Recommended optimization algorithms
3. **Implementation Timeline:** Phased rollout plan
4. **Performance Monitoring:** KPI tracking and alerting
5. **Validation Tests:** Test procedures to verify improvements
6. **Rollback Procedures:** Emergency reversion steps

Include detailed explanations of the technical rationale for each recommendation.`
}

// initializeExamples sets up few-shot learning examples
func (t *TelecomPromptTemplates) initializeExamples() {
	// O-RAN Network Intent Examples
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
		{
			Input: "Configure E2 interface between gNB and Near-RT RIC for performance monitoring",
			Output: `apiVersion: v1
kind: ConfigMap
metadata:
  name: e2-interface-config
  namespace: o-ran
data:
  e2ap.conf: |
    e2ap:
      version: "2.0"
      ran_functions:
        - function_id: 1
          function_definition: |
            {
              "ranFunctionName": "KPM",
              "ranFunctionDescription": "KPI Monitoring Function",
              "ranFunctionRevision": 1,
              "oid": "1.3.6.1.4.1.53148.1.1.2.2"
            }
          service_models:
            - name: "KPM-SM"
              version: "2.0"
              oid: "1.3.6.1.4.1.53148.1.1.2.2"
      subscriptions:
        - reporting_period: 1000  # 1 second
          kpi_list:
            - "DRB.PdcpSduBitrateDl"
            - "DRB.PdcpSduBitrateUl"
            - "RRU.PrbTotDl"
            - "RRU.PrbTotUl"
            - "QosFlow.PdcpPduVolumeDl_Filter.QCI"
      security:
        enabled: true
        certificates:
          ca_cert: "/etc/certs/ca.crt"
          client_cert: "/etc/certs/client.crt"
          client_key: "/etc/certs/client.key"`,
			Explanation: "E2 interface configuration enabling KPI monitoring with standard performance measurements, proper service model registration, and security certificates.",
		},
	}

	// 5G Core Network Intent Examples
	t.examples["5gc_network_intent"] = []PromptExample{
		{
			Input: "Deploy 5G Core with network slicing support for eMBB and URLLC services",
			Output: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf-deployment
  namespace: 5gc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: amf
  template:
    metadata:
      labels:
        app: amf
    spec:
      containers:
      - name: amf
        image: 5gc/amf:v1.5.0
        ports:
        - containerPort: 8080
        - containerPort: 38412  # N2 interface
        env:
        - name: AMF_CONFIG
          value: |
            amf:
              sbi:
                scheme: https
                register_ipv4: amf.5gc.svc.cluster.local
                bind_ipv4: 0.0.0.0
                port: 8080
              ngap:
                addr: 0.0.0.0
                port: 38412
              metrics:
                addr: 0.0.0.0
                port: 9090
              nssai:
                - sst: 1  # eMBB
                  sd: "0x000001"
                - sst: 2  # URLLC  
                  sd: "0x000002"
              plmn_support:
                - plmn_id:
                    mcc: "001"
                    mnc: "01"
                  s_nssai:
                    - sst: 1
                      sd: "0x000001"
                    - sst: 2
                      sd: "0x000002"
              security:
                integrity_order: [NIA2, NIA1, NIA0]
                ciphering_order: [NEA2, NEA1, NEA0]`,
			Explanation: "AMF deployment with network slicing support for both eMBB and URLLC slice types, including proper NSSAI configuration and security parameters.",
		},
	}

	// Network Slicing Examples
	t.examples["network_slicing_intent"] = []PromptExample{
		{
			Input: "Create URLLC slice for industrial automation with 1ms latency requirement",
			Output: `apiVersion: slice.nephoran.io/v1
kind: NetworkSlice
metadata:
  name: urllc-industrial-slice
  namespace: slicing
spec:
  sliceProfile:
    sst: 2  # URLLC
    sd: "0x000010"
    dnn: "industrial.urllc"
  serviceRequirements:
    latency: 1000  # microseconds
    reliability: 99.999
    throughput:
      guaranteed: 10  # Mbps
      maximum: 50
    coverage:
      area_type: "industrial_zone"
      mobility: "stationary"
  resourceAllocation:
    compute:
      upf_instances: 2
      cpu_cores: 8
      memory_gb: 16
    network:
      dedicated_bandwidth: 100  # MHz
      priority: 1  # highest priority
      preemption_capability: true
  qosProfile:
    5qi: 82  # Industrial automation
    priority: 15
    packet_delay_budget: 1  # ms
    packet_error_rate: 1e-6
  nfDeployment:
    upf:
      placement: "edge"
      replicas: 2
      resources:
        cpu: "4"
        memory: "8Gi"
      config:
        buffer_size: "minimal"
        processing_priority: "realtime"
    smf:
      placement: "regional"
      replicas: 1
      resources:
        cpu: "2" 
        memory: "4Gi"`,
			Explanation: "URLLC slice configuration optimized for industrial automation with ultra-low latency, high reliability, and edge UPF placement for minimal processing delay.",
		},
	}

	// RAN Optimization Examples
	t.examples["ran_optimization_intent"] = []PromptExample{
		{
			Input: "Optimize handover parameters to reduce call drops in high mobility scenario",
			Output: `apiVersion: ran.nephoran.io/v1
kind: HandoverOptimization
metadata:
  name: mobility-optimization
  namespace: ran
spec:
  targetScenario: "high_mobility"
  optimizationObjective:
    primary: "minimize_call_drops"
    secondary: "minimize_handover_delay"
  cellParameters:
    - parameter: "a3Offset"
      current_value: 3  # dB
      optimized_value: 2  # dB  
      rationale: "Reduce offset to trigger handover earlier in high mobility"
    - parameter: "hysteresis"
      current_value: 2  # dB
      optimized_value: 1  # dB
      rationale: "Lower hysteresis for faster handover decisions"
    - parameter: "timeToTrigger"
      current_value: 320  # ms
      optimized_value: 160  # ms
      rationale: "Shorter TTT for high mobility scenarios"
  measurementConfiguration:
    a1Threshold: -110  # dBm
    a2Threshold: -115  # dBm
    a3Offset: 2        # dB (optimized)
    a5Threshold1: -110 # dBm
    a5Threshold2: -105 # dBm
  algorithmParameters:
    velocityThreshold: 60  # km/h
    rsrpMargin: 3         # dB
    loadBalancingWeight: 0.3
  validationKPIs:
    handover_success_rate: ">95%"
    call_drop_rate: "<1%"
    handover_latency: "<50ms"
  rollbackTriggers:
    call_drop_rate_increase: ">20%"
    handover_ping_pong_rate: ">5%"`,
			Explanation: "Handover optimization configuration for high mobility scenarios with adjusted measurement parameters, reduced thresholds for faster decisions, and clear validation criteria.",
		},
	}
}

// GetSystemPrompt returns the system prompt for a given intent type
func (t *TelecomPromptTemplates) GetSystemPrompt(intentType string) string {
	if prompt, exists := t.systemPrompts[intentType]; exists {
		return prompt
	}
	return t.systemPrompts["oran_network_intent"] // default
}

// GetUserPrompt returns the user prompt template for a given intent type
func (t *TelecomPromptTemplates) GetUserPrompt(intentType string) string {
	if prompt, exists := t.userPrompts[intentType]; exists {
		return prompt
	}
	return t.userPrompts["network_intent_processing"] // default
}

// GetExamples returns examples for few-shot prompting
func (t *TelecomPromptTemplates) GetExamples(intentType string) []PromptExample {
	if examples, exists := t.examples[intentType]; exists {
		return examples
	}
	return []PromptExample{} // empty if not found
}

// BuildPrompt constructs a complete prompt with system message, examples, and user input
func (t *TelecomPromptTemplates) BuildPrompt(intentType string, context TelecomContext, userIntent string) string {
	var builder strings.Builder

	// Add system prompt
	systemPrompt := t.GetSystemPrompt(intentType)
	builder.WriteString("## System Instructions\n")
	builder.WriteString(systemPrompt)
	builder.WriteString("\n\n")

	// Add examples for few-shot learning
	examples := t.GetExamples(intentType)
	if len(examples) > 0 {
		builder.WriteString("## Examples\n\n")
		for i, example := range examples {
			builder.WriteString(fmt.Sprintf("### Example %d\n", i+1))
			builder.WriteString(fmt.Sprintf("**Input:** %s\n\n", example.Input))
			builder.WriteString(fmt.Sprintf("**Output:**\n```yaml\n%s\n```\n\n", example.Output))
			if example.Explanation != "" {
				builder.WriteString(fmt.Sprintf("**Explanation:** %s\n\n", example.Explanation))
			}
		}
	}

	// Add current context
	builder.WriteString("## Current Network Context\n\n")
	builder.WriteString(t.formatContext(context))
	builder.WriteString("\n")

	// Add user intent
	builder.WriteString("## User Intent\n")
	builder.WriteString(userIntent)
	builder.WriteString("\n\n")

	// Add response format instructions
	builder.WriteString("## Response Requirements\n")
	builder.WriteString("Please provide your response in valid YAML format with:\n")
	builder.WriteString("1. Detailed configuration parameters\n")
	builder.WriteString("2. Explanatory comments for each major section\n")
	builder.WriteString("3. Compliance references to relevant standards\n")
	builder.WriteString("4. Risk assessment and mitigation strategies\n")
	builder.WriteString("5. Validation and testing recommendations\n")

	return builder.String()
}

// formatContext formats the telecom context into a readable string
func (t *TelecomPromptTemplates) formatContext(context TelecomContext) string {
	var builder strings.Builder

	// Network Functions
	if len(context.NetworkFunctions) > 0 {
		builder.WriteString("**Network Functions:**\n")
		for _, nf := range context.NetworkFunctions {
			builder.WriteString(fmt.Sprintf("- %s (%s): Status=%s, Load=%.1f%%, Location=%s\n",
				nf.Name, nf.Type, nf.Status, nf.Load, nf.Location))
		}
		builder.WriteString("\n")
	}

	// Active Slices
	if len(context.ActiveSlices) > 0 {
		builder.WriteString("**Active Network Slices:**\n")
		for _, slice := range context.ActiveSlices {
			builder.WriteString(fmt.Sprintf("- Slice %s (%s): Throughput=%.1fMbps, Latency=%.1fms, Reliability=%.3f%%, UEs=%d\n",
				slice.ID, slice.Type, slice.Throughput, slice.Latency, slice.Reliability, slice.ConnectedUEs))
		}
		builder.WriteString("\n")
	}

	// E2 Nodes
	if len(context.E2Nodes) > 0 {
		builder.WriteString("**E2 Nodes:**\n")
		for _, node := range context.E2Nodes {
			builder.WriteString(fmt.Sprintf("- %s (%s): Status=%s, RRC=%s, Cell=%s\n",
				node.ID, node.Type, node.Status, node.RRCState, node.CellID))
		}
		builder.WriteString("\n")
	}

	// Alarms
	if len(context.Alarms) > 0 {
		builder.WriteString("**Active Alarms:**\n")
		for _, alarm := range context.Alarms {
			builder.WriteString(fmt.Sprintf("- [%s] %s: %s (%s)\n",
				alarm.Severity, alarm.Source, alarm.Description, alarm.Timestamp.Format("2006-01-02 15:04:05")))
		}
		builder.WriteString("\n")
	}

	// Performance KPIs
	if len(context.PerformanceKPIs) > 0 {
		builder.WriteString("**Performance KPIs:**\n")
		for kpi, value := range context.PerformanceKPIs {
			builder.WriteString(fmt.Sprintf("- %s: %.3f\n", kpi, value))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// ValidatePrompt validates a prompt for telecom domain compliance
func (t *TelecomPromptTemplates) ValidatePrompt(prompt string) []string {
	var issues []string

	// Check for required telecom keywords
	requiredKeywords := []string{"network", "configuration", "interface"}
	for _, keyword := range requiredKeywords {
		if !strings.Contains(strings.ToLower(prompt), keyword) {
			issues = append(issues, fmt.Sprintf("Missing required keyword: %s", keyword))
		}
	}

	// Check prompt length
	if len(prompt) < 100 {
		issues = append(issues, "Prompt too short for complex telecom intent")
	}

	if len(prompt) > 10000 {
		issues = append(issues, "Prompt too long, may exceed token limits")
	}

	// Check for standards references
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
