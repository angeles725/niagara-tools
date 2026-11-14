package kitresolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOriginUnreachable: a failed fetch is a named WARN and a refusal; --offline-ok accepts the local
// origin/main and the line says "unfetched".
func TestOriginUnreachable(t *testing.T) {
	f := newFixture(t)
	f.clone(f.kitMain)
	git(t, f.kitMain, "remote", "set-url", "origin", filepath.Join(f.base, "missing.git"))
	code, out, errText := f.resolve()
	expect(t, "offline", code, out, errText, 2, "",
		offlineWarn+"kit-resolve: REFUSED offline: rerun with --offline-ok to accept the local origin/main\n")
	code, out, errText = f.resolve("--offline-ok")
	expect(t, "--offline-ok", code, out, errText, 0, f.okLine(f.kitMain, "origin/main unfetched"), offlineWarn)
}

// TestOfflineKitMainBehindIsNeverRefreshed: without a successful fetch a stale kit-main is refused,
// also with --offline-ok and an explicit --refresh.
func TestOfflineKitMainBehindIsNeverRefreshed(t *testing.T) {
	f := newFixture(t)
	f.clone(f.kitMain)
	f.advance()
	git(t, f.kitMain, "fetch", "-q", "origin")
	git(t, f.kitMain, "remote", "set-url", "origin", filepath.Join(f.base, "missing.git"))
	before := f.head(f.kitMain)
	for _, args := range [][]string{{"--offline-ok"}, {"--offline-ok", "--refresh"}} {
		code, out, errText := f.resolve(args...)
		expect(t, "offline, behind", code, out, errText, 2, "", offlineWarn+
			"kit-resolve: REFUSED stale: "+f.kitMain+" is 1 commit(s) behind origin/main — "+remedyKitMain(f.kitMain)+"\n")
	}
	if f.head(f.kitMain) != before {
		t.Error("kit-main was checked out without a fetch")
	}
}

// TestKitOutsideAWorkTreeOrWithoutOriginMain: a kit git cannot answer for is refused.
func TestKitOutsideAWorkTreeOrWithoutOriginMain(t *testing.T) {
	f := newFixture(t)
	plain := filepath.Join(f.base, "plain", kitDirName)
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plain, "METHODOLOGY.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	f.env["BUILD_N4_KIT"] = plain
	t.Setenv("GIT_CEILING_DIRECTORIES", f.base)
	code, out, errText := f.resolve()
	expect(t, "untracked", code, out, errText, 2, "",
		"kit-resolve: REFUSED untracked: "+plain+" is not in a git work tree; its freshness cannot be checked\n")

	git(t, f.base, "init", "-q", "-b", "main", filepath.Dir(plain))
	code, out, errText = f.resolve("--offline-ok")
	expect(t, "no origin/main", code, out, errText, 2, "",
		offlineWarn+"kit-resolve: REFUSED no origin/main ref in "+filepath.Dir(plain)+"\n")
}

// TestRelativeAndSymlinkedKitIsPrintedPhysical: the KIT path is absolute with symlinks resolved.
func TestRelativeAndSymlinkedKitIsPrintedPhysical(t *testing.T) {
	f := newFixture(t)
	f.clone(f.dev)
	link := filepath.Join(f.base, "link")
	if err := os.Symlink(f.dev, link); err != nil {
		t.Fatal(err)
	}
	f.env["BUILD_N4_KIT"] = filepath.Join(link, kitDirName)
	code, out, errText := f.resolve()
	expect(t, "symlink", code, out, errText, 0, f.okLine(f.dev, "origin/main"), "")
	t.Chdir(f.base)
	f.env["BUILD_N4_KIT"] = filepath.Join("dev", kitDirName)
	code, out, errText = f.resolve()
	expect(t, "relative", code, out, errText, 0, f.okLine(f.dev, "origin/main"), "")
}

// TestUnreadableCandidateIsRefusedNotSkipped: the bash tool read "cannot look" as "not there" and went
// on to the next candidate; a winner that cannot be examined is a refusal.
func TestUnreadableCandidateIsRefusedNotSkipped(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads through a mode-000 directory")
	}
	f := newFixture(t)
	f.clone(f.kitMain)
	f.clone(f.defaultTop)
	if err := os.Chmod(f.kitMain, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f.kitMain, 0o755) })
	marker := filepath.Join(f.kitMain, kitDirName, "METHODOLOGY.md")
	code, out, errText := f.resolve()
	expect(t, "unreadable kit-main", code, out, errText, 2, "",
		"kit-resolve: REFUSED unreadable: cannot check "+marker+": permission denied\n")
}

// TestAStatusGitCannotReadIsRefused: the bash tool read a failed `git status` as a clean tree and
// printed the KIT line; a kit whose state cannot be read is a refusal. (A corrupt index also fails the
// fetch, so the line was reached with --offline-ok.)
func TestAStatusGitCannotReadIsRefused(t *testing.T) {
	f := newFixture(t)
	f.clone(f.kitMain)
	if err := os.WriteFile(filepath.Join(f.kitMain, ".git", "index"), []byte("not an index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.resolve("--offline-ok")
	want := "kit-resolve: REFUSED unreadable: git status failed in " + f.kitMain + "; its freshness cannot be checked\n"
	if code != 2 || out != "" || !strings.HasPrefix(errText, offlineWarn) || !strings.HasSuffix(errText, want) || errText == offlineWarn+want {
		t.Errorf("corrupt index: exit %d, stdout %q, stderr %q; want exit 2 and git's message before %q", code, out, errText, want)
	}
}
