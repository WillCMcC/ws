.PHONY: build test install clean release

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
GO := go

build:
	$(GO) build $(LDFLAGS) -o ws .

test:
	$(GO) test -v ./...

install: build
	mv ws /usr/local/bin/

clean:
	rm -f ws
	rm -rf dist/

release:
	goreleaser release --clean

# Development helpers
fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

.DEFAULT_GOAL := build
