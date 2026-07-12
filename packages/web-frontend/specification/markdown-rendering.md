# Markdown rendering

The `markdown` type uses `marked` to render the generated markdown to HTML. Rendering is a single call: `marked.parse(content)`, using
`marked`'s default options. The rendered HTML is placed verbatim into the blob
document's `<body>` (see the document template under the
[Submit flow's Success section](submit-flow.md#success)).

## Verification

- **Markdown is rendered from the unmodified generated content**
  (`logic.test.js`) — the Markdown renderer receives the generated Markdown
  content exactly as returned by the API.

## Trust model

The generated content is **untrusted** (LLM-produced, may contain anything).
This is consistent with the `html` type: in both cases the generated content is
rendered as live HTML in the blob tab. `marked` passes raw HTML in the markdown
source through to the output by default, so a generated markdown document that
contains HTML (including `<script>`) will have that HTML execute in the rendered
tab — exactly as a generated `html` document would. No sanitization is applied.
This is intentional: the page faithfully opens the representation the user
requested.

### Verification

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
