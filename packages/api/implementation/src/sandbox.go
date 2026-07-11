package main

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// Command sandbox parameters; see
// specification/main.md#command-sandbox.
const (
	commandTimeout   = 60 * time.Second
	maxCommandOutput = 64 * 1024
	truncationMarker = "\n[output truncated]"
)

// sandbox executes the tool commands of a single generation. All commands of
// the generation share one scratch directory, created lazily on the first
// command and mounted read-write at /work inside the sandbox.
type sandbox struct {
	timeout time.Duration
	workDir string
}

func newSandbox(timeout time.Duration) *sandbox {
	return &sandbox{timeout: timeout}
}

// run executes command with `bash -c` inside bubblewrap and returns the
// combined standard output and standard error, truncated to maxCommandOutput
// bytes. A command still running when the timeout elapses is killed; whatever
// output it produced so far is returned along with the error.
func (s *sandbox) run(ctx context.Context, command string) (string, error) {
	if s.workDir == "" {
		dir, err := os.MkdirTemp("", "api-sandbox-")
		if err != nil {
			return "", err
		}
		s.workDir = dir
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bwrap",
		"--unshare-all",
		"--die-with-parent",
		"--ro-bind", "/usr", "/usr",
		"--symlink", "usr/bin", "/bin",
		"--symlink", "usr/sbin", "/sbin",
		"--symlink", "usr/lib", "/lib",
		"--symlink", "usr/lib64", "/lib64",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--bind", s.workDir, "/work",
		"--chdir", "/work",
		"--clearenv",
		"--setenv", "PATH", "/usr/bin:/bin",
		"--setenv", "HOME", "/work",
		"--setenv", "TMPDIR", "/tmp",
		"/bin/bash", "-c", command,
	)
	cmd.WaitDelay = 10 * time.Second

	out, err := cmd.CombinedOutput()
	if len(out) > maxCommandOutput {
		out = append(out[:maxCommandOutput], truncationMarker...)
	}
	return string(out), err
}

// close deletes the scratch directory and everything in it.
func (s *sandbox) close() {
	if s.workDir != "" {
		os.RemoveAll(s.workDir)
		s.workDir = ""
	}
}
