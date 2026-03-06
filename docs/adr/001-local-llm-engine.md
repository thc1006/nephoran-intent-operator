# ADR-001: Local LLM Engine Selection

**Status:** Accepted | **Date:** 2025-03-06

## Context
The system requires a locally-hosted LLM for generating Kubernetes CRDs, KRM packages, and Nephio templates. The workstation has an NVIDIA RTX 5080 (16 GB VRAM). No cloud API calls permitted (data sovereignty).

## Decision
- **Primary engine**: vLLM — GPU-accelerated serving with OpenAI-compatible API, high throughput
- **Fallback engine**: Ollama — simpler setup, useful for development/testing
- **Primary model**: Qwen2.5-Coder-32B-Instruct (GPTQ/AWQ 4-bit quantization to fit 16 GB VRAM)
- **Fallback model**: CodeLlama-34B-Instruct (4-bit)

## Consequences
- Requires CUDA 12.x and `nvidia-container-toolkit`
- 4-bit quantization may reduce output quality; mitigated by multi-layer validation pipeline
- LLM Gateway abstracts backend choice — all services call a unified OpenAI-compatible API
- Model downloads require ~20 GB disk space
