# Nephoran Intent Operator - Neural Command Interface

**Version**: 1.0.0
**Last Updated**: 2026-02-23

---

## 📋 **Overview**

The **Nephoran Intent Operator Neural Command Interface** is a production-grade web UI for submitting natural language intents to orchestrate 5G network functions in Kubernetes. The interface features a cybernetic design inspired by Kubernetes Dashboard and Ollama WebUI.

### **Architecture**

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────┐
│   Web Browser   │──────│  Frontend (Nginx) │──────│  Intent Ingest  │
│  (User Input)   │ HTTP │   (Port 80)      │ API  │   (Port 8080)   │
└─────────────────┘      └──────────────────┘      └─────────────────┘
                                                            │
                                                            ├──────► Ollama (LLM)
                                                            ├──────► RAG Service
                                                            └──────► Weaviate (Vector DB)
```

### **Key Features**

- 🎨 **Distinctive UI**: Cybernetic command center aesthetic with animated effects
- 🧠 **Natural Language Processing**: Convert human intent to structured NetworkIntent CRDs
- 📊 **Real-time Feedback**: Instant validation and response display
- 📜 **Intent History**: LocalStorage-based history with quick replay
- 🔒 **Security**: NetworkPolicies, non-root containers, resource limits
- 📱 **Responsive**: Works on desktop and mobile devices

---

## 🚀 **Quick Start**

### **Prerequisites**

- Kubernetes cluster v1.35.1+ (tested on single-node cluster)
- `kubectl` configured with cluster access
- **Optional**: Docker for building custom image
- **Required Services**:
  - Ollama deployed in namespace `ollama`
  - RAG service deployed in namespace `rag-service`

### **1. Deploy the Complete System**

```bash
cd deployments/frontend
./deploy.sh
```

This will:
- ✅ Create namespace `nephoran-intent`
- ✅ Deploy frontend (Nginx + HTML)
- ✅ Deploy backend (intent-ingest service)
- ✅ Create ConfigMaps for HTML and schema
- ✅ Apply NetworkPolicies
- ✅ Setup port-forwarding to http://localhost:8888

### **2. Access the Interface**

Open your browser to:
```
http://localhost:8888
```

### **3. Submit Your First Intent**

**Example 1: Scaling**
```
scale nf-sim to 5 in ns ran-a
```

**Example 2: Deployment**
```
deploy nginx with 3 replicas in namespace production
```

**Example 3: Service**
```
create service for app frontend on port 8080
```

---

## 🛠️ **Manual Deployment**

### **Step 1: Build Docker Image (Optional)**

If you want to build the intent-ingest image locally:

```bash
cd /home/thc1006/dev/nephoran-intent-operator
docker build -f deployments/frontend/Dockerfile.intent-ingest -t nephoran/intent-ingest:latest .
```

### **Step 2: Deploy Kubernetes Resources**

```bash
# Create namespace
kubectl create namespace nephoran-intent

# Create frontend ConfigMap
kubectl create configmap nephoran-frontend-html \
  --from-file=index.html=deployments/frontend/index.html \
  --namespace=nephoran-intent

# Create schema ConfigMap (if exists)
kubectl create configmap intent-schema \
  --from-file=intent.schema.json=docs/contracts/intent.schema.json \
  --namespace=nephoran-intent

# Deploy all resources
kubectl apply -f deployments/frontend/k8s-manifests.yaml
```

### **Step 3: Verify Deployment**

```bash
# Check pods
kubectl get pods -n nephoran-intent

# Expected output:
# NAME                                READY   STATUS    RESTARTS   AGE
# intent-ingest-xxxxxxxxx-xxxxx       1/1     Running   0          1m
# intent-ingest-xxxxxxxxx-xxxxx       1/1     Running   0          1m
# intent-ingest-xxxxxxxxx-xxxxx       1/1     Running   0          1m
# nephoran-frontend-xxxxxxxxx-xxxxx   1/1     Running   0          1m
# nephoran-frontend-xxxxxxxxx-xxxxx   1/1     Running   0          1m

# Check services
kubectl get svc -n nephoran-intent

# Expected output:
# NAME                   TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
# intent-ingest-service  ClusterIP   10.96.xxx.xxx   <none>        8080/TCP   1m
# nephoran-frontend      ClusterIP   10.96.xxx.xxx   <none>        80/TCP     1m
```

### **Step 4: Access the Frontend**

**Option 1: Port Forward (Recommended for local development)**
```bash
kubectl port-forward -n nephoran-intent svc/nephoran-frontend 8888:80
```
Then open: http://localhost:8888

**Option 2: Ingress (For production)**
```bash
# Edit /etc/hosts
echo "127.0.0.1 nephoran.local" | sudo tee -a /etc/hosts

