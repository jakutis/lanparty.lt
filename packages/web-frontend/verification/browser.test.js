// Browser tests for the behaviors that need a real browser: same-tab blob
// navigation, back-button history, and script execution in rendered markdown
// (the trust model). These automate the checks listed as browser-only in
// ../specification/main.md#manual-fallback-for-the-browser-checks.
//
// Requires playwright-core (run `make install-tools`) and a Chromium
// executable: set CHROMIUM_PATH, or /opt/pw-browsers/chromium is used when
// present, otherwise playwright-core resolves its own browser install. When
// playwright-core is not installed the whole file is skipped, so the base
// `make test` run stays dependency-free.
//
// The page's request for the marked CDN bundle is intercepted and fulfilled
// with a passthrough stub (parse = identity), keeping the suite hermetic; the
// test asserts the page requested the required CDN URL. Real markdown
// rendering is covered by logic.test.js; script *execution* in the rendered
// output is what needs a browser and is verified here.
//
// The stack behind the page is the same as contract.test.js: the real api
// binary behind a Caddyfile-mirroring origin (see harness.js).

"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");

let chromium = null;
try {
  ({ chromium } = require("playwright-core"));
} catch (_) {}

const skip = chromium
  ? false
  : "playwright-core is not installed (run `make install-tools`)";

function chromiumExecutablePath() {
  if (process.env.CHROMIUM_PATH) return process.env.CHROMIUM_PATH;
  const preinstalled = "/opt/pw-browsers/chromium";
  if (fs.existsSync(preinstalled)) return preinstalled;
  return undefined; // let playwright-core resolve its own browser install
}

const CDN_PATTERN =
  /^https:\/\/cdn\.jsdelivr\.net\/npm\/marked@\d+\.\d+\.\d+\/marked\.min\.js$/;
const MARKED_STUB = "window.marked = { parse: function (s) { return s; } };";

test("browser", { skip, concurrency: false }, async (t) => {
  const { startStack } = require("./harness.js");
  const stack = await startStack();
  const browser = await chromium.launch({
    executablePath: chromiumExecutablePath(),
  });
  const context = await browser.newContext();
  const page = await context.newPage();

  t.after(async () => {
    await browser.close();
    await stack.close();
  });

  // Serve the marked stub for the CDN request, recording the requested URL.
  let cdnUrl = null;
  await page.route("https://cdn.jsdelivr.net/**", (route) => {
    cdnUrl = route.request().url();
    route.fulfill({ contentType: "text/javascript", body: MARKED_STUB });
  });

  await t.test(
    "clicking Generate navigates the same tab to an HTML blob URL (steps 3, 9)",
    async () => {
      await page.goto(stack.base + "/");
      assert.match(
        cdnUrl || "",
        CDN_PATTERN,
        "the page requested marked from the required CDN URL",
      );

      await page.fill("#spec", "a hello page");
      await page.click("#submit");
      await page.waitForURL((u) => String(u).startsWith("blob:"));

      assert.ok(
        page.url().startsWith("blob:"),
        "the tab navigated to a blob URL",
      );
      assert.equal(
        context.pages().length,
        1,
        "no new tab or popup was created",
      );
      assert.equal(
        await page.textContent("h1"),
        "hello",
        "the generated HTML is rendered",
      );
    },
  );

  await t.test(
    "the back button returns to the form with inputs retained (step 4)",
    async () => {
      await page.goBack();
      assert.equal(
        page.url(),
        stack.base + "/",
        "back returns to the form page",
      );
      assert.ok(await page.isVisible("#form"), "the form is shown again");
      assert.equal(
        await page.inputValue("#spec"),
        "a hello page",
        "the previous spec is still entered",
      );
      assert.equal(
        await page.inputValue("#type"),
        "html",
        "the previous type is still selected",
      );
    },
  );

  await t.test(
    "a script in rendered markdown executes (step 10, trust model)",
    async () => {
      await page.selectOption("#type", "markdown");
      await page.fill("#spec", "with_script");
      await page.click("#submit");
      await page.waitForURL((u) => String(u).startsWith("blob:"));

      // The fake upstream returned markdown containing
      // <script>window.__pwned = true</script>; with no sanitization it must
      // have executed in the rendered tab, exactly as a generated html page
      // would.
      assert.equal(
        await page.evaluate(() => window.__pwned),
        true,
        "the script embedded in the markdown output ran",
      );
    },
  );
});
