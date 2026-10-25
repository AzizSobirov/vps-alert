VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install release test vet clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/vps-alert ./cmd/vps-alert

# Builds as the current user (root usually has no Go on its PATH), then
# installs with sudo.
install: build
	sudo sh deploy/install.sh bin/vps-alert

release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/vps-alert-linux-amd64 ./cmd/vps-alert
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/vps-alert-linux-arm64 ./cmd/vps-alert

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin dist
