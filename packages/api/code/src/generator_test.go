package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// capturedRequest records what the generator sent to the OpenRouter API.
type capturedRequest struct {
	method      string
	path        string
	auth        string
	contentType string
	body        []byte
}

// apiServer is a test double for the OpenRouter Anthropic Messages API. It
// records the incoming request and replies with a configurable status code
// and body, so each spec can exercise a different response condition.
type apiServer struct {
	*httptest.Server
	captured []capturedRequest
	status   int
	bodies   []string
}

func newAPIServer() *apiServer {
	s := &apiServer{status: http.StatusOK}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *apiServer) handle(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	s.captured = append(s.captured, capturedRequest{
		method:      r.Method,
		path:        r.URL.Path,
		auth:        r.Header.Get("Authorization"),
		contentType: r.Header.Get("Content-Type"),
		body:        b,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.status)
	if len(s.bodies) > 0 {
		_, _ = io.WriteString(w, s.bodies[0])
		s.bodies = s.bodies[1:]
	} else {
		// Default to something that won't panic, but probably fails
		_, _ = io.WriteString(w, `{"content":[]}`)
	}
}

// generator returns an llmGenerator pointed at this test server.
func (s *apiServer) generator() *llmGenerator {
	return newLLMGenerator(config{
		apiKey:  "test-key",
		baseURL: s.URL,
		model:   "test/model",
	})
}

var _ = Describe("llmGenerator", func() {
	var s *apiServer

	BeforeEach(func() {
		s = newAPIServer()
		DeferCleanup(func() { s.Close() })
	})

	Describe("happy path", func() {
		BeforeEach(func() {
			s.bodies = []string{`{"content":[{"type":"text","text":"  <h1>hi</h1>  "}],"stop_reason":"end_turn"}`}
		})

		It("returns the first content block's text with surrounding whitespace trimmed", func() {
			gen := s.generator()
			out, err := gen.Generate(context.Background(), "html", "make a page")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("<h1>hi</h1>"))
		})

		It("sends a bearer token derived from the configured api key", func() {
			gen := s.generator()
			_, _ = gen.Generate(context.Background(), "html", "x")
			Expect(s.captured[0].auth).To(Equal("Bearer test-key"))
		})

		It("posts a JSON body to {baseURL}/messages with model, max_tokens, system and a single user message, stream disabled, and openrouter:bash tool", func() {
			gen := s.generator()
			_, _ = gen.Generate(context.Background(), "html", "make a page")

			Expect(s.captured[0].method).To(Equal(http.MethodPost))
			Expect(s.captured[0].path).To(Equal("/messages"))
			Expect(s.captured[0].contentType).To(Equal("application/json"))

			var req messagesRequest
			Expect(json.Unmarshal(s.captured[0].body, &req)).To(Succeed())
			Expect(req.Model).To(Equal("test/model"))
			Expect(req.MaxTokens).To(Equal(8192))
			Expect(req.Stream).To(BeFalse())
			Expect(req.Tools).To(HaveLen(1))
			Expect(req.Tools[0].Type).To(Equal("openrouter:bash"))

			Expect(req.System).To(ContainSubstring("ONLY"))
			Expect(req.System).To(ContainSubstring("code fences"))
			Expect(req.System).To(ContainSubstring("html"))

			Expect(req.Messages).To(HaveLen(1))
			Expect(req.Messages[0].Role).To(Equal("user"))
		})
	})
	
	Describe("tool execution loop", func() {
		BeforeEach(func() {
			s.bodies = []string{
				`{"content":[{"type":"tool_use","id":"call_1","name":"bash","input":{"command":"echo hello"}}],"stop_reason":"tool_use"}`,
				`{"content":[{"type":"text","text":"result of echo"}],"stop_reason":"end_turn"}`,
			}
		})
		
		It("executes the bash command and passes tool_result back", func() {
			gen := s.generator()
			out, err := gen.Generate(context.Background(), "html", "make a page")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("result of echo"))
			
			Expect(s.captured).To(HaveLen(2))
			
			var req1 messagesRequest
			Expect(json.Unmarshal(s.captured[0].body, &req1)).To(Succeed())
			Expect(req1.Messages).To(HaveLen(1)) // initial user prompt
			
			var req2 messagesRequest
			Expect(json.Unmarshal(s.captured[1].body, &req2)).To(Succeed())
			Expect(req2.Messages).To(HaveLen(3)) // initial user prompt, assistant tool_use, user tool_result
			
			Expect(req2.Messages[1].Role).To(Equal("assistant"))
			Expect(req2.Messages[2].Role).To(Equal("user"))
			
			// Verify tool_result content block
			contentRaw, err := json.Marshal(req2.Messages[2].Content)
			Expect(err).NotTo(HaveOccurred())
			var blocks []contentBlock
			Expect(json.Unmarshal(contentRaw, &blocks)).To(Succeed())
			Expect(blocks).To(HaveLen(1))
			Expect(blocks[0].Type).To(Equal("tool_result"))
			Expect(blocks[0].ToolUseID).To(Equal("call_1"))
			Expect(blocks[0].Content).To(Equal("hello\n"))
		})
		
		It("exits with an error if it hits 20 iterations without a text block", func() {
			s.bodies = make([]string, 25)
			for i := range s.bodies {
				s.bodies[i] = `{"content":[{"type":"tool_use","id":"call_1","name":"bash","input":{"command":"echo hello"}}],"stop_reason":"tool_use"}`
			}
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "make a page")
			Expect(err).To(MatchError(ContainSubstring("maximum 20 iterations")))
			Expect(s.captured).To(HaveLen(20))
		})
	})

	Describe("content block selection", func() {
		BeforeEach(func() {
			// A reasoning model may emit a `thinking` block before the `text`
			// block. Reading the first block blindly would yield empty content.
			s.bodies = []string{`{"content":[{"type":"thinking","thinking":"let me think","signature":"sig"},{"type":"text","text":"<h1>hi</h1>"}],"stop_reason":"end_turn"}`}
		})

		It("skips non-text blocks and returns the first text-typed block's content", func() {
			gen := s.generator()
			out, err := gen.Generate(context.Background(), "html", "make a page")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("<h1>hi</h1>"))
		})
	})

	Describe("content block failures", func() {
		It("fails when no content block is typed text and no tools are used", func() {
			s.bodies = []string{`{"content":[{"type":"thinking","thinking":"let me think","signature":"sig"}],"stop_reason":"end_turn"}`}
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("llm returned no text content block and no tool calls")))
		})
	})

	Describe("configuration failures (before any network call)", func() {
		It("fails when OPENROUTER_API_KEY is not set", func() {
			gen := newLLMGenerator(config{apiKey: "", baseURL: s.URL, model: "test/model"})
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("OPENROUTER_API_KEY")))
		})

		It("fails when OPENROUTER_MODEL is not set", func() {
			gen := newLLMGenerator(config{apiKey: "test-key", baseURL: s.URL, model: ""})
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("OPENROUTER_MODEL")))
		})

		It("does not contact the API when configuration is missing", func() {
			gen := newLLMGenerator(config{apiKey: "", baseURL: s.URL, model: "test/model"})
			_, _ = gen.Generate(context.Background(), "html", "x")
			Expect(s.captured).To(HaveLen(0))
		})
	})

	Describe("response failures", func() {
		It("returns an error on a non-2xx status", func() {
			s.status = http.StatusBadGateway
			s.bodies = []string{`{"error":"upstream"}`}
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("502")))
		})

		It("returns an error when the content list is empty", func() {
			s.bodies = []string{`{"content":[]}`}
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("no content")))
		})

		It("returns an error when the response body cannot be decoded", func() {
			s.bodies = []string{`not json`}
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error on a transport failure", func() {
			dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			dead.Close()
			gen := newLLMGenerator(config{apiKey: "test-key", baseURL: dead.URL, model: "test/model"})
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("transport", func() {
		It("uses an HTTP client with a 4-minute timeout", func() {
			gen := s.generator()
			Expect(gen.client.Timeout).To(Equal(4 * time.Minute))
		})
	})
})
