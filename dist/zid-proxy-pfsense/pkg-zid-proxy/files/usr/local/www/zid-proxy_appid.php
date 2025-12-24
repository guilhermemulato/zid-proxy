<?php
/*
 * zid-proxy_appid.php
 *
 * Application ID (AppID) management page for ZID Proxy
 *
 * Allows blocking/allowing specific applications (Netflix, YouTube, etc.) per group.
 *
 * Format generated to /usr/local/etc/zid-proxy/appid_rules.txt:
 *   BLOCK_APP;group_name;app_name
 *   ALLOW_APP;group_name;app_name
 */

require_once("guiconfig.inc");
require_once("/usr/local/pkg/zid-proxy.inc");

$pgtitle = array(gettext("Services"), gettext("ZID Proxy"), gettext("AppID"));
$pglinks = array("", "/pkg.php?xml=zid-proxy.xml", "@self");
$shortcut_section = "zidproxy";

// Define available applications by category
$app_categories = array(
	'streaming_media' => array(
		'label' => gettext('Streaming Media'),
		'apps' => array(
			'netflix' => 'Netflix',
			'youtube' => 'YouTube',
			'spotify' => 'Spotify',
			'twitch' => 'Twitch',
			'disney_plus' => 'Disney+',
			'amazon_video' => 'Amazon Prime Video',
			'hbo_max' => 'HBO Max',
			'apple_tv' => 'Apple TV+',
			'deezer' => 'Deezer',
			'soundcloud' => 'SoundCloud',
			'tidal' => 'Tidal',
			'vimeo' => 'Vimeo',
			'dailymotion' => 'Dailymotion',
		)
	),
	'social_networking' => array(
		'label' => gettext('Social Networking'),
		'apps' => array(
			'facebook' => 'Facebook',
			'instagram' => 'Instagram',
			'twitter' => 'Twitter/X',
			'tiktok' => 'TikTok',
			'linkedin' => 'LinkedIn',
			'pinterest' => 'Pinterest',
			'reddit' => 'Reddit',
			'snapchat' => 'Snapchat',
			'tumblr' => 'Tumblr',
		)
	),
	'messaging' => array(
		'label' => gettext('Messaging'),
		'apps' => array(
			'whatsapp' => 'WhatsApp',
			'telegram' => 'Telegram',
			'discord' => 'Discord',
			'slack' => 'Slack',
			'microsoft_teams' => 'Microsoft Teams',
			'zoom' => 'Zoom',
			'skype' => 'Skype',
			'signal' => 'Signal',
			'viber' => 'Viber',
			'line' => 'LINE',
		)
	),
	'games' => array(
		'label' => gettext('Games'),
		'apps' => array(
			'steam' => 'Steam',
			'epic_games' => 'Epic Games',
			'playstation' => 'PlayStation Network',
			'xbox' => 'Xbox Live',
			'nintendo' => 'Nintendo Online',
			'riot_games' => 'Riot Games',
			'blizzard' => 'Blizzard',
			'ea' => 'EA Games',
			'ubisoft' => 'Ubisoft',
			'roblox' => 'Roblox',
		)
	),
	'vpn_tunneling' => array(
		'label' => gettext('VPN/Tunneling'),
		'apps' => array(
			'openvpn' => 'OpenVPN',
			'nordvpn' => 'NordVPN',
			'expressvpn' => 'ExpressVPN',
			'surfshark' => 'Surfshark',
			'protonvpn' => 'ProtonVPN',
		)
	),
	'file_transfer' => array(
		'label' => gettext('File Transfer'),
		'apps' => array(
			'dropbox' => 'Dropbox',
			'google_drive' => 'Google Drive',
			'onedrive' => 'OneDrive',
			'icloud' => 'iCloud',
			'wetransfer' => 'WeTransfer',
			'mega' => 'MEGA',
			'mediafire' => 'MediaFire',
		)
	),
	'business' => array(
		'label' => gettext('Business'),
		'apps' => array(
			'office365' => 'Microsoft 365',
			'google_workspace' => 'Google Workspace',
			'salesforce' => 'Salesforce',
			'hubspot' => 'HubSpot',
			'zendesk' => 'Zendesk',
			'atlassian' => 'Atlassian',
			'asana' => 'Asana',
			'notion' => 'Notion',
		)
	),
);

