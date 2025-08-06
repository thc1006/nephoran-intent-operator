telecom-llm-automation/
├── .devcontainer/                    # Development containers
│   ├── devcontainer.json
│   └── Dockerfile
├── cmd/                              # Application entry points
│   ├── llm-processor/
│   ├── nephio-bridge/
│   └── oran-adaptor/
├── pkg/                              # Core packages
│   ├── rag/                          # RAG pipeline implementation
│   ├── nephio/                       # Nephio integration
│   ├── oran/                         # O-RAN interface bridges
│   └── controllers/                  # Kubernetes controllers
├── kpt-packages/                     # Nephio-compatible packages
│   ├── 5g-core/
│   ├── oran-nfs/
│   └── network-slices/
├── deployments/                      # Deployment configurations
│   ├── kubernetes/
│   ├── helm/
│   └── kustomize/
├── config/                          # Configuration files
├── scripts/                          # Automation scripts
├── Makefile                          # Build automation
└── README.md
