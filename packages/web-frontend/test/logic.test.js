// Unit tests for the pure, DOM-free helpers embedded in src/index.html.
//
// The helpers live inline in index.html (no separate served JS file, per the
// spec). These tests read the page, extract the inline <script>, and run it in
// a vm sandbox WITHOUT a `document`, so the browser-only DOM wiring is skipped
// and only the pure helpers execute. The helpers are exported via
// module.exports / globalThis.__lanparty by the script itself.
//
// Run: node --test test/

"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const HTML = fs.readFileSync(path.join(__dirname, "..", "src", "index.html"), "utf8");

// Extract the inline (no-src) <script> block from the page.
function inlineScript() {
  const blocks = HTML.match(/<script\b[^>]*>[\s\S]*?<\/script>/g) || [];
  for (const b of blocks) {
    if (!/\bsrc\s*=/.test(b)) {
      return b.replace(/^<script\b[^>]*>/, "").replace(/<\/script>$/, "");
    }
  }
  throw new Error("no inline <script> block found in index.html");
}

// Run the inline script in a fresh sandbox and return the exported helper API.
function loadHelpers() {
  const ctx = {};
  ctx.globalThis = ctx;
  ctx.module = { exports: {} };
  vm.createContext(ctx);
  vm.runInContext(inlineScript(), ctx, { filename: "index.html.inline.js" });
  return ctx.module.exports;
}

const api = loadHelpers();

// ---------------------------------------------------------------------------
// Page structure (read from the static file). Mirrors verifying.md steps 2,9.
// ---------------------------------------------------------------------------

test("index.html loads marked from a CDN, not a local file", () => {
  assert.match(
    HTML,
    /<script\s+src="https:\/\/cdn\.jsdelivr\.net\/npm\/marked@\d+\.\d+\.\d+\/marked\.min\.js"/,
    "page must load marked from the jsdelivr CDN"
  );
  assert.doesNotMatch(
    HTML,
    /<script\s+src="marked\.min\.js"/,
    "page must not reference a local marked.min.js"
  );
});

test("index.html has the required form controls", () => {
  assert.match(HTML, /<form[^>]*id="form"/);
  assert.match(HTML, /<select[^>]*id="type"/);
  assert.match(HTML, /<option\s+value="html"[^>]*selected/);
  assert.match(HTML, /<option\s+value="markdown"/);
  assert.match(HTML, /<textarea[^>]*id="spec"/);
  assert.match(HTML, /<button[^>]*id="submit"[^>]*>Generate<\/button>/);
  assert.match(HTML, /<title>/);
  assert.match(HTML, /<h1>/);
});

// ---------------------------------------------------------------------------
// validateSpec — verifying.md step 6 (empty spec → inline error, no request).
// ---------------------------------------------------------------------------

test("validateSpec rejects blank spec", () => {
  assert.equal(api.validateSpec(""), "Spec is required.");
  assert.equal(api.validateSpec("   "), "Spec is required.");
  assert.equal(api.validateSpec("\t\n "), "Spec is required.");
});

test("validateSpec rejects nullish spec", () => {
  assert.equal(api.validateSpec(null), "Spec is required.");
  assert.equal(api.validateSpec(undefined), "Spec is required.");
});

test("validateSpec accepts non-empty spec", () => {
  assert.equal(api.validateSpec("hello"), null);
  assert.equal(api.validateSpec("  x  "), null);
});

// ---------------------------------------------------------------------------
// extractErrorMessage — verifying.md error contract + api {error:...} shape.
// ---------------------------------------------------------------------------

test("extractErrorMessage uses the api error message when present", () => {
  assert.equal(
    api.extractErrorMessage(422, '{"error":"fields \'type\' and \'spec\' are required"}'),
    "fields 'type' and 'spec' are required"
  );
  assert.equal(api.extractErrorMessage(500, '{"error":"boom"}'), "boom");
});

test("extractErrorMessage falls back to status code for non-JSON or missing error", () => {
  assert.equal(api.extractErrorMessage(502, "garbage"), "Request failed: 502");
  assert.equal(api.extractErrorMessage(502, ""), "Request failed: 502");
  assert.equal(api.extractErrorMessage(400, '{"msg":"x"}'), "Request failed: 400");
  assert.equal(api.extractErrorMessage(500, '{"error": 42}'), "Request failed: 500");
});

