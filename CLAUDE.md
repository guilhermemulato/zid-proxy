# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

zid-proxy is a transparent SNI proxy for pfSense 2.8.1 (FreeBSD 15.x) written in Go. It performs dual-factor filtering based on source IP and destination hostname (extracted from TLS SNI extension).

## Build Commands

```bash
# Build for FreeBSD (pfSense target)
make build-freebsd
# Or directly:
GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o build/zid-proxy ./cmd/zid-proxy

# Build for local testing
make build

# Run tests
make test

# Run locally for development
make run
```

## Architecture

```
cmd/zid-proxy/main.go      # Entry point, signal handling, PID management
internal/
  config/config.go         # Configuration loading
  sni/parser.go            # TLS ClientHello parsing, SNI extraction
  rules/rules.go           # Rule file parsing and matching logic
  proxy/server.go          # TCP listener, connection accept loop
  proxy/handler.go         # Connection handler, RST blocking, bidirectional proxy
  logger/logger.go         # File-based structured logging
scripts/rc.d/zid-proxy     # FreeBSD service script
```

## Key Technical Details

- **SNI Extraction**: Parse TLS ClientHello to extract server_name extension (type 0x0000)
- **Rule Format**: `TYPE;IP_OR_CIDR;HOSTNAME` (e.g., `BLOCK;192.168.1.0/24;*.facebook.com`)
- **Decision Logic**: ALLOW priority > BLOCK; default ALLOW if no match
- **Block Action**: TCP RST via `SetLinger(0)` before close
- **Config Reload**: SIGHUP signal triggers rules reload
- **Log Format**: `TIMESTAMP | SOURCE_IP | HOSTNAME | ACTION`

## Configuration Files

- Rules: `/usr/local/etc/zid-proxy/access_rules.txt`
- Log: `/var/log/zid-proxy.log`
- PID: `/var/run/zid-proxy.pid`

## References

- SNI Proxy pattern: https://www.agwa.name/blog/post/writing_an_sni_proxy_in_go
- pfSense packages: https://docs.netgate.com/pfsense/en/latest/development/develop-packages.html
- e2guardian reference: https://github.com/marcelloc/e2guardian
