#!/bin/bash
# Build worker distribution with ONNX Runtime bundled
#
# Usage:
#   ./build-worker-dist.sh              - Build tarball distribution
#   ./build-worker-dist.sh --docker     - Build tarball + Docker image
#   BUILD_DOCKER=true ./build-worker-dist.sh  - Build tarball + Docker image
#   SAVE_DOCKER_TAR=true BUILD_DOCKER=true ./build-worker-dist.sh - Also save Docker image as .tar.gz
#
# Environment variables:
#   TARGET_PLATFORM - Override target platform (darwin, linux, windows)
#   TARGET_ARCH - Override target architecture (x86_64, arm64)
#   BUILD_DOCKER - Build Docker image (true/false)
#   SAVE_DOCKER_TAR - Save Docker image to tarball (true/false)
#
set -e

# Allow overriding target platform/arch
TARGET_PLATFORM="${TARGET_PLATFORM:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
TARGET_ARCH="${TARGET_ARCH:-$(uname -m)}"
PLATFORM="$TARGET_PLATFORM"
ARCH="$TARGET_ARCH"

# Normalize architecture names
case "$ARCH" in
    x86_64|amd64)
        ARCH="x86_64"
        ARCH_SHORT="x64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ARCH_SHORT="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "========================================="
echo "Building worker distribution"
echo "Platform: $PLATFORM/$ARCH"
echo "========================================="

# Determine ONNX Runtime library name and search paths
if [ "$PLATFORM" = "darwin" ]; then
    LIB_NAME="libonnxruntime.dylib"
    LIB_SEARCH_PATHS=(
        "lib/onnxruntime/libonnxruntime-macos.dylib"  # Pre-downloaded for all archs
        ".venv/lib/python3.12/site-packages/onnxruntime/capi/libonnxruntime.1.23.0.dylib"
        "$HOME/PycharmProjects/doculyzer-go-conversion/.venv/lib/python3.12/site-packages/onnxruntime/capi/libonnxruntime.1.23.0.dylib"
        "/opt/homebrew/lib/libonnxruntime.dylib"
        "$HOME/homebrew/lib/libonnxruntime.dylib"
        "/usr/local/lib/libonnxruntime.dylib"
    )
elif [ "$PLATFORM" = "linux" ]; then
    LIB_NAME="libonnxruntime.so"
    if [ "$ARCH" = "arm64" ]; then
        LIB_SEARCH_PATHS=(
            "lib/onnxruntime/libonnxruntime-linux-arm64.so"  # Pre-downloaded
            "/usr/lib/aarch64-linux-gnu/libonnxruntime.so"
            "/usr/local/lib/libonnxruntime.so"
            "$HOME/.local/lib/libonnxruntime.so"
        )
    else
        LIB_SEARCH_PATHS=(
            "lib/onnxruntime/libonnxruntime-linux-x64.so"  # Pre-downloaded
            "/usr/lib/x86_64-linux-gnu/libonnxruntime.so"
            "/usr/lib/libonnxruntime.so"
            "/usr/local/lib/libonnxruntime.so"
            "$HOME/.local/lib/libonnxruntime.so"
        )
    fi
elif [ "$PLATFORM" = "windows" ] || [ "$PLATFORM" = "mingw"* ] || [ "$PLATFORM" = "msys"* ]; then
    LIB_NAME="onnxruntime.dll"
    LIB_SEARCH_PATHS=(
        "lib/onnxruntime/onnxruntime-windows-x64.dll"  # Pre-downloaded
        "/c/Program Files/onnxruntime/bin/onnxruntime.dll"
        "$HOME/onnxruntime/bin/onnxruntime.dll"
    )
    PLATFORM="windows"
else
    echo "ERROR: Unsupported platform: $PLATFORM"
    exit 1
fi

# Find ONNX Runtime library
ONNX_LIB=""
if [ -n "$ONNXRUNTIME_SHARED_LIBRARY_PATH" ]; then
    if [ -f "$ONNXRUNTIME_SHARED_LIBRARY_PATH" ]; then
        ONNX_LIB="$ONNXRUNTIME_SHARED_LIBRARY_PATH"
        echo "Using ONNX Runtime from environment variable: $ONNX_LIB"
    fi
