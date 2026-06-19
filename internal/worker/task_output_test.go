package worker

import (
	"os"
	"path/filepath"
	"strings"
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

// --- Bytes / Len ---

func TestTaskOutputWriter_BytesAndLen(t *testing.T) {
	dir := t.TempDir()
	w, err := newTaskOutputWriter(dir, "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Empty state.
	if w.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", w.Len())
	}
	if len(w.Bytes()) != 0 {
		t.Fatalf("Bytes() = %q, want empty", w.Bytes())
	}

	// Write some data.
	payload := []byte("hello world")
	n, err := w.Write(payload)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("Write returned n=%d, want %d", n, len(payload))
	}

	if w.Len() != len(payload) {
		t.Fatalf("Len() = %d, want %d", w.Len(), len(payload))
	}
	if string(w.Bytes()) != string(payload) {
		t.Fatalf("Bytes() = %q, want %q", w.Bytes(), payload)
	}

	// Write more data, verify accumulation.
	n2, err := w.Write([]byte(" extra"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "hello world extra"
	if w.Len() != len(expected) {
		t.Fatalf("Len() = %d, want %d", w.Len(), len(expected))
	}
	if string(w.Bytes()) != expected {
		t.Fatalf("Bytes() = %q, want %q", w.Bytes(), expected)
	}
	_ = n2
}

// --- Truncated ---

func TestTaskOutputWriter_Truncated(t *testing.T) {
	dir := t.TempDir()
	w, err := newTaskOutputWriter(dir, "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Normal data should not trigger truncation.
	if _, err := w.Write([]byte("normal data")); err != nil {
		t.Fatal(err)
	}
	if w.Truncated() {
		t.Fatal("Truncated() = true after small write, want false")
	}
}

func TestTaskOutputWriter_Truncated_limitedBuffer(t *testing.T) {
	// Verify that the underlying limitedBuffer truncates when exceeding maxOutputSize.
	// We test limitedBuffer directly (same package) since maxOutputSize (100 MB) is
	// large but manageable in CI.
	var lb limitedBuffer

	// Fill up to just under the limit.
	chunk := make([]byte, 1024*1024) // 1 MB
	for lb.Len() < maxOutputSize {
		remaining := maxOutputSize - lb.Len()
		if remaining < len(chunk) {
			chunk = chunk[:remaining]
		}
		if _, err := lb.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	if lb.truncated {
		t.Fatal("should not be truncated at exactly maxOutputSize")
	}

	// One more byte should trigger truncation.
	if _, err := lb.Write([]byte("!")); err != nil {
		t.Fatal(err)
	}
	if !lb.truncated {
		t.Fatal("should be truncated after exceeding maxOutputSize")
	}
}

// --- TaskOutputPaths ---

func TestTaskOutputPaths(t *testing.T) {
	workdir := "/some/workdir"
	stdout, stderr := TaskOutputPaths(workdir)

	wantStdout := filepath.Join(workdir, "stdout.txt")
	wantStderr := filepath.Join(workdir, "stderr.txt")

	if stdout != wantStdout {
		t.Fatalf("stdout path = %q, want %q", stdout, wantStdout)
	}
	if stderr != wantStderr {
		t.Fatalf("stderr path = %q, want %q", stderr, wantStderr)
	}
}

// --- CopyFinalOutput ---

func TestCopyFinalOutput_newFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte("final output data")
	if err := CopyFinalOutput(dir, "stdout", data); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "stdout.txt")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("file = %q, want %q", got, data)
	}
}

func TestCopyFinalOutput_existingLarger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")

	// Write a larger file first.
	existing := []byte("existing data that is larger")
	if err := os.WriteFile(path, existing, 0640); err != nil {
		t.Fatal(err)
	}

	// Attempt to copy smaller data — should NOT overwrite.
	smaller := []byte("small")
	if err := CopyFinalOutput(dir, "stdout", smaller); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(existing) {
		t.Fatalf("file was overwritten: got %q, want %q", got, existing)
	}
}

func TestCopyFinalOutput_existingSmaller(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")

	// Write a smaller file first.
	if err := os.WriteFile(path, []byte("small"), 0640); err != nil {
		t.Fatal(err)
	}

	// Copy larger data — should overwrite.
	larger := []byte("larger replacement data")
	if err := CopyFinalOutput(dir, "stdout", larger); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(larger) {
		t.Fatalf("file = %q, want %q", got, larger)
	}
}

func TestCopyFinalOutput_existingEqual(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")

	data := []byte("exact same")
	if err := os.WriteFile(path, data, 0640); err != nil {
		t.Fatal(err)
	}

	// Copy same-length data — should NOT overwrite (>=).
	if err := CopyFinalOutput(dir, "stdout", data); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("file = %q, want %q", got, data)
	}
}

// --- validateOutputStream ---

func TestValidateOutputStream(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   bool
	}{
		{"", "stdout", false},
		{"stdout", "stdout", false},
		{"stderr", "stderr", false},
		{"stdin", "", true},
		{"STDOUT", "", true},
		{"STDERR", "", true},
		{"random", "", true},
	}

	for _, tt := range tests {
		got, err := validateOutputStream(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("validateOutputStream(%q) = %q, nil; want error", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("validateOutputStream(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("validateOutputStream(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	}
}

// --- ReadTaskOutput missing file ---

func TestReadTaskOutput_missingFile(t *testing.T) {
	dir := t.TempDir()

	chunk, off, done, err := ReadTaskOutput(dir, "stdout", 0)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if chunk != "" {
		t.Fatalf("chunk = %q, want empty", chunk)
	}
	if off != 0 {
		t.Fatalf("offset = %d, want 0", off)
	}
	if done {
		t.Fatal("expected done=false for missing file")
	}
}

// --- ReadTaskOutput offset beyond end ---

func TestReadTaskOutput_offsetBeyondEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")
	if err := os.WriteFile(path, []byte("short"), 0640); err != nil {
		t.Fatal(err)
	}

	// Offset way beyond the file size.
	chunk, off, done, err := ReadTaskOutput(dir, "stdout", 99999)
	if err != nil {
		t.Fatal(err)
	}
	if chunk != "" {
		t.Fatalf("chunk = %q, want empty", chunk)
	}
	if off != 5 {
		t.Fatalf("offset = %d, want 5", off)
	}
	if !done {
		t.Fatal("expected done=true when offset clamped to EOF")
	}
}

// --- ReadTaskOutput large payload ---

func TestReadTaskOutput_largePayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout.txt")

	// Write data larger than taskOutputMaxRead.
	big := strings.Repeat("x", taskOutputMaxRead+100)
	if err := os.WriteFile(path, []byte(big), 0640); err != nil {
		t.Fatal(err)
	}

	// First read should return exactly taskOutputMaxRead bytes.
	chunk, off, done, err := ReadTaskOutput(dir, "stdout", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunk) != taskOutputMaxRead {
		t.Fatalf("first read len = %d, want %d", len(chunk), taskOutputMaxRead)
	}
	if off != int64(taskOutputMaxRead) {
		t.Fatalf("offset = %d, want %d", off, taskOutputMaxRead)
	}
	if done {
		t.Fatal("should not be done after first read")
	}

	// Second read should return the remaining 100 bytes.
	chunk2, off2, done2, err := ReadTaskOutput(dir, "stdout", off)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunk2) != 100 {
		t.Fatalf("second read len = %d, want 100", len(chunk2))
	}
	if off2 != int64(len(big)) {
		t.Fatalf("offset = %d, want %d", off2, len(big))
	}
	if !done2 {
		t.Fatal("expected done after second read")
	}
}
