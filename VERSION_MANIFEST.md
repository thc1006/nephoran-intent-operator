# Nephoran Intent Operator - 版本清單

## 📅 最後更新
**日期**: 2026-02-15
**環境**: 生產級測試環境（單節點）
**架構**: **✅ 全 Kubernetes 原生架構**

---

## 🎯 核心平台版本

### Kubernetes 生態系統
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **Kubernetes** | 1.35.1 | ✅ 全 K8s | Server/Client, 完整 K8s 功能 |
| **containerd** | 2.2.1 | ✅ K8s CRI | K8s 原生容器運行時（替代 Docker） |
| **kubelet** | 1.35.1 | ✅ K8s | K8s node agent |
| **kubeadm** | 1.35.1 | ✅ K8s | K8s 集群初始化工具 |
| **kubectl** | 1.35.1 | ✅ K8s | K8s 命令行工具 |
| **Calico CNI** | v3.29.1 | ✅ K8s | K8s 網絡插件 |

### 包管理與部署工具
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **Helm** | 4.1.0 | ✅ K8s | Kubernetes 包管理器 |
| **ric-common** | 3.3.2 | ✅ K8s | O-RAN SC Helm 公共庫 |

---

## 🚀 O-RAN SC RIC Platform (M Release)

### 發布信息
- **Release**: M Release
- **發布日期**: 2025-12-20
- **部署方式**: ✅ **全 Kubernetes 部署**（Helm Charts）
- **Registry**: nexus3.o-ran-sc.org:10002/o-ran-sc

### 核心組件版本

#### 基礎設施組件（Infrastructure）
| 組件 | 版本 | Helm Chart | 部署方式 |
|------|------|------------|----------|
| **infrastructure** | 3.0.0 | infrastructure-3.0.0 | ✅ K8s Deployment |
| Kong (API Gateway) | - | 內建於 infrastructure | ✅ K8s Deployment (2 replicas) |
| Prometheus Server | - | 內建於 infrastructure | ✅ K8s Deployment |
| Prometheus Alertmanager | - | 內建於 infrastructure | ✅ K8s Deployment (2 replicas) |

#### RIC 平台組件（Platform Components）
| 組件 | 版本 | Helm Chart | Docker Image Tag | 部署方式 |
|------|------|------------|------------------|----------|
| **dbaas** (Redis) | 2.0.0 | dbaas-2.0.0 | 0.6.5 | ✅ K8s StatefulSet |
| **appmgr** | 3.0.0 | appmgr-3.0.0 | 0.5.9 | ✅ K8s Deployment |
| **e2mgr** | 3.0.0 | e2mgr-3.0.0 | 6.0.7 | ✅ K8s Deployment |
| **e2term** | 3.0.0 | e2term-3.0.0 | 6.0.7 | ✅ K8s Deployment |
| **rtmgr** | 3.0.0 | rtmgr-3.0.0 | 0.9.7 | ✅ K8s Deployment |
| **submgr** | 3.0.0 | submgr-3.0.0 | 0.10.3 | ✅ K8s Deployment |
| **a1mediator** | 3.0.0 | a1mediator-3.0.0 | 3.2.3 | ✅ K8s Deployment |
| **vespamgr** | 3.0.0 | vespamgr-3.0.0 | 0.7.5 | ✅ K8s Deployment |
| **o1mediator** | 3.0.0 | o1mediator-3.0.0 | 0.6.4 | ✅ K8s Deployment |
| **alarmmanager** | 5.0.0 | alarmmanager-5.0.0 | 0.5.17 | ✅ K8s Deployment |

---

## 🤖 AI/ML 技術棧

### LLM 運行時
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **Ollama** | 0.16.1 | ✅ K8s | 部署在 K8s, GPU 加速 |
| **llama3.1** | 8B Q4_K_M | - | 量化模型（GPU 優化） |
| **mistral** | 7B Q5_K_M | - | 量化模型（GPU 優化） |

### 向量資料庫
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **Weaviate** | 1.34.0 | ✅ K8s | 部署在 K8s (Helm chart) |

### RAG 服務
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **RAG FastAPI** | 自定義 | ✅ K8s | Python FastAPI, K8s Deployment |
| LangChain | - | - | RAG 框架 |

---

## 📊 監控技術棧

### 監控與可觀測性
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **Prometheus** | - | ✅ K8s | 部署在 ricplt namespace |
| **Grafana** | - | ✅ K8s | 部署在 monitoring namespace |
| **Alertmanager** | - | ✅ K8s | 部署在 ricplt namespace |

---

## 🎮 GPU 與加速

