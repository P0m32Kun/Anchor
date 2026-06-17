package pool

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPool_Dedup(t *testing.T) {
	dir := t.TempDir()
	var flushes int
	p := New(Config{BatchSize: 10, FlushTimeout: time.Hour, DataDir: dir, FilePrefix: "t", Label: "test"}, func(FlushEvent) {
		flushes++
	})

	if !p.Add(Member{Value: "a.example.com", AssetID: "1"}) {
		t.Fatal("expected first add")
	}
	if p.Add(Member{Value: "a.example.com", AssetID: "2"}) {
		t.Fatal("expected duplicate rejected")
	}
	if flushes != 0 {
		t.Fatalf("flushes = %d, want 0", flushes)
	}
}

func TestPool_FlushOnBatchSize(t *testing.T) {
	dir := t.TempDir()
	var last FlushEvent
	p := New(Config{BatchSize: 2, FlushTimeout: time.Hour, DataDir: dir, FilePrefix: "t", Label: "test"}, func(ev FlushEvent) {
		last = ev
	})

	p.Add(Member{Value: "a.example.com", AssetID: "1"})
	p.Add(Member{Value: "b.example.com", AssetID: "2"})

	if len(last.Members) != 2 {
		t.Fatalf("members = %d, want 2", len(last.Members))
	}
	if _, err := os.Stat(last.FilePath); err != nil {
		t.Fatalf("batch file: %v", err)
	}
}

func TestPool_FlushOnTimeout(t *testing.T) {
	dir := t.TempDir()
	done := make(chan struct{}, 1)
	p := New(Config{BatchSize: 100, FlushTimeout: 50 * time.Millisecond, DataDir: dir, FilePrefix: "t", Label: "test"}, func(FlushEvent) {
		done <- struct{}{}
	})
	p.Start()
	defer p.Stop()

	p.Add(Member{Value: "solo.example.com", AssetID: "1"})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for flush")
	}
}

func TestReadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	if err := os.WriteFile(path, []byte("one\n\n two \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := ReadLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "one" || lines[1] != "two" {
		t.Fatalf("lines = %#v", lines)
	}
}
