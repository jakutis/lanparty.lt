# Generator

The `POST /representation` endpoint does not produce file content itself.
It delegates generation to a **Generator**.

## LLM implementation

The default and only implementation is backed by an LLM served via the
[OpenRouter](https://openrouter.ai/) Anthropic Messages API endpoint - [Create a message](https://openrouter.ai/docs/api/api-reference/anthropic-messages/create-a-message).

### Configuration

Sourced from the environment by `loadConfig`:

| Variable                 | Required | Default                              | Description                          |
| ------------------------ | -------- | ------------------------------------ | ------------------------------------ |
| `OPENROUTER_API_KEY`     | yes      | —                                    | Bearer token for the OpenRouter API. |
| `OPENROUTER_MODEL`       | yes      | —                                    | The model id to use, e.g. `anthropic/claude-3.5-sonnet`. |
| `OPENROUTER_BASE_URL`    | no       | `https://openrouter.ai/api/v1`       | Base URL of the OpenRouter API. A trailing slash, if present, is stripped before use. |

A missing `OPENROUTER_API_KEY` or `OPENROUTER_MODEL` causes `Generate` to fail
immediately, before any network call is made.

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

### Prompt template

The `system` field and the single user message are produced from the following
literal templates, where `<type>` is the requested file type and `<spec>` is the
given specification:

- **`system`** —
  `You generate raw file content. Output ONLY the file content, with no commentary and no markdown code fences. The file type is "<type>".`
  (the value of `<type>` is quoted).
- **user message** —
  `Generate a <type> file that satisfies the following specification:\n\n<spec>`
  (a literal newline separates the preamble from the specification; the value
  of `<type>` is inserted verbatim, unquoted).

The request carries an `Authorization: Bearer <OPENROUTER_API_KEY>` header and
a `Content-Type: application/json` header.

The HTTP client used for the call has
a 4-minute timeout.

### Response

The response is an Anthropic Messages API message object whose `content` field
is an array of content blocks, each carrying a `type` (e.g. `text` or
`thinking`). The generator reads the `text` field of the first content block
whose `type` is `text`, trims surrounding whitespace, and returns it as the
generated file content. If no content block is typed `text`, `Generate` fails.

A non-2xx status, an empty `content` list, no content block typed `text`, or
any transport/decoding error is returned as a `Generate` error.
