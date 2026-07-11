# API package

An HTTP server written in Go language. Running or testing this package requires
Go 1.26, as declared by its module.

Executing model tool commands — and running the unit tests that cover them —
additionally requires [bubblewrap](https://github.com/containers/bubblewrap)
(`bwrap`) on `PATH` and a Linux kernel that permits unprivileged user
namespaces; see the [command sandbox](generator.md#command-sandbox).

Package-wide convention: the server writes its log to standard error, and
every request received and every error condition produces at least one log
line.

Testing is split into two distinct layers to ensure both comprehensive
coverage and true blackbox isolation:

- **External blackbox tests** — located in `packages/api/verification/`,
  these treat the API as a completely opaque compiled binary and drive its
  HTTP endpoints with [k6](https://k6.io/).
- **Internal unit tests** — located in `packages/api/implementation/src/`,
  these cover complex internal business logic (e.g., the LLM generator's
  iteration limits, content-type mapping) without network overhead, using
  Go's built-in `testing` package, [Ginkgo](https://onsi.github.io/ginkgo/),
  and [Gomega](https://onsi.github.io/gomega/).

Each behavior section ends with a **Verification** subsection enumerating the
test cases that cover its behavior, labelled with the layer they belong to.
The cases specify the behavior covered by every test, not how that behavior
is tested. How the suites are executed is described under
[Running the tests](running.md#running-the-tests).

## Specification contents

The spec is split across these documents:

- [Running](running.md) — how to run the server and how to run both test
  suites.
- [Endpoints](endpoints.md) — the HTTP contract: routes, request validation,
  response headers, and error responses.
- [Generator](generator.md) — the LLM-backed Generator: configuration, the
  OpenRouter request shape, the prompt template, the generation loop, and the
  command sandbox.
