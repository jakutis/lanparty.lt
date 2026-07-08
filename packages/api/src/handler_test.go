package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// stubGenerator is a deterministic Generator used for testing the HTTP layer.
type stubGenerator struct {
	content []byte
	err     error
}

func (s stubGenerator) Generate(_ context.Context, _, _ string) ([]byte, error) {
	return s.content, s.err
}

// newTestServer spins up a local, isolated HTTP server in memory backed by the
// API routes, and records HTTP responses without touching the real network.
func newTestServer(gen Generator) *httptest.Server {
	mux := http.NewServeMux()
	registerRoutes(mux, gen)
	return httptest.NewServer(mux)
}

var _ = Describe("POST /v1/representation", func() {
	var (
		srv *httptest.Server
		gen *stubGenerator
	)

	BeforeEach(func() {
		gen = &stubGenerator{}
		srv = newTestServer(gen)
		DeferCleanup(func() { srv.Close() })
	})

	Describe("happy path", func() {
		BeforeEach(func() {
			gen.content = []byte("<h1>hi</h1>")
		})

		It("returns the generated file with the right content type and filename", func() {
			body := `{"type":"html","spec":"a greeting"}`
			resp, err := http.Post(srv.URL+"/v1/representation", "application/json", strings.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
			Expect(resp.Header.Get("Content-Disposition")).To(Equal(`attachment; filename="representation.html"`))

			b, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("<h1>hi</h1>"))
		})

		It("serves markdown as text/markdown with a .md filename", func() {
			body := `{"type":"markdown","spec":"a greeting"}`
			resp, err := http.Post(srv.URL+"/v1/representation", "application/json", strings.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal("text/markdown; charset=utf-8"))
			Expect(resp.Header.Get("Content-Disposition")).To(Equal(`attachment; filename="representation.md"`))
		})
	})

	Describe("request validation", func() {
		It("rejects requests with missing fields", func() {
			resp, err := http.Post(srv.URL+"/v1/representation", "application/json", strings.NewReader(`{"type":"html"}`))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusUnprocessableEntity))
		})

		It("rejects unsupported types without invoking the generator", func() {
			resp, err := http.Post(srv.URL+"/v1/representation", "application/json",
				strings.NewReader(`{"type":"json","spec":"a config file"}`))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusUnprocessableEntity))

			var e map[string]string
			Expect(json.NewDecoder(resp.Body).Decode(&e)).To(Succeed())
			Expect(e["error"]).To(ContainSubstring("unsupported type"))
		})

		It("rejects malformed JSON bodies with a JSON error entity", func() {
			resp, err := http.Post(srv.URL+"/v1/representation", "application/json", strings.NewReader(`{not json`))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			var e map[string]string
			Expect(json.NewDecoder(resp.Body).Decode(&e)).To(Succeed())
			Expect(e["error"]).NotTo(BeEmpty())
		})
	})

	It("rejects non-POST methods", func() {
		resp, err := http.Get(srv.URL + "/v1/representation")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
	})

	It("surfaces generation failures as 500 responses", func() {
		gen.err = errBoom{}

		resp, err := http.Post(srv.URL+"/v1/representation", "application/json",
			strings.NewReader(`{"type":"html","spec":"x"}`))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
	})
})

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
