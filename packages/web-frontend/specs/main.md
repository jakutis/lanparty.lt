# web-frontend package

A static web page that lets a user request a generated file from the
[api](../../api) backend and immediately view the result in the **same browser
tab**.

There is no build step and no bundler. The frontend is plain HTML, CSS and
JavaScript. The only third-party dependency is
[marked](https://marked.js.org/) (a Markdown-to-HTML renderer), loaded from a
public CDN via a `<script>` tag. The library is not vendored locally, and there
is no npm, no bundler, and no build step.

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

This package ships a **reference Caddyfile** (`Caddyfile`, at the package root)
that implements this routing and listens on `:3000`, forwarding `/v1/*`
unchanged to the api backend at `localhost:8080` and serving `src/` for
everything else. The `/v1` prefix is passed through to the api unchanged,
because the api itself strips that prefix internally. Usage from the package
directory:

```bash
PORT=8080 OPENROUTER_API_KEY=... OPENROUTER_MODEL=... go -C ../api run ./src &
caddy run --config Caddyfile
```

Then open `http://localhost:3000`. The Caddyfile is the canonical example; any
equivalent proxy configuration satisfies the deployment contract.

Cross-origin use (the page and the api on different origins) is not supported
by this package.

## Running

Serve this package's directory with any static file server, behind the shared
reverse proxy described above, then open the page in a browser. The bundled
`Caddyfile` is the recommended way to run it.

## Verifying

See [./main/verifying.md](./main/verifying.md).

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
label. The page has a clear title and heading. The page loads `marked` from a
public CDN via a `<script>` tag so that the global `marked` is available before
any generation can happen. (The CDN is reachable only from the user's browser;
the frontend's own origin serves only `index.html`.)

## Submit flow

On the form `submit` event:

1. The default form submission is prevented (`preventDefault`).
2. The current `type` and `spec` are read and trimmed of leading/trailing
   whitespace.
3. **Client-side validation:** if `spec` is empty after trimming, the
   submission is aborted and an inline error message is shown on the page; no
   request is sent. (The api would reject it anyway; validating client-side
   avoids a round trip.)
4. The Submit button is disabled and a "Generating…" loading indicator is shown
   inline. The form controls remain visible and unmodified.
5. A `POST` request is sent to `/v1/representation` (relative URL) with:
   - `Content-Type: application/json`
   - body `{"type":"<type>","spec":"<spec>"}` (JSON-encoded; `<type>` and
     `<spec>` are the trimmed values)
   - `credentials: 'omit'` (no cookies are sent)
   - `cache: 'no-store'`
6. The request awaits the response.

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
previous one.

## Markdown rendering

The `markdown` type uses `marked` to render the generated markdown to HTML. Rendering is a single call: `marked.parse(content)`, using
`marked`'s default options. The rendered HTML is placed verbatim into the blob
document's `<body>` (see the template above).

### Trust model

The generated content is **untrusted** (LLM-produced, may contain anything).
This is consistent with the `html` type: in both cases the generated content is
rendered as live HTML in the blob tab. `marked` passes raw HTML in the markdown
source through to the output by default, so a generated markdown document that
contains HTML (including `<script>`) will have that HTML execute in the rendered
tab — exactly as a generated `html` document would. No sanitization is applied.
This is intentional: the page faithfully opens the representation the user
requested.

## `marked` from CDN

`marked` is loaded from a public CDN via a `<script>` tag in `index.html`, using
the CDN's prebuilt browser/UMD bundle, which exposes a global `marked` with a
`parse(string) => string` method. The library is not vendored locally; the
frontend's own origin serves only `index.html`, and the CDN is reached from the
user's browser. The spec does not pin a specific minor version; any recent
stable release of `marked` whose browser bundle exposes a global `marked` with
a `parse(string) => string` method satisfies this requirement. (The
implementation pins a known-good version URL for reproducibility.)
