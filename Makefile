.PHONY: all build clean test tests test-quick test-configs fmt install rpi

all: build

build: bin/ys1-dump-config bin/ys1-load-config bin/test-configs bin/lsys1 bin/send-recv bin/test-10-repeat bin/profile-test bin/rf-scanner bin/plot-spectrum bin/fhss-demo

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

bin/plot-spectrum: cmd/plot-spectrum/main.go
	go build -o bin/plot-spectrum ./cmd/plot-spectrum

bin/fhss-demo: cmd/fhss-demo/main.go pkg/**/*.go
	go build -o bin/fhss-demo ./cmd/fhss-demo

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

# Cross-compile for Raspberry Pi 5 (ARM64)
# RPi 5 uses Cortex-A76 (ARMv8.2-A), 64-bit
# Requires: apt install gcc-aarch64-linux-gnu libusb-1.0-0-dev:arm64
# Or use crossbuild-essential-arm64 package
#
# If you don't have the cross-compiler, you can build on the Pi itself:
#   1. Copy source to Pi: rsync -av --exclude bin . pi@hostname:~/gocat/
#   2. On Pi: cd ~/gocat && make build
RPI_CC ?= aarch64-linux-gnu-gcc
RPI_CGO_CFLAGS ?= -I/usr/aarch64-linux-gnu/include
RPI_CGO_LDFLAGS ?= -L/usr/aarch64-linux-gnu/lib

