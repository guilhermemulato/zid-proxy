<?php
/*
 * zid-proxy_settings.php
 *
 * Custom Settings page for ZID Proxy (pfSense) with:
 * - Configuration form (enable/interface/port/logging/rules mode/timeout/log retention)
 * - Installed version display
 * - Update button (runs /usr/local/sbin/zid-proxy-update)
 * - Service controls (start/stop/restart)
 */

require_once("guiconfig.inc");
require_once("services.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Settings"));
$pglinks = array("", "/zid-proxy_settings.php", "@self");
$shortcut_section = "zidproxy";

// pfSense compatibility: some versions don’t expose config_lock()/config_unlock().
if (!function_exists('config_lock')) {
	function config_lock() {
		if (function_exists('lock')) {
			lock('config');
		}
	}
}
if (!function_exists('config_unlock')) {
	function config_unlock() {
		if (function_exists('unlock')) {
			unlock('config');
		}
	}
}

global $config;
zidproxy_ensure_config_defaults();
$zidcfg = $config['installedpackages']['zidproxy']['config'][0] ?? [];

function zidproxy_installed_version_line() {
	$bin = ZIDPROXY_BINARY;
	if (!is_executable($bin)) {
		return gettext("Not installed");
	}
	$out = [];
	$rc = 0;
	exec(escapeshellcmd($bin) . " -version 2>&1", $out, $rc);
	if ($rc !== 0 || empty($out)) {
		return gettext("Unknown");
	}
	return trim($out[0]);
}

$savemsg = '';
$update_msg = '';
$input_errors = [];

// Handle service/update actions
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
	if (isset($_POST['svc_start'])) {
		zidproxy_start();
		$savemsg = gettext("Service start requested.");
	} elseif (isset($_POST['svc_stop'])) {
		zidproxy_stop();
		$savemsg = gettext("Service stop requested.");
	} elseif (isset($_POST['svc_restart'])) {
		zidproxy_stop();
		sleep(1);
		zidproxy_start();
		$savemsg = gettext("Service restart requested.");
	} elseif (isset($_POST['run_update'])) {
		$cmd = "/bin/sh /usr/local/sbin/zid-proxy-update 2>&1";
		$out = [];
		$rc = 0;
		exec($cmd, $out, $rc);
		$joined = trim(implode("\n", $out));

		if (stripos($joined, "Already up-to-date") !== false) {
			$update_msg = $joined;
		} elseif ($rc === 0) {
			$update_msg = "done";
		} else {
			$update_msg = $joined !== '' ? $joined : sprintf(gettext("Update failed (exit %d)."), $rc);
		}
	} elseif (isset($_POST['save_settings'])) {
		$post = $_POST;
		$input_errors = [];

		zidproxy_validate($post, $input_errors);

		// Normalize values
		$new = [];
		$new['enable'] = isset($post['enable']) && $post['enable'] === 'on' ? 'on' : 'off';
		$new['interface'] = $post['interface'] ?? 'all';
		$new['listen_port'] = trim((string)($post['listen_port'] ?? '3129'));
		$new['enable_logging'] = isset($post['enable_logging']) && $post['enable_logging'] === 'on' ? 'on' : 'off';
		$new['rules_mode'] = strtolower(trim((string)($post['rules_mode'] ?? 'legacy')));
		$new['timeout'] = trim((string)($post['timeout'] ?? '30'));
		$new['log_retention_days'] = trim((string)($post['log_retention_days'] ?? '7'));

		if (empty($input_errors)) {
			config_lock();
			if (!is_array($config['installedpackages']['zidproxy']['config'])) {
				$config['installedpackages']['zidproxy']['config'] = array(array());
			}
			$config['installedpackages']['zidproxy']['config'][0] = $new;
			write_config("ZID Proxy settings updated");
			config_unlock();

			zidproxy_resync();

			$savemsg = gettext("Configuration saved.");
			$zidcfg = $new;
		}
	}
}

// Populate form config
$pconfig = [
	'enable' => $zidcfg['enable'] ?? 'off',
	'interface' => $zidcfg['interface'] ?? 'all',
	'listen_port' => $zidcfg['listen_port'] ?? '3129',
	'enable_logging' => $zidcfg['enable_logging'] ?? 'on',
	'rules_mode' => $zidcfg['rules_mode'] ?? 'legacy',
	'timeout' => $zidcfg['timeout'] ?? '30',
	'log_retention_days' => $zidcfg['log_retention_days'] ?? '7',
];

