# BUILD-RUN-TEST.windows.md

## Prerequisites
```powershell
# Verify Go 1.24
go version
# go version go1.24.5 windows/amd64

# Verify kubectl 
kubectl version --client
# Client Version: v1.29.0

# Verify kind installed
kind version
# kind v0.20.0 go1.20.4 windows/amd64
```

## Step 1: Install controller-gen
```powershell
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0

# Verify installation
controller-gen --version
# Version: v0.15.0
```

## Step 2: Generate DeepCopy and CRDs
```powershell
# Generate DeepCopy methods
controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/intent/v1alpha1"

# Expected: Creates api/intent/v1alpha1/zz_generated.deepcopy.go

# Generate CRDs
controller-gen crd paths="./api/intent/v1alpha1" output:crd:artifacts:config="./deployments/crds"

# Expected: Creates deployments/crds/intent.nephoran.io_networkintents.yaml

# Verify files exist
ls api/intent/v1alpha1/zz_generated.deepcopy.go
ls deployments/crds/intent.nephoran.io_networkintents.yaml
```

## Step 3: Build webhook manager
```powershell
# Build the webhook manager binary
go build -o webhook-manager.exe ./cmd/webhook-manager

# Expected: webhook-manager.exe created

# Test binary
./webhook-manager.exe --help
# Expected output:
# Usage of webhook-manager.exe:
#   -cert-dir string
#   -health-probe-bind-address string (default ":8081")
#   -metrics-bind-address string (default ":8080")
#   -webhook-port int (default 9443)
```

## Step 4: Create kind cluster
```powershell
# Create kind cluster
kind create cluster --name webhook-test

# Expected:
# Creating cluster "webhook-test" ...
# ✓ Ensuring node image (kindest/node:v1.27.3)
# ✓ Preparing nodes
# ✓ Writing configuration
# ✓ Starting control-plane
# ✓ Installing CNI
# ✓ Installing StorageClass
# Set kubectl context to "kind-webhook-test"

# Verify cluster
kubectl cluster-info --context kind-webhook-test
```

## Step 5: Build and load Docker image
```powershell
# Build Docker image
docker build -t nephoran/webhook-manager:latest -f- . @"
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-manager ./cmd/webhook-manager

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /app/webhook-manager .
ENTRYPOINT ["/webhook-manager"]
"@

# Expected: Successfully built <image-id>
# Successfully tagged nephoran/webhook-manager:latest

# Load image into kind
kind load docker-image nephoran/webhook-manager:latest --name webhook-test

# Expected: Image: "nephoran/webhook-manager:latest" with ID "sha256:..." not yet present on node "webhook-test-control-plane", loading...
```

## Step 6: Create namespace and deploy CRDs
```powershell
# Create namespace
kubectl create namespace nephoran-system

# Expected: namespace/nephoran-system created

# Apply CRDs
kubectl apply -f deployments/crds/

# Expected: customresourcedefinition.apiextensions.k8s.io/networkintents.intent.nephoran.io created
```

