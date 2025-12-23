# Changelog

All notable changes to zid-proxy will be documented in this file.

## [1.0.11.3.2.5] - 2025-12-23

### Fixed
- Active IPs: corrigido cálculo de idle time que mostrava valores incorretos (103661s)
- Active IPs: identidades (Machine/Username) agora aparecem corretamente no Windows e Linux agents
- Active IPs: resolvido problema de "flickering" (IPs saindo e voltando da tabela)

### Changed
- **Agent TTL**: padrão alterado de 120s para **60s** (identidade expira mais rápido se agent parar)
- **Active IPs Timeout**: padrão alterado de 120s para **300s** (IPs permanecem 5min na tabela após última atividade)
- Agent TTL: validação ajustada para aceitar valores entre 10s e 600s (antes: mínimo 30s)

### Added
- Logs detalhados de heartbeat do agent (IP, Machine, Username) para facilitar debugging
- Logs quando identidade é registrada ou expira por TTL
- Warning quando agent tenta registrar identidade para IP sem tráfego
- Nova coluna "Last Heartbeat" na tabela Active IPs mostrando quando foi o último heartbeat do agent
- Campos `identity_seen` e `identity_idle_seconds` no JSON snapshot de Active IPs

## [1.0.11.3] - 2025-12-19

### Added
- Agent HTTP API (LAN) to map source IP to machine/user
- New pfSense tab: Agent (listener interface/port configuration)

### Changed
- Logs: append optional `| MACHINE | USER` fields (empty when not available)
- Active IPs tab: display Machine/User when available

## [1.0.11.3.1] - 2025-12-19

### Changed
- Packaging: split bundles into separate `latest` tarballs (pfSense, agent Linux, agent Windows)

## [1.0.11.3.2] - 2025-12-19

### Fixed
- pfSense update/install scripts: ensure `/usr/local/www/zid-proxy_agent.php` is installed (Agent tab no longer 404)

## [1.0.11.3.2.1] - 2025-12-19

### Fixed
- pfSense Agent tab: remove duplicated Save button

## [1.0.11.3.2.2] - 2025-12-19

### Fixed
- Active IPs: persist agent machine/user mapping in snapshot to avoid flicker between refreshes

## [1.0.11.3.2.3] - 2025-12-19

### Added
- Logs tab: show Machine/User badges when the source IP is currently active (from Active IPs snapshot)

## [1.0.11.3.2.4] - 2025-12-19

### Added
- Agent identity TTL: clear Machine/User after X seconds without heartbeat (configurable on Agent tab)

## [1.0.11.2] - 2025-12-18

### Changed
- Active IPs: Last Activity now displayed in `America/Sao_Paulo` timezone

## [1.0.11.1] - 2025-12-18

### Changed
- Active IPs: `Bytes Out` now represents client upload (client -> upstream) and accumulates until the IP times out

## [1.0.11] - 2025-12-18

### Added
- Active IPs tracking (aggregated by source IP) with pfSense tab and configurable timeout/refresh

## [1.0.10.7] - 2025-12-18

### Added
- Watchdog cron: monitors `zid-proxy` and starts it when `Enable=on`

## [1.0.10.8] - 2025-12-18

### Fixed
- Watchdog cron: create job with correct column alignment on pfSense cron GUI

## [1.0.10.8.1] - 2025-12-18

### Fixed
- Logrotate cron: remove duplicate/broken cron entries and keep only one correct job

## [1.0.10.6] - 2025-12-18

### Fixed
- Cron install: ensures the cron `command` includes full args (prevents creating a job with only `zid-proxy-logrotate`)

## [1.0.10.5] - 2025-12-18

### Fixed
- Cron install: improved compatibility/verification (ensures cron is actually persisted and applied via `configure_cron()`)

## [1.0.10.4] - 2025-12-18

### Added
- Hourly logrotate cron via `install_cron_job()` (keeps daily rotated logs based on `log_retention_days`)

## [1.0.10.3] - 2025-12-18

### Fixed
- pfSense Settings: prevent wiping existing config on save (merge config + save marker)
- pfSense Menu entry now points to `/zid-proxy_settings.php` (instead of `/pkg.php?xml=zid-proxy.xml`)

## [1.0.10.2] - 2025-12-18

### Fixed
- pfSense Settings: removed duplicate Save button and added a bit more padding to Service Controls buttons row

## [1.0.10.1] - 2025-12-18

### Changed
- Release repack: regenerated `zid-proxy-pfsense-latest.tar.gz` without functional changes
- Updated `TODO-NOVAS-FUNCIONALIDADES.md` to mark implemented phases

## [1.0.10] - 2025-12-18

### Added
- Daily log rotation support via new `zid-proxy-logrotate` binary
- pfSense Settings field: log retention days (default 7)
- pfSense Settings page: shows installed version, update button, and service controls

### Changed
- `cmd/zid-proxy/main.go` - SIGHUP now also reopens the log file (for log rotation)
- `Makefile` - Builds `zid-proxy` and `zid-proxy-logrotate` (local + FreeBSD targets)

## [1.0.8] - 2025-12-17

### Fixed
- **CRITICAL BUG**: BLOCK rules now work correctly
- Fixed rule matching logic - no longer returns ALLOW immediately when first ALLOW rule matches
- Rule priority (ALLOW > BLOCK) now works as intended: checks ALL matching rules before deciding
- BLOCK rules are no longer ignored if ALLOW rules exist earlier in the file

### Changed
- `internal/rules/rules.go` - Rewrote `Match()` function to collect all matching rules before applying priority logic
- Removed early return that prevented BLOCK rules from being evaluated

