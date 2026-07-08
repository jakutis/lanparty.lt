// HTTP contract tests for the web-frontend deployment.
//
// Mirrors the Caddyfile: one origin serves the static frontend (src/) at `/`
// and forwards `/v1/*` unchanged to the api backend. The api backend here is a
// faithful in-process stub implementing the api spec's response contract
// (status codes, Content-Type, Content-Disposition, and the {"error":...}
// shape — see packages/api/specs/main.md) so we can assert the frontend's
// runtime assumptions about it WITHOUT depending on the real LLM.
//
// These tests cover verifying.md steps 1, 2, 4, 6, 8, 9 and the header/error
// contract. Browser-only behaviors (same-tab blob navigation, the back button,
// fetch rejection -> "Network error", and marked rendering) are covered by
// test/logic.test.js or are manual-only.
//
// Run: node --test test/

"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const http = require("node:http");

const SRC_DIR = path.join(__dirname, "..", "src");

// The most recent request received by the api stub (for body-shape checks).
let lastApiRequest = null;

// ---------------------------------------------------------------------------
// Faithful in-process api backend (mirrors packages/api/specs/main.md).
// ---------------------------------------------------------------------------

function writeJsonError(res, code, msg) {
  const body = JSON.stringify({ error: msg });
  res.writeHead(code, { "Content-Type": "application/json; charset=utf-8" });
  res.end(body);
}

function contentTypeFor(typ) {
  switch (String(typ).toLowerCase()) {
    case "html": return ["text/html; charset=utf-8", ".html"];
    case "markdown": return ["text/markdown; charset=utf-8", ".md"];
    default: return null;
  }
}

function apiHandler(req, res) {
  lastApiRequest = { method: req.method, url: req.url, headers: { ...req.headers }, body: "" };

  // The proxy forwards /v1/* unchanged; the api strips the /v1 prefix itself
  // (mirrors the real api's http.StripPrefix("/v1", ...)).
  const path = req.url.replace(/^\/v1/, "");

  // Only POST /representation is allowed; anything else is 405.
  if (req.method !== "POST" || path !== "/representation") {
    res.writeHead(405, { "Content-Type": "text/plain; charset=utf-8" });
    res.end("Method Not Allowed\n");
    return;
  }

  let raw = "";
  req.on("data", (c) => {
    raw += c;
    if (raw.length > 1 << 20) { // 1 MiB limit, like the api
      res.destroy();
      req.destroy();
    }
  });
  req.on("end", () => {
    lastApiRequest.body = raw;

    let parsed;
    try {
      parsed = JSON.parse(raw);
    } catch (_) {
      return writeJsonError(res, 400, "invalid request body: malformed JSON");
    }
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return writeJsonError(res, 400, "invalid request body: not a JSON object");
    }
    // Disallow unknown fields, like the api.
    for (const k of Object.keys(parsed)) {
      if (k !== "type" && k !== "spec") {
        return writeJsonError(res, 400, "invalid request body: unknown field " + JSON.stringify(k));
      }
    }

    const type = String(parsed.type == null ? "" : parsed.type).trim();
    const spec = String(parsed.spec == null ? "" : parsed.spec).trim();
    if (type === "" || spec === "") {
      return writeJsonError(res, 422, "fields 'type' and 'spec' are required");
    }

    const ct = contentTypeFor(type);
    if (!ct) {
      return writeJsonError(res, 422,
        'unsupported type "' + type + '": only "html" and "markdown" are supported');
    }
    const [mimeType, ext] = ct;
    const body = type === "html"
      ? "<!doctype html>\n<html><body><h1>hello</h1></body></html>\n"
      : "# hello\n\n- a\n- b\n";
    res.writeHead(200, {
      "Content-Type": mimeType,
      "Content-Disposition": 'attachment; filename="representation' + ext + '"',
    });
    res.end(body);
  });
}

// ---------------------------------------------------------------------------
// Static file handler (mirrors `file_server` in the Caddyfile).
// ---------------------------------------------------------------------------

function staticHandler(req, res) {
  let p = decodeURIComponent(new URL(req.url, "http://localhost").pathname);
  if (p === "/") p = "/index.html";
  const fp = path.join(SRC_DIR, p);
  if (!path.resolve(fp).startsWith(path.resolve(SRC_DIR) + path.sep)) {
    res.writeHead(403); res.end("Forbidden"); return;
  }
  fs.readFile(fp, (err, data) => {
    if (err) { res.writeHead(404, { "Content-Type": "text/plain" }); res.end("Not Found"); return; }
    const ct = fp.endsWith(".html") ? "text/html; charset=utf-8" : "application/octet-stream";
    res.writeHead(200, { "Content-Type": ct });
    res.end(data);
  });
}

