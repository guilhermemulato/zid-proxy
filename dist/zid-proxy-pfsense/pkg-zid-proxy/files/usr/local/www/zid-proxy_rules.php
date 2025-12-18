<?php
/*
 * zid-proxy_rules.php
 *
 * Access rules management page for ZID Proxy
 *
 * Licensed under the Apache License, Version 2.0
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Access Rules"));
$pglinks = array("", "/zid-proxy_settings.php", "@self");
$shortcut_section = "zidproxy";

// Get rules from config
global $config;
if (zidproxy_get_rules_mode() === 'groups') {
	include("head.inc");
	$tab_array = array();
	$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
	$tab_array[] = array(gettext("Active IPs"), false, "/zid-proxy_active_ips.php");
	$tab_array[] = array(gettext("Groups"), false, "/zid-proxy_groups.php");
	$tab_array[] = array(gettext("Access Rules"), true, "/zid-proxy_rules.php");
	$tab_array[] = array(gettext("Logs"), false, "/zid-proxy_log.php");
	display_top_tabs($tab_array);
	print_info_box(gettext("Rules Mode is set to Groups. Use the Groups tab to manage rules."), 'info');
	include("foot.inc");
	exit;
}

if (!is_array($config['installedpackages']['zidproxyrules']['config'])) {
	$config['installedpackages']['zidproxyrules']['config'] = array();
}
$a_rules = &$config['installedpackages']['zidproxyrules']['config'];

// Handle actions
if ($_POST['act'] == 'del' && isset($_POST['id'])) {
	$id = $_POST['id'];
	if ($a_rules[$id]) {
		unset($a_rules[$id]);
		$a_rules = array_values($a_rules);
		write_config("ZID Proxy rule deleted");
		zidproxy_sync_rules();
		zidproxy_reload();
	}
	header("Location: zid-proxy_rules.php");
	exit;
}

if ($_POST['act'] == 'toggle' && isset($_POST['id'])) {
	$id = $_POST['id'];
	if ($a_rules[$id]) {
		$a_rules[$id]['disabled'] = $a_rules[$id]['disabled'] ? '' : 'yes';
		write_config("ZID Proxy rule toggled");
		zidproxy_sync_rules();
		zidproxy_reload();
	}
	header("Location: zid-proxy_rules.php");
	exit;
}

// Handle form submission for adding/editing rules
if ($_POST['save']) {
	$pconfig = $_POST;
	$input_errors = array();

	// Validate
	$rule = array(
		'type' => $pconfig['type'],
		'source' => $pconfig['source'],
		'hostname' => $pconfig['hostname'],
		'description' => $pconfig['description']
	);

	zidproxy_validate_rule($rule, $rule_errors);
	$input_errors = array_merge($input_errors, $rule_errors);

	if (empty($input_errors)) {
		if (isset($pconfig['id']) && $pconfig['id'] !== '') {
			$a_rules[$pconfig['id']] = $rule;
		} else {
			$a_rules[] = $rule;
		}

		write_config("ZID Proxy rule saved");
		zidproxy_sync_rules();
		zidproxy_reload();

		header("Location: zid-proxy_rules.php");
		exit;
	}
}

// Check if editing
$id = $_GET['id'] ?? $_POST['id'] ?? null;
if (isset($_GET['act']) && $_GET['act'] == 'edit' && isset($id) && $a_rules[$id]) {
	$pconfig = $a_rules[$id];
	$pconfig['id'] = $id;
} elseif (!$_POST['save']) {
	$pconfig = array(
		'type' => 'BLOCK',
		'source' => '',
		'hostname' => '',
		'description' => ''
	);
}

include("head.inc");

// Display tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Active IPs"), false, "/zid-proxy_active_ips.php");
$tab_array[] = array(gettext("Groups"), false, "/zid-proxy_groups.php");
$tab_array[] = array(gettext("Access Rules"), true, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("Logs"), false, "/zid-proxy_log.php");
display_top_tabs($tab_array);

if ($input_errors) {
	print_input_errors($input_errors);
}

// Rule edit form
$form = new Form();

$section = new Form_Section('Add/Edit Rule');

$section->addInput(new Form_Input(
	'id',
	null,
	'hidden',
	$pconfig['id'] ?? ''
));

$section->addInput(new Form_Select(
	'type',
	'*Rule Type',
	$pconfig['type'],
	array(
		'BLOCK' => 'BLOCK',
		'ALLOW' => 'ALLOW'
	)
))->setHelp('ALLOW rules take priority over BLOCK rules.');

$section->addInput(new Form_Input(
	'source',
	'*Source IP/CIDR',
	'text',
	$pconfig['source']
))->setHelp('Source IP address or CIDR notation (e.g., 192.168.1.0/24 or 10.0.0.1)');

$section->addInput(new Form_Input(
	'hostname',
	'*Destination Hostname',
	'text',
	$pconfig['hostname']
))->setHelp('Destination hostname pattern. Supports wildcards (e.g., *.facebook.com)');

$section->addInput(new Form_Input(
	'description',
	'Description',
	'text',
	$pconfig['description']
))->setHelp('Optional description for this rule');

$form->add($section);

print($form);

?>

<div class="panel panel-default">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Current Rules')?></h2></div>
	<div class="panel-body">
		<div class="table-responsive">
			<table class="table table-striped table-hover table-condensed sortable-theme-bootstrap" data-sortable>
				<thead>
					<tr>
						<th><?=gettext('Type')?></th>
						<th><?=gettext('Source')?></th>
						<th><?=gettext('Hostname')?></th>
						<th><?=gettext('Description')?></th>
						<th><?=gettext('Actions')?></th>
					</tr>
				</thead>
				<tbody>
<?php
$i = 0;
foreach ($a_rules as $rule):
	$type_class = ($rule['type'] == 'ALLOW') ? 'success' : 'danger';
?>
					<tr>
						<td>
							<span class="label label-<?=$type_class?>"><?=htmlspecialchars($rule['type'])?></span>
						</td>
						<td><?=htmlspecialchars($rule['source'])?></td>
						<td><?=htmlspecialchars($rule['hostname'])?></td>
						<td><?=htmlspecialchars($rule['description'])?></td>
						<td>
							<a class="fa fa-pencil" title="<?=gettext('Edit')?>" href="?act=edit&amp;id=<?=$i?>"></a>
							<a class="fa fa-trash text-danger" title="<?=gettext('Delete')?>" href="#" onclick="deleteRule(<?=$i?>); return false;"></a>
						</td>
					</tr>
<?php
	$i++;
endforeach;

if (empty($a_rules)):
?>
					<tr>
						<td colspan="5" class="text-center">
							<?=gettext('No rules configured. Add a rule above.')?>
						</td>
					</tr>
<?php endif; ?>
				</tbody>
			</table>
		</div>
	</div>
</div>

<div class="panel panel-info">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Rule Matching Logic')?></h2></div>
	<div class="panel-body">
		<ul>
			<li><strong>ALLOW</strong> rules take priority over <strong>BLOCK</strong> rules</li>
			<li>If no rule matches, the connection is <strong>ALLOWED</strong> (default)</li>
			<li>Wildcards: <code>*.example.com</code> matches <code>www.example.com</code>, <code>api.example.com</code>, and <code>example.com</code></li>
			<li>CIDR notation: <code>192.168.1.0/24</code> matches all IPs from 192.168.1.0 to 192.168.1.255</li>
		</ul>
	</div>
</div>

<script type="text/javascript">
//<![CDATA[
function deleteRule(id) {
	if (confirm('<?=gettext("Are you sure you want to delete this rule?")?>')) {
		var form = document.createElement('form');
		form.method = 'POST';
		form.action = 'zid-proxy_rules.php';

		var actInput = document.createElement('input');
		actInput.type = 'hidden';
		actInput.name = 'act';
		actInput.value = 'del';
		form.appendChild(actInput);

		var idInput = document.createElement('input');
		idInput.type = 'hidden';
		idInput.name = 'id';
		idInput.value = id;
		form.appendChild(idInput);

		document.body.appendChild(form);
		form.submit();
	}
}
//]]>
</script>

<?php include("foot.inc"); ?>
