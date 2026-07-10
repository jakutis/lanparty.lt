Specs are the source of truth: each package's behavior is defined in `packages/<name>/specs/` (entry point `specs/main.md`). The code in `packages/<name>/code/src/` and the blackbox tests in `packages/<name>/code/test/` must implement the spec in full and exhibit no behavior beyond it — the spec fixes behavior, not structure.

Specs describe externally observable behavior only — the contract a blackbox test could verify from outside the process:

- **In scope**: commands and their exit behavior, HTTP requests and responses, environment variables, filesystem effects, log output, and timing.
- **Out of scope**: how the code achieves that contract — file layout, function or type names, data structures, algorithms, and control flow.
- **Litmus test** for every sentence in a spec: if no blackbox test could detect its violation, it is an implementation detail and does not belong.
- **Exception that isn't one**: toolchain and run instructions (language version, build commands) do belong — they are part of the observable contract.

One exemption: specs may also document the verification layers themselves — test frameworks, test-suite architecture, and the enumerated test cases, including internal unit tests that live alongside the code in `packages/<name>/code/src/` and check spec-defined behavior through internal seams. Those sections describe the tests, so they may reference internals where a test does; they never extend the product contract — any behavior a test checks must already be stated in blackbox terms elsewhere in the spec.

When implementing a spec, first read it in full, then follow it literally, but be extremely sensitive and intolerant to any issues in it - anything confusing or unclear (missing details, imprecisions, contradictions, ambiguous language, missing logical steps and similar) that affects behavior. If you detect any, STOP before writing code and describe every issue found. Do not talk yourself out of an issue or resolve it with an assumption and proceed - STOP. Details the spec deliberately leaves open (naming, internal structure) are yours to decide.
