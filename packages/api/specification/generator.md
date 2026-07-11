# Generator

The `POST /representation` endpoint (see [Endpoints](endpoints.md)) does not
produce file content itself. It
delegates generation to a **Generator**: an LLM-backed component that produces
raw file content for a given type and spec. The default and only
implementation is backed by an LLM served via the
[OpenRouter](https://openrouter.ai/) Anthropic Messages API endpoint - [Create a message](https://openrouter.ai/docs/api/api-reference/anthropic-messages/create-a-message).

## Configuration

Sourced from the environment:

| Variable                 | Required | Default                              | Description                          |
| ------------------------ | -------- | ------------------------------------ | ------------------------------------ |
| `OPENROUTER_API_KEY`     | yes      | —                                    | Bearer token for the OpenRouter API. |
| `OPENROUTER_MODEL`       | yes      | —                                    | The model id to use, e.g. `anthropic/claude-3.5-sonnet`. |
| `OPENROUTER_BASE_URL`    | no       | `https://openrouter.ai/api/v1`       | Base URL of the OpenRouter API. Trailing slashes, if present, are stripped before use. |

A missing `OPENROUTER_API_KEY` or `OPENROUTER_MODEL` causes generation to fail
immediately, before any network call is made.

As with the other optional configuration, an empty `OPENROUTER_BASE_URL` is
treated as unset and uses the default.

### Verification

Internal unit cases:

- **Errors when API key is missing** — generation fails with an error mentioning `OPENROUTER_API_KEY`, without making a network call.
- **Errors when model is missing** — generation fails with an error mentioning `OPENROUTER_MODEL`, without making a network call.
- **Does not contact the API when configuration is missing** — with a missing API key, the generator returns without any request reaching the upstream server.

## Request

The generator sends `POST {baseURL}/messages` with a JSON body matching the
Anthropic Messages API ("Create a message") shape:

1. A top-level `model` field set to the configured model id.
2. A top-level `max_tokens` field set to `8192`.
3. A top-level `system` field (a string) set to the **system prompt template**
   (see [Prompt template](#prompt-template)) with `<type>` substituted from the
   requested file type.
4. A `messages` array containing a single user message set to the **user
   message template** (see [Prompt template](#prompt-template)) with `<type>`
   and `<spec>` substituted from the requested file type and the given
   specification respectively.
5. A top-level `stream` field set to `false` (the response is not streamed).
6. A top-level `tools` array containing a single tool object: `{ "type": "openrouter:bash" }`.

## Prompt template

The `system` field and the single user message are produced from the following
literal templates, where `<type>` is the requested file type and `<spec>` is the
given specification:

- **`system`** —
  `You generate raw file content. Output ONLY the file content, with no commentary and no markdown code fences. The file type is "<type>".`
  (the value of `<type>` is quoted).
- **user message** —
  `Generate a <type> file that satisfies the following specification:\n\n<spec>`
  (a blank line — two literal newline characters — separates the preamble from
  the specification; the value of `<type>` is inserted verbatim, unquoted).

The request carries an `Authorization: Bearer <OPENROUTER_API_KEY>` header and
a `Content-Type: application/json` header.

Each HTTP call to the OpenRouter API, including reading its response, times
out after 4 minutes.

### Verification

Internal unit cases covering the [Request](#request) shape, the prompt
template, and the transport:

- **Sends the configured API key as a bearer token** — the outgoing request carries `Authorization: Bearer <key>`.
- **Sends a well-formed request to the messages endpoint** — the outgoing request is a `POST` to `{baseURL}/messages` with:
  - `Content-Type: application/json`
  - `model` set to the configured model id
  - `max_tokens: 8192`
  - `stream: false`
  - a single tool of type `openrouter:bash`
  - a `system` field containing `ONLY`, `code fences`, and the requested type
  - a single `user` message
- **Uses a 4-minute HTTP timeout** — the generator's HTTP client timeout is 4 minutes.

## Generation loop

The generator operates in a loop, repeatedly calling the OpenRouter API until a `text` block is produced, up to a maximum of 20 iterations.

In each iteration, the generator reads the response, which is an Anthropic Messages API message object whose `content` field is an array of content blocks (e.g. `text`, `thinking`, `tool_use`).

1. If the `content` array contains a `text` block, the generator reads the `text` field of the first such block, trims surrounding whitespace, and returns it as the generated file content. The generation loop terminates successfully.
2. If the `content` array contains one or more `tool_use` blocks, the generator executes the shell command carried in each block's `input.command` field using `bash -c` inside the [command sandbox](#command-sandbox). It captures both standard output and standard error.
   The generator then appends two messages to the `messages` array for the next API call:
   - An `assistant` message containing the `content` array received from the model in the current iteration, re-encoded: only the `type`, `text`, `id`, `name`, `input`, `tool_use_id`, `content`, and `is_error` fields of each block are preserved (empty ones are omitted); any other fields — for example a `thinking` block's `thinking` text and `signature` — are dropped.
   - A `user` message containing an array of `tool_result` blocks, one for each `tool_use` block processed. Each `tool_result` block carries the `tool_use_id` and its `content` is a string with the combined standard output and standard error. If a failing command produces no output, its error text is used as `content` instead. If the command exited with a non-zero code or failed to start, the `is_error` field is set to `true`.
3. If the loop completes 20 iterations without producing a `text` block, generation fails.
4. A non-2xx status, an empty `content` list, a `tool_use` block whose input cannot be decoded, an unhandled content block combination, or any transport/decoding error immediately terminates the loop and fails the generation with that error.

### Verification

Internal unit cases:

#### Happy path

- **Returns generated text with whitespace trimmed** — given a response whose first `text` block is `  <h1>hi</h1>  `, returns `<h1>hi</h1>`.

#### Tool execution loop

- **Executes bash commands and returns tool results** — when the response contains a `tool_use` block with a shell command, the generator executes it locally, then continues the conversation with the command output embedded in a `tool_result` block. If the next response contains a `text` block, that text is returned.
- **Errors after 20 iterations without text** — when the model returns only `tool_use` blocks repeatedly, generation fails with an error mentioning the 20-iteration limit, after making exactly 20 upstream requests.

#### Content block selection

- **Skips non-text blocks** — when the response contains a `thinking` block followed by a `text` block, returns the `text` block content.

#### Content block failures

- **Errors when no text block and no tools are used** — when the response contains only a non-text block (e.g. `thinking`) and `stop_reason: end_turn`, generation fails with an error about no text content block.

#### Response failures

- **Errors on non-2xx status** — when the upstream returns a `502` status, generation fails with an error containing the status code.
- **Errors on empty content list** — when the response has `"content": []`, generation fails with an error mentioning `no content`.
- **Errors on undecodable response body** — when the response body is not valid JSON, generation fails.
- **Errors on transport failure** — when the target server is unreachable, generation fails.

## Command sandbox

Tool commands never run directly on the host. The generator wraps every
`bash -c` invocation in [bubblewrap](https://github.com/containers/bubblewrap)
(`bwrap`), which must be installed on the host — see the toolchain
requirements in [main.md](main.md). The sandbox has the following
observable properties:

- **Working directory** — the first command of a generation creates a fresh,
  empty scratch directory on the host; when the generation finishes (whether
  it succeeds or fails), the directory and everything in it are deleted.
  Inside the sandbox the directory is mounted read-write at `/work`, and
  every command starts with `/work` as its current directory. Files written
  under `/work` therefore persist from one command to the next within a
  single generation, and never across generations.
- **Filesystem** — the sandbox root contains only: the host's `/usr`,
  mounted read-only; `/bin`, `/sbin`, `/lib` and `/lib64` as symlinks into
  `/usr`; a private `/proc` and a minimal `/dev`; a private tmpfs at `/tmp`
  (fresh for every command); and `/work`. Nothing else from the host —
  `/etc`, `/home`, `/root`, the server's own binary and working directory —
  is visible. Writes anywhere outside `/work` and `/tmp` fail.
- **Environment** — the environment is cleared and exactly three variables
  are set: `PATH=/usr/bin:/bin`, `HOME=/work` and `TMPDIR=/tmp` (the shell
  itself may add its own bookkeeping variables such as `PWD` and `SHLVL`).
  No variable of the server's environment — in particular
  `OPENROUTER_API_KEY` — is ever visible to a command.
- **Network** — commands run with all namespaces unshared, including the
  network namespace: no usable network interface exists, and no command can
  reach the host's network, loopback included.
- **Time limit** — a command still running 60 seconds after it started is
  killed. Its `tool_result` is reported like any other failing command
  (`is_error` set to `true`), carrying whatever output the command produced
  before it was killed; the generation loop itself continues.
- **Process cleanup** — commands run in a private PID namespace that is torn
  down when the command's shell exits: background processes a command leaves
  behind are killed immediately, and the command's result is returned
  without waiting for them.
- **Output cap** — only the first 65536 bytes (64 KiB) of a command's
  combined output are kept. Truncated output ends with the line
  `[output truncated]`. Truncation alone does not mark the result as an
  error.

A command that cannot be started at all (for example, when `bwrap` is not
installed) is reported through the failed-command contract above: its
`tool_result` carries the error text as `content` with `is_error` set to
`true`.

### Verification

Internal unit cases. They drive commands through the tool execution loop and
inspect the `tool_result` blocks the generator sends upstream. They execute
real sandboxed commands, so they require `bwrap` on `PATH` and a kernel
permitting unprivileged user namespaces (see the toolchain requirements in
[main.md](main.md)).

- **Hides the server's environment** — with `OPENROUTER_API_KEY` set in the
  server process's environment, a command running `env` reports `HOME=/work`
  but never the API key's value.
- **Denies network access** — a command opening a TCP connection (via bash's
  `/dev/tcp`) to the mock upstream server's address — reachable from the
  server process itself — fails, and its `tool_result` reports no
  connectivity.
- **Runs commands in `/work`** — a command running `pwd` outputs `/work`.
- **Persists files across commands within one generation** — a first command
  writes a file, and a second command in the same generation reads its
  content back.
- **Isolates generations from each other** — a file written during one
  `Generate` call does not exist during a subsequent call, whose read
  attempt produces an erroring `tool_result`.
- **Removes the scratch directory when generation ends** — after a
  generation whose command wrote a file completes, no sandbox scratch
  directories remain in the host's temp directory (the test checks the
  internal `api-sandbox-` naming seam).
- **Mounts the system read-only** — a command writing to `/usr` fails with
  an erroring `tool_result` mentioning a read-only file system.
- **Kills long-running commands** — using an internal seam that shortens the
  60-second limit, a command that prints output and then sleeps past the
  limit produces an erroring `tool_result` that still carries the printed
  output, and the generation loop continues to a successful text response.
- **Truncates output at 64 KiB** — a command emitting more than 65536 bytes
  yields a non-erroring `tool_result` whose content is the first 65536 bytes
  followed by the `[output truncated]` marker line.
- **Does not wait for background processes** — a command that starts a
  30-second background `sleep` and echoes a marker returns the marker
  promptly (the test asserts completion well before the sleep could finish).
