package javasrc

import "testing"

// TestUnescape: the quote and backslash escapes a path or ORD literal uses are read; any other is kept.
func TestUnescape(t *testing.T) {
	for in, want := range map[string]string{
		`C:\\Users\\me`:     `C:\Users\me`,
		`say \"hi\" \'x\'`:  `say "hi" 'x'`,
		"a \\` b":           "a ` b",
		`keep \n \t \u0041`: `keep \n \t \u0041`,
		`trailing \`:        `trailing \`,
		`plain`:             `plain`,
	} {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
}