// ---------------------------------------------------------------------------
// networkErrorMessage — verifying.md step 7 (failed fetch → Network error).
// ---------------------------------------------------------------------------

test("networkErrorMessage is 'Network error'", () => {
  assert.equal(api.networkErrorMessage(), "Network error");
});

// ---------------------------------------------------------------------------
// blobMimeType
// ---------------------------------------------------------------------------

test("blobMimeType is text/html for both types", () => {
  assert.equal(api.blobMimeType("html"), "text/html");
  assert.equal(api.blobMimeType("markdown"), "text/html");
});

// ---------------------------------------------------------------------------
// buildBlobDocument — verifying.md steps 3 & 5 (same-tab blob navigation).
// ---------------------------------------------------------------------------

test("buildBlobDocument html passes content through unchanged", () => {
  const content = "<!doctype html><html><body><h1>Hi</h1></body></html>";
  assert.equal(api.buildBlobDocument("html", content, () => {
    throw new Error("parse must not be called for html");
  }), content);
});

test("buildBlobDocument markdown wraps rendered output in the spec document", () => {
  const doc = api.buildBlobDocument("markdown", "# Hi", (s) => "<h1>Hi</h1>");
  assert.ok(doc.startsWith("<!doctype html>\n<html>\n  <head>\n"), "starts with the doc preamble");
  assert.ok(doc.includes('    <meta charset="utf-8">\n'), "has the charset meta");
  assert.ok(doc.includes("    <title>representation</title>\n"), "has the title");
  assert.ok(doc.includes("    <style>\n"), "has the style block");
  // The exact style lines required by the spec.
  for (const line of [
    "body { margin: 2rem; font: 1rem/1.5 system-ui, sans-serif; line-height: 1.6; }",
    "pre, code { font-family: ui-monospace, monospace; }",
    "pre { background: #f4f4f4; padding: 1rem; overflow: auto; white-space: pre-wrap; word-wrap: break-word; }",
    "code { background: #f4f4f4; padding: 0 .2em; }",
    "pre code { background: none; padding: 0; }",
    "blockquote { margin: 0 0 1rem; padding: 0 1rem; border-left: .25rem solid #ccc; color: #555; }",
    "h1, h2, h3 { line-height: 1.25; }",
    "img { max-width: 100%; }",
  ]) {
    assert.ok(doc.includes(line), "style block must include: " + line);
  }
  assert.ok(doc.endsWith("</body>\n</html>"), "ends with closing body/html");
  assert.ok(doc.includes("<body><h1>Hi</h1></body>"), "rendered output is placed verbatim in the body");
});

test("buildBlobDocument markdown calls the injected parse with the source", () => {
  let received = null;
  api.buildBlobDocument("markdown", "# T", (s) => { received = s; return "<h1>T</h1>"; });
  assert.equal(received, "# T", "parse is called with the raw markdown content");
});

// ---------------------------------------------------------------------------
// Trust model — verifying.md step 10. `marked` passes raw HTML through by
// default; the page applies NO sanitization, consistent with the html type.
// We model marked's passthrough with identity `parse`.
// ---------------------------------------------------------------------------

test("buildBlobDocument markdown does not sanitize raw HTML (trust model)", () => {
  const md = "# T\n\n<script>alert(1)</script>\n";
  const doc = api.buildBlobDocument("markdown", md, (s) => s); // identity = marked's default passthrough
  assert.ok(doc.includes("<script>alert(1)</script>"),
    "raw HTML in markdown survives into the blob document (no sanitization)");
});

test("html and markdown trust models are consistent (no sanitization in either)", () => {
  const script = "<script>alert(1)</script>";
  assert.equal(api.buildBlobDocument("html", script, () => null), script,
    "html content is rendered as live HTML, unsanitized");
  assert.ok(api.buildBlobDocument("markdown", script, (s) => s).includes(script),
    "markdown with raw HTML is also rendered as live HTML, unsanitized");
});
