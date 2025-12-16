#!/usr/local/bin/php
<?php
/*
 * register-package.php
 *
 * Registers the ZID Proxy package in pfSense's config.xml.
 * This makes the package visible in the web interface.
 *
 * Usage: php register-package.php
 *
 * Licensed under the Apache License, Version 2.0
 */

echo "=========================================\n";
echo " ZID Proxy Package Registration\n";
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
if (!is_array($config['installedpackages']['zidproxy'])) {
    $config['installedpackages']['zidproxy'] = array();
}
if (!is_array($config['installedpackages']['zidproxy']['config'])) {
    $config['installedpackages']['zidproxy']['config'] = array();
}

// Check if package is already registered
$found = false;
foreach ($config['installedpackages']['package'] as $idx => $pkg) {
    if (isset($pkg['name']) && $pkg['name'] == 'zid-proxy') {
        $found = true;
        echo "Package already registered (updating entry)...\n";
        $config['installedpackages']['package'][$idx] = array(
            'name' => 'zid-proxy',
            'version' => '1.0.0',
            'descr' => 'ZID Proxy - SNI-based transparent HTTPS filtering proxy',
            'website' => '',
            'pkg_includes' => '/usr/local/pkg/zid-proxy.inc',
            'config_file' => '/usr/local/pkg/zid-proxy.xml'
        );
        break;
    }
}

if (!$found) {
    echo "Registering new package...\n";
    $config['installedpackages']['package'][] = array(
        'name' => 'zid-proxy',
        'version' => '1.0.0',
        'descr' => 'ZID Proxy - SNI-based transparent HTTPS filtering proxy',
        'website' => '',
        'pkg_includes' => '/usr/local/pkg/zid-proxy.inc',
        'config_file' => '/usr/local/pkg/zid-proxy.xml'
    );
}

// Initialize default configuration if empty
if (empty($config['installedpackages']['zidproxy']['config'])) {
    echo "Creating default configuration...\n";
    $config['installedpackages']['zidproxy']['config'][0] = array(
        'enable' => 'off',
        'interface' => 'lan',
        'listen_port' => '3129',
        'timeout' => '30',
        'enable_logging' => 'on'
    );
}

// Write configuration
echo "Writing configuration to config.xml...\n";
write_config("ZID Proxy package registered");

// Load package functions and execute install hook
if (file_exists('/usr/local/pkg/zid-proxy.inc')) {
    echo "Executing installation hook...\n";
    require_once('/usr/local/pkg/zid-proxy.inc');
    zidproxy_install();
} else {
    echo "Warning: Package include file not found\n";
}

echo "\n=========================================\n";
echo " Registration Complete!\n";
echo "=========================================\n\n";

echo "The package has been registered in pfSense.\n\n";

echo "To make the 'Services > ZID Proxy' menu appear, you need to reload\n";
echo "the web interface. Choose one of these options:\n\n";

echo "Option 1 - Reload webConfigurator:\n";
echo "  /usr/local/sbin/pfSsh.php playback reloadwebgui\n\n";

echo "Option 2 - Restart pfSense (safest):\n";
echo "  shutdown -r now\n\n";

echo "Option 3 - Restart PHP-FPM:\n";
echo "  /usr/local/etc/rc.d/php-fpm restart\n\n";

echo "After reloading, access Services > ZID Proxy in the web interface.\n\n";

?>
