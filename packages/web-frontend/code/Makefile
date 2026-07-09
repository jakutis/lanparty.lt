# Makefile for the web-frontend package.
#
# The frontend is a static page with no build step. `run` starts the api
# backend and the bundled Caddy reverse proxy so the page and the api share
# one origin (the api sends no CORS headers). `test` runs the automated
# Node.js test suite. See specs/main.md and specs/main/verifying.md.

.PHONY: run test check-env

# Serve the frontend at http://localhost:3000 with /v1/* forwarded to the api
# backend on :8080 (mirrors the bundled Caddyfile). The api backend needs
# OPENROUTER_API_KEY and OPENROUTER_MODEL (checked first). Ctrl+C stops caddy;
# the backgrounded api is cleaned up via the EXIT trap.
run: check-env
	@trap 'kill $$API_PID 2>/dev/null || true' INT TERM EXIT; \
	PORT=8080 go -C ../../api/code run ./src & API_PID=$$!; \
	caddy run --config Caddyfile

# Run the automated test suite (Node.js only, no other dependencies).
test:
	node --test test/*.test.js

# Verify the env vars required by the api backend (started by `run`).
check-env:
	@test -n "$$OPENROUTER_API_KEY" \
		|| { echo "OPENROUTER_API_KEY is not set"; exit 1; }
	@test -n "$$OPENROUTER_MODEL" \
		|| { echo "OPENROUTER_MODEL is not set"; exit 1; }
