.PHONY: build install test lint clean

GOCACHE ?= $(CURDIR)/.cache/go-build

build:
	go build -o bin/nt ./cmd/nt

install:
	go install ./cmd/nt

test:
	GOCACHE=$(GOCACHE) go test ./...

lint:
	GOCACHE=$(GOCACHE) go vet ./...

clean:
	rm -rf bin/ .cache/
