package citeresolve

import "testing"

// TestIssue456Shapes pins the heading and URL shapes of kit issue #456: a title that opens with a code
// span, and a web URL, which is no file.
func TestIssue456Shapes(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"heading that starts with a code span", "(`types/platform.md` § `module-include.xml` vs `META-INF/module.xml`).",
			[]want{{Resolved, "## `module-include.xml` vs"}}},
		{"bold bullet that starts with a code span", "(`types/platform.md` § module-permissions.xml).",
			[]want{{Resolved, "**`module-permissions.xml` — source-only"}}},
		{"a code span heading still needs its words", "(`types/platform.md` § `module-other.xml`).",
			[]want{{MissingSection, "§module-other.xml"}}},
		{"web URL is no file lookup", "[ev: https://docs.gradle.org/current/userguide/build_environment.html, read 2026-10-24]",
			[]want{{UnverifiableKind, "web page"}}},
	})
}
