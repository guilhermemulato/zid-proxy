<?php
/*
 * zid-proxy.inc
 *
 * PHP helper functions for ZID Proxy pfSense package
 *
 * Licensed under the Apache License, Version 2.0
 */

require_once("config.inc");
require_once("functions.inc");
require_once("util.inc");
require_once("service-utils.inc");
require_once("services.inc");
@require_once("cron.inc");

define('ZIDPROXY_BASE', '/usr/local/etc/zid-proxy');
define('ZIDPROXY_RULES_FILE', ZIDPROXY_BASE . '/access_rules.txt');
define('ZIDPROXY_GROUPS_FILE', ZIDPROXY_BASE . '/groups.json');
define('ZIDPROXY_LOG_FILE', '/var/log/zid-proxy.log');
define('ZIDPROXY_PID_FILE', '/var/run/zid-proxy.pid');
define('ZIDPROXY_RCFILE', '/usr/local/etc/rc.d/zid-proxy.sh');
define('ZIDPROXY_BINARY', '/usr/local/sbin/zid-proxy');
define('ZIDPROXY_LOGROTATE_BINARY', '/usr/local/sbin/zid-proxy-logrotate');
define('ZIDPROXY_LOGROTATE_CRON_DESCR', 'ZID Proxy: logrotate');
define('ZIDPROXY_WATCHDOG_SH', '/usr/local/sbin/zid-proxy-watchdog');
define('ZIDPROXY_WATCHDOG_CRON_DESCR', 'ZID Proxy: watchdog');
define('ZIDPROXY_ACTIVE_IPS_JSON', '/var/run/zid-proxy.active_ips.json');

function zidproxy_logrotate_cron_command($keep_days) {
	$keep = is_numeric($keep_days) ? (int)$keep_days : 7;
	if ($keep < 1) {
		$keep = 1;
	}
	return sprintf(
		'%s -log %s -keep-days %d -pid %s -hup >/dev/null 2>&1',
		escapeshellcmd(ZIDPROXY_LOGROTATE_BINARY),
		escapeshellarg(ZIDPROXY_LOG_FILE),
		$keep,
		escapeshellarg(ZIDPROXY_PID_FILE)
	);
}

function zidproxy_configure_cron_compat() {
	if (function_exists('configure_cron')) {
		@configure_cron();
	}
}

function zidproxy_config_lock_compat() {
	if (function_exists('config_lock')) {
		config_lock();
		return;
	}
	if (function_exists('lock')) {
		lock('config');
	}
}

function zidproxy_config_unlock_compat() {
	if (function_exists('config_unlock')) {
		config_unlock();
		return;
	}
	if (function_exists('unlock')) {
		unlock('config');
	}
}

function zidproxy_cron_job_present($id_or_descr, $command = null) {
	global $config;
	if (!is_array($config)) {
		return false;
	}

	// pfSense has used different keys across versions/packages.
	$lists = [];
	if (isset($config['cron']['item']) && is_array($config['cron']['item'])) {
		$lists[] = $config['cron']['item'];
	}
	if (isset($config['cron']['job']) && is_array($config['cron']['job'])) {
		$lists[] = $config['cron']['job'];
	}

	foreach ($lists as $jobs) {
		foreach ($jobs as $job) {
			if (!is_array($job)) {
				continue;
			}
			$job_id = (string)($job['id'] ?? $job['ident'] ?? $job['identifier'] ?? '');
			$job_descr = (string)($job['descr'] ?? '');
			$job_cmd = (string)($job['command'] ?? '');

			if ($job_id !== '' && $job_id === $id_or_descr) {
				return true;
			}
			if ($job_descr !== '' && ($job_descr === $id_or_descr || $job_descr === ZIDPROXY_LOGROTATE_CRON_DESCR)) {
				return true;
			}
			if ($command !== null && $job_cmd !== '' && $job_cmd === $command) {
				return true;
			}
		}
	}

	return false;
}

function zidproxy_remove_cron_jobs_where($predicate) {
	global $config;
	if (!is_array($config)) {
		return false;
	}

	$changed = false;
	foreach (['item', 'job'] as $key) {
		if (!isset($config['cron'][$key]) || !is_array($config['cron'][$key])) {
			continue;
		}
		$new = [];
		foreach ($config['cron'][$key] as $job) {
			if (!is_array($job)) {
				continue;
			}
			$remove = false;
			try {
				$remove = (bool)call_user_func($predicate, $job);
			} catch (Throwable $e) {
				$remove = false;
			}
			if ($remove) {
				$changed = true;
				continue;
			}
			$new[] = $job;
		}
		if ($changed) {
			$config['cron'][$key] = $new;
		}
	}

	return $changed;
}

