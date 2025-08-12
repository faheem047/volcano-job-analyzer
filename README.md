
Volcano Job Analyzer is a Golang CLI that analyzes Volcano Job specifications, computes resource needs, and recommends intelligent multi‑cluster placement strategies for distributed ML/batch workloads on Kubernetes.

### Background: Volcano and VolcanoJob
Volcano is a batch scheduling system for Kubernetes designed for high‑performance computing and AI/ML workloads. VolcanoJob (often abbreviated as vcjob) is Volcano’s custom resource that extends Kubernetes Jobs with batch‑oriented capabilities such as a dedicated scheduler, `minAvailable`, task definitions, queues, priorities, plugins, and lifecycle policies. For the official specification and key fields, see the Volcano VolcanoJob documentation: [VolcanoJob (vcjob) docs](https://volcano.sh/en/docs/vcjob/).

### Key Features
- **Parsing & Validation**: Parses `batch.volcano.sh/v1alpha1` Job YAML and validates structure/best practices
- **Resource Analysis**: CPU, memory, GPU, storage, and custom resources per‑task and aggregated
- **Task Intelligence**: Classifies roles (master/worker/ps/driver/executor), detects common frameworks
- **Network & Scalability**: Communication pattern, latency sensitivity, and scaling characteristics
- **Multi‑Cluster Placement**: Strategy selection, cluster scoring, anti‑affinity, resource distribution
- **Optimization Hints**: Cost/performance recommendations from analysis results
- **Flexible Output**: Human‑readable table, JSON, or YAML

---

## Install

### From source
```bash
git clone https://github.com/faheem047/volcano-job-analyzer.git
cd volcano-job-analyzer
make build
./build/volcano-job-analyzer --version
```

### Go install
```bash
go install github.com/faheem047/volcano-job-analyzer@latest
```

### Docker
```bash
docker build -t volcano-job-analyzer:local .
docker run --rm -v $PWD:/work -w /work volcano-job-analyzer:local --help
```

---

## Usage

The CLI exposes two commands: `analyze` and `validate`.

```bash
volcano-job-analyzer [--config configs/analyzer-config.yaml] [--log-level info] [--output table] <command> <args>
```

### Analyze a job
```bash
# Table output (default)
./build/volcano-job-analyzer analyze examples/pytorch-job.yaml

# JSON output
./build/volcano-job-analyzer --output json analyze examples/tensorflow-job.yaml

# With custom config
./build/volcano-job-analyzer --config configs/analyzer-config.yaml analyze examples/spark-job.yaml
```

What it does:
1) Parses the Volcano Job YAML
2) Performs per‑task and aggregate analysis
3) If `clusterProfilesPath` is configured, generates a multi‑cluster placement plan
4) Emits results as table/JSON/YAML

### Validate a job
```bash
./build/volcano-job-analyzer validate examples/pytorch-job.yaml
```
Validates structure and prints best‑practice warnings (does not change the file).

---

## Configuration

Default configuration lives in `configs/analyzer-config.yaml`. Key options:

```yaml
analyzer:
  enableAdvancedMetrics: true
  networkLatencyThreshold: 10ms
  resourceEfficiencyThreshold: 0.8
  defaultJobTimeout: 2h

placement:
  maxClusters: 5
  minClusterUtilization: 0.2
  maxClusterUtilization: 0.8
  networkLatencyWeight: 0.3
  resourceAffinityWeight: 0.4
  costOptimizationWeight: 0.2
  performanceOptimizationWeight: 0.1

logging:
  level: info
  format: console

clusterProfilesPath: "configs/cluster-profiles.yaml"
```

- **analyzer.***: knobs controlling analysis thresholds
- **placement.***: weights and limits used by the placement engine
- **clusterProfilesPath**: enables multi‑cluster planning when provided

### Cluster profiles
Cluster capabilities are described in `configs/cluster-profiles.yaml`:

```yaml
clusters:
  - name: gpu-cluster-us-east
    availableResources:
      cpu: 1000000m
      memory: 2048Gi
      gpu: 32
      storage: 10Ti
    networkBandwidth: 10000
    networkLatency: 5ms
    costPerHour: 2.5
    gpuTypes: [nvidia-tesla-v100, nvidia-tesla-a100]
    specialCapabilities: [ml-optimized, high-memory, nvlink]
```

---

## Examples

Quick run targets are provided in the `Makefile`:

```bash
make run-pytorch       # analyze examples/pytorch-job.yaml
make run-tensorflow    # analyze examples/tensorflow-job.yaml
make run-complex       # analyze complex multi-framework job
```

Generate JSON outputs for all examples:
```bash
make demo-json
```

---

## Architecture Overview

- `main.go`: CLI entry point; parses flags and routes to commands
- `cmd/analyze/analyze.go`: Orchestrates parse → analyze → placement → optimize → output
- `cmd/validate/validate.go`: Validates job spec and prints warnings
- `pkg/parser/parser.go`: Volcano Job YAML → Go types; basic structural validation
- `pkg/parser/validator.go`: Best‑practice checks and validation helpers
- `pkg/analyzer/analyzer.go`: Core analysis (task roles, totals, utilization, network/security)
- `pkg/analyzer/placement.go`: Multi‑cluster placement strategy (scoring, distribution, anti‑affinity, topology)
- `pkg/analyzer/optimizer.go`: Cost/performance recommendations
- `pkg/analyzer/types.go`: Shared data models (e.g., `JobAnalysis`, `TaskAnalysis`, `PlacementStrategy`)
- `pkg/utils/config.go`: Loads config via Viper with sane defaults
- `pkg/utils/logger.go`: Structured logging (zap)
- `examples/`: Ready‑to‑run Volcano Job specs
- `configs/`: Analyzer/placement configuration and cluster profiles
- `tests/`: Unit tests for analyzer behavior

High‑level flow:
```
YAML → parser → analyzer → (placement) → optimizer → output
```

---

## Output Formats

- `table`: concise human‑readable summary for terminals (default)
- `json`: machine‑readable, great for automation
- `yaml`: human‑ and config‑friendly representation

---

## Development

Prereqs: Go 1.21+

Common tasks:
```bash
make deps         # download dependencies
make lint         # run golangci-lint
make test         # run unit tests
make build        # build local binary in ./build
make build-all    # build for linux/darwin/windows
```

Live‑reload dev container (optional):
```bash
docker-compose up dev
```

---

## Troubleshooting

- "failed to load configuration": check `--config` path and YAML syntax
- No placement shown: ensure `clusterProfilesPath` is set in config
- Lint tool missing: install with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- Windows paths: quote file paths containing spaces

---

## License

This project is open source. See repository for license information.

