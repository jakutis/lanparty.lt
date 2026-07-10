# Automated test cases

This document specifies the behavior covered by every automated test in
`verification/`. It does not prescribe how that behavior is tested.

See [main.md](../implementation/main.md) for the package requirements and
[verifying.md](./verifying.md) for how to run the suite and for
browser-only checks.

Run the automated suite from `verification/`:

```bash
node --test *.test.js
```

`make test` runs the same command. The suite requires Node.js 18 or later and
the Go toolchain (Go 1.26): the contract and browser tests compile and run the
real api binary. It has no required Node package dependencies; the browser
cases additionally use `playwright-core` (installed with
`make install-browser`) and a Chromium executable, and are skipped when
`playwright-core` is not installed.

Each numbered item below corresponds to one test case. Cases are grouped by
their current test file only to make them easy to locate.

## `verification/logic.test.js`

### 1. Markdown is loaded from a CDN

The page loads `marked` from the jsDelivr browser bundle. The URL has a
three-part numeric version and ends in `marked.min.js`. The page does not load
`marked.min.js` from a local relative URL.

### 2. Required page controls are present

The page has:

- a title and a heading;
- a form with ID `form`;
- a type select with ID `type`;
- selected `html` and available `markdown` options, with those lowercase
  values;
- a Spec textarea with ID `spec`; and
- a Generate button with ID `submit`.

### 3. A blank Spec is rejected

An empty Spec, or one containing only whitespace, is invalid. The validation
message is exactly `Spec is required.`.

### 4. A missing Spec is rejected

An absent Spec value is treated as empty and produces exactly
`Spec is required.`.

### 5. A non-empty Spec is accepted

A Spec containing non-whitespace text is valid, including when it has leading
or trailing whitespace.

### 6. An API error message is shown when available

For a non-200 response whose JSON entity has a string `error` member, the
message shown is that member's value. This includes the API's 422 required
fields error and a generic server error such as `boom`.

### 7. Invalid API error entities use the fallback message

For a non-200 response with an empty, non-JSON, missing-error, or non-string
error entity, the message is exactly `Request failed: <status>`, using that
response's status code.

### 8. A failed request reports a network error

When no response is received because the request fails, the message is exactly
`Network error`.

### 9. Successful results use an HTML blob

Both `html` and `markdown` generation results are opened from a blob whose MIME
type is `text/html`.

### 10. HTML output is opened unchanged

For an `html` generation result, the blob content is exactly the generated
content. Markdown rendering is not performed.

### 11. Markdown output is wrapped in the required document

For a `markdown` generation result, the Markdown renderer's HTML is placed
verbatim in the body of an HTML document. That document has:

- the required doctype, HTML structure, UTF-8 meta tag, and title
  `representation`;
- a style element; and
- every Markdown-result style declaration specified in
  [main.md](../implementation/main.md#success), for body text, code and preformatted text,
  blockquotes, headings, and images.

The document ends with its closing body and HTML tags.

### 12. Markdown is rendered from the unmodified generated content

The Markdown renderer receives the generated Markdown content exactly as
returned by the API.

### 13. Raw HTML in Markdown is retained

Markdown output containing raw HTML, including a script element, is not
sanitized before it is put in the result document. This check verifies retained
content; executing the script requires a browser and is not automated here.

### 14. HTML and Markdown have the same trust model

A script element is retained both in direct HTML output and in Markdown output
that renders it as HTML. Neither output path sanitizes generated content.

## `verification/contract.test.js`

### 1. The shared origin serves the form page

`GET /` returns 200 with `Content-Type: text/html; charset=utf-8`. The served
page includes the required form, type options, Spec textarea, and Generate
button.

### 2. The served page references CDN-hosted Markdown support

The page returned by `GET /` contains the required jsDelivr `marked` browser
bundle URL with a three-part numeric version.

### 3. No local copy of `marked` is served

`GET /marked.min.js` returns 404.

### 4. The representation endpoint rejects GET

`GET /v1/representation` returns 405.

### 5. A blank API Spec receives the API validation response

A JSON `POST /v1/representation` request with type `html` and a whitespace-only
Spec returns:

- status 422;
- `Content-Type: application/json; charset=utf-8`; and
- JSON with `error` exactly equal to
  `fields 'type' and 'spec' are required`.

### 6. An unsupported API type receives the API validation response

A JSON `POST /v1/representation` request with type `pdf` and a non-empty Spec
returns status 422. Its JSON `error` is exactly:

```text
unsupported type "pdf": only "html" and "markdown" are supported
```

### 7. An HTML representation has the expected HTTP response

A JSON request for an HTML representation returns:

- status 200;
- `Content-Type: text/html; charset=utf-8`;
- `Content-Disposition: attachment; filename="representation.html"`; and
- generated content containing `<h1>hello</h1>`.

### 8. A Markdown representation has the expected HTTP response

A JSON request for a Markdown representation returns:

- status 200;
- `Content-Type: text/markdown; charset=utf-8`;
- `Content-Disposition: attachment; filename="representation.md"`; and
- generated content beginning with `# hello`.

### 9. The representation request uses the required route and JSON data

A successful HTML representation request reaches
`/v1/representation` without removing the `/v1` prefix. It has request header
`Content-Type: application/json` and a JSON body exactly equivalent to:

```json
{"type":"html","spec":"a hello page"}
```

## `verification/browser.test.js`

These cases run in a real (headless) Chromium and are skipped when
`playwright-core` is not installed (see
[verifying.md](./verifying.md#automated-tests)). The page's request for the
marked CDN bundle is intercepted and answered with a passthrough stub, and the
test asserts the requested URL is the required CDN bundle URL.

### 1. Generate navigates the same tab to an HTML blob

Submitting the form with type `html` and a non-empty Spec navigates the
current tab to a `blob:` URL, without opening a new tab or popup, and the
generated HTML is rendered there.

### 2. The back button returns to the retained form

After a successful generation, the browser back button returns to the form
page, with the previously entered type and Spec still present.

### 3. A script in rendered Markdown output executes

A `markdown` generation whose output carries a raw `<script>` element renders
with that script *executed* in the result tab — the runtime half of the trust
model; the content-level half is covered by `logic.test.js`.

## Browser-only behavior

When `playwright-core` is unavailable, the `browser.test.js` cases above fall
back to the manual checks in [verifying.md](./verifying.md): same-tab blob
navigation, using the back button to return to the retained form, and actual
execution of a raw HTML script in rendered Markdown output.