function zidproxy_remove_cron_job_by_descr($descr) {
	zidproxy_config_lock_compat();
	$changed = zidproxy_remove_cron_jobs_where(function ($job) use ($descr) {
		return (string)($job['descr'] ?? '') === (string)$descr;
	});
	if ($changed) {
		zidproxy_configure_cron_compat();
		@write_config("ZID Proxy cron removed");
	}
	zidproxy_config_unlock_compat();
	return $changed;
}

function zidproxy_remove_cron_jobs_by_command_substr($needle) {
	$needle = (string)$needle;
	if ($needle === '') {
		return false;
	}
	zidproxy_config_lock_compat();
	$changed = zidproxy_remove_cron_jobs_where(function ($job) use ($needle) {
		$cmd = (string)($job['command'] ?? '');
		return $cmd !== '' && strpos($cmd, $needle) !== false;
	});
	if ($changed) {
		zidproxy_configure_cron_compat();
		@write_config("ZID Proxy cron removed");
	}
	zidproxy_config_unlock_compat();
	return $changed;
}

function zidproxy_upsert_cron_job($descr, $minute, $hour, $mday, $month, $wday, $who, $command) {
	global $config;
	if (!is_array($config)) {
		return false;
	}

	$targets = [];
	if (isset($config['cron']['item']) && is_array($config['cron']['item'])) {
		$targets[] = 'item';
	}
	if (isset($config['cron']['job']) && is_array($config['cron']['job'])) {
		$targets[] = 'job';
	}
	if (empty($targets)) {
		$config['cron'] = is_array($config['cron'] ?? null) ? $config['cron'] : [];
		$config['cron']['item'] = [];
		$targets[] = 'item';
	}

	foreach ($targets as $key) {
		foreach ($config['cron'][$key] as $idx => $job) {
			if (!is_array($job)) {
				continue;
			}
			if (($job['descr'] ?? '') === $descr) {
				$config['cron'][$key][$idx]['minute'] = $minute;
				$config['cron'][$key][$idx]['hour'] = $hour;
				$config['cron'][$key][$idx]['mday'] = $mday;
				$config['cron'][$key][$idx]['month'] = $month;
				$config['cron'][$key][$idx]['wday'] = $wday;
				$config['cron'][$key][$idx]['who'] = $who;
				$config['cron'][$key][$idx]['command'] = $command;
				$config['cron'][$key][$idx]['enabled'] = 'true';
				zidproxy_configure_cron_compat();
				@write_config("ZID Proxy cron updated");
				return true;
			}
		}
	}

	// Not found; create new entry (pfSense typically uses cron[item]).
	$config['cron'][$targets[0]][] = [
		'minute' => $minute,
		'hour' => $hour,
		'mday' => $mday,
		'month' => $month,
		'wday' => $wday,
		'who' => $who,
		'command' => $command,
		'descr' => $descr,
		'enabled' => 'true',
	];
	zidproxy_configure_cron_compat();
	@write_config("ZID Proxy cron installed");
	return true;
}

function zidproxy_install_cron_job_compat($minute, $hour, $mday, $month, $wday, $who, $command, $descr) {
	if (!function_exists('install_cron_job')) {
		// Fallback to a safe config upsert.
		zidproxy_config_lock_compat();
		$ok = zidproxy_upsert_cron_job($descr, $minute, $hour, $mday, $month, $wday, $who, $command);
		zidproxy_config_unlock_compat();
		return $ok;
	}

	// pfSense has shipped multiple install_cron_job() signatures historically.
	// Try known argument orderings safely.
	$attempts = [];
	// Common on pfSense: install_cron_job($command, $minute, $hour, $mday, $month, $wday, $who, $descr)
	$attempts[] = [$command, $minute, $hour, $mday, $month, $wday, $who, $descr];
	$attempts[] = [$command, $minute, $hour, $mday, $month, $wday, $who];
	// Some variants accept ($descr,$command,...)
	$attempts[] = [$descr, $command, $minute, $hour, $mday, $month, $wday, $who];
	$attempts[] = [$descr, $command, $minute, $hour, $mday, $month, $wday, $who, $descr];

	foreach ($attempts as $args) {
		// install_cron_job may emit warnings on signature mismatch; do best-effort suppress.
		zidproxy_config_lock_compat();
		@call_user_func_array('install_cron_job', $args);
		zidproxy_config_unlock_compat();
		if (zidproxy_cron_job_present($descr, $command)) {
			zidproxy_configure_cron_compat();
			@write_config("ZID Proxy cron updated");
			return true;
		}
	}

	// If install_cron_job() didn't take, ensure it's present by updating config directly.
	zidproxy_config_lock_compat();
	$ok = zidproxy_upsert_cron_job($descr, $minute, $hour, $mday, $month, $wday, $who, $command);
	zidproxy_config_unlock_compat();
	return $ok;
}

