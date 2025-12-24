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

# zid-appid is optional but recommended
if [ ! -f build/zid-appid ]; then
	echo "WARN: zid-appid not found in ./build. AppID features will use fallback detection." >&2
fi

if [ ! -f build/zid-agent-linux-amd64 ] || [ ! -f build/zid-agent-windows-amd64.exe ]; then
	echo "ERROR: missing agent binaries in ./build. Run: make build-agent-linux build-agent-windows" >&2
	exit 2
fi

STAGE_BASE="dist"
STAGE_DIR_PFSENSE="${STAGE_BASE}/zid-proxy-pfsense"
STAGE_DIR_AGENT_LINUX="${STAGE_BASE}/zid-agent-linux"
STAGE_DIR_AGENT_WINDOWS="${STAGE_BASE}/zid-agent-windows"

rm -rf "${STAGE_DIR_PFSENSE}" "${STAGE_DIR_AGENT_LINUX}" "${STAGE_DIR_AGENT_WINDOWS}"
mkdir -p "${STAGE_DIR_PFSENSE}/build"
mkdir -p "${STAGE_DIR_AGENT_LINUX}" "${STAGE_DIR_AGENT_WINDOWS}"

cp -f CLAUDE.md "${STAGE_DIR_PFSENSE}/CLAUDE.md"
cp -f INSTALL-PFSENSE.md "${STAGE_DIR_PFSENSE}/INSTALL-PFSENSE.md"
cp -R configs "${STAGE_DIR_PFSENSE}/configs"
cp -R scripts "${STAGE_DIR_PFSENSE}/scripts"
cp -R pkg-zid-proxy "${STAGE_DIR_PFSENSE}/pkg-zid-proxy"

cp -f build/zid-proxy "${STAGE_DIR_PFSENSE}/build/zid-proxy"
cp -f build/zid-proxy-logrotate "${STAGE_DIR_PFSENSE}/build/zid-proxy-logrotate"
chmod 755 "${STAGE_DIR_PFSENSE}/build/zid-proxy" "${STAGE_DIR_PFSENSE}/build/zid-proxy-logrotate"

# Include zid-appid if available
if [ -f build/zid-appid ]; then
	cp -f build/zid-appid "${STAGE_DIR_PFSENSE}/build/zid-appid"
	chmod 755 "${STAGE_DIR_PFSENSE}/build/zid-appid"
fi

cp -f build/zid-agent-linux-amd64 "${STAGE_DIR_AGENT_LINUX}/zid-agent-linux-amd64"
chmod 755 "${STAGE_DIR_AGENT_LINUX}/zid-agent-linux-amd64"
cp -f build/zid-agent-windows-amd64.exe "${STAGE_DIR_AGENT_WINDOWS}/zid-agent-windows-amd64.exe"

printf "%s\n" "${VERSION}" > "${STAGE_DIR_PFSENSE}/VERSION"
printf "%s\n" "${VERSION}" > "${STAGE_DIR_AGENT_LINUX}/VERSION"
printf "%s\n" "${VERSION}" > "${STAGE_DIR_AGENT_WINDOWS}/VERSION"

OUT_PFSENSE="zid-proxy-pfsense-latest.tar.gz"
OUT_AGENT_LINUX="zid-agent-linux-latest.tar.gz"
OUT_AGENT_WINDOWS="zid-agent-windows-latest.tar.gz"

bundle_one() {
	src_dir="$1"
	out="$2"
	tmp_out="${out}.tmp.$$"
	rm -f "${tmp_out}"
	tar -czf "${tmp_out}" -C "${STAGE_BASE}" "${src_dir}"
	mv -f "${tmp_out}" "${out}"
}

bundle_one "zid-proxy-pfsense" "${OUT_PFSENSE}"
bundle_one "zid-agent-linux" "${OUT_AGENT_LINUX}"
bundle_one "zid-agent-windows" "${OUT_AGENT_WINDOWS}"

hash_one() {
	out="$1"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "${out}" | awk '{print $1}'
		return 0
	elif command -v sha256 >/dev/null 2>&1; then
		sha256 -q "${out}"
		return 0
	fi
	return 1
}

TMP_SHA="$(mktemp)"
for out in \
	"zid-proxy-pfsense-latest.tar.gz" \
	"zid-agent-linux-latest.tar.gz" \
	"zid-agent-windows-latest.tar.gz" \
	"zid-agent-linux-gui-latest.tar.gz" \
	"zid-agent-windows-gui-latest.tar.gz"; do
	if [ ! -f "${out}" ]; then
		continue
	fi
	HASH="$(hash_one "${out}" || true)"
	if [ -n "${HASH}" ]; then
		printf "%s  %s\n" "${HASH}" "${out}" >> "${TMP_SHA}"
	else
		echo "WARN: could not compute sha256 for ${out}" >&2
	fi
done
mv -f "${TMP_SHA}" sha256.txt

ls -lh "${OUT_PFSENSE}" "${OUT_AGENT_LINUX}" "${OUT_AGENT_WINDOWS}" sha256.txt
