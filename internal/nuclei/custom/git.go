package custom

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/toolguard"
)

// Cloner abstracts the git ingest path so the Manager can be tested without
// shelling out. ExecCloner is the production implementation; tests inject a
// fake.
type Cloner interface {
	Clone(ctx context.Context, url, branch, dest string) error
}

// stderrCap caps the stderr buffer surfaced in clone errors. Enough to see a
// reasonable error message without flooding logs or response bodies.
const stderrCap = 512

// ExecCloner shells out to the `git` binary. Phase 1 only supports HTTPS URLs;
// SSH or token-based ingest is intentionally out of scope.
type ExecCloner struct {
	// Bin is the path to the git executable. Defaults to "git" via PATH when empty.
	Bin string
}

// Clone runs `git clone --depth 1 --single-branch [--branch <branch>] <url> <dest>`.
// dest must not exist; git will create it. URL must use the https:// scheme.
func (c ExecCloner) Clone(ctx context.Context, url, branch, dest string) error {
	if err := validateGitURL(url); err != nil {
		return err
	}
	if dest == "" {
		return errors.New("custom: clone destination is empty")
	}

	bin := c.Bin
	if bin == "" {
		bin = "git"
	}

	args := []string{"clone", "--depth", "1", "--single-branch"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dest)

	// Allowlist: reject unknown binaries or args with shell metacharacters.
	if err := toolguard.NewAllowlist().Validate(bin, args); err != nil {
		return fmt.Errorf("custom: allowlist rejected git clone: %w", err)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &capWriter{w: &stderr, cap: stderrCap}
	cmd.Stdout = io.Discard

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("custom: git clone: %w", err)
		}
		return fmt.Errorf("custom: git clone: %w: %s", err, msg)
	}
	return nil
}

// validateGitURL enforces HTTPS-only ingest.
func validateGitURL(url string) error {
	if url == "" {
		return errors.New("custom: git url is empty")
	}
	low := strings.ToLower(url)
	if !strings.HasPrefix(low, "https://") {
		return errors.New("custom: git url must use https:// scheme")
	}
	return nil
}

// capWriter writes up to cap bytes into w then silently drops further writes.
// Used to bound the stderr we surface in clone errors.
type capWriter struct {
	w   *bytes.Buffer
	cap int
}

func (c *capWriter) Write(p []byte) (int, error) {
	if c.w.Len() >= c.cap {
		return len(p), nil
	}
	room := c.cap - c.w.Len()
	if len(p) > room {
		c.w.Write(p[:room])
	} else {
		c.w.Write(p)
	}
	return len(p), nil
}
