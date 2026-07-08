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

// llmGenerator implements Generator by prompting the OpenRouter Anthropic
// Messages API ("Create a message") endpoint and returning the first content
// block's text as the generated file content.
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

	body := messagesRequest{
		Model:     g.model,
		MaxTokens: 8192,
		System: fmt.Sprintf(
			"You generate raw file content. Output ONLY the file content, "+
				"with no commentary and no markdown code fences. "+
				"The file type is %q.", typ),
		Messages: []message{
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
	log.Printf("llm: sending %d-byte request to %s", len(raw), g.baseURL+"/messages")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.baseURL+"/messages", bytes.NewReader(raw))
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
	log.Printf("llm: raw response body (%d bytes): %s", len(respBody), respBody)
	log.Printf("llm: decoding %d-byte response", len(respBody))

	var mr messagesResponse
	if err := json.Unmarshal(respBody, &mr); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}
	for i, b := range mr.Content {
		log.Printf("llm: content[%d] type=%q text-len=%d", i, b.Type, len(b.Text))
	}
	if len(mr.Content) == 0 {
		return nil, fmt.Errorf("llm returned no content")
	}
	content := []byte(strings.TrimSpace(mr.Content[0].Text))
	log.Printf("llm: produced %d bytes of %q content (stop_reason=%q)",
		len(content), typ, mr.StopReason)
	return content, nil
}

// message is a single entry in the Anthropic Messages API messages array.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesRequest is the Anthropic Messages API ("Create a message") request
// body sent to OpenRouter.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
	Stream    bool      `json:"stream"`
}

// messagesResponse is the Anthropic Messages API message object returned by
// OpenRouter. Its Content field is an array of content blocks; the generator
// uses the text of the first block. StopReason is captured only for logging.
type messagesResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

// contentBlock is a single block within an Anthropic Messages API response.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
