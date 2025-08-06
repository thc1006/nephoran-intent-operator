# kpt-packages/5g-core/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: 5g-core-functions
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: 5G Core Network Functions Package
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-replacements:v0.1.1
      configPath: apply-replacements.yaml
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configMap:
        namespace: telecom-core
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.3.0

---
# kpt-packages/5g-core/upf-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: upf-function
spec:
  replicas: 2
  selector:
    matchLabels:
      app: upf-function
  template:
    metadata:
      labels:
        app: upf-function
    spec:
      containers:
      - name: upf
        image: free5gc/upf:v3.3.0
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
