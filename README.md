# zid-proxy

Transparent SNI proxy for pfSense 2.8.1 (FreeBSD 15.x) with dual-factor filtering based on source IP and destination hostname (extracted from TLS SNI extension).

## Features

- **Transparent HTTPS Proxy**: Intercepts TLS traffic on port 443
- **SNI-based Filtering**: Extracts hostname from TLS ClientHello without terminating TLS
- **Dual-Factor Rules**: Filters based on source IP/CIDR + destination hostname
- **ALLOW Priority**: ALLOW rules take precedence over BLOCK rules
- **Default Allow**: If no rule matches, connection is allowed
- **TCP RST Blocking**: Blocked connections receive immediate TCP RST
- **Runtime Reload**: Rules can be reloaded via SIGHUP without restart
- **Structured Logging**: All connections logged with timestamp, IP, hostname, and action

## Build

### For FreeBSD/pfSense (production)

```bash
make build-freebsd
```

This creates `build/zid-proxy` - a statically linked binary for FreeBSD amd64.

### For local development

```bash
make build    # Build for current platform
make test     # Run tests
make run      # Run locally for testing
```

## Installation on pfSense

1. Copy the binary to pfSense:
```bash
scp build/zid-proxy root@pfsense:/usr/local/sbin/
```

2. Copy the rc.d script:
```bash
scp scripts/rc.d/zid-proxy root@pfsense:/usr/local/etc/rc.d/
chmod +x /usr/local/etc/rc.d/zid-proxy
```

3. Enable the service:
```bash
echo 'zid_proxy_enable="YES"' >> /etc/rc.conf
```

4. Start the service:
```bash
service zid-proxy start
```

## Configuration

### Rules File

Located at `/usr/local/etc/zid-proxy/access_rules.txt`

Format: `TYPE;IP_OR_CIDR;HOSTNAME`

```
# Block social media for entire subnet
BLOCK;192.168.1.0/24;*.facebook.com
BLOCK;192.168.1.0/24;*.twitter.com

# But allow specific host
ALLOW;192.168.1.100;*.facebook.com

# Block streaming for specific IP
BLOCK;192.168.1.50;*.netflix.com
```

### Rule Matching Logic

1. Rules are evaluated in order
2. **ALLOW** rules have priority over BLOCK rules
3. If no rule matches, the connection is **ALLOWED** (default)
4. Hostname wildcards: `*.example.com` matches `www.example.com`, `api.example.com`, and `example.com`

### rc.conf Options

```sh
zid_proxy_enable="YES"                                    # Enable service
zid_proxy_listen=":443"                                   # Listen address
zid_proxy_rules="/usr/local/etc/zid-proxy/access_rules.txt"  # Rules file
zid_proxy_log="/var/log/zid-proxy.log"                    # Log file
```

## Service Management

```bash
service zid-proxy start     # Start the service
service zid-proxy stop      # Stop the service
service zid-proxy status    # Check status
service zid-proxy reload    # Reload rules (SIGHUP)
```

## Log Format

Location: `/var/log/zid-proxy.log`

```
2025-01-15T10:30:45Z | 192.168.1.100 | www.facebook.com | ALLOW
2025-01-15T10:30:46Z | 192.168.1.50 | www.facebook.com | BLOCK
```

## Firewall Integration

To use zid-proxy as a transparent proxy, configure pfSense to redirect HTTPS traffic:

### Port Forward (NAT)
Navigate to: Firewall > NAT > Port Forward

- Interface: LAN
- Protocol: TCP
- Destination: any
- Destination Port: 443
- Redirect Target IP: 127.0.0.1
- Redirect Target Port: 443

## Testing

### Local test (development)

```bash
# Start with test configuration
./build/zid-proxy -listen :8443 -rules configs/access_rules.txt -log /tmp/zid-proxy.log

# In another terminal, test with curl
curl -v --resolve www.example.com:8443:127.0.0.1 https://www.example.com:8443/
```

### Verify blocking

```bash
# Add a BLOCK rule
echo "BLOCK;0.0.0.0/0;blocked.example.com" >> configs/access_rules.txt

# Send SIGHUP to reload
kill -HUP $(cat /tmp/zid-proxy.pid)

# Test - should fail with "Connection reset by peer"
curl -v --resolve blocked.example.com:8443:127.0.0.1 https://blocked.example.com:8443/
```

## Architecture

```
cmd/zid-proxy/main.go        # Entry point, signal handling
internal/
  sni/parser.go              # TLS ClientHello parsing, SNI extraction
  rules/rules.go             # Rule parsing and matching
  proxy/server.go            # TCP listener, connection handling
  proxy/handler.go           # Connection handler, RST blocking, bidirectional proxy
  logger/logger.go           # Structured file logging
  config/config.go           # Configuration management
scripts/rc.d/zid-proxy       # FreeBSD service script
pkg-zid-proxy/               # pfSense package files
  files/usr/local/pkg/       # XML/PHP configuration
  files/usr/local/www/       # Web interface pages
```

## pfSense GUI Installation (Phase 2)

The `pkg-zid-proxy/` directory contains files for pfSense web interface integration.

### Quick Installation

```bash
# On your build machine
make build-freebsd

# Copy everything to pfSense
scp -r build/zid-proxy pkg-zid-proxy root@pfsense:/tmp/

# On pfSense, run the installer
ssh root@pfsense
cd /tmp/pkg-zid-proxy
sh install.sh
cp /tmp/build/zid-proxy /usr/local/sbin/
chmod +x /usr/local/sbin/zid-proxy
```

### Manual Installation

```bash
# Copy binary
scp build/zid-proxy root@pfsense:/usr/local/sbin/

# Copy package files
scp pkg-zid-proxy/files/usr/local/pkg/zid-proxy.xml root@pfsense:/usr/local/pkg/
scp pkg-zid-proxy/files/usr/local/pkg/zid-proxy.inc root@pfsense:/usr/local/pkg/
scp pkg-zid-proxy/files/usr/local/www/zid-proxy_rules.php root@pfsense:/usr/local/www/
scp pkg-zid-proxy/files/usr/local/www/zid-proxy_log.php root@pfsense:/usr/local/www/
scp pkg-zid-proxy/files/etc/inc/priv/zid-proxy.priv.inc root@pfsense:/etc/inc/priv/
scp pkg-zid-proxy/files/usr/local/share/pfSense-pkg-zid-proxy/info.xml root@pfsense:/usr/local/share/pfSense-pkg-zid-proxy/
```

### GUI Features

After installation, navigate to **Services > ZID Proxy** in pfSense:

- **Settings Tab**: Enable/disable service, configure listen interface and port
- **Access Rules Tab**: Add, edit, and delete filtering rules via web interface
- **Logs Tab**: View connection logs with real-time filtering results

### Firewall NAT Configuration

To intercept HTTPS traffic transparently:

1. Go to **Firewall > NAT > Port Forward**
2. Add a new rule:
   - Interface: LAN
   - Protocol: TCP
   - Source: LAN net (or specific subnet)
   - Destination: any (or specific destinations)
   - Destination Port: 443
   - Redirect Target IP: 127.0.0.1 (or interface IP)
   - Redirect Target Port: 3129 (or your configured port)

## License

Apache 2.0
# zid-proxy
