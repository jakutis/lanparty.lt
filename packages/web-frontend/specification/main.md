# web-frontend package

A static web page that lets a user request a generated file from the
[api](../../api) backend and immediately view the result in the **same browser
tab**.

There is no build step and no bundler. The frontend is plain HTML, CSS and
JavaScript. The only third-party dependency is
[marked](https://marked.js.org/) (a Markdown-to-HTML renderer), loaded from a
public CDN via a `<script>` tag. The library is not vendored locally, and there
is no npm, no bundler, and no build step.

Verification has two layers: an **automated test suite** in `verification/`
(four Node.js test files, including browser checks driven through a headless
Chromium) and a **manual procedure** (see
[Manual verification](manual-verification.md)) that doubles as a fallback for
the browser checks when no browser is available. Each behavior section ends
with a **Verification** subsection enumerating the automated test cases that
cover its behavior, each tagged with the test file it lives in; the cases
specify the behavior covered by every test, not how that behavior is tested.
The suite's architecture and run instructions are described under
[Running the tests](running.md#running-the-tests).

## Specification contents

The spec is split across these documents:

- [Deployment](deployment.md) — the shared-origin reverse proxy contract and
  the reference Caddyfile.
- [Running](running.md) — how to run the page and how to run the automated
  test suite.
- [Page](page.md) — the form page and its controls.
- [Submit flow](submit-flow.md) — what happens on form submission: validation,
  the request, the success paths, and error handling.
- [Markdown rendering](markdown-rendering.md) — rendering `markdown` results
  with `marked`, the trust model for generated content, and how the `marked`
  library is loaded from a CDN.
- [Manual verification](manual-verification.md) — the manual procedure and
  the manual fallback for the browser checks.
