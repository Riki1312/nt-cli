.PHONY: build install test lint fmt fmt-check check clean

GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/gomod

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/nt ./cmd/nt

install:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go install ./cmd/nt

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test -race -timeout 2m ./...

lint:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go vet ./...

clean:
	rm -rf bin/ .cache/

fmt:
	gofmt -w cmd internal

fmt-check:
	@files="$$(gofmt -l cmd internal)" || exit $$?; \
	if [ -n "$$files" ]; then printf 'Run make fmt:\n%s\n' "$$files"; exit 1; fi

check: fmt-check lint test build
