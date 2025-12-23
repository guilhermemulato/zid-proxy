<?php
/*
 * zid-proxy_agent.php
 *
 * Agent integration settings for ZID Proxy.
 * The desktop agent sends machine/user info to this pfSense host over HTTP (LAN-only).
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Agent"));
$shortcut_section = "zidproxy";

global $config;
zidproxy_ensure_config_defaults();
$zidcfg = $config['installedpackages']['zidproxy']['config'][0] ?? [];

$savemsg = '';
$input_errors = [];

function zidproxy_agent_listen_addr($iface, $port) {
	$iface = (string)$iface;
	$port = (string)$port;
	if ($port === '' || !is_port($port)) {
		return '';
	}
	if ($iface === 'all') {
		return "0.0.0.0:{$port}";
	}
	$ip = get_interface_ip($iface);
	if (empty($ip)) {
		$ip = '0.0.0.0';
	}
	return "{$ip}:{$port}";
}

if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_POST['save'])) {
	$post = $_POST;
	$input_errors = [];

	zidproxy_validate($post, $input_errors);

	if (empty($input_errors)) {
		if (!is_array($config['installedpackages']['zidproxy']['config'])) {
			$config['installedpackages']['zidproxy']['config'] = array(array());
		}
		if (!is_array($config['installedpackages']['zidproxy']['config'][0] ?? null)) {
			$config['installedpackages']['zidproxy']['config'][0] = array();
		}

		$existing = $config['installedpackages']['zidproxy']['config'][0];
		if (!is_array($existing)) {
			$existing = [];
		}

		$new = $existing;
		if (isset($post['agent_interface']) && trim((string)$post['agent_interface']) !== '') {
			$new['agent_interface'] = trim((string)$post['agent_interface']);
		}
		if (isset($post['agent_listen_port']) && trim((string)$post['agent_listen_port']) !== '') {
			$new['agent_listen_port'] = trim((string)$post['agent_listen_port']);
		}
		if (isset($post['agent_ttl_seconds']) && trim((string)$post['agent_ttl_seconds']) !== '') {
			$new['agent_ttl_seconds'] = trim((string)$post['agent_ttl_seconds']);
		}

		$config['installedpackages']['zidproxy']['config'][0] = $new;
		write_config("ZID Proxy agent settings updated");

		zidproxy_resync();

		$savemsg = gettext("Configuration saved.");
		$zidcfg = $new;
	}
}

$pconfig = [
	'agent_interface' => $zidcfg['agent_interface'] ?? 'lan',
	'agent_listen_port' => $zidcfg['agent_listen_port'] ?? '18443',
	'agent_ttl_seconds' => $zidcfg['agent_ttl_seconds'] ?? '60',
];

$listen_addr = zidproxy_agent_listen_addr($pconfig['agent_interface'], $pconfig['agent_listen_port']);

include("head.inc");

// Tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Active IPs"), false, "/zid-proxy_active_ips.php");
$tab_array[] = array(gettext("Agent"), true, "/zid-proxy_agent.php");
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
?>

<div class="panel panel-default">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Agent Listener')?></h2></div>
	<div class="panel-body">
		<p>
			<?=gettext('The desktop agent posts machine/user information to this HTTP endpoint (recommended: allow only from LAN).')?>
		</p>
		<p>
			<?=gettext('Current listen address:')?> <code><?=htmlspecialchars($listen_addr)?></code>
			<br/>
			<?=gettext('Heartbeat endpoint:')?> <code><?=htmlspecialchars("http://{$listen_addr}/api/v1/agent/heartbeat")?></code>
		</p>
	</div>
</div>

<?php
$form = new Form();
$section = new Form_Section('Agent Settings');

$ifaces = array('all' => gettext('All Interfaces (0.0.0.0)'));
if (function_exists('get_configured_interface_with_descr')) {
	$cfg_ifaces = get_configured_interface_with_descr();
	if (is_array($cfg_ifaces)) {
		foreach ($cfg_ifaces as $if => $descr) {
			$ifaces[$if] = $descr;
		}
	}
} else {
	$ifaces['lan'] = 'LAN';
	$ifaces['wan'] = 'WAN';
}

$section->addInput(new Form_Select(
	'agent_interface',
	gettext('Listen Interface'),
	$pconfig['agent_interface'],
	$ifaces
))->setHelp(gettext('Recommended: LAN.'));

$section->addInput(new Form_Input(
	'agent_listen_port',
	gettext('Listen Port'),
	'number',
	$pconfig['agent_listen_port']
))->setHelp(gettext('Default: 18443 (HTTP).'));

$section->addInput(new Form_Input(
	'agent_ttl_seconds',
	gettext('Identity TTL (s)'),
	'number',
	$pconfig['agent_ttl_seconds']
))->setHelp(gettext('Clear Machine/User after this many seconds without heartbeat. Default: 60.'));

$form->add($section);

print($form);

include("foot.inc");
?>
