<?php
/*
 * zid-proxy_log.php
 *
 * Log viewer page for ZID Proxy with auto-refresh, pause and real-time filtering
 *
 * Licensed under the Apache License, Version 2.0
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("Logs"));
$pglinks = array("", "/pkg.php?xml=zid-proxy.xml", "@self");
$shortcut_section = "zidproxy";

// Auto-refresh interval (default: 20s)
$refresh_interval = isset($_GET['refresh']) ? intval($_GET['refresh']) : 20;
if ($refresh_interval < 0 || $refresh_interval > 300) {
	$refresh_interval = 20;
}

// Filter term
$filter_term = isset($_GET['filter']) ? trim($_GET['filter']) : '';

// Handle clear log action
if ($_POST['clear']) {
	zidproxy_clear_log();
	header("Location: zid-proxy_log.php");
	exit;
}

// Get log entries
$log_entries = zidproxy_get_log_entries(500);

// Apply backend filter if provided (optional optimization)
if (!empty($filter_term)) {
	$log_entries = array_filter($log_entries, function($entry) use ($filter_term) {
		$filter_lower = strtolower($filter_term);
		return (strpos(strtolower($entry['source_ip']), $filter_lower) !== false) ||
		       (strpos(strtolower($entry['hostname']), $filter_lower) !== false) ||
		       (strpos(strtolower($entry['group'] ?? ''), $filter_lower) !== false);
	});
}

include("head.inc");

// Display tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Active IPs"), false, "/zid-proxy_active_ips.php");
$tab_array[] = array(gettext("Groups"), false, "/zid-proxy_groups.php");
$tab_array[] = array(gettext("Access Rules"), false, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("Logs"), true, "/zid-proxy_log.php");
display_top_tabs($tab_array);

?>

<?php if ($refresh_interval > 0): ?>
<?php
	// Build URL with current refresh and filter
	$url_params = array();
	$url_params['refresh'] = $refresh_interval;
	if (!empty($filter_term)) {
		$url_params['filter'] = $filter_term;
	}
	$current_url = '/zid-proxy_log.php?' . http_build_query($url_params);
?>
<meta id="refresh-meta" http-equiv="refresh" content="<?php echo $refresh_interval; ?>;url=<?php echo htmlspecialchars($current_url); ?>">
<?php endif; ?>

<style>
.panel-heading {
    min-height: 60px;
    padding: 15px 20px;
}

.panel-title {
    line-height: 30px;
}

.panel-title .pull-right {
    margin-top: -5px;
}

.panel-title .pull-right .form-control,
.panel-title .pull-right .btn {
    margin-bottom: 5px;
}
</style>

<div class="panel panel-default">
	<div class="panel-heading">
		<h2 class="panel-title">
			<?=gettext('Connection Log')?>
			<span class="pull-right">
				<!-- Seletor de Auto-Refresh -->
				<select id="refreshInterval" class="form-control" style="width: 90px; display: inline-block; padding: 2px 5px; height: 26px; font-size: 12px;">
					<option value="0" <?php echo $refresh_interval == 0 ? 'selected' : ''; ?>><?=gettext('Disabled')?></option>
					<option value="5" <?php echo $refresh_interval == 5 ? 'selected' : ''; ?>>5s</option>
					<option value="10" <?php echo $refresh_interval == 10 ? 'selected' : ''; ?>>10s</option>
					<option value="20" <?php echo $refresh_interval == 20 ? 'selected' : ''; ?>>20s</option>
					<option value="30" <?php echo $refresh_interval == 30 ? 'selected' : ''; ?>>30s</option>
				</select>

				<!-- Checkbox Pause -->
				<label style="margin-left: 10px; font-weight: normal; cursor: pointer; font-size: 12px;">
					<input type="checkbox" id="pauseRefresh" style="margin-right: 3px;">
					<?=gettext('Pause')?>
				</label>

				<!-- Campo de Filtro -->
				<input type="text" id="filterInput" placeholder="<?=gettext('Filter by IP or Domain')?>"
					   value="<?=htmlspecialchars($filter_term)?>"
					   class="form-control" style="width: 180px; display: inline-block; padding: 2px 5px; height: 26px; margin-left: 10px; font-size: 12px;">

				<!-- Botão Refresh Manual -->
				<button type="button" class="btn btn-xs btn-primary" onclick="location.reload();" style="margin-left: 5px;">
					<i class="fa fa-refresh"></i> <?=gettext('Refresh')?>
				</button>

				<!-- Botão Clear Log -->
				<form method="post" style="display: inline; margin-left: 5px;">
					<button type="submit" name="clear" class="btn btn-xs btn-danger"
							onclick="return confirm('<?=gettext("Are you sure you want to clear the log?")?>');">
						<i class="fa fa-trash"></i> <?=gettext('Clear')?>
					</button>
				</form>
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
						<th style="width: 160px;"><?=gettext('Group')?></th>
						<th style="width: 80px;"><?=gettext('Action')?></th>
					</tr>
				</thead>
				<tbody>
<?php
if (!empty($log_entries)):
	foreach ($log_entries as $entry):
		$action_class = ($entry['action'] == 'ALLOW') ? 'success' : 'danger';

		// Convert timestamp from UTC to America/Sao_Paulo
		$timestamp = $entry['timestamp'];
		try {
			$dt = new DateTime($timestamp, new DateTimeZone('UTC'));
			$dt->setTimezone(new DateTimeZone('America/Sao_Paulo'));
			$timestamp_local = $dt->format('Y-m-d H:i:s');
		} catch (Exception $e) {
			$timestamp_local = $timestamp; // Fallback to original if conversion fails
		}
?>
					<tr>
						<td><small><?=htmlspecialchars($timestamp_local)?></small></td>
						<td><?=htmlspecialchars($entry['source_ip'])?></td>
						<td><?=htmlspecialchars($entry['hostname'])?></td>
						<td>
							<?php if (!empty($entry['group'])): ?>
								<span class="label label-info"><?=htmlspecialchars($entry['group'])?></span>
							<?php endif; ?>
						</td>
						<td>
							<span class="label label-<?=$action_class?>"><?=htmlspecialchars($entry['action'])?></span>
						</td>
					</tr>
<?php
	endforeach;
else:
?>
					<tr>
						<td colspan="5" class="text-center">
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

<script type="text/javascript">
//<![CDATA[
(function() {
	'use strict';

	// ========================================
	// AUTO-REFRESH CONTROL
	// ========================================

	var refreshSelect = document.getElementById('refreshInterval');
	var pauseCheckbox = document.getElementById('pauseRefresh');
	var filterInput = document.getElementById('filterInput');
	var refreshMeta = document.getElementById('refresh-meta');

	// Função para atualizar URL sem reload
	function updateURL() {
		var url = new URL(window.location);
		var refresh = refreshSelect.value;
		var filter = filterInput.value.trim();

		if (refresh > 0) {
			url.searchParams.set('refresh', refresh);
		} else {
			url.searchParams.delete('refresh');
		}

		if (filter !== '') {
			url.searchParams.set('filter', filter);
		} else {
			url.searchParams.delete('filter');
		}

		window.history.replaceState({}, '', url);
	}

	// Função para adicionar/remover meta tag de refresh
	function setMetaRefresh(interval) {
		if (refreshMeta) {
			refreshMeta.remove();
			refreshMeta = null;
		}

		if (interval > 0) {
			// Build URL with current refresh and filter values from DOM
			var url = new URL(window.location);
			url.searchParams.set('refresh', interval);

			var filterValue = filterInput.value.trim();
			if (filterValue !== '') {
				url.searchParams.set('filter', filterValue);
			} else {
				url.searchParams.delete('filter');
			}

			refreshMeta = document.createElement('meta');
			refreshMeta.id = 'refresh-meta';
			refreshMeta.httpEquiv = 'refresh';
			refreshMeta.content = interval + ';url=' + url.toString();
			document.head.appendChild(refreshMeta);
		}
	}

	// Event: Mudança no seletor de intervalo
	refreshSelect.addEventListener('change', function() {
		var interval = parseInt(this.value);

		// Se selecionou intervalo > 0, desmarcar pause
		if (interval > 0) {
			pauseCheckbox.checked = false;
			localStorage.setItem('zidproxy_log_paused', 'false');
		}

		setMetaRefresh(interval);
		updateURL();
	});

	// Event: Pause checkbox
	pauseCheckbox.addEventListener('change', function() {
		var isPaused = this.checked;
		localStorage.setItem('zidproxy_log_paused', isPaused ? 'true' : 'false');

		if (isPaused) {
			// Pausar: remover meta tag
			setMetaRefresh(0);
		} else {
			// Resumir: restaurar intervalo atual
			var interval = parseInt(refreshSelect.value);
			if (interval > 0) {
				setMetaRefresh(interval);
			}
		}
	});

	// Restaurar estado de pause do localStorage
	var savedPaused = localStorage.getItem('zidproxy_log_paused');
	if (savedPaused === 'true') {
		pauseCheckbox.checked = true;
		setMetaRefresh(0);
	}

	// ========================================
	// FILTRO EM TEMPO REAL
	// ========================================

	var tableBody = document.querySelector('table tbody');
	var allRows = tableBody ? tableBody.querySelectorAll('tr') : [];

	function filterTable(searchTerm) {
		if (!tableBody || allRows.length === 0) return;

		searchTerm = searchTerm.toLowerCase().trim();

		var visibleCount = 0;

		allRows.forEach(function(row) {
			// Pular linha de "no entries" (colspan)
			if (row.cells.length < 5) {
				return;
			}

			var sourceIp = row.cells[1].textContent.toLowerCase();
			var hostname = row.cells[2].textContent.toLowerCase();
			var group = row.cells[3].textContent.toLowerCase();

			// Mostrar se match em IP, hostname ou group, ou se filtro vazio
			if (searchTerm === '' ||
				sourceIp.indexOf(searchTerm) !== -1 ||
				hostname.indexOf(searchTerm) !== -1 ||
				group.indexOf(searchTerm) !== -1) {
				row.style.display = '';
				visibleCount++;
			} else {
				row.style.display = 'none';
			}
		});

		// Se nenhum resultado, mostrar mensagem
		if (visibleCount === 0 && searchTerm !== '') {
			// Procurar por linha de "no results" ou criar uma
			var noResultRow = tableBody.querySelector('tr.no-results');
			if (!noResultRow && allRows.length > 0) {
				noResultRow = document.createElement('tr');
				noResultRow.className = 'no-results';
				noResultRow.innerHTML = '<td colspan="5" class="text-center" style="padding: 20px;">' +
					'<?=gettext("No log entries match the filter.")?>' +
					'</td>';
				tableBody.appendChild(noResultRow);
			}
			if (noResultRow) noResultRow.style.display = '';
		} else {
			var noResultRow = tableBody.querySelector('tr.no-results');
			if (noResultRow) noResultRow.style.display = 'none';
		}
	}

	// Event: Filtro em tempo real (keyup)
	filterInput.addEventListener('keyup', function() {
		filterTable(this.value);
		updateURL();

		// CRÍTICO: Atualizar meta tag para incluir novo filtro
		var currentInterval = parseInt(refreshSelect.value);
		if (currentInterval > 0 && !pauseCheckbox.checked) {
			setMetaRefresh(currentInterval);
		}
	});

	// Aplicar filtro inicial (se veio da URL)
	if (filterInput.value !== '') {
		filterTable(filterInput.value);
	}

})();
//]]>
</script>

<?php include("foot.inc"); ?>