### NVIDIA GPU 支持
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **NVIDIA Driver** | 580.126.09 | 主機 | Blackwell 架構支持 |
| **GPU Operator** | v25.10.1 | ✅ K8s | 以 Operator 方式部署 |
| **DRA Driver** | v25.12.0 | ✅ K8s | K8s 1.35 DRA 支持 |
| **CUDA** | 12.8 | - | GPU 運算框架 |

### GPU 硬體
| 組件 | 規格 | 說明 |
|------|------|------|
| **GPU 型號** | NVIDIA GeForce RTX 5080 | Blackwell 架構 |
| **VRAM** | 16,303 MiB | 顯存容量 |
| **架構** | Blackwell (sm_100) | 最新世代 |

---

## 🏗️ Nephoran Intent Operator

### 應用程式版本
| 組件 | 版本 | 架構 | 說明 |
|------|------|------|------|
| **Intent Operator** | 自定義 | ✅ K8s | Go 1.24, K8s Operator |
| **API Group** | intent.nephoran.com/v1alpha1 | ✅ K8s CRD | NetworkIntent CRD |
| **Manager Binary** | 自編譯 | ✅ K8s | 52 MiB, 靜態編譯 |
| **Container Image** | nephoran-intent-operator:latest | ✅ K8s | 17.9 MiB, distroless |

### 開發工具鏈
| 組件 | 版本 | 說明 |
|------|------|------|
| **Go** | 1.24.x | 主要開發語言 |
| **controller-runtime** | - | K8s Operator SDK |
| **buildah** | - | 無 Docker 容器構建工具 |
| **nerdctl** | - | containerd 原生 CLI |

---

## 🐧 作業系統與環境

### 系統信息
| 項目 | 版本/配置 | 說明 |
|------|-----------|------|
| **作業系統** | Ubuntu 22.04 LTS | Linux 5.15.0-161-generic |
| **架構** | x86_64 | 64-bit |
| **cgroup** | v2 (cgroup2fs) | K8s 1.35 要求 |
| **cgroup driver** | systemd | K8s 標準配置 |

### 系統調優
| 項目 | 值 | 說明 |
|------|-----|------|
| `fs.file-max` | 2,097,152 | 系統最大文件描述符 |
| `fs.inotify.max_user_instances` | 8,192 | inotify 實例限制 |
| `fs.inotify.max_user_watches` | 524,288 | inotify 監視限制 |

---

## 🏛️ 架構說明

### ✅ 全 Kubernetes 原生架構

本專案採用 **完全 Kubernetes 原生架構**，所有組件均部署在 Kubernetes 之上：

#### 為什麼是全 K8s？

1. **無 Docker 依賴**
   - ❌ 沒有使用 Docker Engine
   - ❌ 沒有使用 Docker Compose
   - ✅ 使用 containerd 作為 K8s CRI（Container Runtime Interface）
   - ✅ 所有容器由 K8s 統一管理

2. **Helm 統一部署**
   - ✅ O-RAN SC RIC: 11 個 Helm releases
   - ✅ Weaviate: Helm chart 部署
   - ✅ Prometheus/Grafana: Helm chart 部署
   - ✅ GPU Operator: Helm chart 部署
   - ✅ Intent Operator: K8s Deployment manifest

3. **K8s 原生資源**
   - ✅ Deployments (應用部署)
   - ✅ StatefulSets (有狀態服務，如 Redis)
   - ✅ Services (服務發現與負載均衡)
   - ✅ ConfigMaps (配置管理)
   - ✅ Secrets (密鑰管理)
   - ✅ CRDs (自定義資源，如 NetworkIntent)
   - ✅ PersistentVolumeClaims (持久化存儲)

4. **K8s 1.35 新特性**
   - ✅ DRA (Dynamic Resource Allocation) for GPU
   - ✅ cgroup v2 支持
   - ✅ containerd 2.x 整合

#### 架構優勢

| 優勢 | 說明 |
|------|------|
| **統一編排** | 所有服務由 K8s 統一管理和調度 |
| **自動恢復** | Pod 失敗自動重啟 |
| **服務發現** | K8s Service 提供內建 DNS |
| **負載均衡** | K8s Service 提供內建負載均衡 |
| **滾動更新** | K8s Deployment 支持零停機更新 |
| **資源限制** | K8s 提供 CPU/Memory/GPU 資源管理 |
| **可擴展性** | 可輕易擴展到多節點集群 |

---

## 📦 容器映像倉庫

### 使用的 Registry
| Registry | 用途 | 說明 |
|----------|------|------|
| `nexus3.o-ran-sc.org:10002` | O-RAN SC 官方發布倉庫 | RIC 所有組件 |
| `docker.io` | Docker Hub | Weaviate, InfluxDB 等第三方組件 |
| `nvcr.io` | NVIDIA GPU Cloud | GPU Operator 組件 |
| `registry.k8s.io` | Kubernetes 官方 | K8s 系統組件 |
| `localhost` | 本地構建 | Intent Operator (containerd 本地) |

