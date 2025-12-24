.PHONY: all build build-freebsd clean test install bundle-latest

BINARY=zid-proxy
LOGROTATE_BINARY=zid-proxy-logrotate
AGENT_BINARY=zid-agent
VERSION=1.0.11.3.2.9
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
		@echo "Building Linux agent (requires system dependencies)..."
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(AGENT_BINARY)-linux-amd64 ./cmd/zid-agent

build-agent-windows:
		@echo "Building Windows agent (requires MinGW cross-compiler)..."
		@if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then \
			CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
			CC=x86_64-w64-mingw32-gcc \
			$(GO) build -ldflags="-s -w -H windowsgui -X main.Version=$(VERSION)" \
				-o $(BUILD_DIR)/$(AGENT_BINARY)-windows-amd64.exe ./cmd/zid-agent; \
			echo "Build complete: $(BUILD_DIR)/$(AGENT_BINARY)-windows-amd64.exe"; \
		else \
			echo "ERROR: MinGW compiler not found. Install with:"; \
			echo "  Ubuntu/Debian: sudo apt-get install gcc-mingw-w64-x86-64"; \
			echo "  Fedora: sudo dnf install mingw64-gcc"; \
			echo "Or use fyne-cross: go install github.com/fyne-io/fyne-cross@latest"; \
			exit 1; \
		fi

# GUI bundles (same binaries, different names)
build-agent-linux-gui:
		$(MAKE) build-agent-linux
		cp -f $(BUILD_DIR)/$(AGENT_BINARY)-linux-amd64 $(BUILD_DIR)/$(AGENT_BINARY)-linux-gui

build-agent-windows-gui:
		$(MAKE) build-agent-windows
		cp -f $(BUILD_DIR)/$(AGENT_BINARY)-windows-amd64.exe $(BUILD_DIR)/$(AGENT_BINARY)-windows-gui.exe

# Alternative: use fyne-cross for cross-platform builds
build-agent-fyne-cross:
	@echo "Building with fyne-cross (recommended for production)..."
	@if command -v fyne-cross >/dev/null 2>&1; then \
		fyne-cross linux -arch=amd64 -output $(AGENT_BINARY)-linux-gui ./cmd/zid-agent; \
		fyne-cross windows -arch=amd64 -output $(AGENT_BINARY)-windows-gui ./cmd/zid-agent; \
	else \
		echo "ERROR: fyne-cross not found. Install with:"; \
		echo "  go install github.com/fyne-io/fyne-cross@latest"; \
		exit 1; \
	fi

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
