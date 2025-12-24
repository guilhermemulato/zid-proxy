#!/bin/bash
# Build GUI agents using Docker (controlled environment with all dependencies)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

DOCKER_IMAGE="fyne-cross/base:latest"

echo "============================================"
echo "ZID Agent GUI - Docker Build"
echo "============================================"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker is not installed or not in PATH"
    echo ""
    echo "Install Docker:"
    echo "  Ubuntu/Debian: sudo apt-get install docker.io"
    echo "  Fedora: sudo dnf install docker"
    echo "  Or download from: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if fyne-cross is installed
if ! command -v fyne-cross &> /dev/null; then
    echo "Installing fyne-cross..."
    go install github.com/fyne-io/fyne-cross@latest
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

VERSION=$(grep '^VERSION=' Makefile | cut -d'=' -f2)
echo "Building version: $VERSION"
echo ""

# Build Linux
echo "Building Linux GUI agent..."
fyne-cross linux -arch=amd64 \
    -app-id com.soulsolucoes.zidagent \
    -name zid-agent-linux-gui \
    -ldflags "-s -w -X main.Version=$VERSION" \
    ./cmd/zid-agent

# Build Windows
echo ""
echo "Building Windows GUI agent..."
fyne-cross windows -arch=amd64 \
    -app-id com.soulsolucoes.zidagent \
    -name zid-agent-windows-gui \
    -ldflags "-s -w -H windowsgui -X main.Version=$VERSION" \
    ./cmd/zid-agent

echo ""
echo "============================================"
echo "Build Complete!"
echo "============================================"
echo ""

# Move binaries to build directory
mkdir -p build
if [ -f "fyne-cross/dist/linux-amd64/zid-agent-linux-gui" ]; then
    cp fyne-cross/dist/linux-amd64/zid-agent-linux-gui build/
    chmod +x build/zid-agent-linux-gui
    echo "✓ Linux binary: build/zid-agent-linux-gui"
fi

if [ -f "fyne-cross/dist/windows-amd64/zid-agent-windows-gui.exe" ]; then
    cp fyne-cross/dist/windows-amd64/zid-agent-windows-gui.exe build/
    echo "✓ Windows binary: build/zid-agent-windows-gui.exe"
fi

echo ""
echo "Next steps:"
echo "  1. Run: ./scripts/bundle-latest-gui.sh"
echo "  2. Distribute: zid-agent-*-gui-latest.tar.gz"
