.PHONY: all build clean test fmt install

all: build

build: bin/ys1-dump-config bin/ys1-load-config bin/test-configs bin/lsys1

bin/ys1-dump-config: cmd/ys1-dump-config/main.go pkg/**/*.go
	go build -o bin/ys1-dump-config ./cmd/ys1-dump-config

bin/ys1-load-config: cmd/ys1-load-config/main.go pkg/**/*.go
	go build -o bin/ys1-load-config ./cmd/ys1-load-config

bin/test-configs: cmd/test-configs/main.go pkg/**/*.go
	go build -o bin/test-configs ./cmd/test-configs

bin/lsys1: cmd/lsys1/main.go pkg/**/*.go
	go build -o bin/lsys1 ./cmd/lsys1

clean:
	rm -rf bin/
	go clean

test:
	go test ./...

fmt:
	go fmt ./...

install: build
	install -m 755 bin/ys1-dump-config /usr/local/bin/
	install -m 755 bin/ys1-load-config /usr/local/bin/
