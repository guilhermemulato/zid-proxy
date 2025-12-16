# Changelog

All notable changes to zid-proxy will be documented in this file.

## [1.0.3] - 2025-12-16

### Added
- **Automatic package registration** in install.sh - menu now appears without user intervention
- Automatic GUI reload after package installation
- Comprehensive documentation of architectural limitations
- Detailed workaround instructions for IP access without SNI

### Fixed
- **Critical**: Menu "Services > ZID Proxy" now appears automatically after installation
- Installation script no longer requires manual registration
- GUI reloads automatically, making menu visible immediately

### Changed
- `pkg-zid-proxy/install.sh` - removed interactive prompt, always registers package
- Installation process now fully automatic (register + GUI reload)

### Documentation
- Enhanced `internal/proxy/handler.go` with detailed comments explaining SNI limitation
- Added comprehensive "Direct IP Access" section in TROUBLESHOOTING.md
- Added "Known Limitations" section in README.md with workarounds
- Updated INSTALL-PFSENSE.md with mandatory configuration steps
- Clarified that IP access requires NAT bypass configuration

### Known Limitations (Documented)
- Direct IP access (e.g., https://192.168.1.1) requires NAT bypass configuration
- This is an architectural limitation - cannot be fixed without complete rewrite using divert sockets
- Workarounds: NAT bypass, hostname access, or different port for GUI

## [1.0.2] - 2025-12-16

### Added
- Support for private IP access without SNI (allows access to pfSense GUI via https://192.168.1.1)
- Comprehensive TROUBLESHOOTING.md documentation
- Private IP detection for RFC 1918 ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)

### Fixed
- **Critical**: BLOCK rules now apply immediately after saving via GUI
- Rule reload mechanism changed from SIGHUP to service restart for reliability
- Connections to private IPs without SNI are now allowed (previously blocked with RST)

### Changed
- `zidproxy_save_rules()` now restarts service instead of sending SIGHUP
- Handler now distinguishes between private and public IPs for no-SNI connections

### Documentation
- Added detailed QUIC/HTTP3 troubleshooting (ERR_QUIC_PROTOCOL_ERROR solution)
- Documented rule reload issues and manual reload procedures
- Added diagnostics commands reference

## [1.0.1] - 2025-12-16

### Added
- New "All Interfaces (0.0.0.0)" option in GUI dropdown for Listen Interface
- Interface selection now defaults to "All Interfaces" for better NAT compatibility

### Fixed
- **Critical**: GUI no longer overwrites listen address with specific interface IP
- Proxy now continues working after saving configuration via web interface
- Listen address correctly set to `0.0.0.0:3129` when "All Interfaces" is selected

### Changed
- Default interface changed from "lan" to "all" for new installations
- Updated help text in GUI to recommend "All Interfaces" for NAT setups

## [1.0.0] - 2025-12-16

### Added
- Initial release of ZID Proxy for pfSense
- Transparent SNI-based HTTPS proxy
- Dual-factor filtering: source IP + destination hostname (SNI)
- pfSense web interface integration (Services > ZID Proxy)
- Complete package installation scripts (install.sh, activate-package.php, register-package.php)
- Diagnostic script (diagnose.sh) for troubleshooting
- Uninstall script for complete removal
- Support for ALLOW/BLOCK rules with priority (ALLOW > BLOCK)
- Runtime rule reloading via SIGHUP
- Structured logging with timestamp, source IP, hostname, and action
- NAT port forward compatibility
- FreeBSD rc.d service script

### Features
- Listen on configurable port (default: 3129)
- Rules file format: `TYPE;IP_OR_CIDR;HOSTNAME`
- Wildcard hostname support (e.g., `*.example.com`)
- TCP RST for blocked connections
- PID file management
- Automatic service startup on boot

---

## Release Notes

### v1.0.1 Highlights

This release fixes a critical issue where the GUI would overwrite the listen address configuration, causing the proxy to stop working after saving settings. The new "All Interfaces" option ensures the proxy works correctly with NAT port forwarding.

**Upgrade Instructions:**
1. Extract new package: `tar -xzf zid-proxy-pfsense-v1.0.1.tar.gz`
2. Run installer: `cd zid-proxy-pfsense/pkg-zid-proxy && sh install.sh`
3. In GUI, select "Listen Interface: All Interfaces (0.0.0.0)"
4. Save configuration

The proxy will now continue working after saving GUI changes.

### v1.0.0 Highlights

First stable release of ZID Proxy for pfSense. Provides transparent HTTPS filtering based on SNI without terminating TLS connections. Integrates seamlessly with pfSense via web interface and supports NAT port forwarding for transparent proxy deployment.
