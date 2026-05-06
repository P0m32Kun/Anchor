package custom

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestValidateGitURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		ok   bool
	}{
		{"https-ok", "https://github.com/example/x.git", true},
		{"http-rejected", "http://github.com/example/x.git", false},
		{"git-protocol-rejected", "git://github.com/example/x.git", false},
		{"ssh-rejected", "git@github.com:example/x.git", false},
		{"file-rejected", "file:///etc/passwd", false},
		{"empty-rejected", "", false},
		{"uppercase-https-ok", "HTTPS://github.com/example/x.git", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGitURL(tc.url)
			if tc.ok && err != nil {
				t.Errorf("want ok, got err: %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("want err, got nil")
			}
		})
	}
}

func TestExecCloner_RejectsNonHTTPSWithoutShellingOut(t *testing.T) {
	c := ExecCloner{Bin: "/nonexistent/git-binary"}
	err := c.Clone(context.Background(), "git@github.com:foo/bar.git", "", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("error should mention https requirement; got %v", err)
	}
}

func TestCapWriter_TruncatesAtCap(t *testing.T) {
	var buf bytes.Buffer
	w := &capWriter{w: &buf, cap: 4}

	n, err := w.Write([]byte("abcdefgh"))
	if err != nil {
		t.Fatalf("write1: %v", err)
	}
	if n != 8 {
		t.Errorf("write1 n: want 8, got %d", n)
	}
	if buf.String() != "abcd" {
		t.Errorf("buf after write1: %q", buf.String())
	}

	n, err = w.Write([]byte("xyz"))
	if err != nil {
		t.Fatalf("write2: %v", err)
	}
	if n != 3 {
		t.Errorf("write2 n: want 3 (consumed), got %d", n)
	}
	if buf.String() != "abcd" {
		t.Errorf("buf after write2 should be unchanged: %q", buf.String())
	}
}
