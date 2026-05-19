package builtin

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SyncAll clones or updates dict, templates, and finger repos per LoadConfig.
// When sync is disabled (ANCHOR_BUILTIN_SYNC=off), it returns nil immediately.
// Per-repo failures are logged and joined; callers may log the returned error
// without treating it as fatal (fail-soft).
func SyncAll() error {
	cfg := LoadConfig()
	if !cfg.ShouldSync() {
		return nil
	}

	var errs []error

	if err := syncRepo(cfg.DictRepo, cfg.DictRef, cfg.DictRoot); err != nil {
		log.Printf("[builtin] dict sync: %v", err)
		errs = append(errs, err)
	}
	if err := syncRepo(cfg.TemplatesRepo, cfg.TemplatesRef, cfg.TemplatesRoot); err != nil {
		log.Printf("[builtin] templates sync: %v", err)
		errs = append(errs, err)
	}
	if err := syncRepo(cfg.FingerRepo, cfg.FingerRef, cfg.FingerRoot); err != nil {
		log.Printf("[builtin] finger sync: %v", err)
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func syncRepo(repo, ref, dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return exec.Command("git", "clone", "--depth", "1", "--branch", ref, repo, dir).Run()
	}

	cmds := [][]string{
		{"git", "-C", dir, "fetch", "origin"},
		{"git", "-C", dir, "checkout", ref},
		{"git", "-C", dir, "pull", "--ff-only"},
	}
	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}

// HeadShort returns the short commit hash at dir, or "" if rev-parse fails.
func HeadShort(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
