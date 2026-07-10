# Verifying

Verification has two layers: an **automated test suite** (encoding the
validations and checks below, including browser checks driven through a
headless Chromium) and a **manual procedure** that doubles as a fallback for
the browser checks when no browser is available.

## Automated tests

Run, from the package's `verification/` directory (requires Node.js 18 or
later and the Go toolchain — Go 1.26, used to compile the api binary the
contract tests run against; no Node package dependencies):

```bash
node --test *.test.js
```

Equivalently, run `make test` from the `verification/` directory.

The suite is split into three files (plus `verification/harness.js`, a shared
non-test helper that builds the api binary and starts the origin/api/fake
OpenRouter stack):

- `verification/logic.test.js` — extracts the pure, DOM-free helpers embedded inline
  in `implementation/src/index.html` (validation, error extraction, the blob-document
  builder, MIME mapping, the network-error message) and asserts their behavior.
  When `module.exports` is available, that inline script exposes these helpers
  as `validateSpec`, `extractErrorMessage`, `networkErrorMessage`,
  `blobMimeType`, and `buildBlobDocument` so the test can invoke them without a
  browser DOM. This export is a test hook; normal browser behavior still uses
  the same helpers through the form's submit listener.
  Covers: empty-spec validation (step 6), the error message contract, the
  `Network error` message (step 7), the blob document contents for `html` and
  `markdown` (steps 3 & 5, at the content level), and the trust model (step 10).
  It also reads `index.html` directly (no HTTP server) to assert the page
  loads `marked` from a CDN and has the required form controls (steps 2 & 9),
  as a quick file-level check that duplicates part of what
  `verification/contract.test.js` verifies over HTTP.
- `verification/contract.test.js` — spins up a Caddyfile-mirroring origin (static
  `src/` at `/`, `/v1/*` proxied unchanged to the REAL api binary, compiled from
  `packages/api/implementation` and configured against a fake OpenRouter
  upstream; see `verification/harness.js`) and asserts the HTTP contract.
  Because the api is the real binary, these tests cannot drift from the api
  spec.
  Covers: the page is served with the required controls (step 2), `marked` is
  loaded from a CDN with no local vendored copy (step 9), method-not-allowed,
  the 422 validations and error shape (step 6), and the 200 html/markdown
  headers plus the exact request body/`Content-Type` (step 8). Rejecting
  non-`POST` methods with `405` has no corresponding manual procedure step
  below; it is only ever verified automatically.

- `verification/browser.test.js` — drives the page in a real (headless)
  Chromium via `playwright-core`, against the same origin/api stack as the
  contract tests. Covers the checks that need a browser: clicking **Generate**
  **navigates the same tab** to a `text/html` blob URL with no new tab or
  popup (steps 3 & 9), the **back button** returns to the form with the
  previous type and spec still entered (step 4), and a `<script>` in rendered
  markdown output actually *executes* in the rendered tab (step 10 — the
  trust model). To keep the suite hermetic, the page's request for the marked
  CDN bundle is intercepted and fulfilled with a passthrough stub
  (`parse` = identity); the test asserts the page requested the required CDN
  URL, and real markdown rendering stays covered by `logic.test.js`.
  `playwright-core` is an optional dependency: install it with
  `make install-browser`; when it is absent this file is skipped and the rest
  of the suite runs unchanged. The Chromium executable is resolved from
  `CHROMIUM_PATH`, then `/opt/pw-browsers/chromium` when present, then
  `playwright-core`'s own browser installation.

### Manual fallback (browser-only)

When `playwright-core` or a Chromium executable is unavailable, verify the
browser checks by hand:

- Step 3 / step 5: clicking **Generate** actually **navigates the same tab** to
  the blob URL (blob navigation, not just the blob's content).
- Step 4: the browser **back button** returns to the form (session-history
  behavior of blob URLs).
- Step 10 runtime: that a `<script>` in generated markdown actually *executes*
  in the rendered tab (the content-level check that it is not stripped is
  automated; execution is browser-only).

## Manual procedure

The frontend is a static page with no build step. Manual verification:

1. Start the api backend (see
   [../../../api/specification/implementation/main.md](../../../api/specification/implementation/main.md)) and put it behind the
   shared reverse proxy (see [../implementation/main.md](../implementation/main.md#deployment)) so that
   `/v1/representation` reaches the api and `/` serves this package, under one
   origin. The bundled `Caddyfile` is the reference configuration for this;
   `caddy run --config Caddyfile` serves the frontend at `http://localhost:3000`
   and forwards `/v1/*` to the api on `localhost:8080`.
2. Open `http://localhost:3000`.
3. Select `html`, type a spec, click **Generate** → the same tab navigates to a
   rendered version of the generated HTML page.
4. Press the browser **back** button → returns to the form, with the previous
   type and spec still entered.
5. Select `markdown`, type a spec, click **Generate** → the same tab navigates
   to a page showing the generated markdown **rendered to HTML** by `marked`
   (headings, lists, code, emphasis, links, etc.).
6. Leave the **Spec** field empty and click **Generate** → an inline error is
   shown on the form page, and no network request is sent.
7. Stop the api backend and click **Generate** → an inline "Network error"
   message is shown and the form remains usable.
8. Inspect the request sent for a successful generation: it is a `POST` to
   `/v1/representation` with `Content-Type: application/json` and a body of the
   form `{"type":"html","spec":"..."}`.
9. Confirm `marked` is loaded from a public CDN (the `<script src>` is a CDN
   URL); the frontend's own origin serves only `index.html`.
10. Markdown raw HTML: request a `markdown` generation whose spec coaxes the LLM
    to include a `<script>` tag in its output. In the rendered tab the script
    runs, consistent with the `html` type (generated content is rendered as
    live HTML, no sanitization). This is the documented trust model.
