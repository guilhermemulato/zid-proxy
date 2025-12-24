#!/bin/bash
# ZID Agent - Linux Uninstallation Script

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BOLD}============================================${NC}"
echo -e "${BOLD}ZID Agent - Linux Uninstaller${NC}"
echo -e "${BOLD}============================================${NC}"
echo ""

INSTALL_DIR="/usr/local/bin"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
AUTOSTART_DIR="$HOME/.config/autostart"

echo "This will remove the ZID Agent from your system."
echo ""
read -p "Are you sure you want to uninstall? (y/N): " CONFIRM

if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    echo "Uninstall cancelled."
    exit 0
fi

echo ""

# Stop and disable systemd service if exists
if [ -f "$SYSTEMD_USER_DIR/zid-agent.service" ]; then
    echo -e "${YELLOW}Stopping systemd service...${NC}"
    systemctl --user stop zid-agent.service 2>/dev/null || true
    systemctl --user disable zid-agent.service 2>/dev/null || true
    rm -f "$SYSTEMD_USER_DIR/zid-agent.service"
    systemctl --user daemon-reload
    echo "Systemd service removed."
fi

# Remove XDG autostart file if exists
if [ -f "$AUTOSTART_DIR/zid-agent.desktop" ]; then
    echo -e "${YELLOW}Removing autostart entry...${NC}"
    rm -f "$AUTOSTART_DIR/zid-agent.desktop"
    echo "Autostart entry removed."
fi

# Kill any running agent processes
echo -e "${YELLOW}Stopping running agent processes...${NC}"
pkill -f zid-agent || echo "No running agent processes found."

# Remove binary
if [ -f "$INSTALL_DIR/zid-agent" ]; then
    echo -e "${YELLOW}Removing binary...${NC}"
    if [ "$EUID" -eq 0 ]; then
        rm -f "$INSTALL_DIR/zid-agent"
    else
        sudo rm -f "$INSTALL_DIR/zid-agent"
    fi
    echo "Binary removed."
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}Uninstall Complete!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo "ZID Agent has been removed from your system."
echo ""
