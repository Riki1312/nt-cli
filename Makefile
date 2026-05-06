.PHONY: build install test lint clean

GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/gomod

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/nt ./cmd/nt

install:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go install ./cmd/nt

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...

lint:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go vet ./...

clean:
	rm -rf bin/ .cache/
