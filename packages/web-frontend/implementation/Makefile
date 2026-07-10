# Makefile for the web-frontend package.
#
# The frontend is a static page with no build step. `run` starts the api
# backend and the bundled Caddy reverse proxy so the page and the api share
# one origin (the api sends no CORS headers). See ../specification/main.md
# and ../specification/main/verifying.md.

.PHONY: run check-env

# Serve the frontend at http://localhost:3000 with /v1/* forwarded to the api
# backend on :8080 (mirrors the bundled Caddyfile). The api backend needs
# OPENROUTER_API_KEY and OPENROUTER_MODEL (checked first). Ctrl+C stops caddy;
# the backgrounded api is cleaned up via the EXIT trap.
run: check-env
	@trap 'kill $$API_PID 2>/dev/null || true' INT TERM EXIT; \
	PORT=8080 go -C ../../api/implementation run ./src & API_PID=$$!; \
	caddy run --config Caddyfile

# Verify the env vars required by the api backend (started by `run`).
check-env:
	@test -n "$$OPENROUTER_API_KEY" \
		|| { echo "OPENROUTER_API_KEY is not set"; exit 1; }
	@test -n "$$OPENROUTER_MODEL" \
		|| { echo "OPENROUTER_MODEL is not set"; exit 1; }
