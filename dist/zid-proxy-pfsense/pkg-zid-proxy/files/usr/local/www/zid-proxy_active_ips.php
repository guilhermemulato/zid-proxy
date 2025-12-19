<?php
/*
 * zid-proxy_active_ips.php
 *
 * Displays active source IPs aggregated by zid-proxy (no per-connection view).
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Active IPs"));
$shortcut_section = "zidproxy";

@date_default_timezone_set('America/Sao_Paulo');

global $config;
zidproxy_ensure_config_defaults();
$zidcfg = $config['installedpackages']['zidproxy']['config'][0] ?? [];

$refresh_seconds = (int)($zidcfg['active_ips_refresh_seconds'] ?? 5);
if ($refresh_seconds < 1) {
	$refresh_seconds = 1;
}
if ($refresh_seconds > 300) {
	$refresh_seconds = 300;
}

$json_path = ZIDPROXY_ACTIVE_IPS_JSON;
$data = null;
if (file_exists($json_path)) {
	$raw = @file_get_contents($json_path);
	if ($raw !== false && trim($raw) !== '') {
		$data = json_decode($raw, true);
	}
}

function zidproxy_format_bytes($bytes) {
	$bytes = (float)$bytes;
	$units = array('B', 'KB', 'MB', 'GB', 'TB');
	$u = 0;
	while ($bytes >= 1024 && $u < count($units) - 1) {
		$bytes /= 1024;
		$u++;
	}
	return sprintf($u === 0 ? "%.0f %s" : "%.1f %s", $bytes, $units[$u]);
}

function zidproxy_ago($iso8601) {
	$ts = strtotime((string)$iso8601);
	if ($ts <= 0) {
		return gettext("Unknown");
	}
	$delta = time() - $ts;
	if ($delta < 0) {
		$delta = 0;
	}
	if ($delta < 60) {
		return sprintf(gettext("%ds ago"), $delta);
	}
	if ($delta < 3600) {
		return sprintf(gettext("%dm ago"), (int)floor($delta / 60));
	}
	return sprintf(gettext("%dh ago"), (int)floor($delta / 3600));
}

function zidproxy_format_local_time($iso8601) {
	$ts = strtotime((string)$iso8601);
	if ($ts <= 0) {
		return '';
	}
	return date('Y-m-d H:i:s', $ts);
}

include("head.inc");

$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Active IPs"), true, "/zid-proxy_active_ips.php");
$tab_array[] = array(gettext("Agent"), false, "/zid-proxy_agent.php");
$tab_array[] = array(gettext("Groups"), false, "/zid-proxy_groups.php");
$tab_array[] = array(gettext("Access Rules"), false, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("Logs"), false, "/zid-proxy_log.php");
display_top_tabs($tab_array);

?>

<script>
	function zidproxyReloadActiveIPs() {
		window.location.reload();
	}
	setTimeout(zidproxyReloadActiveIPs, <?=htmlspecialchars((string)($refresh_seconds * 1000))?>);
</script>

<div class="panel panel-default">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Active IPs')?></h2></div>
	<div class="panel-body">
		<p>
			<?=gettext('Auto-refresh')?>: <strong><?=htmlspecialchars((string)$refresh_seconds)?>s</strong>
			&nbsp;|&nbsp;
			<?=gettext('Source')?>: <code><?=htmlspecialchars($json_path)?></code>
		</p>
	</div>
</div>

<?php if (!is_array($data) || !is_array($data['ips'] ?? null)): ?>
	<?php print_info_box(gettext("No active IP data yet. Ensure the service is running and generating traffic."), 'info'); ?>
<?php else: ?>
	<div class="panel panel-default">
		<div class="panel-heading"><h2 class="panel-title"><?=gettext('IPs')?></h2></div>
		<div class="panel-body">
			<div class="table-responsive">
				<table class="table table-striped table-hover">
					<thead>
					<tr>
						<th><?=gettext('IP')?></th>
						<th><?=gettext('Machine')?></th>
						<th><?=gettext('User')?></th>
						<th><?=gettext('Last Activity')?></th>
						<th><?=gettext('Idle')?></th>
						<th><?=gettext('Bytes Total')?></th>
						<th><?=gettext('Bytes In')?></th>
						<th><?=gettext('Bytes Out')?></th>
						<th><?=gettext('Active Conns')?></th>
					</tr>
					</thead>
					<tbody>
					<?php foreach (($data['ips'] ?? []) as $row): ?>
						<tr>
							<td><code><?=htmlspecialchars((string)($row['src_ip'] ?? ''))?></code></td>
							<td><?=htmlspecialchars((string)($row['machine'] ?? ''))?></td>
							<td><?=htmlspecialchars((string)($row['username'] ?? ''))?></td>
							<td><?=htmlspecialchars(zidproxy_format_local_time($row['last_activity'] ?? ''))?></td>
							<td>
								<?php if (isset($row['idle_seconds'])): ?>
									<?=htmlspecialchars(sprintf("%ds", (int)$row['idle_seconds']))?>
								<?php else: ?>
									<?=htmlspecialchars(zidproxy_ago($row['last_activity'] ?? ''))?>
								<?php endif; ?>
							</td>
							<td><?=htmlspecialchars(zidproxy_format_bytes((int)($row['bytes_total'] ?? 0)))?></td>
							<td><?=htmlspecialchars(zidproxy_format_bytes((int)($row['bytes_in'] ?? 0)))?></td>
							<td><?=htmlspecialchars(zidproxy_format_bytes((int)($row['bytes_out'] ?? 0)))?></td>
							<td><?=htmlspecialchars((string)($row['active_conns'] ?? 0))?></td>
						</tr>
					<?php endforeach; ?>
					</tbody>
				</table>
			</div>
		</div>
	</div>
<?php endif; ?>

<?php include("foot.inc"); ?>
