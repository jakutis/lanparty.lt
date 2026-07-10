# Generator

The `POST /representation` endpoint does not produce file content itself.
It delegates generation to a **Generator**.

## LLM implementation

The default and only implementation is backed by an LLM served via the
[OpenRouter](https://openrouter.ai/) Anthropic Messages API endpoint - [Create a message](https://openrouter.ai/docs/api/api-reference/anthropic-messages/create-a-message).

### Configuration

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

### Request

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

### Prompt template

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

### Generation loop

The generator operates in a loop, repeatedly calling the OpenRouter API until a `text` block is produced, up to a maximum of 20 iterations.

In each iteration, the generator reads the response, which is an Anthropic Messages API message object whose `content` field is an array of content blocks (e.g. `text`, `thinking`, `tool_use`).

1. If the `content` array contains a `text` block, the generator reads the `text` field of the first such block, trims surrounding whitespace, and returns it as the generated file content. The generation loop terminates successfully.
2. If the `content` array contains one or more `tool_use` blocks, the generator executes the shell command carried in each block's `input.command` field using `bash -c` inside the [command sandbox](#command-sandbox). It captures both standard output and standard error.
   The generator then appends two messages to the `messages` array for the next API call:
   - An `assistant` message containing the `content` array received from the model in the current iteration, re-encoded: only the `type`, `text`, `id`, `name`, `input`, `tool_use_id`, `content`, and `is_error` fields of each block are preserved (empty ones are omitted); any other fields — for example a `thinking` block's `thinking` text and `signature` — are dropped.
   - A `user` message containing an array of `tool_result` blocks, one for each `tool_use` block processed. Each `tool_result` block carries the `tool_use_id` and its `content` is a string with the combined standard output and standard error. If a failing command produces no output, its error text is used as `content` instead. If the command exited with a non-zero code or failed to start, the `is_error` field is set to `true`.
3. If the loop completes 20 iterations without producing a `text` block, generation fails.
4. A non-2xx status, an empty `content` list, a `tool_use` block whose input cannot be decoded, an unhandled content block combination, or any transport/decoding error immediately terminates the loop and fails the generation with that error.

### Command sandbox

Tool commands never run directly on the host. The generator wraps every
`bash -c` invocation in [bubblewrap](https://github.com/containers/bubblewrap)
(`bwrap`), which must be installed on the host — see the toolchain
requirements in [main.md](./main.md). The sandbox has the following
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
