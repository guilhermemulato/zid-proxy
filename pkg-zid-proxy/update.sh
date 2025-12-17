#!/bin/sh
#
# update.sh
#
# Downloads and applies the latest ZID Proxy pfSense bundle.
# - Fetches a tar.gz from the URL below (override with -u or $ZID_PROXY_UPDATE_URL)
# - Extracts to a temporary directory
# - Runs the bundled pkg-zid-proxy/install.sh (no uninstall required)
#
# Important:
# - Existing pfSense settings (config.xml) are kept.
# - The rules file (/usr/local/etc/zid-proxy/access_rules.txt) is not overwritten by install.sh.
#
# Usage:
#   sh update.sh
#   sh update.sh -u https://.../zid-proxy-pfsense-v1.0.8.tar.gz
#   ZID_PROXY_UPDATE_URL=... sh update.sh
#

set -eu

URL_DEFAULT="https://s3.soulsolucoes.com.br/soul/portal/zid-proxy-pfsense-latest.tar.gz"
URL="${ZID_PROXY_UPDATE_URL:-$URL_DEFAULT}"
KEEP_TMP=0
WAS_RUNNING=0

usage() {
	cat <<EOF
ZID Proxy updater

Usage:
  sh update.sh [-u <url>] [-k]

Options:
  -u <url>  Bundle URL (default: ${URL_DEFAULT})
  -k        Keep temporary directory (debug)
EOF
}

die() {
	echo "ERROR: $*" >&2
	exit 1
}

pids() {
	if command -v pgrep >/dev/null 2>&1; then
		pgrep -f '/usr/local/sbin/zid-proxy' 2>/dev/null || true
		return
	fi
	ps ax -o pid= -o command= | awk '/\/usr\/local\/sbin\/zid-proxy/ {print $1}'
}

while getopts "u:kh" opt; do
	case "$opt" in
		u) URL="$OPTARG" ;;
		k) KEEP_TMP=1 ;;
		h) usage; exit 0 ;;
		*) usage; exit 2 ;;
	esac
done

if [ "$(id -u)" != "0" ]; then
	die "This script must be run as root"
fi

if ! command -v tar >/dev/null 2>&1; then
	die "tar not found"
fi

DOWNLOADER=""
if command -v fetch >/dev/null 2>&1; then
	DOWNLOADER="fetch"
elif command -v curl >/dev/null 2>&1; then
	DOWNLOADER="curl"
else
	die "Neither 'fetch' nor 'curl' found (pfSense usually provides 'fetch')"
fi

TMP_DIR="$(mktemp -d /tmp/zid-proxy-update.XXXXXX)"
cleanup() {
	if [ "${KEEP_TMP}" -eq 1 ]; then
		echo "Keeping temp dir: ${TMP_DIR}"
		return
	fi
	rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

stop_all() {
	echo "Stopping service (best-effort)..."
	# Do not call rc.d stop here: it can block indefinitely ("Waiting for PIDS").
	# We terminate processes directly and then start cleanly via rc.d later.

	# Kill any remaining processes matching /usr/local/sbin/zid-proxy (includes daemon wrapper).
	PIDS="$(pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
	if [ -n "${PIDS}" ]; then
		echo "Stopping running processes: ${PIDS}"
		kill ${PIDS} 2>/dev/null || true

		i=0
		while [ $i -lt 10 ]; do
			PIDS_NOW="$(pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
			if [ -z "${PIDS_NOW}" ]; then
				break
			fi
			sleep 1
			i=$((i + 1))
		done

		PIDS_NOW="$(pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
		if [ -n "${PIDS_NOW}" ]; then
			echo "Processes still running; sending SIGKILL: ${PIDS_NOW}"
			kill -9 ${PIDS_NOW} 2>/dev/null || true
			sleep 1
		fi
	fi

	# Remove stale PID file if no process is running.
	if [ -f /var/run/zid-proxy.pid ]; then
		PID="$(cat /var/run/zid-proxy.pid 2>/dev/null || true)"
		if [ -z "${PID}" ] || ! kill -0 "${PID}" 2>/dev/null; then
			rm -f /var/run/zid-proxy.pid 2>/dev/null || true
		fi
	fi
}

TARBALL="${TMP_DIR}/bundle.tar.gz"
EXTRACT_DIR="${TMP_DIR}/extract"
mkdir -p "${EXTRACT_DIR}"

echo "========================================="
echo " ZID Proxy Update"
echo "========================================="
echo ""
echo "Downloading: ${URL}"

if [ "${DOWNLOADER}" = "fetch" ]; then
	fetch -o "${TARBALL}" "${URL}"
else
	curl -fL -o "${TARBALL}" "${URL}"
fi

echo "Extracting bundle..."
tar -xzf "${TARBALL}" -C "${EXTRACT_DIR}"

INSTALL_SH="$(find "${EXTRACT_DIR}" -maxdepth 5 -type f -path "*/pkg-zid-proxy/install.sh" | head -n 1 || true)"
if [ -z "${INSTALL_SH}" ]; then
	die "install.sh not found inside bundle (expected */pkg-zid-proxy/install.sh)"
fi

PKG_DIR="$(dirname "${INSTALL_SH}")"

# Detect if service is running before update (so we can restart at the end).
if [ -n "$(pids | head -n 1)" ]; then
	WAS_RUNNING=1
fi

stop_all

echo ""
echo "Applying update from: ${PKG_DIR}"
sh "${INSTALL_SH}"

# Safety net: restore rules file only if it disappeared (install.sh won't overwrite it).
RULES_FILE="/usr/local/etc/zid-proxy/access_rules.txt"
if [ ! -f "${RULES_FILE}" ]; then
	echo "Rules file missing after update; recreating a default one..."
	mkdir -p /usr/local/etc/zid-proxy
	cat > "${RULES_FILE}" << 'EOF'
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

echo ""
echo "Restarting service..."
if [ "${WAS_RUNNING}" -eq 1 ]; then
	# install.sh may start/restart via pfSense hooks; enforce a single clean start.
	stop_all
	if [ -x /usr/local/etc/rc.d/zid-proxy.sh ]; then
		/usr/local/etc/rc.d/zid-proxy.sh start 2>/dev/null || true
	else
		service zid-proxy start 2>/dev/null || true
	fi
else
	echo "(Service was not running before update; not forcing start.)"
fi

echo ""
echo "========================================="
echo " Update Complete"
echo "========================================="
echo ""
echo "Tips:"
echo "  - Validate install: sh ${PKG_DIR}/diagnose.sh"
echo "  - If the menu doesn't appear: /etc/rc.restart_webgui"
echo ""
