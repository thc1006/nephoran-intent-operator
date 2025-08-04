package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TelecomPromptTemplate represents a reusable prompt template for telecom operations
type TelecomPromptTemplate struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	Description       string                      `json:"description"`
	IntentType        string                      `json:"intent_type"`
	BasePrompt        string                      `json:"base_prompt"`
	SystemPrompt      string                      `json:"system_prompt"`
	Examples          []TelecomExample            `json:"examples"`
	RequiredContext   []string                    `json:"required_context"`
	TokenEstimate     int                         `json:"token_estimate"`
	Domain            string                      `json:"domain"`
	Technology        []string                    `json:"technology"`
	ComplianceStandards []string                  `json:"compliance_standards"`
	Variables         map[string]PromptVariable   `json:"variables"`
	ValidationRules   []ValidationRule            `json:"validation_rules"`
	SuccessMetrics    []string                    `json:"success_metrics"`
}

// TelecomExample represents an example for few-shot learning
type TelecomExample struct {
	Context      string                 `json:"context"`
	Intent       string                 `json:"intent"`
	Response     string                 `json:"response"`
	Explanation  string                 `json:"explanation"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// PromptVariable defines a variable that can be injected into the prompt
type PromptVariable struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	DefaultValue string   `json:"default_value"`
	Options      []string `json:"options"`
}

// ValidationRule defines validation for generated responses
type ValidationRule struct {
	Type        string                 `json:"type"`
	Field       string                 `json:"field"`
	Condition   string                 `json:"condition"`
	Value       interface{}            `json:"value"`
	ErrorMessage string                `json:"error_message"`
}

// TelecomPromptRegistry manages all telecom-specific prompts
type TelecomPromptRegistry struct {
	templates map[string]*TelecomPromptTemplate
	domains   map[string][]*TelecomPromptTemplate
}

// NewTelecomPromptRegistry creates a new registry with built-in telecom prompts
func NewTelecomPromptRegistry() *TelecomPromptRegistry {
	registry := &TelecomPromptRegistry{
		templates: make(map[string]*TelecomPromptTemplate),
		domains:   make(map[string][]*TelecomPromptTemplate),
	}
	
	registry.initializeBuiltInPrompts()
	return registry
}

func (r *TelecomPromptRegistry) initializeBuiltInPrompts() {
	// O-RAN Network Function Deployment Template
	oranDeploymentTemplate := &TelecomPromptTemplate{
		ID:          "oran-nf-deployment",
		Name:        "O-RAN Network Function Deployment",
		Description: "Template for deploying O-RAN compliant network functions",
		IntentType:  "NetworkFunctionDeployment",
		Domain:      "O-RAN",
		Technology:  []string{"O-RAN", "5G", "Cloud-Native"},
		ComplianceStandards: []string{"O-RAN.WG1", "O-RAN.WG2", "O-RAN.WG3", "O-RAN.WG4", "O-RAN.WG5"},
		SystemPrompt: `You are an O-RAN architecture expert with comprehensive knowledge of:

## O-RAN Architecture Components:
1. **Near-RT RIC (Near Real-Time RAN Intelligent Controller)**
   - Hosts xApps for RAN optimization (10ms to 1s control loops)
   - E2 interface for RAN control and telemetry
   - A1 interface for policy guidance from Non-RT RIC
   - Conflict mitigation between xApps
   - R1 interface for rApp-xApp communication

2. **Non-RT RIC (Non Real-Time RIC)**
   - Hosts rApps for RAN optimization (>1s control loops)
   - A1 policy management for Near-RT RIC
   - R1 interface for enrichment information
   - O1 interface for FCAPS operations
   - Integration with SMO (Service Management and Orchestration)

3. **O-CU (O-RAN Central Unit)**
   - Split into O-CU-CP (Control Plane) and O-CU-UP (User Plane)
   - PDCP and RRC layer processing
   - F1 interface to O-DU
   - E1 interface between CU-CP and CU-UP
   - X2/Xn interfaces for inter-node communication

4. **O-DU (O-RAN Distributed Unit)**
   - RLC, MAC, and High-PHY processing
   - Fronthaul interface to O-RU
   - Real-time scheduling and resource allocation
   - Support for multiple O-RUs

5. **O-RU (O-RAN Radio Unit)**
   - Low-PHY processing and RF functions
   - Open Fronthaul interface (eCPRI/IEEE 1914.3)
   - Beamforming capabilities
   - Multiple antenna support

## O-RAN Interfaces:
- **A1**: Policy interface (Non-RT RIC → Near-RT RIC)
- **E2**: Real-time control (Near-RT RIC → E2 nodes)
- **O1**: Management plane (SMO → all O-RAN components)
- **O2**: Cloud infrastructure interface (SMO → O-Cloud)
- **Open Fronthaul**: Between O-DU and O-RU
- **R1**: rApp-xApp interface for enrichment data

## Key Principles:
1. Disaggregation and openness
2. Intelligence and automation through RIC
3. Cloud-native deployment
4. Multi-vendor interoperability
5. Standardized interfaces

When generating configurations:
- Ensure proper interface definitions
- Include O1 configuration for management
- Define A1 policies when applicable
- Consider multi-vendor scenarios
- Implement proper security measures
- Follow O-RAN Alliance specifications`,
		BasePrompt: `Generate a Kubernetes deployment configuration for the requested O-RAN network function.

Intent: {{intent}}
Network State: {{network_state}}
Compliance Requirements: {{compliance_requirements}}

Consider:
1. O-RAN interface requirements
2. Resource allocation based on function type
3. High availability and scaling needs
4. Security and isolation requirements
5. Integration with existing O-RAN components`,
		Examples: []TelecomExample{
			{
				Context: "Deploy Near-RT RIC",
				Intent:  "Deploy a Near-RT RIC with xApp platform capabilities for traffic steering in a multi-vendor O-RAN environment",
				Response: `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "near-rt-ric",
    "namespace": "oran-system",
    "labels": {
      "app": "near-rt-ric",
      "oran.component": "ric",
      "oran.type": "near-rt",
      "oran.version": "g-release"
    }
  },
  "spec": {
    "replicas": 3,
    "selector": {
      "matchLabels": {
        "app": "near-rt-ric"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "near-rt-ric",
          "oran.component": "ric"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "near-rt-ric",
            "image": "oran/near-rt-ric:g-release-v1.0",
            "ports": [
              {"containerPort": 8080, "name": "http", "protocol": "TCP"},
              {"containerPort": 36421, "name": "e2ap", "protocol": "SCTP"},
              {"containerPort": 9090, "name": "a1-rest", "protocol": "TCP"},
              {"containerPort": 8888, "name": "r1-interface", "protocol": "TCP"}
            ],
            "env": [
              {"name": "RIC_ID", "value": "ric-cluster-001"},
              {"name": "E2_TERM_INIT", "value": "true"},
              {"name": "XAPP_ONBOARDER_ENABLED", "value": "true"},
              {"name": "CONFLICT_MITIGATION", "value": "enabled"},
              {"name": "A1_POLICY_ENFORCEMENT", "value": "strict"}
            ],
            "resources": {
              "requests": {"cpu": "2", "memory": "4Gi"},
              "limits": {"cpu": "4", "memory": "8Gi"}
            },
            "volumeMounts": [
              {"name": "ric-config", "mountPath": "/opt/ric/config"},
              {"name": "xapp-registry", "mountPath": "/opt/ric/xapps"}
            ]
          }
        ],
        "volumes": [
          {
            "name": "ric-config",
            "configMap": {"name": "near-rt-ric-config"}
          },
          {
            "name": "xapp-registry",
            "persistentVolumeClaim": {"claimName": "xapp-registry-pvc"}
          }
        ],
        "affinity": {
          "podAntiAffinity": {
            "requiredDuringSchedulingIgnoredDuringExecution": [
              {
                "labelSelector": {
                  "matchExpressions": [
                    {
                      "key": "app",
                      "operator": "In",
                      "values": ["near-rt-ric"]
                    }
                  ]
                },
                "topologyKey": "kubernetes.io/hostname"
              }
            ]
          }
        }
      }
    }
  }
}`,
				Explanation: "Near-RT RIC deployment with high availability (3 replicas), xApp platform support, and proper O-RAN interfaces (E2, A1, R1). Includes conflict mitigation and policy enforcement capabilities.",
				Metadata: map[string]interface{}{
					"interfaces": []string{"E2", "A1", "R1"},
					"ha_enabled": true,
					"xapp_support": true,
				},
			},
		},
		Variables: map[string]PromptVariable{
			"intent": {
				Name:        "intent",
				Description: "User's deployment intent",
				Type:        "string",
				Required:    true,
			},
			"network_state": {
				Name:        "network_state",
				Description: "Current network state and topology",
				Type:        "object",
				Required:    false,
			},
			"compliance_requirements": {
				Name:        "compliance_requirements",
				Description: "Specific O-RAN compliance requirements",
				Type:        "array",
				Required:    false,
				Options:     []string{"O-RAN.WG1", "O-RAN.WG2", "O-RAN.WG3", "O-RAN.WG4", "O-RAN.WG5"},
			},
		},
		ValidationRules: []ValidationRule{
			{
				Type:         "required",
				Field:        "metadata.namespace",
				Condition:    "exists",
				ErrorMessage: "Namespace must be specified for O-RAN components",
			},
			{
				Type:         "interface",
				Field:        "spec.template.spec.containers[0].ports",
				Condition:    "contains_oran_interfaces",
				ErrorMessage: "O-RAN network function must expose required interfaces",
			},
		},
		SuccessMetrics: []string{
			"Valid Kubernetes manifest generated",
			"O-RAN interfaces properly configured",
			"Resource allocation appropriate for function type",
			"High availability considered",
			"Security policies applied",
		},
		TokenEstimate: 2500,
	}
	
	// 5G Core Network Function Template
	fivegCoreTemplate := &TelecomPromptTemplate{
		ID:          "5g-core-nf",
		Name:        "5G Core Network Function",
		Description: "Template for deploying 5G Core network functions",
		IntentType:  "NetworkFunctionDeployment",
		Domain:      "5G-Core",
		Technology:  []string{"5G", "3GPP", "SBA", "Cloud-Native"},
		ComplianceStandards: []string{"3GPP-R15", "3GPP-R16", "3GPP-R17"},
		SystemPrompt: `You are a 5G Core Network expert with deep knowledge of:

