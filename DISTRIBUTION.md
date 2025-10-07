# Go-Doc-Go Distribution Guide

This document explains how to build and distribute the Go-Doc-Go worker for all supported platforms from a single development machine.

## Quick Start

Build distributions for **all platforms** with a single command:

```bash
./scripts/build-all-distributions.sh
```

This creates ready-to-deploy tarballs for:
- ✅ macOS (Intel & Apple Silicon)
- ✅ Linux x86_64
- ✅ Linux ARM64 (AWS Graviton, Raspberry Pi)
- ✅ Windows x86_64

## What's Included in Each Distribution

Every distribution is a **complete, standalone package** (~120MB):

```
worker-{platform}-{arch}/
├── worker                          # Platform-specific binary (69MB)
├── libonnxruntime.{dylib,so,dll}  # ONNX Runtime library (26MB)
├── models/all-MiniLM-L6-v2/       # Pre-exported ONNX model (88MB)
│   ├── model.onnx
│   ├── config.json
│   ├── tokenizer.json
│   └── vocab.txt
├── run-worker.sh                   # Launcher script
├── config.example.toml             # Example configuration
└── README.md                       # Distribution documentation
```

**No dependencies required on target machine!**
- No Python
- No pip install
- No model download
- Just extract and run

## Prerequisites (Development Machine Only)

```bash
# 1. Go 1.24+ (for compilation)
go version

# 2. Python 3.12+ with packages (for model export)
pip install onnx sentence-transformers torch

# 3. Bash shell (macOS, Linux, WSL on Windows, or Git Bash)
```

## Step-by-Step Build Process

### Step 1: Download ONNX Runtime Libraries

Download pre-built ONNX Runtime libraries for all platforms:

```bash
./scripts/download_onnx_runtime.sh
```

This downloads (~100MB total):
- `lib/onnxruntime/libonnxruntime-macos.dylib` (macOS Universal)
- `lib/onnxruntime/libonnxruntime-linux-x64.so` (Linux x86_64)
- `lib/onnxruntime/libonnxruntime-linux-arm64.so` (Linux ARM64)
- `lib/onnxruntime/onnxruntime-windows-x64.dll` (Windows)

**This step is cached** - only needs to be run once (or after ONNX Runtime version updates).

### Step 2: Export ONNX Model

Export the embedding model to ONNX format:

```bash
python scripts/export_model_to_onnx.py
```

Creates (~88MB):
- `go/models/all-MiniLM-L6-v2/model.onnx`
- `go/models/all-MiniLM-L6-v2/config.json`
- Tokenizer files

**Supported models**:
- `sentence-transformers/all-MiniLM-L6-v2` (default, 384 dims, recommended)
- `sentence-transformers/all-mpnet-base-v2` (768 dims, higher quality)
- `BAAI/bge-small-en-v1.5` (384 dims, good balance)
- `BAAI/bge-base-en-v1.5` (768 dims, best quality)

To export a different model:

```bash
python scripts/export_model_to_onnx.py \
  "sentence-transformers/all-mpnet-base-v2" \
  "go/models/all-mpnet-base-v2"
```

**This step is also cached** - only run once per model.

### Step 3: Build All Distributions

Build for all platforms:

```bash
./scripts/build-all-distributions.sh
```

This script:
1. ✅ Verifies prerequisites (Go, model files, ONNX libraries)
2. ✅ Cross-compiles Go binaries for all platforms
3. ✅ Packages each binary with:
   - Platform-specific ONNX Runtime library
   - ONNX model files (shared across all platforms)
   - Launcher scripts
   - Documentation
4. ✅ Creates compressed tarballs in `dist/`

**Output**:
```
dist/
├── go-doc-go-worker-darwin-x86_64.tar.gz   # macOS (Intel & Apple Silicon)
├── go-doc-go-worker-linux-x86_64.tar.gz    # Linux x86_64
├── go-doc-go-worker-linux-arm64.tar.gz     # Linux ARM64
└── go-doc-go-worker-windows-x86_64.tar.gz  # Windows x86_64
```

**Build time**: ~2-5 minutes (depending on machine)

## Building for a Single Platform

To build just one platform:

