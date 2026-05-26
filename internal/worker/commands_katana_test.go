package worker

import "testing"

func TestBuildKatanaCommand(t *testing.T) {
	args := BuildKatanaCommand("/tmp/seeds.txt", 2, 10, 15)
	joined := stringsJoin(args)
	for _, want := range []string{"katana", "-list", "/tmp/seeds.txt", "-jsonl", "-jc", "-silent", "-fs", "rdn", "-depth", "2", "-rate-limit", "10", "-timeout", "15"} {
		if !containsArg(args, want) {
			t.Fatalf("missing %q in %s", want, joined)
		}
	}
}

func stringsJoin(args []string) string {
	s := ""
	for _, a := range args {
		s += a + " "
	}
	return s
}

func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
