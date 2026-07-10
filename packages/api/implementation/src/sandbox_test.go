package main

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// toolUseBody builds a mock model response asking to run the given command.
func toolUseBody(command string) string {
	quoted, err := json.Marshal(command)
	Expect(err).NotTo(HaveOccurred())
	return `{"content":[{"type":"tool_use","id":"call_1","name":"bash","input":{"command":` +
		string(quoted) + `}}],"stop_reason":"tool_use"}`
}

const textDoneBody = `{"content":[{"type":"text","text":"done"}],"stop_reason":"end_turn"}`

// lastToolResult decodes the tool_result block from the final user message of
// a captured upstream request body.
func lastToolResult(body []byte) contentBlock {
	var req messagesRequest
	Expect(json.Unmarshal(body, &req)).To(Succeed())
	Expect(req.Messages).NotTo(BeEmpty())

	contentRaw, err := json.Marshal(req.Messages[len(req.Messages)-1].Content)
	Expect(err).NotTo(HaveOccurred())
	var blocks []contentBlock
	Expect(json.Unmarshal(contentRaw, &blocks)).To(Succeed())
	Expect(blocks).To(HaveLen(1))
	Expect(blocks[0].Type).To(Equal("tool_result"))
	return blocks[0]
}

var _ = Describe("command sandbox", func() {
	var s *apiServer

	BeforeEach(func() {
		s = newAPIServer()
		DeferCleanup(func() { s.Close() })
	})

	// runCommands drives one generation through the given commands and
	// returns the tool_result each command produced.
	runCommands := func(commands ...string) []contentBlock {
		s.bodies = nil
		for _, c := range commands {
			s.bodies = append(s.bodies, toolUseBody(c))
		}
		s.bodies = append(s.bodies, textDoneBody)

		before := len(s.captured)
		gen := s.generator()
		out, err := gen.Generate(context.Background(), "html", "x")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(out)).To(Equal("done"))

		var results []contentBlock
		for i := before + 1; i < len(s.captured); i++ {
			results = append(results, lastToolResult(s.captured[i].body))
		}
		Expect(results).To(HaveLen(len(commands)))
		return results
	}

	It("hides the server's environment from commands", func() {
		os.Setenv("OPENROUTER_API_KEY", "sekrit-key-value")
		DeferCleanup(func() { os.Unsetenv("OPENROUTER_API_KEY") })

		res := runCommands("env")[0]
		Expect(res.IsError).To(BeFalse())
		Expect(res.Content).To(ContainSubstring("HOME=/work"))
		Expect(res.Content).NotTo(ContainSubstring("sekrit-key-value"))
		Expect(res.Content).NotTo(ContainSubstring("OPENROUTER"))
	})

	It("denies network access, even to addresses the server can reach", func() {
		u, err := url.Parse(s.URL)
		Expect(err).NotTo(HaveOccurred())

		res := runCommands(
			`(exec 3<>/dev/tcp/` + u.Hostname() + `/` + u.Port() +
				`) 2>/dev/null && echo connected || echo no-network`)[0]
		Expect(res.Content).To(Equal("no-network\n"))
	})

	It("runs commands with /work as the working directory", func() {
		res := runCommands("pwd")[0]
		Expect(res.IsError).To(BeFalse())
		Expect(res.Content).To(Equal("/work\n"))
	})

	It("persists files across commands within one generation", func() {
		results := runCommands("echo persisted > f.txt", "cat f.txt")
		Expect(results[1].IsError).To(BeFalse())
		Expect(results[1].Content).To(Equal("persisted\n"))
	})

	It("isolates generations from each other", func() {
		runCommands("echo persisted > f.txt")

		res := runCommands("cat f.txt")[0]
		Expect(res.IsError).To(BeTrue())
		Expect(res.Content).To(ContainSubstring("No such file"))
	})

	It("removes the scratch directory when generation ends", func() {
		runCommands("echo persisted > f.txt")

		entries, err := os.ReadDir(os.TempDir())
		Expect(err).NotTo(HaveOccurred())
		for _, e := range entries {
			Expect(e.Name()).NotTo(HavePrefix("api-sandbox-"))
		}
	})

	It("mounts the system read-only", func() {
		res := runCommands("touch /usr/x")[0]
		Expect(res.IsError).To(BeTrue())
		Expect(res.Content).To(ContainSubstring("Read-only file system"))
	})

	It("kills commands that exceed the time limit and keeps their output", func() {
		s.bodies = []string{toolUseBody("echo started; sleep 30"), textDoneBody}

		gen := s.generator()
		gen.cmdTimeout = time.Second

		start := time.Now()
		out, err := gen.Generate(context.Background(), "html", "x")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(out)).To(Equal("done"))
		Expect(time.Since(start)).To(BeNumerically("<", 15*time.Second))

		res := lastToolResult(s.captured[1].body)
		Expect(res.IsError).To(BeTrue())
		Expect(res.Content).To(ContainSubstring("started"))
	})

	It("truncates combined output at 64 KiB", func() {
		res := runCommands("printf 'a%.0s' {1..70000}")[0]
		Expect(res.IsError).To(BeFalse())
		Expect(res.Content).To(HaveLen(64*1024 + len("\n[output truncated]")))
		Expect(res.Content).To(HavePrefix(strings.Repeat("a", 64*1024)))
		Expect(res.Content).To(HaveSuffix("\n[output truncated]"))
	})

	It("does not wait for background processes", func() {
		start := time.Now()
		res := runCommands("sleep 30 & echo bg-done")[0]
		Expect(res.IsError).To(BeFalse())
		Expect(res.Content).To(Equal("bg-done\n"))
		Expect(time.Since(start)).To(BeNumerically("<", 15*time.Second))
	})
})
