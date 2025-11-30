.PHONY: all build clean test tests test-quick test-configs fmt install

all: build

build: bin/ys1-dump-config bin/ys1-load-config bin/test-configs bin/lsys1 bin/send-recv bin/test-10-repeat bin/profile-test bin/rf-scanner

bin/ys1-dump-config: cmd/ys1-dump-config/main.go pkg/**/*.go
	go build -o bin/ys1-dump-config ./cmd/ys1-dump-config

bin/ys1-load-config: cmd/ys1-load-config/main.go pkg/**/*.go
	go build -o bin/ys1-load-config ./cmd/ys1-load-config

bin/test-configs: cmd/test-configs/main.go pkg/**/*.go
	go build -o bin/test-configs ./cmd/test-configs

bin/lsys1: cmd/lsys1/main.go pkg/**/*.go
	go build -o bin/lsys1 ./cmd/lsys1

bin/send-recv: cmd/send-recv/main.go pkg/**/*.go
	go build -o bin/send-recv ./cmd/send-recv

bin/test-10-repeat: cmd/test-10-repeat/main.go pkg/**/*.go
	go build -o bin/test-10-repeat ./cmd/test-10-repeat

bin/profile-test: cmd/profile-test/main.go pkg/**/*.go
	go build -o bin/profile-test ./cmd/profile-test

bin/rf-scanner: cmd/rf-scanner/main.go pkg/**/*.go
	go build -o bin/rf-scanner ./cmd/rf-scanner

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

# Hardware tests - requires YardStick One devices connected
# Tests representative profiles across bands and modulation types

# Quick test - single profile for fast validation
test-quick: build
	@echo "=== Quick Hardware Test ==="
	./bin/profile-test -profile 433-2fsk-std-4.8k -repeat 1
	@echo "=== Quick Test Complete ==="

# Full test suite - representative profiles from each band/modulation
# Note: Only profiles with sync word enabled are used for reliable loopback testing
# Tests run twice with swapped device roles to ensure both devices work as TX and RX
tests: build
	@echo "=== Running Hardware Tests ==="
	@echo ""
	@echo "=========================================="
	@echo "=== PASS 1: Device #0=TX, Device #1=RX ==="
	@echo "=========================================="
	@echo ""
	@echo "--- 315 MHz Band Tests ---"
	./bin/profile-test -profile 315-2fsk-sync-4.8k -repeat 2 -tx "#0" -rx "#1"
	./bin/profile-test -profile 315-2fsk-sync-9.6k -repeat 2 -tx "#0" -rx "#1"
	@echo ""
	@echo "--- 433 MHz Band Tests ---"
	./bin/profile-test -profile 433-2fsk-std-4.8k -repeat 2 -tx "#0" -rx "#1"
	./bin/profile-test -profile 433-gfsk-crc-9.6k -repeat 2 -tx "#0" -rx "#1"
	./bin/profile-test -profile 433-2fsk-std-9.6k -repeat 2 -tx "#0" -rx "#1"
	@echo ""
	@echo "--- 868 MHz Band Tests ---"
	./bin/profile-test -profile 868-gfsk-smart-9.6k -repeat 2 -tx "#0" -rx "#1"
	./bin/profile-test -profile 868-gfsk-fec-19.2k -repeat 2 -tx "#0" -rx "#1"
	@echo ""
	@echo "--- 915 MHz Band Tests ---"
	./bin/profile-test -profile 915-2fsk-sensor-9.6k -repeat 2 -tx "#0" -rx "#1"
	./bin/profile-test -profile 915-gfsk-std-38.4k -repeat 2 -tx "#0" -rx "#1"
	@echo ""
	@echo "=========================================="
	@echo "=== PASS 2: Device #1=TX, Device #0=RX ==="
	@echo "=========================================="
	@echo ""
	@echo "--- 315 MHz Band Tests ---"
	./bin/profile-test -profile 315-2fsk-sync-4.8k -repeat 2 -tx "#1" -rx "#0"
	./bin/profile-test -profile 315-2fsk-sync-9.6k -repeat 2 -tx "#1" -rx "#0"
	@echo ""
	@echo "--- 433 MHz Band Tests ---"
	./bin/profile-test -profile 433-2fsk-std-4.8k -repeat 2 -tx "#1" -rx "#0"
	./bin/profile-test -profile 433-gfsk-crc-9.6k -repeat 2 -tx "#1" -rx "#0"
	./bin/profile-test -profile 433-2fsk-std-9.6k -repeat 2 -tx "#1" -rx "#0"
	@echo ""
	@echo "--- 868 MHz Band Tests ---"
	./bin/profile-test -profile 868-gfsk-smart-9.6k -repeat 2 -tx "#1" -rx "#0"
	./bin/profile-test -profile 868-gfsk-fec-19.2k -repeat 2 -tx "#1" -rx "#0"
	@echo ""
	@echo "--- 915 MHz Band Tests ---"
	./bin/profile-test -profile 915-2fsk-sensor-9.6k -repeat 2 -tx "#1" -rx "#0"
	./bin/profile-test -profile 915-gfsk-std-38.4k -repeat 2 -tx "#1" -rx "#0"
	@echo ""
	@echo "--- RF Reliability Test ---"
	./bin/test-10-repeat -c tests/etc/433-2fsk-std-4.8k.json -n 10 -delay 100ms
	@echo ""
	@echo "=== All Hardware Tests Complete ==="

# Config verification test - tests register load/verify on single device
test-configs: build
	@echo "=== Config Verification Tests ==="
	./bin/test-configs -c tests/etc/315-2fsk-sync-4.8k.json
	./bin/test-configs -c tests/etc/433-2fsk-std-4.8k.json
	./bin/test-configs -c tests/etc/868-gfsk-smart-9.6k.json
	./bin/test-configs -c tests/etc/915-2fsk-sensor-9.6k.json
	@echo "=== Config Tests Complete ==="
