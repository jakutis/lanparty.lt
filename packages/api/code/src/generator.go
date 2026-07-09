package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
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

	reqBody := messagesRequest{
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
		Tools: []tool{
			{Type: "openrouter:bash"},
		},
	}

	for i := 0; i < 20; i++ {
		raw, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		log.Printf("llm: sending %d-byte request to %s (iteration %d)", len(raw), g.baseURL+"/messages", i+1)

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
		
		log.Printf("llm: received status %d from %s", resp.StatusCode, g.baseURL)

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
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
		for j, b := range mr.Content {
			log.Printf("llm: content[%d] type=%q text-len=%d", j, b.Type, len(b.Text))
		}
		if len(mr.Content) == 0 {
			return nil, fmt.Errorf("llm returned no content")
		}

		selected := -1
		for j := range mr.Content {
			if mr.Content[j].Type == "text" {
				selected = j
				break
			}
		}
		if selected >= 0 {
			block := &mr.Content[selected]
			log.Printf("llm: selected content[%d] type=%q text-len=%d as the file content",
				selected, block.Type, len(block.Text))
			content := []byte(strings.TrimSpace(block.Text))
			log.Printf("llm: produced %d bytes of %q content (stop_reason=%q)",
				len(content), typ, mr.StopReason)
			return content, nil
		}

		var toolResults []contentBlock
		for _, b := range mr.Content {
			if b.Type == "tool_use" {
				var input struct {
					Command string `json:"command"`
				}
				if err := json.Unmarshal(b.Input, &input); err != nil {
					return nil, fmt.Errorf("unmarshal tool input: %w", err)
				}

				log.Printf("llm: executing bash command: %s", input.Command)
				cmd := exec.CommandContext(ctx, "bash", "-c", input.Command)
				out, err := cmd.CombinedOutput()
				
				res := contentBlock{
					Type:      "tool_result",
					ToolUseID: b.ID,
					Content:   string(out),
					IsError:   err != nil,
				}
				if err != nil && len(out) == 0 {
					res.Content = err.Error()
				}
				toolResults = append(toolResults, res)
			}
		}

		if len(toolResults) == 0 {
			return nil, fmt.Errorf("llm returned no text content block and no tool calls")
		}

		reqBody.Messages = append(reqBody.Messages, message{
			Role:    "assistant",
			Content: mr.Content,
		})
		reqBody.Messages = append(reqBody.Messages, message{
			Role:    "user",
			Content: toolResults,
		})
	}

	return nil, fmt.Errorf("llm exceeded maximum 20 iterations without producing text content")
}

// message is a single entry in the Anthropic Messages API messages array.
type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// tool defines a tool available to the model.
type tool struct {
	Type string `json:"type"`
}

// messagesRequest is the Anthropic Messages API ("Create a message") request
// body sent to OpenRouter.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
	Stream    bool      `json:"stream"`
	Tools     []tool    `json:"tools,omitempty"`
}

// messagesResponse is the Anthropic Messages API message object returned by
// OpenRouter.
type messagesResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

// contentBlock is a single block within an Anthropic Messages API response,
// or a block representing tool use/result in requests.
type contentBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ID         string          `json:"id,omitempty"`          // for tool_use
	Name       string          `json:"name,omitempty"`        // for tool_use
	Input      json.RawMessage `json:"input,omitempty"`       // for tool_use
	ToolUseID  string          `json:"tool_use_id,omitempty"` // for tool_result
	Content    string          `json:"content,omitempty"`     // for tool_result
	IsError    bool            `json:"is_error,omitempty"`    // for tool_result
}
