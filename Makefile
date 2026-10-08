# Blockworld - build helpers.
# Each target must be run ON that platform (raylib is compiled via cgo).

BIN := blockworld

.PHONY: run build build-macos build-linux build-windows clean app-macos app-linux app-windows

include packaging.mk

run:
	go run .

build:
	go build -o $(BIN) .

build-macos:
	go build -ldflags="-s -w" -o $(BIN)-macos .

build-linux:
	go build -ldflags="-s -w" -o $(BIN)-linux .

build-windows:
	go build -ldflags="-s -w -H windowsgui" -o $(BIN).exe .

clean:
	rm -rf $(BIN) $(BIN)-macos $(BIN)-linux $(BIN).exe dist *.syso
