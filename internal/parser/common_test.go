package parser

import "testing"

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
