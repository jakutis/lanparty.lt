# Running

All commands below are run from the `packages/api/implementation` directory.

Using `make`:
```bash
PORT=8080 OPENROUTER_API_KEY=... OPENROUTER_MODEL=... make run
```

`make run` first checks that `OPENROUTER_API_KEY` and `OPENROUTER_MODEL` are
both non-empty, and fails without starting the server when either is absent.

`make curl` sends the example `html` request below to the server. It uses
`PORT` when non-empty and otherwise uses port `8080`.

Alternatively, using `go run`:
```bash
PORT=8080 OPENROUTER_API_KEY=... OPENROUTER_MODEL=... go run ./src
```

```bash
curl -XPOST localhost:8080/v1/representation -d '{"type":"html","spec":"a hello world web page"}'
```

`PORT` is optional and defaults to `8080`; it selects the TCP port the server
listens on. The `OPENROUTER_*` variables are documented in the
[Generator configuration](generator.md#configuration).

The HTTP server is configured with a 10-second read-header timeout, a
60-second read timeout, and a 5-minute write timeout (headroom above the
Generator's 4-minute upstream HTTP timeout).

## Running the tests

The two test layers live in separate Go modules and are run independently
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

Alternatively, using `go test` directly in either directory:

```bash
go test ./...
```

For verbose output:

```bash
go test -v ./...
```

The internal command-sandbox cases execute real sandboxed commands, so they
need `bwrap` on `PATH` and a kernel permitting unprivileged user namespaces
(see the toolchain requirements in [main.md](main.md)).

### The external test harness

When `go test` runs in the `verification` directory, a Go test wrapper acts
as the orchestrator:

1. Dynamically compiles the API into a standalone executable binary using `go build`.
2. Starts a **fake OpenRouter mock server** on a local port.
3. Finds a free local network port and starts the compiled API binary as a background child process, configured via environment variables (e.g. `PORT` and `OPENROUTER_BASE_URL`).
4. Invokes the `k6 run` command against the running API server. A single
   failing check fails the k6 run, and with it the Go test wrapper.
5. Cleans up all processes and binaries upon completion.

This architecture guarantees that the API is tested exactly as it would run in production.

The external test wrapper requires the `k6` executable. It first looks for
`k6` on `PATH`, then falls back to `$HOME/go/bin/k6`. If k6 is not installed,
`make install-k6` (from the `verification` directory) builds and installs it
into `$HOME/go/bin` using the Go toolchain (`go install go.k6.io/k6@latest`) —
useful where prebuilt k6 binaries cannot be downloaded.

### Source formatting

Verification (external blackbox):

- **Go sources are gofmt-formatted** — `gofmt -l` over the package's
  `implementation/` and `verification/` directories reports no files.
