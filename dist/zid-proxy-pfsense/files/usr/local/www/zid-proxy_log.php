<?php
/*
 * zid-proxy_log.php
 *
 * Log viewer page for ZID Proxy
 *
 * Licensed under the Apache License, Version 2.0
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Logs"));
$pglinks = array("", "/pkg.php?xml=zid-proxy.xml", "@self");
$shortcut_section = "zidproxy";

// Handle clear log action
if ($_POST['clear']) {
	zidproxy_clear_log();
	header("Location: zid-proxy_log.php");
	exit;
}

// Get log entries
$log_entries = zidproxy_get_log_entries(500);

include("head.inc");

// Display tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/pkg.php?xml=zid-proxy.xml");
$tab_array[] = array(gettext("Access Rules"), false, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("Logs"), true, "/zid-proxy_log.php");
display_top_tabs($tab_array);

?>

<div class="panel panel-default">
	<div class="panel-heading">
		<h2 class="panel-title">
			<?=gettext('Connection Log')?>
			<span class="pull-right">
				<form method="post" style="display: inline;">
					<button type="submit" name="clear" class="btn btn-xs btn-danger" onclick="return confirm('<?=gettext("Are you sure you want to clear the log?")?>');">
						<i class="fa fa-trash"></i> <?=gettext('Clear Log')?>
					</button>
				</form>
				<button type="button" class="btn btn-xs btn-primary" onclick="location.reload();">
					<i class="fa fa-refresh"></i> <?=gettext('Refresh')?>
				</button>
			</span>
		</h2>
	</div>
	<div class="panel-body">
		<div class="table-responsive">
			<table class="table table-striped table-hover table-condensed">
				<thead>
					<tr>
						<th style="width: 180px;"><?=gettext('Timestamp')?></th>
						<th style="width: 140px;"><?=gettext('Source IP')?></th>
						<th><?=gettext('Hostname')?></th>
						<th style="width: 80px;"><?=gettext('Action')?></th>
					</tr>
				</thead>
				<tbody>
<?php
if (!empty($log_entries)):
	foreach ($log_entries as $entry):
		$action_class = ($entry['action'] == 'ALLOW') ? 'success' : 'danger';
?>
					<tr>
						<td><small><?=htmlspecialchars($entry['timestamp'])?></small></td>
						<td><?=htmlspecialchars($entry['source_ip'])?></td>
						<td><?=htmlspecialchars($entry['hostname'])?></td>
						<td>
							<span class="label label-<?=$action_class?>"><?=htmlspecialchars($entry['action'])?></span>
						</td>
					</tr>
<?php
	endforeach;
else:
?>
					<tr>
						<td colspan="4" class="text-center">
							<?=gettext('No log entries found.')?>
						</td>
					</tr>
<?php endif; ?>
				</tbody>
			</table>
		</div>
	</div>
	<div class="panel-footer">
		<small>
			<?=gettext('Log file location:')?>
			<code><?=ZIDPROXY_LOG_FILE?></code>
			&nbsp;|&nbsp;
			<?=gettext('Showing last 500 entries (newest first)')?>
		</small>
	</div>
</div>

<div class="panel panel-info">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('Service Status')?></h2></div>
	<div class="panel-body">
		<?php if (zidproxy_status()): ?>
			<span class="label label-success"><?=gettext('Running')?></span>
			<?php
			if (file_exists(ZIDPROXY_PID_FILE)) {
				$pid = trim(file_get_contents(ZIDPROXY_PID_FILE));
				echo " (PID: {$pid})";
			}
			?>
		<?php else: ?>
			<span class="label label-danger"><?=gettext('Stopped')?></span>
		<?php endif; ?>
	</div>
</div>

<?php include("foot.inc"); ?>