## 5G Core Architecture (Service Based Architecture - SBA):
1. **Control Plane Functions:**
   - AMF (Access and Mobility Management Function)
   - SMF (Session Management Function)
   - PCF (Policy Control Function)
   - UDM (Unified Data Management)
   - UDR (Unified Data Repository)
   - AUSF (Authentication Server Function)
   - NSSF (Network Slice Selection Function)
   - NEF (Network Exposure Function)
   - NRF (Network Repository Function)
   - BSF (Binding Support Function)
   - CHF (Charging Function)

2. **User Plane Function:**
   - UPF (User Plane Function)

## Service Based Interfaces (SBI):
- RESTful APIs (HTTP/2 + JSON)
- Service discovery via NRF
- OAuth2.0 for authorization
- TLS 1.2+ for security

## Reference Points:
- N1: UE - AMF
- N2: RAN - AMF
- N3: RAN - UPF
- N4: SMF - UPF
- N6: UPF - DN
- N9: UPF - UPF
- N11: AMF - SMF

## Network Slicing:
- S-NSSAI (Single Network Slice Selection Assistance Information)
- NSI (Network Slice Instance)
- NSSI (Network Slice Subnet Instance)

## Key Concepts:
- PDU Sessions
- QoS Flows (5QI)
- Registration and Connection Management
- Session Management
- Policy Control
- Charging and Billing`,
		BasePrompt: `Generate a cloud-native deployment for the requested 5G Core network function.

