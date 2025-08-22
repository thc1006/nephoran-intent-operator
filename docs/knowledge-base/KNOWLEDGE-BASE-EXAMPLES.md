# Knowledge Base Examples for Nephoran Intent Operator
## 3GPP/O-RAN Specification Queries and Expected Responses

### Overview

This document provides comprehensive examples of knowledge base queries and expected responses for the Nephoran Intent Operator RAG system. The examples demonstrate how the system processes natural language intents related to telecommunications specifications, network function deployments, and O-RAN architecture components.

The knowledge base includes comprehensive telecommunications domain documentation covering:
- **3GPP Technical Specifications**: Complete coverage of 5G Core, RAN, and interface specifications
- **O-RAN Alliance Specifications**: Working Group documents, use cases, and architecture definitions
- **Network Function Definitions**: AMF, SMF, UPF, gNB, CU, DU, and other network function specifications
- **Interface Specifications**: A1, O1, O2, E1, E2, F1, N1-N6, and other standardized interfaces
- **Deployment Patterns**: Cloud-native deployment, containerization, and Kubernetes orchestration guidelines
- **Configuration Templates**: Standard configuration examples for network functions and interfaces

### Table of Contents

1. [3GPP Core Network Specifications](#3gpp-core-network-specifications)
2. [O-RAN Architecture and Components](#o-ran-architecture-and-components)
3. [Network Function Deployment](#network-function-deployment)
4. [Interface Configuration](#interface-configuration)
5. [Performance Optimization](#performance-optimization)
6. [Troubleshooting and Monitoring](#troubleshooting-and-monitoring)
7. [Network Slicing Configuration](#network-slicing-configuration)
8. [Security and Authentication](#security-and-authentication)
9. [Complex Multi-Domain Queries](#complex-multi-domain-queries)
10. [Intent Pattern Recognition](#intent-pattern-recognition)

## 3GPP Core Network Specifications

### Example 1: AMF Registration Procedure

**Query:**
```
"Explain the AMF registration procedure for a UE in a 5G network according to 3GPP specifications"
```

**Expected RAG Response:**
```json
{
  "intent_id": "amf-reg-001",
  "original_intent": "Explain the AMF registration procedure for a UE in a 5G network according to 3GPP specifications",
  "structured_output": {
    "type": "TechnicalExplanation",
    "domain": "5G Core",
    "specification_references": [
      "3GPP TS 23.501 - 5G System Architecture",
      "3GPP TS 23.502 - Procedures for the 5G System"
    ],
    "explanation": {
      "overview": "The AMF (Access and Mobility Management Function) registration procedure is a fundamental process in 5G networks that establishes the initial connection between a UE and the 5G Core network.",
      "procedure_steps": [
        {
          "step": 1,
          "description": "UE initiates registration request via N1 interface",
          "message": "Registration Request",
          "components": ["UE", "gNB", "AMF"]
        },
        {
          "step": 2,
          "description": "AMF performs identity verification and authentication",
          "message": "Authentication Request/Response",
          "components": ["AMF", "AUSF", "UDM"]
        },
        {
          "step": 3,
          "description": "Security mode establishment and NAS security activation",
          "message": "Security Mode Command/Complete",
          "components": ["AMF", "UE"]
        },
        {
          "step": 4,
          "description": "Subscription data retrieval and validation",
          "message": "Update Location Request/Answer",
          "components": ["AMF", "UDM", "UDR"]
        },
        {
          "step": 5,
          "description": "Registration acceptance and context establishment",
          "message": "Registration Accept",
          "components": ["AMF", "UE"]
        }
      ],
      "interfaces_involved": ["N1", "N8", "N12", "N13", "N15"],
      "key_parameters": {
        "registration_type": ["initial", "mobility", "periodic"],
        "access_type": ["3GPP", "non-3GPP"],
        "security_algorithms": ["5G-EA0", "5G-IA0", "5G-EA1", "5G-IA1"]
      }
    },
    "implementation_considerations": [
      "AMF selection based on GUAMI and tracking area",
      "Load balancing across AMF instances in AMF Set",
      "Fallback procedures for registration failures",
      "Integration with network slicing for specific service requirements"
    ]
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 2340.5,
    "tokens_used": 1850,
    "retrieval_score": 0.94,
    "confidence_score": 0.96,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "source_documents": [
    {
      "title": "5G System Architecture - AMF Functions",
      "source": "3GPP TS 23.501",
      "version": "17.9.0",
      "relevance_score": 0.95,
      "content_preview": "The Access and Mobility Management Function (AMF) provides registration management, connection management, reachability management, mobility management..."
    },
    {
      "title": "5G System Procedures - Registration Procedures",
      "source": "3GPP TS 23.502",
      "version": "17.9.0", 
      "relevance_score": 0.93,
      "content_preview": "The registration procedure allows the UE to register with the network for services. The procedure includes authentication..."
    }
  ],
  "timestamp": 1704067200.123
}
```

### Example 2: SMF Session Management

**Query:**
```
"How does SMF manage PDU sessions in 5G networks? Include session establishment and modification procedures."
```

**Expected RAG Response:**
```json
{
  "intent_id": "smf-session-001",
  "original_intent": "How does SMF manage PDU sessions in 5G networks? Include session establishment and modification procedures.",
  "structured_output": {
    "type": "TechnicalExplanation",
    "domain": "5G Core",
    "specification_references": [
      "3GPP TS 23.501 - 5G System Architecture",
      "3GPP TS 23.502 - Procedures for the 5G System",
      "3GPP TS 29.502 - 5G System; Session Management Services"
    ],
    "explanation": {
      "overview": "The SMF (Session Management Function) is responsible for session management, UE IP address allocation and management, DHCP functions, and traffic steering configuration for UPF.",
      "session_establishment": {
        "trigger": "UE-requested PDU Session Establishment via AMF",
        "steps": [
          {
            "step": 1,
            "description": "SMF receives PDU Session Establishment Request from AMF",
            "interface": "N11",
            "message": "Nsmf_PDUSession_CreateSMContext"
          },
          {
            "step": 2,
            "description": "SMF selects appropriate UPF based on location and DNN",
            "component": "UPF Selection Function",
            "criteria": ["UE location", "Data Network Name", "S-NSSAI", "operator policies"]
          },
          {
            "step": 3,
            "description": "SMF establishes N4 session with selected UPF",
            "interface": "N4",
            "message": "PFCP Session Establishment Request"
          },
          {
            "step": 4,
            "description": "IP address allocation for UE",
            "methods": ["Static allocation", "Dynamic allocation via DHCPv4/v6", "IPv6 SLAAC"]
          },
          {
            "step": 5,
            "description": "QoS rules and policies application",
            "components": ["QoS Flow establishment", "Policy enforcement via PCF"]
          }
        ]
      },
      "session_modification": {
        "triggers": [
          "QoS modification request",
          "Traffic routing updates", 
          "UPF reselection",
          "Policy changes from PCF"
        ],
        "procedure": {
          "initiation": "SMF or AMF triggered modification",
          "upf_update": "N4 Session Modification Request to UPF",
          "ue_notification": "PDU Session Modification Command via AMF",
          "completion": "Confirmation and context update"
        }
      },
      "key_functions": [
        "Session context management",
        "UPF selection and reselection",
        "IP address management",
        "QoS control and policy enforcement",
        "Charging and billing coordination",
        "Lawful intercept coordination"
      ]
    },
    "deployment_considerations": [
      "SMF scaling based on session load",
      "Geographic distribution for latency optimization",
      "Integration with network slicing architecture",
      "Support for edge computing deployments"
    ]
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 2680.3,
    "tokens_used": 2140,
    "retrieval_score": 0.92,
    "confidence_score": 0.94,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067260.456
}
```

## O-RAN Architecture and Components

### Example 3: O-RAN Near-RT RIC Deployment

**Query:**
```
"Deploy a Near-RT RIC with 2 replicas for O-RAN network management with A1 policy interface"
```

**Expected RAG Response:**
```json
{
  "intent_id": "near-rt-ric-deploy-001",
  "original_intent": "Deploy a Near-RT RIC with 2 replicas for O-RAN network management with A1 policy interface",
  "structured_output": {
    "type": "NetworkFunctionDeployment",
    "name": "near-rt-ric-deployment",
    "namespace": "oran-platform",
    "spec": {
      "replicas": 2,
      "image": "registry.nephoran.com/oran/near-rt-ric:v1.2.0",
      "resources": {
        "requests": {
          "cpu": "2000m",
          "memory": "4Gi"
        },
        "limits": {
          "cpu": "4000m", 
          "memory": "8Gi"
        }
      },
      "ports": [
        {
          "name": "a1-interface",
          "port": 8080,
          "protocol": "TCP",
          "description": "A1 Policy Management Interface"
        },
        {
          "name": "e2-interface", 
          "port": 8081,
          "protocol": "SCTP",
          "description": "E2 Interface for RAN Functions"
        },
        {
          "name": "metrics",
          "port": 9090,
          "protocol": "TCP", 
          "description": "Prometheus metrics endpoint"
        }
      ],
      "env": [
        {
          "name": "RIC_INSTANCE_ID",
          "value": "near-rt-ric-001"
        },
        {
          "name": "A1_POLICY_ENGINE_ENABLED",
          "value": "true"
        },
        {
          "name": "E2_SUBSCRIPTION_MANAGER_ENABLED", 
          "value": "true"
        },
        {
          "name": "XAPP_MANAGER_ENABLED",
          "value": "true"
        },
        {
          "name": "DATABASE_HOST",
          "value": "ric-database.oran-platform.svc.cluster.local"
        }
      ],
      "volumes": [
        {
          "name": "ric-config",
          "configMap": {
            "name": "near-rt-ric-config"
          }
        },
        {
          "name": "ric-data",
          "persistentVolumeClaim": {
            "claimName": "near-rt-ric-pvc"
          }
        }
      ]
    },
    "o1_config": {
      "management_endpoint": "https://near-rt-ric.oran-platform.svc.cluster.local:8443",
      "fcaps_config": {
        "pm_enabled": true,
        "fm_enabled": true,
        "cm_enabled": true,
        "performance_monitoring": {
          "metrics_collection_interval": "15s",
          "kpi_reporting_interval": "60s"
        },
        "fault_management": {
          "alarm_forwarding": true,
          "alarm_correlation": true
        }
      }
    },
    "a1_policy": {
      "policy_type_definitions": [
        {
          "policy_type_id": "1000",
          "policy_type_name": "QoS Management Policy",
          "description": "Policy for managing QoS parameters in RAN"
        },
        {
          "policy_type_id": "2000", 
          "policy_type_name": "Traffic Steering Policy",
          "description": "Policy for intelligent traffic steering decisions"
        }
      ],
      "default_policies": [
        {
          "policy_id": "default-qos-001",
          "policy_type_id": "1000",
          "policy_data": {
            "priority_level": 5,
            "resource_allocation": "guaranteed",
            "latency_threshold_ms": 20
          }
        }
      ]
    },
    "e2_interface": {
      "ran_function_definitions": [
        {
          "ran_function_id": 1,
          "function_description": "RAN Control",
          "function_revision": 1
        },
        {
          "ran_function_id": 2,
          "function_description": "RAN Indication",
          "function_revision": 1
        }
      ],
      "subscription_management": {
        "max_subscriptions": 1000,
        "subscription_timeout_ms": 30000
      }
    },
    "xapp_platform": {
      "enabled": true,
      "xapp_registry": "registry.nephoran.com/oran/xapps",
      "helm_repository": "https://charts.nephoran.com/oran",
      "supported_xapps": [
        "anomaly-detection",
        "qos-optimization", 
        "load-balancing",
        "interference-mitigation"
      ]
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 3240.7,
    "tokens_used": 2650,
    "retrieval_score": 0.89,
    "confidence_score": 0.91,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067320.789
}
```

### Example 4: O-RAN CU-CP and CU-UP Separation

**Query:**
```
"Configure O-RAN CU-CP and CU-UP separation with F1-C and F1-U interfaces"
```

**Expected RAG Response:**
```json
{
  "intent_id": "cu-separation-001",
  "original_intent": "Configure O-RAN CU-CP and CU-UP separation with F1-C and F1-U interfaces",
  "structured_output": {
    "type": "NetworkFunctionConfiguration",
    "configuration_name": "cu-cp-cu-up-separation",
    "domain": "O-RAN",
    "components": [
      {
        "name": "o-ran-cu-cp",
        "type": "CU-CP",
        "description": "Centralized Unit Control Plane",
        "functions": [
          "RRC functions",
          "F1-AP protocol termination",
          "PDCP-C and SDAP-C processing",
          "Inter-CU mobility coordination"
        ],
        "interfaces": {
          "f1_c": {
            "description": "F1-C interface to DU",
            "protocol": "F1AP over SCTP",
            "endpoint": "f1c.cu-cp.oran-ran.svc.cluster.local:38472",
            "configuration": {
              "sctp_streams": 16,
              "heartbeat_interval": 30,
              "association_timeout": 60
            }
          },
          "e1": {
            "description": "E1 interface to CU-UP",
            "protocol": "E1AP over SCTP", 
            "endpoint": "e1.cu-cp.oran-ran.svc.cluster.local:38462",
            "configuration": {
              "bearer_context_management": true,
              "qos_flow_management": true
            }
          },
          "xn_c": {
            "description": "Xn-C interface to other gNBs",
            "protocol": "XnAP over SCTP",
            "endpoint": "xnc.cu-cp.oran-ran.svc.cluster.local:38422"
          }
        }
      },
      {
        "name": "o-ran-cu-up",
        "type": "CU-UP", 
        "description": "Centralized Unit User Plane",
        "functions": [
          "PDCP-U processing",
          "F1-U tunnel termination",
          "QoS handling and marking",
          "Security functions (encryption/decryption)"
        ],
        "interfaces": {
          "f1_u": {
            "description": "F1-U interface to DU",
            "protocol": "GTP-U over UDP",
            "endpoint": "f1u.cu-up.oran-ran.svc.cluster.local:2152",
            "configuration": {
              "tunnel_management": true,
              "qos_marking": true,
              "echo_procedures": true
            }
          },
          "e1": {
            "description": "E1 interface to CU-CP",
            "protocol": "E1AP over SCTP",
            "endpoint": "e1.cu-up.oran-ran.svc.cluster.local:38463",
            "role": "server"
          },
          "n3": {
            "description": "N3 interface to UPF",
            "protocol": "GTP-U over UDP", 
            "endpoint": "n3.cu-up.oran-ran.svc.cluster.local:2152"
          }
        }
      }
    ],
    "deployment_configuration": {
      "cu_cp_spec": {
        "replicas": 2,
        "image": "registry.nephoran.com/oran/cu-cp:v2.1.0",
        "resources": {
          "requests": {"cpu": "1500m", "memory": "3Gi"},
          "limits": {"cpu": "3000m", "memory": "6Gi"}
        },
        "anti_affinity": true,
        "service_mesh_enabled": true
      },
      "cu_up_spec": {
        "replicas": 3,
        "image": "registry.nephoran.com/oran/cu-up:v2.1.0", 
        "resources": {
          "requests": {"cpu": "2000m", "memory": "4Gi"},
          "limits": {"cpu": "4000m", "memory": "8Gi"}
        },
        "data_plane_acceleration": true,
        "dpdk_enabled": true
      }
    },
    "interface_configuration": {
      "f1_interface": {
        "split_architecture": "7.2x",
        "functional_split": {
          "pdcp": "CU-CP/CU-UP",
          "rlc": "DU", 
          "mac": "DU",
          "phy": "DU"
        },
        "qos_handling": {
          "cu_up_qos_marking": true,
          "du_qos_enforcement": true,
          "end_to_end_qos": true
        }
      },
      "e1_interface": {
        "bearer_context_management": {
          "setup_procedures": true,
          "modification_procedures": true,
          "release_procedures": true
        },
        "qos_flow_management": {
          "establishment": true,
          "modification": true,
          "mapping_to_drb": true
        }
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 4120.2,
    "tokens_used": 3180,
    "retrieval_score": 0.87,
    "confidence_score": 0.89,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067380.234
}
```

## Network Function Deployment

### Example 5: Multi-NF Core Network Deployment

**Query:**
```
"Deploy a complete 5G core network with AMF, SMF, UPF, and NSSF for network slicing support"
```

**Expected RAG Response:**
```json
{
  "intent_id": "5g-core-deploy-001", 
  "original_intent": "Deploy a complete 5G core network with AMF, SMF, UPF, and NSSF for network slicing support",
  "structured_output": {
    "type": "NetworkFunctionDeployment",
    "name": "5g-core-network-deployment",
    "namespace": "core-network",
    "deployment_strategy": "rolling_update",
    "network_functions": [
      {
        "name": "amf-deployment",
        "type": "AMF",
        "spec": {
          "replicas": 3,
          "image": "registry.nephoran.com/5g-core/amf:v3.1.0",
          "resources": {
            "requests": {"cpu": "1000m", "memory": "2Gi"},
            "limits": {"cpu": "2000m", "memory": "4Gi"}
          },
          "ports": [
            {"name": "sbi", "port": 8080, "protocol": "TCP"},
            {"name": "n1-n2", "port": 38412, "protocol": "SCTP"}
          ],
          "env": [
            {"name": "AMF_REGION_ID", "value": "128"}, 
            {"name": "AMF_SET_ID", "value": "1"},
            {"name": "AMF_POINTER", "value": "1"},
            {"name": "NETWORK_SLICING_ENABLED", "value": "true"}
          ]
        },
        "sbi_interfaces": [
          {"interface": "Namf_Communication", "port": 8080},
          {"interface": "Namf_EventExposure", "port": 8080},
          {"interface": "Namf_Location", "port": 8080}
        ]
      },
      {
        "name": "smf-deployment",
        "type": "SMF", 
        "spec": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/smf:v3.1.0",
          "resources": {
            "requests": {"cpu": "800m", "memory": "1.5Gi"},
            "limits": {"cpu": "1600m", "memory": "3Gi"}
          },
          "ports": [
            {"name": "sbi", "port": 8080, "protocol": "TCP"},
            {"name": "n4", "port": 8805, "protocol": "UDP"}
          ],
          "env": [
            {"name": "SMF_INSTANCE_ID", "value": "1"},
            {"name": "DNN_SUPPORT", "value": "internet,ims,mec"},
            {"name": "SLICE_SUPPORT", "value": "true"}
          ]
        },
        "sbi_interfaces": [
          {"interface": "Nsmf_PDUSession", "port": 8080},
          {"interface": "Nsmf_EventExposure", "port": 8080}
        ]
      },
      {
        "name": "upf-deployment", 
        "type": "UPF",
        "spec": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/upf:v3.1.0",
          "resources": {
            "requests": {"cpu": "2000m", "memory": "4Gi"},
            "limits": {"cpu": "4000m", "memory": "8Gi"}
          },
          "ports": [
            {"name": "n4", "port": 8805, "protocol": "UDP"},
            {"name": "n3", "port": 2152, "protocol": "UDP"},
            {"name": "n6", "port": 2153, "protocol": "UDP"}
          ],
          "env": [
            {"name": "UPF_INSTANCE_ID", "value": "1"},
            {"name": "DATA_NETWORK_ACCESS_ENABLED", "value": "true"},
            {"name": "TRAFFIC_STEERING_ENABLED", "value": "true"}
          ]
        },
        "data_plane_optimization": {
          "dpdk_enabled": true,
          "sr_iov_enabled": true,
          "huge_pages": "2Gi"
        }
      },
      {
        "name": "nssf-deployment",
        "type": "NSSF",
        "spec": {
          "replicas": 2, 
          "image": "registry.nephoran.com/5g-core/nssf:v3.1.0",
          "resources": {
            "requests": {"cpu": "500m", "memory": "1Gi"},
            "limits": {"cpu": "1000m", "memory": "2Gi"}
          },
          "ports": [
            {"name": "sbi", "port": 8080, "protocol": "TCP"}
          ],
          "env": [
            {"name": "NSSF_INSTANCE_ID", "value": "1"},
            {"name": "SLICE_SELECTION_ENABLED", "value": "true"}
          ]
        },
        "sbi_interfaces": [
          {"interface": "Nnssf_NSSelection", "port": 8080},
          {"interface": "Nnssf_NSSAIAvailability", "port": 8080}
        ]
      }
    ],
    "network_slicing": {
      "slice_configurations": [
        {
          "slice_id": "embb-001",
          "s_nssai": {"sst": 1, "sd": "000001"},
          "slice_type": "eMBB",
          "description": "Enhanced Mobile Broadband",
          "associated_nfs": ["AMF", "SMF", "UPF"],
          "sla_parameters": {
            "latency_ms": 20,
            "throughput_mbps": 1000,
            "reliability": 0.999
          }
        },
        {
          "slice_id": "urllc-001", 
          "s_nssai": {"sst": 2, "sd": "000001"},
          "slice_type": "URLLC",
          "description": "Ultra-Reliable Low Latency Communications",
          "associated_nfs": ["AMF", "SMF", "UPF"],
          "sla_parameters": {
            "latency_ms": 1,
            "throughput_mbps": 100,
            "reliability": 0.99999
          }
        }
      ]
    },
    "service_mesh": {
      "enabled": true,
      "mutual_tls": true,
      "service_discovery": "consul",
      "load_balancing": "round_robin"
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 5240.8,
    "tokens_used": 4150,
    "retrieval_score": 0.91,
    "confidence_score": 0.93,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067440.567
}
```

## Interface Configuration

### Example 6: A1 Policy Interface Configuration

**Query:**
```
"Configure A1 policy interface for RIC with QoS and traffic steering policies"
```

**Expected RAG Response:**
```json
{
  "intent_id": "a1-policy-config-001",
  "original_intent": "Configure A1 policy interface for RIC with QoS and traffic steering policies",
  "structured_output": {
    "type": "PolicyConfiguration",
    "policy_name": "a1-policy-management-config",
    "interface": "A1",
    "policy_framework": {
      "policy_types": [
        {
          "policy_type_id": "20000",
          "policy_type_name": "QoS Policy Type",
          "description": "Quality of Service management policies for RAN optimization",
          "policy_type_version": "1.0.0",
          "create_schema": {
            "type": "object",
            "properties": {
              "scope": {
                "type": "object",
                "properties": {
                  "ue_id": {"type": "string"},
                  "cell_id": {"type": "array", "items": {"type": "string"}},
                  "qci": {"type": "integer", "minimum": 1, "maximum": 9}
                }
              },
              "statement": {
                "type": "object",
                "properties": {
                  "priority_level": {"type": "integer", "minimum": 1, "maximum": 15},
                  "resource_allocation": {"type": "string", "enum": ["guaranteed", "non-guaranteed"]},
                  "max_bitrate_dl": {"type": "integer"},
                  "max_bitrate_ul": {"type": "integer"},
                  "latency_threshold": {"type": "integer"}
                }
              }
            }
          }
        },
        {
          "policy_type_id": "20001",
          "policy_type_name": "Traffic Steering Policy Type", 
          "description": "Intelligent traffic steering and load balancing policies",
          "policy_type_version": "1.0.0",
          "create_schema": {
            "type": "object",
            "properties": {
              "scope": {
                "type": "object",
                "properties": {
                  "coverage_area": {"type": "array", "items": {"type": "string"}},
                  "service_type": {"type": "string", "enum": ["eMBB", "URLLC", "mMTC"]},
                  "time_window": {"type": "string"}
                }
              },
              "statement": {
                "type": "object", 
                "properties": {
                  "steering_algorithm": {"type": "string", "enum": ["load_based", "latency_based", "distance_based"]},
                  "target_cells": {"type": "array", "items": {"type": "string"}},
                  "load_threshold": {"type": "number", "minimum": 0, "maximum": 1},
                  "handover_parameters": {
                    "type": "object",
                    "properties": {
                      "hysteresis": {"type": "number"},
                      "time_to_trigger": {"type": "integer"}
                    }
                  }
                }
              }
            }
          }
        }
      ]
    },
    "policy_instances": [
      {
        "policy_id": "qos-policy-001",
        "policy_type_id": "20000",
        "ric_id": "near-rt-ric-001",
        "policy_data": {
          "scope": {
            "ue_id": "*",
            "cell_id": ["cell-001", "cell-002", "cell-003"],
            "qci": 1
          },
          "statement": {
            "priority_level": 2,
            "resource_allocation": "guaranteed", 
            "max_bitrate_dl": 100000000,
            "max_bitrate_ul": 50000000,
            "latency_threshold": 10
          }
        },
        "enforcement_scope": "per_ue",
        "enforcement_method": "immediate"
      },
      {
        "policy_id": "traffic-steering-001",
        "policy_type_id": "20001", 
        "ric_id": "near-rt-ric-001",
        "policy_data": {
          "scope": {
            "coverage_area": ["area-1", "area-2"],
            "service_type": "eMBB",
            "time_window": "continuous"
          },
          "statement": {
            "steering_algorithm": "load_based",
            "target_cells": ["cell-001", "cell-002", "cell-003", "cell-004"],
            "load_threshold": 0.8,
            "handover_parameters": {
              "hysteresis": 2.0,
              "time_to_trigger": 160
            }
          }
        },
        "enforcement_scope": "per_cell",
        "enforcement_method": "progressive"
      }
    ],
    "a1_interface_configuration": {
      "endpoint": "https://a1-interface.near-rt-ric.oran-platform.svc.cluster.local:8080",
      "api_version": "v2.1.0",
      "authentication": {
        "method": "oauth2",
        "token_endpoint": "https://auth.oran-platform.svc.cluster.local/oauth/token",
        "client_credentials": true
      },
      "communication_settings": {
        "timeout_seconds": 30,
        "retry_count": 3,
        "retry_delay_seconds": 5,
        "keep_alive": true
      },
      "policy_lifecycle": {
        "create_endpoint": "/a1-p/v2/policytypes/{policy_type_id}/policies/{policy_instance_id}",
        "status_endpoint": "/a1-p/v2/policytypes/{policy_type_id}/policies/{policy_instance_id}/status",
        "delete_endpoint": "/a1-p/v2/policytypes/{policy_type_id}/policies/{policy_instance_id}",
        "notification_callback": "https://nonrt-ric.oran-platform.svc.cluster.local:8080/a1-policy-status"
      }
    },
    "monitoring_and_reporting": {
      "policy_compliance_monitoring": true,
      "kpi_reporting": {
        "enabled": true,
        "reporting_interval": "60s",
        "metrics": ["throughput", "latency", "success_rate", "resource_utilization"]
      },
      "alarm_generation": {
        "policy_violation_alarms": true,
        "performance_degradation_alarms": true,  
        "connectivity_loss_alarms": true
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 3890.4,
    "tokens_used": 3240,
    "retrieval_score": 0.88,
    "confidence_score": 0.90,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067500.891
}
```

## Performance Optimization

### Example 7: Network Function Scaling

**Query:**
```
"Scale AMF to 5 replicas with CPU optimization for high load scenarios"
```

**Expected RAG Response:**
```json
{
  "intent_id": "amf-scale-001",
  "original_intent": "Scale AMF to 5 replicas with CPU optimization for high load scenarios",
  "structured_output": {
    "type": "NetworkFunctionScale",
    "name": "amf-deployment",
    "namespace": "core-network",
    "current_replicas": 3,
    "target_replicas": 5,
    "scaling_strategy": "horizontal",
    "resource_adjustments": {
      "cpu_scale_factor": 1.5,
      "memory_scale_factor": 1.2,
      "optimized_resources": {
        "requests": {
          "cpu": "1500m",
          "memory": "2.4Gi"
        },
        "limits": {
          "cpu": "3000m", 
          "memory": "4.8Gi"
        }
      }
    },
    "high_load_optimizations": {
      "jvm_tuning": {
        "heap_size": "3Gi",
        "gc_algorithm": "G1GC",
        "gc_tuning_flags": [
          "-XX:+UseG1GC",
          "-XX:MaxGCPauseMillis=200",
          "-XX:+UnlockExperimentalVMOptions",
          "-XX:+UseCGroupMemoryLimitForHeap"
        ]
      },
      "application_tuning": {
        "connection_pool_size": 200,
        "thread_pool_size": 100,
        "queue_size": 1000,
        "keepalive_timeout": 300
      },
      "kubernetes_optimizations": {
        "pod_disruption_budget": {
          "min_available": 3,
          "max_unavailable": 1
        },
        "horizontal_pod_autoscaler": {
          "enabled": true,
          "min_replicas": 3,
          "max_replicas": 10,
          "target_cpu_utilization": 70,
          "target_memory_utilization": 80,
          "scale_up_stabilization": "60s",
          "scale_down_stabilization": "300s"
        },
        "topology_spread_constraints": [
          {
            "max_skew": 1,
            "topology_key": "kubernetes.io/hostname",
            "when_unsatisfiable": "DoNotSchedule"
          },
          {
            "max_skew": 1,
            "topology_key": "topology.kubernetes.io/zone", 
            "when_unsatisfiable": "ScheduleAnyway"
          }
        ]
      }
    },
    "load_balancing_configuration": {
      "service_type": "LoadBalancer",
      "session_affinity": "ClientIP",
      "load_balancer_class": "metallb",
      "health_check_settings": {
        "health_check_grace_period": "30s",
        "health_check_timeout": "5s",
        "healthy_threshold": 2,
        "unhealthy_threshold": 3
      }
    },
    "monitoring_configuration": {
      "prometheus_metrics": {
        "enabled": true,
        "scrape_interval": "15s",
        "metrics_path": "/metrics",
        "additional_labels": {
          "scaling_event": "high_load_optimization",
          "scaling_timestamp": "2024-07-28T10:30:00Z"
        }
      },
      "custom_metrics": [
        {
          "name": "amf_registration_rate",
          "description": "Registration requests per second",
          "type": "gauge"
        },
        {
          "name": "amf_context_count",
          "description": "Active UE contexts", 
          "type": "gauge"
        },
        {
          "name": "amf_sbi_response_time",
          "description": "SBI interface response time",
          "type": "histogram"
        }
      ]
    },
    "rollout_strategy": {
      "type": "RollingUpdate",
      "max_surge": "25%",
      "max_unavailable": "25%",
      "revision_history_limit": 10,
      "progress_deadline_seconds": 600,
      "pre_stop_hook": {
        "command": ["/bin/sh", "-c", "sleep 30"]
      }
    }
  },
  "status": "completed", 
  "metrics": {
    "processing_time_ms": 2140.6,
    "tokens_used": 1890,
    "retrieval_score": 0.85,
    "confidence_score": 0.88,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067560.123
}
```

## Troubleshooting and Monitoring

### Example 8: Network Function Health Monitoring

**Query:**
```
"Set up comprehensive monitoring for 5G core network functions with alerting on performance degradation"
```

**Expected RAG Response:**
```json
{
  "intent_id": "monitoring-setup-001",
  "original_intent": "Set up comprehensive monitoring for 5G core network functions with alerting on performance degradation",
  "structured_output": {
    "type": "MonitoringConfiguration",
    "name": "5g-core-monitoring-stack", 
    "namespace": "monitoring",
    "monitoring_components": [
      {
        "name": "prometheus-server",
        "type": "MetricsCollection",
        "configuration": {
          "retention_period": "30d",
          "scrape_interval": "15s",
          "evaluation_interval": "15s",
          "storage_size": "100Gi",
          "external_labels": {
            "cluster": "5g-core-production",
            "region": "us-central1"
          }
        },
        "scrape_configs": [
          {
            "job_name": "amf-metrics",
            "kubernetes_sd_configs": [{
              "role": "pod",
              "namespaces": {"names": ["core-network"]}
            }],
            "relabel_configs": [
              {
                "source_labels": ["__meta_kubernetes_pod_label_app"],
                "action": "keep",
                "regex": "amf"
              }
            ]
          },
          {
            "job_name": "smf-metrics",
            "static_configs": [{
              "targets": ["smf-deployment.core-network.svc.cluster.local:9090"]
            }],
            "metrics_path": "/metrics",
            "scrape_interval": "10s"
          },
          {
            "job_name": "upf-metrics",
            "static_configs": [{
              "targets": ["upf-deployment.core-network.svc.cluster.local:9090"]
            }],
            "metrics_path": "/metrics",
            "scrape_interval": "10s"
          }
        ]
      },
      {
        "name": "grafana-dashboard",
        "type": "Visualization",
        "configuration": {
          "admin_password_secret": "grafana-admin-credentials",
          "data_sources": [
            {
              "name": "Prometheus",
              "type": "prometheus",
              "url": "http://prometheus-server.monitoring.svc.cluster.local:9090",
              "is_default": true
            }
          ],
          "dashboards": [
            {
              "name": "5G Core Network Overview",
              "panels": [
                {
                  "title": "AMF Registration Rate",
                  "type": "graph",
                  "targets": ["rate(amf_registration_requests_total[5m])"],
                  "alert_threshold": 1000
                },
                {
                  "title": "SMF Session Establishment Success Rate",
                  "type": "singlestat",
                  "targets": ["(rate(smf_session_establishment_success_total[5m]) / rate(smf_session_establishment_total[5m])) * 100"],
                  "alert_threshold": 95
                },
                {
                  "title": "UPF Throughput",
                  "type": "graph", 
                  "targets": ["rate(upf_bytes_transmitted_total[5m])", "rate(upf_bytes_received_total[5m])"],
                  "unit": "bytes/sec"
                }
              ]
            },
            {
              "name": "Network Function Health",
              "panels": [
                {
                  "title": "Pod Status",
                  "type": "table",
                  "targets": ["up{job=~'amf-metrics|smf-metrics|upf-metrics'}"]
                },
                {
                  "title": "Memory Usage",
                  "type": "graph",
                  "targets": ["container_memory_usage_bytes{pod=~'amf-.*|smf-.*|upf-.*'}"]
                },
                {
                  "title": "CPU Usage",
                  "type": "graph",
                  "targets": ["rate(container_cpu_usage_seconds_total{pod=~'amf-.*|smf-.*|upf-.*'}[5m])"]
                }
              ]
            }
          ]
        }
      },
      {
        "name": "alertmanager",
        "type": "AlertingAndNotification",
        "configuration": {
          "global": {
            "smtp_smarthost": "smtp.company.com:587",
            "smtp_from": "alerts@nephoran.com"
          },
          "routes": [
            {
              "match": {"severity": "critical"},
              "receiver": "critical-alerts",
              "group_wait": "10s",
              "group_interval": "5m",
              "repeat_interval": "12h"
            },
            {
              "match": {"severity": "warning"}, 
              "receiver": "warning-alerts",
              "group_wait": "30s",
              "group_interval": "15m",
              "repeat_interval": "24h"
            }
          ],
          "receivers": [
            {
              "name": "critical-alerts",
              "email_configs": [
                {
                  "to": "oncall@nephoran.com",
                  "subject": "CRITICAL: 5G Core Network Alert - {{ .GroupLabels.alertname }}",
                  "body": "{{ range .Alerts }}{{ .Annotations.description }}{{ end }}"
                }
              ],
              "slack_configs": [
                {
                  "api_url": "https://hooks.slack.com/services/.../",
                  "channel": "#critical-alerts",
                  "title": "5G Core Critical Alert",
                  "text": "{{ .CommonAnnotations.summary }}"
                }
              ]
            },
            {
              "name": "warning-alerts",
              "email_configs": [
                {
                  "to": "operations@nephoran.com",
                  "subject": "WARNING: 5G Core Network Alert - {{ .GroupLabels.alertname }}"
                }
              ]
            }
          ]
        }
      }
    ],
    "alerting_rules": [
      {
        "name": "5g_core_network_alerts",
        "rules": [
          {
            "alert": "AMFHighRegistrationFailureRate",
            "expr": "(rate(amf_registration_failures_total[5m]) / rate(amf_registration_requests_total[5m])) > 0.05",
            "for": "2m",
            "labels": {"severity": "critical", "component": "amf"},
            "annotations": {
              "summary": "AMF registration failure rate is above 5%",
              "description": "AMF {{ $labels.instance }} has registration failure rate of {{ $value | humanizePercentage }}"
            }
          },
          {
            "alert": "SMFSessionEstablishmentFailure", 
            "expr": "(rate(smf_session_establishment_failures_total[5m]) / rate(smf_session_establishment_requests_total[5m])) > 0.1",
            "for": "3m",
            "labels": {"severity": "critical", "component": "smf"},
            "annotations": {
              "summary": "SMF session establishment failure rate is above 10%",
              "description": "SMF {{ $labels.instance }} has session establishment failure rate of {{ $value | humanizePercentage }}"
            }
          },
          {
            "alert": "UPFHighPacketLoss",
            "expr": "(rate(upf_packets_dropped_total[5m]) / rate(upf_packets_received_total[5m])) > 0.01",
            "for": "1m",
            "labels": {"severity": "warning", "component": "upf"},
            "annotations": {
              "summary": "UPF packet loss rate is above 1%",
              "description": "UPF {{ $labels.instance }} has packet loss rate of {{ $value | humanizePercentage }}"
            }
          },
          {
            "alert": "NetworkFunctionDown",
            "expr": "up{job=~'amf-metrics|smf-metrics|upf-metrics|nssf-metrics'} == 0",
            "for": "1m",
            "labels": {"severity": "critical", "component": "{{ $labels.job }}"},
            "annotations": {
              "summary": "Network Function is down",
              "description": "Network function {{ $labels.job }} on {{ $labels.instance }} is down"
            }
          },
          {
            "alert": "HighMemoryUsage",
            "expr": "(container_memory_usage_bytes{pod=~'amf-.*|smf-.*|upf-.*'} / container_spec_memory_limit_bytes) > 0.9",
            "for": "5m",
            "labels": {"severity": "warning", "component": "{{ $labels.pod }}"},
            "annotations": {
              "summary": "High memory usage detected",
              "description": "Pod {{ $labels.pod }} is using {{ $value | humanizePercentage }} of its memory limit"
            }
          }
        ]
      }
    ],
    "log_aggregation": {
      "fluentd_configuration": {
        "input_sources": [
          {
            "type": "kubernetes_logs",
            "namespaces": ["core-network", "oran-platform"],
            "log_level_filter": ["ERROR", "WARN", "INFO"]
          }
        ],
        "output_destinations": [
          {
            "type": "elasticsearch",
            "endpoint": "https://elasticsearch.logging.svc.cluster.local:9200",
            "index_pattern": "5g-core-logs-%Y.%m.%d"
          }
        ],
        "parsing_rules": [
          {
            "source": "amf",
            "format": "json",
            "time_key": "timestamp",
            "keep_time_key": false
          }
        ]
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 4670.2,
    "tokens_used": 3890,
    "retrieval_score": 0.90,
    "confidence_score": 0.92,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067620.456
}
```

## Network Slicing Configuration

### Example 9: Network Slice Template Configuration

**Query:**
```
"Create network slice template for URLLC with 1ms latency requirement and dedicated AMF instance"
```

**Expected RAG Response:**
```json
{
  "intent_id": "urllc-slice-001",
  "original_intent": "Create network slice template for URLLC with 1ms latency requirement and dedicated AMF instance",
  "structured_output": {
    "type": "NetworkSliceConfiguration",
    "slice_id": "urllc-ultra-low-latency-001",
    "name": "URLLC Ultra Low Latency Slice",
    "slice_type": "URLLC",
    "s_nssai": {
      "sst": 2,
      "sd": "000002"
    },
    "sla_parameters": {
      "latency_ms": 1,
      "latency_jitter_ms": 0.1,
      "throughput_mbps": 100,
      "reliability": 0.99999,
      "availability": 0.9999,
      "packet_error_rate": 0.00001,
      "service_continuity": "service_continuity_level_1"
    },
    "network_functions": [
      {
        "name": "urllc-dedicated-amf",
        "type": "AMF",
        "deployment_spec": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/amf-urllc:v3.1.0",
          "resources": {
            "requests": {"cpu": "2000m", "memory": "4Gi"},
            "limits": {"cpu": "4000m", "memory": "8Gi"}
          },
          "specialized_configuration": {
            "urllc_optimizations": true,
            "fast_path_processing": true,
            "priority_queuing": true,
            "preemption_capability": true
          },
          "node_affinity": {
            "required": {
              "node_type": "high_performance",
              "cpu_architecture": "x86_64",
              "network_acceleration": "dpdk"
            }
          }
        },
        "sla_enforcement": {
          "latency_monitoring": true,
          "priority_handling": "highest",
          "resource_reservation": "guaranteed"
        }
      },
      {
        "name": "urllc-optimized-smf",
        "type": "SMF", 
        "deployment_spec": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/smf-urllc:v3.1.0",
          "resources": {
            "requests": {"cpu": "1500m", "memory": "3Gi"},
            "limits": {"cpu": "3000m", "memory": "6Gi"}
          },
          "specialized_configuration": {
            "session_establishment_optimization": true,
            "fast_forwarding": true,
            "qos_enforcement": "strict"
          }
        },
        "qos_configuration": {
          "default_qos_flow": {
            "qci": 1,
            "arp": {"priority_level": 1, "pre_emption_capability": "may_preempt", "pre_emption_vulnerability": "not_preemptable"},
            "gbr_qos_info": {"max_flow_bit_rate_dl": 100000000, "max_flow_bit_rate_ul": 100000000, "guaranteed_flow_bit_rate_dl": 50000000, "guaranteed_flow_bit_rate_ul": 50000000}
          }
        }
      },
      {
        "name": "urllc-edge-upf",
        "type": "UPF",
        "deployment_spec": {
          "replicas": 3,
          "image": "registry.nephoran.com/5g-core/upf-edge:v3.1.0",
          "resources": {
            "requests": {"cpu": "3000m", "memory": "6Gi"},
            "limits": {"cpu": "6000m", "memory": "12Gi"}
          },
          "edge_deployment": true,
          "proximity_requirements": {
            "max_distance_km": 10,
            "edge_zone_preference": ["industrial", "manufacturing", "autonomous_vehicle"]
          }
        },
        "data_plane_optimizations": {
          "dpdk_enabled": true,
          "sr_iov_enabled": true,
          "cpu_pinning": true,
          "huge_pages": "4Gi",
          "numa_alignment": true,
          "kernel_bypass": true
        }
      }
    ],
    "ran_requirements": {
      "functional_split": "7.2x",
      "fronthaul_latency_budget_us": 200,
      "scheduling_optimization": {
        "urllc_priority_scheduling": true,
        "preemptive_scheduling": true,
        "mini_slot_scheduling": true
      },
      "radio_resource_management": {
        "dedicated_pucch_resources": true,
        "shortened_tti": true,
        "repetition_coding": true
      }
    },
    "transport_requirements": {
      "network_topology": "mesh",
      "redundancy": "dual_path",
      "transport_latency_budget_us": 500,
      "jitter_control": true,
      "traffic_engineering": {
        "mpls_te_enabled": true,
        "segment_routing_enabled": true,
        "path_protection": "1_plus_1"
      }
    },
    "policies": [
      {
        "policy_id": "urllc-qos-enforcement-001",
        "policy_type": "QoS",
        "enforcement_points": ["AMF", "SMF", "UPF", "gNB"],
        "policy_data": {
          "latency_enforcement": {
            "max_acceptable_latency_ms": 1,
            "latency_violation_action": "traffic_rerouting",
            "latency_monitoring_interval_ms": 10
          },
          "reliability_enforcement": {
            "packet_duplication": true,
            "path_diversity": true,
            "fast_failure_detection": true
          }
        }
      },
      {
        "policy_id": "urllc-resource-reservation-001", 
        "policy_type": "ResourceReservation",
        "enforcement_points": ["Scheduler", "ResourceManager"],
        "policy_data": {
          "cpu_reservation": "guaranteed",
          "memory_reservation": "guaranteed", 
          "bandwidth_reservation": "guaranteed",
          "preemption_priority": "highest"
        }
      }
    ],
    "monitoring_configuration": {
      "kpi_monitoring": [
        {
          "kpi_name": "end_to_end_latency",
          "target_value": 1.0,
          "threshold_warning": 0.8,
          "threshold_critical": 0.9,
          "measurement_interval_ms": 100
        },
        {
          "kpi_name": "packet_loss_rate",
          "target_value": 0.00001,
          "threshold_warning": 0.00005,
          "threshold_critical": 0.0001,
          "measurement_interval_ms": 1000
        },
        {
          "kpi_name": "service_availability",
          "target_value": 0.9999,
          "threshold_warning": 0.999,
          "threshold_critical": 0.995,
          "measurement_interval_s": 60
        }
      ],
      "sla_violation_handling": {
        "automatic_remediation": true,
        "escalation_procedures": true,
        "root_cause_analysis": true
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 5120.7,
    "tokens_used": 4290,
    "retrieval_score": 0.93,
    "confidence_score": 0.95,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067680.789
}
```

## Security and Authentication

### Example 10: OAuth2 Authentication Configuration

**Query:**
```
"Configure OAuth2 authentication for 5G core network functions with mutual TLS"
```

**Expected RAG Response:**
```json
{
  "intent_id": "oauth2-auth-001",
  "original_intent": "Configure OAuth2 authentication for 5G core network functions with mutual TLS",
  "structured_output": {
    "type": "SecurityConfiguration",
    "name": "5g-core-oauth2-mtls-config",
    "security_domain": "core-network",
    "authentication_framework": {
      "primary_method": "oauth2",
      "secondary_method": "mutual_tls",
      "fallback_method": "api_key"
    },
    "oauth2_configuration": {
      "authorization_server": {
        "endpoint": "https://auth.nephoran.com/oauth2",
        "issuer": "https://auth.nephoran.com",
        "discovery_endpoint": "https://auth.nephoran.com/.well-known/oauth_authorization_server",
        "token_endpoint": "https://auth.nephoran.com/oauth2/token",
        "revocation_endpoint": "https://auth.nephoran.com/oauth2/revoke",
        "introspection_endpoint": "https://auth.nephoran.com/oauth2/introspect"
      },
      "client_credentials_flow": {
        "enabled": true,
        "scope_requirements": {
          "amf": ["amf:read", "amf:write", "amf:admin"],
          "smf": ["smf:read", "smf:write", "smf:session"],
          "upf": ["upf:read", "upf:write", "upf:dataplane"],
          "nssf": ["nssf:read", "nssf:write", "nssf:slice"]
        },
        "client_authentication": "client_secret_jwt",
        "token_lifetime_seconds": 3600
      },
      "jwt_configuration": {
        "algorithm": "RS256",
        "public_key_source": "jwks_uri",
        "jwks_uri": "https://auth.nephoran.com/.well-known/jwks.json",
        "audience_validation": true,
        "issuer_validation": true,
        "expiration_validation": true,
        "not_before_validation": true
      }
    },
    "mutual_tls_configuration": {
      "ca_certificate": {
        "secret_name": "5g-core-ca-cert",
        "namespace": "core-network",
        "certificate_authority": "Nephoran Internal CA",
        "validity_period_days": 365
      },
      "client_certificates": [
        {
          "service_name": "amf",
          "certificate_secret": "amf-client-cert",
          "common_name": "amf.core-network.svc.cluster.local",
          "san_dns_names": [
            "amf.core-network.svc.cluster.local",
            "amf-deployment.core-network.svc.cluster.local"
          ],
          "key_usage": ["digital_signature", "key_encipherment", "client_auth"]
        },
        {
          "service_name": "smf",
          "certificate_secret": "smf-client-cert", 
          "common_name": "smf.core-network.svc.cluster.local",
          "san_dns_names": [
            "smf.core-network.svc.cluster.local",
            "smf-deployment.core-network.svc.cluster.local"
          ],
          "key_usage": ["digital_signature", "key_encipherment", "client_auth"]
        },
        {
          "service_name": "upf",
          "certificate_secret": "upf-client-cert",
          "common_name": "upf.core-network.svc.cluster.local", 
          "san_dns_names": [
            "upf.core-network.svc.cluster.local",
            "upf-deployment.core-network.svc.cluster.local"
          ],
          "key_usage": ["digital_signature", "key_encipherment", "client_auth"]
        }
      ],
      "server_certificates": [
        {
          "service_name": "amf-sbi",
          "certificate_secret": "amf-server-cert",
          "common_name": "amf-sbi.core-network.svc.cluster.local",
          "san_dns_names": [
            "amf-sbi.core-network.svc.cluster.local",
            "*.amf.core-network.svc.cluster.local"
          ],
          "key_usage": ["digital_signature", "key_encipherment", "server_auth"]
        }
      ],
      "tls_settings": {
        "min_tls_version": "1.2",
        "preferred_tls_version": "1.3",
        "cipher_suites": [
          "TLS_AES_256_GCM_SHA384",
          "TLS_CHACHA20_POLY1305_SHA256",
          "TLS_AES_128_GCM_SHA256"
        ],
        "verify_client_cert": true,
        "verify_server_cert": true
      }
    },
    "service_mesh_integration": {
      "istio_configuration": {
        "mtls_mode": "STRICT",
        "authentication_policy": {
          "principals": [
            "cluster.local/ns/core-network/sa/amf-service-account",
            "cluster.local/ns/core-network/sa/smf-service-account",
            "cluster.local/ns/core-network/sa/upf-service-account"
          ]
        },
        "authorization_policy": {
          "rules": [
            {
              "from": [{"source": {"principals": ["cluster.local/ns/core-network/sa/amf-service-account"]}}],
              "to": [{"operation": {"methods": ["GET", "POST"], "paths": ["/namf-comm/*"]}}]
            },
            {
              "from": [{"source": {"principals": ["cluster.local/ns/core-network/sa/smf-service-account"]}}],
              "to": [{"operation": {"methods": ["GET", "POST"], "paths": ["/nsmf-pdusession/*"]}}]
            }
          ]
        }
      }
    },
    "network_function_configurations": [
      {
        "function_name": "AMF",
        "oauth2_client": {
          "client_id": "amf-client-001",
          "client_secret_ref": {
            "secret_name": "amf-oauth2-credentials",
            "secret_key": "client_secret"
          },
          "scopes": ["amf:read", "amf:write", "amf:admin"],
          "token_cache_ttl_seconds": 3300
        },
        "mtls_client": {
          "cert_file": "/etc/certs/tls.crt",
          "key_file": "/etc/certs/tls.key",
          "ca_file": "/etc/certs/ca.crt",
          "verify_server_cert": true
        },
        "sbi_security": {
          "require_authentication": true,
          "authentication_methods": ["oauth2", "mtls"],
          "rate_limiting": {
            "requests_per_minute": 1000,
            "burst_size": 100
          }
        }
      },
      {
        "function_name": "SMF",
        "oauth2_client": {
          "client_id": "smf-client-001",
          "client_secret_ref": {
            "secret_name": "smf-oauth2-credentials", 
            "secret_key": "client_secret"
          },
          "scopes": ["smf:read", "smf:write", "smf:session"],
          "token_cache_ttl_seconds": 3300
        },
        "mtls_client": {
          "cert_file": "/etc/certs/tls.crt",
          "key_file": "/etc/certs/tls.key",
          "ca_file": "/etc/certs/ca.crt",
          "verify_server_cert": true
        }
      }
    ],
    "security_monitoring": {
      "authentication_logging": {
        "log_successful_authentications": true,
        "log_failed_authentications": true,
        "log_token_usage": true,
        "log_certificate_validation": true
      },
      "security_metrics": [
        {
          "metric_name": "authentication_success_rate",
          "description": "Percentage of successful authentications",
          "type": "gauge"
        },
        {
          "metric_name": "authentication_failure_count",
          "description": "Number of authentication failures",
          "type": "counter"
        },
        {
          "metric_name": "certificate_expiry_days",
          "description": "Days until certificate expiration",
          "type": "gauge"
        }
      ],
      "security_alerts": [
        {
          "alert_name": "HighAuthenticationFailureRate",
          "condition": "authentication_failure_rate > 0.05",
          "severity": "warning",
          "notification_channels": ["security-team"]
        },
        {
          "alert_name": "CertificateExpiringSoon",
          "condition": "certificate_expiry_days < 30",
          "severity": "critical",
          "notification_channels": ["infrastructure-team", "security-team"]
        }
      ]
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 4560.3,
    "tokens_used": 3790,
    "retrieval_score": 0.86,
    "confidence_score": 0.89,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067740.012
}
```

## Complex Multi-Domain Queries

### Example 11: End-to-End Service Deployment

**Query:**
```
"Deploy a complete end-to-end 5G service including RAN, Core, and Edge components with automated scaling and monitoring"
```

**Expected RAG Response:**
```json
{
  "intent_id": "e2e-service-001",
  "original_intent": "Deploy a complete end-to-end 5G service including RAN, Core, and Edge components with automated scaling and monitoring",
  "structured_output": {
    "type": "ComplexServiceDeployment",
    "name": "end-to-end-5g-service",
    "service_domains": ["RAN", "Core", "Edge", "Management"],
    "deployment_architecture": {
      "topology": "distributed",
      "high_availability": true,
      "disaster_recovery": true,
      "geographic_distribution": ["us-central1", "us-east1", "us-west1"]
    },
    "ran_components": {
      "o_ran_deployment": {
        "near_rt_ric": {
          "replicas": 2,
          "image": "registry.nephoran.com/oran/near-rt-ric:v1.2.0",
          "resources": {"cpu": "2000m", "memory": "4Gi"},
          "xapps": ["qos-optimization", "load-balancing", "anomaly-detection"]
        },
        "cu_cp": {
          "replicas": 2, 
          "image": "registry.nephoran.com/oran/cu-cp:v2.1.0",
          "resources": {"cpu": "1500m", "memory": "3Gi"},
          "interfaces": ["f1_c", "e1", "xn_c"]
        },
        "cu_up": {
          "replicas": 3,
          "image": "registry.nephoran.com/oran/cu-up:v2.1.0", 
          "resources": {"cpu": "2000m", "memory": "4Gi"},
          "interfaces": ["f1_u", "e1", "n3"],
          "data_plane_acceleration": true
        },
        "du": {
          "replicas": 5,
          "image": "registry.nephoran.com/oran/du:v2.1.0",
          "resources": {"cpu": "3000m", "memory": "6Gi"},
          "interfaces": ["f1_c", "f1_u"],
          "fronthaul_interface": "eCPRI"
        }
      }
    },
    "core_components": {
      "5g_core_deployment": {
        "amf": {
          "replicas": 3,
          "image": "registry.nephoran.com/5g-core/amf:v3.1.0",
          "resources": {"cpu": "1000m", "memory": "2Gi"},
          "scaling": {
            "hpa_enabled": true,
            "min_replicas": 2,
            "max_replicas": 10,
            "target_cpu": 70
          }
        },
        "smf": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/smf:v3.1.0", 
          "resources": {"cpu": "800m", "memory": "1.5Gi"},
          "scaling": {
            "hpa_enabled": true,
            "min_replicas": 2,
            "max_replicas": 8,
            "target_cpu": 70
          }
        },
        "upf": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/upf:v3.1.0",
          "resources": {"cpu": "2000m", "memory": "4Gi"},
          "data_plane_optimization": {
            "dpdk_enabled": true,
            "sr_iov_enabled": true
          },
          "scaling": {
            "hpa_enabled": true,
            "min_replicas": 2,
            "max_replicas": 6,
            "target_cpu": 80
          }
        },
        "nssf": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/nssf:v3.1.0",
          "resources": {"cpu": "500m", "memory": "1Gi"}
        },
        "udm": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/udm:v3.1.0", 
          "resources": {"cpu": "600m", "memory": "1.2Gi"}
        },
        "ausf": {
          "replicas": 2,
          "image": "registry.nephoran.com/5g-core/ausf:v3.1.0",
          "resources": {"cpu": "400m", "memory": "800Mi"}
        }
      }
    },
    "edge_components": {
      "mec_platform": {
        "mec_orchestrator": {
          "replicas": 1,
          "image": "registry.nephoran.com/mec/orchestrator:v1.3.0",
          "resources": {"cpu": "1000m", "memory": "2Gi"},
          "capabilities": ["app_lifecycle", "traffic_rules", "dns_rules"]
        },
        "edge_applications": [
          {
            "name": "ar-vr-processing",
            "replicas": 2,
            "image": "registry.nephoran.com/mec/ar-vr:v1.0.0",
            "resources": {"cpu": "4000m", "memory": "8Gi", "gpu": 1},
            "services": ["real_time_rendering", "motion_tracking"]
          },
          {
            "name": "iot-data-analytics",
            "replicas": 3,
            "image": "registry.nephoran.com/mec/iot-analytics:v2.1.0",
            "resources": {"cpu": "2000m", "memory": "4Gi"},
            "services": ["stream_processing", "ml_inference"]
          }
        ]
      }
    },
    "management_components": {
      "network_management": {
        "oss_bss": {
          "name": "network-oss",
          "replicas": 2,
          "image": "registry.nephoran.com/management/oss:v4.2.0",
          "resources": {"cpu": "1500m", "memory": "3Gi"},
          "modules": ["fcaps", "inventory", "configuration"]
        },
        "service_orchestrator": {
          "name": "service-orchestrator",
          "replicas": 2,
          "image": "registry.nephoran.com/management/orchestrator:v2.3.0",
          "resources": {"cpu": "1000m", "memory": "2Gi"},
          "capabilities": ["service_lifecycle", "sla_management", "resource_allocation"]
        }
      }
    },
    "network_slicing": {
      "slice_definitions": [
        {
          "slice_id": "embb-premium",
          "s_nssai": {"sst": 1, "sd": "000001"},
          "slice_type": "eMBB",
          "sla": {"latency_ms": 20, "throughput_mbps": 1000, "reliability": 0.999},
          "associated_components": {
            "ran": ["near_rt_ric", "cu_cp", "cu_up", "du"],
            "core": ["amf", "smf", "upf"],
            "edge": ["mec_orchestrator"]
          }
        },
        {
          "slice_id": "urllc-industrial",
          "s_nssai": {"sst": 2, "sd": "000001"},
          "slice_type": "URLLC", 
          "sla": {"latency_ms": 1, "throughput_mbps": 100, "reliability": 0.99999},
          "associated_components": {
            "ran": ["near_rt_ric", "cu_cp", "cu_up", "du"],
            "core": ["amf", "smf", "upf"],
            "edge": ["mec_orchestrator", "iot-data-analytics"]
          }
        }
      ]
    },
    "automated_scaling": {
      "scaling_policies": [
        {
          "component": "amf",
          "metrics": [
            {"name": "cpu_utilization", "target": 70},
            {"name": "registration_rate", "target": 1000}
          ],
          "scale_up": {"threshold": 80, "step": 1, "cooldown": "5m"},
          "scale_down": {"threshold": 50, "step": 1, "cooldown": "10m"}
        },
        {
          "component": "upf",
          "metrics": [
            {"name": "throughput_utilization", "target": 80},
            {"name": "packet_processing_rate", "target": 10000}
          ],
          "scale_up": {"threshold": 90, "step": 1, "cooldown": "3m"},
          "scale_down": {"threshold": 60, "step": 1, "cooldown": "15m"}
        }
      ],
      "predictive_scaling": {
        "enabled": true,
        "ml_model": "time_series_forecasting",
        "prediction_horizon_minutes": 30,
        "confidence_threshold": 0.8
      }
    },
    "comprehensive_monitoring": {
      "metrics_collection": {
        "prometheus_config": {
          "scrape_interval": "15s",
          "retention": "30d",
          "external_labels": {"deployment": "e2e-5g-service"}
        },
        "custom_metrics": [
          {"name": "e2e_latency", "description": "End-to-end service latency"},
          {"name": "slice_sla_compliance", "description": "Network slice SLA compliance rate"},
          {"name": "service_availability", "description": "Overall service availability"}
        ]
      },
      "alerting": {
        "alert_rules": [
          {
            "name": "ServiceDegradation",
            "condition": "e2e_latency > 50",
            "severity": "warning",
            "for": "2m"
          },
          {
            "name": "SLAViolation", 
            "condition": "slice_sla_compliance < 0.95",
            "severity": "critical",
            "for": "1m"
          }
        ]
      },
      "dashboards": [
        {
          "name": "End-to-End Service Overview",
          "panels": ["service_topology", "kpi_summary", "alert_status"]
        },
        {
          "name": "Network Slice Performance",
          "panels": ["slice_latency", "slice_throughput", "slice_reliability"]
        }
      ]
    },
    "deployment_orchestration": {
      "deployment_order": [
        "core_components",
        "ran_components", 
        "edge_components",
        "management_components",
        "network_slicing",
        "monitoring"
      ],
      "rollout_strategy": {
        "type": "blue_green",
        "validation_gates": ["health_check", "smoke_test", "performance_test"],
        "rollback_trigger": "sla_violation"
      },
      "dependency_management": {
        "core_to_ran": "required",
        "ran_to_edge": "optional",
        "monitoring_to_all": "parallel"
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 7340.5,
    "tokens_used": 5890,
    "retrieval_score": 0.88,
    "confidence_score": 0.91,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1704067800.345
}
```

## Intent Pattern Recognition

### Example 12: Pattern-Based Intent Classification

**Query:**
```
"The network is experiencing high latency in the RAN domain, please optimize QoS settings and increase CU-UP resources"
```

**Expected RAG Response:**
```json
{
  "intent_id": "pattern-recognition-001",
  "original_intent": "The network is experiencing high latency in the RAN domain, please optimize QoS settings and increase CU-UP resources",
  "structured_output": {
    "type": "MultiActionIntent",
    "intent_classification": {
      "primary_intent": "PerformanceOptimization", 
      "secondary_intents": ["ResourceScaling", "QoSConfiguration"],
      "domain": "RAN",
      "urgency": "high",
      "complexity": "medium"
    },
    "recognized_patterns": [
      {
        "pattern_id": "latency_optimization_pattern",
        "pattern_name": "Network Latency Optimization",
        "confidence": 0.94,
        "trigger_keywords": ["high latency", "RAN domain", "optimize"],
        "associated_actions": ["qos_optimization", "resource_scaling", "configuration_tuning"]
      },
      {
        "pattern_id": "resource_scaling_pattern", 
        "pattern_name": "Component Resource Scaling",
        "confidence": 0.89,
        "trigger_keywords": ["increase", "CU-UP resources", "optimize"],
        "associated_actions": ["horizontal_scaling", "vertical_scaling", "resource_allocation"]
      }
    ],
    "action_plan": [
      {
        "action_id": "cu_up_scaling",
        "action_type": "NetworkFunctionScale",
        "priority": 1,
        "target_component": "cu-up",
        "scaling_parameters": {
          "current_replicas": 3,
          "target_replicas": 5,
          "resource_increase": {
            "cpu": "50%",
            "memory": "30%"
          },
          "scaling_reason": "latency_optimization"
        }
      },
      {
        "action_id": "qos_optimization",
        "action_type": "QoSConfiguration", 
        "priority": 2,
        "target_components": ["cu-cp", "cu-up", "du"],
        "qos_parameters": {
          "scheduling_policy": "low_latency_first",
          "buffer_management": "aggressive_dropping",
          "priority_queuing": "enabled",
          "traffic_shaping": {
            "latency_critical_flows": {
              "max_delay_ms": 1,
              "jitter_control": "strict"
            }
          }
        }
      },
      {
        "action_id": "ran_optimization",
        "action_type": "RanConfiguration",
        "priority": 3,
        "optimization_areas": [
          {
            "area": "radio_resource_management",
            "parameters": {
              "scheduler_algorithm": "proportional_fair_low_latency",
              "tti_bundling": "enabled",
              "carrier_aggregation": "optimized"
            }
          },
          {
            "area": "fronthaul_optimization",
            "parameters": {
              "compression_ratio": "reduced_for_latency",
              "precoding_optimization": "enabled",
              "beamforming_updates": "increased_frequency"
            }
          }
        ]
      }
    ],
    "implementation_sequence": {
      "phase_1": {
        "name": "Immediate Scaling",
        "duration_minutes": 5,
        "actions": ["cu_up_scaling"],
        "success_criteria": ["increased_processing_capacity", "reduced_queue_depth"]
      },
      "phase_2": {
        "name": "QoS Optimization",
        "duration_minutes": 10,
        "actions": ["qos_optimization"],
        "success_criteria": ["improved_scheduling_efficiency", "reduced_packet_delay"]
      },
      "phase_3": {
        "name": "RAN Fine-tuning", 
        "duration_minutes": 15,
        "actions": ["ran_optimization"],
        "success_criteria": ["optimized_radio_efficiency", "improved_user_experience"]
      }
    },
    "expected_outcomes": [
      {
        "metric": "end_to_end_latency",
        "current_value": "25ms",
        "target_value": "15ms",
        "improvement_percentage": "40%"
      },
      {
        "metric": "cu_up_processing_delay",
        "current_value": "8ms", 
        "target_value": "4ms",
        "improvement_percentage": "50%"
      },
      {
        "metric": "user_plane_throughput",
        "current_value": "5Gbps",
        "target_value": "7.5Gbps", 
        "improvement_percentage": "50%"
      }
    ],
    "monitoring_and_validation": {
      "kpi_monitoring": [
        {
          "kpi": "latency_p95",
          "target": "<15ms",
          "measurement_interval": "30s"
        },
        {
          "kpi": "packet_loss_rate",
          "target": "<0.01%",
          "measurement_interval": "60s"
        },
        {
          "kpi": "resource_utilization",
          "target": "60-80%",
          "measurement_interval": "60s"
        }
      ],
      "validation_tests": [
        {
          "test_name": "latency_validation",
          "test_type": "synthetic_traffic",
          "duration_minutes": 30,
          "success_criteria": "latency_improvement >= 30%"
        },
        {
          "test_name": "capacity_validation",
          "test_type": "load_test",
          "duration_minutes": 60,
          "success_criteria": "no_performance_degradation"
        }
      ],
      "rollback_conditions": [
        {
          "condition": "latency_increase > 10%",
          "action": "revert_scaling"
        },
        {
          "condition": "error_rate > 1%",
          "action": "revert_qos_changes"
        }
      ]
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 4890.2,
    "tokens_used": 4120,
    "retrieval_score": 0.91,
    "confidence_score": 0.93,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "source_documents": [
    {
      "title": "O-RAN RAN Latency Optimization Guidelines",
      "source": "O-RAN.WG1",
      "version": "v5.0",
      "relevance_score": 0.92
    },
    {
      "title": "5G QoS Framework and Implementation",
      "source": "3GPP TS 23.501",
      "version": "17.9.0",
      "relevance_score": 0.89
    },
    {
      "title": "CU-UP Performance Optimization Best Practices",
      "source": "O-RAN.WG2",
      "version": "v4.1",
      "relevance_score": 0.87
    }
  ],
  "timestamp": 1704067860.678
}
```

## Usage Guidelines and Best Practices

### Query Optimization Tips

1. **Be Specific**: Include specific network function names, interface types, and technical requirements
2. **Use Standard Terminology**: Leverage 3GPP and O-RAN standard terminology for better results
3. **Provide Context**: Include deployment environment, scale requirements, and performance targets
4. **Multi-Domain Queries**: For complex scenarios, break down into domain-specific components

### Response Interpretation

1. **Confidence Scores**: Higher scores (>0.9) indicate high reliability of the generated configuration
2. **Source Documents**: Review source documents for additional context and validation
3. **Implementation Considerations**: Pay attention to deployment considerations and prerequisites
4. **Monitoring Recommendations**: Follow provided monitoring guidelines for production deployments

### Integration with NetworkIntent Controller

The knowledge base examples demonstrate how natural language intents are processed and converted into structured NetworkIntent CRD instances. The RAG system provides the intelligent translation layer between human operators and cloud-native network function deployments.

This comprehensive knowledge base enables operators to express complex network operations in natural language while ensuring accurate, standards-compliant network function deployments across O-RAN and 5G Core infrastructure.