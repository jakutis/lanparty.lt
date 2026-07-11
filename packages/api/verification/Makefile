# Makefile for the api package's external blackbox tests. See
# ../specification/main.md#verification. Requires the k6 executable.

.PHONY: test test-verbose install-k6

test:
	go test ./...

test-verbose:
	go test -v ./...

# Install the k6 executable into $HOME/go/bin using the Go toolchain. Useful
# where prebuilt k6 binaries cannot be downloaded.
install-k6:
	go install go.k6.io/k6@latest
