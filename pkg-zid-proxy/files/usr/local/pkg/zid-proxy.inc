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

define('ZIDPROXY_BASE', '/usr/local/etc/zid-proxy');
define('ZIDPROXY_RULES_FILE', ZIDPROXY_BASE . '/access_rules.txt');
define('ZIDPROXY_LOG_FILE', '/var/log/zid-proxy.log');
define('ZIDPROXY_PID_FILE', '/var/run/zid-proxy.pid');
define('ZIDPROXY_RCFILE', '/usr/local/etc/rc.d/zid-proxy.sh');
define('ZIDPROXY_BINARY', '/usr/local/sbin/zid-proxy');

/**
 * Install hook - called when package is installed
 */
function zidproxy_install() {
	global $config;

	// Create configuration directory
	safe_mkdir(ZIDPROXY_BASE);

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
}

/**
 * Resync hook - called when configuration changes
 */
function zidproxy_resync() {
	global $config;

	$zidproxy_config = $config['installedpackages']['zidproxy']['config'][0] ?? [];

	// Write rc.d script
	zidproxy_write_rcfile();

	if (isset($zidproxy_config['enable']) && $zidproxy_config['enable'] == 'on') {
		// Get configuration values
		$interface = $zidproxy_config['interface'] ?? 'lan';
		$port = $zidproxy_config['listen_port'] ?? '3129';
		$timeout = $zidproxy_config['timeout'] ?? '30';
		$enable_logging = isset($zidproxy_config['enable_logging']) && $zidproxy_config['enable_logging'] == 'on';

		// Get interface IP
		$if_ip = get_interface_ip($interface);
		if (empty($if_ip)) {
			$if_ip = '0.0.0.0';
		}

		$listen_addr = "{$if_ip}:{$port}";

		// Write rc.conf variables
		set_rcvar('zid_proxy_enable', 'YES');
		set_rcvar('zid_proxy_listen', $listen_addr);
		set_rcvar('zid_proxy_rules', ZIDPROXY_RULES_FILE);
		set_rcvar('zid_proxy_log', $enable_logging ? ZIDPROXY_LOG_FILE : '/dev/null');

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
	zidproxy_reload();
}

/**
 * Sync rules from pfSense config to rules file
 */
function zidproxy_sync_rules() {
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

		// Parse: TIMESTAMP | SOURCE_IP | HOSTNAME | ACTION
		$parts = array_map('trim', explode('|', $line));
		if (count($parts) >= 4) {
			$entries[] = [
				'timestamp' => $parts[0],
				'source_ip' => $parts[1],
				'hostname' => $parts[2],
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
