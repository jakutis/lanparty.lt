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

// chatServer is a test double for the OpenRouter chat completions API. It
// records the incoming request and replies with a configurable status code
// and body, so each spec can exercise a different response condition.
type chatServer struct {
	*httptest.Server
	captured capturedRequest
	status   int
	body     string
}

func newChatServer() *chatServer {
	cs := &chatServer{status: http.StatusOK}
	cs.Server = httptest.NewServer(http.HandlerFunc(cs.handle))
	return cs
}

func (cs *chatServer) handle(w http.ResponseWriter, r *http.Request) {
	cs.captured.method = r.Method
	cs.captured.path = r.URL.Path
	cs.captured.auth = r.Header.Get("Authorization")
	cs.captured.contentType = r.Header.Get("Content-Type")
	cs.captured.body, _ = io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cs.status)
	_, _ = io.WriteString(w, cs.body)
}

// generator returns an llmGenerator pointed at this test server.
func (cs *chatServer) generator() *llmGenerator {
	return newLLMGenerator(config{
		apiKey:  "test-key",
		baseURL: cs.URL,
		model:   "test/model",
	})
}

var _ = Describe("llmGenerator", func() {
	var cs *chatServer

	BeforeEach(func() {
		cs = newChatServer()
		DeferCleanup(func() { cs.Close() })
	})

	Describe("happy path", func() {
		BeforeEach(func() {
			cs.body = `{"choices":[{"message":{"role":"assistant","content":"  <h1>hi</h1>  "},"finish_reason":"stop"}]}`
		})

		It("returns the first choice's content with surrounding whitespace trimmed", func() {
			gen := cs.generator()
			out, err := gen.Generate(context.Background(), "html", "make a page")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("<h1>hi</h1>"))
		})

		It("sends a bearer token derived from the configured api key", func() {
			gen := cs.generator()
			_, _ = gen.Generate(context.Background(), "html", "x")
			Expect(cs.captured.auth).To(Equal("Bearer test-key"))
		})

		It("posts a JSON body to {baseURL}/chat/completions with two messages and stream disabled", func() {
			gen := cs.generator()
			_, _ = gen.Generate(context.Background(), "html", "make a page")

			Expect(cs.captured.method).To(Equal(http.MethodPost))
			Expect(cs.captured.path).To(Equal("/chat/completions"))
			Expect(cs.captured.contentType).To(Equal("application/json"))

			var req chatCompletionsRequest
			Expect(json.Unmarshal(cs.captured.body, &req)).To(Succeed())
			Expect(req.Model).To(Equal("test/model"))
			Expect(req.Stream).To(BeFalse())
			Expect(req.Messages).To(HaveLen(2))

			// System message: raw file content only, no commentary, no code
			// fences, and it names the requested file type.
			Expect(req.Messages[0].Role).To(Equal("system"))
			Expect(req.Messages[0].Content).To(ContainSubstring("ONLY"))
			Expect(req.Messages[0].Content).To(ContainSubstring("code fences"))
			Expect(req.Messages[0].Content).To(ContainSubstring("html"))

			// User message: restates the file type and carries the spec.
			Expect(req.Messages[1].Role).To(Equal("user"))
			Expect(req.Messages[1].Content).To(ContainSubstring("html"))
			Expect(req.Messages[1].Content).To(ContainSubstring("make a page"))
		})
	})

	Describe("configuration failures (before any network call)", func() {
		It("fails when OPENROUTER_API_KEY is not set", func() {
			gen := newLLMGenerator(config{apiKey: "", baseURL: cs.URL, model: "test/model"})
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("OPENROUTER_API_KEY")))
		})

		It("fails when OPENROUTER_MODEL is not set", func() {
			gen := newLLMGenerator(config{apiKey: "test-key", baseURL: cs.URL, model: ""})
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("OPENROUTER_MODEL")))
		})

		It("does not contact the API when configuration is missing", func() {
			gen := newLLMGenerator(config{apiKey: "", baseURL: cs.URL, model: "test/model"})
			_, _ = gen.Generate(context.Background(), "html", "x")
			Expect(cs.captured).To(BeZero())
		})
	})

	Describe("response failures", func() {
		It("returns an error on a non-2xx status", func() {
			cs.status = http.StatusBadGateway
			cs.body = `{"error":"upstream"}`
			gen := cs.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("502")))
		})

		It("returns an error when the choices list is empty", func() {
			cs.body = `{"choices":[]}`
			gen := cs.generator()
			_, err := gen.Generate(context.Background(), "html", "x")
			Expect(err).To(MatchError(ContainSubstring("no choices")))
		})

		It("returns an error when the response body cannot be decoded", func() {
			cs.body = `not json`
			gen := cs.generator()
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
			gen := cs.generator()
			Expect(gen.client.Timeout).To(Equal(4 * time.Minute))
		})
	})
})
