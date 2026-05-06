//go:build integration

package custom

import (
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/safefs"
	_ "github.com/mattn/go-sqlite3"
)

const realRepoURL = "https://github.com/RBKD-SEC/templates"

// TestExecCloner_RealClone is gated behind the `integration` build tag and
// exercises the real `git` binary against a small public repository.
//
// Run with:  go test -tags=integration ./internal/nuclei/custom/...
func TestExecCloner_RealClone(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "repo")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := ExecCloner{}
	if err := c.Clone(ctx, realRepoURL, "main", dest); err != nil {
		t.Fatalf("clone: %v", err)
	}
}

// TestPhase1_EndToEnd_RealRepo simulates the full Phase 1 flow against a real
// public templates repo: ingest → ready, walk files, classify by allow-list.
//
// Run with:  go test -tags=integration -run Phase1_EndToEnd ./internal/nuclei/custom/...
func TestPhase1_EndToEnd_RealRepo(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	defer rawDB.Close()
	if err := db.Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dataDir := t.TempDir()
	m := NewManager(db.New(rawDB), rawDB, dataDir, ExecCloner{})
	if err := m.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	src, err := m.CreateFromGit(ctx, "rbkd-sec-templates", realRepoURL, "main", "manual")
	if err != nil {
		t.Fatalf("CreateFromGit: %v", err)
	}
	if src.Status != "ready" {
		t.Fatalf("expected status ready, got %q", src.Status)
	}
	t.Logf("ingest ok: id=%s status=%s last_sync_at=%v", src.ID, src.Status, src.LastSyncAt)

	files, err := m.Layout().WalkFiles(src.ID)
	if err != nil {
		t.Fatalf("walk files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("walk returned 0 files")
	}

	type bucket struct {
		count int
		bytes int64
	}
	all := bucket{}
	yamlBucket := bucket{}
	payloadsBucket := bucket{}
	disallowed := bucket{}
	dirCounts := map[string]int{}
	var disallowedSamples []string

	for _, f := range files {
		all.count++
		all.bytes += f.Size
		segs := strings.Split(f.Path, "/")
		if len(segs) > 0 {
			dirCounts[segs[0]]++
		}
		switch {
		case safefs.IsAllowedTemplateFile(f.Path):
			if strings.HasSuffix(strings.ToLower(f.Path), ".yaml") || strings.HasSuffix(strings.ToLower(f.Path), ".yml") {
				yamlBucket.count++
				yamlBucket.bytes += f.Size
			} else {
				payloadsBucket.count++
				payloadsBucket.bytes += f.Size
			}
		default:
			disallowed.count++
			disallowed.bytes += f.Size
			if len(disallowedSamples) < 8 {
				disallowedSamples = append(disallowedSamples, f.Path)
			}
		}
	}

	t.Logf("== walk summary ==")
	t.Logf("total files: %d (%d bytes)", all.count, all.bytes)
	t.Logf("yaml/yml templates: %d (%d bytes)", yamlBucket.count, yamlBucket.bytes)
	t.Logf("payloads/* (any ext): %d (%d bytes)", payloadsBucket.count, payloadsBucket.bytes)
	t.Logf("disallowed (not exposed via file CRUD): %d (%d bytes)", disallowed.count, disallowed.bytes)

	type kv struct {
		dir   string
		count int
	}
	var dirs []kv
	for k, v := range dirCounts {
		dirs = append(dirs, kv{k, v})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].count > dirs[j].count })
	t.Logf("== top-level directories ==")
	for _, d := range dirs {
		t.Logf("  %-24s %d files", d.dir, d.count)
	}

	if len(disallowedSamples) > 0 {
		t.Logf("== disallowed samples (clone keeps them, file API hides them) ==")
		for _, p := range disallowedSamples {
			t.Logf("  %s", p)
		}
	}

	if yamlBucket.count == 0 {
		t.Error("expected at least one yaml template; allow-list may be misconfigured")
	}

	// Spot-check: read a known template file end-to-end through the Manager.
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".yaml") {
			data, err := m.ReadFile(src.ID, f.Path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", f.Path, err)
			}
			if len(data) == 0 {
				t.Errorf("ReadFile %s returned empty body", f.Path)
			} else {
				t.Logf("read sample %s: %d bytes", f.Path, len(data))
			}
			break
		}
	}

	// Spot-check: a payloads/* file should be readable too if the repo has any.
	for _, f := range files {
		if strings.HasPrefix(f.Path, "payloads/") {
			data, err := m.ReadFile(src.ID, f.Path)
			if err != nil {
				t.Errorf("ReadFile payloads %s: %v", f.Path, err)
			} else {
				t.Logf("read payload %s: %d bytes", f.Path, len(data))
			}
			break
		}
	}

	// Spot-check: README.md must be hidden from the file API even though it
	// exists on disk.
	if _, err := m.ReadFile(src.ID, "README.md"); err == nil {
		t.Error("README.md should not be readable via file API")
	}

	// Refresh round-trip: same repo, expect status ready again.
	if _, err := m.Refresh(ctx, src.ID); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	t.Logf("refresh ok: id=%s", src.ID)
}
