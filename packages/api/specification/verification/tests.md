# Tests

This document enumerates the test cases for the `api` package.

## External blackbox tests

These treat the API as an opaque compiled binary.

### Happy Path

- **Generates HTML successfully** — `POST /v1/representation` with `type: html` and a non-empty `spec` returns `200 OK`, `Content-Type: text/html; charset=utf-8`, `Content-Disposition: attachment; filename="representation.html"`, and a non-empty body.
- **Generates Markdown successfully** — `POST /v1/representation` with `type: markdown` and a non-empty `spec` returns `200 OK`, `Content-Type: text/markdown; charset=utf-8`, `Content-Disposition: attachment; filename="representation.md"`, and a non-empty body.
- **Accepts types case-insensitively and preserves their casing** — a `POST`
  with `type: HTML` and a non-empty `spec` returns `200 OK` with the html
  `Content-Type` and `Content-Disposition` headers. The fake upstream replies
  with a marker body when the user prompt contains `Generate a HTML file`
  (the original casing), and the test asserts the response body is that
  marker, proving the Generator received the type with its casing preserved.

### Request Validation

- **Rejects requests with missing fields** — a `POST` with `type` but no `spec` returns `422 Unprocessable Entity` with the JSON error message exactly `fields 'type' and 'spec' are required`.
- **Rejects requests with empty spec after trimming** — a `POST` with `type: html` and `spec: "   "` returns `422 Unprocessable Entity` with the same exact error message.
- **Rejects a `null` JSON body as missing fields** — a `POST` whose body is the JSON literal `null` returns `422 Unprocessable Entity` (not `400`) with the same exact error message.
- **Rejects unsupported types** — a `POST` with `type: json` and a non-empty `spec` returns `422 Unprocessable Entity` with the JSON error message exactly `unsupported type "json": only "html" and "markdown" are supported`.
- **Rejects bodies larger than 1 MiB** — a `POST` whose body exceeds 1 MiB returns `400 Bad Request` with a JSON error beginning `invalid request body: `.
- **Rejects malformed JSON bodies** — a `POST` with an invalid JSON body returns `400 Bad Request` with a JSON error beginning `invalid request body: `.
- **Rejects trailing content after the JSON object** — a `POST` whose body is a valid object followed by non-whitespace content returns `400 Bad Request` with a JSON error beginning `invalid request body: `.
- **Rejects non-POST HTTP methods** — a `GET` to `/v1/representation` returns `405 Method Not Allowed` with an `Allow: POST` header and a plain-text body (`Content-Type: text/plain; charset=utf-8`).

### Routing

- **Redirects the bare `/v1` path** — a `GET /v1` returns `307 Temporary Redirect` with a `Location: /v1/` header.
- **Rejects unknown paths** — a `GET` to a path outside `/v1` returns `404 Not Found` with `Content-Type: text/plain; charset=utf-8`.

### Error Handling

- **Surfaces upstream generation failures as 500** — when the upstream generator fails, `POST /v1/representation` returns `500 Internal Server Error` with a JSON error beginning `generation failed: `.

## Source formatting

Run as part of the external suite (`go test` in `verification/`).

- **Go sources are gofmt-formatted** — `gofmt -l` over the package's
  `implementation/` and `verification/` directories reports no files.

## Internal unit tests

These cover internal business logic without network overhead.

### Content type mapping

- **Maps `html` to its content type and extension** — returns `text/html; charset=utf-8` and `.html`.
- **Mapping is case-insensitive** — input `HTML` returns the same content type and extension as `html`.
- **Maps `markdown` to its content type and extension** — returns `text/markdown; charset=utf-8` and `.md`.
- **Rejects `htm`** — returns not-ok.
- **Rejects `md`** — returns not-ok.
- **Rejects `pdf`** — returns not-ok.
- **Rejects `json`** — returns not-ok.
- **Rejects unknown types** — returns not-ok.

### LLM Generator

#### Happy path

- **Returns generated text with whitespace trimmed** — given a response whose first `text` block is `  <h1>hi</h1>  `, returns `<h1>hi</h1>`.
- **Sends the configured API key as a bearer token** — the outgoing request carries `Authorization: Bearer <key>`.
- **Sends a well-formed request to the messages endpoint** — the outgoing request is a `POST` to `{baseURL}/messages` with:
  - `Content-Type: application/json`
  - `model` set to the configured model id
  - `max_tokens: 8192`
  - `stream: false`
  - a single tool of type `openrouter:bash`
  - a `system` field containing `ONLY`, `code fences`, and the requested type
  - a single `user` message

#### Tool execution loop

- **Executes bash commands and returns tool results** — when the response contains a `tool_use` block with a shell command, the generator executes it locally, then continues the conversation with the command output embedded in a `tool_result` block. If the next response contains a `text` block, that text is returned.
- **Errors after 20 iterations without text** — when the model returns only `tool_use` blocks repeatedly, generation fails with an error mentioning the 20-iteration limit, after making exactly 20 upstream requests.

#### Command sandbox

These cases drive commands through the tool execution loop and inspect the
`tool_result` blocks the generator sends upstream. They require `bwrap`
(see [verifying.md](./verifying.md)).

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

#### Content block selection

- **Skips non-text blocks** — when the response contains a `thinking` block followed by a `text` block, returns the `text` block content.

#### Content block failures

- **Errors when no text block and no tools are used** — when the response contains only a non-text block (e.g. `thinking`) and `stop_reason: end_turn`, generation fails with an error about no text content block.

#### Configuration failures

- **Errors when API key is missing** — generation fails with an error mentioning `OPENROUTER_API_KEY`, without making a network call.
- **Errors when model is missing** — generation fails with an error mentioning `OPENROUTER_MODEL`, without making a network call.
- **Does not contact the API when configuration is missing** — with a missing API key, the generator returns without any request reaching the upstream server.

#### Response failures

- **Errors on non-2xx status** — when the upstream returns a `502` status, generation fails with an error containing the status code.
- **Errors on empty content list** — when the response has `"content": []`, generation fails with an error mentioning `no content`.
- **Errors on undecodable response body** — when the response body is not valid JSON, generation fails.
- **Errors on transport failure** — when the target server is unreachable, generation fails.

#### Transport

- **Uses a 4-minute HTTP timeout** — the generator's HTTP client timeout is 4 minutes.
