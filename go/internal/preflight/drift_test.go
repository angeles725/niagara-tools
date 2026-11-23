package preflight

import (
	"os"
	"strings"
	"testing"
)

func TestVersionLess(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.1", true}, {"2.0.0", "2.0.10", true}, {"2.0.10", "2.0.9", false}, {"9.0.0", "10.0.0", true},
		{"1.0.01", "1.0.1", true}, {"1.0.1", "1.0.01", false}, {"1.0.0", "1.0.0", false}, {"3.0.0", "1.9.9", false},
	} {
		if got := versionLess(c.a, c.b); got != c.want {
			t.Errorf("versionLess(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestVersionDrift(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode 000 does not block reads")
	}
	d := t.TempDir()
	mk(t, d, "nh/", "w/p/me/build.gradle.kts=defaultModuleVersion(\"1.0.0\")",
		"w/p/a/x.gradle.kts=version = \"9.9.9\"\n// defaultModuleVersion(v)\ndefaultModuleVersion(\"1.5.0\") // \"8.8.8\"\n",
		"w/p/b/d1/d2/d3/d4.gradle.kts=defaultModuleVersion(\"6.0.0\")", "w/p/c/c.gradle.kts=defaultModuleVersion(\"0.9.0\")", "w/p/c/locked/")
	if err := os.Chmod(d+"/w/p/c/locked", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d+"/w/p/c/locked", 0o755) })
	_, out, errOut := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/w/p/me")
	want := "WARN  version-drift  " + d + "/w/p/me version 1.0.0 < sibling " + d + "/w/p/a version 1.5.0\n"
	if !strings.Contains(out, want) || strings.Contains(out, "6.0.0") || strings.Contains(out, "0.9.0") {
		t.Errorf("rows\n%s\nwant %q and no row for b (depth 4) or c (lower)", out, want)
	}
	if errOut != "" { // retro build-kit-bench-session Δ16: a directory it cannot read is skipped without a line
		t.Errorf("stderr %q, want none for the unreadable directory", errOut)
	}
}

// Retro build-kit-bench-session Δ16: the walk stays inside the tree of the gradle root. A level whose parent is the
// temporary directory (or the filesystem root, or the home) is not walked: its entries are other people's trees,
// and walking them took 28 s on a gradle root two levels below a temporary directory.
func TestVersionDriftDoesNotWalkASharedDirectory(t *testing.T) {
	d := t.TempDir()
	t.Setenv("TMPDIR", d)
	mk(t, d, "nh/", "x/out/me/build.gradle.kts=defaultModuleVersion(\"1.0.0\")",
		"x/other/build.gradle.kts=defaultModuleVersion(\"1.2.0\")", "stranger/build.gradle.kts=defaultModuleVersion(\"9.9.9\")")
	_, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/x/out/me")
	if want := "sibling " + d + "/x/other version 1.2.0"; !strings.Contains(out, want) || strings.Contains(out, "9.9.9") {
		t.Errorf("rows\n%s\nwant the sibling inside the tree (1.2.0) and no row for a tree beside it in the temporary directory", out)
	}
}

// Retro kit-followups-rare-cases Δ3: a `<repo>-worktrees/` checkout of the same repository and the repository root
// that holds the gradle root are not sibling modules, so a newer version there is no drift.
func TestVersionDriftSkipsWorktreesAndTheRepoRoot(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "w/repo/me/build.gradle.kts=defaultModuleVersion(\"1.0.0\")",
		"w/repo/other/build.gradle.kts=defaultModuleVersion(\"1.1.0\")",
		"w/repo/settings.gradle.kts=defaultModuleVersion(\"7.0.0\")",
		"w/repo-worktrees/feat/me/build.gradle.kts=defaultModuleVersion(\"2.0.0\")")
	_, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/w/repo/me")
	if want := "sibling " + d + "/w/repo/other version 1.1.0"; !strings.Contains(out, want) ||
		strings.Contains(out, "2.0.0") || strings.Contains(out, "7.0.0") {
		t.Errorf("rows\n%s\nwant the real sibling (1.1.0) and no row for the worktrees checkout or the repo root", out)
	}
}
