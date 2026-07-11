# `marked` from CDN

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

## Verification

- **Markdown is loaded from a CDN** (`logic.test.js`) — the page loads
  `marked` from the jsDelivr browser bundle. The URL has a three-part
  numeric version and ends in `marked.min.js`. The page does not load
  `marked.min.js` from a local relative URL.
- **The served page references CDN-hosted Markdown support**
  (`contract.test.js`) — the page returned by `GET /` contains the required
  jsDelivr `marked` browser bundle URL with a three-part numeric version.
- **No local copy of `marked` is served** (`contract.test.js`) —
  `GET /marked.min.js` returns 404.
