{
  "name": "Telecom LLM Automation Environment",
  "build": {
    "dockerfile": "Dockerfile",
    "context": ".."
  },
  "features": {
    "ghcr.io/devcontainers/features/go:1": {"version": "1.21"},
    "ghcr.io/devcontainers/features/python:1": {"version": "3.11"},
    "ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
      "version": "latest",
      "helm": "3.12.0",
      "kubectl": "1.27.3"
    }
  },
  "customizations": {
    "vscode": {
      "extensions": [
        "ms-vscode.Go",
        "ms-python.python",
        "ms-kubernetes-tools.vscode-kubernetes-tools",
        "redhat.vscode-yaml"
      ]
    }
  },
  "postCreateCommand": "make setup-dev"
}
