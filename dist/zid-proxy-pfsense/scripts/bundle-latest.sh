#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "${ROOT_DIR}"

VERSION_FILE="zid-proxy-pfsense-latest.version"
if [ ! -f "${VERSION_FILE}" ]; then
	echo "ERROR: ${VERSION_FILE} not found" >&2
	exit 2
fi
VERSION="$(head -n 1 "${VERSION_FILE}" | tr -d '\r' | tr -d '\n')"
if [ -z "${VERSION}" ]; then
	echo "ERROR: ${VERSION_FILE} is empty" >&2
	exit 2
fi

if [ ! -f build/zid-proxy ] || [ ! -f build/zid-proxy-logrotate ]; then
	echo "ERROR: missing binaries in ./build. Run: make build-freebsd" >&2
	exit 2
fi

STAGE_BASE="dist"
STAGE_DIR="${STAGE_BASE}/zid-proxy-pfsense"

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/build"

cp -f CLAUDE.md "${STAGE_DIR}/CLAUDE.md"
cp -f INSTALL-PFSENSE.md "${STAGE_DIR}/INSTALL-PFSENSE.md"
cp -R configs "${STAGE_DIR}/configs"
cp -R scripts "${STAGE_DIR}/scripts"
cp -R pkg-zid-proxy "${STAGE_DIR}/pkg-zid-proxy"

cp -f build/zid-proxy "${STAGE_DIR}/build/zid-proxy"
cp -f build/zid-proxy-logrotate "${STAGE_DIR}/build/zid-proxy-logrotate"
chmod 755 "${STAGE_DIR}/build/zid-proxy" "${STAGE_DIR}/build/zid-proxy-logrotate"

printf "%s\n" "${VERSION}" > "${STAGE_DIR}/VERSION"

OUT="zid-proxy-pfsense-latest.tar.gz"
TMP_OUT="${OUT}.tmp.$$"
rm -f "${TMP_OUT}"
tar -czf "${TMP_OUT}" -C "${STAGE_BASE}" zid-proxy-pfsense
mv -f "${TMP_OUT}" "${OUT}"

HASH=""
if command -v sha256sum >/dev/null 2>&1; then
	HASH="$(sha256sum "${OUT}" | awk '{print $1}')"
elif command -v sha256 >/dev/null 2>&1; then
	HASH="$(sha256 -q "${OUT}")"
fi

if [ -n "${HASH}" ]; then
	TMP_SHA="$(mktemp)"
	if [ -f sha256.txt ]; then
		awk '$2 != "zid-proxy-pfsense-latest.tar.gz" {print}' sha256.txt > "${TMP_SHA}" || true
	fi
	printf "%s  %s\n" "${HASH}" "${OUT}" >> "${TMP_SHA}"
	mv -f "${TMP_SHA}" sha256.txt
else
	echo "WARN: could not compute sha256 for ${OUT}" >&2
fi

ls -lh "${OUT}" sha256.txt

