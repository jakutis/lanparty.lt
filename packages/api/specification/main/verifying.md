# Verifying

Testing is split into two distinct layers to ensure both comprehensive coverage and true blackbox isolation. The individual test cases of both layers are enumerated in [../tests.md](../tests.md).

## 1. API Endpoint Tests (External Blackbox)

Located in `packages/api/verification/`, these tests treat the API as a completely opaque compiled binary.

- **Framework**: HTTP endpoints are tested using [k6](https://k6.io/).
- **Execution Strategy**: A Go test wrapper acts as the orchestrator. When `go test` runs in the `verification` directory, the wrapper:
  1. Dynamically compiles the API into a standalone executable binary using `go build`.
  2. Starts a **fake OpenRouter mock server** on a local port.
  3. Finds a free local network port and starts the compiled API binary as a background child process, configured via environment variables (e.g. `PORT` and `OPENROUTER_BASE_URL`).
  4. Invokes the `k6 run` command against the running API server. A single
     failing check fails the k6 run, and with it the Go test wrapper.
  5. Cleans up all processes and binaries upon completion.

This architecture guarantees that the API is tested exactly as it would run in production.

## 2. Internal Unit Tests

Located in `packages/api/implementation/src/`, these tests cover complex internal business logic without network overhead.

- **Framework**: Go's built-in `testing` package, [Ginkgo](https://onsi.github.io/ginkgo/), and [Gomega](https://onsi.github.io/gomega/).
- **Execution Strategy**: Runs standard Go unit tests to verify internal components (e.g., the LLM generator's iteration limits, content-type mapping).

## Running tests

The two layers live in separate Go modules and are run independently
(both require Go 1.26): the internal unit tests from the
`packages/api/implementation` directory, and the external blackbox tests
from the `packages/api/verification` directory. The implementation does not
reference the verification layer; the dependency points only from
verification to implementation.

In either directory, using `make`:

```bash
make test
```

For verbose output (in `implementation`, this includes the Ginkgo spec tree):

```bash
make test-verbose
```

`make test` and `make test-verbose` do not require OpenRouter configuration.

The external test wrapper requires the `k6` executable. It first looks for
`k6` on `PATH`, then falls back to `$HOME/go/bin/k6`.

Alternatively, using `go test` directly in either directory:

```bash
go test ./...
```

For verbose output:

```bash
go test -v ./...
```
