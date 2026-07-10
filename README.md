# lanparty.lt

## Repository layout

Specs are the source of truth — see [AGENTS.md](./AGENTS.md) for the rules.
Each package under `packages/<name>/` has three parts:

- `specification/` — the package's behavior contract
  (`specification/implementation/main.md` is the entry point) and the docs for
  how it is verified (`specification/verification/`).
- `implementation/` — the code, which must exhibit exactly the behavior the
  spec states.
- `verification/` — the blackbox tests that verify it.

Packages:

- [api](packages/api/specification/implementation/main.md) — an HTTP server
  that generates files with an LLM via OpenRouter.
- [web-frontend](packages/web-frontend/specification/implementation/main.md) —
  a static page for requesting a generated file and viewing it in the browser.

## Maintenance scripts

- `clean_git.sh` — interactive and destructive git cleanup: removes all git
  worktrees, checks out `main`, and force-deletes every local and remote
  branch except `main`. It prints what it is about to do and asks for
  confirmation first.
