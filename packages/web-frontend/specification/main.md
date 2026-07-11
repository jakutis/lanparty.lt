# web-frontend package

A static web page that lets a user request a generated file from the
[api](../../api) backend and immediately view the result in the **same browser
tab**.

There is no build step and no bundler. The frontend is plain HTML, CSS and
JavaScript. The only third-party dependency is
[marked](https://marked.js.org/) (a Markdown-to-HTML renderer), loaded from a
public CDN via a `<script>` tag. The library is not vendored locally, and there
is no npm, no bundler, and no build step.

Verification has two layers: an **automated test suite** in `verification/`
(four Node.js test files, including browser checks driven through a headless
Chromium) and a **manual procedure** (see
[Manual verification](#manual-verification)) that doubles as a fallback for
the browser checks when no browser is available. Each section below ends
with a **Verification** subsection enumerating the automated test cases that
cover its behavior, each tagged with the test file it lives in; the cases
specify the behavior covered by every test, not how that behavior is tested.
The suite's architecture and run instructions are described under
[Running the tests](#running-the-tests).

## Deployment

The page talks to the api at the relative URL `/v1/representation`. The api
does not send CORS headers, so the page and the api MUST share a single origin.
This is achieved with a **shared reverse proxy**: one origin serves both the
static frontend and the api backend, forwarding requests by path:

| Path            | Routed to                                           |
| --------------- | --------------------------------------------------- |
| `/v1/...`       | the api backend (e.g. `localhost:8080`)             |
| everything else | the static files of this package (e.g. `index.html`)|

For example, a minimal proxy (any HTTP reverse proxy will do — nginx, Caddy,
`httpd`, etc.): serve this package's files at `/` and proxy `/v1/*` to the api.

This package ships a **reference Caddyfile** (`Caddyfile`, in the `implementation/` directory)
that implements this routing and listens on `:3000`, forwarding `/v1/*`
unchanged to the api backend at `localhost:8080` and serving `src/` for
everything else. The `/v1` prefix is passed through to the api unchanged,
because the api itself strips that prefix internally. Usage from the
`implementation/` directory:

```bash
PORT=8080 OPENROUTER_API_KEY=... OPENROUTER_MODEL=... go -C ../../api/implementation run ./src &
caddy run --config Caddyfile
```

Then open `http://localhost:3000`. The Caddyfile is the canonical example; any
equivalent proxy configuration satisfies the deployment contract.

Cross-origin use (the page and the api on different origins) is not supported
by this package.

### Verification

- **The shared origin serves the form page** (`contract.test.js`) — `GET /`
  returns 200 with `Content-Type: text/html; charset=utf-8`. The served page
  includes the required form, type options, Spec textarea, and Generate
  button.
- **The representation endpoint rejects GET** (`contract.test.js`) —
  `GET /v1/representation` returns 405. Rejecting non-`POST` methods has no
  corresponding step in the [manual procedure](#manual-verification); it is
  only ever verified automatically.

## Running

Serve this package's directory with any static file server, behind the shared
reverse proxy described above, then open the page in a browser. The bundled
`Caddyfile` is the recommended way to run it.

The package also provides these `make` targets:

- `make run`, from the `implementation/` directory, checks that
  `OPENROUTER_API_KEY` and `OPENROUTER_MODEL` are non-empty, starts the api
  backend on port `8080`, and starts Caddy with the bundled `Caddyfile`.
  Stopping the command also stops the backend it started.
- `make test`, from the `verification/` directory, runs the automated Node.js
  test suite (see [Running the tests](#running-the-tests)).

### Running the tests

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

#### Source formatting

Verification:

- **Sources are prettier-formatted** (`format.test.js`) —
  `implementation/src/index.html` and every JavaScript file in
  `verification/` pass a prettier formatting check with default options (one
  test case per file). (The api package's Go sources have the equivalent
  check built into its own verification suite, via `gofmt`.)

## Page

A single page, `index.html`, containing a form with exactly these controls:

| Control | Element | Purpose |
| ------- | ----------------------------------------------- | ------------------------------------------------- |
| Type | `<select>` with two `<option>`s: `html`, `markdown` | The `type` to request. `html` is selected by default. |
| Spec | `<textarea>` | The natural-language specification. |
| Submit | `<button>` (label "Generate") | Submits the form. |

The `value` of each type option is the lowercase type string sent to the api
(`html` / `markdown`).

The form is labelled and laid out so each control is visibly associated with its
label. Its integration hooks are stable: the form and controls have the IDs
`form`, `type`, `spec`, and `submit` respectively, and the Type and Spec labels
use `for="type"` and `for="spec"`. The page has a clear title and heading. The
page loads `marked` from a public CDN via a `<script>` tag so that the global
`marked` is available before any generation can happen. (The CDN is reachable
only from the user's browser; the frontend's own origin serves only
`index.html`.)

### Verification

- **Required page controls are present** (`logic.test.js`) — the page has:
  - a title and a heading;
  - a form with ID `form`;
  - a type select with ID `type`;
  - selected `html` and available `markdown` options, with those lowercase
    values;
  - a Spec textarea with ID `spec`; and
  - a Generate button with ID `submit`.

## Submit flow

On the form `submit` event:

1. The default form submission is prevented (`preventDefault`).
2. The current `type` and `spec` are read and trimmed of leading/trailing
   whitespace.
3. **Client-side validation:** if `spec` is empty after trimming, the
   submission is aborted and an inline error message is shown on the page; no
   request is sent. The message is `Spec is required.` (The api would reject
   it anyway; validating client-side avoids a round trip.)
4. The Submit button is disabled and a "Generating…" loading indicator is shown
   inline. The form controls remain visible and unmodified. The indicator has
   ID `status`, uses `role="status"` with `aria-live="polite"`, and is hidden
   when no request is in progress.
5. A `POST` request is sent to `/v1/representation` (relative URL) with:
   - `Content-Type: application/json`
   - body `{"type":"<type>","spec":"<spec>"}` (JSON-encoded; `<type>` and
     `<spec>` are the trimmed values)
   - `credentials: 'omit'` (no cookies are sent)
   - `cache: 'no-store'`
6. The request awaits the response.

### Verification

- **A blank Spec is rejected** (`logic.test.js`) — an empty Spec, or one
  containing only whitespace, is invalid. The validation message is exactly
  `Spec is required.`.
- **A missing Spec is rejected** (`logic.test.js`) — an absent Spec value is
  treated as empty and produces exactly `Spec is required.`.
- **A non-empty Spec is accepted** (`logic.test.js`) — a Spec containing
  non-whitespace text is valid, including when it has leading or trailing
  whitespace.
- **The representation request uses the required route and JSON data**
  (`contract.test.js`) — a successful HTML representation request reaches
  `/v1/representation` without removing the `/v1` prefix. It has request
  header `Content-Type: application/json` and a JSON body exactly equivalent
  to:

  ```json
  {"type":"html","spec":"a hello page"}
  ```

### Success

When the response status is `200`, the response body is read as text (the
generated file content). The generated content is **immediately opened in the
same browser tab** by navigating the current tab to a blob URL built from the
content. No new tab or popup is created.

- **`html`** — a blob is created with MIME type `text/html` from the generated
  content, and the tab navigates to its object URL. The browser renders the
  generated HTML page in place.
  (`new Blob([content], {type:"text/html"})`, then
  `window.location.href = URL.createObjectURL(blob)`.)

- **`markdown`** — the markdown is **rendered to HTML with `marked`** and the
  result is displayed in the same tab. A minimal HTML document is built that
  presents the rendered markdown:

  ```html
  <!doctype html>
  <html>
    <head>
      <meta charset="utf-8">
      <title>representation</title>
      <style>
        body { margin: 2rem; font: 1rem/1.5 system-ui, sans-serif; line-height: 1.6; }
        pre, code { font-family: ui-monospace, monospace; }
        pre { background: #f4f4f4; padding: 1rem; overflow: auto; white-space: pre-wrap; word-wrap: break-word; }
        code { background: #f4f4f4; padding: 0 .2em; }
        pre code { background: none; padding: 0; }
        blockquote { margin: 0 0 1rem; padding: 0 1rem; border-left: .25rem solid #ccc; color: #555; }
        h1, h2, h3 { line-height: 1.25; }
        img { max-width: 100%; }
      </style>
    </head>
    <body>RENDERED_HTML</body>
  </html>
  ```

  where `RENDERED_HTML` is `marked.parse(content)` — the generated markdown
  rendered to HTML by `marked` using its default options. A blob of MIME type
  `text/html` is created from that document and the tab navigates to its object
  URL.

In both cases the navigation creates a new history entry, so the browser
**back button** returns to the form page. The object URLs created are
intentionally not revoked, so they remain valid for the tab's session history.

#### Verification

- **An HTML representation has the expected HTTP response**
  (`contract.test.js`) — a JSON request for an HTML representation returns:
  - status 200;
  - `Content-Type: text/html; charset=utf-8`;
  - `Content-Disposition: attachment; filename="representation.html"`; and
  - generated content containing `<h1>hello</h1>`.
- **A Markdown representation has the expected HTTP response**
  (`contract.test.js`) — a JSON request for a Markdown representation
  returns:
  - status 200;
  - `Content-Type: text/markdown; charset=utf-8`;
  - `Content-Disposition: attachment; filename="representation.md"`; and
  - generated content beginning with `# hello`.
- **Successful results use an HTML blob** (`logic.test.js`) — both `html` and
  `markdown` generation results are opened from a blob whose MIME type is
  `text/html`.
- **HTML output is opened unchanged** (`logic.test.js`) — for an `html`
  generation result, the blob content is exactly the generated content.
  Markdown rendering is not performed.
- **Markdown output is wrapped in the required document** (`logic.test.js`)
  — for a `markdown` generation result, the Markdown renderer's HTML is
  placed verbatim in the body of an HTML document. That document has:
  - the required doctype, HTML structure, UTF-8 meta tag, and title
    `representation`;
  - a style element; and
  - every Markdown-result style declaration specified above, for body text,
    code and preformatted text, blockquotes, headings, and images.

  The document ends with its closing body and HTML tags.
- **Generate navigates the same tab to an HTML blob** (`browser.test.js`) —
  submitting the form with type `html` and a non-empty Spec navigates the
  current tab to a `blob:` URL, without opening a new tab or popup, and the
  generated HTML is rendered there.
- **The back button returns to the retained form** (`browser.test.js`) —
  after a successful generation, the browser back button returns to the form
  page, with the previously entered type and Spec still present.

### Errors

When the response status is not `200`, or the `fetch` itself fails (network
error, blocked by CORS, etc.):

- The loading state is cleared and the Submit button is re-enabled.
- The form remains on the page so the user can adjust inputs and retry.
- An inline error message is shown below the form (not via `alert`):
  - For a non-200 response whose entity is JSON of the shape
    `{"error":"<message>"}` (the api's error shape), the message from `error`
    is shown.
  - If the entity cannot be parsed as that shape, a generic message is shown
    that includes the HTTP status code (e.g. "Request failed: 502").
  - For a failed `fetch` (no response at all), the message is "Network error".

Only one error message is shown at a time; showing a new one replaces any
previous one. The message has ID `error`, uses `role="alert"`, and is hidden
when no error is shown.

#### Verification

- **An API error message is shown when available** (`logic.test.js`) — for a
  non-200 response whose JSON entity has a string `error` member, the
  message shown is that member's value. This includes the API's 422 required
  fields error and a generic server error such as `boom`.
- **Invalid API error entities use the fallback message** (`logic.test.js`)
  — for a non-200 response with an empty, non-JSON, missing-error, or
  non-string error entity, the message is exactly `Request failed: <status>`,
  using that response's status code.
- **A failed request reports a network error** (`logic.test.js`) — when no
  response is received because the request fails, the message is exactly
  `Network error`.
- **A blank API Spec receives the API validation response**
  (`contract.test.js`) — a JSON `POST /v1/representation` request with type
  `html` and a whitespace-only Spec returns:
  - status 422;
  - `Content-Type: application/json; charset=utf-8`; and
  - JSON with `error` exactly equal to
    `fields 'type' and 'spec' are required`.
- **An unsupported API type receives the API validation response**
  (`contract.test.js`) — a JSON `POST /v1/representation` request with type
  `pdf` and a non-empty Spec returns status 422. Its JSON `error` is exactly:

  ```text
  unsupported type "pdf": only "html" and "markdown" are supported
  ```

## Markdown rendering

The `markdown` type uses `marked` to render the generated markdown to HTML. Rendering is a single call: `marked.parse(content)`, using
`marked`'s default options. The rendered HTML is placed verbatim into the blob
document's `<body>` (see the template above).

### Verification

- **Markdown is rendered from the unmodified generated content**
  (`logic.test.js`) — the Markdown renderer receives the generated Markdown
  content exactly as returned by the API.

### Trust model

The generated content is **untrusted** (LLM-produced, may contain anything).
This is consistent with the `html` type: in both cases the generated content is
rendered as live HTML in the blob tab. `marked` passes raw HTML in the markdown
source through to the output by default, so a generated markdown document that
contains HTML (including `<script>`) will have that HTML execute in the rendered
tab — exactly as a generated `html` document would. No sanitization is applied.
This is intentional: the page faithfully opens the representation the user
requested.

#### Verification

- **Raw HTML in Markdown is retained** (`logic.test.js`) — Markdown output
  containing raw HTML, including a script element, is not sanitized before
  it is put in the result document. This check verifies retained content;
  executing the script requires a browser and is not automated here.
- **HTML and Markdown have the same trust model** (`logic.test.js`) — a
  script element is retained both in direct HTML output and in Markdown
  output that renders it as HTML. Neither output path sanitizes generated
  content.
- **A script in rendered Markdown output executes** (`browser.test.js`) — a
  `markdown` generation whose output carries a raw `<script>` element renders
  with that script *executed* in the result tab — the runtime half of the
  trust model; the content-level half is covered by `logic.test.js`.

## `marked` from CDN

`marked` is loaded from jsDelivr via a `<script>` tag in `index.html`, using the
CDN's prebuilt browser/UMD bundle at
`https://cdn.jsdelivr.net/npm/marked@<version>/marked.min.js`, where
`<version>` is a three-part numeric release. It exposes a global `marked` with
a `parse(string) => string` method. The library is not vendored locally; the
frontend's own origin serves only `index.html`, and jsDelivr is reached from the
user's browser. The spec does not pin a specific minor version; any recent
stable release at that URL shape whose browser bundle exposes the required
global satisfies this requirement. (The implementation pins a known-good
version URL for reproducibility.)

### Verification

- **Markdown is loaded from a CDN** (`logic.test.js`) — the page loads
  `marked` from the jsDelivr browser bundle. The URL has a three-part
  numeric version and ends in `marked.min.js`. The page does not load
  `marked.min.js` from a local relative URL.
- **The served page references CDN-hosted Markdown support**
  (`contract.test.js`) — the page returned by `GET /` contains the required
  jsDelivr `marked` browser bundle URL with a three-part numeric version.
- **No local copy of `marked` is served** (`contract.test.js`) —
  `GET /marked.min.js` returns 404.

## Manual verification

The frontend is a static page with no build step. Manual verification:

1. Start the api backend (see
   [../../api/specification/main.md](../../api/specification/main.md)) and put it behind the
   shared reverse proxy (see [Deployment](#deployment)) so that
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

### Manual fallback for the browser checks

When `playwright-core` or a Chromium executable is unavailable, verify the
browser checks by hand:

- Step 3 / step 5: clicking **Generate** actually **navigates the same tab** to
  the blob URL (blob navigation, not just the blob's content).
- Step 4: the browser **back button** returns to the form (session-history
  behavior of blob URLs).
- Step 10 runtime: that a `<script>` in generated markdown actually *executes*
  in the rendered tab (the content-level check that it is not stripped is
  automated; execution is browser-only).