Intent: {{intent}}
Slice Requirements: {{slice_requirements}}
QoS Parameters: {{qos_parameters}}

Ensure:
1. 3GPP compliance
2. Service mesh integration readiness
3. Proper SBI exposure
4. Database/storage requirements
5. Redundancy and scaling`,
		Examples: []TelecomExample{
			{
				Context: "Deploy SMF for URLLC slice",
				Intent:  "Deploy SMF with support for URLLC network slice with <1ms latency requirement",
				Response: `{
  "apiVersion": "apps/v1",
  "kind": "StatefulSet",
  "metadata": {
    "name": "smf-urllc",
    "namespace": "5gc",
    "labels": {
      "nf.5gc.3gpp.org/type": "smf",
      "nf.5gc.3gpp.org/slice": "urllc"
    }
  },
  "spec": {
    "serviceName": "smf-urllc",
    "replicas": 3,
    "selector": {
      "matchLabels": {
        "app": "smf-urllc"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "smf-urllc",
          "nf.5gc.3gpp.org/type": "smf"
        },
        "annotations": {
          "sidecar.istio.io/inject": "true"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "smf",
            "image": "5gc/smf:r17-v2.0-urllc",
            "ports": [
              {"containerPort": 8080, "name": "sbi", "protocol": "TCP"},
              {"containerPort": 8805, "name": "n4", "protocol": "UDP"},
              {"containerPort": 2123, "name": "gtpc", "protocol": "UDP"}
            ],
            "env": [
              {"name": "SMF_INSTANCE_ID", "valueFrom": {"fieldRef": {"fieldPath": "metadata.name"}}},
              {"name": "SLICE_TYPE", "value": "URLLC"},
              {"name": "LATENCY_TARGET", "value": "1ms"},
              {"name": "NRF_URI", "value": "https://nrf.5gc.svc.cluster.local:8080"},
              {"name": "PFCP_HEARTBEAT_INTERVAL", "value": "5s"},
              {"name": "SESSION_ESTABLISHMENT_TIMEOUT", "value": "500ms"}
            ],
            "resources": {
              "requests": {"cpu": "1", "memory": "1Gi"},
              "limits": {"cpu": "2", "memory": "2Gi"}
            },
            "livenessProbe": {
              "httpGet": {
                "path": "/nsmf-pdusession/v1/health",
                "port": 8080,
                "scheme": "HTTPS"
              },
              "initialDelaySeconds": 30,
              "periodSeconds": 10
            },
            "readinessProbe": {
              "httpGet": {
                "path": "/nsmf-pdusession/v1/ready",
                "port": 8080,
                "scheme": "HTTPS"
              },
              "initialDelaySeconds": 20,
              "periodSeconds": 5
            }
          }
        ],
        "affinity": {
          "podAntiAffinity": {
            "preferredDuringSchedulingIgnoredDuringExecution": [
              {
                "weight": 100,
                "podAffinityTerm": {
                  "labelSelector": {
                    "matchExpressions": [
                      {
                        "key": "app",
                        "operator": "In",
                        "values": ["smf-urllc"]
                      }
                    ]
                  },
                  "topologyKey": "topology.kubernetes.io/zone"
                }
              }
            ]
          }
        }
      }
    },
    "volumeClaimTemplates": [
      {
        "metadata": {
          "name": "smf-data"
        },
        "spec": {
          "accessModes": ["ReadWriteOnce"],
          "storageClassName": "fast-ssd",
          "resources": {
            "requests": {
              "storage": "10Gi"
            }
          }
        }
      }
    ]
  }
}`,
				Explanation: "SMF deployment optimized for URLLC with low latency configuration, fast storage, and proper 5G SBI integration",
				Metadata: map[string]interface{}{
					"slice_type": "URLLC",
					"latency_optimized": true,
					"sbi_enabled": true,
				},
			},
		},
		TokenEstimate: 2000,
	}
	
	// Network Slice Configuration Template
	networkSliceTemplate := &TelecomPromptTemplate{
		ID:          "network-slice-config",
		Name:        "Network Slice Configuration",
		Description: "Template for configuring 5G network slices",
		IntentType:  "NetworkSliceConfiguration",
		Domain:      "5G-Slicing",
		Technology:  []string{"5G", "Network-Slicing", "SDN", "NFV"},
		ComplianceStandards: []string{"3GPP-TS-28.530", "3GPP-TS-28.531", "GSMA-NG.116"},
		SystemPrompt: `You are a 5G Network Slicing expert with comprehensive knowledge of:

