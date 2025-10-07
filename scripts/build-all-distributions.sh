#!/bin/bash
# Build Go-Doc-Go worker distributions for all supported platforms
# This script can be run from any development machine to create distributions
# for macOS, Linux (x86_64 & ARM64), and Windows

set -e

echo "========================================="
echo "Building Go-Doc-Go Worker Distributions"
echo "for ALL Platforms"
echo "========================================="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed. Install from: https://golang.org/dl/"
    exit 1
fi
echo "✓ Go installed: $(go version)"

if [ ! -d "go/models/all-MiniLM-L6-v2" ] || [ ! -f "go/models/all-MiniLM-L6-v2/model.onnx" ]; then
    echo ""
    echo "WARNING: ONNX model not found at go/models/all-MiniLM-L6-v2"
    echo "Running model export script..."
    python scripts/export_model_to_onnx.py
    if [ $? -ne 0 ]; then
        echo "ERROR: Model export failed"
        exit 1
    fi
fi
echo "✓ ONNX model available"

# Download ONNX Runtime libraries for all platforms if not present
if [ ! -d "lib/onnxruntime" ] || [ ! -f "lib/onnxruntime/libonnxruntime-macos.dylib" ]; then
    echo ""
    echo "Downloading ONNX Runtime libraries for all platforms..."
    ./scripts/download_onnx_runtime.sh
    if [ $? -ne 0 ]; then
        echo "ERROR: ONNX Runtime download failed"
        exit 1
    fi
fi
echo "✓ ONNX Runtime libraries available"

echo ""
echo "========================================="
echo "Building for all platforms..."
echo "========================================="
echo ""

# Create dist directory
mkdir -p dist

# Array to track built distributions
DISTRIBUTIONS=()

# Build for macOS (Universal - works on Intel & Apple Silicon)
echo "1. Building for macOS (Universal: Intel + Apple Silicon)..."
echo "   GOOS=darwin GOARCH=amd64"
cd go
GOOS=darwin GOARCH=amd64 go build -o bin/worker-darwin ./cmd/worker
cd ..
TARGET_PLATFORM=darwin TARGET_ARCH=x86_64 ./scripts/build-worker-dist.sh
DISTRIBUTIONS+=("dist/go-doc-go-worker-darwin-x86_64.tar.gz")
echo "✓ macOS distribution built"
echo ""

# Build for Linux x86_64
echo "2. Building for Linux x86_64..."
echo "   GOOS=linux GOARCH=amd64"
cd go
GOOS=linux GOARCH=amd64 go build -o bin/worker-linux-x64 ./cmd/worker
cd ..
TARGET_PLATFORM=linux TARGET_ARCH=x86_64 ./scripts/build-worker-dist.sh
DISTRIBUTIONS+=("dist/go-doc-go-worker-linux-x86_64.tar.gz")
echo "✓ Linux x86_64 distribution built"
echo ""

# Build for Linux ARM64 (AWS Graviton, Raspberry Pi, etc.)
echo "3. Building for Linux ARM64..."
echo "   GOOS=linux GOARCH=arm64"
cd go
GOOS=linux GOARCH=arm64 go build -o bin/worker-linux-arm64 ./cmd/worker
cd ..
TARGET_PLATFORM=linux TARGET_ARCH=arm64 ./scripts/build-worker-dist.sh
DISTRIBUTIONS+=("dist/go-doc-go-worker-linux-arm64.tar.gz")
echo "✓ Linux ARM64 distribution built"
echo ""

# Build for Windows x86_64
echo "4. Building for Windows x86_64..."
echo "   GOOS=windows GOARCH=amd64"
cd go
GOOS=windows GOARCH=amd64 go build -o bin/worker-windows.exe ./cmd/worker
cd ..
TARGET_PLATFORM=windows TARGET_ARCH=x86_64 ./scripts/build-worker-dist.sh
DISTRIBUTIONS+=("dist/go-doc-go-worker-windows-x86_64.tar.gz")
echo "✓ Windows x86_64 distribution built"
echo ""

echo "========================================="
echo "✓ ALL DISTRIBUTIONS BUILT SUCCESSFULLY"
echo "========================================="
echo ""
echo "Created distributions:"
for dist in "${DISTRIBUTIONS[@]}"; do
    if [ -f "$dist" ]; then
        SIZE=$(du -h "$dist" | cut -f1)
        echo "  - $dist ($SIZE)"
    fi
done

echo ""
echo "Distribution contents:"
ls -lh dist/*.tar.gz

echo ""
echo "Total distribution size:"
du -sh dist/

echo ""
echo "========================================="
echo "Distribution Details"
echo "========================================="
echo ""
echo "Each distribution includes:"
echo "  - Worker binary (compiled Go executable)"
echo "  - ONNX Runtime library (platform-specific)"
echo "  - ONNX model: all-MiniLM-L6-v2 (384 dims, ~88MB)"
echo "  - Launcher script (sets up environment)"
echo "  - Example configuration"
echo "  - Documentation"
echo ""
echo "Supported platforms:"
echo "  ✓ macOS (Intel & Apple Silicon)"
echo "  ✓ Linux x86_64 (Ubuntu, Debian, RHEL, etc.)"
echo "  ✓ Linux ARM64 (AWS Graviton, Raspberry Pi 4+, etc.)"
echo "  ✓ Windows x86_64"
echo ""
echo "To deploy:"
echo "  1. Copy tarball to target machine"
echo "  2. Extract: tar -xzf go-doc-go-worker-{platform}-{arch}.tar.gz"
echo "  3. Configure: cp config.example.toml config.toml && edit config.toml"
echo "  4. Run: ./run-worker.sh --config config.toml"
echo ""
echo "Hardware acceleration:"
echo "  - macOS: CoreML (Apple Silicon GPU/Neural Engine)"
echo "  - Linux: CPU optimizations (AVX2, etc.)"
echo "  - Windows: CPU optimizations"
echo ""
