BINARY  := logbee
PKG     := github.com/neatnettech/logbee
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build install test vet lint fmt clean release-snapshot

all: vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

vet:
	go vet ./...

# Requires golangci-lint (https://golangci-lint.run).
lint:
	golangci-lint run

fmt:
	gofmt -w .

# Local release dry-run; requires goreleaser (https://goreleaser.com).
release-snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -f $(BINARY)
	rm -rf dist
