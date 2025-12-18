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
cp ${FILES_DIR}${PREFIX}/pkg/zid-proxy_watchdog.php ${PREFIX}/pkg/ 2>/dev/null || true

# Install web pages
echo "Installing web pages..."
cp -f ${FILES_DIR}${PREFIX}/www/zid-proxy_settings.php ${PREFIX}/www/
cp -f ${FILES_DIR}${PREFIX}/www/zid-proxy_rules.php ${PREFIX}/www/
cp -f ${FILES_DIR}${PREFIX}/www/zid-proxy_log.php ${PREFIX}/www/
cp -f ${FILES_DIR}${PREFIX}/www/zid-proxy_groups.php ${PREFIX}/www/

# Install privilege definitions
echo "Installing privilege definitions..."
cp -f ${FILES_DIR}/etc/inc/priv/zid-proxy.priv.inc /etc/inc/priv/

# Install package info
echo "Installing package info..."
cp -f ${FILES_DIR}${PREFIX}/share/pfSense-pkg-zid-proxy/info.xml ${PREFIX}/share/pfSense-pkg-zid-proxy/

# Install updater helper (so future updates don't require manual tar/scp)
if [ -f "${PKG_DIR}/update-bootstrap.sh" ]; then
    echo "Installing updater helper..."
    # Avoid truncating a currently-running updater script (can cause odd errors).
    TMP_UPDATER="${PREFIX}/sbin/.zid-proxy-update.new.$$"
    cp "${PKG_DIR}/update-bootstrap.sh" "${TMP_UPDATER}"
    chmod 755 "${TMP_UPDATER}"
    mv -f "${TMP_UPDATER}" "${PREFIX}/sbin/zid-proxy-update"

    # Keep a copy alongside package info for reference (also atomic).
    TMP_UPDATER_INFO="${PREFIX}/share/pfSense-pkg-zid-proxy/.zid-proxy-update.new.$$"
    cp "${PKG_DIR}/update-bootstrap.sh" "${TMP_UPDATER_INFO}"
    chmod 755 "${TMP_UPDATER_INFO}"
    mv -f "${TMP_UPDATER_INFO}" "${PREFIX}/share/pfSense-pkg-zid-proxy/zid-proxy-update"
fi

# Install watchdog helper (used by cron)
if [ -f "${FILES_DIR}${PREFIX}/sbin/zid-proxy-watchdog" ]; then
    echo "Installing watchdog helper..."
    TMP_WD="${PREFIX}/sbin/.zid-proxy-watchdog.new.$$"
    cp "${FILES_DIR}${PREFIX}/sbin/zid-proxy-watchdog" "${TMP_WD}"
    chmod 755 "${TMP_WD}"
    mv -f "${TMP_WD}" "${PREFIX}/sbin/zid-proxy-watchdog"
fi

# Check if binary exists in parent directory
BINARY_PATH="${PKG_DIR}/../build/zid-proxy"
if [ -f "${BINARY_PATH}" ]; then
    echo "Installing binary..."
    # Avoid "Text file busy" by never writing in-place to an executing binary.
    # Copy to a temp file and atomically replace with mv.
    TMP_BIN="${PREFIX}/sbin/.zid-proxy.new.$$"
    cp "${BINARY_PATH}" "${TMP_BIN}"
    chmod 755 "${TMP_BIN}"
    mv -f "${TMP_BIN}" "${PREFIX}/sbin/zid-proxy"
    chmod 755 ${PREFIX}/sbin/zid-proxy
else
    echo "Warning: Binary not found at ${BINARY_PATH}"
    echo "         You need to copy the zid-proxy binary to ${PREFIX}/sbin/ manually"
fi

# Optional helper binary: zid-proxy-logrotate
LOGROTATE_BINARY_PATH="${PKG_DIR}/../build/zid-proxy-logrotate"
if [ -f "${LOGROTATE_BINARY_PATH}" ]; then
    echo "Installing logrotate binary..."
    TMP_BIN="${PREFIX}/sbin/.zid-proxy-logrotate.new.$$"
    cp "${LOGROTATE_BINARY_PATH}" "${TMP_BIN}"
    chmod 755 "${TMP_BIN}"
    mv -f "${TMP_BIN}" "${PREFIX}/sbin/zid-proxy-logrotate"
    chmod 755 ${PREFIX}/sbin/zid-proxy-logrotate
else
    echo "Warning: Logrotate binary not found at ${LOGROTATE_BINARY_PATH}"
    echo "         You can still use ZID Proxy without daily log rotation."
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
if [ -f ${PREFIX}/pkg/zid-proxy_watchdog.php ]; then
    chmod 644 ${PREFIX}/pkg/zid-proxy_watchdog.php
fi
chmod 644 ${PREFIX}/www/zid-proxy_settings.php
chmod 644 ${PREFIX}/www/zid-proxy_rules.php
chmod 644 ${PREFIX}/www/zid-proxy_log.php
chmod 644 ${PREFIX}/www/zid-proxy_groups.php
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
                echo "Reloading pfSense web GUI (to pick up updated PHP pages)..."
                if [ -x /usr/local/sbin/pfSsh.php ]; then
                    /usr/local/sbin/pfSsh.php playback reloadwebgui >/dev/null 2>&1 || true
                elif [ -x /etc/rc.restart_webgui ]; then
                    /etc/rc.restart_webgui >/dev/null 2>&1 || true
                elif [ -x /usr/local/etc/rc.d/php-fpm ]; then
                    /usr/local/etc/rc.d/php-fpm restart >/dev/null 2>&1 || true
                fi
                echo "[OK] Web GUI reload requested"
                echo ""
                echo "IMPORTANT: Wait ~10 seconds, then reload your browser (Ctrl+Shift+R)"
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
echo "  • Updater: ${PREFIX}/sbin/zid-proxy-update"
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
echo "  • Update (latest): ${PREFIX}/sbin/zid-proxy-update"
echo ""
