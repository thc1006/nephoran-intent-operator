# Product Overview

The Nephoran Intent Operator is a cloud-native orchestration system that manages O-RAN compliant network functions using Large Language Models (LLM) as the primary control interface. It serves as a proof-of-concept for autonomous network operations.

## Core Concept
- Translates natural language intents into concrete Kubernetes configurations
- Leverages GitOps principles from Nephio for network function orchestration
- Enables autonomous scale-out/scale-in of O-RAN E2 Nodes based on high-level goals
- Bridges the gap between human intent and complex telecom network configurations

## Key Components
- **LLM Intent Processing**: Converts natural language to structured NetworkIntent CRs
- **Nephio Bridge Controller**: Core controller that watches intents and generates KRM manifests
- **O-RAN Interface Bridges**: Manages O-RAN specific network functions (Near-RT RIC, E2 simulators)
- **GitOps Workflow**: Uses Git repositories for intent storage and deployment synchronization

## Target Use Cases
- Autonomous network operations for telecom operators
- O-RAN network function lifecycle management
- Intent-driven network slice management
- Development and testing of AI-driven network orchestration