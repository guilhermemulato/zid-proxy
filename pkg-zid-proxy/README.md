# ZID Proxy - pfSense Package Installation Scripts

This directory contains installation scripts for deploying ZID Proxy on pfSense.

## Quick Start

```bash
# From your build machine
make build-freebsd
scp -r build/zid-proxy pkg-zid-proxy root@pfsense:/tmp/

# On pfSense
ssh root@pfsense
cd /tmp/pkg-zid-proxy
sh install.sh
```

## Installation Scripts

### install.sh
Main installation script that:
- Copies all package files to their correct locations
- Installs binary, web interface, and configuration files
- Automatically runs activation and registration scripts
- Provides interactive prompts for package registration

### activate-package.php
Activates the package by executing PHP installation hooks:
- Creates the RC service script
- Initializes configuration directory
- Creates default rules file
- Sets up logging

**When to use:** Run this if `service zid-proxy` command doesn't work.

```bash
php activate-package.php
```

### register-package.php
Registers the package in pfSense's config.xml:
- Adds package entry to installed packages list
- Makes the package visible in the web interface
- Initializes default configuration

**When to use:** Run this if "Services > ZID Proxy" menu doesn't appear in the web interface.

```bash
php register-package.php
# Then reload web interface
/usr/local/sbin/pfSsh.php playback reloadwebgui
```

### diagnose.sh
Diagnostic script that checks:
- Binary installation
- Package files presence
- RC script existence
- Configuration directory
- Service status
- Package registration

**When to use:** Run this to troubleshoot installation issues.

```bash
sh diagnose.sh
```

### uninstall.sh
Complete removal script that:
- Stops the service
- Removes all package files
- Optionally removes configuration and logs
- Unregisters from pfSense
- Cleans up rc.conf entries

**When to use:** Run this to completely remove ZID Proxy from pfSense.

```bash
sh uninstall.sh
```

## Fixing "service not found" Error

If you get this error:
```
zid-proxy does not exist in /etc/rc.d or the local startup directories
```

**Solution 1 - Run activation script:**
```bash
cd /root/zid-proxy-pfsense  # or wherever you extracted the files
php activate-package.php
```

This creates the RC script at `/usr/local/etc/rc.d/zid-proxy.sh`

**Solution 2 - Manual activation:**
```bash
php <<'EOF'
<?php
require_once('/usr/local/pkg/zid-proxy.inc');
zidproxy_install();
?>
EOF
```

**Solution 3 - Copy RC script directly:**
```bash
cd /root/zid-proxy-pfsense
cp scripts/rc.d/zid-proxy /usr/local/etc/rc.d/
chmod +x /usr/local/etc/rc.d/zid-proxy
echo 'zid_proxy_enable="YES"' >> /etc/rc.conf.local
service zid-proxy start
```

After running any solution, test with:
```bash
service zid-proxy start
service zid-proxy status
```

## Making the Web Interface Menu Appear

If you've installed the package but can't see "Services > ZID Proxy" in the pfSense web interface:

```bash
# Register the package
php register-package.php

# Option 1: Reload web interface (fastest)
/usr/local/sbin/pfSsh.php playback reloadwebgui

# Option 2: Restart PHP-FPM
/usr/local/etc/rc.d/php-fpm restart

# Option 3: Restart pfSense (most reliable)
shutdown -r now
```

## File Locations

After installation, files are located at:

```
/usr/local/sbin/zid-proxy                          # Binary
/usr/local/pkg/zid-proxy.xml                       # Package manifest
/usr/local/pkg/zid-proxy.inc                       # PHP functions
/usr/local/www/zid-proxy_rules.php                 # Rules management page
/usr/local/www/zid-proxy_log.php                   # Log viewer page
/usr/local/etc/rc.d/zid-proxy.sh                   # RC service script
/usr/local/etc/zid-proxy/access_rules.txt          # Rules file
/var/log/zid-proxy.log                             # Log file
/var/run/zid-proxy.pid                             # PID file
/etc/inc/priv/zid-proxy.priv.inc                   # Privileges
/usr/local/share/pfSense-pkg-zid-proxy/info.xml    # Package info
```

## Architecture

The pfSense package uses a different approach than standalone installation:

1. **Package files are copied** by `install.sh`
2. **RC script is generated dynamically** by PHP function `zidproxy_write_rcfile()` in `zid-proxy.inc`
3. **Package is registered** in `/cf/conf/config.xml` by `register-package.php`
4. **Service is managed** via pfSense GUI or rc.d script

Note: The RC script is created at `/usr/local/etc/rc.d/zid-proxy.sh` (with `.sh` extension), not `/usr/local/etc/rc.d/zid-proxy`.

## Common Issues

### Issue: "Package files not found" when running activate-package.php
**Solution:** Run `install.sh` first to copy package files.

### Issue: Service starts but menu doesn't appear in GUI
**Solution:** Run `register-package.php` and reload web interface.

### Issue: "Permission denied" errors
**Solution:** Make sure you're running as root (`su -` or `sudo -i`).

### Issue: Service won't start
**Solution:**
1. Check if binary exists: `ls -lh /usr/local/sbin/zid-proxy`
2. Check RC script: `ls -lh /usr/local/etc/rc.d/zid-proxy.sh`
3. Check logs: `tail -f /var/log/zid-proxy.log`
4. Run diagnostics: `sh diagnose.sh`

### Issue: Need to start over
**Solution:** Run `uninstall.sh`, then `install.sh` again.

## Testing the Installation

After installation:

```bash
# Test service commands
/usr/local/etc/rc.d/zid-proxy.sh status
/usr/local/etc/rc.d/zid-proxy.sh start
/usr/local/etc/rc.d/zid-proxy.sh status

# Check if process is running
ps aux | grep zid-proxy

# View logs
tail -f /var/log/zid-proxy.log

# Test with a connection (from another machine)
openssl s_client -connect pfsense_ip:3129 -servername www.example.com
```

## Support

For issues or questions:
1. Run `diagnose.sh` and review the output
2. Check `/var/log/zid-proxy.log` for errors
3. Review pfSense system logs in the web interface
4. Check the main README.md in the repository root

## License

Apache License 2.0