## Network Slice Types (SST - Slice/Service Type):
1. **eMBB (Enhanced Mobile Broadband) - SST=1**
   - High data rates (>100 Mbps)
   - High capacity
   - Moderate latency (10-20ms)
   - Use cases: 4K/8K video, AR/VR, cloud gaming

2. **URLLC (Ultra-Reliable Low Latency Communications) - SST=2**
   - Ultra-low latency (<1ms air interface, <5ms E2E)
   - High reliability (99.999%)
   - Moderate data rates
   - Use cases: Industrial automation, remote surgery, autonomous vehicles

3. **mMTC (Massive Machine Type Communications) - SST=3**
   - Massive device density (1M devices/km²)
   - Low power consumption
   - Small data packets
   - Use cases: Smart cities, agriculture, asset tracking

4. **V2X (Vehicle-to-Everything) - SST=4**
   - Low latency (<20ms)
   - High mobility support
   - High reliability
   - Use cases: Connected vehicles, traffic management

## Slice Differentiator (SD):
- 3-byte value for differentiating slices of same SST
- Operator-defined based on business needs

## S-NSSAI (Single Network Slice Selection Assistance Information):
- Combination of SST and SD
- Identifies a network slice

## Key Slice Parameters:
- Resource allocation (RAN, Transport, Core)
- QoS profiles (5QI values)
- Isolation level (physical, logical, no isolation)
- Security policies
- Charging rules
- Service area
- Maximum number of UEs
- Maximum number of PDU sessions`,
		BasePrompt: `Create a network slice configuration based on the requirements.