function zidproxy_remove_cron_job_compat($id, $command = null) {
	if (function_exists('remove_cron_job')) {
		try {
			remove_cron_job($id);
			return true;
		} catch (Throwable $e) {}
	}
	if (function_exists('uninstall_cron_job')) {
		try {
			uninstall_cron_job($id);
			return true;
		} catch (Throwable $e) {}
	}
	if (function_exists('delete_cron_job')) {
		try {
			delete_cron_job($id);
			return true;
		} catch (Throwable $e) {}
	}

	// Some variants require both id and command
	if ($command !== null && function_exists('uninstall_cron_job')) {
		try {
			uninstall_cron_job($id, $command);
			return true;
		} catch (Throwable $e) {}
	}
	if ($command !== null && function_exists('remove_cron_job')) {
		try {
			remove_cron_job($id, $command);
			zidproxy_configure_cron_compat();
			@write_config("ZID Proxy cron removed");
			return true;
		} catch (Throwable $e) {}
	}

	// Fallback by descr (more reliable)
	return zidproxy_remove_cron_job_by_descr($id);
}

function zidproxy_install_logrotate_cron() {
	global $config;
	$zidproxy_config = $config['installedpackages']['zidproxy']['config'][0] ?? [];
	$keep_days = $zidproxy_config['log_retention_days'] ?? '7';

	// Install only when helper binary exists.
	if (!file_exists(ZIDPROXY_LOGROTATE_BINARY)) {
		// Best-effort remove an existing cron if the binary is missing.
		zidproxy_remove_logrotate_cron();
		return false;
	}

	$cmd = zidproxy_logrotate_cron_command($keep_days);
	// Ensure we don't accumulate broken/duplicate cron entries from prior versions.
	zidproxy_config_lock_compat();
	zidproxy_remove_cron_jobs_where(function ($job) {
		$descr = (string)($job['descr'] ?? '');
		$command = (string)($job['command'] ?? '');
		if ($descr === ZIDPROXY_LOGROTATE_CRON_DESCR) {
			return true;
		}
		return $command !== '' && strpos($command, 'zid-proxy-logrotate') !== false;
	});
	$ok = zidproxy_upsert_cron_job(
		ZIDPROXY_LOGROTATE_CRON_DESCR,
		'0',  // minute
		'*',  // hour
		'*',  // mday
		'*',  // month
		'*',  // wday
		'root',
		$cmd
	);
	zidproxy_configure_cron_compat();
	@write_config("ZID Proxy logrotate cron updated");
	zidproxy_config_unlock_compat();
	return $ok;
}

function zidproxy_remove_logrotate_cron() {
	$cmd = zidproxy_logrotate_cron_command(7);
	$removed = false;
	$removed = zidproxy_remove_cron_job_compat(ZIDPROXY_LOGROTATE_CRON_DESCR, $cmd) || $removed;
	$removed = zidproxy_remove_cron_jobs_by_command_substr('zid-proxy-logrotate') || $removed;
	return $removed;
}

function zidproxy_watchdog_cron_command() {
	// Use `sh -c` to ensure shell redirection works even if cron uses a different shell.
	return sprintf(
		'/bin/sh -c %s',
		escapeshellarg(escapeshellcmd(ZIDPROXY_WATCHDOG_SH) . ' >/dev/null 2>&1')
	);
}

