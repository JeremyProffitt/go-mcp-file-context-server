#!/bin/bash

# Build script for go-mcp-file-context-server
# Creates binaries for all supported platforms including macOS universal binary

set -e

APP_NAME="go-mcp-file-context-server"
OUTPUT_DIR="bin"
VERSION=$(grep 'Version.*=' main.go | head -1 | sed 's/.*"\(.*\)".*/\1/')

echo "Building $APP_NAME v$VERSION"
echo "================================"

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build function
build() {
    local os=$1
    local arch=$2
    local output=$3

    echo "Building for $os/$arch..."
    GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "$OUTPUT_DIR/$output" .

    if [ $? -eq 0 ]; then
        echo "  -> $OUTPUT_DIR/$output"
    else
        echo "  -> FAILED"
        exit 1
    fi
}

# macOS builds (for universal binary)
echo ""
echo "Building macOS binaries..."
build darwin arm64 "${APP_NAME}-darwin-arm64"
build darwin amd64 "${APP_NAME}-darwin-amd64"

# Create universal binary (requires macOS with lipo)
if command -v lipo &> /dev/null; then
    echo "Creating macOS universal binary..."
    lipo -create -output "$OUTPUT_DIR/${APP_NAME}-darwin-universal" \
        "$OUTPUT_DIR/${APP_NAME}-darwin-arm64" \
        "$OUTPUT_DIR/${APP_NAME}-darwin-amd64"
    echo "  -> $OUTPUT_DIR/${APP_NAME}-darwin-universal"
else
    echo "Note: lipo not available, skipping universal binary creation"
    echo "      (Universal binaries can only be created on macOS)"
fi

# Linux builds
echo ""
echo "Building Linux binaries..."
build linux amd64 "${APP_NAME}-linux-amd64"
build linux arm64 "${APP_NAME}-linux-arm64"

# Windows builds
echo ""
echo "Building Windows binaries..."
build windows amd64 "${APP_NAME}-windows-amd64.exe"

# Generate checksums
echo ""
echo "Generating checksums..."
cd "$OUTPUT_DIR"
if command -v sha256sum &> /dev/null; then
    sha256sum * > checksums.txt
elif command -v shasum &> /dev/null; then
    shasum -a 256 * > checksums.txt
else
    echo "Note: sha256sum/shasum not available, skipping checksum generation"
fi
cd ..

echo ""
echo "================================"
echo "Build complete!"
echo ""
echo "Output directory: $OUTPUT_DIR/"
ls -la "$OUTPUT_DIR/"
