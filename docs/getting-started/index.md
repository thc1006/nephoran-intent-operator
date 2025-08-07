# Getting Started

Welcome to the Nephoran Intent Operator! This section will help you get up and running quickly with intent-driven network automation.

## Overview

The Nephoran Intent Operator transforms how you manage telecommunications networks by using natural language intents to automate complex network function deployments. Whether you're deploying 5G core networks, managing O-RAN functions, or orchestrating network slices, the system handles the complexity while you focus on your business requirements.

## What You'll Learn

In this getting started guide, you'll learn how to:

1. **[Quick Start](quickstart.md)** - Get the operator running in minutes
2. **[Installation](installation.md)** - Detailed installation instructions for different environments
3. **[Configuration](configuration.md)** - Configure the operator for your environment
4. **[First Intent](first-intent.md)** - Create and deploy your first network intent

## Prerequisites

Before you begin, ensure you have:

- Kubernetes cluster (1.24+) or Docker Desktop
- kubectl configured to access your cluster
- Helm 3.8+ (for Helm-based installation)
- Basic understanding of Kubernetes concepts

## Quick Preview

Here's what a simple network intent looks like:

```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: example-5g-deployment
spec:
  intent: "Deploy a basic 5G core network with AMF and SMF for testing"
  priority: medium
  context:
    environment: development
```

The operator will automatically:

1. Parse your natural language intent
2. Generate appropriate network function configurations
3. Create Kubernetes resources
4. Deploy and monitor the network functions
5. Provide status updates and health monitoring

## Choose Your Path

=== "I want to try it quickly"
    
    Jump straight to our [Quick Start](quickstart.md) guide for a 5-minute setup.

=== "I need production deployment"
    
    Follow our comprehensive [Installation](installation.md) guide with security and scalability considerations.

=== "I'm exploring the technology"
    
    Start with [Architecture Overview](../architecture/system-architecture.md) to understand how it works.

## Support

If you run into any issues during setup:

- Check our [Troubleshooting Guide](../user-guide/troubleshooting.md)
- Browse [Frequently Asked Questions](../reference/faq.md)
- Join our [Community Slack](https://nephoran.slack.com)
- [Report issues on GitHub](https://github.com/nephoran/nephoran-intent-operator/issues)

Ready to get started? Let's begin with the [Quick Start](quickstart.md)!