fi

if [ -z "$ONNX_LIB" ]; then
    echo "Searching for ONNX Runtime library..."
    for path in "${LIB_SEARCH_PATHS[@]}"; do
        if [ -f "$path" ]; then
            ONNX_LIB="$path"
            echo "Found ONNX Runtime: $ONNX_LIB"
            break
        fi
    done
fi

if [ -z "$ONNX_LIB" ]; then
    echo "WARNING: ONNX Runtime library not found!"
    echo "Searched in:"
    for path in "${LIB_SEARCH_PATHS[@]}"; do
        echo "  - $path"
    done
    echo ""
    echo "Distribution will be created without ONNX Runtime."
    echo "Embeddings will not work unless you manually add the library."
    echo ""
    read -p "Continue without ONNX Runtime? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Build worker (or use pre-built binary from build-all-distributions.sh)
WORKER_BINARY=""
POSSIBLE_BINARIES=(
    "go/bin/worker-${PLATFORM}-${ARCH_SHORT}"
    "go/bin/worker-${PLATFORM}"
    "go/bin/worker"
)

# Check for pre-built binary first
for binary in "${POSSIBLE_BINARIES[@]}"; do
    if [ -f "$binary" ]; then
        WORKER_BINARY="$binary"
        echo ""
        echo "Using pre-built worker binary: $WORKER_BINARY"
        break
    fi
done

# If no pre-built binary found, build it now
if [ -z "$WORKER_BINARY" ]; then
    echo ""
    echo "Building worker binary for $PLATFORM/$ARCH..."
    cd go
    mkdir -p bin

    # Set Go cross-compilation environment
    export GOOS="$PLATFORM"
    if [ "$PLATFORM" = "windows" ]; then
        export GOARCH="${ARCH_SHORT}"
        go build -o "bin/worker-${PLATFORM}.exe" ./cmd/worker
        WORKER_BINARY="../go/bin/worker-${PLATFORM}.exe"
    else
        export GOARCH="${ARCH_SHORT}"
        go build -o "bin/worker-${PLATFORM}-${ARCH_SHORT}" ./cmd/worker
        WORKER_BINARY="../go/bin/worker-${PLATFORM}-${ARCH_SHORT}"
    fi
    cd ..
    echo "✓ Built worker binary"
fi

# Create distribution directory
DIST_DIR="dist/worker-$PLATFORM-$ARCH"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "Creating distribution in: $DIST_DIR"

# Copy worker binary
if [ "$PLATFORM" = "windows" ]; then
    cp "$WORKER_BINARY" "$DIST_DIR/worker.exe"
    echo "✓ Copied worker binary: worker.exe"
else
    cp "$WORKER_BINARY" "$DIST_DIR/worker"
    chmod +x "$DIST_DIR/worker"
    echo "✓ Copied worker binary: worker"
fi

# Copy ONNX Runtime library if found
if [ -n "$ONNX_LIB" ]; then
    cp "$ONNX_LIB" "$DIST_DIR/$LIB_NAME"
    echo "✓ Copied ONNX Runtime library: $LIB_NAME"
fi

