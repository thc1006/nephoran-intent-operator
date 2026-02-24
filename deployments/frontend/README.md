# Nephoran Intent Operator - Web Frontend

Production-grade web UI for the Nephoran Intent Operator natural language input interface.

## Features

- **Dark Theme with Glassmorphism**: Modern, professional UI inspired by Kubernetes Dashboard and Ollama WebUI
- **Natural Language Processing**: Submit intents in plain English
- **Real-time Validation**: Instant feedback on intent processing
- **Intent History**: LocalStorage-based history of recent intents
- **Quick Examples**: Pre-filled example intents for common operations
- **Responsive Design**: Works on desktop and mobile devices

## Architecture

```
User Browser → Nginx (NodePort 30080) → ConfigMap (index.html) → API Proxy → intent-ingest-service:8080
```

### Components

1. **Frontend HTML** (`index.html`): Single-file application with embedded CSS and JavaScript
2. **Nginx Deployment**: 2 replicas for high availability
3. **ConfigMap**: Stores HTML content
4. **Service**: NodePort on port 30080
5. **NetworkPolicy**: Restricts traffic to DNS and intent-ingest service

## Deployment

### Prerequisites

- Kubernetes cluster 1.35.1+
- `intent-ingest-service` deployed in namespace `intent-ingest`

### Deploy

```bash
# Deploy frontend
kubectl apply -f deployments/frontend/frontend-deployment.yaml

# Verify deployment
kubectl get all -n nephoran-frontend

# Check logs
kubectl logs -n nephoran-frontend deployment/nephoran-frontend -f
```

### Access

```bash
# Get node IP
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')

# Access frontend
echo "Frontend available at: http://$NODE_IP:30080"

# Or use port-forward for local development
kubectl port-forward -n nephoran-frontend svc/nephoran-frontend 8080:80
# Then access at http://localhost:8080
```

## Usage

### Example Intents

The UI supports natural language intents like:

1. **Scaling**: `scale nf-sim to 5 in ns ran-a`
2. **Deployment**: `deploy nginx with 3 replicas in namespace production`
3. **Service**: `create service for app frontend on port 8080`
4. **5G Network Functions**: `scale free5gc-amf to 2 replicas in namespace free5gc`

### Keyboard Shortcuts

- **Ctrl+Enter**: Submit intent
- **Tab**: Navigate between fields
- **Click example tags**: Auto-fill intent input

### API Integration

The frontend calls the intent-ingest API:

```
POST http://intent-ingest-service:8080/intent
Content-Type: text/plain

scale nf-sim to 5 in ns ran-a
```

Response:
```json
{
  "apiVersion": "intent.nephoran.com/v1alpha1",
  "kind": "NetworkIntent",
  "metadata": {
    "name": "scale-nf-sim-ran-a",
    "namespace": "ran-a"
  },
  "spec": {
    "intent_type": "scaling",
    "target": "nf-sim",
    "replicas": 5
  }
}
```

## Design Philosophy

### Aesthetic Direction

- **Brutally Minimal Futurism**: Clean lines, generous whitespace, purposeful animations
- **Kubernetes-Native**: Professional color scheme (blues, purples) matching K8s ecosystem
- **Glassmorphism**: Backdrop filters and layered transparencies for depth
- **Monospace Typography**: 
  - **JetBrains Mono**: Display text, headings, code
  - **Azeret Mono**: Body text, inputs

### Visual Details

1. **Animated Gradient Background**: Subtle radial gradients that shift over 20 seconds
2. **Grid Overlay**: Low-opacity grid for technical aesthetic
3. **Glowing Accents**: Pulsing status indicators
4. **Smooth Transitions**: All interactions have 0.3s ease transitions
5. **Loading States**: Spinning border animation during API calls

### Color Palette

```css
--bg-primary: #0a0e1a     /* Deep dark blue */
--accent-blue: #3b82f6     /* Kubernetes blue */
--accent-purple: #8b5cf6   /* Secondary accent */
--accent-cyan: #06b6d4     /* Code/JSON highlight */
--success: #10b981         /* Success green */
--error: #ef4444           /* Error red */
```

