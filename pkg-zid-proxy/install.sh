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
echo " Installation complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Navigate to Services > ZID Proxy in the pfSense web interface"
echo "2. Enable the service and configure the listen interface/port"
echo "3. Add access rules on the 'Access Rules' tab"
echo "4. Configure firewall NAT to redirect HTTPS traffic to the proxy"
echo ""
echo "If you haven't installed the binary yet:"
echo "  scp build/zid-proxy root@pfsense:${PREFIX}/sbin/"
echo ""
