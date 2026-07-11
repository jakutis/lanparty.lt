# Deployment

The page talks to the api at the relative URL `/v1/representation`. The api
does not send CORS headers, so the page and the api MUST share a single origin.
This is achieved with a **shared reverse proxy**: one origin serves both the
static frontend and the api backend, forwarding requests by path:

| Path            | Routed to                                           |
| --------------- | --------------------------------------------------- |
| `/v1/...`       | the api backend (e.g. `localhost:8080`)             |
| everything else | the static files of this package (e.g. `index.html`)|

For example, a minimal proxy (any HTTP reverse proxy will do — nginx, Caddy,
`httpd`, etc.): serve this package's files at `/` and proxy `/v1/*` to the api.

This package ships a **reference Caddyfile** (`Caddyfile`, in the `implementation/` directory)
that implements this routing and listens on `:3000`, forwarding `/v1/*`
unchanged to the api backend at `localhost:8080` and serving `src/` for
everything else. The `/v1` prefix is passed through to the api unchanged,
because the api itself strips that prefix internally. Usage from the
`implementation/` directory:

```bash
PORT=8080 OPENROUTER_API_KEY=... OPENROUTER_MODEL=... go -C ../../api/implementation run ./src &
caddy run --config Caddyfile
```

Then open `http://localhost:3000`. The Caddyfile is the canonical example; any
equivalent proxy configuration satisfies the deployment contract.

Cross-origin use (the page and the api on different origins) is not supported
by this package.

## Verification

- **The shared origin serves the form page** (`contract.test.js`) — `GET /`
  returns 200 with `Content-Type: text/html; charset=utf-8`. The served page
  includes the required form, type options, Spec textarea, and Generate
  button.
- **The representation endpoint rejects GET** (`contract.test.js`) —
  `GET /v1/representation` returns 405. Rejecting non-`POST` methods has no
  corresponding step in the [manual procedure](manual-verification.md); it is
  only ever verified automatically.
