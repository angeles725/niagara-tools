package checkclientsource

import (
	"path/filepath"
	"testing"
)

// TestCheckoutRows pins one row per rule of the work-tree checks (bats CS2-CS9): the exact text and
// the exit code. prepare changes the checkout before the run.
func TestCheckoutRows(t *testing.T) {
	const absent = "0123456789abcdef0123456789abcdef01234567"
	cases := []struct {
		name     string
		prepare  func(f *fixture)
		declared func(f *fixture) string
		code     int
		out      func(f *fixture) string
	}{
		{"CS2 detached HEAD", func(f *fixture) { gitOut(f.t, f.top, "checkout", "-q", "--detach", f.first) },
			func(f *fixture) string { return "r@main@" + f.second }, ExitFail,
			func(f *fixture) string {
				return "FAIL  source-of-truth  detached HEAD at " + f.first + " in " + f.top + " — check out the source-of-truth branch before building\n" +
					"FAIL  source-of-truth  HEAD " + f.first[:12] + " is behind or diverged from the declared commit " + f.second + " — stale checkout\n"
			}},
		{"CS3 at the declared commit", nil, func(f *fixture) string { return "r@main@" + f.second }, ExitOK,
			func(f *fixture) string {
				return "PASS  source-of-truth  " + f.top + " at the declared commit " + f.second + " on main\n"
			}},
		{"CS4 behind the declared commit", func(f *fixture) { gitOut(f.t, f.top, "reset", "-q", "--hard", f.first) },
			func(f *fixture) string { return "r@main@" + f.second }, ExitFail,
			func(f *fixture) string {
				return "FAIL  source-of-truth  HEAD " + f.first[:12] + " is behind or diverged from the declared commit " + f.second + " — stale checkout\n"
			}},
		{"CS5 declared commit not in the checkout", nil, func(f *fixture) string { return "r@main@" + absent }, ExitFail,
			func(f *fixture) string {
				return "FAIL  source-of-truth  declared commit " + absent + " is not in this checkout — a stale clone (fetch it) or the wrong tree\n"
			}},
		{"CS6 another branch", func(f *fixture) { gitOut(f.t, f.top, "checkout", "-q", "-b", "feature") },
			func(f *fixture) string { return "r@main@" + f.second }, ExitFail,
			func(f *fixture) string {
				return "FAIL  source-of-truth  checked-out branch feature is not the declared source-of-truth branch main\n" +
					"PASS  source-of-truth  " + f.top + " at the declared commit " + f.second + " on feature\n"
			}},
		{"CS7 another checkout", nil, func(f *fixture) string { return "Cliente/other-tree@main@" + f.second }, ExitFail,
			func(f *fixture) string {
				return "FAIL  source-of-truth  work tree " + f.top + " is not the declared source of truth Cliente/other-tree — build from the declared checkout\n" +
					"PASS  source-of-truth  " + f.top + " at the declared commit " + f.second + " on main\n"
			}},
		{"CS8 ahead of the declared commit", nil, func(f *fixture) string { return "r@main@" + f.first }, ExitOK,
			func(f *fixture) string {
				return "PASS  source-of-truth  " + f.top + " is ahead of the declared commit " + f.first + " (new local work) — update source_of_truth at close\n"
			}},
		{"CS9 unknown", nil, func(f *fixture) string { return "unknown" }, ExitOK,
			func(f *fixture) string {
				return "WARN  source-of-truth  source_of_truth is unknown — honest, but this checkout (" + f.top + ") is unchecked\n"
			}},
		{"an empty declared value is undeclared", nil, func(f *fixture) string { return "" }, ExitOK,
			func(f *fixture) string {
				return "WARN  source-of-truth  source_of_truth undeclared for this module — record <path>@<branch>@<commit> in its BUILD-STATE.src.md envelope\n"
			}},
		{"commit unknown checks path and branch", nil, func(f *fixture) string { return "r@main@unknown" }, ExitOK,
			func(f *fixture) string {
				return "PASS  source-of-truth  " + f.top + " on main matches the declared path/branch (commit unknown)\n"
			}},
		{"branch unknown skips the branch", func(f *fixture) { gitOut(f.t, f.top, "checkout", "-q", "-b", "feature") },
			func(f *fixture) string { return "r@unknown@" + f.second }, ExitOK,
			func(f *fixture) string {
				return "PASS  source-of-truth  " + f.top + " at the declared commit " + f.second + " on feature\n"
			}},
		{"a failed path leaves no PASS when the commit is unknown", nil, func(f *fixture) string { return "elsewhere@main@unknown" }, ExitFail,
			func(f *fixture) string {
				return "FAIL  source-of-truth  work tree " + f.top + " is not the declared source of truth elsewhere — build from the declared checkout\n"
			}},
		{"the declared path may be the absolute top level", nil, func(f *fixture) string { return f.top + "/@main@unknown" }, ExitOK,
			func(f *fixture) string {
				return "PASS  source-of-truth  " + f.top + " on main matches the declared path/branch (commit unknown)\n"
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			if c.prepare != nil {
				c.prepare(f)
			}
			expect(t, c.name, []string{"--declared", c.declared(f), f.root}, c.code, c.out(f), "")
		})
	}
}

// TestOutsideAWorkTree: CS1. A module root no repository holds is a WARN and exit 0.
func TestOutsideAWorkTree(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "plain", "Mod")
	write(t, filepath.Join(plain, "a.txt"), "a\n")
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(filepath.Dir(plain)))
	expect(t, "CS1", []string{"--declared", "unknown", plain}, ExitOK,
		"WARN  source-of-truth  "+plain+" is not inside a git work tree — it cannot answer which commit is deployed; put the module in a repository\n", "")
}

// TestBehindUpstream: CS12. HEAD behind its upstream tracking ref as of the last fetch is a WARN naming
// the count; nothing is fetched.
func TestBehindUpstream(t *testing.T) {
	f := newFixture(t)
	down := filepath.Join(f.base, "down")
	gitOut(t, f.base, "clone", "-q", f.top, down)
	write(t, filepath.Join(f.root, "c.txt"), "c\n")
	f.commit(f.top, "three")
	write(t, filepath.Join(f.root, "d.txt"), "d\n")
	f.commit(f.top, "four")
	gitOut(t, down, "fetch", "-q")
	expect(t, "CS12", []string{"--declared", "down@main@" + f.second, filepath.Join(down, "Group", "Mod")}, ExitOK,
		"PASS  source-of-truth  "+down+" at the declared commit "+f.second+" on main\n"+
			"WARN  source-of-truth  main is behind its upstream tracking ref by 2 commit(s) as of the last fetch — pull before building\n", "")
}