rpi:
	@mkdir -p bin/rpi
	@echo "Cross-compiling for Raspberry Pi 5 (linux/arm64)..."
	@echo "Using CC=$(RPI_CC)"
	@if ! command -v $(RPI_CC) >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: Cross-compiler not found: $(RPI_CC)"; \
		echo ""; \
		echo "Install with:"; \
		echo "  sudo apt install gcc-aarch64-linux-gnu"; \
		echo "  sudo apt install libusb-1.0-0-dev"; \
		echo ""; \
		echo "For cross-compiled libusb, you may need:"; \
		echo "  sudo dpkg --add-architecture arm64"; \
		echo "  sudo apt update"; \
		echo "  sudo apt install libusb-1.0-0-dev:arm64"; \
		echo ""; \
		echo "Alternative: Build directly on the Pi:"; \
		echo "  rsync -av --exclude bin . pi@hostname:~/gocat/"; \
		echo "  ssh pi@hostname 'cd ~/gocat && make build'"; \
		exit 1; \
	fi
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/ys1-dump-config ./cmd/ys1-dump-config
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/ys1-load-config ./cmd/ys1-load-config
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/test-configs ./cmd/test-configs
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/lsys1 ./cmd/lsys1
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/send-recv ./cmd/send-recv
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/test-10-repeat ./cmd/test-10-repeat
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/profile-test ./cmd/profile-test
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/rf-scanner ./cmd/rf-scanner
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/plot-spectrum ./cmd/plot-spectrum
	CGO_ENABLED=1 CC=$(RPI_CC) CGO_CFLAGS="$(RPI_CGO_CFLAGS)" CGO_LDFLAGS="$(RPI_CGO_LDFLAGS)" \
		GOOS=linux GOARCH=arm64 go build -o bin/rpi/fhss-demo ./cmd/fhss-demo
	@echo ""
	@echo "Done. Binaries in bin/rpi/"
	@echo "Copy to Pi with: scp bin/rpi/* pi@<hostname>:~/"
	@echo ""
	@echo "On the Pi, ensure libusb is installed:"
	@echo "  sudo apt install libusb-1.0-0"

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
# All tests run regardless of failures; summary shown at end
tests: build
	@echo "=== Running Hardware Tests ==="
	@echo ""
	@failed=0; passed=0; \
	run_test() { \
		echo "Running: $$1"; \
		if $$1; then \
			passed=$$((passed + 1)); \
			echo "[PASSED] $$1"; \
		else \
			failed=$$((failed + 1)); \
			echo "[FAILED] $$1"; \
		fi; \
		echo ""; \
	}; \
	echo "=========================================="; \
	echo "=== PASS 1: Device #0=TX, Device #1=RX ==="; \
	echo "=========================================="; \
	echo ""; \
	echo "--- 315 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 315-2fsk-sync-4.8k -repeat 2 -tx '#0' -rx '#1'"; \
	run_test "./bin/profile-test -profile 315-2fsk-sync-9.6k -repeat 2 -tx '#0' -rx '#1'"; \
	echo "--- 433 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 433-2fsk-std-4.8k -repeat 2 -tx '#0' -rx '#1'"; \
	run_test "./bin/profile-test -profile 433-gfsk-crc-9.6k -repeat 2 -tx '#0' -rx '#1'"; \
	run_test "./bin/profile-test -profile 433-2fsk-std-9.6k -repeat 2 -tx '#0' -rx '#1'"; \
	echo "--- 868 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 868-gfsk-smart-9.6k -repeat 2 -tx '#0' -rx '#1'"; \
	run_test "./bin/profile-test -profile 868-gfsk-fec-19.2k -repeat 2 -tx '#0' -rx '#1'"; \
	echo "--- 915 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 915-2fsk-sensor-9.6k -repeat 2 -tx '#0' -rx '#1'"; \
	run_test "./bin/profile-test -profile 915-gfsk-std-38.4k -repeat 2 -tx '#0' -rx '#1'"; \
	echo "=========================================="; \
	echo "=== PASS 2: Device #1=TX, Device #0=RX ==="; \
	echo "=========================================="; \
	echo ""; \
	echo "--- 315 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 315-2fsk-sync-4.8k -repeat 2 -tx '#1' -rx '#0'"; \
	run_test "./bin/profile-test -profile 315-2fsk-sync-9.6k -repeat 2 -tx '#1' -rx '#0'"; \
	echo "--- 433 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 433-2fsk-std-4.8k -repeat 2 -tx '#1' -rx '#0'"; \
	run_test "./bin/profile-test -profile 433-gfsk-crc-9.6k -repeat 2 -tx '#1' -rx '#0'"; \
	run_test "./bin/profile-test -profile 433-2fsk-std-9.6k -repeat 2 -tx '#1' -rx '#0'"; \
	echo "--- 868 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 868-gfsk-smart-9.6k -repeat 2 -tx '#1' -rx '#0'"; \
	run_test "./bin/profile-test -profile 868-gfsk-fec-19.2k -repeat 2 -tx '#1' -rx '#0'"; \
	echo "--- 915 MHz Band Tests ---"; \
	run_test "./bin/profile-test -profile 915-2fsk-sensor-9.6k -repeat 2 -tx '#1' -rx '#0'"; \
	run_test "./bin/profile-test -profile 915-gfsk-std-38.4k -repeat 2 -tx '#1' -rx '#0'"; \
	echo "--- RF Reliability Test ---"; \
	run_test "./bin/test-10-repeat -c tests/etc/433-2fsk-std-4.8k.json -n 10 -delay 100ms"; \
	echo ""; \
	echo "=========================================="; \
	echo "=== TEST SUMMARY ==="; \
	echo "=========================================="; \
	echo "Passed: $$passed"; \
	echo "Failed: $$failed"; \
	echo "Total:  $$((passed + failed))"; \
	if [ $$failed -gt 0 ]; then \
		echo ""; \
		echo "*** $$failed TEST(S) FAILED ***"; \
		exit 1; \
	else \
		echo ""; \
		echo "=== ALL TESTS PASSED ==="; \
	fi

# Config verification test - tests register load/verify on single device
test-configs: build
	@echo "=== Config Verification Tests ==="
	./bin/test-configs -c tests/etc/315-2fsk-sync-4.8k.json
	./bin/test-configs -c tests/etc/433-2fsk-std-4.8k.json
	./bin/test-configs -c tests/etc/868-gfsk-smart-9.6k.json
	./bin/test-configs -c tests/etc/915-2fsk-sensor-9.6k.json
	@echo "=== Config Tests Complete ==="
