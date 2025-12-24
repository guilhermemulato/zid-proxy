#!/bin/bash
# Creates placeholder binaries for bundle generation when real compilation isn't possible
# These are NOT functional - just for structure demonstration

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p build

echo "Creating placeholder binaries for bundle demonstration..."
echo ""
echo "⚠️  WARNING: These are NOT functional binaries!"
echo "   They are placeholders to demonstrate bundle structure."
echo "   Real binaries require compilation with system dependencies."
echo ""

# Create placeholder Linux binary
cat > build/zid-agent-linux-gui << 'EOF'
#!/bin/bash
echo "=================================================="
echo "ZID Agent GUI - Placeholder Binary"
echo "=================================================="
echo ""
echo "This is a PLACEHOLDER binary for demonstration purposes."
echo ""
echo "To compile the real GUI agent, you need:"
echo "  1. System dependencies (see BUILD-AGENT.md)"
echo "  2. Run: make build-agent-linux-gui"
echo "  OR use Docker: ./scripts/build-gui-docker.sh"
echo ""
echo "For more info: BUILD-AGENT.md"
echo ""
exit 1
EOF
chmod +x build/zid-agent-linux-gui

# Create placeholder Windows binary (batch file)
cat > build/zid-agent-windows-gui.exe << 'EOF'
@echo off
echo ==================================================
echo ZID Agent GUI - Placeholder Binary
echo ==================================================
echo.
echo This is a PLACEHOLDER binary for demonstration purposes.
echo.
echo To compile the real GUI agent, you need:
echo   1. MinGW cross-compiler or Docker
echo   2. Run: make build-agent-windows-gui
echo   OR use Docker: scripts/build-gui-docker.sh
echo.
echo For more info: BUILD-AGENT.md
echo.
pause
exit /b 1
EOF

echo "✓ Created: build/zid-agent-linux-gui (placeholder)"
echo "✓ Created: build/zid-agent-windows-gui.exe (placeholder)"
echo ""
echo "These binaries will show an error message when executed."
echo "To create real binaries, see BUILD-AGENT.md"
