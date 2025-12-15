#!/bin/sh
#
# ZID Proxy pfSense Package Installation Script
#
# Usage:
#   1. Copy this folder to pfSense: scp -r zid-proxy-pfsense root@pfsense:/tmp/
#   2. SSH into pfSense: ssh root@pfsense
#   3. Run: cd /tmp/zid-proxy-pfsense && sh install.sh
#

set -e

echo "========================================="
echo " ZID Proxy pfSense Package Installer"
echo "========================================="
echo ""

# Check if running as root
if [ "$(id -u)" != "0" ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FILES_DIR="${SCRIPT_DIR}/files"
PREFIX="/usr/local"

echo "Installing from: ${SCRIPT_DIR}"
echo ""

# Create directories
echo "[1/7] Creating directories..."
mkdir -p ${PREFIX}/pkg
mkdir -p ${PREFIX}/www
mkdir -p ${PREFIX}/etc/rc.d
mkdir -p ${PREFIX}/sbin
mkdir -p ${PREFIX}/etc/zid-proxy
mkdir -p ${PREFIX}/share/pfSense-pkg-zid-proxy
mkdir -p /etc/inc/priv
mkdir -p /var/log

# Install binary
echo "[2/7] Installing binary..."
if [ -f "${SCRIPT_DIR}/zid-proxy" ]; then
    cp "${SCRIPT_DIR}/zid-proxy" ${PREFIX}/sbin/zid-proxy
    chmod 755 ${PREFIX}/sbin/zid-proxy
    echo "       Binary installed: ${PREFIX}/sbin/zid-proxy"
else
    echo "       WARNING: Binary not found!"
    echo "       Copy it manually: scp zid-proxy root@pfsense:${PREFIX}/sbin/"
fi

# Install rc.d script
echo "[3/7] Installing rc.d service script..."
cat > ${PREFIX}/etc/rc.d/zid-proxy.sh << 'RCEOF'
#!/bin/sh
#
# PROVIDE: zid_proxy
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="zid_proxy"
rcvar="zid_proxy_enable"

load_rc_config $name

: ${zid_proxy_enable:="NO"}
: ${zid_proxy_listen:=":3129"}
: ${zid_proxy_rules:="/usr/local/etc/zid-proxy/access_rules.txt"}
: ${zid_proxy_log:="/var/log/zid-proxy.log"}
: ${zid_proxy_pid:="/var/run/zid-proxy.pid"}

pidfile="${zid_proxy_pid}"
procname="/usr/local/sbin/zid-proxy"
command="/usr/sbin/daemon"
command_args="-f -p ${pidfile} ${procname} -listen ${zid_proxy_listen} -rules ${zid_proxy_rules} -log ${zid_proxy_log} -pid ${zid_proxy_pid}"

start_precmd="zid_proxy_prestart"
stop_postcmd="zid_proxy_poststop"
extra_commands="reload status"
reload_cmd="zid_proxy_reload"
status_cmd="zid_proxy_status"

zid_proxy_prestart()
{
    mkdir -p /usr/local/etc/zid-proxy
    touch ${zid_proxy_log}
    return 0
}

zid_proxy_poststop()
{
    rm -f ${pidfile}
}

zid_proxy_reload()
{
    if [ -f ${pidfile} ]; then
        kill -HUP $(cat ${pidfile})
        echo "Rules reloaded."
    else
        echo "${name} is not running."
        return 1
    fi
}

zid_proxy_status()
{
    if [ -f ${pidfile} ]; then
        pid=$(cat ${pidfile})
        if kill -0 ${pid} 2>/dev/null; then
            echo "${name} is running as pid ${pid}."
            return 0
        fi
    fi
    echo "${name} is not running."
    return 1
}

run_rc_command "$1"
RCEOF
chmod 755 ${PREFIX}/etc/rc.d/zid-proxy.sh
echo "       RC script installed: ${PREFIX}/etc/rc.d/zid-proxy.sh"

# Install package configuration
echo "[4/7] Installing package configuration..."
cp "${FILES_DIR}${PREFIX}/pkg/zid-proxy.xml" ${PREFIX}/pkg/
cp "${FILES_DIR}${PREFIX}/pkg/zid-proxy.inc" ${PREFIX}/pkg/
chmod 644 ${PREFIX}/pkg/zid-proxy.xml
chmod 644 ${PREFIX}/pkg/zid-proxy.inc

# Install web pages
echo "[5/7] Installing web pages..."
cp "${FILES_DIR}${PREFIX}/www/zid-proxy_rules.php" ${PREFIX}/www/
cp "${FILES_DIR}${PREFIX}/www/zid-proxy_log.php" ${PREFIX}/www/
chmod 644 ${PREFIX}/www/zid-proxy_rules.php
chmod 644 ${PREFIX}/www/zid-proxy_log.php

# Install privilege definitions
echo "[6/7] Installing privilege definitions..."
cp "${FILES_DIR}/etc/inc/priv/zid-proxy.priv.inc" /etc/inc/priv/
chmod 644 /etc/inc/priv/zid-proxy.priv.inc

# Install package info
echo "[7/7] Installing package info..."
cp "${FILES_DIR}${PREFIX}/share/pfSense-pkg-zid-proxy/info.xml" ${PREFIX}/share/pfSense-pkg-zid-proxy/
chmod 644 ${PREFIX}/share/pfSense-pkg-zid-proxy/info.xml

# Create default rules file
if [ ! -f ${PREFIX}/etc/zid-proxy/access_rules.txt ]; then
    echo ""
    echo "Creating default rules file..."
    cat > ${PREFIX}/etc/zid-proxy/access_rules.txt << 'RULESEOF'
# ZID Proxy Access Rules
# Format: TYPE;IP_OR_CIDR;HOSTNAME
# TYPE: ALLOW or BLOCK
# ALLOW rules take priority over BLOCK rules
# Default action (no match): ALLOW

# Example rules:
# BLOCK;192.168.1.0/24;*.facebook.com
# ALLOW;192.168.1.100;*.facebook.com
RULESEOF
fi

# Create log file
touch /var/log/zid-proxy.log
chmod 644 /var/log/zid-proxy.log

echo ""
echo "========================================="
echo " Installation complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Access the pfSense web interface"
echo "2. Navigate to: Services > ZID Proxy"
echo "3. Enable the service and configure settings"
echo "4. Add access rules on the 'Access Rules' tab"
echo ""
echo "To configure NAT redirection:"
echo "  Firewall > NAT > Port Forward"
echo "  Redirect port 443 to 127.0.0.1:3129"
echo ""
echo "Service commands:"
echo "  service zid-proxy.sh start"
echo "  service zid-proxy.sh stop"
echo "  service zid-proxy.sh status"
echo "  service zid-proxy.sh reload"
echo ""
