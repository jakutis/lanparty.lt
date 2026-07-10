package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port), nil
}

func TestAPIEndpointsWithK6(t *testing.T) {
	// 1. Compile the main binary
	binPath := "./testapi.bin"
	buildCmd := exec.Command("go", "build", "-C", "../implementation", "-o", "../verification/testapi.bin", "./src")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build API binary: %v\n%s", err, out)
	}
	defer os.Remove(binPath)

	// 2. Start the fake OpenRouter server
	fakeOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		// Check if we should fail generation
		if messages, ok := req["messages"].([]interface{}); ok {
			for _, msgObj := range messages {
				msg := msgObj.(map[string]interface{})
				if content, ok := msg["content"].(string); ok {
					if strings.Contains(content, "fail_generation") {
						w.WriteHeader(http.StatusInternalServerError)
						io.WriteString(w, `{"error":"upstream"}`)
						return
					}
				}
			}
		}

		// Happy path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"content":[{"type":"text","text":"<h1>hi</h1>"}],"stop_reason":"end_turn"}`)
	}))
	defer fakeOpenRouter.Close()

	// 3. Find a free port for the API server
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// 4. Start the API binary
	apiCmd := exec.Command(binPath)
	apiCmd.Env = append(os.Environ(),
		"PORT="+port,
		"OPENROUTER_API_KEY=test-key",
		"OPENROUTER_MODEL=test/model",
		"OPENROUTER_BASE_URL="+fakeOpenRouter.URL,
	)
	
	// Start the API process
	if err := apiCmd.Start(); err != nil {
		t.Fatalf("failed to start API binary: %v", err)
	}
	defer func() {
		apiCmd.Process.Kill()
		apiCmd.Wait()
	}()

	// Wait a moment for the server to start listening
	apiURL := "http://localhost:" + port
	
	// Quick health check loop
	ready := false
	for i := 0; i < 20; i++ {
		resp, err := http.Get(apiURL + "/v1/representation") // It will return 405 Method Not Allowed, but that proves it's up
		if err == nil {
			resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("API server did not start in time")
	}

	// 5. Find k6 binary
	k6Path, err := exec.LookPath("k6")
	if err != nil {
		home, _ := os.UserHomeDir()
		fallback := home + "/go/bin/k6"
		if _, err := os.Stat(fallback); err == nil {
			k6Path = fallback
		} else {
			t.Fatalf("k6 is not installed or not in PATH: %v", err)
		}
	}

	// 6. Run the k6 script
	cmd := exec.Command(k6Path, "run", "k6/api_test.js")
	cmd.Env = append(os.Environ(), "API_URL="+apiURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("k6 test failed: %v", err)
	}
}