// Combined origin: /v1/* -> api, everything else -> static (like the Caddyfile).
function combinedHandler(req, res) {
  if (req.url.startsWith("/v1/")) return apiHandler(req, res);
  return staticHandler(req, res);
}

let server;
let base;

test("contract", { concurrency: false }, async (t) => {
  server = http.createServer(combinedHandler);
  await new Promise((r) => server.listen(0, "127.0.0.1", r));
  base = `http://127.0.0.1:${server.address().port}`;

  t.after(() => new Promise((r) => server.close(r)));

  await t.test("GET / serves index.html (verifying step 2)", async () => {
    const res = await fetch(base + "/");
    assert.equal(res.status, 200);
    assert.equal(res.headers.get("content-type"), "text/html; charset=utf-8");
    const body = await res.text();
    assert.match(body, /<form[^>]*id="form"/);
    assert.match(body, /<select[^>]*id="type"/);
    assert.match(body, /<option\s+value="html"[^>]*selected/);
    assert.match(body, /<option\s+value="markdown"/);
    assert.match(body, /<textarea[^>]*id="spec"/);
    assert.match(body, /<button[^>]*id="submit"[^>]*>Generate<\/button>/);
  });

  await t.test("GET / references marked from a CDN (verifying step 9)", async () => {
    const body = await (await fetch(base + "/")).text();
    assert.match(body, /<script\s+src="https:\/\/cdn\.jsdelivr\.net\/npm\/marked@\d+\.\d+\.\d+\/marked\.min\.js"/);
  });

  await t.test("GET /marked.min.js is 404 (no local vendored marked)", async () => {
    const res = await fetch(base + "/marked.min.js");
    assert.equal(res.status, 404);
  });

  await t.test("GET /v1/representation is 405 (verifying step 4: method check)", async () => {
    const res = await fetch(base + "/v1/representation", { method: "GET" });
    assert.equal(res.status, 405);
  });

  await t.test("POST /v1/representation with blank spec is 422 + api error shape (step 6)", async () => {
    const res = await fetch(base + "/v1/representation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "html", spec: "   " }),
    });
    assert.equal(res.status, 422);
    assert.equal(res.headers.get("content-type"), "application/json; charset=utf-8");
    const json = await res.json();
    assert.equal(json.error, "fields 'type' and 'spec' are required");
  });

  await t.test("POST /v1/representation with unsupported type is 422 (step 6)", async () => {
    const res = await fetch(base + "/v1/representation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "pdf", spec: "x" }),
    });
    assert.equal(res.status, 422);
    const json = await res.json();
    assert.equal(json.error, 'unsupported type "pdf": only "html" and "markdown" are supported');
  });

  await t.test("POST /v1/representation html -> 200 with the right headers (step 8)", async () => {
    const res = await fetch(base + "/v1/representation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "html", spec: "a hello page" }),
    });
    assert.equal(res.status, 200);
    assert.equal(res.headers.get("content-type"), "text/html; charset=utf-8");
    assert.equal(res.headers.get("content-disposition"), 'attachment; filename="representation.html"');
    const body = await res.text();
    assert.match(body, /<h1>hello<\/h1>/);
  });

  await t.test("POST /v1/representation markdown -> 200 with the right headers (step 8)", async () => {
    const res = await fetch(base + "/v1/representation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "markdown", spec: "a doc" }),
    });
    assert.equal(res.status, 200);
    assert.equal(res.headers.get("content-type"), "text/markdown; charset=utf-8");
    assert.equal(res.headers.get("content-disposition"), 'attachment; filename="representation.md"');
    const body = await res.text();
    assert.match(body, /^# hello/);
  });

  await t.test("the request body and Content-Type match the spec (step 8)", async () => {
    lastApiRequest = null;
    await fetch(base + "/v1/representation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "html", spec: "a hello page" }),
    });
    assert.ok(lastApiRequest, "the api stub received a request");
    // The proxy forwards /v1/* to the api UNCHANGED (the api strips the prefix
    // internally), so the backend sees /v1/representation.
    assert.equal(lastApiRequest.url, "/v1/representation");
    assert.equal(lastApiRequest.headers["content-type"], "application/json");
    assert.deepEqual(JSON.parse(lastApiRequest.body), { type: "html", spec: "a hello page" });
  });
});
