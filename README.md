# LLM-Enhanced Nephio R5 and O-RAN Network Automation System

This project is an LLM-Enhanced Nephio R5 and O-RAN Network Automation System. It integrates a Large Language Model with Nephio's intent-based automation to provide a natural language interface for managing and orchestrating telecommunications network functions.

## Project Architecture

The system is composed of several key layers that work together to translate natural language commands into network configurations.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Operator Interface Layer                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  Web UI / CLI / REST API                    Natural Language Intent Input   │
└─────────────────────────┬───────────────────────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────────────────────┐
│                        LLM/RAG Processing Layer                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────────────────┐ │
│  │   Mistral-8x22B │  │  Haystack RAG    │  │   Telco-RAG Optimization   │ │
│  │   Inference     │  │  Framework       │  │   - Technical Glossaries   │ │
│  │   Engine        │  │  - Vector DB     │  │   - Query Enhancement      │ │
│  │                 │  │  - Semantic      │  │   - Document Router        │ │
│  │                 │  │    Retrieval     │  │                             │ │
│  └─────────────────┘  └──────────────────┘  └─────────────────────────────┘ │
└─────────────────────────┬───────────────────────────────────────────────────┘
                          │ Intent Translation & Validation
┌─────────────────────────▼───────────────────────────────────────────────────┐
│                      Nephio R5 Control Plane                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────────────────┐ │
│  │  Porch Package  │  │   ConfigSync/    │  │    Nephio Controllers      │ │
│  │  Orchestration  │  │   ArgoCD GitOps  │  │    - Intent Reconciliation │ │
│  │  - API Server   │  │   - Multi-cluster│  │    - Policy Enforcement    │ │
│  │  - Function     │  │   - Drift Detect │  │    - Resource Management   │ │
│  │    Runtime      │  │                  │  │                             │ │
│  └─────────────────┘  └──────────────────┘  └─────────────────────────────┘ │
└─────────────────────────┬───────────────────────────────────────────────────┘
                          │ KRM Generation & Git Synchronization
┌─────────────────────────▼───────────────────────────────────────────────────┐
│                       O-RAN Interface Bridge Layer                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────────────────┐ │
│  │  SMO Adaptor    │  │  RIC Integration │  │    Interface Controllers    │ │
│  │  - A1 Interface │  │  - Non-RT RIC    │  │    - O1 (FCAPS Mgmt)       │ │
│  │  - R1 Interface │  │  - Near-RT RIC   │  │    - O2 (Cloud Infra)      │ │
│  │  - Policy Mgmt  │  │  - xApp/rApp     │  │    - E2 (RAN Control)      │ │
│  │                 │  │    Orchestration │  │                             │ │
│  └─────────────────┘  └──────────────────┘  └─────────────────────────────┘ │
└─────────────────────────┬───────────────────────────────────────────────────┘
                          │ Network Function Control & Monitoring
┌─────────────────────────▼───────────────────────────────────────────────────┐
│                    Network Function Orchestration                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────────────────┐ │
│  │   5G Core NFs   │  │   O-RAN Network  │  │    Network Slice Manager    │ │
│  │   - UPF, SMF    │  │   Functions      │  │    - Dynamic Allocation    │ │
│  │   - AMF, NSSF   │  │   - O-DU, O-CU   │  │    - QoS Management        │ │
│  │   - Custom NFs  │  │   - Near-RT RIC  │  │    - SLA Enforcement       │ │
│  └─────────────────┘  └──────────────────┘  └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

1.  **Operator Input:** A network operator provides a natural language command via a CLI, Web UI, or REST API.
2.  **LLM/RAG Processing:** The command is sent to the LLM/RAG Processing Layer, which uses a telecom-optimized RAG framework to understand the intent and translate it into a structured format.
3.  **Nephio Control Plane:** The structured intent is passed to the Nephio R5 Control Plane, which generates the necessary Kubernetes Resource Model (KRM) packages.
4.  **GitOps Sync:** The KRM packages are committed to a Git repository, and a GitOps tool (like ArgoCD or FluxCD) synchronizes them with the cluster.
5.  **O-RAN Interface Bridge:** The O-RAN Interface Bridge Layer watches for the KRM resources and translates them into the appropriate O-RAN interface calls (A1, O1, O2, etc.).
6.  **Network Function Orchestration:** The O-RAN components then orchestrate the network functions as specified in the intent.

## Getting Started

This project uses a development container to provide a consistent and reproducible development environment.

### Prerequisites

*   Docker
*   VS Code with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

### Setup

1.  Clone this repository.
2.  Open the repository in VS Code.
3.  When prompted, click "Reopen in Container". This will build the development container and install all the necessary dependencies.
4.  Once the container is running, open a terminal in VS Code and run the following command to set up the project:

    ```bash
    make setup-dev
    ```

### Development Workflow

The `Makefile` provides several targets to streamline the development process:

*   `make setup-dev`: Sets up the development environment by installing Go and Python dependencies.
*   `make build-all`: Builds all the service binaries.
*   `make deploy-dev`: Deploys all the components to the development environment using Kustomize.
*   `make test-integration`: Runs all integration tests.