// Load groups from zid-proxy
$groups = zidproxy_get_groups();
if (!is_array($groups)) {
	$groups = [];
}

// AppID rules file
define('APPID_RULES_FILE', '/usr/local/etc/zid-proxy/appid_rules.txt');

// Load current AppID rules
function load_appid_rules() {
	$rules = array();
	if (!file_exists(APPID_RULES_FILE)) {
		return $rules;
	}

	$content = file_get_contents(APPID_RULES_FILE);
	$lines = explode("\n", $content);

	foreach ($lines as $line) {
		$line = trim($line);
		if ($line === '' || strpos($line, '#') === 0) {
			continue;
		}

		// Remove inline comments
		$hash = strpos($line, '#');
		if ($hash !== false) {
			$line = trim(substr($line, 0, $hash));
		}

		$parts = explode(';', $line);
		if (count($parts) !== 3) {
			continue;
		}

		$type = strtoupper(trim($parts[0]));
		$group = trim($parts[1]);
		$app = strtolower(trim($parts[2]));

		if (!in_array($type, ['BLOCK_APP', 'ALLOW_APP'])) {
			continue;
		}

		$key = $group . ':' . $app;
		$rules[$key] = $type;
	}

	return $rules;
}

// Save AppID rules
function save_appid_rules($rules) {
	$content = "# ZID Proxy AppID Rules\n";
	$content .= "# Format: TYPE;GROUP;APP_NAME\n";
	$content .= "# Generated: " . date('Y-m-d H:i:s') . "\n\n";

	foreach ($rules as $key => $type) {
		list($group, $app) = explode(':', $key, 2);
		$content .= "{$type};{$group};{$app}\n";
	}

	// Ensure directory exists
	$dir = dirname(APPID_RULES_FILE);
	if (!is_dir($dir)) {
		mkdir($dir, 0755, true);
	}

	file_put_contents(APPID_RULES_FILE, $content);

	// Reload zid-appid if running
	if (is_service_running('zid-appid')) {
		exec('/bin/pkill -HUP -f zid-appid');
	}
}

// Handle form submission
$input_errors = array();
$savemsg = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_POST['save'])) {
	$new_rules = array();

	foreach ($groups as $g) {
		$group_name = $g['name'];

		foreach ($app_categories as $cat_id => $category) {
			foreach ($category['apps'] as $app_id => $app_label) {
				$field_name = "app_{$group_name}_{$app_id}";

				if (isset($_POST[$field_name])) {
					$value = $_POST[$field_name];
					if ($value === 'block') {
						$new_rules["{$group_name}:{$app_id}"] = 'BLOCK_APP';
					} elseif ($value === 'allow') {
						$new_rules["{$group_name}:{$app_id}"] = 'ALLOW_APP';
					}
					// 'default' means no rule - don't add to list
				}
			}
		}
	}

	save_appid_rules($new_rules);
	$savemsg = gettext("AppID rules saved successfully.");
}

// Load current rules
$current_rules = load_appid_rules();

// Check zid-appid daemon status
$appid_running = is_service_running('zid-appid');

include("head.inc");

// Display tabs
$tab_array = array();
$tab_array[] = array(gettext("Settings"), false, "/zid-proxy_settings.php");
$tab_array[] = array(gettext("Active IPs"), false, "/zid-proxy_active_ips.php");
$tab_array[] = array(gettext("Agent"), false, "/zid-proxy_agent.php");
$tab_array[] = array(gettext("Groups"), false, "/zid-proxy_groups.php");
$tab_array[] = array(gettext("Access Rules"), false, "/zid-proxy_rules.php");
$tab_array[] = array(gettext("AppID"), true, "/zid-proxy_appid.php");
$tab_array[] = array(gettext("Logs"), false, "/zid-proxy_log.php");
display_top_tabs($tab_array);

if ($input_errors) {
	print_input_errors($input_errors);
}

if ($savemsg) {
	print_info_box($savemsg, 'success');
}

// Show warning if no groups defined
if (empty($groups)) {
	print_info_box(gettext("No groups are defined. Please create groups in the Groups tab first to configure AppID rules."), 'warning');
}

