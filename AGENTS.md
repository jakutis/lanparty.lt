Specs are the source of truth. Each package's behavior is defined in `packages/<name>/specs/`, with `specs/main.md` as the entry point. The code in `packages/<name>/code/src/` must exhibit exactly the behavior the spec states — all of it, and nothing beyond it — and the blackbox tests in `packages/<name>/code/test/` verify that it does.

A spec describes only what can be observed from outside the running process — the contract a blackbox test could verify:

- **In scope**: commands and their exit behavior, HTTP requests and responses, environment variables, filesystem effects, log output, and timing. Toolchain and run instructions (language version, build commands) also belong — they are part of the observable contract.
- **Out of scope**: how the code achieves that contract — file layout, function or type names, data structures, algorithms, and control flow.
- **Litmus test** for every sentence in a spec: if no blackbox test could detect its violation, it is an implementation detail and does not belong.

One exemption: specs may also document the verification layers themselves — test frameworks, test-suite architecture, and the enumerated test cases, including internal unit tests that live alongside the code in `packages/<name>/code/src/` and check spec-defined behavior through internal seams. Those sections describe the tests, so they may reference internals where a test does; they never extend the product contract — any behavior a test checks must already be stated in blackbox terms elsewhere in the spec.

When implementing a spec, first read it in full, then follow it literally, but be extremely sensitive and intolerant to any issues in it - anything confusing or unclear (missing details, imprecisions, contradictions, ambiguous language, missing logical steps and similar) that affects behavior. If you detect any, STOP before writing code and describe every issue found. Do not talk yourself out of an issue or resolve it with an assumption and proceed - STOP. Details the spec deliberately leaves open (naming, internal structure) are yours to decide.
