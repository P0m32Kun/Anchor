package parser

import (
	"bufio"
	"fmt"
	"io"
)

// maxJSONLineBytes caps a single JSONL line at 16 MiB. Anything longer is a
// runaway tool (or hostile input) and is skipped with a logged error rather
// than being allowed to OOM the server. The previous implementation used
// bufio.Scanner whose default 64 KiB buffer silently truncated finding rows
// that carried large HTTP request/response bodies (e.g. nuclei output for
// templates that capture the full response).
const maxJSONLineBytes = 16 * 1024 * 1024

// ParseError records a single line parse failure.
type ParseError struct {
	Line    int    `json:"line"`
	Raw     string `json:"raw"`
	Message string `json:"message"`
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s (raw: %s)", e.Line, e.Message, e.Raw)
}

// SubfinderResult is the extracted data from a Subfinder JSONL line.
type SubfinderResult struct {
	Host   string `json:"host"`
	Input  string `json:"input"`
	Source string `json:"source"`
}

// HTTPXResult is the extracted data from an httpx JSONL line.
type HTTPXResult struct {
	URL        string `json:"url"`
	Input      string `json:"input"`
	Title      string `json:"title"`
	WebServer  string `json:"webserver"`
	StatusCode int    `json:"status_code"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	Scheme     string `json:"scheme"`
	Path       string `json:"path"`
	Tech       []string
}

// NaabuResult is the extracted data from a Naabu JSONL line.
type NaabuResult struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	IP   string `json:"ip"`
}

// PortInfo represents a single open port from naabu output.
type PortInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// parseJSONLines reads JSONL input line-by-line, skipping empty lines and
// tracking line numbers. The decode callback receives each non-empty line and
// its 1-based line number, returning a parsed value and an optional ParseError.
//
// Implementation note: uses bufio.Reader.ReadBytes('\n') rather than
// bufio.Scanner so that single lines may exceed 64 KiB. A hard ceiling of
// maxJSONLineBytes protects against runaway tools / hostile input. Lines that
// exceed the ceiling are reported as ParseError and skipped, the reader then
// resumes at the next newline.
func parseJSONLines[T any](r io.Reader, decode func(line []byte, lineNo int) (T, ParseError)) ([]T, []ParseError) {
	var results []T
	var errs []ParseError

	br := bufio.NewReaderSize(r, 64*1024)
	lineNo := 0
	for {
		line, err := br.ReadBytes('\n')
		// Handle the line content first, then decide whether to continue
		// based on err. ReadBytes may return non-empty content plus io.EOF
		// for the last unterminated line.
		if len(line) > 0 {
			lineNo++
			// Strip trailing \n (and \r if present).
			trim := line
			if trim[len(trim)-1] == '\n' {
				trim = trim[:len(trim)-1]
			}
			if len(trim) > 0 && trim[len(trim)-1] == '\r' {
				trim = trim[:len(trim)-1]
			}
			if len(trim) > maxJSONLineBytes {
				errs = append(errs, ParseError{
					Line:    lineNo,
					Raw:     "",
					Message: fmt.Sprintf("line exceeds %d byte hard limit (got %d bytes), skipped", maxJSONLineBytes, len(trim)),
				})
			} else if len(trim) > 0 {
				res, perr := decode(trim, lineNo)
				if perr.Message != "" {
					errs = append(errs, perr)
				} else {
					results = append(results, res)
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			errs = append(errs, ParseError{
				Line:    lineNo,
				Raw:     "",
				Message: "reader error: " + err.Error(),
			})
			break
		}
	}

	return results, errs
}
