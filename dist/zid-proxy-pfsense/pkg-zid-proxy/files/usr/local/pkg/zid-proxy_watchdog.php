<?php
/*
 * zid-proxy_watchdog.php
 *
 * Watchdog helper for ZID Proxy:
 * - If package Enable is ON and the daemon is not running, start it.
 * - If Enable is OFF, do nothing.
 *
 * Designed to run from cron (non-interactive).
 */

require_once("config.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

global $config;
zidproxy_ensure_config_defaults();

$cfg = $config['installedpackages']['zidproxy']['config'][0] ?? [];
if (($cfg['enable'] ?? 'off') !== 'on') {
	exit(0);
}

if (function_exists('zidproxy_status') && zidproxy_status()) {
	exit(0);
}

if (function_exists('zidproxy_start')) {
	@zidproxy_start();
}

exit(0);

