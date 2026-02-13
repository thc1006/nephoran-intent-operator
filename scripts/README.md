# Scripts overview (consolidation pass 1)

這份指南先以分類導覽取代大量零散檔案，未移動實體檔案；後續可依此分區搬移或淘汰。

## 主要分區（建議後續實體分組）
- **ci/**：CI/測試/覆蓋率/規範檢查
- **security/**：掃描、滲透測試、供應鏈/合規
- **deploy/**：部署與安裝腳本
- **ops/**：營運、監控、DR、效能/混沌
- **tools/**：資料與 prompt 工具
- **archive/**：一次性/舊版/待淘汰

## 目前檔案導覽（暫未搬移）

### CI / Build / Test
- ci-build.sh, ci-environment-optimizer.sh, ci-performance-monitor.sh, migrate-to-consolidated-pipeline.sh, verify-ci-consolidation.sh
- run-all-validations.sh, run-comprehensive-tests.sh, run-tests-with-coverage.sh, validate-build.sh, validate-coverage.sh, validate-workflows.sh, validate-ci-fixes.sh
- validate-root-allowlist.sh（Root Phase 3：檢查根目錄是否符合 allowlist，並輸出候選搬移項）
- lint-* 系列、apply-lint-fixes-2025.sh, lint-fixes/, lint-commands-guide.md
- test-* 系列（test-ci-build.sh, test-docker-build*.sh, test-race-detection.sh, test-llm-pipeline.sh, test-security-workflows.sh, test-workflow-*.py 等）

### Security / Compliance
- **moved to security/**: security-scan.sh, execute-security-audit.sh, security-penetration-test.sh, docker-security-scan.sh, vulnerability-scanner.sh, cve-scan.sh, check-critical-vulnerabilities.py, security-config-validator.sh, verify-security-scans.sh, verify-supply-chain.sh, daily-compliance-check.sh, compliance-checker.py, security-updates.sh
- benchmark-security-scan.sh, security-policy-templates.yaml

### Deploy / Install
- **moved to deploy/**: deploy.sh, deploy-optimized.sh, deploy-istio-mesh.sh, deploy-multi-region.sh, deploy-monitoring-stack.sh, deploy-mtls.sh, deploy-rag-system.sh, deploy-performance-monitoring.sh, deploy-webhook.sh, docker-build*.sh, smart-docker-build.sh, enable-build-cache.sh
- setup-enhanced-cicd.sh, install-hooks.sh, setup-secondary-cluster.sh, o2-production-readiness-check.sh

### Ops / DR / Monitoring / Performance
- **moved to ops/**: disaster-recovery.sh, disaster-recovery-system.sh, dr-automation-scheduler.sh, failover-to-secondary.sh, execute-production-load-test.sh, run-chaos-suite.sh, performance-benchmark-suite.sh, performance-monitor.sh
- edge-deployment-validation.sh, monitor-ci.sh, monitor-sla.sh, monitoring-operations.sh, monitoring/
- performance-regression/, performance-comparison/, optimize-build.sh, optimize-coverage.sh, optimize-test-execution.sh, quality-gate.sh, quality-metrics/, resource-optimizer.sh

### Tools / Data / Prompts
- populate_vector_store.py, populate_vector_store_enhanced.py
- prompt_ab_testing.py, prompt_evaluation_metrics.py, prompt_optimization_demo.py
- mcp-helm-server.py, postgres_optimize.sql, env-config-example.sh

### Archive / One-off / Needs review
- Archived to `archive/`: benchmark_results.md, format-bypass-check.sh, quick-fix-syntax.sh, fix-ci-timeout.sh
- Still to review: fix-disaster-recovery-test.sh, fix-gosec-issues.go, quickstart.sh, quick-db-check, deployment/ (mixed assets), hooks/, technical-debt/, validation/legacy helpers

## 後續建議
1) 按上述分區建立子目錄並移動腳本；移動後更新 Makefile/文檔入口。  
2) 標記一次性或重疊腳本（如多個 docker-build/test 變體）並淘汰，保留「標準慢全量」與「快速增量」各一。  
3) 為各分區補充簡短 README（用途、依賴、維護狀態）。  
4) 為主要入口（CI、部署、安全掃描）在根 README 或 Makefile 補連結。  
5) 在 PROGRESS.md 記錄每次搬移批次，保持可追蹤。