# Copy ONNX model files if they exist
MODEL_DIR="go/models/all-MiniLM-L6-v2"
if [ -d "$MODEL_DIR" ] && [ -f "$MODEL_DIR/model.onnx" ]; then
    echo ""
    echo "Copying ONNX model files..."
    mkdir -p "$DIST_DIR/models/all-MiniLM-L6-v2"
    cp -r "$MODEL_DIR"/* "$DIST_DIR/models/all-MiniLM-L6-v2/"

    # Check what was copied
    MODEL_SIZE=$(du -sh "$DIST_DIR/models/all-MiniLM-L6-v2" | cut -f1)
    echo "✓ Copied ONNX model: all-MiniLM-L6-v2 ($MODEL_SIZE)"
    echo "  - model.onnx"
    echo "  - config.json"
    echo "  - tokenizer files"
    INCLUDE_MODEL=true
else
    echo ""
    echo "⚠️  WARNING: ONNX model not found at $MODEL_DIR"
    echo "   To enable embeddings, run: python scripts/export_model_to_onnx.py"
    echo "   Distribution will work but embeddings will be disabled."
    INCLUDE_MODEL=false
fi

# Create wrapper script for macOS/Linux
cat > "$DIST_DIR/run-worker.sh" << 'EOF'
#!/bin/bash
# Worker launcher script - sets up environment and runs worker

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Detect platform
PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')

# Set ONNX Runtime library path
if [ "$PLATFORM" = "darwin" ]; then
    export ONNXRUNTIME_SHARED_LIBRARY_PATH="$SCRIPT_DIR/libonnxruntime.dylib"
    export DYLD_LIBRARY_PATH="$SCRIPT_DIR:$DYLD_LIBRARY_PATH"
elif [ "$PLATFORM" = "linux" ]; then
    export ONNXRUNTIME_SHARED_LIBRARY_PATH="$SCRIPT_DIR/libonnxruntime.so"
    export LD_LIBRARY_PATH="$SCRIPT_DIR:$LD_LIBRARY_PATH"
fi

# Check if library exists
if [ ! -f "$ONNXRUNTIME_SHARED_LIBRARY_PATH" ]; then
    echo "WARNING: ONNX Runtime library not found at: $ONNXRUNTIME_SHARED_LIBRARY_PATH"
    echo "Embeddings will not work. Set embedding.enabled = false in config to suppress this warning."
fi

# Run worker with all arguments passed through
exec "$SCRIPT_DIR/worker" "$@"
EOF

chmod +x "$DIST_DIR/run-worker.sh"
echo "✓ Created launcher script: run-worker.sh"

# Create README
cat > "$DIST_DIR/README.md" << EOF
# Go-Doc-Go Worker Distribution

Platform: $PLATFORM/$ARCH
Build Date: $(date)

## Contents

- \`worker\` - Worker binary (compiled Go executable)
- \`run-worker.sh\` - Launcher script (sets up environment)
- \`libonnxruntime.{dylib,so}\` - ONNX Runtime library (if available)
- \`models/all-MiniLM-L6-v2/\` - ONNX embedding model (if available, ~92MB)
  - \`model.onnx\` - ONNX model file
  - \`config.json\` - Model configuration
  - \`tokenizer.json\` - Tokenizer
  - \`vocab.txt\` - Vocabulary
- \`config.example.toml\` - Example configuration file
- \`README.md\` - This file

## Quick Start

1. Create a configuration file (see example below)
2. Run the worker:

   \`\`\`bash
   ./run-worker.sh --config config.toml
   \`\`\`

## Configuration Example

\`\`\`toml
# config.toml - minimal example

[processing.job_control]
backend = "sqlite"
path = "./data/jobs.db"
claim_timeout = 300
heartbeat_interval = 30
max_retries = 3

[[content_sources]]
name = "documents"
type = "file"
base_path = "./docs"
pattern = "**/*.pdf"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"

