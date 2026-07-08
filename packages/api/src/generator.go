package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Generator produces the raw content of a file for the requested type,
// according to the given specification (a natural-language prompt).
type Generator interface {
	Generate(ctx context.Context, typ, spec string) ([]byte, error)
}

// llmGenerator implements Generator by prompting an OpenRouter chat
// completions API and returning its textual output as the generated file
// content.
type llmGenerator struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func newLLMGenerator(cfg config) *llmGenerator {
	return &llmGenerator{
		apiKey:  cfg.apiKey,
		baseURL: strings.TrimRight(cfg.baseURL, "/"),
		model:   cfg.model,
		client:  &http.Client{Timeout: 4 * time.Minute},
	}
}

func (g *llmGenerator) Generate(ctx context.Context, typ, spec string) ([]byte, error) {
	log.Printf("llm: generating %q content for spec (len=%d)", typ, len(spec))
	if g.apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is not configured")
	}
	if g.model == "" {
		return nil, fmt.Errorf("OPENROUTER_MODEL is not configured")
	}
	log.Printf("llm: using model=%q base-url=%q", g.model, g.baseURL)

	body := chatCompletionsRequest{
		Model: g.model,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: fmt.Sprintf(
					"You generate raw file content. Output ONLY the file content, "+
						"with no commentary and no markdown code fences. "+
						"The file type is %q.", typ),
			},
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Generate a %s file that satisfies the following specification:\n\n%s",
					typ, spec),
			},
		},
		Stream: false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	log.Printf("llm: sending %d-byte request to %s", len(raw), g.baseURL+"/chat/completions")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()
	log.Printf("llm: received status %d from %s", resp.StatusCode, g.baseURL)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm returned status %d: %s",
			resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	log.Printf("llm: decoding %d-byte response", len(respBody))

	var cr chatCompletionsResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("llm returned no choices")
	}
	content := []byte(strings.TrimSpace(cr.Choices[0].Message.Content))
	log.Printf("llm: produced %d bytes of %q content (finish_reason=%q)",
		len(content), typ, cr.Choices[0].FinishReason)
	return content, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}