# Access via
http://nephoran.local
```

---

## 📖 **Usage Guide**

### **Interface Components**

1. **Top Navigation Bar**
   - Nephoran logo with hexagonal animation
   - System status indicators
   - Cluster version display

2. **Left Sidebar**
   - Quick Actions: Pre-configured intent templates
   - Intent Types: Filter by scaling, deployment, service

3. **Main Content Area**
   - Intent Type selector
   - Namespace input
   - Large text area for natural language commands
   - Example intents (click to populate)
   - Submit and Clear buttons

4. **Right Panel (Logs)**
   - Intent history with timestamps
   - Click history item to replay intent
   - Auto-saved to browser LocalStorage

5. **Bottom Status Bar**
   - API endpoint information
   - Current timestamp
   - System mode (LLM)

### **Supported Intent Types**

#### **1. Scaling Intents**
```
scale <target> to <replicas> in ns <namespace>
```
Example:
```
scale nf-sim to 5 in ns ran-a
```

#### **2. Deployment Intents**
```
deploy <app> with <replicas> replicas in namespace <namespace>
```
Example:
```
deploy nginx with 3 replicas in namespace production
```

#### **3. Service Intents**
```
create service for app <name> on port <port>
```
Example:
```
create service for app frontend on port 8080
```

### **Keyboard Shortcuts**

- **Ctrl + Enter**: Submit intent
- **Ctrl + L**: Clear input

---

## 🔧 **Configuration**

### **Environment Variables (Backend)**

Configure the `intent-ingest` deployment:

```yaml
env:
- name: MODE
  value: "llm"  # or "rules"
- name: PROVIDER
  value: "ollama"  # LLM provider
- name: OLLAMA_ENDPOINT
  value: "http://ollama-service.ollama.svc.cluster.local:11434"
- name: RAG_ENDPOINT
  value: "http://rag-service.rag-service.svc.cluster.local:8000"
- name: HANDOFF_DIR
  value: "/var/nephoran/handoff"
```

### **Frontend Configuration**

Edit `deployments/frontend/index.html` to change:

```javascript
const CONFIG = {
    apiEndpoint: window.location.hostname === 'localhost'
        ? 'http://localhost:8080'
        : 'http://intent-ingest-service:8080',
    historyKey: 'nephoran_intent_history',
    maxHistoryItems: 50
};
```

### **Nginx Configuration**

Modify nginx settings in ConfigMap:

```bash
kubectl edit configmap nginx-config -n nephoran-intent
```

---

## 🔒 **Security Features**

### **Network Policies**

- Frontend can only communicate with backend API
- Backend can only communicate with Ollama and RAG services
- DNS resolution allowed for service discovery

### **Container Security**

- Non-root user (UID 1000 for backend, 101 for frontend)
- Read-only root filesystem (where possible)
- No privilege escalation
- Capabilities dropped
- Seccomp profile applied

### **Resource Limits**

**Frontend (Nginx):**
- Requests: 100m CPU, 64Mi Memory
- Limits: 200m CPU, 128Mi Memory

**Backend (Intent Ingest):**
- Requests: 250m CPU, 256Mi Memory
- Limits: 500m CPU, 512Mi Memory

---

## 📊 **Monitoring & Debugging**

### **View Logs**

**Frontend logs:**
```bash
kubectl logs -n nephoran-intent -l app=nephoran-frontend -f
```

**Backend logs:**
```bash
kubectl logs -n nephoran-intent -l app=intent-ingest -f
```

### **Debug API Calls**

Test the backend directly:

```bash
# Port-forward backend
kubectl port-forward -n nephoran-intent svc/intent-ingest-service 8080:8080

# Test with curl
curl -X POST http://localhost:8080/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 5 in ns ran-a"
```

### **Health Checks**

```bash
# Frontend health
kubectl exec -n nephoran-intent deployment/nephoran-frontend -- wget -qO- http://localhost/healthz

# Backend health
kubectl exec -n nephoran-intent deployment/intent-ingest -- wget -qO- http://localhost:8080/healthz
```

---

## 🔄 **Upgrade & Maintenance**

### **Update Frontend HTML**

```bash
kubectl create configmap nephoran-frontend-html \
  --from-file=index.html=deployments/frontend/index.html \
  --namespace=nephoran-intent \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/nephoran-frontend -n nephoran-intent
