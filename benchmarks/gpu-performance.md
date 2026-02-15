# GPU Performance Benchmarks

**Date**: 2026-02-15T12:22:30+00:00
**GPU**: NVIDIA GeForce RTX 5080 (Blackwell Architecture)
**Driver**: 580.126.09
**CUDA**: 13.0

---

## Hardware Specifications

| Property               | Value             |
|------------------------|-------------------|
| GPU Model              | RTX 5080          |
| Architecture           | Blackwell         |
| VRAM Total             | 16,303 MiB        |
| VRAM Reserved          | 464 MiB           |
| VRAM Available (Free)  | 8,957 MiB         |
| BAR1 Memory            | 16,384 MiB        |
| Max Graphics Clock     | 3,090 MHz         |
| Max SM Clock           | 3,090 MHz         |
| Max Memory Clock       | 15,001 MHz        |
| Power Limit            | 360W (range: 300-390W) |
| PCIe Gen (Max)         | Gen 4             |
| PCIe Width             | x16               |
| PCIe Gen (Current)     | Gen 1 (idle)      |

## Idle State Characteristics

| Metric                 | Value             |
|------------------------|-------------------|
| Idle Power Draw        | 3.45W             |
| Idle Temperature       | 38-39C            |
| Performance State      | P8 (lowest power) |
| Graphics Clock (idle)  | 180 MHz           |
| Memory Clock (idle)    | 405 MHz           |

## Under-Load Characteristics (LLM Inference)

Measured during Llama 3.1 8B inference generating 512 tokens:

| Metric                 | Idle      | Under Load | Delta    |
|------------------------|-----------|------------|----------|
| Power Draw             | 3.45W     | 247-250W   | +246.5W  |
| Temperature            | 38C       | 50-51C     | +12C     |
| SM Utilization         | 0%        | 93%        | +93%     |
| Memory Bandwidth Util. | 0%        | 80%        | +80%     |
| Graphics Clock         | 180 MHz   | 2,820 MHz  | +2,640 MHz |
| Memory Clock           | 405 MHz   | 14,801 MHz | +14,396 MHz |

## GPU Utilization Timeline During Inference

```
Time(s)  Power(W)  Temp(C)  SM%  Mem%  GClk(MHz)  MClk(MHz)
  0       2         36       0    0     180        405        (idle)
  1       2         36       0    0     180        405        (prompt eval start)
  2       79        49       93   80    2820       14801      (generation ramp-up)
  3       247       50       93   80    2820       14801      (sustained generation)
  4       249       50       93   80    2820       14801      (sustained generation)
  5       250       51       93   80    2820       14801      (sustained generation)
  6       87        40       0    0     2857       14801      (cool-down)
  7       40        40       0    0     2857       14801      (returning to idle)
  8       35        39       0    0     2610       14801      (returning to idle)
  9       33        39       0    0     2610       14801      (near idle)
```

## Memory Bandwidth Estimate

Based on GDDR7 specifications for RTX 5080:
- Memory clock: 15,001 MHz max
- Memory bus: 256-bit
- Effective bandwidth: ~960 GB/s theoretical (GDDR7)
- Observed utilization during inference: 80% (~768 GB/s effective)

## PCIe Bandwidth

| Property              | Value          |
|-----------------------|----------------|
| Max Supported         | PCIe Gen 4 x16 |
| Current Link (idle)   | Gen 1 x16      |
| Theoretical Max BW    | ~32 GB/s (Gen4) |
| Note                  | Link auto-scales to Gen1 at idle for power savings |

## Thermal Analysis

- Idle temperature: 38-39C (ambient ~25C assumed)
- Peak temperature under load: 51C
- Thermal headroom: Excellent (~39C below typical throttle point of 90C)
- Fan speed at peak: Not reported (likely 0% at 51C due to adequate cooling)
- No thermal throttling events observed during benchmarks

## Key Observations

1. **Power Efficiency**: The GPU draws only 3.45W at idle (P8 state) and ramps to ~250W under LLM inference, well within the 360W TDP.
2. **Thermal Performance**: Excellent thermal characteristics with only 12C rise under sustained load, leaving 39C of headroom.
3. **Memory Bandwidth**: 80% memory bandwidth utilization during inference indicates the workload is memory-bandwidth bound, which is typical for LLM inference.
4. **SM Utilization**: 93% SM utilization shows the GPU compute units are well-utilized during inference.
5. **Clock Management**: Dynamic frequency scaling works correctly -- clocks ramp from 180MHz to 2,820MHz under load.
6. **PCIe Scaling**: PCIe link drops to Gen 1 at idle for power savings; model weights are already in VRAM, so host-to-device bandwidth is not a bottleneck during inference.

## Methodology

- GPU idle measurements: `nvidia-smi -q -d MEMORY,CLOCK,POWER,PERFORMANCE`
- Under-load measurements: `nvidia-smi dmon -c 10 -d 1 -s pucvme` during Ollama inference
- PCIe measurements: `nvidia-smi --query-gpu=pcie.link.gen.max,pcie.link.gen.current,pcie.link.width.max,pcie.link.width.current --format=csv`
- All measurements taken on a single-node Kubernetes cluster with Ollama as the only GPU consumer
