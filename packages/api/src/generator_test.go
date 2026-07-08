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
	captured capturedRequest
	status   int
	body     string
}

func newAPIServer() *apiServer {
	s := &apiServer{status: http.StatusOK}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *apiServer) handle(w http.ResponseWriter, r *http.Request) {
	s.captured.method = r.Method
	s.captured.path = r.URL.Path
	s.captured.auth = r.Header.Get("Authorization")
	s.captured.contentType = r.Header.Get("Content-Type")
	s.captured.body, _ = io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.status)
	_, _ = io.WriteString(w, s.body)
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
			s.body = `{"content":[{"type":"text","text":"  <h1>hi</h1>  "}],"stop_reason":"end_turn"}`
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
			Expect(s.captured.auth).To(Equal("Bearer test-key"))
		})

		It("posts a JSON body to {baseURL}/messages with model, max_tokens, system and a single user message, stream disabled", func() {
			gen := s.generator()
			_, _ = gen.Generate(context.Background(), "html", "make a page")

			Expect(s.captured.method).To(Equal(http.MethodPost))
			Expect(s.captured.path).To(Equal("/messages"))
			Expect(s.captured.contentType).To(Equal("application/json"))

			var req messagesRequest
			Expect(json.Unmarshal(s.captured.body, &req)).To(Succeed())
			Expect(req.Model).To(Equal("test/model"))
			Expect(req.MaxTokens).To(Equal(8192))
			Expect(req.Stream).To(BeFalse())

			// Top-level system field: raw file content only, no commentary, no
			// code fences, and it names the requested file type.
			Expect(req.System).To(ContainSubstring("ONLY"))
			Expect(req.System).To(ContainSubstring("code fences"))
			Expect(req.System).To(ContainSubstring("html"))

			// A single user message that restates the file type and carries
			// the spec.
			Expect(req.Messages).To(HaveLen(1))
			Expect(req.Messages[0].Role).To(Equal("user"))
			Expect(req.Messages[0].Content).To(ContainSubstring("html"))
			Expect(req.Messages[0].Content).To(ContainSubstring("make a page"))
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
			Expect(s.captured).To(BeZero())
		})
	})

	Describe("response failures", func() {
		It("returns an error on a non-2xx status", func() {
			s.status = http.StatusBadGateway
			s.body = `{"error":"upstream"}`
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("502")))
		})

		It("returns an error when the content list is empty", func() {
			s.body = `{"content":[]}`
			gen := s.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("no content")))
		})

		It("returns an error when the response body cannot be decoded", func() {
			s.body = `not json`
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
