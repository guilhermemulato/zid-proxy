#!/bin/sh
#
# zid-proxy-update (bootstrap)
#
# Small, stable updater wrapper that always runs the updater shipped in the latest bundle.
#
# Usage:
#   sh /usr/local/sbin/zid-proxy-update
#   sh /usr/local/sbin/zid-proxy-update -u https://.../zid-proxy-pfsense-latest.tar.gz
#

set -eu

URL_DEFAULT="https://s3.soulsolucoes.com.br/soul/portal/zid-proxy-pfsense-latest.tar.gz"
URL="${ZID_PROXY_UPDATE_URL:-$URL_DEFAULT}"
KEEP_TMP=0

usage() {
	cat <<EOF
ZID Proxy updater (bootstrap)

Usage:
  sh /usr/local/sbin/zid-proxy-update [-u <url>] [-k]

Options:
  -u <url>  Bundle URL (default: ${URL_DEFAULT})
  -k        Keep temporary directory (debug)
EOF
}

die() {
	echo "ERROR: $*" >&2
	exit 1
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

UPDATER_SH="$(find "${EXTRACT_DIR}" -maxdepth 5 -type f -path "*/pkg-zid-proxy/update.sh" | head -n 1 || true)"
if [ -z "${UPDATER_SH}" ]; then
	die "update.sh not found inside bundle (expected */pkg-zid-proxy/update.sh)"
fi

echo "Running bundled updater: ${UPDATER_SH}"
echo ""
sh "${UPDATER_SH}"