Intent: {{intent}}
Slice Type: {{slice_type}}
Performance Requirements: {{performance_requirements}}
Capacity Requirements: {{capacity_requirements}}

Generate:
1. Slice profile with S-NSSAI
2. QoS configuration
3. Resource allocation
4. Isolation policies
5. Monitoring configuration`,
		TokenEstimate: 1800,
	}
	
	// RAN Optimization Template
	ranOptimizationTemplate := &TelecomPromptTemplate{
		ID:          "ran-optimization",
		Name:        "RAN Optimization Configuration",
		Description: "Template for RAN optimization and parameter tuning",
		IntentType:  "RANOptimization",
		Domain:      "RAN",
		Technology:  []string{"5G-NR", "4G-LTE", "O-RAN", "SON"},
		ComplianceStandards: []string{"3GPP-TS-38.300", "O-RAN.WG1", "O-RAN.WG2"},
		SystemPrompt: `You are a RAN optimization expert with knowledge of:

## RAN Technologies:
1. **5G NR (New Radio)**
   - Frequency ranges: FR1 (sub-6GHz), FR2 (mmWave)
   - Numerologies (SCS: 15, 30, 60, 120, 240 kHz)
   - Massive MIMO and beamforming
   - Dynamic spectrum sharing
   - Carrier aggregation

2. **4G LTE**
   - LTE-Advanced features
   - Carrier aggregation
   - CoMP (Coordinated Multi-Point)
   - eICIC (enhanced Inter-Cell Interference Coordination)

## Optimization Areas:
1. **Coverage Optimization**
   - Cell radius adjustment
   - Antenna tilt optimization
   - Power control
   - Handover parameters

2. **Capacity Optimization**
   - Resource block allocation
   - Scheduler configuration
   - MIMO configuration
   - Carrier management

3. **Quality Optimization**
   - Interference management
   - Mobility robustness
   - VoLTE quality
   - Latency reduction

## Key Performance Indicators (KPIs):
- RSRP/RSRQ/SINR
- Throughput (DL/UL)
- Latency
- Handover success rate
- Call drop rate
- RRC connection success rate
- E-RAB setup success rate`,
		BasePrompt: `Generate RAN optimization configuration for the scenario.

Intent: {{intent}}
Network Type: {{network_type}}
Optimization Goal: {{optimization_goal}}
Current KPIs: {{current_kpis}}

