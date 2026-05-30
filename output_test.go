package main

import "testing"

func TestStripANSI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"\033[1;38;5;2mweb\033[0m| listening on :3000", "web| listening on :3000"},
		{"\033[0;31merror\033[0m", "error"},
		{"a\033[2Kb", "ab"},
	}

	for _, c := range cases {
		if got := string(stripANSI([]byte(c.in))); got != c.want {
			t.Errorf("stripANSI(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
