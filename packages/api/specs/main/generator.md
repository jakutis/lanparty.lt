# Generator

The `POST /representation` endpoint does not produce file content itself.
It delegates generation to a **Generator**.
All logic, all error conditions, everything is extensively logged.

## LLM implementation

The default and only implementation is backed by an LLM served via the
[OpenRouter](https://openrouter.ai/) chat completions API.

### Configuration

Sourced from the environment by `loadConfig`:

| Variable                 | Required | Default                              | Description                          |
| ------------------------ | -------- | ------------------------------------ | ------------------------------------ |
| `OPENROUTER_API_KEY`     | yes      | —                                    | Bearer token for the OpenRouter API. |
| `OPENROUTER_MODEL`       | yes      | —                                    | The model id to use, e.g. `openai/gpt-4o`. |
| `OPENROUTER_BASE_URL`   | no       | `https://openrouter.ai/api/v1`      | Base URL of the chat completions API. |

A missing `OPENROUTER_API_KEY` or `OPENROUTER_MODEL` causes `Generate` to fail
immediately, before any network call is made.

### Request

The generator sends `POST {baseURL}/chat/completions` with a JSON body made up
of two messages:

1. A **system** message instructing the model to output ONLY raw file content,
   with no commentary and no markdown code fences, naming the requested file
   type.
2. A **user** message restating the file type and containing the specification
   to satisfy.

Streaming is disabled (`stream: false`). The HTTP client used for the call has
a 4-minute timeout.

### Response

The generator reads the first choice's message content, trims surrounding
whitespace, and returns it as the generated file content.

A non-2xx status, an empty choices list, or any transport/decoding error is
returned as a `Generate` error.
