// HTTP contract tests for the web-frontend deployment.
//
// Runs against the REAL api binary behind a Caddyfile-mirroring origin (see
// harness.js): the static frontend at `/` and `/v1/*` forwarded unchanged to
// the api, which is configured against a fake OpenRouter upstream. Because
// the api is the real compiled binary, these tests cannot drift from the api
// spec — its status codes, headers, and error messages are asserted directly.
//
// These tests cover verifying.md steps 1, 2, 4, 6, 8, 9 and the header/error
// contract. Browser-only behaviors (same-tab blob navigation, the back button,
// fetch rejection -> "Network error", and marked rendering) are covered by
// logic.test.js and browser.test.js.
//
// Requires the Go toolchain to build the api binary.
// Run: make test (or node --test *.test.js) from this directory.

"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { startStack } = require("./harness.js");

test("contract", { concurrency: false }, async (t) => {
  const stack = await startStack();
  const base = stack.base;

  t.after(() => stack.close());

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
    await fetch(base + "/v1/representation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "html", spec: "a hello page" }),
    });
    const last = stack.lastApiRequest();
    assert.ok(last, "the api received a request");
    // The origin forwards /v1/* to the api UNCHANGED (the api strips the
    // prefix internally), so the api sees /v1/representation.
    assert.equal(last.url, "/v1/representation");
    assert.equal(last.headers["content-type"], "application/json");
    assert.deepEqual(JSON.parse(last.body), { type: "html", spec: "a hello page" });
  });
});
