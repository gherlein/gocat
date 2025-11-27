.PHONY: all build clean test fmt install

all: build

build: bin/ys1-dump-config bin/ys1-load-config

bin/ys1-dump-config: cmd/ys1-dump-config/main.go pkg/**/*.go
	go build -o bin/ys1-dump-config ./cmd/ys1-dump-config

bin/ys1-load-config: cmd/ys1-load-config/main.go pkg/**/*.go
	go build -o bin/ys1-load-config ./cmd/ys1-load-config

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
