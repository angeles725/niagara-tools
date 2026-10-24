package kitticket

import (
	"os"
	"slices"
	"testing"
)

// TestSlugMultiLine pins the slug of titles a golden case cannot spell (its args are one per line); each want
// is what the bash pipeline printed for that title under LC_ALL=C.
func TestSlugMultiLine(t *testing.T) {
	for title, want := range map[string]string{
		"ab\ncdefg":  "ab\ncdefg",
		"Fix it\n":   "fix-it",
		"\n\nabc":    "\n\nabc-ticket",
		"abcdef\n\n": "abcdef",
		"-x-\n-y-":   "x\ny-ticket",
		"ABC\tDEF":   "abc-def",
	} {
		if got := slug(title); got != want {
			t.Errorf("slug(%q) = %q, want %q", title, got, want)
		}
	}
}

// TestSplitGlob pins the unquoted-$REPO_FLAG expansion beyond the goldens: hidden entries need a dot pattern,
// [!x] negates, a malformed pattern stays literal.
func TestSplitGlob(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, f := range []string{"ga", "gb", ".gc", "hx"} {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for in, want := range map[string][]string{
		"--repo g*":      {"--repo", "ga", "gb"},
		"--repo .g*":     {"--repo", ".gc"},
		"--repo g[!a]":   {"--repo", "gb"},
		"--repo [":       {"--repo", "["},
		"--repo \ta\nb ": {"--repo", "a", "b"},
	} {
		if got := splitGlob(in); !slices.Equal(got, want) {
			t.Errorf("splitGlob(%q) = %q, want %q", in, got, want)
		}
	}
}
