package e2e_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestGoSourcesAreFormatted verifies that every Go source file in the api
// package — implementation and verification — is gofmt-formatted.
func TestGoSourcesAreFormatted(t *testing.T) {
	out, err := exec.Command("gofmt", "-l", "../implementation", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt failed: %v\n%s", err, out)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		t.Fatalf("Go files are not gofmt-formatted (run gofmt -w):\n%s", files)
	}
}
