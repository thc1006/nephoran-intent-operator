---
name: gateway-api
description: |
  Kubernetes Gateway API 配置技能。使用此技能來：
  - 安裝 Envoy Gateway
  - 配置 Gateway 和 HTTPRoute
  - 設定 TLS/HTTPS
  - 配置流量路由、權重分流、Header 匹配
  替代已退役的 Ingress-NGINX。
triggers:
  - gateway api
  - envoy gateway
  - httproute
  - ingress 替代
  - 流量路由
  - tls 配置
---

# Gateway API 配置技能

## 為什麼使用 Gateway API

```
Ingress-NGINX 將於 2026 年 3 月進入 kubernetes-retired：
- 不再發布新版本
- 不再修復安全漏洞
- 官方推薦遷移至 Gateway API

Gateway API 優勢：
- Kubernetes 原生標準
- 更強大的路由功能
- 多種實作可選 (Envoy, Istio, Traefik)
```

## 安裝 Envoy Gateway

```bash
# 使用 Helm 安裝
helm install eg oci://docker.io/envoyproxy/gateway-helm \
  --version v1.6.3 \
  -n envoy-gateway-system \
  --create-namespace

# 等待就緒
kubectl wait --timeout=5m -n envoy-gateway-system \
  deployment/envoy-gateway --for=condition=Available

# 驗證安裝
kubectl get pods -n envoy-gateway-system
kubectl get gatewayclass
```

## 基本配置

### GatewayClass

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
```

### Gateway

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: main-gateway
  namespace: default
spec:
  gatewayClassName: envoy
  listeners:
  - name: http
    port: 80
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: All
  - name: https
    port: 443
    protocol: HTTPS
    tls:
      mode: Terminate
      certificateRefs:
      - name: tls-secret
    allowedRoutes:
      namespaces:
        from: All
```

### HTTPRoute

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: app-route
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
  hostnames:
  - "app.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: app-service
      port: 80
```

## 進階路由

### 路徑匹配

```yaml
rules:
- matches:
  - path:
      type: Exact
      value: /api/v1/users
  - path:
      type: PathPrefix
      value: /api/v1/
  - path:
      type: RegularExpression
      value: /api/v[0-9]+/.*
```

### Header 匹配

```yaml
rules:
- matches:
  - headers:
    - name: X-Canary
      value: "true"
  backendRefs:
  - name: canary-service
    port: 80
```

### 權重分流

```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: /api
  backendRefs:
  - name: api-v1
    port: 8080
    weight: 90
  - name: api-v2
    port: 8080
    weight: 10
```

### 請求重寫

```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: /old-api
  filters:
  - type: URLRewrite
    urlRewrite:
      path:
        type: ReplacePrefixMatch
        replacePrefixMatch: /new-api
  backendRefs:
  - name: api-service
    port: 80
```

## TLS 配置

### 建立 TLS Secret

```bash
# 自簽憑證 (開發用)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=*.example.com"

kubectl create secret tls tls-secret \
  --cert=tls.crt --key=tls.key
```

### HTTPS Gateway

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: https-gateway
spec:
  gatewayClassName: envoy
  listeners:
  - name: https
    port: 443
    protocol: HTTPS
    hostname: "*.example.com"
    tls:
      mode: Terminate
      certificateRefs:
      - name: tls-secret
```

## 從 Ingress 遷移

### Ingress (舊)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80
```

### HTTPRoute (新)

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: my-route
spec:
  parentRefs:
  - name: main-gateway
  hostnames:
  - "app.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: my-service
      port: 80
```

## 監控與除錯

```bash
# 檢查 Gateway 狀態
kubectl get gateways -A
kubectl describe gateway main-gateway

# 檢查 HTTPRoute 狀態
kubectl get httproutes -A
kubectl describe httproute app-route

# 檢查 Envoy 代理日誌
kubectl logs -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=main-gateway

# 取得 Gateway IP
kubectl get gateway main-gateway -o jsonpath='{.status.addresses[0].value}'
```

## 常見問題

### Gateway 沒有 IP 地址
需要 LoadBalancer 實作，如 MetalLB：
```bash
helm install metallb metallb/metallb -n metallb-system --create-namespace
```

### HTTPRoute 不生效
```bash
# 檢查 parentRefs 是否正確
kubectl get httproute -o yaml | grep -A5 parentRefs

# 檢查 Gateway 是否接受此路由
kubectl get gateway -o yaml | grep -A10 allowedRoutes
```
