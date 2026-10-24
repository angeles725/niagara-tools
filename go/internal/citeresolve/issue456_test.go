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

// TestIssue456Roots pins the root shapes of kit issue #456: a directory of a module copy, a `~/` path,
// and a javap jar in the Gradle cache.
func TestIssue456Roots(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"directory in a module copy", "[ev: code gamma-ux imageOverrides/]", []want{{Resolved, "directory"}}},
		{"directory missing from the module copies", "[ev: code gamma-ux nopeOverrides/]", []want{{MissingFile, "nopeOverrides/ in the copies of gamma-ux"}}},
		{"home file with a section", "[ev: ~/notes/home-plan.md §3 Provenance markers]", []want{{Resolved, "## 3. Provenance markers"}}},
		{"home directory", "[ev: ~/notes/ model: frontmatter]", []want{{Resolved, "directory"}}},
		{"home file missing", "[ev: ~/notes/nope.md]", []want{{MissingFile, "nope.md does not exist"}}},
		{"a kit path is never home-expanded", "[ev: kit ~/notes/home-plan.md]", []want{{MissingFile, "~/notes/home-plan.md"}}},
		{"javap jar in the Gradle jars cache", "javap of fixture-plugins-1.0.jar from `~/.gradle/caches`",
			[]want{{Resolved, "jars-9/0a1b2c/fixture-plugins-1.0.jar"}}},
		{"javap jar in the Gradle modules cache", "javap of `fixture-plugin-2.0.jar`", []want{{Resolved, "files-2.1/com.example"}}},
		{"javap jar in no known root", "javap of `nowhere-9.jar`", []want{{MissingFile, "organized/ or the Gradle cache"}}},
	})
}
