#!/bin/sh
# Bundle script for GUI agents
# This creates separate bundles for Linux and Windows GUI agents

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

# Check for GUI agent binaries
if [ ! -f build/zid-agent-linux-gui ] && [ ! -f build/zid-agent-windows-gui.exe ]; then
	echo "ERROR: No GUI agent binaries found in ./build" >&2
	echo "Run: make build-agent-linux-gui build-agent-windows-gui" >&2
	exit 2
fi

STAGE_BASE="dist"
STAGE_DIR_AGENT_LINUX_GUI="${STAGE_BASE}/zid-agent-linux-gui"
STAGE_DIR_AGENT_WINDOWS_GUI="${STAGE_BASE}/zid-agent-windows-gui"

rm -rf "${STAGE_DIR_AGENT_LINUX_GUI}" "${STAGE_DIR_AGENT_WINDOWS_GUI}"
mkdir -p "${STAGE_DIR_AGENT_LINUX_GUI}" "${STAGE_DIR_AGENT_WINDOWS_GUI}"

# Linux GUI bundle
if [ -f build/zid-agent-linux-gui ]; then
	echo "Bundling Linux GUI agent..."
	cp -f build/zid-agent-linux-gui "${STAGE_DIR_AGENT_LINUX_GUI}/zid-agent-linux-gui"
	chmod 755 "${STAGE_DIR_AGENT_LINUX_GUI}/zid-agent-linux-gui"

	# Copy installation scripts
	cp -f scripts/agent-installers/install-linux.sh "${STAGE_DIR_AGENT_LINUX_GUI}/"
	cp -f scripts/agent-installers/uninstall-linux.sh "${STAGE_DIR_AGENT_LINUX_GUI}/"
	cp -f scripts/agent-installers/update-linux.sh "${STAGE_DIR_AGENT_LINUX_GUI}/"
	chmod +x "${STAGE_DIR_AGENT_LINUX_GUI}/install-linux.sh"
	chmod +x "${STAGE_DIR_AGENT_LINUX_GUI}/uninstall-linux.sh"
	chmod +x "${STAGE_DIR_AGENT_LINUX_GUI}/update-linux.sh"

	# Create README
	cat > "${STAGE_DIR_AGENT_LINUX_GUI}/README.txt" <<'EOF'
ZID Agent - Linux GUI Edition

INSTALLATION:
  ./install-linux.sh

This will install the agent and configure it to start automatically.

UPDATE:
  ./update-linux.sh

This will update an existing installation without reconfiguring.

MANUAL START:
  ./zid-agent-linux-gui

UNINSTALL:
  ./uninstall-linux.sh

REQUIREMENTS:
- X11 or Wayland display server
- System tray support (GNOME: install gnome-shell-extension-appindicator)
- Network access to pfSense gateway

For more information, see BUILD-AGENT.md in the main repository.
EOF

	printf "%s\n" "${VERSION}" > "${STAGE_DIR_AGENT_LINUX_GUI}/VERSION"
fi

# Windows GUI bundle
if [ -f build/zid-agent-windows-gui.exe ]; then
	echo "Bundling Windows GUI agent..."
	cp -f build/zid-agent-windows-gui.exe "${STAGE_DIR_AGENT_WINDOWS_GUI}/zid-agent-windows-gui.exe"

	# Copy installation scripts
	cp -f scripts/agent-installers/install-windows.bat "${STAGE_DIR_AGENT_WINDOWS_GUI}/"
	cp -f scripts/agent-installers/uninstall-windows.bat "${STAGE_DIR_AGENT_WINDOWS_GUI}/"
	cp -f scripts/agent-installers/update-windows.bat "${STAGE_DIR_AGENT_WINDOWS_GUI}/"

	# Create README
	cat > "${STAGE_DIR_AGENT_WINDOWS_GUI}/README.txt" <<'EOF'
ZID Agent - Windows GUI Edition

INSTALLATION:
  Double-click: install-windows.bat

This will install the agent to %LOCALAPPDATA%\ZIDAgent and configure it
to start automatically on login.

UPDATE:
  Double-click: update-windows.bat

This will update an existing installation without reconfiguring.

MANUAL START:
  Double-click: zid-agent-windows-gui.exe

UNINSTALL:
  Double-click: uninstall-windows.bat

REQUIREMENTS:
- Windows 10 or later
- Network access to pfSense gateway

The agent runs in the system tray. Right-click the icon to access logs.

For more information, see BUILD-AGENT.md in the main repository.
EOF

	printf "%s\n" "${VERSION}" > "${STAGE_DIR_AGENT_WINDOWS_GUI}/VERSION"
fi

# Create bundles
bundle_one() {
	src_dir="$1"
	out="$2"

	if [ ! -d "${STAGE_BASE}/${src_dir}" ]; then
		echo "SKIP: ${src_dir} not found" >&2
		return 0
	fi

	tmp_out="${out}.tmp.$$"
	rm -f "${tmp_out}"
	tar -czf "${tmp_out}" -C "${STAGE_BASE}" "${src_dir}"
	mv -f "${tmp_out}" "${out}"
	echo "Created: ${out}"
}

OUT_AGENT_LINUX_GUI="zid-agent-linux-gui-latest.tar.gz"
OUT_AGENT_WINDOWS_GUI="zid-agent-windows-gui-latest.tar.gz"

bundle_one "zid-agent-linux-gui" "${OUT_AGENT_LINUX_GUI}"
bundle_one "zid-agent-windows-gui" "${OUT_AGENT_WINDOWS_GUI}"

# Update sha256.txt
hash_one() {
	out="$1"
	if [ ! -f "${out}" ]; then
		return 0
	fi
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
if [ -f sha256.txt ]; then
	awk '
		$2 != "zid-agent-linux-gui-latest.tar.gz" &&
		$2 != "zid-agent-windows-gui-latest.tar.gz"
		{print}
	' sha256.txt > "${TMP_SHA}" || true
fi

for out in "${OUT_AGENT_LINUX_GUI}" "${OUT_AGENT_WINDOWS_GUI}"; do
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

echo ""
echo "Bundle complete!"
ls -lh zid-agent-*-gui-latest.tar.gz 2>/dev/null || true
echo ""
echo "SHA256 checksums updated in sha256.txt"
