# Tests

This document enumerates the test cases for the `api` package.

## External blackbox tests

These treat the API as an opaque compiled binary.

### Happy Path

- **Generates HTML successfully** — `POST /v1/representation` with `type: html` and a non-empty `spec` returns `200 OK`, `Content-Type: text/html; charset=utf-8`, `Content-Disposition: attachment; filename="representation.html"`, and a non-empty body.
- **Generates Markdown successfully** — `POST /v1/representation` with `type: markdown` and a non-empty `spec` returns `200 OK`, `Content-Type: text/markdown; charset=utf-8`, `Content-Disposition: attachment; filename="representation.md"`, and a non-empty body.

### Request Validation

- **Rejects requests with missing fields** — a `POST` with `type` but no `spec` returns `422 Unprocessable Entity` with a JSON error body containing a non-empty `error` string.
- **Rejects requests with empty spec after trimming** — a `POST` with `type: html` and `spec: "   "` returns `422 Unprocessable Entity` with a non-empty JSON error.
- **Rejects unsupported types** — a `POST` with `type: json` and a non-empty `spec` returns `422 Unprocessable Entity` with a non-empty JSON error.
- **Rejects malformed JSON bodies** — a `POST` with an invalid JSON body returns `400 Bad Request` with a non-empty JSON error.
- **Rejects non-POST HTTP methods** — a `GET` to `/v1/representation` returns `405 Method Not Allowed`.

### Error Handling

- **Surfaces upstream generation failures as 500** — when the upstream generator fails, `POST /v1/representation` returns `500 Internal Server Error` with a non-empty JSON error.

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
- **Errors after 20 iterations without text** — when the model returns only `tool_use` blocks repeatedly, `Generate` fails with an error mentioning the 20-iteration limit, after making exactly 20 upstream requests.

#### Content block selection

- **Skips non-text blocks** — when the response contains a `thinking` block followed by a `text` block, returns the `text` block content.

#### Content block failures

- **Errors when no text block and no tools are used** — when the response contains only a non-text block (e.g. `thinking`) and `stop_reason: end_turn`, `Generate` fails with an error about no text content block.

#### Configuration failures

- **Errors when API key is missing** — `Generate` fails with an error mentioning `OPENROUTER_API_KEY`, without making a network call.
- **Errors when model is missing** — `Generate` fails with an error mentioning `OPENROUTER_MODEL`, without making a network call.
- **Does not contact the API when configuration is missing** — with a missing API key, `Generate` returns without any request reaching the upstream server.

#### Response failures

- **Errors on non-2xx status** — when the upstream returns a `502` status, `Generate` fails with an error containing the status code.
- **Errors on empty content list** — when the response has `"content": []`, `Generate` fails with an error mentioning `no content`.
- **Errors on undecodable response body** — when the response body is not valid JSON, `Generate` fails.
- **Errors on transport failure** — when the target server is unreachable, `Generate` fails.

#### Transport

- **Uses a 4-minute HTTP timeout** — the generator's HTTP client timeout is 4 minutes.
