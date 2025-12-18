#!/usr/local/bin/php
<?php
/*
 * register-package.php
 *
 * Registers the ZID Proxy package in pfSense's config.xml.
 * This makes the package visible in the web interface AND enables auto-start on boot.
 *
 * CRITICAL: This script adds BOTH the <package> and <menu> tags to config.xml.
 * Without the <menu> tag, the menu won't appear AND the service won't auto-start after reboot.
 *
 * Usage: php register-package.php
 *
 * Licensed under the Apache License, Version 2.0
 */

echo "=========================================\n";
echo " ZID Proxy Package Registration v1.0.5\n";
echo "=========================================\n\n";

// Check if running as root
if (posix_geteuid() !== 0) {
    echo "Error: This script must be run as root\n";
    exit(1);
}

// Check if this is actually pfSense
if (!file_exists('/etc/inc/config.inc')) {
    echo "Error: This does not appear to be a pfSense system\n";
    exit(1);
}

echo "Loading pfSense configuration system...\n";
require_once('/etc/inc/config.inc');
require_once('/etc/inc/util.inc');

// Parse current configuration
echo "Parsing configuration...\n";
$config = parse_config(true);

// Initialize arrays if they don't exist
if (!is_array($config['installedpackages'])) {
    $config['installedpackages'] = array();
}
if (!is_array($config['installedpackages']['package'])) {
    $config['installedpackages']['package'] = array();
}
if (!is_array($config['installedpackages']['menu'])) {
    $config['installedpackages']['menu'] = array();
}
if (!is_array($config['installedpackages']['zidproxy'])) {
    $config['installedpackages']['zidproxy'] = array();
}
if (!is_array($config['installedpackages']['zidproxy']['config'])) {
    $config['installedpackages']['zidproxy']['config'] = array();
}

// Remove old package entries to avoid duplicates
echo "Removing old package entries (if any)...\n";
foreach ($config['installedpackages']['package'] as $idx => $pkg) {
    if (isset($pkg['name']) && $pkg['name'] == 'zid-proxy') {
        unset($config['installedpackages']['package'][$idx]);
        echo "  - Removed old package entry\n";
    }
}
// Reindex array to avoid gaps
$config['installedpackages']['package'] = array_values($config['installedpackages']['package']);

// Add package entry with CORRECT tag names
echo "Adding package entry...\n";
$config['installedpackages']['package'][] = array(
    'name' => 'zid-proxy',
    'version' => '1.0.10.8.1',
    'descr' => 'ZID Proxy - SNI-based transparent HTTPS filtering proxy',
    'website' => '',
    'configurationfile' => 'zid-proxy.xml',  // Correct tag (not config_file), no path
    'include_file' => '/usr/local/pkg/zid-proxy.inc'
);
echo "  ✓ Package entry added\n";

// Remove old menu entries to avoid duplicates
echo "Removing old menu entries (if any)...\n";
foreach ($config['installedpackages']['menu'] as $idx => $menu) {
    if (isset($menu['name']) && $menu['name'] === 'ZID Proxy') {
        unset($config['installedpackages']['menu'][$idx]);
        echo "  - Removed old menu entry\n";
    }
}
// Reindex array to avoid gaps
$config['installedpackages']['menu'] = array_values($config['installedpackages']['menu']);

// Add menu entry - THIS IS CRITICAL FOR BOTH MENU AND AUTO-START!
echo "Adding menu entry to config.xml...\n";
$config['installedpackages']['menu'][] = array(
    'name' => 'ZID Proxy',
    'tooltiptext' => 'Configure SNI-based transparent HTTPS proxy',
    'section' => 'Services',
    'url' => '/zid-proxy_settings.php'
);
echo "  ✓ Menu entry added (this enables menu display AND boot auto-start)\n";

// Initialize default configuration if empty
if (empty($config['installedpackages']['zidproxy']['config'])) {
    echo "Creating default configuration...\n";
    $config['installedpackages']['zidproxy']['config'][0] = array(
        'enable' => 'off',
        'interface' => 'all',  // Changed from 'lan' to 'all' for better NAT compatibility
        'listen_port' => '3129',
        'timeout' => '30',
        'enable_logging' => 'on',
        'rules_mode' => 'legacy',
        'log_retention_days' => '7'
    );
    echo "  ✓ Default config created (interface: all, port: 3129)\n";
}

// Write configuration
echo "Writing configuration to /cf/conf/config.xml...\n";
write_config("ZID Proxy package registered");
echo "  ✓ Configuration saved\n";

// Load package functions and execute install hook
if (file_exists('/usr/local/pkg/zid-proxy.inc')) {
    echo "Executing installation hook...\n";
    require_once('/usr/local/pkg/zid-proxy.inc');
    zidproxy_install();
    echo "  ✓ Install hook executed\n";
} else {
    echo "Warning: Package include file not found\n";
}

echo "\n=========================================\n";
echo " Registration Complete!\n";
echo "=========================================\n\n";

echo "✓ Package entry added to config.xml\n";
echo "✓ Menu entry added to config.xml\n";
echo "✓ Default configuration created\n\n";

echo "IMPORTANT: To see the menu in the web interface, reload the web GUI:\n\n";
echo "  /etc/rc.restart_webgui\n\n";
echo "  (Wait ~10 seconds for the GUI to reload)\n\n";

echo "Then reload your browser (Ctrl+Shift+R) and check Services > ZID Proxy\n\n";

echo "The menu entry in config.xml also enables the service to auto-start\n";
echo "after pfSense reboots (when Enable is checked in Settings tab).\n\n";

?>
