#!/bin/sh
#
# diagnose.sh
#
# Diagnostic script for ZID Proxy installation on pfSense
# Checks if all required files are installed and properly configured
#
# Usage: ./diagnose.sh
#
# Licensed under the Apache License, Version 2.0
#

echo "========================================="
echo " ZID Proxy Installation Diagnostic"
echo "========================================="
echo ""

# Check if running as root
if [ "$(id -u)" != "0" ]; then
    echo "Warning: Not running as root. Some checks may fail."
    echo ""
fi

# Function to check file
check_file() {
    file=$1
    description=$2

    if [ -f "$file" ]; then
        size=$(ls -lh "$file" | awk '{print $5}')
        perms=$(ls -l "$file" | awk '{print $1}')
        echo "✓ $description"
        echo "  Path: $file"
        echo "  Size: $size  Permissions: $perms"
    else
        echo "✗ $description"
        echo "  Path: $file (NOT FOUND)"
    fi
    echo ""
}

# Function to check directory
check_dir() {
    dir=$1
    description=$2

    if [ -d "$dir" ]; then
        count=$(ls -1 "$dir" 2>/dev/null | wc -l | tr -d ' ')
        perms=$(ls -ld "$dir" | awk '{print $1}')
        echo "✓ $description"
        echo "  Path: $dir"
        echo "  Files: $count  Permissions: $perms"
    else
        echo "✗ $description"
        echo "  Path: $dir (NOT FOUND)"
    fi
    echo ""
}

echo "=== BINARY ==="
echo ""
check_file "/usr/local/sbin/zid-proxy" "ZID Proxy binary"

if [ -f /usr/local/sbin/zid-proxy ]; then
    echo "Binary version info:"
    /usr/local/sbin/zid-proxy -version 2>/dev/null || echo "  (no version flag available)"
    echo ""
fi

echo "=== PACKAGE FILES ==="
echo ""
check_file "/usr/local/pkg/zid-proxy.xml" "Package XML manifest"
check_file "/usr/local/pkg/zid-proxy.inc" "Package PHP functions"
check_file "/usr/local/www/zid-proxy_rules.php" "Rules management page"
check_file "/usr/local/www/zid-proxy_log.php" "Log viewer page"
check_file "/usr/local/www/zid-proxy_agent.php" "Agent settings page"
check_file "/etc/inc/priv/zid-proxy.priv.inc" "Privilege definitions"
check_file "/usr/local/share/pfSense-pkg-zid-proxy/info.xml" "Package info"

echo "=== RC SCRIPTS ==="
echo ""
check_file "/usr/local/etc/rc.d/zid-proxy.sh" "RC script (pfSense package)"
check_file "/usr/local/etc/rc.d/zid-proxy" "RC script (standalone)"

echo "=== CONFIGURATION ==="
echo ""
check_dir "/usr/local/etc/zid-proxy" "Configuration directory"
check_file "/usr/local/etc/zid-proxy/access_rules.txt" "Access rules file"

if [ -f /usr/local/etc/zid-proxy/access_rules.txt ]; then
    lines=$(wc -l < /usr/local/etc/zid-proxy/access_rules.txt | tr -d ' ')
    echo "Rules file: $lines lines"
    echo ""
fi

echo "=== RUNTIME FILES ==="
echo ""
check_file "/var/log/zid-proxy.log" "Log file"

if [ -f /var/log/zid-proxy.log ]; then
    lines=$(wc -l < /var/log/zid-proxy.log 2>/dev/null | tr -d ' ')
    echo "Log entries: $lines lines"
    echo ""
fi

check_file "/var/run/zid-proxy.pid" "PID file"

if [ -f /var/run/zid-proxy.pid ]; then
    pid=$(cat /var/run/zid-proxy.pid)
    if kill -0 "$pid" 2>/dev/null; then
        echo "Process status: Running (PID: $pid)"
    else
        echo "Process status: NOT running (stale PID file)"
    fi
    echo ""
fi

echo "=== RC.CONF SETTINGS ==="
echo ""

if [ -f /etc/rc.conf.local ]; then
    echo "Checking /etc/rc.conf.local:"
    grep "zid_proxy" /etc/rc.conf.local 2>/dev/null || echo "  (no zid_proxy settings found)"
    echo ""
else
    echo "/etc/rc.conf.local not found"
    echo ""
fi

if [ -f /etc/rc.conf ]; then
    echo "Checking /etc/rc.conf:"
    grep "zid_proxy" /etc/rc.conf 2>/dev/null || echo "  (no zid_proxy settings found)"
    echo ""
fi

echo "=== PFSENSE PACKAGE REGISTRATION ==="
echo ""

if [ -f /cf/conf/config.xml ]; then
    echo "Checking if package is registered in config.xml:"
    if grep -q "zid-proxy" /cf/conf/config.xml 2>/dev/null; then
        echo "✓ Package found in config.xml"
    else
        echo "✗ Package NOT found in config.xml"
        echo "  Run register-package.php to register the package"
    fi
    echo ""
else
    echo "Warning: config.xml not found (not a pfSense system?)"
    echo ""
fi

echo "=== SERVICE STATUS ==="
echo ""

# Try different ways to check service status
if [ -x /usr/local/etc/rc.d/zid-proxy.sh ]; then
    echo "Using /usr/local/etc/rc.d/zid-proxy.sh status:"
    /usr/local/etc/rc.d/zid-proxy.sh status 2>&1
elif [ -x /usr/local/etc/rc.d/zid-proxy ]; then
    echo "Using /usr/local/etc/rc.d/zid-proxy status:"
    /usr/local/etc/rc.d/zid-proxy status 2>&1
else
    echo "No executable RC script found"
fi
echo ""

echo "Using 'service zid-proxy status':"
service zid-proxy status 2>&1
echo ""

echo "========================================="
echo " Diagnostic Complete"
echo "========================================="
echo ""

# Summary and recommendations
echo "SUMMARY:"
echo ""

errors=0

if [ ! -f /usr/local/sbin/zid-proxy ]; then
    echo "✗ Binary not installed - copy build/zid-proxy to /usr/local/sbin/"
    errors=$((errors + 1))
fi

if [ ! -f /usr/local/pkg/zid-proxy.inc ] || [ ! -f /usr/local/pkg/zid-proxy.xml ]; then
    echo "✗ Package files not installed - run install.sh"
    errors=$((errors + 1))
fi

if [ ! -f /usr/local/etc/rc.d/zid-proxy.sh ] && [ ! -f /usr/local/etc/rc.d/zid-proxy ]; then
    echo "✗ RC script not found - run activate-package.php"
    errors=$((errors + 1))
fi

if [ ! -d /usr/local/etc/zid-proxy ]; then
    echo "✗ Config directory missing - run activate-package.php"
    errors=$((errors + 1))
fi

if [ $errors -eq 0 ]; then
    echo "✓ All components appear to be installed correctly!"
    echo ""
    echo "If 'Services > ZID Proxy' doesn't appear in web interface:"
    echo "  1. Run: php register-package.php"
    echo "  2. Reload web interface or restart pfSense"
else
    echo ""
    echo "Found $errors issue(s). Follow the recommendations above."
fi

echo ""
