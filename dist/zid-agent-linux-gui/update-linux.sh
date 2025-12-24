#!/bin/bash
# ZID Agent - Linux Update Script
# Updates an existing ZID Agent installation

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BOLD}============================================${NC}"
echo -e "${BOLD}ZID Agent - Linux Updater${NC}"
echo -e "${BOLD}============================================${NC}"
echo ""

# Check if binary exists
BINARY="zid-agent-linux-gui"
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}ERROR: $BINARY not found in current directory.${NC}"
    echo "Please run this script from the extracted agent folder."
    exit 1
fi

# Check if agent is installed
INSTALL_PATH="/usr/local/bin/zid-agent"
if [ ! -f "$INSTALL_PATH" ]; then
    echo -e "${RED}ERROR: ZID Agent is not installed at $INSTALL_PATH${NC}"
    echo "Please install the agent first using install-linux.sh"
    exit 1
fi

echo "Current installed version:"
if [ -x "$INSTALL_PATH" ]; then
    $INSTALL_PATH -version 2>/dev/null || echo "  (version check not available)"
fi

echo ""
echo "New version in this bundle:"
if [ -x "$BINARY" ]; then
    ./$BINARY -version 2>/dev/null || echo "  (version check not available)"
fi

echo ""
read -p "Continue with update? (Y/n): " CONTINUE
if [ "$CONTINUE" = "n" ] || [ "$CONTINUE" = "N" ]; then
    echo "Update cancelled."
    exit 0
fi

# Detect installation method
SYSTEMD_INSTALLED=false
XDG_INSTALLED=false

if systemctl --user is-enabled zid-agent.service &>/dev/null; then
    SYSTEMD_INSTALLED=true
    echo ""
    echo -e "${GREEN}Detected: Systemd user service installation${NC}"
fi

if [ -f "$HOME/.config/autostart/zid-agent.desktop" ]; then
    XDG_INSTALLED=true
    echo ""
    echo -e "${GREEN}Detected: XDG autostart installation${NC}"
fi

# Stop the agent
if [ "$SYSTEMD_INSTALLED" = true ]; then
    echo ""
    echo "Stopping systemd service..."
    systemctl --user stop zid-agent || true
fi

if [ "$XDG_INSTALLED" = true ]; then
    echo ""
    echo "Stopping running agent..."
    pkill -f zid-agent || true
    sleep 2
fi

# Update the binary
echo ""
echo "Updating binary..."
if [ "$EUID" -eq 0 ]; then
    cp "$BINARY" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"
else
    sudo cp "$BINARY" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
fi

echo -e "${GREEN}Binary updated successfully!${NC}"

# Restart the agent
if [ "$SYSTEMD_INSTALLED" = true ]; then
    echo ""
    echo "Restarting systemd service..."
    systemctl --user start zid-agent
    sleep 2

    echo ""
    echo "Service status:"
    systemctl --user status zid-agent --no-pager -l

elif [ "$XDG_INSTALLED" = true ]; then
    echo ""
    read -p "Start the updated agent now? (Y/n): " START_NOW
    if [ "$START_NOW" != "n" ] && [ "$START_NOW" != "N" ]; then
        /usr/local/bin/zid-agent &
        echo -e "${GREEN}Agent started! Look for the ZID icon in your system tray.${NC}"
    else
        echo "Agent will start on next login."
    fi
fi

echo ""
echo -e "${BOLD}============================================${NC}"
echo -e "${GREEN}Update complete!${NC}"
echo -e "${BOLD}============================================${NC}"
echo ""

if [ "$SYSTEMD_INSTALLED" = true ]; then
    echo "To view logs:"
    echo -e "  ${BOLD}journalctl --user -u zid-agent -f${NC}"
fi

echo ""
