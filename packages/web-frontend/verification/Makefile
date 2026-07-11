# Makefile for the web-frontend package's automated test suite. See
# ../specification/main.md#running-the-tests. Requires Node.js 18 or later and
# the Go toolchain (the contract tests compile and run the real api binary).

.PHONY: test install-tools

test:
	node --test *.test.js

# Install the optional test tooling in one npm call (npm prunes packages it
# was not asked for, so installing them separately would remove the other):
# playwright-core enables the browser tests in browser.test.js (they also
# need a Chromium executable: set CHROMIUM_PATH, or /opt/pw-browsers/chromium
# is used when present), and prettier enables the formatting checks in
# format.test.js. Each group is skipped when its package is absent.
install-tools:
	npm install --no-save playwright-core prettier