[embedding]
enabled = true  # Set to false if you don't need embeddings
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"  # Bundled with distribution
contextual = true
predecessor_count = 2
successor_count = 2
\`\`\`

## Embeddings

This distribution includes:
- **ONNX Runtime library** (\`libonnxruntime.{dylib,so}\`) for inference
- **Pre-exported ONNX model** (\`models/all-MiniLM-L6-v2/\`) ready to use

The worker will automatically use hardware acceleration if available:
- **macOS**: CoreML (Apple Silicon GPU/Neural Engine)
- **Linux**: CPU optimizations

### Running Without Embeddings

If you don't need embeddings, disable them in config:

\`\`\`toml
[embedding]
enabled = false
\`\`\`

Documents will be processed but embeddings will not be generated.

## Advanced Options

\`\`\`bash
# Set number of concurrent goroutine workers (default: 1)
./run-worker.sh --config config.toml --workers 4

# Set number of worker processes (multiprocessing)
./run-worker.sh --config config.toml --instances 2

# Limit number of documents to process
./run-worker.sh --config config.toml --max-documents 100

# Set custom worker ID
./run-worker.sh --config config.toml --worker-id "my-worker-1"
\`\`\`

## Environment Variables

- \`GO_DOC_GO_CONFIG_PATH\` - Default config file path
- \`NUM_WORKERS\` - Number of concurrent goroutine workers
- \`NUM_INSTANCES\` - Number of worker processes to spawn
- \`ONNXRUNTIME_SHARED_LIBRARY_PATH\` - Custom ONNX Runtime library path

## Support

For issues and documentation: https://github.com/kennethstott/go-doc-go
EOF

echo "✓ Created README.md"

# Create example config
cat > "$DIST_DIR/config.example.toml" << 'EOF'
# Go-Doc-Go Worker Configuration Example

[processing]
max_workers = 1  # Deprecated - use --workers CLI flag instead

[processing.job_control]
backend = "sqlite"  # or "postgres"
path = "./data/jobs.db"  # SQLite path or PostgreSQL connection string
claim_timeout = 300
heartbeat_interval = 30
max_retries = 3

# Content Sources - where to find documents
[[content_sources]]
name = "local_documents"
type = "file"
base_path = "./docs"
pattern = "**/*.{pdf,docx,xlsx,pptx,html,md,txt,json,csv,xml}"

# [[content_sources]]
# name = "s3_documents"
# type = "s3"
# bucket = "my-bucket"
# prefix = "documents/"
# region = "us-east-1"

# [[content_sources]]
# name = "web_documents"
# type = "web"
# base_url = "https://example.com/docs"
# pattern = "**/*.html"

# Analytics - where to store parsed data
[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"

# [[analytics.outputs]]
# type = "neo4j"
# uri = "bolt://localhost:7687"
# username = "neo4j"
# password = "password"
# database = "neo4j"

# Embeddings - vector embeddings for semantic search
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"  # Bundled ONNX model
contextual = true
predecessor_count = 2
successor_count = 2

# Neo4j Export - automatic export when queue is empty
# [processing.neo4j_export]
# enabled = false
# empty_queue_wait_time = 300  # seconds
# source_analytics = "parquet"
# batch_size = 1000
#
# [processing.neo4j_export.connection]
# uri = "bolt://localhost:7687"
# username = "neo4j"
# password = "password"
# database = "neo4j"
EOF

echo "✓ Created config.example.toml"

# Create tarball
TARBALL="dist/go-doc-go-worker-$PLATFORM-$ARCH.tar.gz"
echo ""
echo "Creating tarball: $TARBALL"
cd dist
tar -czf "go-doc-go-worker-$PLATFORM-$ARCH.tar.gz" "worker-$PLATFORM-$ARCH"
cd ..

# Print summary
echo ""
echo "========================================="
echo "✓ Distribution created successfully!"
echo "========================================="
echo ""
echo "Distribution directory: $DIST_DIR"
echo "Tarball: $TARBALL"
echo ""
echo "Contents:"
ls -lh "$DIST_DIR"
echo ""
echo "To use:"
echo "  1. Extract tarball on target machine"
echo "  2. Copy config.example.toml to config.toml and edit"
echo "  3. Run: ./run-worker.sh --config config.toml"
echo ""

if [ -z "$ONNX_LIB" ]; then
    echo "⚠️  WARNING: ONNX Runtime library was not included!"
    echo "   Embeddings will not work unless you:"
    echo "   - Add the library manually, or"
    echo "   - Set embedding.enabled = false in config"
    echo ""
fi

if [ "$INCLUDE_MODEL" = false ]; then
    echo "⚠️  WARNING: ONNX model files were not included!"
    echo "   To enable embeddings:"
    echo "   1. Export model: python scripts/export_model_to_onnx.py"
    echo "   2. Rebuild distribution with: ./scripts/build-worker-dist.sh"
    echo "   Or set embedding.enabled = false in config"
    echo ""
fi

if [ -n "$ONNX_LIB" ] && [ "$INCLUDE_MODEL" = true ]; then
    echo "✅ Full embedding support included!"
    echo "   - ONNX Runtime library: $LIB_NAME"
    echo "   - Model: all-MiniLM-L6-v2 (384 dimensions)"
    echo "   - Hardware acceleration enabled (CoreML on macOS)"
    echo ""
fi

# Create Docker image if requested
if [ "$BUILD_DOCKER" = "true" ] || [ "$1" = "--docker" ]; then
    echo "========================================="
    echo "Building Docker image"
    echo "========================================="

    # Create Dockerfile in dist directory
    cat > "$DIST_DIR/Dockerfile" << 'DOCKERFILE'
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy distribution files
COPY worker /usr/local/bin/worker
COPY libonnxruntime.* /usr/local/lib/ 2>/dev/null || true
COPY models/ /app/models/ 2>/dev/null || true
COPY config.example.toml /app/config.example.toml

# Set library path
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
ENV ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so

# Make worker executable
RUN chmod +x /usr/local/bin/worker

# Run ldconfig to register shared libraries
RUN ldconfig 2>/dev/null || true

# Default config path (can be overridden)
ENV GO_DOC_GO_CONFIG_PATH=/app/config.toml

ENTRYPOINT ["/usr/local/bin/worker"]
CMD ["--config", "/app/config.toml"]
DOCKERFILE

    echo "✓ Created Dockerfile"

    # Build Docker image
    IMAGE_TAG="go-doc-go/worker:${PLATFORM}-${ARCH}"
    IMAGE_TAG_LATEST="go-doc-go/worker:latest"

    echo ""
    echo "Building Docker image: $IMAGE_TAG"
    docker build -t "$IMAGE_TAG" -t "$IMAGE_TAG_LATEST" "$DIST_DIR"

    if [ $? -eq 0 ]; then
        echo ""
        echo "✅ Docker image built successfully!"
        echo "   Image: $IMAGE_TAG"
        echo "   Also tagged as: $IMAGE_TAG_LATEST"
        echo ""
        echo "Usage examples:"
        echo ""
        echo "  # Option 1: Mount config file (recommended)"
        echo "  docker run -v \$(pwd)/config.toml:/app/config.toml \\"
        echo "             -v \$(pwd)/docs:/app/docs \\"
        echo "             -v \$(pwd)/data:/app/data \\"
        echo "             $IMAGE_TAG_LATEST"
        echo ""
        echo "  # Option 2: Pass different config path"
        echo "  docker run -v \$(pwd)/my-config.toml:/etc/worker.toml \\"
        echo "             $IMAGE_TAG_LATEST --config /etc/worker.toml"
        echo ""
        echo "  # Option 3: Use environment variable"
        echo "  docker run -e GO_DOC_GO_CONFIG_PATH=/etc/worker.toml \\"
        echo "             -v \$(pwd)/config.toml:/etc/worker.toml \\"
        echo "             $IMAGE_TAG_LATEST"
        echo ""
        echo "  # Option 4: Override CLI flags"
        echo "  docker run -v \$(pwd)/config.toml:/app/config.toml \\"
        echo "             $IMAGE_TAG_LATEST --config /app/config.toml --workers 4 --instances 2"
        echo ""

        # Save Docker image to tarball if requested
        if [ "$SAVE_DOCKER_TAR" = "true" ]; then
            DOCKER_TAR="dist/go-doc-go-worker-${PLATFORM}-${ARCH}.docker.tar"
            echo "Saving Docker image to: $DOCKER_TAR"
            docker save "$IMAGE_TAG" | gzip > "$DOCKER_TAR.gz"
            echo "✓ Saved Docker image tarball: $DOCKER_TAR.gz"
            echo ""
            echo "To load on another machine:"
            echo "  gunzip -c $DOCKER_TAR.gz | docker load"
            echo ""
        fi
    else
        echo "❌ Docker image build failed"
        exit 1
    fi
fi

echo "========================================="
echo "Build complete!"
echo "========================================="