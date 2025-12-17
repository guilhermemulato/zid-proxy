#!/bin/sh
#
# uninstall.sh
#
# Uninstalls ZID Proxy from pfSense
# Removes all package files, configuration, and service scripts
#
# Usage: ./uninstall.sh
#
# Licensed under the Apache License, Version 2.0
#

set -e

echo "========================================="
echo " ZID Proxy Uninstaller"
echo "========================================="
echo ""

# Check if running as root
if [ "$(id -u)" != "0" ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Confirmation prompt
echo "This will remove ZID Proxy and all its files from your system."
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Uninstallation cancelled."
    exit 0
fi

echo ""
echo "Starting uninstallation..."
echo ""

# Stop the service if running
echo "Stopping service..."
if [ -f /usr/local/etc/rc.d/zid-proxy.sh ]; then
    /usr/local/etc/rc.d/zid-proxy.sh stop 2>/dev/null || true
elif [ -f /usr/local/etc/rc.d/zid-proxy ]; then
    /usr/local/etc/rc.d/zid-proxy stop 2>/dev/null || true
fi

# Kill process if still running
if [ -f /var/run/zid-proxy.pid ]; then
    pid=$(cat /var/run/zid-proxy.pid)
    if kill -0 "$pid" 2>/dev/null; then
        echo "Killing process $pid..."
        kill "$pid" 2>/dev/null || true
        sleep 1
    fi
fi

# Execute deinstall hook if available
if [ -f /usr/local/pkg/zid-proxy.inc ]; then
    echo "Executing deinstall hook..."
    php <<'EOF'
<?php
require_once('/usr/local/pkg/zid-proxy.inc');
zidproxy_deinstall();
?>
EOF
fi

# Remove binary
echo "Removing binary..."
rm -f /usr/local/sbin/zid-proxy

# Remove updater helper
rm -f /usr/local/sbin/zid-proxy-update

# Remove package files
echo "Removing package files..."
rm -f /usr/local/pkg/zid-proxy.xml
rm -f /usr/local/pkg/zid-proxy.inc

# Remove web interface files
echo "Removing web interface files..."
rm -f /usr/local/www/zid-proxy_rules.php
rm -f /usr/local/www/zid-proxy_log.php

# Remove privilege definitions
echo "Removing privilege definitions..."
rm -f /etc/inc/priv/zid-proxy.priv.inc

# Remove package info
echo "Removing package info..."
rm -rf /usr/local/share/pfSense-pkg-zid-proxy

# Remove RC scripts
echo "Removing RC scripts..."
rm -f /usr/local/etc/rc.d/zid-proxy.sh
rm -f /usr/local/etc/rc.d/zid-proxy

# Remove runtime files
echo "Removing runtime files..."
rm -f /var/run/zid-proxy.pid

# Ask about configuration and logs
echo ""
read -p "Remove configuration directory (/usr/local/etc/zid-proxy)? (yes/no): " remove_config

if [ "$remove_config" = "yes" ]; then
    echo "Removing configuration directory..."
    rm -rf /usr/local/etc/zid-proxy
else
    echo "Keeping configuration directory..."
fi

read -p "Remove log file (/var/log/zid-proxy.log)? (yes/no): " remove_log

if [ "$remove_log" = "yes" ]; then
    echo "Removing log file..."
    rm -f /var/log/zid-proxy.log
else
    echo "Keeping log file..."
fi

# Remove rc.conf entries
echo "Removing rc.conf entries..."
if [ -f /etc/rc.conf.local ]; then
    sed -i.bak '/zid_proxy/d' /etc/rc.conf.local
fi
if [ -f /etc/rc.conf ]; then
    sed -i.bak '/zid_proxy/d' /etc/rc.conf
fi

# Unregister from pfSense config.xml
if [ -f /cf/conf/config.xml ]; then
    echo "Unregistering from pfSense..."
    php <<'EOF'
<?php
require_once('/etc/inc/config.inc');
$config = parse_config(true);

// Remove package registration
if (isset($config['installedpackages']['package'])) {
    foreach ($config['installedpackages']['package'] as $idx => $pkg) {
        if (isset($pkg['name']) && $pkg['name'] == 'zid-proxy') {
            unset($config['installedpackages']['package'][$idx]);
            break;
        }
    }
    // Re-index array
    $config['installedpackages']['package'] = array_values($config['installedpackages']['package']);
}

// Remove configuration
unset($config['installedpackages']['zidproxy']);
unset($config['installedpackages']['zidproxyrules']);

write_config("ZID Proxy package uninstalled");
echo "Package unregistered from config.xml\n";
?>
EOF
fi

echo ""
echo "========================================="
echo " Uninstallation Complete!"
echo "========================================="
echo ""
echo "ZID Proxy has been removed from your system."
echo ""
echo "To complete the removal, you should reload the web interface:"
echo "  /usr/local/sbin/pfSsh.php playback reloadwebgui"
echo ""
echo "Or restart pfSense:"
echo "  shutdown -r now"
echo ""
