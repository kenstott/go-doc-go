#!/bin/bash
# Download ONNX Runtime libraries for multiple platforms
# This script downloads pre-built ONNX Runtime libraries directly from GitHub releases

set -e

VERSION="1.23.0"
BASE_URL="https://github.com/microsoft/onnxruntime/releases/download/v${VERSION}"
OUTPUT_DIR="lib/onnxruntime"

echo "========================================="
echo "Downloading ONNX Runtime ${VERSION}"
echo "for multiple platforms"
echo "========================================="
echo ""

mkdir -p "$OUTPUT_DIR"

# Download for macOS (Universal - works on Intel & Apple Silicon)
echo "Downloading macOS (Universal)..."
MACOS_FILE="onnxruntime-osx-universal2-${VERSION}.tgz"
if [ ! -f "$OUTPUT_DIR/$MACOS_FILE" ]; then
    curl -L "$BASE_URL/$MACOS_FILE" -o "$OUTPUT_DIR/$MACOS_FILE"
    echo "✓ Downloaded $MACOS_FILE"
else
    echo "✓ Already exists: $MACOS_FILE"
fi

# Extract macOS library
echo "  Extracting macOS library..."
tar -xzf "$OUTPUT_DIR/$MACOS_FILE" -C "$OUTPUT_DIR" \
    "onnxruntime-osx-universal2-${VERSION}/lib/libonnxruntime.${VERSION}.dylib" \
    --strip-components=2
mv "$OUTPUT_DIR/libonnxruntime.${VERSION}.dylib" "$OUTPUT_DIR/libonnxruntime-macos.dylib"
echo "✓ Extracted to: $OUTPUT_DIR/libonnxruntime-macos.dylib"
echo ""

# Download for Linux x86_64
echo "Downloading Linux x86_64..."
LINUX_X64_FILE="onnxruntime-linux-x64-${VERSION}.tgz"
if [ ! -f "$OUTPUT_DIR/$LINUX_X64_FILE" ]; then
    curl -L "$BASE_URL/$LINUX_X64_FILE" -o "$OUTPUT_DIR/$LINUX_X64_FILE"
    echo "✓ Downloaded $LINUX_X64_FILE"
else
    echo "✓ Already exists: $LINUX_X64_FILE"
fi

# Extract Linux x86_64 library
echo "  Extracting Linux x86_64 library..."
tar -xzf "$OUTPUT_DIR/$LINUX_X64_FILE" -C "$OUTPUT_DIR" \
    "onnxruntime-linux-x64-${VERSION}/lib/libonnxruntime.so.${VERSION}" \
    --strip-components=2
mv "$OUTPUT_DIR/libonnxruntime.so.${VERSION}" "$OUTPUT_DIR/libonnxruntime-linux-x64.so"
echo "✓ Extracted to: $OUTPUT_DIR/libonnxruntime-linux-x64.so"
echo ""

# Download for Linux ARM64
echo "Downloading Linux ARM64..."
LINUX_ARM64_FILE="onnxruntime-linux-aarch64-${VERSION}.tgz"
if [ ! -f "$OUTPUT_DIR/$LINUX_ARM64_FILE" ]; then
    curl -L "$BASE_URL/$LINUX_ARM64_FILE" -o "$OUTPUT_DIR/$LINUX_ARM64_FILE"
    echo "✓ Downloaded $LINUX_ARM64_FILE"
else
    echo "✓ Already exists: $LINUX_ARM64_FILE"
fi

# Extract Linux ARM64 library
echo "  Extracting Linux ARM64 library..."
tar -xzf "$OUTPUT_DIR/$LINUX_ARM64_FILE" -C "$OUTPUT_DIR" \
    "onnxruntime-linux-aarch64-${VERSION}/lib/libonnxruntime.so.${VERSION}" \
    --strip-components=2
mv "$OUTPUT_DIR/libonnxruntime.so.${VERSION}" "$OUTPUT_DIR/libonnxruntime-linux-arm64.so"
echo "✓ Extracted to: $OUTPUT_DIR/libonnxruntime-linux-arm64.so"
echo ""

# Download for Windows x86_64
echo "Downloading Windows x86_64..."
WINDOWS_FILE="onnxruntime-win-x64-${VERSION}.zip"
if [ ! -f "$OUTPUT_DIR/$WINDOWS_FILE" ]; then
    curl -L "$BASE_URL/$WINDOWS_FILE" -o "$OUTPUT_DIR/$WINDOWS_FILE"
    echo "✓ Downloaded $WINDOWS_FILE"
else
    echo "✓ Already exists: $WINDOWS_FILE"
fi

# Extract Windows library
echo "  Extracting Windows x86_64 library..."
unzip -q "$OUTPUT_DIR/$WINDOWS_FILE" \
    "onnxruntime-win-x64-${VERSION}/lib/onnxruntime.dll" \
    -d "$OUTPUT_DIR"
mv "$OUTPUT_DIR/onnxruntime-win-x64-${VERSION}/lib/onnxruntime.dll" "$OUTPUT_DIR/onnxruntime-windows-x64.dll"
rm -rf "$OUTPUT_DIR/onnxruntime-win-x64-${VERSION}"
echo "✓ Extracted to: $OUTPUT_DIR/onnxruntime-windows-x64.dll"
echo ""

# Clean up archives (optional - keep them for caching)
# echo "Cleaning up archives..."
# rm -f "$OUTPUT_DIR"/*.tgz "$OUTPUT_DIR"/*.zip

echo "========================================="
echo "✓ All ONNX Runtime libraries downloaded"
echo "========================================="
echo ""
echo "Downloaded libraries:"
ls -lh "$OUTPUT_DIR"/*.dylib "$OUTPUT_DIR"/*.so "$OUTPUT_DIR"/*.dll 2>/dev/null || true
echo ""
echo "Library sizes:"
du -sh "$OUTPUT_DIR"/*.dylib "$OUTPUT_DIR"/*.so "$OUTPUT_DIR"/*.dll 2>/dev/null || true
echo ""
echo "Total size:"
du -sh "$OUTPUT_DIR"
echo ""
echo "These libraries can now be bundled with distributions:"
echo "  - macOS (Intel & Apple Silicon): libonnxruntime-macos.dylib"
echo "  - Linux x86_64: libonnxruntime-linux-x64.so"
echo "  - Linux ARM64: libonnxruntime-linux-arm64.so"
echo "  - Windows x86_64: onnxruntime-windows-x64.dll"
