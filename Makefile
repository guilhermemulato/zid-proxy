.PHONY: all build build-freebsd clean test install bundle-latest

BINARY=zid-proxy
LOGROTATE_BINARY=zid-proxy-logrotate
AGENT_BINARY=zid-agent
VERSION=1.0.11.3.2.4
BUILD_DIR=build
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

# Go toolchain helper:
# - On normal shells, uses `go`.
# - Inside Flatpak (e.g. VSCode Flatpak), uses `flatpak-spawn --host go` so builds/tests
#   run with the host toolchain.
GO_CMD?=go
GO?=$(GO_CMD)
ifneq ($(FLATPAK_ID),)
ifneq ($(shell command -v flatpak-spawn 2>/dev/null),)
GO=flatpak-spawn --host $(GO_CMD)
endif
endif

all: test build-freebsd

build:
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/zid-proxy
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(LOGROTATE_BINARY) ./cmd/zid-proxy-logrotate
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(AGENT_BINARY) ./cmd/zid-agent

build-freebsd:
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/zid-proxy
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(LOGROTATE_BINARY) ./cmd/zid-proxy-logrotate

build-agent-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(AGENT_BINARY)-linux-amd64 ./cmd/zid-agent

build-agent-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(AGENT_BINARY)-windows-amd64.exe ./cmd/zid-agent

test:
	$(GO) test -v ./...

clean:
	rm -rf $(BUILD_DIR)

install: build-freebsd
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/sbin/
	install -m 755 scripts/rc.d/zid-proxy /usr/local/etc/rc.d/
	mkdir -p /usr/local/etc/zid-proxy

run:
	$(GO) run ./cmd/zid-proxy -listen :8443 -rules configs/access_rules.txt -log /tmp/zid-proxy.log

bundle-latest: build-freebsd build-agent-linux build-agent-windows
	chmod +x scripts/bundle-latest.sh
	./scripts/bundle-latest.sh
