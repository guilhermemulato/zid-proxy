#!/bin/sh
#
# ZID Proxy pfSense Package Installation Script
#
# This script installs the ZID Proxy package files to a pfSense system.
# Run this script on the pfSense firewall after copying the package files.
#
# Usage: ./install.sh
#

set -e

echo "========================================="
echo " ZID Proxy pfSense Package Installer"
echo "========================================="

# Check if running as root
if [ "$(id -u)" != "0" ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Define paths
PREFIX="/usr/local"
PKG_DIR="$(dirname "$0")"
FILES_DIR="${PKG_DIR}/files"

echo ""
echo "Installing from: ${PKG_DIR}"
echo ""

# Create directories
echo "Creating directories..."
mkdir -p ${PREFIX}/pkg
mkdir -p ${PREFIX}/www
mkdir -p ${PREFIX}/etc/rc.d
mkdir -p ${PREFIX}/sbin
mkdir -p ${PREFIX}/etc/zid-proxy
mkdir -p ${PREFIX}/share/pfSense-pkg-zid-proxy
mkdir -p /etc/inc/priv
mkdir -p /var/log

# Install package configuration
echo "Installing package configuration..."
cp ${FILES_DIR}${PREFIX}/pkg/zid-proxy.xml ${PREFIX}/pkg/
cp ${FILES_DIR}${PREFIX}/pkg/zid-proxy.inc ${PREFIX}/pkg/

# Install web pages
echo "Installing web pages..."
cp ${FILES_DIR}${PREFIX}/www/zid-proxy_rules.php ${PREFIX}/www/
cp ${FILES_DIR}${PREFIX}/www/zid-proxy_log.php ${PREFIX}/www/

# Install privilege definitions
echo "Installing privilege definitions..."
cp ${FILES_DIR}/etc/inc/priv/zid-proxy.priv.inc /etc/inc/priv/

# Install package info
echo "Installing package info..."
cp ${FILES_DIR}${PREFIX}/share/pfSense-pkg-zid-proxy/info.xml ${PREFIX}/share/pfSense-pkg-zid-proxy/

# Check if binary exists in parent directory
BINARY_PATH="${PKG_DIR}/../build/zid-proxy"
if [ -f "${BINARY_PATH}" ]; then
    echo "Installing binary..."
    cp ${BINARY_PATH} ${PREFIX}/sbin/zid-proxy
    chmod 755 ${PREFIX}/sbin/zid-proxy
else
    echo "Warning: Binary not found at ${BINARY_PATH}"
    echo "         You need to copy the zid-proxy binary to ${PREFIX}/sbin/ manually"
fi

# Create default rules file
if [ ! -f ${PREFIX}/etc/zid-proxy/access_rules.txt ]; then
    echo "Creating default rules file..."
    cat > ${PREFIX}/etc/zid-proxy/access_rules.txt << 'EOF'
# ZID Proxy Access Rules
# Format: TYPE;IP_OR_CIDR;HOSTNAME
# TYPE: ALLOW or BLOCK
# ALLOW rules take priority over BLOCK rules
# Default action (no match): ALLOW

# Example rules:
# BLOCK;192.168.1.0/24;*.facebook.com
# ALLOW;192.168.1.100;*.facebook.com
EOF
fi

# Create log file
touch /var/log/zid-proxy.log
chmod 644 /var/log/zid-proxy.log

# Set permissions
echo "Setting permissions..."
chmod 644 ${PREFIX}/pkg/zid-proxy.xml
chmod 644 ${PREFIX}/pkg/zid-proxy.inc
chmod 644 ${PREFIX}/www/zid-proxy_rules.php
chmod 644 ${PREFIX}/www/zid-proxy_log.php
chmod 644 /etc/inc/priv/zid-proxy.priv.inc

echo ""
echo "========================================="
echo " File Installation Complete!"
echo "========================================="
echo ""

# Run activation script
SCRIPT_DIR=$(dirname "$0")

if [ -f "${SCRIPT_DIR}/activate-package.php" ]; then
    echo "Activating package (creating RC script)..."
    php "${SCRIPT_DIR}/activate-package.php"
    activation_result=$?
    echo ""

    if [ $activation_result -eq 0 ]; then
        # Register the package automatically
        echo "========================================="
        echo " Package Registration"
        echo "========================================="
        echo ""
        echo "Registering package with pfSense..."

        if [ -f "${SCRIPT_DIR}/register-package.php" ]; then
            php "${SCRIPT_DIR}/register-package.php"
            register_result=$?
            echo ""

            if [ $register_result -eq 0 ]; then
                echo "[OK] Package registered successfully"
                echo ""

                # Reload pfSense web GUI to make menu appear
                echo "Reloading pfSense web GUI..."
                /etc/rc.restart_webgui 2>/dev/null &
                reload_result=$?

                if [ $reload_result -eq 0 ]; then
                    echo "[OK] Web GUI reload initiated"
                    echo ""
                    echo "IMPORTANT: Wait ~10 seconds, then reload your browser (Ctrl+Shift+R) to see the new menu"
                else
                    echo "[WARNING] Web GUI reload may have failed"
                    echo "          Try manually: /etc/rc.restart_webgui"
                fi
                echo ""
            else
                echo "[ERROR] Package registration failed!"
                echo "        You can try running manually:"
                echo "        php ${SCRIPT_DIR}/register-package.php"
                echo ""
            fi
        else
            echo "[ERROR] register-package.php not found"
            echo ""
        fi
    else
        echo "Warning: Package activation failed!"
        echo "You may need to run 'php ${SCRIPT_DIR}/activate-package.php' manually."
        echo ""
    fi
else
    echo "Warning: activate-package.php not found"
    echo "You may need to manually create the RC script."
    echo ""
fi

echo "========================================="
echo " Installation Summary"
echo "========================================="
echo ""
echo "Files installed:"
echo "  • Binary: ${PREFIX}/sbin/zid-proxy"
echo "  • Package files: ${PREFIX}/pkg/zid-proxy.*"
echo "  • Web interface: ${PREFIX}/www/zid-proxy_*.php"
echo "  • RC script: ${PREFIX}/etc/rc.d/zid-proxy.sh"
echo ""
echo "Next steps:"
echo ""
echo "1. Test the service:"
echo "   /usr/local/etc/rc.d/zid-proxy.sh start"
echo "   /usr/local/etc/rc.d/zid-proxy.sh status"
echo ""
echo "2. Access pfSense web interface:"
echo "   - Navigate to Services > ZID Proxy"
echo "   - Enable the service and configure listen interface/port"
echo "   - Add access rules on the 'Access Rules' tab"
echo ""
echo "3. Configure firewall NAT:"
echo "   - Go to Firewall > NAT > Port Forward"
echo "   - Redirect HTTPS (443) traffic to the proxy port"
echo ""
echo "Troubleshooting:"
echo "  • Run diagnostics: sh ${SCRIPT_DIR}/diagnose.sh"
echo "  • View logs: tail -f /var/log/zid-proxy.log"
echo "  • Manual activation: php ${SCRIPT_DIR}/activate-package.php"
echo "  • Register package: php ${SCRIPT_DIR}/register-package.php"
echo ""
