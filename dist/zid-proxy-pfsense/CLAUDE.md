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

# Repository Guidelines

## Project Structure

- `cmd/zid-proxy/`: main entrypoint and CLI flags.
- `internal/`: core implementation (`config/`, `sni/`, `rules/`, `proxy/`, `logger/`).
- `configs/`: example configuration (e.g., `configs/access_rules.txt`).
- `scripts/rc.d/`: FreeBSD/pfSense service script.
- `pkg-zid-proxy/`: pfSense package assets (XML/PHP pages, install helpers).
- `build/`: local build output (generated).
- `dist/`: packaged artifacts for pfSense releases (generated).
- Docs: `README.md`, `INSTALL-PFSENSE.md`, `TROUBLESHOOTING.md`, `CHANGELOG.md`.

## Build, Test, and Development Commands

- `make build`: build for your current OS into `build/zid-proxy`.
- `make build-freebsd`: build the pfSense target binary (`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0`).
- `make test`: run all Go tests (`go test -v ./...`).
- `make run`: run locally with sample rules and log file.
- `make clean`: remove `build/`.

## Coding Style & Naming Conventions

- Go code follows standard formatting: run `gofmt -w .` before opening a PR.
- Keep packages lowercase and cohesive (domain-oriented folders under `internal/`).
- Prefer explicit names over abbreviations (e.g., `rulesPath`, `listenAddr`).
- Keep configuration defaults and flags aligned with `README.md`/rc.conf examples.

## Testing Guidelines

- Tests use the Go standard library (`testing`) and live alongside code as `*_test.go`.
- Name tests `TestXxx` and keep them deterministic (no network access required).
- When changing rule matching or SNI parsing, add/adjust unit tests in:
  `internal/rules/`, `internal/sni/`, and `internal/logger/`.

## Commit & Pull Request Guidelines

- Current history uses short, release-style subjects (e.g., `Versao estavel 1.0.8`).
- For new work, use clear, imperative subjects; recommended patterns:
  `feat: ...`, `fix: ...`, `chore(release): 1.0.9`, or keep `Versao X.Y.Z` consistently.
- PRs should include: what changed, how you tested (`make test`, `make build-freebsd`), and
  any pfSense UI changes (screenshots for `pkg-zid-proxy/files/usr/local/www/`).

## Release Notes (pfSense)

- If you bump versions, update `Makefile` (`VERSION=...`) and relevant docs/artifacts
  (e.g., `CHANGELOG.md`, `INSTALL-PFSENSE.md`, and checksums like `sha256.txt`).


  ## Specs
- Sempre ao final, se necessario gere novamente os binarios e compacte todos os arquivos em um tar com a versao latest, para facilitar o scp
- Separe sempre os bundle dos binarios, um para zid-proxy, um para agente windows e outro para o agente linux
- Sempre apos alguma alteracao nos codigo registro o que foi alterado no CHANGELOG.md, criando uma nova versao na sequencia
- Caso seja uma alteracao bem pequena, so adicione um numero na versao, Tipo 1.0.8.1
- Deixe algum arquivo ou algum lugar salvo com a versao atual, para que quando o update.sh for executado no cliente, ele consiga comparar se é a mesma versao, caso seja significa que ja esta atualizado, e para o processo de update
- 