### Technical Details
- Bug: Code returned ALLOW immediately on first match, ignoring subsequent BLOCK rules
- Fix: Now iterates through ALL rules, tracks both `allowMatched` and `blockMatched`, then decides based on priority
- All existing unit tests pass, confirming correct behavior restoration
- Behavior now matches documentation: "ALLOW priority > BLOCK; default ALLOW if no match"

## [1.0.7] - 2025-12-17

### Added
- **UX Improvement**: Settings tab now displays configuration summary table
- Added `<adddeleteeditpagefields>` section to zid-proxy.xml
- Configuration table shows 5 columns: Enable, Listen Interface, Listen Port, Logging, Timeout

### Changed
- Settings page no longer shows empty table with only edit/delete icons
- Users can now see current configuration values at a glance without clicking Edit
- Improved visual feedback for enabled/disabled status with checkmarks

### Technical Details
- Added XML section defining column display for pfSense package GUI
- Each `<columnitem>` maps to a configuration field
- pfSense automatically formats checkbox values (✓ when enabled)
- No backend changes required - pure XML configuration update

## [1.0.6] - 2025-12-17

### Fixed
- **Critical**: Log latency reduced from 3 minutes to ≤1 second on pfSense 2.8.1/FreeBSD 15
- Activated automatic log flush ticker (was implemented but never called)
- Logger now flushes buffer every 1 second for near real-time logging

### Changed
- `cmd/zid-proxy/main.go` - Added call to `startFlushTicker()` after logger initialization
- `startFlushTicker()` now accepts configurable interval parameter
- Added error handling for flush operations in ticker goroutine

### Technical Details
- Root cause: `bufio.Writer` buffered logs in 4KB buffer without periodic flushing
- Only flushed on process shutdown, causing 3-minute delays on low-traffic systems
- Solution: Enabled existing flush ticker with 1-second interval
- Impact: Minimal overhead (1 flush/second), significant UX improvement

## [1.0.5] - 2025-12-17 (hotfix updated)

### Fixed
- **GUI Reload Command**: Changed from `/usr/local/etc/rc.d/php-fpm onerestart` to `/etc/rc.restart_webgui` (no more 502 errors)
- GUI now reloads correctly after installation without causing Bad Gateway errors
- **Filter Persistence (HOTFIX)**: JavaScript now updates meta tag when filter changes - filter persists correctly across auto-refresh
- **Timezone**: Log timestamps now display in America/Sao_Paulo timezone instead of UTC
- **Navbar Layout**: Increased panel-heading height to 60px - controls no longer compressed

### Changed
- `setMetaRefresh()` function now reads filter value directly from DOM instead of relying on stale URL
- Filter keyup handler now calls `setMetaRefresh()` to update meta tag with current filter
- PHP meta tag generation simplified using `http_build_query()`
- Added CSS to increase navbar height and spacing for better control layout

### Added
- **Log Viewer Auto-Refresh**: Dropdown selector with options: Disabled, 5s, 10s, 20s, 30s (default: 20s)
- **Pause Auto-Refresh**: Checkbox to pause auto-refresh for detailed log analysis
- **Real-Time Filter**: Input field to filter logs by IP or domain name
  - Filters as you type (no delay)
  - Case-insensitive substring matching
  - Works on both source IP and hostname columns
  - Filter persists across auto-refresh cycles
  - Filter state saved in URL query string

### Changed
- `pkg-zid-proxy/install.sh` - Uses `/etc/rc.restart_webgui` instead of PHP-FPM restart
- `pkg-zid-proxy/register-package.php` - Updated to v1.0.5 with corrected instructions
- `pkg-zid-proxy/files/usr/local/www/zid-proxy_log.php` - Complete rewrite with new features:
  - PHP backend filtering for optimization
  - JavaScript auto-refresh control with meta tag manipulation
  - localStorage-based pause state persistence
  - URL-based filter and refresh interval persistence
  - Real-time table filtering without page reload

### User Experience Improvements
- Monitor logs in real-time with configurable refresh rate
- Pause refresh when needed to analyze specific entries
- Quickly filter by IP (e.g., "192.168.1") or domain (e.g., "facebook")
- Combine filter + auto-refresh for focused monitoring
- Filter remains active during page reloads

## [1.0.4] - 2025-12-17

### Fixed
- **Critical**: Menu "Services > ZID Proxy" now appears automatically after installation
- **Critical**: Service now auto-starts after pfSense reboot when Enable is checked
- Both issues fixed by adding `<menu>` tag to config.xml in register-package.php

### Changed
- `pkg-zid-proxy/register-package.php` - Complete rewrite:
  - Now adds `<menu>` tag directly to config.xml (enables menu display AND boot auto-start)
  - Uses `configurationfile` instead of `config_file` (correct pfSense convention)
  - Uses filename only (`zid-proxy.xml`) instead of full path
  - Default interface changed from `lan` to `all` for better NAT compatibility
  - Version updated to 1.0.4
- `pkg-zid-proxy/install.sh` - Uses `/usr/local/etc/rc.d/php-fpm onerestart` instead of `reloadwebgui`

### Root Cause (Documented)
- Without `<menu>` tag in config.xml, pfSense doesn't recognize the package during boot
- This prevents `<custom_php_resync_config_command>` from being called
- Which means `zidproxy_resync()` doesn't execute to configure rc.conf.local
- Result: No menu in GUI AND no service auto-start after reboot
- Solution: Add `<menu>` tag during package registration

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
