// Source formatting checks for the web-frontend package: the page
// (implementation/src/index.html) and every JavaScript file in verification/
// must be prettier-formatted (default options).
//
// prettier is an optional dependency, like playwright-core: install it with
// `make install-tools`. When it is not installed this file is skipped, so
// the base `make test` run stays dependency-free.

"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

let prettier = null;
try {
  prettier = require("prettier");
} catch (_) {}

const skip = prettier
  ? false
  : "prettier is not installed (run `make install-tools`)";

const FILES = [
  path.join(__dirname, "..", "implementation", "src", "index.html"),
  ...fs
    .readdirSync(__dirname)
    .filter((f) => f.endsWith(".js"))
    .map((f) => path.join(__dirname, f)),
];

test("formatting", { skip }, async (t) => {
  for (const file of FILES) {
    const name = path.relative(path.join(__dirname, ".."), file);
    await t.test(`${name} is prettier-formatted`, async () => {
      const source = fs.readFileSync(file, "utf8");
      assert.ok(
        await prettier.check(source, { filepath: file }),
        `${name} is not formatted; run npx prettier --write on it`,
      );
    });
  }
});
