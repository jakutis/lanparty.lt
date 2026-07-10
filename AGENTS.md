# Specs are the source of truth

Each package's behavior is defined in `packages/<name>/specification/`, with `specification/implementation/main.md` as the entry point; the docs describing how the package is verified live in `specification/verification/`. The code in `packages/<name>/implementation/` must exhibit exactly the behavior the spec states — all of it, and nothing beyond it. The blackbox tests in `packages/<name>/verification/` verify that it does.

## What belongs in a spec

A spec describes only what can be observed from outside the running process — the contract a blackbox test could verify.

- **In scope**: commands and their exit behavior, HTTP requests and responses, environment variables, filesystem effects, log output, and timing. Toolchain and run instructions (language version, build commands) also belong: they are part of the observable contract.
- **Out of scope**: how the code achieves that contract — file layout, function and type names, data structures, algorithms, and control flow.
- **Litmus test**: for every sentence in a spec, ask whether a blackbox test could detect its violation. If none could, the sentence states an implementation detail and does not belong.

### One exception: the verification layers

A spec may also document its own verification layers: the test frameworks, the test-suite architecture, and the enumerated test cases. This includes internal unit tests that live alongside the code in `packages/<name>/implementation/` and check spec-defined behavior through internal seams. Because these sections describe tests, they may reference internals wherever a test does. They never extend the product contract: any behavior a test checks must already be stated in blackbox terms elsewhere in the spec.

## How to implement a spec

1. Read the spec in full before writing any code.
2. Follow it literally.
3. Be extremely sensitive to, and intolerant of, any issue in the spec that affects behavior: missing details, imprecision, contradictions, ambiguous language, missing logical steps, or anything else confusing or unclear.
4. If you detect any such issue, STOP before writing code and describe every issue you found. Do not talk yourself out of an issue, and do not paper over it with an assumption — STOP.
5. Details the spec deliberately leaves open (naming, internal structure) are yours to decide.
