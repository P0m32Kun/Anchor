package worker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const taskOutputMaxRead = 256 * 1024 // per tail request

// taskOutputWriter mirrors process output to a workdir file (live tail) and a size-limited buffer (final artifact).
type taskOutputWriter struct {
	file *os.File
	buf  *limitedBuffer
}

func newTaskOutputWriter(workdir, baseName string) (*taskOutputWriter, error) {
	if err := os.MkdirAll(workdir, 0750); err != nil {
		return nil, err
	}
	path := filepath.Join(workdir, baseName+".txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, err
	}
	return &taskOutputWriter{file: f, buf: &limitedBuffer{}}, nil
}

func (w *taskOutputWriter) Write(p []byte) (int, error) {
	if w.file != nil {
		_, _ = w.file.Write(p)
	}
	return w.buf.Write(p)
}

func (w *taskOutputWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *taskOutputWriter) Bytes() []byte        { return w.buf.Bytes() }
func (w *taskOutputWriter) Len() int             { return w.buf.Len() }
func (w *taskOutputWriter) Truncated() bool      { return w.buf.truncated }
func (w *taskOutputWriter) FilePath(workdir, base string) string {
	return filepath.Join(workdir, base+".txt")
}

// ReadTaskOutput returns up to taskOutputMaxRead bytes from a task output file starting at offset.
func ReadTaskOutput(workdir, stream string, offset int64) (content string, nextOffset int64, complete bool, err error) {
	base := "stdout"
	if stream == "stderr" {
		base = "stderr"
	}
	path := filepath.Join(workdir, base+".txt")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", offset, false, nil
		}
		return "", offset, false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", offset, false, err
	}
	if offset > info.Size() {
		offset = info.Size()
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return "", offset, false, err
	}
	buf := make([]byte, taskOutputMaxRead)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", offset, false, err
	}
	nextOffset = offset + int64(n)
	return string(buf[:n]), nextOffset, nextOffset >= info.Size(), nil
}

// TaskOutputPaths returns stdout/stderr paths under workdir.
func TaskOutputPaths(workdir string) (stdout, stderr string) {
	return filepath.Join(workdir, "stdout.txt"), filepath.Join(workdir, "stderr.txt")
}

// CopyFinalOutput writes buffer bytes to the tail file if the file is missing or smaller (artifact consistency).
func CopyFinalOutput(workdir, base string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	path := filepath.Join(workdir, base+".txt")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(existing) >= len(data) {
		return nil
	}
	return os.WriteFile(path, data, 0640)
}

func validateOutputStream(stream string) (string, error) {
	switch stream {
	case "", "stdout":
		return "stdout", nil
	case "stderr":
		return "stderr", nil
	default:
		return "", fmt.Errorf("invalid stream %q", stream)
	}
}
