# LLM Inference Benchmarks

**Date**: 2026-02-15T12:24:01+00:00
**GPU**: NVIDIA GeForce RTX 5080 (16,303 MiB VRAM)
**Ollama Version**: v0.16.1
**Runtime**: CUDA 13.0, Driver 580.126.09

---

## Models Tested

| Model                          | Parameters | Quantization | Disk Size | Family       |
|--------------------------------|------------|-------------|-----------|--------------|
| Llama 3.1 8B Instruct         | 8.0B       | Q5_K_M      | 5.7 GB    | Llama        |
| DeepSeek Coder V2 16B Lite    | 15.7B      | Q4_K_M      | 10.0 GB   | DeepSeek MoE |
| Mistral Nemo 12B Instruct     | 12.2B      | Q5_K_M      | 8.7 GB    | Llama        |
| Qwen 2.5 14B Instruct         | 14.8B      | Q4_K_M      | 9.0 GB    | Qwen2        |

## Single-Request Generation Performance (tokens/sec)

| Model                    | Short Prompt | Medium Prompt | Long Prompt | Average  |
|--------------------------|-------------|--------------|-------------|----------|
| **DeepSeek Coder V2**    | **260.58**  | **291.45**   | **290.04**  | **280.69** |
| Llama 3.1 8B             | 156.40      | 143.04       | 142.41      | 147.28   |
| Mistral Nemo 12B         | 97.47       | 95.06        | 94.78       | 95.77    |
| Qwen 2.5 14B             | 97.62       | 88.92        | 88.63       | 91.72    |

## Prompt Processing Speed (tokens/sec)

| Model                    | Short Prompt | Medium Prompt | Long Prompt |
|--------------------------|-------------|--------------|-------------|
| Llama 3.1 8B             | 1,424       | 3,242        | **6,195**   |
| **DeepSeek Coder V2**    | 7*          | 1,539        | 5,359       |
| Mistral Nemo 12B         | 474         | 2,252        | 4,216       |
| Qwen 2.5 14B             | 1,070       | 2,704        | 3,905       |

*DeepSeek short prompt shows cold-start penalty (model swap into VRAM).

## Time to First Token (seconds)

| Model                    | Short | Medium | Long  |
|--------------------------|-------|--------|-------|
| Llama 3.1 8B             | 0.118 | 0.119  | 0.139 |
| DeepSeek Coder V2        | 2.212*| 0.090  | 0.103 |
| Mistral Nemo 12B         | 0.120 | 0.125  | 0.154 |
| Qwen 2.5 14B             | 0.119 | 0.108  | 0.156 |

*First request includes model loading time; subsequent requests show ~100ms TTFT.

## Total Request Duration (seconds, 256 output tokens)

| Model                    | Short | Medium | Long  |
|--------------------------|-------|--------|-------|
| Llama 3.1 8B             | 0.18  | 2.03   | 2.06  |
| **DeepSeek Coder V2**    | 2.29* | **1.08**| **1.08** |
| Mistral Nemo 12B         | 0.41  | 2.93   | 2.98  |
| Qwen 2.5 14B             | 0.21  | 3.12   | 3.19  |

## VRAM Usage Per Model

| Model                    | VRAM Used (MiB) | VRAM % of Total |
|--------------------------|-----------------|-----------------|
| Llama 3.1 8B             | 6,884           | 42.2%           |
| DeepSeek Coder V2 16B    | 14,708          | 90.2%           |
| Mistral Nemo 12B         | 9,962           | 61.1%           |
| Qwen 2.5 14B             | 10,554          | 64.7%           |

**Note**: Ollama dynamically loads/unloads models. Only one model fits in VRAM at a time for the larger models. Llama 3.1 8B is the most VRAM-efficient.

## Concurrent Request Performance

Model: Llama 3.1 8B Instruct (Q5_K_M)
Prompt: Medium-length networking question
Output: 128 tokens per request

| Concurrency | Wall Time (avg) | Tokens/sec (avg) | Throughput Loss | GPU Util | VRAM Used |
|-------------|-----------------|-------------------|-----------------|----------|-----------|
| 1 (baseline)| 1.109s          | 143.58 tok/s      | --              | ~93%     | 6,886 MiB |
| 2           | 1.355s          | 113.04 tok/s      | -21.3%          | 93%      | 6,886 MiB |
| 4           | 1.632s          | 96.16 tok/s       | -33.0%          | 93%*     | 6,886 MiB |

*GPU util reported as 0% on the second sample due to measurement timing.

### Concurrent Request Analysis

```
Concurrency vs Throughput (tokens/sec per request)

  160 |  X baseline
  150 |  |
  140 |  |
  130 |  |
  120 |  |
  110 |  |...X 2-concurrent
  100 |     |
   90 |     |...X 4-concurrent
   80 |        |
      +--------+--------+--------+
      1        2        3        4  (concurrent requests)

Total system throughput:
  1 request:  143.58 tok/s total
  2 requests: 226.08 tok/s total (+57.5%)
  4 requests: 384.64 tok/s total (+167.9%)
```

**Key Insight**: While per-request throughput decreases with concurrency, total system throughput increases substantially. At 4 concurrent requests, the system delivers 2.68x the throughput of a single request, indicating good GPU utilization sharing.

## Performance Summary

### Fastest Generator: DeepSeek Coder V2 (291 tok/s)
- Mixture-of-Experts architecture enables exceptional generation speed
- Only activates a subset of parameters per token, reducing compute per step
- Penalty: Requires 90.2% of VRAM (14.7 GB), leaving little room for other models

### Best All-Round: Llama 3.1 8B (143 tok/s, 42% VRAM)
- Best VRAM efficiency at only 6.9 GB
- Strong prompt processing (up to 6,195 tok/s)
- Good concurrent request scaling
- Recommended as the default model for the RAG pipeline

### Best for Quality (large context): Qwen 2.5 14B (89 tok/s)
- Largest standard model at 14.8B parameters
- Strongest instruction following and reasoning capability
- Acceptable speed for non-real-time use cases

### Coding Specialist: DeepSeek Coder V2 (291 tok/s)
- Optimized for code generation tasks
- Highest raw generation speed due to MoE architecture

## Recommendations

1. **Default RAG model**: Use Llama 3.1 8B for intent processing (best speed/VRAM ratio)
2. **Code generation**: Use DeepSeek Coder V2 when generating Kubernetes manifests
3. **Complex reasoning**: Use Qwen 2.5 14B for multi-step intent analysis
4. **Concurrent load**: At 4+ concurrent users, expect ~33% per-request slowdown
5. **Model preloading**: Keep the primary model warm in VRAM; model swaps cost 2+ seconds

## Methodology

- **Benchmark script**: `benchmarks/llm-benchmark.sh`
- **Concurrent script**: `benchmarks/llm-concurrent-benchmark.sh`
- **Raw data**: `benchmarks/llm-raw-results.json`
- **Prompt lengths**: Short (~10 tokens), Medium (~55 tokens), Long (~230 tokens)
- **Output**: 256 tokens (single) / 128 tokens (concurrent)
- **Warm-up**: Each model warmed with a 1-token generation before benchmarking
- **GPU monitoring**: `nvidia-smi dmon` during inference runs

## Reproducibility

```bash
# Run single-model benchmarks
bash benchmarks/llm-benchmark.sh

# Run concurrent benchmarks
bash benchmarks/llm-concurrent-benchmark.sh

# Monitor GPU during inference
nvidia-smi dmon -d 1 -s pucvme
```
