# 🌸 Kasumi Engine

**APPI-Compliant, Low-Latency Real-Time Recommendation & Reranking System**

[![CI](https://github.com/kasumi-engine/kasumi-engine/actions/workflows/ci.yml/badge.svg)](https://github.com/kasumi-engine/kasumi-engine/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python)](https://python.org)
[![C++](https://img.shields.io/badge/C++-17-00599C?logo=cplusplus)](https://isocpp.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

---

## Overview

Kasumi Engine is a production-grade, end-to-end real-time recommendation and reranking platform. It ingests live clickstream and dwell-time events, performs privacy-preserving candidate retrieval using vector search, and applies latency-constrained Learning-to-Rank (LTR) reranking — all while embedding **Japan's APPI (Act on the Protection of Personal Information)** compliance through privacy-by-design principles.

### Key Features

- **Sub-100ms P99 latency** end-to-end recommendation pipeline
- **APPI-compliant** privacy-native design with ephemeral session hashing and differential privacy
- **Two-stage retrieval-reranking** with ANN candidate generation (<20ms) and LTR reranking (<30ms)
- **Graceful degradation** with latency-budget enforcement and automatic fallback
- **Full observability** with Prometheus metrics, Grafana dashboards, and drift detection

### Architecture

```
Client → Go Ingestion (anonymize + aggregate) → Redis Feature Store
       → Python Retrieval (ANN, ~500 candidates, <20ms)
       → C++/Triton Reranking (LTR, <30ms, fallback-safe)
       → Ranked Response
```

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Ingestion** | Go, gRPC, Redis | High-throughput event intake, anonymization, feature store |
| **Retrieval** | Python, FastAPI, FAISS/Milvus | ANN candidate generation, ephemeral user embeddings |
| **Reranking** | C++, TensorRT, Triton | Latency-constrained LTR inference |
| **Infra** | Docker, K8s, Prometheus, Grafana | Deployment, monitoring, observability |

## Quick Start

### Prerequisites

- Go 1.22+
- Python 3.10+
- CMake 3.20+
- Docker & Docker Compose
- Redis

### Setup

```bash
# Clone the repository
git clone https://github.com/kasumi-engine/kasumi-engine.git
cd kasumi-engine

# Run all tests
make test

# Lint all code
make lint

# Start the full stack locally
make up
```

### Configuration

Configuration is managed through YAML files in `config/`:

```bash
config/
├── schema.json     # JSON Schema for validation
├── default.yaml    # Baseline defaults
├── local.yaml      # Local development overrides
└── prod.yaml       # Production overrides
```

Environment variables with the `KASUMI_` prefix override file-based config:

```bash
export KASUMI_REDIS_HOST=localhost
export KASUMI_REDIS_PORT=6379
export KASUMI_PRIVACY_EPSILON=1.0
```

## Project Structure

```
kasumi-engine/
├── ingestion/          # Go — gRPC intake, anonymization, windowed aggregation
├── retrieval/          # Python — FAISS/Milvus ANN service (FastAPI)
├── reranking/          # C++/TensorRT — Triton-served LTR model
├── proto/              # Shared protobuf definitions
├── config/             # Shared YAML configuration + JSON Schema
├── deployment/         # Docker Compose, K8s manifests, CI/CD
│   ├── k8s/
│   ├── monitoring/     # Prometheus + Grafana configs
│   └── ci/
├── scripts/            # Data generation & utility scripts
├── notebooks/          # Offline training, nDCG evaluation, drift analysis
├── benchmarks/         # Latency/throughput load-test scripts + results
└── docs/               # Documentation, architecture, API specs
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — System design and component details
- [APPI Compliance](docs/appi_compliance.md) — Privacy design and compliance documentation
- [Developer Guide](docs/dev_guide.md) — Local setup and development workflows

## License

This project is licensed under the Apache License 2.0 — see the [LICENSE](LICENSE) file for details.