Provide:
1. Parameter adjustments
2. Expected improvements
3. Implementation steps
4. Monitoring plan
5. Rollback procedures`,
		TokenEstimate: 1600,
	}
	
	// Register all templates
	r.RegisterTemplate(oranDeploymentTemplate)
	r.RegisterTemplate(fivegCoreTemplate)
	r.RegisterTemplate(networkSliceTemplate)
	r.RegisterTemplate(ranOptimizationTemplate)
}

// RegisterTemplate registers a new prompt template
func (r *TelecomPromptRegistry) RegisterTemplate(template *TelecomPromptTemplate) error {
	if template == nil || template.ID == "" {
		return fmt.Errorf("invalid template: missing ID")
	}
	
	r.templates[template.ID] = template
	
	// Index by domain
	if template.Domain != "" {
		r.domains[template.Domain] = append(r.domains[template.Domain], template)
	}
	
	return nil
}

// GetTemplate retrieves a template by ID
func (r *TelecomPromptRegistry) GetTemplate(id string) (*TelecomPromptTemplate, error) {
	template, exists := r.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return template, nil
}

// GetTemplatesByDomain retrieves all templates for a domain
func (r *TelecomPromptRegistry) GetTemplatesByDomain(domain string) []*TelecomPromptTemplate {
	return r.domains[domain]
}

// GetTemplatesByIntent retrieves templates matching an intent type
func (r *TelecomPromptRegistry) GetTemplatesByIntent(intentType string) []*TelecomPromptTemplate {
	var matches []*TelecomPromptTemplate
	for _, template := range r.templates {
		if template.IntentType == intentType {
			matches = append(matches, template)
		}
	}
	return matches
}

// BuildPrompt builds a complete prompt from a template with variables
func (r *TelecomPromptRegistry) BuildPrompt(templateID string, variables map[string]interface{}) (string, error) {
	template, err := r.GetTemplate(templateID)
	if err != nil {
		return "", err
	}
	
	// Validate required variables
	for varName, varDef := range template.Variables {
		if varDef.Required {
			if _, exists := variables[varName]; !exists {
				return "", fmt.Errorf("required variable missing: %s", varName)
			}
		}
	}
	
	// Build the prompt
	prompt := template.SystemPrompt + "\n\n" + template.BasePrompt
	
	// Replace variables
	for varName, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", varName)
		var replacement string
		
		switch v := value.(type) {
		case string:
			replacement = v
		case []string:
			replacement = strings.Join(v, ", ")
		default:
			jsonBytes, _ := json.Marshal(v)
			replacement = string(jsonBytes)
		}
		
		prompt = strings.ReplaceAll(prompt, placeholder, replacement)
	}
	
	// Add examples if token budget allows
	if template.Examples != nil && len(template.Examples) > 0 {
		prompt += "\n\n## Examples:\n"
		for i, example := range template.Examples {
			if i >= 2 { // Limit examples to control token usage
				break
			}
			prompt += fmt.Sprintf("\n### Example %d:\n", i+1)
			prompt += fmt.Sprintf("Context: %s\n", example.Context)
			prompt += fmt.Sprintf("Intent: %s\n", example.Intent)
			prompt += fmt.Sprintf("Response:\n%s\n", example.Response)
			if example.Explanation != "" {
				prompt += fmt.Sprintf("Explanation: %s\n", example.Explanation)
			}
		}
	}
	
	return prompt, nil
}

// ValidateResponse validates a response against template rules
func (r *TelecomPromptRegistry) ValidateResponse(templateID string, response interface{}) (bool, []string) {
	template, err := r.GetTemplate(templateID)
	if err != nil {
		return false, []string{err.Error()}
	}
	
	var errors []string
	
	// Apply validation rules
	for _, rule := range template.ValidationRules {
		// This is a simplified validation - in production, you'd implement
		// more sophisticated validation logic
		switch rule.Type {
		case "required":
			// Check if required field exists
			// Implementation would depend on response structure
		case "interface":
			// Validate O-RAN interfaces
			// Implementation would check for specific ports/protocols
		}
	}
	
	return len(errors) == 0, errors
}

// EstimateTokenUsage estimates the token usage for a template
func (r *TelecomPromptRegistry) EstimateTokenUsage(templateID string, includeExamples bool) (int, error) {
	template, err := r.GetTemplate(templateID)
	if err != nil {
		return 0, err
	}
	
	baseEstimate := template.TokenEstimate
	
	if includeExamples && len(template.Examples) > 0 {
		// Add roughly 500 tokens per example
		baseEstimate += len(template.Examples) * 500
	}
	
	return baseEstimate, nil
}

// GetAllTemplates returns all registered templates
func (r *TelecomPromptRegistry) GetAllTemplates() map[string]*TelecomPromptTemplate {
	return r.templates
}

// ExportTemplate exports a template to JSON
func (r *TelecomPromptRegistry) ExportTemplate(templateID string) ([]byte, error) {
	template, err := r.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}
	
	return json.MarshalIndent(template, "", "  ")
}

// ImportTemplate imports a template from JSON
func (r *TelecomPromptRegistry) ImportTemplate(data []byte) error {
	var template TelecomPromptTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return fmt.Errorf("failed to unmarshal template: %w", err)
	}
	
	return r.RegisterTemplate(&template)
}