## Configuration

### Update API Endpoint

Edit the ConfigMap or modify `index.html`:

```javascript
const API_ENDPOINT = 'http://intent-ingest-service:8080/intent';
```

### Adjust Resources

Edit `frontend-deployment.yaml`:

```yaml
resources:
  requests:
    cpu: 50m    # Increase for more traffic
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 128Mi
```

### Change NodePort

```yaml
ports:
- name: http
  port: 80
  targetPort: 80
  nodePort: 30080  # Change this (30000-32767)
```

## Security

### Implemented

- ✅ NetworkPolicy restricts egress to intent-ingest and DNS only
- ✅ Security headers (X-Frame-Options, X-Content-Type-Options, X-XSS-Protection)
- ✅ Read-only volume mounts
- ✅ Non-root nginx container
- ✅ Resource limits to prevent DoS

### Recommendations for Production

1. **Enable TLS**: Use Ingress with cert-manager for HTTPS
2. **Add Authentication**: Integrate with OAuth2 or OIDC
3. **Rate Limiting**: Add nginx rate limiting for API endpoints
4. **WAF**: Deploy Web Application Firewall (ModSecurity)
5. **CSP Headers**: Add Content Security Policy headers

## Troubleshooting

### Frontend not accessible

```bash
# Check pods
kubectl get pods -n nephoran-frontend

# Check service
kubectl get svc -n nephoran-frontend

# Check logs
kubectl logs -n nephoran-frontend deployment/nephoran-frontend

# Verify NetworkPolicy
kubectl describe networkpolicy -n nephoran-frontend
```

### API calls failing

```bash
# Check intent-ingest service exists
kubectl get svc -n intent-ingest intent-ingest-service

# Test from frontend pod
kubectl exec -n nephoran-frontend deployment/nephoran-frontend -- wget -O- http://intent-ingest-service.intent-ingest.svc.cluster.local:8080/health

# Check DNS
kubectl exec -n nephoran-frontend deployment/nephoran-frontend -- nslookup intent-ingest-service.intent-ingest.svc.cluster.local
```

### History not saving

LocalStorage is browser-specific. Check:
- Browser console for errors (F12 → Console)
- LocalStorage size limits (usually 5-10 MB)
- Clear cache if corrupted: `localStorage.clear()`

## Development

### Local Development

```bash
# Open index.html directly in browser
google-chrome deployments/frontend/index.html

# Or serve with Python
cd deployments/frontend
python3 -m http.server 8000
# Access at http://localhost:8000
```

### Modify Frontend

1. Edit `index.html` directly
2. Test locally
3. Update ConfigMap:

```bash
kubectl create configmap frontend-html \
  --from-file=index.html=deployments/frontend/index.html \
  --namespace=nephoran-frontend \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods to pick up changes
kubectl rollout restart deployment/nephoran-frontend -n nephoran-frontend
```

## Performance

### Metrics

- **Initial Load**: < 500ms (with CDN fonts)
- **Time to Interactive**: < 1s
- **Bundle Size**: ~45 KB (single HTML file)
- **API Response**: Depends on LLM processing (1-3s typical)

### Optimization

- Gzip compression enabled (nginx)
- Static asset caching (1 year for fonts)
- Minimal JavaScript (vanilla, no frameworks)
- CSS animations use GPU acceleration

## Roadmap

Future enhancements:

- [ ] Real-time status updates via WebSocket
- [ ] Syntax highlighting for JSON responses (highlight.js)
- [ ] Intent validation preview before submit
- [ ] Export intent history as JSON
- [ ] Dark/Light theme toggle
- [ ] Multi-language support (i18n)
- [ ] Accessibility improvements (WCAG 2.1 AA)

---

**Created**: 2026-02-24  
**Version**: 1.0.0  
**License**: Apache 2.0