---

## 🔗 網絡配置

### K8s 網絡
| 項目 | 配置 | 說明 |
|------|------|------|
| **CNI** | Calico | K8s 網絡插件 |
| **Pod CIDR** | 10.244.0.0/16 | Pod 網絡範圍 |
| **Service CIDR** | 10.96.0.0/12 | Service 網絡範圍 |
| **Node IP** | 192.168.10.65 | 單節點 IP |

### RIC 服務端口
| 服務 | 端口 | 類型 |
|------|------|------|
| A1 Mediator | 10000 | ClusterIP |
| E2 Manager | 3800 | ClusterIP |
| E2 Termination | 36422/SCTP | NodePort (32222) |
| Kong Proxy | 80, 443 | LoadBalancer |
| Prometheus | 80 | ClusterIP |

---

## 📝 K8s Namespaces

### 使用的 Namespaces
| Namespace | 用途 | Pod 數量 |
|-----------|------|----------|
| **ricplt** | RIC 平台組件 | 13 |
| **ricinfra** | RIC 基礎設施 | 0 (預留) |
| **ricxapp** | RIC xApp 應用 | 0 (預留) |
| **ricaux** | RIC 輔助服務 | 0 (預留) |
| **nephoran-system** | Intent Operator | 1 |
| **gpu-operator** | GPU Operator | ~5 |
| **monitoring** | Prometheus/Grafana | ~3 |
| **weaviate** | Weaviate 向量資料庫 | 1 |

---

## 🔐 儲存

### K8s 存儲類
| StorageClass | 提供者 | 用途 |
|--------------|--------|------|
| `local-path` | Rancher Local Path Provisioner | 本地存儲 (開發/測試) |

### PVC 使用
- ✅ Redis (dbaas): StatefulSet PVC
- ✅ Prometheus: Server PVC
- ✅ Weaviate: Data PVC
- ✅ E2 Termination: Data volume

---

## 🎯 版本兼容性驗證

### ✅ 已驗證的組合
- **K8s 1.35.1** + **containerd 2.2.1** + **Helm 4.1.0** = ✅ 完全兼容
- **O-RAN SC M Release** + **K8s 1.35.1** = ✅ 成功運行
- **GPU Operator v25.10.1** + **K8s 1.35 DRA** = ✅ 完全兼容
- **Ollama 0.16.1** + **RTX 5080 Blackwell** = ✅ GPU 加速正常

### ⚠️ 已知問題與解決
1. **Helm 4 本地倉庫**: 需使用 HTTP server (`python3 -m http.server`)
2. **文件描述符限制**: 需增加系統限制 (`fs.inotify.*`)
3. **API Deprecation 警告**: v1 Endpoints 已棄用（不影響功能）

---

## 🚀 部署統計

### 部署完成時間
- **K8s 集群**: ~10 分鐘
- **GPU Operator**: ~5 分鐘
- **O-RAN SC RIC**: ~5 分鐘（首次 ~10 分鐘含映像下載）
- **Intent Operator**: ~2 分鐘
- **總計**: 約 30-40 分鐘（全新環境）

### 資源使用（單節點）
- **CPU 使用**: ~2-3 cores (requests)
- **Memory 使用**: ~6-8 GB (requests)
- **存儲使用**: ~15 GB (映像 + 資料)
- **GPU 使用**: RTX 5080 (Ollama LLM 推理時使用)

---

## 📚 參考文件

### 本專案文檔
- 主文檔: `README.md`
- RIC 部署報告: `deployments/ric/DEPLOYMENT_SUCCESS.md`
- Memory 記錄: `.claude/projects/.../memory/ric-deployment.md`
- 進度記錄: `docs/PROGRESS.md`

### 官方文檔
- [O-RAN SC M Release](https://docs.o-ran-sc.org/)
- [Kubernetes 1.35 Release Notes](https://kubernetes.io/blog/)
- [Helm 4 Documentation](https://helm.sh/)
- [containerd Documentation](https://containerd.io/)

---

## ✅ 總結

### 架構確認
✅ **本專案使用 100% Kubernetes 原生架構**
- 無 Docker 依賴
- 所有服務均為 K8s 資源
- 使用 containerd 作為 CRI
- Helm 統一包管理
- K8s 1.35 最新特性支持

### 部署狀態
✅ **所有組件成功運行**
- 13/13 RIC pods Running
- 1/1 Intent Operator Running
- GPU 加速正常工作
- 監控堆棧正常運作

---

**最後更新**: 2026-02-15 13:25 UTC
**維護者**: Nephoran Intent Operator Team
**環境**: 單節點 Kubernetes 1.35.1 生產級測試環境
