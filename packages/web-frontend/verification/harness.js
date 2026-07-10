// Test harness for the web-frontend verification suite.
//
// Starts the full deployment contract described in
// ../specification/implementation/main.md#deployment, using the REAL api
// binary (compiled from ../../api/implementation with the Go toolchain) so
// the suite cannot drift from the api's actual behavior:
//
//   origin (Node http server, mirrors the Caddyfile)
//     /v1/*            -> proxied unchanged to the api binary
//     everything else  -> static files from implementation/src/
//   api binary         -> configured against a fake OpenRouter upstream
//
// The fake OpenRouter upstream picks a canned generation from the user
// prompt: markdown requests get a markdown document, requests whose spec
// contains "with_script" get markdown carrying a <script> element (for the
// trust-model checks), and everything else gets an HTML document.
//
// Not a test file itself; required by contract.test.js and browser.test.js.

"use strict";

const child_process = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const net = require("node:net");
const os = require("node:os");
const path = require("node:path");

const API_IMPL_DIR = path.join(__dirname, "..", "..", "api", "implementation");
const SRC_DIR = path.join(__dirname, "..", "implementation", "src");

function freePort() {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.listen(0, "127.0.0.1", () => {
      const port = srv.address().port;
      srv.close(() => resolve(port));
    });
    srv.on("error", reject);
  });
}

// Build the real api binary into a temp dir and return its path.
function buildApiBinary() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "web-frontend-verify-"));
  const bin = path.join(dir, "api.bin");
  child_process.execFileSync(
    "go",
    ["build", "-C", API_IMPL_DIR, "-o", bin, "./src"],
    {
      stdio: ["ignore", "inherit", "inherit"],
    },
  );
  return bin;
}

// A fake OpenRouter Anthropic Messages endpoint returning canned generations.
function fakeOpenRouterHandler(req, res) {
  let raw = "";
  req.on("data", (c) => {
    raw += c;
  });
  req.on("end", () => {
    let prompt = "";
    try {
      const body = JSON.parse(raw);
      for (const msg of body.messages || []) {
        if (typeof msg.content === "string") prompt += msg.content + "\n";
      }
    } catch (_) {}

    const reply = (text) => {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          content: [{ type: "text", text }],
          stop_reason: "end_turn",
        }),
      );
    };

    if (prompt.includes("fail_generation")) {
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "upstream" }));
    } else if (prompt.includes("with_script")) {
      reply("# hello\n\n<script>window.__pwned = true</script>\n");
    } else if (prompt.includes("Generate a markdown file")) {
      reply("# hello\n\n- a\n- b\n");
    } else {
      reply("<!doctype html>\n<html><body><h1>hello</h1></body></html>\n");
    }
  });
}

// Static file handler (mirrors `file_server` in the Caddyfile).
function staticHandler(req, res) {
  let p = decodeURIComponent(new URL(req.url, "http://localhost").pathname);
  if (p === "/") p = "/index.html";
  const fp = path.join(SRC_DIR, p);
  if (!path.resolve(fp).startsWith(path.resolve(SRC_DIR) + path.sep)) {
    res.writeHead(403);
    res.end("Forbidden");
    return;
  }
  fs.readFile(fp, (err, data) => {
    if (err) {
      res.writeHead(404, { "Content-Type": "text/plain" });
      res.end("Not Found");
      return;
    }
    const ct = fp.endsWith(".html")
      ? "text/html; charset=utf-8"
      : "application/octet-stream";
    res.writeHead(200, { "Content-Type": ct });
    res.end(data);
  });
}

// Start the whole stack. Returns { base, lastApiRequest(), close() } where
// base is the shared-origin URL and lastApiRequest() is the most recent
// request the origin forwarded to the api (method, url, headers, body).
async function startStack() {
  const bin = buildApiBinary();

  const fake = http.createServer(fakeOpenRouterHandler);
  await new Promise((r) => fake.listen(0, "127.0.0.1", r));
  const fakeUrl = `http://127.0.0.1:${fake.address().port}`;

  const apiPort = await freePort();
  const api = child_process.spawn(bin, {
    env: {
      ...process.env,
      PORT: String(apiPort),
      OPENROUTER_API_KEY: "test-key",
      OPENROUTER_MODEL: "test/model",
      OPENROUTER_BASE_URL: fakeUrl,
    },
    stdio: ["ignore", "ignore", "ignore"],
  });

  // Wait for the api to accept requests (a GET is answered with 405).
  const deadline = Date.now() + 15000;
  for (;;) {
    try {
      const res = await fetch(`http://127.0.0.1:${apiPort}/v1/representation`);
      if (res.status === 405) break;
    } catch (_) {}
    if (Date.now() > deadline)
      throw new Error("api binary did not start in time");
    await new Promise((r) => setTimeout(r, 50));
  }

  // Combined origin: /v1* -> api (recorded and forwarded unchanged),
  // everything else -> static files. Mirrors the Caddyfile.
  let last = null;
  const origin = http.createServer((req, res) => {
    if (req.url === "/v1" || req.url.startsWith("/v1/")) {
      const record = {
        method: req.method,
        url: req.url,
        headers: { ...req.headers },
        body: "",
      };
      const proxied = http.request(
        {
          host: "127.0.0.1",
          port: apiPort,
          method: req.method,
          path: req.url,
          headers: req.headers,
        },
        (apiRes) => {
          res.writeHead(apiRes.statusCode, apiRes.headers);
          apiRes.pipe(res);
        },
      );
      req.on("data", (c) => {
        record.body += c;
        proxied.write(c);
      });
      req.on("end", () => {
        last = record;
        proxied.end();
      });
      proxied.on("error", () => {
        res.writeHead(502);
        res.end("Bad Gateway");
      });
      return;
    }
    staticHandler(req, res);
  });
  await new Promise((r) => origin.listen(0, "127.0.0.1", r));
  const base = `http://127.0.0.1:${origin.address().port}`;

  return {
    base,
    lastApiRequest: () => last,
    close: async () => {
      await new Promise((r) => origin.close(r));
      api.kill();
      await new Promise((r) => fake.close(r));
      fs.rmSync(path.dirname(bin), { recursive: true, force: true });
    },
  };
}

module.exports = { startStack };
