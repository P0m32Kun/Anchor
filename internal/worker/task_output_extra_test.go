package worker

import (
	"os"
	"path/filepath"
	"testing"
)

// --- newTaskOutputWriter ---

func TestNewTaskOutputWriter_createsDirAndFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	w, err := newTaskOutputWriter(dir, "stdout")
	if err != nil {
		t.Fatalf("newTaskOutputWriter: %v", err)
	}
	defer w.Close()

	// Verify file was created.
	path := filepath.Join(dir, "stdout.txt")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}

	// Write some data and verify.
	n, err := w.Write([]byte("test output"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 11 {
		t.Errorf("n = %d, want 11", n)
	}
	if w.Len() != 11 {
		t.Errorf("Len = %d, want 11", w.Len())
	}
}

func TestNewTaskOutputWriter_stderr(t *testing.T) {
	dir := t.TempDir()
	w, err := newTaskOutputWriter(dir, "stderr")
	if err != nil {
		t.Fatalf("newTaskOutputWriter: %v", err)
	}
	defer w.Close()

	path := filepath.Join(dir, "stderr.txt")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
}

func TestTaskOutputWriter_Close_nilFile(t *testing.T) {
	w := &taskOutputWriter{file: nil, buf: &limitedBuffer{}}
	if err := w.Close(); err != nil {
		t.Fatalf("Close with nil file: %v", err)
	}
}

func TestTaskOutputWriter_FilePath(t *testing.T) {
	w := &taskOutputWriter{buf: &limitedBuffer{}}
	got := w.FilePath("/workdir", "stdout")
	want := filepath.Join("/workdir", "stdout.txt")
	if got != want {
		t.Errorf("FilePath = %q, want %q", got, want)
	}
}

// --- CopyFinalOutput ---

func TestCopyFinalOutput_emptyData(t *testing.T) {
	dir := t.TempDir()
	if err := CopyFinalOutput(dir, "stdout", nil); err != nil {
		t.Fatalf("CopyFinalOutput nil: %v", err)
	}
	if err := CopyFinalOutput(dir, "stdout", []byte{}); err != nil {
		t.Fatalf("CopyFinalOutput empty: %v", err)
	}
}

func TestCopyFinalOutput_writesWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	data := []byte("hello world")
	if err := CopyFinalOutput(dir, "stdout", data); err != nil {
		t.Fatalf("CopyFinalOutput: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "stdout.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("content = %q, want 'hello world'", string(got))
	}
}

func TestCopyFinalOutput_skipsWhenFileLarger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")
	os.WriteFile(path, []byte("existing data is longer"), 0640)

	// Shorter data should not overwrite.
	if err := CopyFinalOutput(dir, "stdout", []byte("short")); err != nil {
		t.Fatalf("CopyFinalOutput: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "existing data is longer" {
		t.Errorf("content = %q, want 'existing data is longer'", string(got))
	}
}

// --- ReadTaskOutput additional ---

func TestReadTaskOutput_offsetBeyondSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "stdout.txt"), []byte("short"), 0640)

	content, next, atEOF, err := ReadTaskOutput(dir, "stdout", 1000)
	if err != nil {
		t.Fatalf("ReadTaskOutput: %v", err)
	}
	if content != "" {
		t.Errorf("content = %q, want empty", content)
	}
	if next != 5 {
		t.Errorf("next = %d, want 5 (file size)", next)
	}
	if !atEOF {
		t.Error("expected atEOF=true")
	}
}

func TestReadTaskOutput_stderr(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "stderr.txt"), []byte("error output"), 0640)

	content, _, _, err := ReadTaskOutput(dir, "stderr", 0)
	if err != nil {
		t.Fatalf("ReadTaskOutput: %v", err)
	}
	if content != "error output" {
		t.Errorf("content = %q, want 'error output'", content)
	}
}
