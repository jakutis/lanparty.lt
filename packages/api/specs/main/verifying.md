# Verifying

Each endpoint has a test suite.
The test suite uses net/http/httptest - spins up a local, isolated HTTP server in memory and records HTTP responses without actually opening network ports.
The test suites are written using Go's built in `testing` package, Ginkgo and Gomega.

## Running tests

From the `packages/api` directory:

```bash
go test ./...
```

For verbose output, including the Ginkgo spec tree:

```bash
go test -v ./...
```