```

### **Update Backend Image**

```bash
# Build new image
docker build -f deployments/frontend/Dockerfile.intent-ingest \
  -t nephoran/intent-ingest:v1.1.0 .

# Update deployment
kubectl set image deployment/intent-ingest \
  intent-ingest=nephoran/intent-ingest:v1.1.0 \
  -n nephoran-intent
```

### **Scale Deployments**

```bash
# Scale frontend
kubectl scale deployment/nephoran-frontend --replicas=3 -n nephoran-intent

# Scale backend
kubectl scale deployment/intent-ingest --replicas=5 -n nephoran-intent
```

---

## 🗑️ **Uninstall**

### **Complete Removal**

```bash
kubectl delete -f deployments/frontend/k8s-manifests.yaml
kubectl delete namespace nephoran-intent
```

### **Keep Namespace, Remove Deployments**

```bash
kubectl delete deployment,svc,configmap,ingress,networkpolicy \
  -n nephoran-intent --all
```

---

## 🐛 **Troubleshooting**

### **Issue: Frontend shows "Network error"**

**Cause**: Cannot connect to backend API

**Solutions**:
1. Verify backend is running:
   ```bash
   kubectl get pods -n nephoran-intent -l app=intent-ingest
   ```

2. Check service:
   ```bash
   kubectl get svc -n nephoran-intent intent-ingest-service
   ```

3. Test connectivity:
   ```bash
   kubectl exec -n nephoran-intent deployment/nephoran-frontend -- \
     wget -qO- http://intent-ingest-service:8080/healthz
   ```

### **Issue: Backend returns "validation failed"**

**Cause**: Intent schema validation error

**Solutions**:
1. Check schema ConfigMap exists:
   ```bash
   kubectl get configmap intent-schema -n nephoran-intent
   ```

2. Verify schema content:
   ```bash
   kubectl get configmap intent-schema -n nephoran-intent -o yaml
   ```

3. Use correct intent format:
   ```
   scale <target> to <number> in ns <namespace>
   ```

### **Issue: History not saving**

**Cause**: Browser LocalStorage disabled or full

**Solutions**:
1. Enable LocalStorage in browser settings
2. Clear browser cache
3. Try incognito/private mode

---

## 📚 **API Documentation**

### **POST /intent**

Submit a natural language intent for processing.

**Request:**
```bash
curl -X POST http://intent-ingest-service:8080/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 5 in ns ran-a"
```

**Response (Success):**
```json
{
  "intent_type": "scaling",
  "target": "nf-sim",
  "namespace": "ran-a",
  "replicas": 5,
  "source": "user",
  "status": "pending"
}
```

**Response (Error):**
```json
{
  "error": "Invalid plain text format. Expected: 'scale <target> to <replicas> in ns <namespace>'"
}
```

### **GET /healthz**

Health check endpoint.

**Response:**
```
ok
```

---

## 🎨 **Design Philosophy**

The **Neural Command Interface** uses a **Tactical Cybernetic Command Center** aesthetic:

- **Typography**: IBM Plex Mono (body) + Orbitron (headers)
- **Colors**: Deep blues (#0a0e1a) with electric cyan (#00ffff) accents
- **Effects**: Scanning lines, hexagonal grid, pulsing animations
- **Motion**: Smooth transitions, staggered reveals, hover states
- **Layout**: Asymmetric with tactical information density

**Design Goals**:
- ✅ Professional and production-ready
- ✅ Visually distinctive (not generic "AI slop")
- ✅ Accessible and usable
- ✅ Memorable and engaging

---

## 🤝 **Contributing**

To modify the frontend:

1. Edit `deployments/frontend/index.html`
2. Test locally by opening in browser
3. Deploy to cluster with `./deploy.sh`

To modify the backend:

1. Edit `cmd/intent-ingest/main.go` or `internal/ingest/handler.go`
2. Rebuild Docker image
3. Update deployment

---

## 📄 **License**

Apache License 2.0

---

## 📞 **Support**

- **Documentation**: See `docs/` directory
- **Issues**: File at project GitHub repository
- **Logs**: `kubectl logs -n nephoran-intent -l app=intent-ingest`

---

**Created**: 2026-02-23
**Last Updated**: 2026-02-23
**Version**: 1.0.0
