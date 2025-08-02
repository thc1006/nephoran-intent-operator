# Nephoran RAG Pipeline - Architecture Diagrams and Data Flow

## Overview

This document provides comprehensive architectural diagrams and data flow documentation for the Nephoran Intent Operator's RAG (Retrieval-Augmented Generation) pipeline. The diagrams illustrate system components, data flows, integration patterns, and deployment architectures across different environments.

## Table of Contents

1. [System Architecture Overview](#system-architecture-overview)
2. [Component Architecture](#component-architecture)
3. [Data Flow Diagrams](#data-flow-diagrams)
4. [Integration Patterns](#integration-patterns)
5. [Deployment Architectures](#deployment-architectures)
6. [Network Architecture](#network-architecture)
7. [Security Architecture](#security-architecture)
8. [Monitoring Architecture](#monitoring-architecture)

## System Architecture Overview

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                           Nephoran Intent Operator - Complete System Architecture                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                                 User Interface Layer                                                               │ │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐  │ │
│  │  │   Web UI        │    │   CLI Tools     │    │   REST API      │    │   GraphQL API   │    │   Third-party Integrations │  │ │
│  │  │   Dashboard     │    │   kubectl       │    │   Endpoints     │    │   Query Engine  │    │   External Systems         │  │ │
│  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│                                                           ▼                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                                Kubernetes API Layer                                                                │ │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐  │ │
│  │  │ NetworkIntent   │    │ E2NodeSet       │    │ ManagedElement  │    │ ConfigMaps      │    │   Secrets & Service        │  │ │
│  │  │ CRD             │    │ CRD             │    │ CRD             │    │ & PVCs          │    │   Accounts                  │  │ │
│  │  │                 │    │                 │    │                 │    │                 │    │                             │  │ │
│  │  │ • Intent Text   │    │ • Replica Count │    │ • Element Info  │    │ • Config Data   │    │ • API Keys                  │  │ │
│  │  │ • Status        │    │ • Node Config   │    │ • O-RAN Config  │    │ • Templates     │    │ • Certificates              │  │ │
│  │  │ • Parameters    │    │ • Health Status │    │ • Policies      │    │ • Cache Data    │    │ • Authentication            │  │ │
│  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│                                                           ▼                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                               Controller & Processing Layer                                                        │ │
│  │                                                                                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                           NetworkIntent Controller                                                             │ │ │
│  │  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────────────────────────┐ │ │ │
│  │  │  │   Reconciler    │    │   Status        │    │   Validation    │    │         LLM Integration                         │ │ │ │
│  │  │  │   Loop          │────│   Manager       │────│   Engine        │────│         & RAG Processing                       │ │ │ │
│  │  │  │                 │    │                 │    │                 │    │                                                 │ │ │ │
│  │  │  │ • Event Watch   │    │ • Health Check  │    │ • Schema Valid  │    │ • Intent Classification                         │ │ │ │
│  │  │  │ • Resource Sync │    │ • Error Track   │    │ • Semantic Val  │    │ • Context Retrieval                             │ │ │ │
│  │  │  │ • Queue Mgmt    │    │ • Metrics       │    │ • Business Rules│    │ • Parameter Generation                          │ │ │ │
│  │  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                              LLM Processor Service                                                            │ │ │
│  │  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────────────────────────┐ │ │ │
│  │  │  │   REST API      │    │   Health &      │    │   Circuit       │    │         Request/Response                        │ │ │ │
│  │  │  │   Server        │────│   Monitoring    │────│   Breaker       │────│         Processing                              │ │ │ │
│  │  │  │                 │    │                 │    │                 │    │                                                 │ │ │ │
│  │  │  │ • /process      │    │ • /healthz      │    │ • Failure Detect│    │ • Request Validation                            │ │ │ │
│  │  │  │ • /readyz       │    │ • /metrics      │    │ • Auto Recovery │    │ • Response Formatting                           │ │ │ │
│  │  │  │ • Load Balance  │    │ • Dependency    │    │ • Rate Limiting │    │ • Error Handling                                │ │ │ │
│  │  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                             E2NodeSet Controller                                                              │ │ │
│  │  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────────────────────────┐ │ │ │
│  │  │  │   Replica       │    │   ConfigMap     │    │   Scaling       │    │         Health Monitoring                       │ │ │ │
│  │  │  │   Manager       │────│   Generator     │────│   Operations    │────│         & Status Updates                        │ │ │ │
│  │  │  │                 │    │                 │    │                 │    │                                                 │ │ │ │
│  │  │  │ • Desired State │    │ • Node Config   │    │ • Scale Up/Down │    │ • Node Health Checks                            │ │ │ │
│  │  │  │ • Current State │    │ • Simulated gNB │    │ • Rolling Update│    │ • Ready Replica Count                           │ │ │ │
│  │  │  │ • Reconcile     │    │ • Labels/Annot  │    │ • Error Recovery│    │ • Event Generation                              │ │ │ │
│  │  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│                                                           ▼                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                               RAG (Retrieval-Augmented Generation) Layer                                         │ │
│  │                                                                                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                              RAG API Service                                                                  │ │ │
│  │  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────────────────────────┐ │ │ │
│  │  │  │   Flask API     │    │   Document      │    │   Statistics    │    │         Health & Metrics                        │ │ │ │
│  │  │  │   Server        │────│   Management    │────│   & Analytics   │────│         Collection                              │ │ │ │
│  │  │  │                 │    │                 │    │                 │    │                                                 │ │ │ │
│  │  │  │ • /process_intent│   │ • /upload       │    │ • /stats        │    │ • /healthz                                      │ │ │ │
│  │  │  │ • /query        │    │ • /populate     │    │ • /metrics      │    │ • /readyz                                       │ │ │ │
│  │  │  │ • Async Process │    │ • Batch Upload  │    │ • Cache Status  │    │ • Dependency Checks                             │ │ │ │
│  │  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                           RAG Pipeline Components                                                             │ │ │
│  │  │                                                                                                                               │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                        Document Processing Pipeline                                                      │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Document        │  │ Intelligent     │  │ Metadata        │  │ Embedding       │  │    Quality Assessment      │ │ │ │ │
│  │  │  │  │ Loader          │──│ Chunking        │──│ Extraction      │──│ Generation      │──│    & Validation            │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • PDF Parser    │  │ • Hierarchy     │  │ • 3GPP/O-RAN    │  │ • OpenAI API    │  │ • Content Quality Score     │ │ │ │ │
│  │  │  │  │ • Multi-format  │  │ • Boundaries    │  │ • Tech Terms    │  │ • Batch Process │  │ • Confidence Calculation    │ │ │ │ │
│  │  │  │  │ • Remote URLs   │  │ • Context Pres  │  │ • Working Groups│  │ • Rate Limiting │  │ • Metadata Validation       │ │ │ │ │
│  │  │  │  │ • Batch Process │  │ • Quality Score │  │ • Categories    │  │ • Caching       │  │ • Duplicate Detection       │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                                    │                                                           │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                          Vector Storage Layer                                                           │ │ │ │
│  │  │  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │ │
│  │  │  │  │                                  Weaviate Vector Database Cluster                                                │ │ │ │ │
│  │  │  │  │                                                                                                                   │ │ │ │ │
│  │  │  │  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────────────┐ │ │ │ │ │
│  │  │  │  │  │ TelecomDocument │    │ IntentPattern   │    │ NetworkFunction │    │        Backup & Recovery           │ │ │ │ │ │
│  │  │  │  │  │ Collection      │    │ Collection      │    │ Collection      │    │        System                      │ │ │ │ │ │
│  │  │  │  │  │                 │    │                 │    │                 │    │                                     │ │ │ │ │ │
│  │  │  │  │  │ • 3GPP Specs    │    │ • NL Intents    │    │ • AMF/SMF/UPF   │    │ • Daily Backups                     │ │ │ │ │ │
│  │  │  │  │  │ • O-RAN Docs    │    │ • Config Temps  │    │ • gNB/CU/DU     │    │ • Point-in-time Recovery            │ │ │ │ │ │
│  │  │  │  │  │ • ETSI Standards│    │ • Query Patterns│    │ • Interface Specs│   │ • Cross-AZ Replication              │ │ │ │ │ │
│  │  │  │  │  │ • ITU Documents │    │ • Response Schema│   │ • Policy Templates│  │ • Automated Cleanup                 │ │ │ │ │ │
│  │  │  │  │  │ • Embeddings    │    │ • Context Examples│  │ • Deploy Patterns│  │ • Health Monitoring                 │ │ │ │ │ │
│  │  │  │  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────────────┘ │ │ │ │ │
│  │  │  │  │                                                                                                                   │ │ │ │ │
│  │  │  │  │  Features: High Availability (3-10 replicas), Auto-scaling, API Authentication,                               │ │ │ │ │
│  │  │  │  │           500GB-2TB Storage, Monitoring Integration, Network Policies                                           │ │ │ │ │
│  │  │  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                                    │                                                           │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                         Query Processing Pipeline                                                       │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Query           │  │ Hybrid Search   │  │ Semantic        │  │ Context         │  │    Response Assembly        │ │ │ │ │
│  │  │  │  │ Enhancement     │──│ Engine          │──│ Reranking       │──│ Assembly        │──│    & Validation             │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Acronym Exp   │  │ • Vector Sim    │  │ • Cross-encoder │  │ • Strategy Sel  │  │ • JSON Schema Validation    │ │ │ │ │
│  │  │  │  │ • Synonym Enh   │  │ • Keyword Match │  │ • Multi-factor  │  │ • Hierarchy     │  │ • Quality Metrics           │ │ │ │ │
│  │  │  │  │ • Spell Correct │  │ • Hybrid Alpha  │  │ • Authority Wgt │  │ • Source Balance│  │ • Processing Metadata       │ │ │ │ │
│  │  │  │  │ • Context Integ │  │ • Filtering     │  │ • Freshness     │  │ • Token Mgmt    │  │ • Confidence Scoring        │ │ │ │ │
│  │  │  │  │ • Intent Class  │  │ • Boost Weights │  │ • Diversity     │  │ • Quality Opt   │  │ • Debug Information         │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                            Support Infrastructure                                                              │ │ │
│  │  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐ │ │ │
│  │  │  │ Redis Cache     │    │ Configuration   │    │ Secret          │    │ Health &        │    │   External API              │ │ │ │
│  │  │  │ System          │    │ Management      │    │ Management      │    │ Monitoring      │    │   Integrations              │ │ │ │
│  │  │  │                 │    │                 │    │                 │    │                 │    │                             │ │ │ │
│  │  │  │ • Multi-level   │    │ • Environment   │    │ • Vault         │    │ • Prometheus    │    │ • OpenAI API                │ │ │ │
│  │  │  │ • TTL Mgmt      │    │ • ConfigMaps    │    │ • Kubernetes    │    │ • Grafana       │    │ • Azure OpenAI              │ │ │ │
│  │  │  │ • Compression   │    │ • Multi-tenant  │    │ • Rotation      │    │ • Custom Metrics│    │ • Local Models              │ │ │ │
│  │  │  │ • Performance   │    │ • Validation    │    │ • Encryption    │    │ • Alerting      │    │ • Third-party APIs          │ │ │ │
│  │  │  │ • Health Check  │    │ • Profiles      │    │ • Audit Logs    │    │ • Tracing       │    │ • Webhook Integrations      │ │ │ │
│  │  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│                                                           ▼                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                              O-RAN Interface & Network Function Layer                                             │ │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐  │ │
│  │  │ A1 Interface    │    │ O1 Interface    │    │ O2 Interface    │    │ E2 Interface    │    │   GitOps Package           │  │ │
│  │  │ (Policy Mgmt)   │    │ (FCAPS Mgmt)    │    │ (Cloud Infra)   │    │ (RAN Control)   │    │   Generation               │  │ │
│  │  │                 │    │                 │    │                 │    │                 │    │                             │  │ │
│  │  │ • Near-RT RIC   │    │ • Configuration │    │ • Infrastructure│    │ • Real-time     │    │ • KRM Templates             │  │ │
│  │  │ • Policy Types  │    │ • Performance   │    │ • Orchestration │    │ • RAN Functions │    │ • Kpt Packages              │  │ │
│  │  │ • QoS Control   │    │ • Fault Mgmt    │    │ • Resource Mgmt │    │ • Handovers     │    │ • Nephio Integration        │  │ │
│  │  │ • Traffic Steer │    │ • Security      │    │ • Scaling       │    │ • Measurements  │    │ • Git Repository Sync       │  │ │
│  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│                                                           ▼                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                              Network Function Orchestration Layer                                                 │ │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐  │ │
│  │  │ 5G Core NFs     │    │ O-RAN Network   │    │ Network Slice   │    │ Edge Computing  │    │   Monitoring &             │  │ │
│  │  │                 │    │ Functions       │    │ Management      │    │ Functions       │    │   Observability            │  │ │
│  │  │                 │    │                 │    │                 │    │                 │    │                             │  │ │
│  │  │ • AMF, SMF, UPF │    │ • gNB, CU, DU   │    │ • Slice Types   │    │ • MEC Apps      │    │ • Telemetry Collection      │  │ │
│  │  │ • AUSF, UDM     │    │ • Near-RT RIC   │    │ • QoS Policies  │    │ • Edge Services │    │ • Performance Metrics       │  │ │
│  │  │ • PCF, NRF      │    │ • Non-RT RIC    │    │ • SLA Mgmt      │    │ • CDN Services  │    │ • Health Monitoring         │  │ │
│  │  │ • NSSF, NEF     │    │ • O-Cloud       │    │ • Tenant Isolat │    │ • AI/ML Inference│   │ • Alert Management          │  │ │
│  │  └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
```

### Component Relationships and Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                  Component Dependency Graph                                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                     │
│  ┌─────────────────┐                    ┌─────────────────┐                    ┌─────────────────────────────┐    │
│  │ User Interface  │───────────────────▶│ Kubernetes API  │───────────────────▶│ Controllers                 │    │
│  │ Layer           │                    │ Server          │                    │ (NetworkIntent, E2NodeSet)  │    │
│  └─────────────────┘                    └─────────────────┘                    └─────────────────────────────┘    │
│                                                                                               │                     │
│                                                                                               ▼                     │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐             ┌─────────────────────────────┐    │
│  │ External APIs   │◄───│ LLM Processor   │◄───│ RAG API Service │◄────────────│ RAG Pipeline                │    │
│  │ (OpenAI, etc.)  │    │ Service         │    │                 │             │ Components                  │    │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘             └─────────────────────────────┘    │
│           ▲                                               │                                   │                     │
│           │                                               ▼                                   ▼                     │
│           │              ┌─────────────────┐    ┌─────────────────┐             ┌─────────────────────────────┐    │
│           └──────────────│ Redis Cache     │    │ Weaviate Vector │             │ Document Processing         │    │
│                          │ Cluster         │    │ Database        │             │ Pipeline                    │    │
│                          └─────────────────┘    └─────────────────┘             └─────────────────────────────┘    │
│                                    │                       │                                   │                     │
│                                    │                       ▼                                   ▼                     │
│                                    │              ┌─────────────────┐             ┌─────────────────────────────┐    │
│                                    │              │ Vector Storage  │             │ Knowledge Base              │    │
│                                    │              │ & Indexing      │             │ Management                  │    │
│                                    │              └─────────────────┘             └─────────────────────────────┘    │
│                                    │                                                                                  │
│                                    ▼                                                                                  │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐             ┌─────────────────────────────┐    │
│  │ Monitoring &    │    │ Configuration   │    │ Secret          │             │ Network Policies            │    │
│  │ Observability   │    │ Management      │    │ Management      │             │ & Security                  │    │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘             └─────────────────────────────┘    │
│                                                                                                                     │
│  Dependencies Legend:                                                                                               │
│  ───▶ Direct Dependency        ◄──▶ Bidirectional Communication        ┈┈▶ Optional Dependency                   │
│  ═══▶ Critical Path            ░░░▶ Monitoring/Observability            ▓▓▶ Configuration Dependency              │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### RAG Pipeline Detailed Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                           RAG Pipeline - Detailed Component Architecture                                                │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                              Input Processing Layer                                                                │ │
│  │                                                                                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                           Document Ingestion                                                                  │ │ │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │ │ │
│  │  │  │ File System     │  │ HTTP/HTTPS      │  │ Object Storage  │  │ Database        │  │    Version Control          │  │ │ │
│  │  │  │ Watcher         │  │ Endpoints       │  │ (S3, GCS, etc.) │  │ Connectors      │  │    (Git, SVN)               │  │ │ │
│  │  │  │                 │  │                 │  │                 │  │                 │  │                             │  │ │ │
│  │  │  │ • Local Files   │  │ • Remote URLs   │  │ • Bucket Watch  │  │ • SQL Queries   │  │ • Repository Hooks          │  │ │ │
│  │  │  │ • Directory Scan│  │ • API Endpoints │  │ • Event Triggers│  │ • NoSQL Docs    │  │ • Branch Monitoring         │  │ │ │
│  │  │  │ • File Events   │  │ • Webhooks      │  │ • Scheduled Sync│  │ • Graph Queries │  │ • Commit Tracking           │  │ │ │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                          Format Detection & Parsing                                                            │ │ │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │ │ │
│  │  │  │ PDF Parser      │  │ Microsoft       │  │ Markup          │  │ Structured      │  │    Binary & Media           │  │ │ │
│  │  │  │                 │  │ Office Docs     │  │ Languages       │  │ Data Formats    │  │    Processors               │  │ │ │
│  │  │  │                 │  │                 │  │                 │  │                 │  │                             │  │ │ │
│  │  │  │ • Text Extract  │  │ • Word (.docx)  │  │ • Markdown      │  │ • JSON, YAML    │  │ • Image OCR                 │  │ │ │
│  │  │  │ • Metadata      │  │ • Excel (.xlsx) │  │ • HTML, XML     │  │ • CSV, TSV      │  │ • Audio Transcription       │  │ │ │
│  │  │  │ • Structure     │  │ • PowerPoint    │  │ • ReStructured  │  │ • Protocol Buf  │  │ • Video Analysis            │  │ │ │
│  │  │  │ • Table Extract │  │ • Visio         │  │ • AsciiDoc      │  │ • Apache Parquet│  │ • Archive Extraction        │  │ │ │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                         Content Preprocessing                                                                   │ │ │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │ │ │
│  │  │  │ Text            │  │ Language        │  │ Content         │  │ Structure       │  │    Quality                  │  │ │ │
│  │  │  │ Normalization   │  │ Detection       │  │ Filtering       │  │ Analysis        │  │    Assessment               │  │ │ │
│  │  │  │                 │  │                 │  │                 │  │                 │  │                             │  │ │ │
│  │  │  │ • Unicode Fix   │  │ • Auto Detect   │  │ • Noise Removal │  │ • Header Detect │  │ • Content Length            │  │ │ │
│  │  │  │ • Encoding      │  │ • Confidence    │  │ • Boilerplate   │  │ • Section Parse │  │ • Readability Score         │  │ │ │
│  │  │  │ • Line Breaks   │  │ • Multi-lang    │  │ • Copyright     │  │ • TOC Extract   │  │ • Technical Density         │  │ │ │
│  │  │  │ • Whitespace    │  │ • Translation   │  │ • Navigation    │  │ • Hierarchy     │  │ • Information Entropy       │  │ │ │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│  ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────────────────────┐ │
│  │                                              Processing Layer                                                                       │ │
│  │                                                                                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                      Metadata Extraction Engine                                                               │ │ │
│  │  │                                                                                                                               │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                 Telecom-Specific Extractors                                                            │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Standards       │  │ Network         │  │ Technical       │  │ Organizational  │  │    Temporal                 │ │ │ │ │
│  │  │  │  │ Recognition     │  │ Function        │  │ Classification  │  │ Metadata        │  │    Information              │ │ │ │ │
│  │  │  │  │                 │  │ Detection       │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • 3GPP TS/TR    │  │ • AMF, SMF, UPF │  │ • RAN, Core     │  │ • Working Group │  │ • Release Dates             │ │ │ │ │
│  │  │  │  │ • O-RAN WG      │  │ • gNB, CU, DU   │  │ • Transport     │  │ • Authors       │  │ • Version History           │ │ │ │ │
│  │  │  │  │ • ETSI GS       │  │ • AUSF, UDM     │  │ • Management    │  │ • Approval      │  │ • Change Tracking           │ │ │ │ │
│  │  │  │  │ • ITU-T Rec     │  │ • PCF, NRF      │  │ • Security      │  │ • Status        │  │ • Obsolescence              │ │ │ │ │
│  │  │  │  │ • IEEE Std      │  │ • Near-RT RIC   │  │ • Use Cases     │  │ • Dependencies  │  │ • Lifecycle Stage           │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                           │                                                                   │ │ │
│  │  │  ┌───────────────────────────────────────────────────────▼───────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                   Keyword & Entity Extraction                                                          │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Technical       │  │ Named Entity    │  │ Acronym         │  │ Relationship    │  │    Context                  │ │ │ │ │
│  │  │  │  │ Term Extract    │  │ Recognition     │  │ Expansion       │  │ Extraction      │  │    Analysis                 │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Domain Terms  │  │ • Organizations │  │ • 3GPP Acronyms │  │ • Dependencies  │  │ • Co-occurrence             │ │ │ │ │
│  │  │  │  │ • Protocols     │  │ • Technologies  │  │ • O-RAN Terms   │  │ • References    │  │ • Semantic Relations        │ │ │ │ │
│  │  │  │  │ • Procedures    │  │ • Standards     │  │ • Technical     │  │ • Hierarchies   │  │ • Topic Modeling            │ │ │ │ │
│  │  │  │  │ • Interfaces    │  │ • Products      │  │ • Expansions    │  │ • Cross-refs    │  │ • Concept Clustering        │ │ │ │ │
│  │  │  │  │ • Parameters    │  │ • People        │  │ • Definitions   │  │ • Citations     │  │ • Knowledge Graphs          │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                           │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                         Intelligent Chunking Engine                                                           │ │ │
│  │  │                                                                                                                               │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                      Chunking Strategies                                                                │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Hierarchical    │  │ Semantic        │  │ Technical       │  │ Sliding Window  │  │    Adaptive                 │ │ │ │ │
│  │  │  │  │ Chunking        │  │ Boundary        │  │ Context         │  │ Approach        │  │    Chunking                 │ │ │ │ │
│  │  │  │  │                 │  │ Detection       │  │ Preservation    │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Document      │  │ • Paragraph     │  │ • Term Grouping │  │ • Overlap Mgmt  │  │ • Content-based Sizing      │ │ │ │ │
│  │  │  │  │ • Section       │  │ • Sentence      │  │ • Procedure     │  │ • Context Chain │  │ • Quality-driven Split      │ │ │ │ │
│  │  │  │  │ • Subsection    │  │ • Topic Shift   │  │ • Interface     │  │ • Token Balance │  │ • Information Density       │ │ │ │ │
│  │  │  │  │ • List Items    │  │ • Coherence     │  │ • Protocol      │  │ • Boundary Opt  │  │ • Semantic Completeness     │ │ │ │ │
│  │  │  │  │ • Table Rows    │  │ • Flow Breaks   │  │ • Standards     │  │ • Memory Aware  │  │ • Multi-objective Opt       │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                           │                                                                   │ │ │
│  │  │  ┌───────────────────────────────────────────────────────▼───────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                      Quality Assessment                                                                 │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Content         │  │ Coherence       │  │ Completeness    │  │ Technical       │  │    Information              │ │ │ │ │
│  │  │  │  │ Quality         │  │ Analysis        │  │ Verification    │  │ Accuracy        │  │    Density                  │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Length Check  │  │ • Flow Analysis │  │ • Concept Cover │  │ • Term Validity │  │ • Keyword Density           │ │ │ │ │
│  │  │  │  │ • Readability   │  │ • Topic Unity   │  │ • Reference     │  │ • Standard Comp │  │ • Information Entropy       │ │ │ │ │
│  │  │  │  │ • Language      │  │ • Logical Flow  │  │ • Context Depth │  │ • Fact Checking │  │ • Technical Complexity      │ │ │ │ │
│  │  │  │  │ • Structure     │  │ • Transition    │  │ • Scope Clarity │  │ • Cross Validat │  │ • Knowledge Richness        │ │ │ │ │
│  │  │  │  │ • Formatting    │  │ • Coherence Sc  │  │ • Detail Level  │  │ • Authority Sc  │  │ • Semantic Richness         │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│  ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────────────────────┐ │
│  │                                              Vector Generation Layer                                                               │ │
│  │                                                                                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                       Embedding Generation Engine                                                            │ │ │
│  │  │                                                                                                                               │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                   Multi-Provider Embedding Support                                                     │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ OpenAI          │  │ Azure OpenAI    │  │ Local Models    │  │ Hugging Face    │  │    Custom                   │ │ │ │ │
│  │  │  │  │ Embeddings      │  │ Service         │  │                 │  │ Embeddings      │  │    Embedding Models         │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • ada-002       │  │ • ada-002       │  │ • Sentence-BERT │  │ • all-MiniLM    │  │ • Domain-specific           │ │ │ │ │
│  │  │  │  │ • text-embed-3  │  │ • text-embed-3  │  │ • E5 Models     │  │ • BGE Models    │  │ • Fine-tuned Models         │ │ │ │ │
│  │  │  │  │ • Rate Limiting │  │ • Private Deploy│  │ • GPU Optimized │  │ • Multilingual  │  │ • Telecom-optimized         │ │ │ │ │
│  │  │  │  │ • Cost Tracking │  │ • Data Residency│  │ • No API Limits │  │ • Lightweight   │  │ • Industry-specific         │ │ │ │ │
│  │  │  │  │ • Quality Focus │  │ • Enterprise Sec│  │ • Privacy First │  │ • Open Source   │  │ • Multi-modal Support       │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                           │                                                                   │ │ │
│  │  │  ┌───────────────────────────────────────────────────────▼───────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                    Processing Optimization                                                             │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Batch           │  │ Rate Limiting   │  │ Caching         │  │ Load Balancing  │  │    Quality Assurance        │ │ │ │ │
│  │  │  │  │ Processing      │  │ & Throttling    │  │ System          │  │ & Failover      │  │    & Validation             │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Optimal Size  │  │ • Token/Min     │  │ • Content Hash  │  │ • Provider Rot  │  │ • Dimension Validation      │ │ │ │ │
│  │  │  │  │ • Parallel Proc │  │ • Request/Min   │  │ • TTL Mgmt      │  │ • Health Check  │  │ • Vector Normalization      │ │ │ │ │
│  │  │  │  │ • Memory Opt    │  │ • Exponential   │  │ • LRU Eviction  │  │ • Auto Failover │  │ • Similarity Validation     │ │ │ │ │
│  │  │  │  │ • Progress Track│  │ • Backoff       │  │ • Compression   │  │ • Circuit Break │  │ • Outlier Detection         │ │ │ │ │
│  │  │  │  │ • Error Recovery│  │ • Queue Mgmt    │  │ • Persistence   │  │ • Retry Logic   │  │ • Quality Scoring           │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                           │                                                                             │
│  ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────────────────────┐ │
│  │                                              Storage & Indexing Layer                                                             │ │
│  │                                                                                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                         Weaviate Vector Database                                                             │ │ │
│  │  │                                                                                                                               │ │ │
│  │  │  ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                     Schema Management                                                                   │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ TelecomDocument │  │ IntentPattern   │  │ NetworkFunction │  │ ConfigTemplate  │  │    Domain Knowledge         │ │ │ │ │
│  │  │  │  │ Class           │  │ Class           │  │ Class           │  │ Class           │  │    Extensions               │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Content       │  │ • Intent Text   │  │ • Function Name │  │ • Config YAML   │  │ • Custom Properties         │ │ │ │ │
│  │  │  │  │ • Title         │  │ • Intent Type   │  │ • Description   │  │ • Template      │  │ • Relationships            │ │ │ │ │
│  │  │  │  │ • Source        │  │ • Parameters    │  │ • Interfaces    │  │ • Parameters    │  │ • Cross-references          │ │ │ │ │
│  │  │  │  │ • Category      │  │ • Examples      │  │ • Dependencies  │  │ • Validation    │  │ • Semantic Links            │ │ │ │ │
│  │  │  │  │ • Keywords      │  │ • Responses     │  │ • Use Cases     │  │ • Best Practice │  │ • Knowledge Graphs          │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                           │                                                                   │ │ │
│  │  │  ┌───────────────────────────────────────────────────────▼───────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                       Index Optimization                                                               │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ HNSW            │  │ Quantization    │  │ Compression     │  │ Partitioning    │  │    Performance              │ │ │ │ │
│  │  │  │  │ Configuration   │  │ Strategies      │  │ Algorithms      │  │ Strategies      │  │    Tuning                   │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • efConstruction│  │ • Product Quant │  │ • PCA           │  │ • Horizontal    │  │ • Memory Optimization       │ │ │ │ │
│  │  │  │  │ • M Connections │  │ • Scalar Quant  │  │ • Random Proj   │  │ • Vertical      │  │ • CPU Optimization          │ │ │ │ │
│  │  │  │  │ • ef Search     │  │ • Binary Quant  │  │ • LSH           │  │ • Time-based    │  │ • I/O Optimization          │ │ │ │ │
│  │  │  │  │ • Dynamic ef    │  │ • Hierarchical  │  │ • Sparse Encode │  │ • Content-based │  │ • Network Optimization      │ │ │ │ │
│  │  │  │  │ • Max Conn      │  │ • Adaptive      │  │ • Delta Encoding│  │ • Load Balancing│  │ • Query Optimization        │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  │                                                           │                                                                   │ │ │
│  │  │  ┌───────────────────────────────────────────────────────▼───────────────────────────────────────────────────────────────┐ │ │ │
│  │  │  │                                     Cluster Management                                                                 │ │ │ │
│  │  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │ │ │
│  │  │  │  │ Replication     │  │ Sharding        │  │ Backup &        │  │ Health          │  │    Auto-scaling             │ │ │ │ │
│  │  │  │  │ Management      │  │ Strategy        │  │ Recovery        │  │ Monitoring      │  │    & Load Balancing         │ │ │ │ │
│  │  │  │  │                 │  │                 │  │                 │  │                 │  │                             │ │ │ │ │
│  │  │  │  │ • Read Replicas │  │ • Hash-based    │  │ • Full Backup   │  │ • Node Health   │  │ • HPA Integration           │ │ │ │ │
│  │  │  │  │ • Write Master  │  │ • Range-based   │  │ • Incremental   │  │ • Cluster State │  │ • Resource Monitoring       │ │ │ │ │
│  │  │  │  │ • Consistency   │  │ • Content-based │  │ • Point-in-time │  │ • Performance   │  │ • Traffic Distribution      │ │ │ │ │
│  │  │  │  │ • Failover      │  │ • Load Balance  │  │ • Cross-region  │  │ • Alert System │  │ • Capacity Planning         │ │ │ │ │
│  │  │  │  │ • Split Brain   │  │ • Rebalancing   │  │ • Disaster Rec  │  │ • Metrics Exp   │  │ • Cost Optimization         │ │ │ │ │
│  │  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │ │ │
│  │  │  └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Data Flow Diagrams

### Document Processing Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    Document Processing Data Flow                                                   │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                     │
│  Input Sources                   Processing Pipeline                     Storage & Indexing                       │
│  ┌─────────────┐                                                                                                   │
│  │ 3GPP TS     │────┐                                                                                              │
│  │ Documents   │    │            ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────────────────┐ │
│  └─────────────┘    │            │ Document        │     │ Content         │     │ Metadata Extraction         │ │
│                     ├───────────▶│ Loader          │────▶│ Preprocessing   │────▶│ Engine                      │ │
│  ┌─────────────┐    │            │                 │     │                 │     │                             │ │
│  │ O-RAN WG    │────┤            │ • PDF Parser    │     │ • Text Cleaning │     │ • Standards Recognition     │ │
│  │ Specs       │    │            │ • Format Detect │     │ • Language Det  │     │ • Network Function Extract  │ │
│  └─────────────┘    │            │ • Batch Process │     │ • Structure Ana │     │ • Technical Classification  │ │
│                     │            │ • Error Handle  │     │ • Quality Check │     │ • Keyword Extraction        │ │
│  ┌─────────────┐    │            └─────────────────┘     └─────────────────┘     └─────────────────────────────┘ │
│  │ ETSI        │────┤                      │                      │                              │                │
│  │ Standards   │    │                      ▼                      ▼                              ▼                │
│  └─────────────┘    │            ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────────────────┐ │
│                     │            │ Content         │     │ Intelligent     │     │ Quality Assessment          │ │
│  ┌─────────────┐    │            │ Validation      │     │ Chunking        │     │ & Scoring                   │ │
│  │ ITU-T Rec   │────┤            │                 │     │ Engine          │     │                             │ │
│  │ Documents   │    │            │ • Length Check  │     │                 │     │ • Content Quality Score     │ │
│  └─────────────┘    │            │ • Format Valid  │     │ • Hierarchy     │     │ • Technical Density         │ │
│                     │            │ • Encoding Fix  │     │ • Boundaries    │     │ • Information Completeness  │ │
│  ┌─────────────┐    │            │ • Deduplication │     │ • Context Pres  │     │ • Confidence Calculation    │ │
│  │ Local Files │────┘            │ • Error Correct │     │ • Adaptive Size │     │ • Metadata Validation       │ │
│  │ & URLs      │                 └─────────────────┘     └─────────────────┘     └─────────────────────────────┘ │
│  └─────────────┘                           │                      │                              │                │
│                                            ▼                      ▼                              ▼                │
│                                  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────────────────┐ │
│                                  │ Document        │     │ Chunk           │     │ Enhanced Chunk              │ │
│                                  │ Metadata        │     │ Collection      │     │ Metadata                    │ │
│                                  │                 │     │                 │     │                             │ │
│                                  │ • Source Info   │     │ • Text Content  │     │ • Parent Document ID        │ │
│                                  │ • Version       │     │ • Clean Content │     │ • Section Hierarchy         │ │
│                                  │ • Working Group │     │ • Raw Content   │     │ • Technical Terms           │ │
│                                  │ • Category      │     │ • Chunk Index   │     │ • Cross-references          │ │
│                                  │ • Keywords      │     │ • Position Info │     │ • Quality Metrics           │ │
│                                  │ • Technologies  │     │ • Overlap Data  │     │ • Processing Timestamp      │ │
│                                  └─────────────────┘     └─────────────────┘     └─────────────────────────────┘ │
│                                            │                      │                              │                │
│                                            ▼                      ▼                              ▼                │
│                                                     ┌─────────────────────────────────────────────┐              │
│                                                     │ Embedding Generation Engine                │              │
│                                                     │                                             │              │
│  ┌─────────────────────────────────────────────────│ • Batch Processing (100 chunks/batch)      │              │
│  │                                                 │ • Rate Limiting (RPM/TPM management)       │              │
│  │  ┌─────────────────┐     ┌─────────────────┐    │ • Multi-provider Support (OpenAI, Azure)   │              │
│  │  │ Content Hash    │     │ Cache Lookup    │    │ • Quality Validation (dimension check)     │              │
│  │  │ Generation      │     │ System          │    │ • Error Handling & Retry Logic             │              │
│  │  │                 │     │                 │    │ • Performance Optimization                 │              │
│  │  │ • SHA-256       │────▶│ • LRU Cache     │    │ • Token Management & Truncation            │              │
│  │  │ • Content-based │     │ • TTL Mgmt      │    │ • Progress Tracking & Monitoring           │              │
│  │  │ • Deduplication │     │ • Hit/Miss      │    └─────────────────────────────────────────────┘              │
│  │  │ • Version Track │     │ • Compression   │                              │                                    │
│  │  └─────────────────┘     └─────────────────┘                              ▼                                    │
│  │            │                      │                      ┌─────────────────────────────────────────────┐      │
│  │            ▼                      ▼                      │ Vector Embeddings                           │      │
│  │  ┌─────────────────┐     ┌─────────────────┐             │                                             │      │
│  │  │ Cache Miss      │     │ Cache Hit       │             │ • 3072-dimensional vectors (text-embed-3)  │      │
│  │  │ Processing      │     │ Return          │             │ • Normalized vectors (L2 normalization)    │      │
│  │  │                 │     │                 │             │ • Quality-validated embeddings             │      │
│  │  │ • API Call      │     │ • Instant       │             │ • Batch-generated for efficiency           │      │
│  │  │ • Token Count   │     │ • Return        │             │ • Cached for future reuse                  │      │
│  │  │ • Cost Track    │     │ • Update LRU    │             │ • Metadata-enriched vectors                │      │
│  │  │ • Store Cache   │     │ • Metrics       │             └─────────────────────────────────────────────┘      │
│  │  └─────────────────┘     └─────────────────┘                              │                                    │
│  └────────────────────────────────────────────────────────────────────────────┘                                    │
│                                                                               ▼                                    │
│                                         ┌─────────────────────────────────────────────────────────────────────┐ │
│                                         │ Weaviate Vector Database Storage                                   │ │
│                                         │                                                                     │ │
│                                         │  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────┐  │ │
│                                         │  │ Document        │     │ Vector Index    │     │ Metadata    │  │ │
│                                         │  │ Storage         │     │ (HNSW)          │     │ Storage     │  │ │
│                                         │  │                 │     │                 │     │             │  │ │
│                                         │  │ • TelecomDoc    │◄────│ • Similarity    │────▶│ • Schema    │  │ │
│                                         │  │ • IntentPattern │     │ • Search Opt    │     │ • Relations │  │ │
│                                         │  │ • NetworkFunc   │     │ • Index Tuning  │     │ • Cross-ref │  │ │
│                                         │  │ • Collections   │     │ • Performance   │     │ • Taxonomy  │  │ │
│                                         │  └─────────────────┘     └─────────────────┘     └─────────────┘  │ │
│                                         │                                │                                   │ │
│                                         │  ┌─────────────────┐          │          ┌─────────────────┐     │ │
│                                         │  │ Backup &        │          │          │ Health          │     │ │
│                                         │  │ Recovery        │          │          │ Monitoring      │     │ │
│                                         │  │                 │          │          │                 │     │ │
│                                         │  │ • Daily Backup  │◄─────────┼─────────▶│ • Cluster State │     │ │
│                                         │  │ • Point Recovery│          │          │ • Performance   │     │ │
│                                         │  │ • Cross-AZ Rep  │          │          │ • Alerts        │     │ │
│                                         │  │ • Disaster Rec  │          │          │ • Metrics       │     │ │
│                                         │  └─────────────────┘          │          └─────────────────┘     │ │
│                                         └───────────────────────────────────────────────────────────────────┘ │
│                                                                        │                                        │
│                                                                        ▼                                        │
│                                         ┌─────────────────────────────────────────────────────────────────────┐ │
│                                         │ Processing Status & Metrics Collection                             │ │
│                                         │                                                                     │ │
│                                         │ • Documents Processed: Counter, Rate, Success/Failure Metrics     │ │
│                                         │ • Chunks Generated: Count, Size Distribution, Quality Scores      │ │
│                                         │ • Embeddings Created: Batch Performance, API Usage, Cost Tracking │ │
│                                         │ • Storage Utilization: Vector Count, Index Size, Memory Usage     │ │
│                                         │ • Processing Time: End-to-end Latency, Component Breakdown        │ │
│                                         │ • Quality Metrics: Content Quality, Embedding Quality, Accuracy   │ │
│                                         │ • Error Tracking: Failed Documents, Retry Counts, Error Types     │ │
│                                         └─────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Query Processing Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      Query Processing Data Flow                                                   │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                     │
│  User Input                     Processing Pipeline                         Response Generation                    │
│                                                                                                                     │
│  ┌─────────────────┐                                                                                               │
│  │ Natural         │             ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │ Language        │────────────▶│                      Query Enhancement Engine                              │ │
│  │ Intent          │             │                                                                             │ │
│  │                 │             │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │
│  │ Examples:       │             │  │ Intent          │  │ Query           │  │ Technical                   │ │ │
│  │ • "Configure    │             │  │ Classification  │  │ Preprocessing   │  │ Enhancement                 │ │ │
│  │   AMF for 5G"   │             │  │                 │  │                 │  │                             │ │ │
│  │ • "Troubleshoot │             │  │ • Configuration │  │ • Spell Check   │  │ • Acronym Expansion         │ │ │
│  │   gNB handover" │             │  │ • Troubleshoot  │  │ • Normalize     │  │ • Synonym Addition          │ │ │
│  │ • "Optimize     │             │  │ • Optimization  │  │ • Tokenization  │  │ • Context Integration       │ │ │
│  │   URLLC slice"  │             │  │ • Monitoring    │  │ • Language Det  │  │ • Domain Specialization     │ │ │
│  └─────────────────┘             │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │
│            │                     │           │                      │                      │                     │ │
│            │                     │           ▼                      ▼                      ▼                     │ │
│            │                     │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│            │                     │  │                      Enhanced Query                                      │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ Original: "Configure AMF for 5G"                                         │ │ │
│            │                     │  │ Enhanced: "Configure AMF (Access and Mobility Management Function)      │ │ │
│            │                     │  │           for 5G standalone deployment network slice eMBB parameters    │ │ │
│            │                     │  │           registration authentication procedures"                        │ │ │
│            │                     │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│            │                     └─────────────────────────────────────────────────────────────────────────────┘ │
│            │                               │                                                                       │
│            ▼                               ▼                                                                       │
│  ┌─────────────────┐             ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │ User Context    │             │                        Hybrid Search Engine                                │ │
│  │ & Preferences   │────────────▶│                                                                             │ │
│  │                 │             │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │
│  │ • Intent Type   │             │  │ Vector          │  │ Keyword         │  │ Hybrid                      │ │ │
│  │ • Domain        │             │  │ Similarity      │  │ Matching        │  │ Combination                 │ │ │
│  │ • Tech Level    │             │  │ Search          │  │ (BM25)          │  │                             │ │ │
│  │ • Preferences   │             │  │                 │  │                 │  │ • Alpha Weighting (0.7)     │ │ │
│  │ • History       │             │  │ • Query Embed   │  │ • Term Match    │  │ • Score Fusion              │ │ │
│  │ • Sources       │             │  │ • Cosine Sim    │  │ • TF-IDF        │  │ • Rank Aggregation          │ │ │
│  └─────────────────┘             │  │ • Top-K (50)    │  │ • Phrase Match  │  │ • Result Deduplication      │ │ │
│            │                     │  │ • Threshold     │  │ • Fuzzy Match   │  │ • Score Normalization       │ │ │
│            │                     │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │
│            │                     │           │                      │                      │                     │ │
│            │                     │           ▼                      ▼                      ▼                     │ │
│            │                     │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│            │                     │  │                    Search Results (Top 20-50)                           │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ • Document chunks with similarity scores                                 │ │ │
│            │                     │  │ • Metadata (source, category, version, confidence)                      │ │ │
│            │                     │  │ • Technical terms and cross-references                                   │ │ │
│            │                     │  │ • Hierarchical context (parent sections)                                 │ │ │
│            │                     │  │ • Initial relevance ranking                                               │ │ │
│            │                     │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│            │                     └─────────────────────────────────────────────────────────────────────────────┘ │
│            │                               │                                                                       │
│            ▼                               ▼                                                                       │
│  ┌─────────────────┐             ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │ Cache           │             │                       Semantic Reranking Engine                            │ │
│  │ Management      │◄────────────│                                                                             │ │
│  │                 │             │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │
│  │ • Query Cache   │             │  │ Cross-encoder   │  │ Multi-factor    │  │ Authority &                 │ │ │
│  │ • Result Cache  │             │  │ Scoring         │  │ Relevance       │  │ Freshness                   │ │ │
│  │ • Context Cache │             │  │                 │  │ Assessment      │  │ Weighting                   │ │ │
│  │ • TTL Mgmt      │             │  │ • Query-Doc     │  │                 │  │                             │ │ │
│  │ • Hit/Miss      │             │  │ • Deep Learning │  │ • Semantic Sim  │  │ • Source Priority (3GPP>    │ │ │
│  │ • Compression   │             │  │ • Transformer   │  │ • Content Qual  │  │   O-RAN>ETSI>ITU)           │ │ │
│  └─────────────────┘             │  │ • Fine-tuned    │  │ • Technical Acc │  │ • Document Recency          │ │ │
│            ▲                     │  │ • Batch Process │  │ • Completeness  │  │ • Update Frequency          │ │ │
│            │                     │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │
│            │                     │           │                      │                      │                     │ │
│            │                     │           ▼                      ▼                      ▼                     │ │
│            │                     │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│            │                     │  │                    Reranked Results (Top 10-15)                        │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ • Improved relevance ordering                                             │ │ │
│            │                     │  │ • Combined semantic + lexical + authority scores                        │ │ │
│            │                     │  │ • Diversity filtering (remove duplicates)                                │ │ │
│            │                     │  │ • Quality threshold filtering                                             │ │ │
│            │                     │  │ • Context relevance optimization                                          │ │ │
│            │                     │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│            │                     └─────────────────────────────────────────────────────────────────────────────┘ │
│            │                               │                                                                       │
│            │                               ▼                                                                       │
│            │                     ┌─────────────────────────────────────────────────────────────────────────────┐ │
│            │                     │                       Context Assembly Engine                               │ │
│            │                     │                                                                             │ │
│            │                     │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │
│            │                     │  │ Strategy        │  │ Hierarchy       │  │ Source                      │ │ │
│            │                     │  │ Selection       │  │ Preservation    │  │ Balancing                   │ │ │
│            │                     │  │                 │  │                 │  │                             │ │ │
│            │                     │  │ • Rank-based    │  │ • Doc Structure │  │ • Multi-source              │ │ │
│            │                     │  │ • Topical       │  │ • Section Order │  │ • Proportional              │ │ │
│            │                     │  │ • Hierarchical  │  │ • Parent Context│  │ • Authority Weight          │ │ │
│            │                     │  │ • Progressive   │  │ • Cross-refs    │  │ • Coverage Balance          │ │ │
│            │                     │  │ • Quality-first │  │ • Relationships │  │ • Redundancy Removal        │ │ │
│            │                     │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │
│            │                     │           │                      │                      │                     │ │
│            │                     │           ▼                      ▼                      ▼                     │ │
│            │                     │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│            │                     │  │                    Token Management                                      │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ • Target Length: 4000-8000 tokens (configurable)                        │ │ │
│            │                     │  │ • Smart Truncation: Preserve high-value content                         │ │ │
│            │                     │  │ • Overlap Management: Ensure context continuity                          │ │ │
│            │                     │  │ • Quality Preservation: Maintain information density                     │ │ │
│            │                     │  │ • Structure Integrity: Keep section boundaries                           │ │ │
│            │                     │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│            │                     │                              │                                                │ │
│            │                     │                              ▼                                                │ │
│            │                     │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│            │                     │  │                 Assembled Context                                       │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ === 3GPP Technical Specifications ===                                    │ │ │
│            │                     │  │ TS 23.501 v17.9.0: 5G System Architecture                               │ │ │
│            │                     │  │ The Access and Mobility Management Function (AMF) provides...            │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ === O-RAN Alliance Specifications ===                                    │ │ │
│            │                     │  │ O-RAN.WG2.AAD v04.00: Architecture Description                          │ │ │
│            │                     │  │ The AMF integration with O-RAN components requires...                    │ │ │
│            │                     │  │                                                                           │ │ │
│            │                     │  │ === Configuration Examples ===                                           │ │ │
│            │                     │  │ YAML configuration for AMF deployment...                                 │ │ │
│            │                     │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│            │                     └─────────────────────────────────────────────────────────────────────────────┘ │
│            │                               │                                                                       │
│            └───────────────────────────────┼───────────────────────────────────────────────────────────────────┘ │
│                                            │                                                                       │
│                                            ▼                                                                       │
│                                  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│                                  │                     Response Assembly & Validation                          │ │
│                                  │                                                                             │ │
│                                  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │ │
│                                  │  │ JSON Schema     │  │ Quality         │  │ Metadata                    │ │ │
│                                  │  │ Validation      │  │ Metrics         │  │ Enhancement                 │ │ │
│                                  │  │                 │  │ Calculation     │  │                             │ │ │
│                                  │  │ • Response      │  │                 │  │ • Processing Time           │ │ │
│                                  │  │   Structure     │  │ • Relevance Avg │  │ • Component Latency         │ │ │
│                                  │  │ • Field Types   │  │ • Coverage Score│  │ • Quality Scores            │ │ │
│                                  │  │ • Required      │  │ • Diversity Sc  │  │ • Source Distribution       │ │ │
│                                  │  │ • Constraints   │  │ • Confidence    │  │ • Enhancement Details       │ │ │
│                                  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │ │
│                                  │           │                      │                      │                     │ │
│                                  │           ▼                      ▼                      ▼                     │ │
│                                  │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│                                  │  │                    Enhanced Search Response                             │ │ │
│                                  │  │                                                                           │ │ │
│                                  │  │ {                                                                         │ │ │
│                                  │  │   "query": "Configure AMF for 5G",                                       │ │ │
│                                  │  │   "processed_query": "Configure AMF (Access and Mobility...",            │ │ │
│                                  │  │   "results": [...],                                                       │ │ │
│                                  │  │   "assembled_context": "=== 3GPP Technical...",                          │ │ │
│                                  │  │   "processing_time": "1.847s",                                            │ │ │
│                                  │  │   "average_relevance_score": 0.87,                                       │ │ │
│                                  │  │   "coverage_score": 0.94,                                                 │ │ │
│                                  │  │   "context_metadata": {...},                                              │ │ │
│                                  │  │   "query_enhancements": {...}                                             │ │ │
│                                  │  │ }                                                                         │ │ │
│                                  │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│                                  └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                            │                                                       │
│                                                            ▼                                                       │
│                                  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│                                  │                     Metrics & Analytics Collection                          │ │
│                                  │                                                                             │ │
│                                  │ • Query Performance: Latency, Throughput, Success Rate                     │ │
│                                  │ • Enhancement Effectiveness: Improvement in Relevance Scores               │ │
│                                  │ • Search Quality: Coverage, Diversity, User Satisfaction                   │ │
│                                  │ • Resource Utilization: Cache Hit Rates, API Usage, Costs                 │ │
│                                  │ • System Health: Component Status, Error Rates, Alerts                     │ │
│                                  └─────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

This comprehensive architecture and data flow documentation provides detailed insights into the Nephoran RAG pipeline's structure, component relationships, and processing workflows. The diagrams illustrate how natural language intents are transformed into structured network function deployments through sophisticated document processing, vector search, semantic reranking, and context assembly mechanisms.