function zidproxy_install_watchdog_cron() {
	if (!file_exists(ZIDPROXY_WATCHDOG_SH)) {
		zidproxy_remove_watchdog_cron();
		return false;
	}

	$cmd = zidproxy_watchdog_cron_command();
	// Some pfSense versions have an install_cron_job() signature that can misplace fields.
	// For watchdog, always use a direct upsert into config to ensure correct columns.
	zidproxy_config_lock_compat();
	// Clean up any previously-created broken entries.
	zidproxy_remove_cron_jobs_where(function ($job) {
		$cmd = (string)($job['command'] ?? '');
		return $cmd !== '' && strpos($cmd, 'zid-proxy-watchdog') !== false;
	});
	$ok = zidproxy_upsert_cron_job(
		ZIDPROXY_WATCHDOG_CRON_DESCR,
		'*', // minute
		'*', // hour
		'*', // mday
		'*', // month
		'*', // wday
		'root',
		$cmd
	);
	zidproxy_configure_cron_compat();
	@write_config("ZID Proxy watchdog cron updated");
	zidproxy_config_unlock_compat();
	return $ok;
}

function zidproxy_remove_watchdog_cron() {
	$cmd = zidproxy_watchdog_cron_command();
	$removed = false;
	$removed = zidproxy_remove_cron_job_compat(ZIDPROXY_WATCHDOG_CRON_DESCR, $cmd) || $removed;
	$removed = zidproxy_remove_cron_jobs_by_command_substr('zid-proxy-watchdog') || $removed;
	return $removed;
}

function zidproxy_get_rules_mode() {
	global $config;
	$zidproxy_config = $config['installedpackages']['zidproxy']['config'][0] ?? [];
	$mode = strtolower(trim($zidproxy_config['rules_mode'] ?? 'legacy'));
	if ($mode !== 'groups' && $mode !== 'legacy') {
		$mode = 'legacy';
	}
	return $mode;
}

function zidproxy_ensure_config_defaults() {
	global $config;

	if (!is_array($config['installedpackages']['zidproxy']['config'])) {
		$config['installedpackages']['zidproxy']['config'] = array(array());
	}
	if (!is_array($config['installedpackages']['zidproxy']['config'][0] ?? null)) {
		$config['installedpackages']['zidproxy']['config'][0] = array();
	}

	$changed = false;
	$cfg = &$config['installedpackages']['zidproxy']['config'][0];

	$defaults = [
		'enable' => 'off',
		'interface' => 'all',
		'listen_port' => '3129',
		'timeout' => '30',
		'enable_logging' => 'on',
		'rules_mode' => 'legacy',
		'log_retention_days' => '7',
		'active_ips_timeout_seconds' => '120',
		'active_ips_refresh_seconds' => '5',
	];

	foreach ($defaults as $k => $v) {
		if (!isset($cfg[$k]) || trim((string)$cfg[$k]) === '') {
			$cfg[$k] = $v;
			$changed = true;
		}
	}

	// Normalize rules_mode if invalid
	$mode = strtolower(trim((string)($cfg['rules_mode'] ?? 'legacy')));
	if ($mode !== 'legacy' && $mode !== 'groups') {
		$cfg['rules_mode'] = 'legacy';
		$changed = true;
	}

	if ($changed) {
		write_config("ZID Proxy settings normalized");
	}
}

// Ensure defaults are present even before the first save, so pfSense "add/delete/edit"
// summary tables show values for columns like listen_port, enable_logging, rules_mode.
zidproxy_ensure_config_defaults();

function zidproxy_get_groups() {
	// Groups are stored in a local JSON file to avoid pfSense config.xml serialization/concurrency issues.
	if (!file_exists(ZIDPROXY_GROUPS_FILE)) {
		return [];
	}
	$raw = @file_get_contents(ZIDPROXY_GROUPS_FILE);
	if ($raw === false || trim($raw) === '') {
		return [];
	}
	$arr = json_decode($raw, true);
	if (!is_array($arr)) {
		return [];
	}
	return $arr;
}

