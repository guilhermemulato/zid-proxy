#!/bin/bash
# ZID Agent - Linux Installation Script
# Installs the ZID Agent as a systemd user service

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BOLD}============================================${NC}"
echo -e "${BOLD}ZID Agent - Linux Installer${NC}"
echo -e "${BOLD}============================================${NC}"
echo ""

# Check if binary exists
BINARY="zid-agent-linux-gui"
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}ERROR: $BINARY not found in current directory.${NC}"
    echo "Please run this script from the extracted agent folder."
    exit 1
fi

# Installation directories
INSTALL_DIR="/usr/local/bin"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
AUTOSTART_DIR="$HOME/.config/autostart"

echo -e "Installation method:"
echo "  1) Systemd user service (recommended)"
echo "  2) XDG autostart (alternative)"
echo ""
read -p "Choose installation method (1 or 2): " METHOD

if [ "$METHOD" = "1" ]; then
    # Systemd installation
    echo ""
    echo -e "${GREEN}Installing as systemd user service...${NC}"

    # Copy binary
    echo "Installing binary to $INSTALL_DIR..."
    if [ "$EUID" -eq 0 ]; then
        cp "$BINARY" "$INSTALL_DIR/zid-agent"
        chmod +x "$INSTALL_DIR/zid-agent"
    else
        sudo cp "$BINARY" "$INSTALL_DIR/zid-agent"
        sudo chmod +x "$INSTALL_DIR/zid-agent"
    fi

    # Create systemd user directory
    mkdir -p "$SYSTEMD_USER_DIR"

    # Create systemd service file
    cat > "$SYSTEMD_USER_DIR/zid-agent.service" <<'EOF'
[Unit]
Description=ZID Agent - Network Monitoring
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/zid-agent
Restart=always
RestartSec=30

[Install]
WantedBy=default.target
EOF

    # Reload systemd and enable service
    systemctl --user daemon-reload
    systemctl --user enable --now zid-agent.service

    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo ""
    echo "The ZID Agent has been installed as a systemd user service."
    echo ""
    echo "By default, systemd user services start when you log in."
    echo "If you want it to also start when Linux boots (even before login), enable 'linger'."
    echo ""

    read -p "Enable auto-start on boot with systemd linger? (y/N): " ENABLE_LINGER
    if [ "$ENABLE_LINGER" = "y" ] || [ "$ENABLE_LINGER" = "Y" ]; then
        if command -v loginctl >/dev/null 2>&1; then
            echo ""
            echo "Enabling linger for user $USER..."
            if [ "$EUID" -eq 0 ]; then
                loginctl enable-linger "$USER" || true
            else
                sudo loginctl enable-linger "$USER" || true
            fi
            echo -e "${GREEN}Linger enabled.${NC}"
            echo -e "${YELLOW}Note:${NC} tray icon will only show after you log in to the desktop session."
        else
            echo -e "${YELLOW}WARN: loginctl not found. Skipping linger.${NC}"
        fi
    fi
    echo ""
    echo "To check status:"
    echo -e "  ${BOLD}systemctl --user status zid-agent${NC}"
    echo ""
    echo "To view logs:"
    echo -e "  ${BOLD}journalctl --user -u zid-agent -f${NC}"
    echo ""
    echo -e "${GREEN}Agent started! Look for the ZID icon in your system tray.${NC}"

elif [ "$METHOD" = "2" ]; then
    # XDG autostart installation
    echo ""
    echo -e "${GREEN}Installing with XDG autostart...${NC}"

    # Copy binary
    echo "Installing binary to $INSTALL_DIR..."
    if [ "$EUID" -eq 0 ]; then
        cp "$BINARY" "$INSTALL_DIR/zid-agent"
        chmod +x "$INSTALL_DIR/zid-agent"
    else
        sudo cp "$BINARY" "$INSTALL_DIR/zid-agent"
        sudo chmod +x "$INSTALL_DIR/zid-agent"
    fi

    # Create autostart directory
    mkdir -p "$AUTOSTART_DIR"

    # Create desktop file
    cat > "$AUTOSTART_DIR/zid-agent.desktop" <<'EOF'
[Desktop Entry]
Type=Application
Name=ZID Agent
Comment=Network monitoring agent for ZID Proxy
Exec=/usr/local/bin/zid-agent
Icon=network-wired
Terminal=false
Categories=Network;System;
X-GNOME-Autostart-enabled=true
EOF

    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo ""
    echo "The ZID Agent will start automatically on next login."
    echo ""
    echo "To start the agent now:"
    echo -e "  ${BOLD}/usr/local/bin/zid-agent &${NC}"
    echo ""

    read -p "Start ZID Agent now? (Y/n): " START_NOW
    if [ "$START_NOW" != "n" ] && [ "$START_NOW" != "N" ]; then
        /usr/local/bin/zid-agent &
        echo ""
        echo -e "${GREEN}Agent started! Look for the ZID icon in your system tray.${NC}"
    fi

else
    echo -e "${RED}Invalid choice. Installation cancelled.${NC}"
    exit 1
fi

echo ""
