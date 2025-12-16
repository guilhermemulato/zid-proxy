.PHONY: all build build-freebsd clean test install

BINARY=zid-proxy
VERSION=1.0.3
BUILD_DIR=build
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

all: test build-freebsd

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/zid-proxy

build-freebsd:
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/zid-proxy

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)

install: build-freebsd
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/sbin/
	install -m 755 scripts/rc.d/zid-proxy /usr/local/etc/rc.d/
	mkdir -p /usr/local/etc/zid-proxy

run:
	go run ./cmd/zid-proxy -listen :8443 -rules configs/access_rules.txt -log /tmp/zid-proxy.log