function zidproxy_set_groups($groups) {
	if (!is_array($groups)) {
		$groups = [];
	}
	safe_mkdir(ZIDPROXY_BASE);
	$json = json_encode($groups, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
	if ($json === false) {
		$json = "[]\n";
	} else {
		$json .= "\n";
	}
	$tmp = ZIDPROXY_GROUPS_FILE . ".new";
	file_put_contents($tmp, $json);
	@chmod($tmp, 0600);
	rename($tmp, ZIDPROXY_GROUPS_FILE);
}

/**
 * Install hook - called when package is installed
 */
function zidproxy_install() {
	global $config;

	// Create configuration directory
	safe_mkdir(ZIDPROXY_BASE);

	// Ensure package settings have defaults (so the Settings summary table is complete)
	zidproxy_ensure_config_defaults();

	// Create default groups file if it doesn't exist (fresh installs)
	if (!file_exists(ZIDPROXY_GROUPS_FILE)) {
		$default_groups = [
			[
				'name' => 'access_restricted',
				'descr' => 'access_restricted',
				'members' => '',
				'rules' => ''
			],
			[
				'name' => 'access_controlled',
				'descr' => 'access_controlled',
				'members' => '',
				'rules' => ''
			],
			[
				'name' => 'access_full',
				'descr' => 'access_full',
				'members' => '',
				'rules' => ''
			],
		];
		zidproxy_set_groups($default_groups);
	}

	// Create default rules file if it doesn't exist
	if (!file_exists(ZIDPROXY_RULES_FILE)) {
		$default_rules = <<<'RULES'
# ZID Proxy Access Rules
# Format: TYPE;IP_OR_CIDR;HOSTNAME
# TYPE: ALLOW or BLOCK
# ALLOW rules take priority over BLOCK rules
# Default action (no match): ALLOW

# Example rules:
# BLOCK;192.168.1.0/24;*.facebook.com
# ALLOW;192.168.1.100;*.facebook.com
RULES;
		file_put_contents(ZIDPROXY_RULES_FILE, $default_rules);
	}

	// Create log file
	touch(ZIDPROXY_LOG_FILE);
	chmod(ZIDPROXY_LOG_FILE, 0644);

	// Install/update hourly logrotate cron (idempotent)
	zidproxy_install_logrotate_cron();

	// Sync configuration
	zidproxy_resync();

	return true;
}

/**
 * Deinstall hook - called when package is removed
 */
function zidproxy_deinstall() {
	// Stop the service
	zidproxy_stop();

	// Remove logrotate cron
	zidproxy_remove_logrotate_cron();
	// Remove watchdog cron
	zidproxy_remove_watchdog_cron();

	// Remove rc.conf entries
	unset_rcvar('zid_proxy_enable');
	unset_rcvar('zid_proxy_listen');

	return true;
}

/**
 * Validation hook - called before saving configuration
 */
function zidproxy_validate($post, &$input_errors) {
	if (!empty($post['listen_port'])) {
		if (!is_port($post['listen_port'])) {
			$input_errors[] = "Listen port must be a valid port number (1-65535).";
		}
	}

	if (!empty($post['timeout'])) {
		if (!is_numeric($post['timeout']) || $post['timeout'] < 1 || $post['timeout'] > 300) {
			$input_errors[] = "Timeout must be a number between 1 and 300 seconds.";
		}
	}

	if (!empty($post['log_retention_days'])) {
		if (!is_numeric($post['log_retention_days']) || $post['log_retention_days'] < 1 || $post['log_retention_days'] > 365) {
			$input_errors[] = "Log retention days must be a number between 1 and 365.";
		}
	}

	if (!empty($post['active_ips_timeout_seconds'])) {
		if (!is_numeric($post['active_ips_timeout_seconds']) || $post['active_ips_timeout_seconds'] < 10 || $post['active_ips_timeout_seconds'] > 86400) {
			$input_errors[] = "Active IPs timeout must be a number between 10 and 86400 seconds.";
		}
	}

	if (!empty($post['active_ips_refresh_seconds'])) {
		if (!is_numeric($post['active_ips_refresh_seconds']) || $post['active_ips_refresh_seconds'] < 1 || $post['active_ips_refresh_seconds'] > 300) {
			$input_errors[] = "Active IPs refresh must be a number between 1 and 300 seconds.";
		}
	}

	if (!empty($post['rules_mode'])) {
		$mode = strtolower(trim($post['rules_mode']));
		if ($mode !== 'legacy' && $mode !== 'groups') {
			$input_errors[] = "Rules mode must be legacy or groups.";
		}
	}
}

/**
 * Resync hook - called when configuration changes
 */
function zidproxy_resync() {
	global $config;

	zidproxy_ensure_config_defaults();

	$zidproxy_config = $config['installedpackages']['zidproxy']['config'][0] ?? [];

	// Write rc.d script
	zidproxy_write_rcfile();

	// Ensure hourly logrotate cron exists (idempotent) and reflects current retention settings.
	zidproxy_install_logrotate_cron();

	// Watchdog cron: only enabled when the service is enabled in Settings.
	if (isset($zidproxy_config['enable']) && $zidproxy_config['enable'] === 'on') {
		zidproxy_install_watchdog_cron();
	} else {
		zidproxy_remove_watchdog_cron();
	}

	if (isset($zidproxy_config['enable']) && $zidproxy_config['enable'] == 'on') {
		// Get configuration values
		$interface = $zidproxy_config['interface'] ?? 'all';
		$port = $zidproxy_config['listen_port'] ?? '3129';
		$timeout = $zidproxy_config['timeout'] ?? '30';
		$enable_logging = isset($zidproxy_config['enable_logging']) && $zidproxy_config['enable_logging'] == 'on';

		// Determine listen address
		if ($interface == 'all') {
			// Listen on all interfaces (0.0.0.0) for NAT compatibility
			$listen_addr = "0.0.0.0:{$port}";
		} else {
			// Listen on specific interface IP
			$if_ip = get_interface_ip($interface);
			if (empty($if_ip)) {
				$if_ip = '0.0.0.0';
			}
			$listen_addr = "{$if_ip}:{$port}";
		}

		// Write rc.conf variables
		set_rcvar('zid_proxy_enable', 'YES');
		set_rcvar('zid_proxy_listen', $listen_addr);
		set_rcvar('zid_proxy_rules', ZIDPROXY_RULES_FILE);
		set_rcvar('zid_proxy_log', $enable_logging ? ZIDPROXY_LOG_FILE : '/dev/null');
		$active_ips_timeout = $zidproxy_config['active_ips_timeout_seconds'] ?? '120';
		$active_ips_refresh = $zidproxy_config['active_ips_refresh_seconds'] ?? '5';
		set_rcvar('zid_proxy_active_ips_json', ZIDPROXY_ACTIVE_IPS_JSON);
		set_rcvar('zid_proxy_active_ips_timeout_seconds', $active_ips_timeout);
		set_rcvar('zid_proxy_active_ips_interval_seconds', $active_ips_refresh);

		// Start or restart service
		zidproxy_start();
	} else {
		set_rcvar('zid_proxy_enable', 'NO');
		zidproxy_stop();
	}

	// Sync rules from config to file
	zidproxy_sync_rules();
}

/**
 * Start the service
 */
function zidproxy_start() {
	$status = zidproxy_status();
	if ($status) {
		// Already running, restart
		zidproxy_stop();
		sleep(1);
	}
	mwexec('/usr/local/etc/rc.d/zid-proxy.sh start');
}

/**
 * Stop the service
 */
function zidproxy_stop() {
	if (file_exists(ZIDPROXY_PID_FILE)) {
		$pid = trim(file_get_contents(ZIDPROXY_PID_FILE));
		if (is_numeric($pid) && posix_kill($pid, 0)) {
			mwexec("/bin/kill {$pid}");
			// Wait for process to stop
			for ($i = 0; $i < 10; $i++) {
				if (!posix_kill($pid, 0)) {
					break;
				}
				usleep(100000);
			}
		}
		@unlink(ZIDPROXY_PID_FILE);
	}
	mwexec('/usr/local/etc/rc.d/zid-proxy.sh stop 2>/dev/null');
}

/**
 * Reload rules (send SIGHUP)
 */
function zidproxy_reload() {
	if (file_exists(ZIDPROXY_PID_FILE)) {
		$pid = trim(file_get_contents(ZIDPROXY_PID_FILE));
		if (is_numeric($pid) && posix_kill($pid, 0)) {
			posix_kill($pid, SIGHUP);
			return true;
		}
	}
	return false;
}

/**
 * Check if service is running
 */
function zidproxy_status() {
	if (file_exists(ZIDPROXY_PID_FILE)) {
		$pid = trim(file_get_contents(ZIDPROXY_PID_FILE));
		if (is_numeric($pid) && posix_kill($pid, 0)) {
			return true;
		}
	}
	return false;
}

/**
 * Get access rules from configuration
 */
function zidproxy_get_rules() {
	global $config;

	$rules = [];
	if (isset($config['installedpackages']['zidproxyrules']['config'])) {
		foreach ($config['installedpackages']['zidproxyrules']['config'] as $rule) {
			$rules[] = [
				'type' => $rule['type'] ?? 'BLOCK',
				'source' => $rule['source'] ?? '',
				'hostname' => $rule['hostname'] ?? '',
				'description' => $rule['description'] ?? ''
			];
		}
	}
	return $rules;
}

/**
 * Save access rules to configuration
 */
function zidproxy_save_rules($rules) {
	global $config;

	// If user is using group mode, ignore legacy rules UI writes.
	if (zidproxy_get_rules_mode() === 'groups') {
		write_config("ZID Proxy rules ignored (groups mode)");
		zidproxy_sync_rules();
		if (zidproxy_status()) {
			zidproxy_stop();
			sleep(1);
			zidproxy_start();
		}
		return;
	}

	$config['installedpackages']['zidproxyrules']['config'] = [];
	foreach ($rules as $rule) {
		$config['installedpackages']['zidproxyrules']['config'][] = [
			'type' => $rule['type'],
			'source' => $rule['source'],
			'hostname' => $rule['hostname'],
			'description' => $rule['description'] ?? ''
		];
	}

	write_config("ZID Proxy rules updated");
	zidproxy_sync_rules();

	// Ensure rules are reloaded by restarting the service
	// This is more reliable than SIGHUP in some cases
	if (zidproxy_status()) {
		zidproxy_stop();
		sleep(1);
		zidproxy_start();
	}
}

/**
 * Sync rules from pfSense config to rules file
 */
function zidproxy_sync_rules() {
	if (zidproxy_get_rules_mode() === 'groups') {
		zidproxy_sync_group_rules();
		return;
	}

	$rules = zidproxy_get_rules();

	$content = "# ZID Proxy Access Rules\n";
	$content .= "# Format: TYPE;IP_OR_CIDR;HOSTNAME\n";
	$content .= "# Auto-generated by pfSense - " . date('Y-m-d H:i:s') . "\n\n";

	foreach ($rules as $rule) {
		if (!empty($rule['source']) && !empty($rule['hostname'])) {
			$type = strtoupper($rule['type']);
			$content .= "{$type};{$rule['source']};{$rule['hostname']}";
			if (!empty($rule['description'])) {
				$content .= " # {$rule['description']}";
			}
			$content .= "\n";
		}
	}

	file_put_contents(ZIDPROXY_RULES_FILE, $content);
}

function zidproxy_sync_group_rules() {
	$groups = zidproxy_get_groups();

	$content = "# ZID Proxy Access Rules (GROUPS)\n";
	$content .= "# Order matters: first matching group wins\n";
	$content .= "# Auto-generated by pfSense - " . date('Y-m-d H:i:s') . "\n\n";

	foreach ($groups as $g) {
		$name = trim((string)($g['name'] ?? ''));
		if ($name === '') {
			continue;
		}
		$content .= "GROUP;{$name}\n";

		// Members and rules are stored as plain multi-line strings in config.xml
		$members_text = (string)($g['members'] ?? '');
		$members_lines = preg_split("/\\r\\n|\\n|\\r/", $members_text);
		foreach ($members_lines as $line) {
			$line = trim((string)$line);
			if ($line === '' || strpos($line, '#') === 0) {
				continue;
			}
			$hash = strpos($line, '#');
			if ($hash !== false) {
				$line = trim(substr($line, 0, $hash));
				if ($line === '') {
					continue;
				}
			}
			$content .= "MEMBER;{$line}\n";
		}

		$rules_text = (string)($g['rules'] ?? '');
		$rules_lines = preg_split("/\\r\\n|\\n|\\r/", $rules_text);
		foreach ($rules_lines as $line) {
			$line = trim((string)$line);
			if ($line === '' || strpos($line, '#') === 0) {
				continue;
			}

			$comment = '';
			$hash = strpos($line, '#');
			if ($hash !== false) {
				$comment = trim(substr($line, $hash + 1));
				$line = trim(substr($line, 0, $hash));
				if ($line === '') {
					continue;
				}
			}

			$parts = array_map('trim', explode(';', $line));
			if (count($parts) !== 2) {
				continue;
			}
			$type = strtoupper($parts[0]);
			$hostname = strtolower($parts[1]);
			if ($hostname === '' || ($type !== 'ALLOW' && $type !== 'BLOCK')) {
				continue;
			}
			$content .= "{$type};{$hostname}";
			if ($comment !== '') {
				$content .= " # {$comment}";
			}
			$content .= "\n";
		}

		$content .= "\n";
	}

	file_put_contents(ZIDPROXY_RULES_FILE, $content);
}

/**
 * Validate a single rule
 */
function zidproxy_validate_rule($rule, &$errors) {
	$errors = [];

	// Validate type
	if (!in_array(strtoupper($rule['type']), ['ALLOW', 'BLOCK'])) {
		$errors[] = "Rule type must be ALLOW or BLOCK.";
	}

	// Validate source IP/CIDR
	if (empty($rule['source'])) {
		$errors[] = "Source IP/CIDR is required.";
	} elseif (!is_ipaddr($rule['source']) && !is_subnet($rule['source'])) {
		$errors[] = "Source must be a valid IP address or CIDR notation.";
	}

	// Validate hostname
	if (empty($rule['hostname'])) {
		$errors[] = "Hostname is required.";
	}

	return empty($errors);
}

/**
 * Get recent log entries
 */
function zidproxy_get_log_entries($lines = 200) {
	if (!file_exists(ZIDPROXY_LOG_FILE)) {
		return [];
	}

	$output = [];
	exec("/usr/bin/tail -n " . escapeshellarg($lines) . " " . escapeshellarg(ZIDPROXY_LOG_FILE), $output);

	$entries = [];
	foreach (array_reverse($output) as $line) {
		$line = trim($line);
		if (empty($line)) {
			continue;
		}

		// Parse:
		// - New: TIMESTAMP | SOURCE_IP | HOSTNAME | GROUP | ACTION
		// - Old: TIMESTAMP | SOURCE_IP | HOSTNAME | ACTION
		$parts = array_map('trim', explode('|', $line));
		if (count($parts) >= 5) {
			$entries[] = [
				'timestamp' => $parts[0],
				'source_ip' => $parts[1],
				'hostname' => $parts[2],
				'group' => $parts[3],
				'action' => $parts[4]
			];
		} elseif (count($parts) >= 4) {
			$entries[] = [
				'timestamp' => $parts[0],
				'source_ip' => $parts[1],
				'hostname' => $parts[2],
				'group' => '',
				'action' => $parts[3]
			];
		}
	}

	return $entries;
}

/**
 * Clear log file
 */
function zidproxy_clear_log() {
	file_put_contents(ZIDPROXY_LOG_FILE, '');
}

/**
 * Write the rc.d script
 */
function zidproxy_write_rcfile() {
	$rcfile_content = <<<'RCFILE'
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
	: ${zid_proxy_active_ips_json:="/var/run/zid-proxy.active_ips.json"}
	: ${zid_proxy_active_ips_timeout_seconds:="120"}
	: ${zid_proxy_active_ips_interval_seconds:="5"}

pidfile="${zid_proxy_pid}"
procname="/usr/local/sbin/zid-proxy"
command="/usr/sbin/daemon"
	command_args="-f -p ${pidfile} ${procname} -listen ${zid_proxy_listen} -rules ${zid_proxy_rules} -log ${zid_proxy_log} -pid ${zid_proxy_pid} -active-ips ${zid_proxy_active_ips_json} -active-ips-timeout-seconds ${zid_proxy_active_ips_timeout_seconds} -active-ips-interval-seconds ${zid_proxy_active_ips_interval_seconds}"

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
RCFILE;

	file_put_contents(ZIDPROXY_RCFILE, $rcfile_content);
	chmod(ZIDPROXY_RCFILE, 0755);
}

/**
 * Helper function to set rc.conf variable
 */
function set_rcvar($var, $value) {
	$rcconf = '/etc/rc.conf.local';
	$content = file_exists($rcconf) ? file_get_contents($rcconf) : '';

	// Remove existing entry
	$content = preg_replace("/^{$var}=.*\n?/m", '', $content);

	// Add new entry
	$content .= "{$var}=\"{$value}\"\n";

	file_put_contents($rcconf, $content);
}

/**
 * Helper function to unset rc.conf variable
 */
function unset_rcvar($var) {
	$rcconf = '/etc/rc.conf.local';
	if (!file_exists($rcconf)) {
		return;
	}

	$content = file_get_contents($rcconf);
	$content = preg_replace("/^{$var}=.*\n?/m", '', $content);
	file_put_contents($rcconf, $content);
}

?>