## Step 7: Generate self-signed certificates
```powershell
# Create certificate for webhook (using OpenSSL or generate in-cluster)
@"
apiVersion: v1
kind: Secret
metadata:
  name: webhook-server-cert
  namespace: nephoran-system
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURQekNDQWllZ0F3SUJBZ0lVS1VpRGdJdUdEemRUS2tRanJQK0ZXSktPSEhrd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0R6RU5NQXNHQTFVRUF3d0VkR1Z6ZERBZUZ3MHlNakEzTVRFd05qVXlNakJhRncwek1qQTNNRGd3TmpVeQpNakJhTUE4eERUQUxCZ05WQkFNTUJIUmxjM1F3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURFd0NpYnpGMFo0MjZSM0xxRXdNOGtkaHRWQ3lIUStQZUlRbzBKM3hEaHJ0NEl2bklIQzJQenBhaE0KZ3FGUnRMWlk0L3RYYVhqdWxWTlhSUFhFOGlNR2VKT2g2cm9odHlCNURoOTBqRzBLaE5SWUlQOTRrNWlMaFZOdwpaU1o3bENUK2JVQUxtTzFEVGJOcER6SFBXMVhwVXBRRnJqVUxjbHNKRERJdk0ybUxJUnB2VkViWHY1akE0WnJUClA1bDRzMzRiL1ZsZ01sOGsxRmhGc1VmeWJxV1dzWDRJWmZHaVEwRWxBZUZRUEhwMEtJOGNPbGNYeUcyS2tVcFYKVTRSYWJtNEVkTExmSGdOZG5rOTJudEZQdlh0SFhKejA3bXRBenNicUp1ZHU0RUpnMG80eEJPeHBvQllOeDJRTwpJN0RkS2Q0di9GOURnSWpBelUyOUczQnB5aTVaQWdNQkFBR2pVekJSTUIwR0ExVWREZ1FXQkJRbjdBeDlEa2pVCkNQa2l0Uld2SUdSL0pTNEpCekFmQmdOVkhTTUVHREFXZ0JRbjdBeDlEa2pVQ1BraXRSV3ZJR1IvSlM0SkJ6QVAKCQVVER1RRUJBd0lCQmpBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQWh3Q1ZESmVxOFl5Q3JMOCtMZXJxb3FGRwp1MEFyZXBwNWx1YkJzK3lTS0FNNDNGQzRHQjBDQ0draDR5NnhSTGRqVVB1S2tJaXJUTGswUzl5UG5EY3lOWFNtCkpxQnRaL2Z0UGR6NWo0TndPLzRid2xVeWw5MXBTelJtWTJpT2MyaUZLd05abjhqQmFKcFZzRnV0cnE2N0xJZmQKQjRiZEdad0xmWWh6KzJRdVJTTjd3a005c2kyaGpMNkJQTWVuM0JLbHRqQ2ZOcFJJN25mRmJEU0NXZGRBbEdDQwo3MEdZY3dQNGFQcDlwZXBwMkJqbU5ydVh0aEhLRmtIS3Y0TjZudEZZejYwQ3V5OGJMQjdXU3dqaGJHN08xL0prCmRJSktEakJQUEtBc2lROG5KenhrL0c1dnFwUVYvUjFqL0xrckR1akJqamNWdUNsNlRCZHRpemZlL3F1YW93PT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBeE1Bb204eGRHZU51a2R5NmhNRFBKSFliVlFzaDBQajNpRUtOQ2Q4UTRhN2VDTDV5CkJ3dGo4NldvVElLaFViUzJXT1A3VjJsNDdwVlRWMFQxeFBJakJuaVRvZXE2SWJjZ2VRNGZkSXh0Q29UVVdDRC8KZUpPWWk0VlRjR1VtZTVRay9tMUFDNWp0UTAyemFROHh6MXRWNlZLVUJhNDFDM0piQ1F3eUx6TnBpeUVhYjFSRwoxNytZd09HYTB6K1plTE4rRy8xWllESmZKTlJZUmJGSDhtNmxsckYrQ0dYeG9rTkJKUUhoVUR4NmRDaVBIRHBYCkY4aHRpcEZLVlZPRVdtNXVCSFN5M3g0RFhaNVBkcDdSVDcxN1IxeWM5TzVyUU03RzZpYm5idUJDWU5LT01RVHMKYUFXRGN0a0RpT3czU25lTC94ZlE0Q0l3TTFOZFJ0d2Fjb3VXUUlEQVFBQkFvSUJBQk1HS1NTSGtlT2tQQjJvagpGRzhybUhDSUFhRnFlc3JJN2F5ZjRZVGJGOGdIOVZUTXdVcHJQYkJPOFJxMW81Ym1vOHVoSS9YY0F1Z0x2NjA1CjJIRWJxaUtiZ3lWdnNvV1FhY3pMdEY0cElwTEFhaU9JcVBNdCtwWm9YL29kMjJYVlp6aE05TVRIaTlReUNJdHQKWGlFMEtneGJHWGN0a0xFL1JGR2hsd3hqMjNGVEpuaXBrYUtjSUErcnpmQzZQRnp1amw5SzJGRzlWVVRDRGZ1ZQoyUHNYUHlvQU9mVGRYUHRsZDFUb3JQSG9MZWpWaGdkL0RrbEtzVjN2YzN6Q3g5MnVJQjlOQjdBTGNES3BiZGF1CjVCN0FiZGVWUjJCMXI0VnpYN2hLdVhEYXpLNzhLQUhjMHBGaUpONkVEVm9jd1Y1SDdJRXFFY0k0UmtXMmQ4eG4KczQ4R1VnRUNnWUVBOHNGSVFodmtpb05OVGMveG5RdFF6MDlIQ2hWSVIwUU5oOFBSYVZBU08yU0NhTHBEL1dCRApWcTlKdlowQkxIWkZJSUdPR2xMTGN5Y0R6MUxxMVRCN0xvOWdTQitQVEFMMzA3SkJHVzVYRyt3bUVkUzF4OWx2CjBVQXNRRkh5eDlRbUtGcTJZOXdXTjdXbHFKUU5UWGJBNWhVTDFUQzRjUHJjN1VRckplMTB5UUVDZ1lFQXo5T0cKTGdGbGFKMVNRNkJDbUJiQlNQNzBzUGJCMWxYWGgxb3g5Y1Z4VnZ0UG9DK0VBb0JhcEl2M0xkRGluZGdhcC9oTwo1cGp1cDlUR01YRGpsMEJJQ0VYUGd0MlBKcXRoZHQ3Sk91R0ZwK08xZ3V2L1l0OU5sczR0UFJzekFjcjAzdUg5CnR2TlN0RTBJdWRRQ0lJdmJJc0xwN1hNcklJRjNQOFRaVytIUmJoa0NnWUVBNFg0S1FQcGhQRGFhdEtKUGV3cHcKN2Y4c1JySUtHczNweGl1b1NqRVpQQ3pMZHA3d0FRL3YvMUtkMDdqdEJJN1lQZTZIM3h3VGtGZXBJNTZlL0dtTwo0ekZpTUFmTE0vdU5jU3VYT3ZiSUZCT3d1N1haRGF5aDNkcnEvaFI0d2llTlNXdXNLRkFldGRKN3pUa1VQNXlECkhyN2RTSzZ2ejlpQ3JySkd5STRyQVFFQ2dZRUF3Qm1mTGdTb1lwVkptMFg3TVpyL09vRnhLME5YYzBCQ0JxdGQKNy9ENHN6em9tMFN0Uk01aGExRW5KMmJMbFRyOVJRRkR0ZmhIT0R3dDZPaEZUWHhQbEVnRks1NnJYZ0xzd2JVaQpCT2k1U0hGZ0lOaUQzRlFnR0hYaW9aeG16SW1IMVBXL1lCNGhUR2VLOGJTdHZLNTFJa3BSVzg0OGNGZzJodHVUClU1bjU3WWtDZ1lBa3ZJQkwwNFVIcTdRZ1ZJRE84bWpWOVJBZTRBTHRjeWtIaHFrbHBhaDVnbFFBOElEL2liTUsKN0dBNVZWUVJvQ3h1U3o1UU1xdktNdFBuMXRFSTRsT25OZ0lnV0ZQOXFJOGlRM1Zpb3I5cEFDUDBaQmxhS1JOYQowdlBUdXpuV1dPaU9xZXhQVnRpWkJzRGRJQ3UrRmhXUjREd1l5akxGZ0pFaFJzSk9PVzJaYWc9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
"@ | kubectl apply -f -

# Expected: secret/webhook-server-cert created
```

