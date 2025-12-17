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
