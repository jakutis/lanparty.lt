# Generator

The `POST /representation` endpoint does not produce file content itself.
It delegates generation to a **Generator**.
All logic, all error conditions, everything is extensively logged.

## LLM implementation

The default and only implementation is backed by an LLM served via the
[OpenRouter](https://openrouter.ai/) Anthropic Messages API endpoint - Create a message.


### Configuration

Sourced from the environment by `loadConfig`:

| Variable                 | Required | Default                              | Description                          |
| ------------------------ | -------- | ------------------------------------ | ------------------------------------ |
| `OPENROUTER_API_KEY`     | yes      | —                                    | Bearer token for the OpenRouter API. |
| `OPENROUTER_MODEL`       | yes      | —                                    | The model id to use, e.g. `anthropic/claude-3.5-sonnet`. |
| `OPENROUTER_BASE_URL`   | no       | `https://openrouter.ai/api/v1`      | Base URL of the OpenRouter API. |

A missing `OPENROUTER_API_KEY` or `OPENROUTER_MODEL` causes `Generate` to fail
immediately, before any network call is made.

### Request

The generator sends `POST {baseURL}/messages` with a JSON body matching the
Anthropic Messages API ("Create a message") shape:

1. A top-level `model` field set to the configured model id.
2. A top-level `max_tokens` field set to `8192`.
3. A top-level `system` field (a string) instructing the model to output ONLY
   raw file content, with no commentary and no markdown code fences, naming the
   requested file type.
4. A `messages` array containing a single user message that restates the file
   type and contains the specification to satisfy.

The HTTP client used for the call has
a 4-minute timeout.

### Response

The response is an Anthropic Messages API message object whose `content` field
is an array of content blocks. The generator reads the `text` field of the
first content block, trims surrounding whitespace, and returns it as the
generated file content.

A non-2xx status, an empty `content` list, or any transport/decoding error is
returned as a `Generate` error.