// Show AppID daemon status
$status_class = $appid_running ? 'success' : 'warning';
$status_text = $appid_running ? gettext('Running') : gettext('Not Running');
print_info_box(
	sprintf(gettext("AppID Daemon Status: <strong>%s</strong>"), $status_text) .
	($appid_running ? '' : ' - ' . gettext('AppID features require the zid-appid daemon to be running.')),
	$status_class
);

?>

<form action="zid-proxy_appid.php" method="post" name="iform" id="iform">
	<input type="hidden" name="save" value="1">

<?php if (!empty($groups)): ?>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h2 class="panel-title"><?=gettext('Application Rules by Group')?></h2>
		</div>
		<div class="panel-body">
			<p><?=gettext('Configure which applications to block or allow for each group. Leave as "Default" to apply normal SNI-based rules.')?></p>

			<?php foreach ($groups as $g): ?>
				<?php $group_name = $g['name']; ?>

				<div class="panel panel-info">
					<div class="panel-heading">
						<h3 class="panel-title">
							<i class="fa fa-users"></i>
							<?=htmlspecialchars($g['name'])?>
							<?php if (!empty($g['descr'])): ?>
								<small> - <?=htmlspecialchars($g['descr'])?></small>
							<?php endif; ?>
						</h3>
					</div>
					<div class="panel-body">
						<div class="table-responsive">
							<table class="table table-striped table-condensed">
								<thead>
									<tr>
										<th style="width: 200px;"><?=gettext('Application')?></th>
										<th style="width: 150px;"><?=gettext('Action')?></th>
									</tr>
								</thead>
								<tbody>
								<?php foreach ($app_categories as $cat_id => $category): ?>
									<tr class="info">
										<td colspan="2"><strong><?=$category['label']?></strong></td>
									</tr>
									<?php foreach ($category['apps'] as $app_id => $app_label): ?>
										<?php
										$field_name = "app_{$group_name}_{$app_id}";
										$rule_key = "{$group_name}:{$app_id}";
										$current_value = 'default';
										if (isset($current_rules[$rule_key])) {
											$current_value = ($current_rules[$rule_key] === 'BLOCK_APP') ? 'block' : 'allow';
										}
										?>
										<tr>
											<td style="padding-left: 30px;"><?=$app_label?></td>
											<td>
												<select name="<?=$field_name?>" class="form-control input-sm" style="width: 120px;">
													<option value="default" <?=($current_value === 'default') ? 'selected' : ''?>><?=gettext('Default')?></option>
													<option value="block" <?=($current_value === 'block') ? 'selected' : ''?>><?=gettext('Block')?></option>
													<option value="allow" <?=($current_value === 'allow') ? 'selected' : ''?>><?=gettext('Allow')?></option>
												</select>
											</td>
										</tr>
									<?php endforeach; ?>
								<?php endforeach; ?>
								</tbody>
							</table>
						</div>
					</div>
				</div>
			<?php endforeach; ?>

			<div class="form-group">
				<button type="submit" class="btn btn-primary">
					<i class="fa fa-save"></i> <?=gettext('Save')?>
				</button>
			</div>
		</div>
	</div>

<?php endif; ?>

</form>

<div class="panel panel-info">
	<div class="panel-heading"><h2 class="panel-title"><?=gettext('How AppID Works')?></h2></div>
	<div class="panel-body">
		<ul>
			<li><strong><?=gettext('Block')?></strong>: <?=gettext('Block this application for the selected group, even if SNI rules would allow it.')?></li>
			<li><strong><?=gettext('Allow')?></strong>: <?=gettext('Allow this application for the selected group, even if a block rule exists.')?></li>
			<li><strong><?=gettext('Default')?></strong>: <?=gettext('No AppID rule - normal SNI-based Access Rules apply.')?></li>
		</ul>
		<p><strong><?=gettext('Priority')?></strong>: <?=gettext('ALLOW_APP has priority over BLOCK_APP. AppID rules are evaluated before SNI rules.')?></p>
		<p><strong><?=gettext('Detection')?></strong>: <?=gettext('Applications are detected using hostname matching (SNI). The zid-appid daemon provides enhanced detection using Deep Packet Inspection.')?></p>
	</div>
</div>

<?php include("foot.inc"); ?>
