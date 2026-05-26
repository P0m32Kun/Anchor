package worker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTaskOutput_incremental(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0640); err != nil {
		t.Fatal(err)
	}

	chunk, off, done, err := ReadTaskOutput(dir, "stdout", 0)
	if err != nil {
		t.Fatal(err)
	}
	if chunk != "hello world" {
		t.Fatalf("chunk = %q", chunk)
	}
	if off != 11 {
		t.Fatalf("offset = %d", off)
	}
	if !done {
		t.Fatal("expected done at EOF when not running")
	}

	chunk2, off2, _, err := ReadTaskOutput(dir, "stdout", off)
	if err != nil {
		t.Fatal(err)
	}
	if chunk2 != "" || off2 != 11 {
		t.Fatalf("expected no more data, got %q off=%d", chunk2, off2)
	}
}

func TestTaskOutputWriter_writesFile(t *testing.T) {
	dir := t.TempDir()
	w, err := newTaskOutputWriter(dir, "stdout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("line1\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()

	data, err := os.ReadFile(filepath.Join(dir, "stdout.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\n" {
		t.Fatalf("file = %q", data)
	}
}
