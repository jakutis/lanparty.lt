# lanparty.lt

## Repository layout

Specs are the source of truth — see [AGENTS.md](./AGENTS.md) for the rules.
Each package under `packages/<name>/` has three parts:

- `specification/` — the package's integrated spec, entered through
  `specification/main.md`: the behavior contract together with how the
  package is verified.
- `implementation/` — the code, which must exhibit exactly the behavior the
  spec states.
- `verification/` — the blackbox tests that verify it.

Packages:

- [api](packages/api/specification/main.md) — an HTTP server
  that generates files with an LLM via OpenRouter.
- [web-frontend](packages/web-frontend/specification/main.md) —
  a static page for requesting a generated file and viewing it in the browser.

## Maintenance scripts

- `clean_git.sh` — interactive and destructive git cleanup: removes all git
  worktrees, checks out `main`, and force-deletes every local and remote
  branch except `main`. It prints what it is about to do and asks for
  confirmation first.