include("head.inc");

// Tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), true, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Groups"), false, "/zid-proxy_groups.php");
$tab_array[] = array(gettext("Access Rules"), false, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("Logs"), false, "/zid-proxy_log.php");
display_top_tabs($tab_array);

if (!empty($input_errors)) {
	print_input_errors($input_errors);
}
if (!empty($savemsg)) {
	print_info_box($savemsg, 'success');
}
if (!empty($update_msg)) {
	print_info_box(htmlspecialchars($update_msg), 'info');
}
?>

<div class="panel panel-default">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Installed Version')?></h2></div>
	<div class="panel-body">
		<code><?=htmlspecialchars(zidproxy_installed_version_line())?></code>
	</div>
</div>

<div class="panel panel-default">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Service Controls')?></h2></div>
	<div class="panel-body">
		<form method="post">
			<?php if (zidproxy_status()): ?>
				<span class="label label-success"><?=gettext('Running')?></span>
				&nbsp;
				<button type="submit" name="svc_stop" class="btn btn-sm btn-warning">
					<i class="fa fa-stop"></i> <?=gettext('Stop')?>
				</button>
				<button type="submit" name="svc_restart" class="btn btn-sm btn-primary">
					<i class="fa fa-refresh"></i> <?=gettext('Restart')?>
				</button>
			<?php else: ?>
				<span class="label label-danger"><?=gettext('Stopped')?></span>
				&nbsp;
				<button type="submit" name="svc_start" class="btn btn-sm btn-success">
					<i class="fa fa-play"></i> <?=gettext('Start')?>
				</button>
			<?php endif; ?>

			<button type="submit" name="run_update" class="btn btn-sm btn-default pull-right"
			        onclick="return confirm('<?=gettext("Run update now?")?>');">
				<i class="fa fa-download"></i> <?=gettext('Update')?>
			</button>
		</form>
	</div>
</div>

<?php
// Settings form
$form = new Form();
$form->setAction('zid-proxy_settings.php');
$section = new Form_Section('General Settings');

$section->addInput(new Form_Checkbox(
	'enable',
	gettext('Enable'),
	gettext('Enable ZID Proxy service'),
	($pconfig['enable'] === 'on')
));

$if = new Form_Select(
	'interface',
	gettext('Listen Interface'),
	$pconfig['interface'],
	[
		'all' => gettext('All Interfaces (0.0.0.0) - Recommended for NAT'),
		'lan' => gettext('LAN'),
		'wan' => gettext('WAN'),
	]
);
$if->setHelp(gettext('Select the interface where ZID Proxy will listen.'));
$section->addInput($if);

$section->addInput(new Form_Input(
	'listen_port',
	gettext('Listen Port'),
	'number',
	$pconfig['listen_port']
))->setHelp(gettext('Port to listen on. Default: 3129.'));

$section->addInput(new Form_Checkbox(
	'enable_logging',
	gettext('Enable Logging'),
	gettext('Enable connection logging to /var/log/zid-proxy.log'),
	($pconfig['enable_logging'] === 'on')
));

$section2 = new Form_Section('Advanced Settings');

$rm = new Form_Select(
	'rules_mode',
	gettext('Rules Mode'),
	$pconfig['rules_mode'],
	[
		'legacy' => gettext('Legacy (IP/CIDR + Hostname rules)'),
		'groups' => gettext('Groups (ordered groups + hostname rules)'),
	]
);
$rm->setHelp(gettext('Select how rules are generated.'));
$section2->addInput($rm);

$section2->addInput(new Form_Input(
	'timeout',
	gettext('Connection Timeout (s)'),
	'number',
	$pconfig['timeout']
))->setHelp(gettext('Default: 30 (range 1-300).'));

$section2->addInput(new Form_Input(
	'log_retention_days',
	gettext('Log Retention Days'),
	'number',
	$pconfig['log_retention_days']
))->setHelp(gettext('How many days of daily rotated logs to keep. Default: 7 (range 1-365).'));

$form->add($section);
$form->add($section2);

$form->addGlobal(new Form_Button(
	'save_settings',
	gettext('Save'),
	null,
	'fa-save'
))->addClass('btn-primary');

print($form);
?>

<?php include("foot.inc"); ?>
