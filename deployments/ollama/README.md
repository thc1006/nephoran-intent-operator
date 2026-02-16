# Ollama Deployment for Nephoran Intent Operator

**Deployed**: 2026-02-16
**Version**: v0.16.1
**Purpose**: LLM inference engine for NetworkIntent processing

## Deployment Info

- **Namespace**: `ollama`
- **Helm Chart**: `otwld/ollama` v1.43.0
- **Resources**: 2 CPU / 8GB RAM (requests), 4 CPU / 16GB RAM (limits)
- **GPU**: NVIDIA GPU via device plugin
- **Storage**: 50Gi PVC (local-path)
- **Models**: llama3.2:3b (primary)

## Installation

```bash
# Add Helm repo
helm repo add otwld https://helm.otwld.com/
helm repo update

# Install Ollama
helm install ollama otwld/ollama \
  --namespace ollama \
  --create-namespace \
  --values values.yaml

# Verify deployment
kubectl get all -n ollama
kubectl logs -n ollama deployment/ollama -f
```

## Testing

```bash
# Port-forward to test API
kubectl port-forward -n ollama svc/ollama 11434:11434

# Test API (in another terminal)
curl http://localhost:11434/

# Pull model
curl -X POST http://localhost:11434/api/pull -d '{"name": "llama3.2:3b"}'

# List models
kubectl exec -n ollama deployment/ollama -- ollama list

# Generate test
curl http://localhost:11434/api/generate -d '{
  "model": "llama3.2:3b",
  "prompt": "Why is the sky blue?"
}'
```

## Integration with RAG Service

Ollama is integrated with the RAG service for NetworkIntent processing:

1. **Service URL**: `http://ollama.ollama.svc.cluster.local:11434`
2. **RAG Config**: Update `rag-service` to point to Ollama endpoint
3. **Model**: llama3.2:3b for embeddings and generation

## Troubleshooting

```bash
# Check pod status
kubectl describe pod -n ollama -l app.kubernetes.io/name=ollama

# Check logs
kubectl logs -n ollama deployment/ollama --tail=100

# Check PVC
kubectl get pvc -n ollama

# Restart deployment
kubectl rollout restart deployment/ollama -n ollama
```

## Upgrade

```bash
# Upgrade to newer version
helm upgrade ollama otwld/ollama \
  --namespace ollama \
  --values values.yaml
```

## Uninstall

```bash
helm uninstall ollama -n ollama
kubectl delete namespace ollama
```

## References

- [Ollama Official](https://github.com/ollama/ollama)
- [Ollama Helm Chart](https://github.com/otwld/ollama-helm)
- [Ollama Kubernetes Guide](https://collabnix.com/running-ollama-on-kubernetes/)
