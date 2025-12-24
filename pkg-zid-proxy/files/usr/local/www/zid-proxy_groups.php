<?php
/*
 * zid-proxy_groups.php
 *
 * Group-based rules management page for ZID Proxy
 *
 * Format generated to /usr/local/etc/zid-proxy/access_rules.txt:
 *   GROUP;name
 *   MEMBER;IP_OR_CIDR
 *   ALLOW;HOSTNAME
 *   BLOCK;HOSTNAME
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Groups"));
$pglinks = array("", "/pkg.php?xml=zid-proxy.xml", "@self");
$shortcut_section = "zidproxy";

// pfSense compatibility: some versions don’t expose config_lock()/config_unlock().
// Provide wrappers using lock()/unlock() so config.xml writes remain serialized.
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
$a_groups = zidproxy_get_groups();
if (!is_array($a_groups)) {
	$a_groups = [];
}

// Helpers
function zidproxy_groups_textarea_lines($s) {
	$lines = preg_split("/\\r\\n|\\n|\\r/", (string)$s);
	$out = [];
	foreach ($lines as $line) {
		$line = trim($line);
		if ($line === '' || strpos($line, '#') === 0) {
			continue;
		}
		// Strip inline comments
		$hash = strpos($line, '#');
		if ($hash !== false) {
			$line = trim(substr($line, 0, $hash));
			if ($line === '') {
				continue;
			}
		}
		$out[] = $line;
	}
	return $out;
}

function zidproxy_groups_normalize_name($name) {
	return strtolower(trim((string)$name));
}

// Handle actions
if ($_POST['act'] == 'del' && isset($_POST['id'])) {
	$id = $_POST['id'];
	if (isset($a_groups[$id])) {
		unset($a_groups[$id]);
		$a_groups = array_values($a_groups);
		zidproxy_set_groups($a_groups);
		zidproxy_sync_rules();
		zidproxy_reload();
	}
	header("Location: zid-proxy_groups.php");
	exit;
}

if ($_POST['act'] == 'moveup' && isset($_POST['id'])) {
	$id = (int)$_POST['id'];
	if ($id > 0 && isset($a_groups[$id]) && isset($a_groups[$id - 1])) {
		$tmp = $a_groups[$id - 1];
		$a_groups[$id - 1] = $a_groups[$id];
		$a_groups[$id] = $tmp;
		zidproxy_set_groups($a_groups);
		zidproxy_sync_rules();
		zidproxy_reload();
	}
	header("Location: zid-proxy_groups.php");
	exit;
}

if ($_POST['act'] == 'movedown' && isset($_POST['id'])) {
	$id = (int)$_POST['id'];
	if (isset($a_groups[$id]) && isset($a_groups[$id + 1])) {
		$tmp = $a_groups[$id + 1];
		$a_groups[$id + 1] = $a_groups[$id];
		$a_groups[$id] = $tmp;
		zidproxy_set_groups($a_groups);
		zidproxy_sync_rules();
		zidproxy_reload();
	}
	header("Location: zid-proxy_groups.php");
	exit;
}

// Handle save (add/edit)
if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_POST['save'])) {
	$pconfig = $_POST;
	$input_errors = array();

	$id = isset($pconfig['id']) && $pconfig['id'] !== '' ? (int)$pconfig['id'] : null;
	$name = zidproxy_groups_normalize_name($pconfig['name'] ?? '');
	$descr = trim((string)($pconfig['descr'] ?? ''));

	if ($name === '') {
		$input_errors[] = gettext("Group name is required.");
	} elseif (!preg_match('/^[a-z0-9_-]{2,64}$/', $name)) {
		$input_errors[] = gettext("Group name must be 2-64 chars (a-z, 0-9, underscore, dash).");
	} else {
		foreach ($a_groups as $idx => $g) {
			if ($id !== null && $idx === $id) {
				continue;
			}
			if (zidproxy_groups_normalize_name($g['name'] ?? '') === $name) {
				$input_errors[] = gettext("Group name must be unique.");
				break;
			}
		}
	}

	// Members
	$members_lines = zidproxy_groups_textarea_lines($pconfig['members'] ?? '');
	$members = [];
	foreach ($members_lines as $line) {
		if (!is_ipaddr($line) && !is_subnet($line)) {
			$input_errors[] = sprintf(gettext("Invalid member IP/CIDR: %s"), htmlspecialchars($line));
			continue;
		}
		$members[] = $line;
	}

	// Rules: "ALLOW;hostname" / "BLOCK;hostname" (optional inline comment after #)
	$rules_lines = zidproxy_groups_textarea_lines($pconfig['rules'] ?? '');
	$normalized_rules_lines = [];
	foreach ($rules_lines as $line) {
		$parts = array_map('trim', explode(';', $line));
		if (count($parts) != 2) {
			$input_errors[] = sprintf(gettext("Invalid rule format: %s (expected TYPE;HOSTNAME)"), htmlspecialchars($line));
			continue;
		}
		$type = strtoupper($parts[0]);
		$hostname = strtolower($parts[1]);
		if (!in_array($type, ['ALLOW', 'BLOCK'])) {
			$input_errors[] = sprintf(gettext("Invalid rule type: %s"), htmlspecialchars($parts[0]));
			continue;
		}
		if ($hostname === '') {
			$input_errors[] = gettext("Rule hostname cannot be empty.");
			continue;
		}
		$normalized_rules_lines[] = $type . ';' . $hostname;
	}

	if (empty($members)) {
		$input_errors[] = gettext("At least one MEMBER is required per group.");
	}

	$group = [
		'name' => $name,
		'descr' => $descr,
		// Store as plain strings to keep pfSense config.xml valid.
		// Arrays of raw strings can produce invalid XML in pfSense config serialization.
		'members' => implode("\n", $members),
		'rules' => implode("\n", $normalized_rules_lines)
	];

	if (empty($input_errors)) {
		if ($id !== null && isset($a_groups[$id])) {
			$a_groups[$id] = $group;
		} else {
			$a_groups[] = $group;
		}
		zidproxy_set_groups($a_groups);
		zidproxy_sync_rules();
		zidproxy_reload();

		header("Location: zid-proxy_groups.php");
		exit;
	}
}

// Check if editing
$id = $_GET['id'] ?? $_POST['id'] ?? null;
if (isset($_GET['act']) && $_GET['act'] == 'edit' && isset($id) && isset($a_groups[$id])) {
	$pconfig = $a_groups[$id];
	$pconfig['id'] = $id;
	$pconfig['members_text'] = (string)($pconfig['members'] ?? '');
	$pconfig['rules_text'] = (string)($pconfig['rules'] ?? '');
} elseif (!$_POST['save']) {
	$pconfig = array(
		'id' => '',
		'name' => '',
		'descr' => '',
		'members_text' => '',
		'rules_text' => "BLOCK;*.facebook.com\nBLOCK;*.twitter.com"
	);
}

include("head.inc");

// CSRF tokens for dynamic action forms (move/delete)
$csrf_token_key = '';
$csrf_token = '';
if (function_exists('csrf_get_tokens')) {
	$csrf_tokens = csrf_get_tokens();
	if (is_array($csrf_tokens)) {
		$csrf_token_key = (string)($csrf_tokens['csrf_token_key'] ?? '');
		$csrf_token = (string)($csrf_tokens['csrf_token'] ?? '');
	}
}
if ($csrf_token_key === '' && function_exists('csrf_get_token_key')) {
	$csrf_token_key = (string)csrf_get_token_key();
}
if ($csrf_token === '' && function_exists('csrf_get_token')) {
	$csrf_token = (string)csrf_get_token();
}

// Display tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Active IPs"), false, "/zid-proxy_active_ips.php");
$tab_array[] = array(gettext("Agent"), false, "/zid-proxy_agent.php");
$tab_array[] = array(gettext("Groups"), true, "/zid-proxy_groups.php");
$tab_array[] = array(gettext("Access Rules"), false, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("AppID"), false, "/zid-proxy_appid.php");
$tab_array[] = array(gettext("Logs"), false, "/zid-proxy_log.php");
display_top_tabs($tab_array);

if ($input_errors) {
	print_input_errors($input_errors);
}

// Group edit form
$form = new Form();
$form->setAction('zid-proxy_groups.php');
$section = new Form_Section('Add/Edit Group');

$section->addInput(new Form_Input(
	'id',
	null,
	'hidden',
	$pconfig['id'] ?? ''
));

$section->addInput(new Form_Input(
	'name',
	'*Group Name',
	'text',
	$pconfig['name'] ?? ''
))->setHelp('Unique name (e.g., acesso_liberado). Order matters: first matching group wins.');

$section->addInput(new Form_Input(
	'descr',
	'Description',
	'text',
	$pconfig['descr'] ?? ''
));

$section->addInput(new Form_Textarea(
	'members',
	'*Members (one per line)',
	$pconfig['members_text'] ?? ''
))->setHelp("IPs/CIDRs that belong to this group (e.g., 192.168.1.0/24).");

$section->addInput(new Form_Textarea(
	'rules',
	'Rules (one per line)',
	$pconfig['rules_text'] ?? ''
))->setHelp("Format: TYPE;HOSTNAME (e.g., BLOCK;*.facebook.com). Optional inline comment: # description");

$form->add($section);
print($form);

?>

<div class="panel panel-default">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Current Groups (Order = Priority)')?></h2></div>
	<div class="panel-body">
		<div class="table-responsive">
			<table class="table table-striped table-hover table-condensed">
				<thead>
					<tr>
						<th style="width: 50px;"><?=gettext('#')?></th>
						<th><?=gettext('Name')?></th>
						<th><?=gettext('Description')?></th>
						<th style="width: 120px;"><?=gettext('Members')?></th>
						<th style="width: 120px;"><?=gettext('Rules')?></th>
						<th style="width: 140px;"><?=gettext('Order')?></th>
						<th style="width: 80px;"><?=gettext('Actions')?></th>
					</tr>
				</thead>
				<tbody>
<?php
$i = 0;
foreach ($a_groups as $g):
	$member_count = count(zidproxy_groups_textarea_lines($g['members'] ?? ''));
	$rule_count = count(zidproxy_groups_textarea_lines($g['rules'] ?? ''));
?>
					<tr>
						<td><?=htmlspecialchars($i + 1)?></td>
						<td><strong><?=htmlspecialchars($g['name'] ?? '')?></strong></td>
						<td><?=htmlspecialchars($g['descr'] ?? '')?></td>
						<td><?=$member_count?></td>
						<td><?=$rule_count?></td>
						<td>
							<a class="fa fa-arrow-up" title="<?=gettext('Move Up')?>" href="#" onclick="moveGroup('moveup', <?=$i?>); return false;"></a>
							&nbsp;
							<a class="fa fa-arrow-down" title="<?=gettext('Move Down')?>" href="#" onclick="moveGroup('movedown', <?=$i?>); return false;"></a>
						</td>
						<td>
							<a class="fa fa-pencil" title="<?=gettext('Edit')?>" href="?act=edit&amp;id=<?=$i?>"></a>
							<a class="fa fa-trash text-danger" title="<?=gettext('Delete')?>" href="#" onclick="deleteGroup(<?=$i?>); return false;"></a>
						</td>
					</tr>
<?php
	$i++;
endforeach;

if (empty($a_groups)):
?>
					<tr>
						<td colspan="7" class="text-center">
							<?=gettext('No groups configured. Add a group above.')?>
						</td>
					</tr>
<?php endif; ?>
				</tbody>
			</table>
		</div>
	</div>
</div>

<div class="panel panel-info">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('How It Works')?></h2></div>
	<div class="panel-body">
		<ul>
			<li><?=gettext('Groups are evaluated top-to-bottom. The first group that contains the source IP is selected.')?></li>
			<li><?=gettext('Only the selected group rules apply. Within the group, ALLOW has priority over BLOCK.')?></li>
			<li><?=gettext('If no rule matches, the connection is ALLOWED (default).')?></li>
		</ul>
	</div>
</div>

<script type="text/javascript">
//<![CDATA[
function deleteGroup(id) {
	if (confirm('<?=gettext("Are you sure you want to delete this group?")?>')) {
		var form = document.createElement('form');
		form.method = 'POST';
		form.action = 'zid-proxy_groups.php';

		// CSRF token
		var csrfInput = document.createElement('input');
		csrfInput.type = 'hidden';
		csrfInput.name = '<?=htmlspecialchars($csrf_token_key)?>';
		csrfInput.value = '<?=htmlspecialchars($csrf_token)?>';
		form.appendChild(csrfInput);

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

function moveGroup(act, id) {
	var form = document.createElement('form');
	form.method = 'POST';
	form.action = 'zid-proxy_groups.php';

	// CSRF token
	var csrfInput = document.createElement('input');
	csrfInput.type = 'hidden';
	csrfInput.name = '<?=htmlspecialchars($csrf_token_key)?>';
	csrfInput.value = '<?=htmlspecialchars($csrf_token)?>';
	form.appendChild(csrfInput);

	var actInput = document.createElement('input');
	actInput.type = 'hidden';
	actInput.name = 'act';
	actInput.value = act;
	form.appendChild(actInput);

	var idInput = document.createElement('input');
	idInput.type = 'hidden';
	idInput.name = 'id';
	idInput.value = id;
	form.appendChild(idInput);

	document.body.appendChild(form);
	form.submit();
}
//]]>
</script>

<?php include("foot.inc"); ?>