```bash
# macOS only
TARGET_PLATFORM=darwin TARGET_ARCH=x86_64 ./scripts/build-worker-dist.sh

# Linux x86_64 only
TARGET_PLATFORM=linux TARGET_ARCH=x86_64 ./scripts/build-worker-dist.sh

# Linux ARM64 only
TARGET_PLATFORM=linux TARGET_ARCH=arm64 ./scripts/build-worker-dist.sh

# Windows only
TARGET_PLATFORM=windows TARGET_ARCH=x86_64 ./scripts/build-worker-dist.sh
```

## Deployment

### On Target Machine

```bash
# 1. Extract distribution
tar -xzf go-doc-go-worker-linux-x86_64.tar.gz
cd worker-linux-x86_64

# 2. Create configuration
cp config.example.toml config.toml
# Edit config.toml with your settings

# 3. Run worker
./run-worker.sh --config config.toml --workers 4
```

### Example Deployment Configurations

#### Single Server

```bash
# Process 10,000 documents with 8 concurrent workers
./run-worker.sh --config config.toml --workers 8 --max-documents 10000
```

#### Distributed (Multiple Servers)

```toml
# config.toml (shared via NFS or config management)
[processing.job_control]
backend = "postgres"
path = "postgres://user:pass@db.example.com/godocgo"
```

```bash
# Server 1
./run-worker.sh --config config.toml --worker-id "server-01" --workers 8

# Server 2
./run-worker.sh --config config.toml --worker-id "server-02" --workers 8

# Server 3
./run-worker.sh --config config.toml --worker-id "server-03" --workers 8
```

#### Docker

```dockerfile
FROM ubuntu:22.04

# Copy distribution
COPY go-doc-go-worker-linux-x86_64.tar.gz /tmp/
RUN cd /opt && tar -xzf /tmp/go-doc-go-worker-linux-x86_64.tar.gz && \
    mv worker-linux-x86_64 godocgo && \
    rm /tmp/go-doc-go-worker-linux-x86_64.tar.gz

WORKDIR /opt/godocgo

# Config will be mounted as volume
ENTRYPOINT ["./run-worker.sh"]
CMD ["--config", "/config/config.toml"]
```

```bash
# Build and run
docker build -t godocgo/worker:latest .
docker run -v $(pwd)/config.toml:/config/config.toml godocgo/worker:latest
```

## Platform-Specific Notes

### macOS

**Universal Binary**: Works on both Intel and Apple Silicon
**Hardware Acceleration**: CoreML (GPU/Neural Engine) automatically enabled
**Library**: `libonnxruntime.dylib` (26MB)

### Linux x86_64

**Compatible with**: Ubuntu, Debian, RHEL, CentOS, Amazon Linux
**Hardware Acceleration**: CPU optimizations (AVX2, etc.)
**Library**: `libonnxruntime.so` (26MB)

### Linux ARM64

**Compatible with**: AWS Graviton, Raspberry Pi 4+, ARM servers
**Hardware Acceleration**: CPU optimizations (NEON)
**Library**: `libonnxruntime.so` (26MB)

### Windows x86_64

**Compatible with**: Windows 10+, Windows Server 2016+
**Hardware Acceleration**: CPU optimizations (AVX2)
**Library**: `onnxruntime.dll` (26MB)

**Run on Windows**:
```powershell
# PowerShell
.\worker.exe --config config.toml --workers 4

# Or use the launcher script in Git Bash/WSL
bash run-worker.sh --config config.toml --workers 4
```

## Troubleshooting

### "ONNX Runtime library not found"

The ONNX Runtime library is bundled with the distribution. If you see this error:

1. Check the distribution was extracted completely
2. Verify `libonnxruntime.{dylib,so,dll}` exists in the directory
3. Use the launcher script (`run-worker.sh`) which sets up paths automatically

### "ONNX model not found"

The model files should be in `models/all-MiniLM-L6-v2/`. If missing:

1. Rebuild distribution after running `python scripts/export_model_to_onnx.py`
2. Or manually copy model directory to target machine
3. Update `embedding.model_path` in config.toml

### Disable Embeddings

If you don't need embeddings or encounter issues:

```toml
[embedding]
enabled = false
```

Documents will still be parsed and stored, just without vector embeddings.

### Cross-Platform Build Errors

If cross-compilation fails:

