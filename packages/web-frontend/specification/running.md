# Running

Serve this package's directory with any static file server, behind the shared
reverse proxy described in [Deployment](deployment.md), then open the page in
a browser. The bundled `Caddyfile` is the recommended way to run it.

The package also provides these `make` targets:

- `make run`, from the `implementation/` directory, checks that
  `OPENROUTER_API_KEY` and `OPENROUTER_MODEL` are non-empty, starts the api
  backend on port `8080`, and starts Caddy with the bundled `Caddyfile`.
  Stopping the command also stops the backend it started.
- `make test`, from the `verification/` directory, runs the automated Node.js
  test suite (see [Running the tests](#running-the-tests)).

## Running the tests

Run, from the package's `verification/` directory (requires Node.js 18 or
later and the Go toolchain — Go 1.26, used to compile the api binary the
contract tests run against; no required Node package dependencies, though
optional tooling unlocks the browser and formatting checks — see
`make install-tools` below):

```bash
node --test *.test.js
```

Equivalently, run `make test` from the `verification/` directory.

The suite is split into four files, plus `verification/harness.js`, a shared
non-test helper that builds the api binary and starts the origin/api/fake
OpenRouter stack:

- **`logic.test.js`** — extracts the pure, DOM-free helpers embedded inline
  in `implementation/src/index.html` (validation, error extraction, the
  blob-document builder, MIME mapping, the network-error message) and asserts
  their behavior. When `module.exports` is available, that inline script
  exposes these helpers as `validateSpec`, `extractErrorMessage`,
  `networkErrorMessage`, `blobMimeType`, and `buildBlobDocument` so the test
  can invoke them without a browser DOM. This export is a test hook; normal
  browser behavior still uses the same helpers through the form's submit
  listener. It also reads `index.html` directly (no HTTP server) to assert
  the page loads `marked` from a CDN and has the required form controls, as
  a quick file-level check that duplicates part of what `contract.test.js`
  verifies over HTTP.
- **`contract.test.js`** — spins up a Caddyfile-mirroring origin (static
  `src/` at `/`, `/v1/*` proxied unchanged to the REAL api binary, compiled
  from `packages/api/implementation` and configured against a fake OpenRouter
  upstream; see `verification/harness.js`) and asserts the HTTP contract.
  Because the api is the real binary, these tests cannot drift from the api
  spec.
- **`browser.test.js`** — drives the page in a real (headless) Chromium via
  `playwright-core`, against the same origin/api stack as the contract
  tests, covering the checks that need a browser. To keep the suite
  hermetic, the page's request for the marked CDN bundle is intercepted and
  fulfilled with a passthrough stub (`parse` = identity); the test asserts
  the page requested the required CDN URL, and real markdown rendering stays
  covered by `logic.test.js`. `playwright-core` is an optional dependency:
  install it with `make install-tools`; when it is absent this file is
  skipped and the rest of the suite runs unchanged. The Chromium executable
  is resolved from `CHROMIUM_PATH`, then `/opt/pw-browsers/chromium` when
  present, then `playwright-core`'s own browser installation.
- **`format.test.js`** — verifies source formatting (see
  [Source formatting](#source-formatting) below). `prettier` is an optional
  dependency, mirroring `playwright-core`: install it with
  `make install-tools`; when it is absent this file is skipped and the rest
  of the suite runs unchanged.

Both optional packages are installed by the single `make install-tools`
target (one npm call — npm prunes packages it was not asked for, so
installing them separately would remove the other).

### Source formatting

Verification:

- **Sources are prettier-formatted** (`format.test.js`) —
  `implementation/src/index.html` and every JavaScript file in
  `verification/` pass a prettier formatting check with default options (one
  test case per file). (The api package's Go sources have the equivalent
  check built into its own verification suite, via `gofmt`.)
