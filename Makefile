.PHONY: build install test lint clean

build:
	go build -o bin/nt ./cmd/nt

install:
	go install ./cmd/nt

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