```bash
# Install cross-compilation support
go env -w CGO_ENABLED=0  # Disable CGO for pure Go build

# Or install cross-compilation toolchains
# For Linux ARM64 from macOS:
brew install FiloSottile/musl-cross/musl-cross
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build Distributions

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.12'

      - name: Install dependencies
        run: pip install onnx sentence-transformers torch

      - name: Download ONNX Runtime
        run: ./scripts/download_onnx_runtime.sh

      - name: Export ONNX model
        run: python scripts/export_model_to_onnx.py

      - name: Build all distributions
        run: ./scripts/build-all-distributions.sh

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: distributions
          path: dist/*.tar.gz

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*.tar.gz
```

### GitLab CI

```yaml
build-distributions:
  stage: build
  image: golang:1.24
  script:
    - apt-get update && apt-get install -y python3-pip
    - pip3 install onnx sentence-transformers torch
    - ./scripts/download_onnx_runtime.sh
    - python3 scripts/export_model_to_onnx.py
    - ./scripts/build-all-distributions.sh
  artifacts:
    paths:
      - dist/*.tar.gz
    expire_in: 1 week
  only:
    - tags
```

## Performance Benchmarks by Platform

| Platform | Architecture | Docs/sec (no embeddings) | Docs/sec (with embeddings) |
|----------|-------------|--------------------------|----------------------------|
| macOS M2 | ARM64 | 150-300 | 80-120 (CoreML) |
| macOS Intel | x86_64 | 100-200 | 50-80 |
| Linux (AWS c6i.2xlarge) | x86_64 | 120-250 | 60-100 |
| Linux (AWS c7g.2xlarge) | ARM64 | 140-280 | 70-110 |
| Windows (Azure D4s_v5) | x86_64 | 110-220 | 55-90 |

**Notes**:
- Benchmarks with 4 concurrent workers
- Document processing depends on document type and size
- Embedding performance includes contextual embedding generation

## Distribution Size Optimization

To reduce distribution size:

### Option 1: Exclude Model Files

Build without bundling the model (user provides it):

```bash
# Remove model before building
rm -rf go/models/all-MiniLM-L6-v2
./scripts/build-all-distributions.sh
# Distributions will be ~30MB instead of ~120MB
```

### Option 2: Use Smaller Model

Export a smaller model:

```bash
python scripts/export_model_to_onnx.py \
  "sentence-transformers/all-MiniLM-L6-v2" \
  "go/models/all-MiniLM-L6-v2"
# 87MB model

# vs larger, higher quality:
python scripts/export_model_to_onnx.py \
  "sentence-transformers/all-mpnet-base-v2" \
  "go/models/all-mpnet-base-v2"
# 420MB model
```

### Option 3: Separate Model Package

Distribute model separately:

```bash
# Create model-only tarball
tar -czf go-doc-go-models-all-MiniLM-L6-v2.tar.gz \
  go/models/all-MiniLM-L6-v2/

# User downloads both:
# - go-doc-go-worker-linux-x86_64.tar.gz (32MB)
# - go-doc-go-models-all-MiniLM-L6-v2.tar.gz (88MB)
```

## Maintenance

### Updating ONNX Runtime Version

1. Edit `scripts/download_onnx_runtime.sh`
2. Update `VERSION="1.23.0"` to new version
3. Run `./scripts/download_onnx_runtime.sh`
4. Rebuild distributions

### Updating Embedding Model

```bash
# Export new model
python scripts/export_model_to_onnx.py \
  "new-model/name" \
  "go/models/new-model-name"

# Update default in build scripts if needed
# Edit scripts/build-worker-dist.sh: MODEL_DIR="go/models/new-model-name"

# Rebuild distributions
./scripts/build-all-distributions.sh
```

### Adding New Platforms

To add support for new platforms (e.g., Linux RISC-V):

1. Add platform to `scripts/download_onnx_runtime.sh`
2. Add build step to `scripts/build-all-distributions.sh`
3. Update `scripts/build-worker-dist.sh` library search paths

## Support

For issues with:
- **Building**: Check Go installation, Python packages
- **Model export**: Ensure `onnx`, `torch`, `sentence-transformers` installed
- **Distribution**: Verify ONNX Runtime libraries downloaded
- **Deployment**: Check platform compatibility, library paths

**Documentation**: See `go/README.md` for detailed Go worker documentation
**Issues**: https://github.com/kennethstott/go-doc-go/issues
