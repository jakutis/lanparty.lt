# Makefile for the web-frontend package's automated test suite. See
# ../specification/verification/verifying.md. Requires Node.js 18 or later and
# the Go toolchain (the contract tests compile and run the real api binary).

.PHONY: test install-browser

test:
	node --test *.test.js

# Install playwright-core, enabling the browser tests in browser.test.js
# (they are skipped when it is absent). Needs a Chromium executable: set
# CHROMIUM_PATH, or /opt/pw-browsers/chromium is used when present.
install-browser:
	npm install --no-save playwright-core
