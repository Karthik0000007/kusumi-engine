# Kasumi Engine — Reranking Service

C++/TensorRT-based latency-constrained LTR reranking service.

See the main [README](../README.md) for project overview.

## Building

```bash
mkdir build && cd build
cmake ..
cmake --build .
```

## Implementation Status

- [ ] LTR model training pipeline (Week 4)
- [ ] ONNX export & TensorRT engine build (Week 4)
- [ ] Triton serving with latency budget enforcement (Week 4)