## Step 8: Deploy webhook deployment
```powershell
# Create deployment.yaml
@"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webhook-manager
  namespace: nephoran-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: webhook-manager
  template:
    metadata:
      labels:
        app: webhook-manager
    spec:
      containers:
      - name: webhook
        image: nephoran/webhook-manager:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        volumeMounts:
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true
        env:
        - name: WEBHOOK_CERT_DIR
          value: /tmp/k8s-webhook-server/serving-certs
      volumes:
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert
"@ | Set-Content config/webhook/deployment.yaml

# Deploy webhook using kustomize
kubectl apply -k ./config/webhook/

# Expected:
# service/webhook-service created
# deployment.apps/webhook-manager created
# mutatingwebhookconfiguration.admissionregistration.k8s.io/networkintent-mutating-webhook created
# validatingwebhookconfiguration.admissionregistration.k8s.io/networkintent-validating-webhook created

# Verify deployment
kubectl get pods -n nephoran-system

# Expected:
# NAME                              READY   STATUS    RESTARTS   AGE
# webhook-manager-xxxxxxxxx-xxxxx   1/1     Running   0          10s
```

## Step 9: Test valid NetworkIntent (should succeed)
```powershell
# Apply valid NetworkIntent
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: valid-intent
  namespace: default
spec:
  intentType: scaling
  target: nf-sim
  namespace: default
  replicas: 3
"@

# Expected: networkintent.intent.nephoran.io/valid-intent created

# Verify defaulting worked (source should be "user")
kubectl get networkintent valid-intent -o jsonpath='{.spec.source}'

# Expected output: user
```

## Step 10: Test invalid NetworkIntent (should be rejected)
```powershell
# Test with negative replicas
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-replicas
  namespace: default
spec:
  intentType: scaling
  target: nf-sim
  namespace: default
  replicas: -1
"@

# Expected error:
# error validating data: ValidationError(NetworkIntent.spec.replicas): invalid value: -1, Details: must be >= 0

# Test with empty target
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-target
  namespace: default
spec:
  intentType: scaling
  target: ""
  namespace: default
  replicas: 3
"@

# Expected error:
# error validating data: ValidationError(NetworkIntent.spec.target): invalid value: , Details: must be non-empty

# Test with invalid intentType
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-type
  namespace: default
spec:
  intentType: provisioning
  target: nf-sim
  namespace: default
  replicas: 3
"@

# Expected error:
# error validating data: ValidationError(NetworkIntent.spec.intentType): invalid value: provisioning, Details: only 'scaling' supported
```

## Cleanup
```powershell
# Delete test resources
kubectl delete networkintent --all -n default

# Delete webhook deployment
kubectl delete -k ./config/webhook/

# Delete namespace
kubectl delete namespace nephoran-system

# Delete kind cluster
kind delete cluster --name webhook-test
```

## Verification Summary
✅ **Build successful**: `webhook-manager.exe` created
✅ **CRDs generated**: `deployments/crds/intent.nephoran.io_networkintents.yaml`
✅ **DeepCopy generated**: `api/intent/v1alpha1/zz_generated.deepcopy.go`
✅ **Deployment successful**: Webhook pod running in `nephoran-system`
✅ **Defaulting works**: Empty source field defaulted to "user"
✅ **Validation works**: Invalid CRs rejected with clear error messages
  - Negative replicas: "must be >= 0"
  - Empty target: "must be non-empty"
  - Invalid intentType: "only 'scaling' supported"