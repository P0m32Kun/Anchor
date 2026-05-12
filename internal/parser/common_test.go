package parser

import (
	"strings"
	"testing"
)

func TestParseError_Error(t *testing.T) {
	cases := []struct {
		name string
		err  ParseError
		want string
	}{
		{"full", ParseError{Line: 1, Raw: "test", Message: "bad"}, "line 1: bad (raw: test)"},
		{"no raw", ParseError{Line: 2, Message: "empty"}, "line 2: empty (raw: )"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.err.Error(); got != c.want {
				t.Errorf("Error() = %q, want %q", got, c.want)
			}
		})
	}
}

// decodeLen is a minimal decode function used by parseJSONLines tests: it
// records each line's length so we can assert which lines were processed.
func decodeLen(line []byte, lineNo int) (int, ParseError) {
	return len(line), ParseError{}
}

// TestParseJSONLines_LargeLine verifies that lines larger than bufio.Scanner's
// default 64 KiB buffer are processed without truncation or scanner error. This
// regression-guards the saveNucleiFindings "token too long" bug seen in run
// id-1778560237476298804-140 where a nuclei JSONL line carrying a large HTTP
// response body silently caused all subsequent findings to be dropped.
func TestParseJSONLines_LargeLine(t *testing.T) {
	// 200 KiB single line — well past the 64 KiB scanner default, well under
	// the 16 MiB hard limit.
	large := strings.Repeat("a", 200*1024)
	input := "first\n" + large + "\nthird\n"

	results, errs := parseJSONLines(strings.NewReader(input), decodeLen)

	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d: %+v", len(errs), errs)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 lines, got %d: %+v", len(results), results)
	}
	if results[0] != len("first") {
		t.Errorf("line 1 length: got %d, want %d", results[0], len("first"))
	}
	if results[1] != 200*1024 {
		t.Errorf("line 2 length: got %d, want %d", results[1], 200*1024)
	}
	if results[2] != len("third") {
		t.Errorf("line 3 length: got %d, want %d", results[2], len("third"))
	}
}

// TestParseJSONLines_ExceedsHardLimit verifies that a line larger than the
// 16 MiB hard ceiling is skipped (with a ParseError) and the reader continues
// processing subsequent lines instead of aborting the whole stream.
func TestParseJSONLines_ExceedsHardLimit(t *testing.T) {
	oversized := strings.Repeat("a", maxJSONLineBytes+1024)
	input := "first\n" + oversized + "\nthird\n"

	results, errs := parseJSONLines(strings.NewReader(input), decodeLen)

	if len(errs) != 1 {
		t.Fatalf("expected 1 hard-limit error, got %d: %+v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Message, "hard limit") {
		t.Errorf("error message should mention hard limit, got: %s", errs[0].Message)
	}
	if errs[0].Line != 2 {
		t.Errorf("error should report line 2, got line %d", errs[0].Line)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 successful lines (1 and 3), got %d: %+v", len(results), results)
	}
}

// TestParseJSONLines_EdgeCases covers empty lines, CRLF terminators, and the
// last unterminated line returned with io.EOF.
func TestParseJSONLines_EdgeCases(t *testing.T) {
	// Mix: empty line, CRLF terminated, final line without trailing newline.
	input := "one\n\r\nthree\r\nfinal-no-newline"

	results, errs := parseJSONLines(strings.NewReader(input), decodeLen)

	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d: %+v", len(errs), errs)
	}
	// Expected: "one" (3) + skip empty + "three" with \r stripped (5) + "final-no-newline" (16)
	if len(results) != 3 {
		t.Fatalf("expected 3 results (empty line skipped), got %d: %+v", len(results), results)
	}
	if results[0] != 3 {
		t.Errorf("line 1: got %d, want 3", results[0])
	}
	if results[1] != 5 {
		t.Errorf("line 3 (\"three\" after CRLF strip): got %d, want 5", results[1])
	}
	if results[2] != len("final-no-newline") {
		t.Errorf("final unterminated line: got %d, want %d", results[2], len("final-no-newline"))
	}
}
