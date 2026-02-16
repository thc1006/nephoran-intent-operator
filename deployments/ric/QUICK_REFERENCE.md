# O-RAN RIC - E2 & KPM 快速參考

## 🚀 快速開始

### 驗證部署
```bash
cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric
./verify-e2-kpm.sh
```

### 查看所有資源
```bash
kubectl get all -n ricxapp
kubectl get all -n ricplt | grep -E 'e2|submgr'
```

---

## 📦 已部署組件

### E2 測試客戶端
```bash
# 進入 pod
kubectl exec -it -n ricxapp deployment/e2-test-client -- bash

# 運行連接測試
kubectl exec -n ricxapp deployment/e2-test-client -- bash /scripts/test-e2-connectivity.sh

# 查看日誌
kubectl logs -n ricxapp -l app=e2-test-client --tail=50
```

### KPM xApp
```bash
# 進入 pod
kubectl exec -it -n ricxapp deployment/ricxapp-kpimon -- bash

# 健康檢查
kubectl exec -n ricxapp deployment/ricxapp-kpimon -- curl -s http://localhost:8080/health

# 查看日誌
kubectl logs -n ricxapp -l app=ricxapp-kpimon --tail=50
```

---

## 🔧 常用命令

### E2 Manager API
```bash
# 查詢連接的 E2 節點
kubectl exec -n ricxapp deployment/e2-test-client -- \
  curl -s http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/nodeb/states

# 獲取 E2 Manager 健康狀態
kubectl exec -n ricxapp deployment/e2-test-client -- \
  curl -s http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/health
```

### DNS 測試
```bash
kubectl exec -n ricxapp deployment/e2-test-client -- \
  nslookup service-ricplt-e2term-sctp-alpha.ricplt.svc.cluster.local

kubectl exec -n ricxapp deployment/e2-test-client -- \
  nslookup service-ricplt-e2mgr-http.ricplt.svc.cluster.local
```

### 端口掃描
```bash
kubectl exec -n ricxapp deployment/e2-test-client -- \
  nmap -p 36422 service-ricplt-e2term-sctp-alpha.ricplt.svc.cluster.local
```

---

## 📊 服務端點

### RIC 平台 (ricplt)
| Service | Type | IP | Port(s) |
|---------|------|-----|---------|
| E2 Manager HTTP | ClusterIP | 10.100.165.50 | 3800 |
| E2 Manager RMR | ClusterIP | 10.107.251.91 | 4561, 3801 |
| E2 Term SCTP | NodePort | 10.100.232.16 | 36422:32222/SCTP |
| Subscription Manager | ClusterIP | None | 4560, 4561 |
| Routing Manager | ClusterIP | 10.111.72.6 | 4560, 4561 |

### xApps (ricxapp)
| Service | Type | IP | Port(s) |
|---------|------|-----|---------|
| KPM xApp HTTP | ClusterIP | 10.109.211.192 | 8080 |
| KPM xApp RMR | ClusterIP | 10.102.95.217 | 4560, 4561 |

---

## 🐛 故障排除

### Pod 不運行
```bash
kubectl get pods -n ricxapp
kubectl describe pod -n ricxapp <pod-name>
kubectl logs -n ricxapp <pod-name>
```

### 服務無法訪問
```bash
kubectl get svc -n ricxapp
kubectl get endpoints -n ricxapp
```

### 網絡問題
```bash
# 從 E2 測試客戶端測試連接
kubectl exec -it -n ricxapp deployment/e2-test-client -- bash

# 在 pod 內執行
ping service-ricplt-e2mgr-http.ricplt.svc.cluster.local
nc -zv service-ricplt-e2mgr-http.ricplt.svc.cluster.local 3800
curl http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/nodeb/states
```

---

## 📝 配置文件位置

- **E2 測試客戶端**: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/e2sim/e2-test-client.yaml`
- **KPM xApp**: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/xapps/kpm/kpm-xapp-deployment.yaml`
- **驗證腳本**: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/verify-e2-kpm.sh`

---

## 🔄 重新部署

### 清理現有部署
```bash
kubectl delete deployment -n ricxapp e2-test-client ricxapp-kpimon
kubectl delete svc -n ricxapp service-ricxapp-kpimon-http service-ricxapp-kpimon-rmr
kubectl delete configmap -n ricxapp e2-test-scripts kpm-xapp-config
```

### 重新部署
```bash
cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric

# 部署 E2 測試客戶端
kubectl apply -f e2sim/e2-test-client.yaml

# 部署 KPM xApp
kubectl apply -f xapps/kpm/kpm-xapp-deployment.yaml

# 驗證
./verify-e2-kpm.sh
```

---

## 📚 相關文檔

- **完整部署報告**: [E2SIM_KPM_DEPLOYMENT.md](./E2SIM_KPM_DEPLOYMENT.md)
- **RIC M Release 部署**: [DEPLOYMENT_SUCCESS.md](./DEPLOYMENT_SUCCESS.md)
- **RIC 功能測試**: [RIC_FUNCTIONAL_TEST.md](./RIC_FUNCTIONAL_TEST.md)

---

## 🎯 下一步

1. **部署真實 E2 節點** - 使用 srsRAN gNB 或 E2SIM
2. **實現 E2 訂閱** - KPM xApp 訂閱 RAN 指標
3. **集成 Nephoran** - 與 Intent Operator 連接
4. **監控和可觀測性** - Prometheus + Grafana 儀表板

---

**更新時間**: 2026-02-15
**狀態**: ✅ 測試環境就緒
