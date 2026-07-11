# Manual verification

The frontend is a static page with no build step. Manual verification:

1. Start the api backend (see
   [../../api/specification/main.md](../../api/specification/main.md)) and put it behind the
   shared reverse proxy (see [Deployment](deployment.md)) so that
   `/v1/representation` reaches the api and `/` serves this package, under one
   origin. The bundled `Caddyfile` is the reference configuration for this;
   `caddy run --config Caddyfile` serves the frontend at `http://localhost:3000`
   and forwards `/v1/*` to the api on `localhost:8080`.
2. Open `http://localhost:3000`.
3. Select `html`, type a spec, click **Generate** → the same tab navigates to a
   rendered version of the generated HTML page.
4. Press the browser **back** button → returns to the form, with the previous
   type and spec still entered.
5. Select `markdown`, type a spec, click **Generate** → the same tab navigates
   to a page showing the generated markdown **rendered to HTML** by `marked`
   (headings, lists, code, emphasis, links, etc.).
6. Leave the **Spec** field empty and click **Generate** → an inline error is
   shown on the form page, and no network request is sent.
7. Stop the api backend and click **Generate** → an inline "Network error"
   message is shown and the form remains usable.
8. Inspect the request sent for a successful generation: it is a `POST` to
   `/v1/representation` with `Content-Type: application/json` and a body of the
   form `{"type":"html","spec":"..."}`.
9. Confirm `marked` is loaded from a public CDN (the `<script src>` is a CDN
   URL); the frontend's own origin serves only `index.html`.
10. Markdown raw HTML: request a `markdown` generation whose spec coaxes the LLM
    to include a `<script>` tag in its output. In the rendered tab the script
    runs, consistent with the `html` type (generated content is rendered as
    live HTML, no sanitization). This is the documented trust model.

## Manual fallback for the browser checks

When `playwright-core` or a Chromium executable is unavailable, verify the
browser checks by hand:

- Step 3 / step 5: clicking **Generate** actually **navigates the same tab** to
  the blob URL (blob navigation, not just the blob's content).
- Step 4: the browser **back button** returns to the form (session-history
  behavior of blob URLs).
- Step 10 runtime: that a `<script>` in generated markdown actually *executes*
  in the rendered tab (the content-level check that it is not stripped is
  automated; execution is browser-only).
