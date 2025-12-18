#!/usr/local/bin/php
<?php
/*
 * activate-package.php
 *
 * Activates the ZID Proxy package by executing installation hooks.
 * This script creates the rc.d script and initializes the configuration.
 *
 * Usage: php activate-package.php
 *
 * Licensed under the Apache License, Version 2.0
 */

echo "=========================================\n";
echo " ZID Proxy Package Activation\n";
echo "=========================================\n\n";

// Check if running as root
if (posix_geteuid() !== 0) {
    echo "Error: This script must be run as root\n";
    exit(1);
}

// Check if package files exist
if (!file_exists('/usr/local/pkg/zid-proxy.inc')) {
    echo "Error: Package files not found. Please run install.sh first.\n";
    exit(1);
}

echo "Loading package functions...\n";
require_once('/usr/local/pkg/zid-proxy.inc');

echo "Executing installation hook...\n";
$result = zidproxy_install();

if ($result === false) {
    echo "\nError: Installation hook failed!\n";
    exit(1);
}

echo "\nInstallation hook executed successfully!\n\n";

// Verify rc.d script was created
$rcfile = '/usr/local/etc/rc.d/zid-proxy.sh';
if (file_exists($rcfile)) {
    echo "✓ RC script created: {$rcfile}\n";
    chmod($rcfile, 0755);
} else {
    echo "✗ Warning: RC script not found at {$rcfile}\n";
}

// Verify config directory
$config_dir = '/usr/local/etc/zid-proxy';
if (is_dir($config_dir)) {
    echo "✓ Config directory created: {$config_dir}\n";
} else {
    echo "✗ Warning: Config directory not found at {$config_dir}\n";
}

// Verify rules file
$rules_file = '/usr/local/etc/zid-proxy/access_rules.txt';
if (file_exists($rules_file)) {
    echo "✓ Rules file created: {$rules_file}\n";
} else {
    echo "✗ Warning: Rules file not found at {$rules_file}\n";
}

echo "\n=========================================\n";
echo " Activation Complete!\n";
echo "=========================================\n\n";

echo "You can now use the service:\n";
echo "  {$rcfile} start\n";
echo "  {$rcfile} status\n";
echo "  {$rcfile} stop\n\n";

echo "Note: The pfSense web interface may not show the 'Services > ZID Proxy'\n";
echo "menu until you run register-package.php or restart pfSense.\n\n";

?>
