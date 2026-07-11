# Submit flow

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

## Verification

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

## Success

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

### Verification

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

## Errors

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

### Verification

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
