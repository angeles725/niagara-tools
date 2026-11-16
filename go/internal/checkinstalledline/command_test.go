package checkinstalledline

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// gitEnv keeps every test repository off the host's git configuration and identity.
func gitEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": os.DevNull, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.invalid",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.invalid",
	} {
		t.Setenv(k, v)
	}
}

func git(t *testing.T, repo string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// versions writes each GROUP build.gradle.kts with its version; "" removes the group.
func versions(t *testing.T, repo string, byGroup map[string]string) {
	t.Helper()
	for group, v := range byGroup {
		dir := filepath.Join(repo, group)
		if v == "" {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
			continue
		}
		text := "niagaraModule {\n  // defaultModuleVersion sets the \"vendorVersion\" attribute on all modules\n  defaultModuleVersion(\"" + v + "\")\n}\n"
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// installed is a repository whose tag `installed` holds Dashboard 2.10.0, Compresores 2.8.0 and Paccadia
// 2.4.2 (the incident fixture), checked out on a new branch `fix`.
func installed(t *testing.T) string {
	t.Helper()
	gitEnv(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	versions(t, repo, map[string]string{"Dashboard": "2.10.0", "Compresores": "2.8.0", "Paccadia": "2.4.2"})
	commit(t, repo, "installed")
	git(t, repo, "tag", "installed")
	git(t, repo, "checkout", "-q", "-b", "fix")
	return repo
}

func commit(t *testing.T, repo, message string) {
	t.Helper()
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", message)
}

func run(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// TestMainRows pins each row and exit code of the guard on the incident fixture: rows in path order.
func TestMainRows(t *testing.T) {
	cases := []struct {
		name   string
		branch map[string]string
		code   int
		out    string
	}{
		{"ILG-downgrade", map[string]string{"Dashboard": "2.9.2", "Paccadia": "2.4.4"}, ExitRefused,
			"ok Compresores/build.gradle.kts 2.8.0 -> 2.8.0\nDOWNGRADE Dashboard/build.gradle.kts 2.10.0 -> 2.9.2\nok Paccadia/build.gradle.kts 2.4.2 -> 2.4.4\n"},
		{"ILG-equal-higher", map[string]string{"Dashboard": "2.10.1", "Paccadia": "2.4.10"}, ExitOK,
			"ok Compresores/build.gradle.kts 2.8.0 -> 2.8.0\nok Dashboard/build.gradle.kts 2.10.0 -> 2.10.1\nok Paccadia/build.gradle.kts 2.4.2 -> 2.4.10\n"},
		{"ILG-missing", map[string]string{"Compresores": ""}, ExitRefused,
			"MISSING Compresores/build.gradle.kts 2.8.0\nok Dashboard/build.gradle.kts 2.10.0 -> 2.10.0\nok Paccadia/build.gradle.kts 2.4.2 -> 2.4.2\n"},
		{"ILG-incomparable", map[string]string{"Dashboard": "2.10.0-rc"}, ExitRefused,
			"ok Compresores/build.gradle.kts 2.8.0 -> 2.8.0\nINCOMPARABLE Dashboard/build.gradle.kts 2.10.0 -> 2.10.0-rc\nok Paccadia/build.gradle.kts 2.4.2 -> 2.4.2\n"},
		{"unchanged branch", nil, ExitOK,
			"ok Compresores/build.gradle.kts 2.8.0 -> 2.8.0\nok Dashboard/build.gradle.kts 2.10.0 -> 2.10.0\nok Paccadia/build.gradle.kts 2.4.2 -> 2.4.2\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := installed(t)
			versions(t, repo, c.branch)
			commit(t, repo, c.name)
			if code, out, errText := run(repo, "fix", "installed"); code != c.code || out != c.out || errText != "" {
				t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit %d\nstdout %q", code, out, errText, c.code, c.out)
			}
		})
	}
}

// TestMainUsage: a wrong argument count, an unknown ref and an installed ref with nothing to guard are
// exit 2 (nothing to guard is never a pass). The usage line is stderr; the other two are stdout.
func TestMainUsage(t *testing.T) {
	repo := installed(t)
	git(t, repo, "checkout", "-q", "--orphan", "empty")
	git(t, repo, "rm", "-q", "-r", "-f", ".")
	commit(t, repo, "empty")
	const usage = "usage: check-installed-line.sh <repo> <branch> <installed-ref>\n"
	cases := []struct {
		name     string
		args     []string
		out, err string
	}{
		{"two arguments", []string{repo, "fix"}, "", usage},
		{"four arguments", []string{repo, "fix", "installed", "x"}, "", usage},
		{"unknown branch", []string{repo, "no-such-branch", "installed"}, "check-installed-line: unknown ref no-such-branch\n", ""},
		{"unknown installed ref", []string{repo, "fix", "no-such-ref"}, "check-installed-line: unknown ref no-such-ref\n", ""},
		{"ILG-no-groups", []string{repo, "fix", "empty"}, "check-installed-line: no defaultModuleVersion in any build.gradle.kts at empty\n", ""},
	}
	for _, c := range cases {
		if code, out, errText := run(c.args...); code != ExitUsage || out != c.out || errText != c.err {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 2 stdout %q stderr %q", c.name, code, out, errText, c.out, c.err)
		}
	}
}

// TestMainSkipsFilesWithNoCall: a build.gradle.kts that only mentions defaultModuleVersion( in a comment
// at the installed ref is not a group; it is neither a row nor counted.
func TestMainSkipsFilesWithNoCall(t *testing.T) {
	gitEnv(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	versions(t, repo, map[string]string{"Dashboard": "1.3"})
	if err := os.WriteFile(filepath.Join(repo, "build.gradle.kts"), []byte("// defaultModuleVersion(\"9.9\") is set per group\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit(t, repo, "installed")
	if code, out, errText := run(repo, "main", "main"); code != ExitOK || out != "ok Dashboard/build.gradle.kts 1.3 -> 1.3\n" || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q", code, out, errText)
	}
}

// TestMainComparesAGroupUnderAPathGitQuotes (ILG-quoted-path): git prints a path holding a non-ASCII byte in quotes unless
// it is asked for NUL-separated names; such a group is compared like any other, so its downgrade refuses
// the branch (retro go-check-scripts-ports Δ1). The row names the path as it is on disk.
func TestMainComparesAGroupUnderAPathGitQuotes(t *testing.T) {
	gitEnv(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	versions(t, repo, map[string]string{"Cámaras": "5.1", "Dashboard": "1.3"})
	commit(t, repo, "installed")
	git(t, repo, "tag", "installed")
	versions(t, repo, map[string]string{"Cámaras": "5.0"})
	commit(t, repo, "downgrade")
	want := "DOWNGRADE Cámaras/build.gradle.kts 5.1 -> 5.0\nok Dashboard/build.gradle.kts 1.3 -> 1.3\n"
	if code, out, errText := run(repo, "main", "installed"); code != ExitRefused || out != want || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q, want exit %d stdout %q", code, out, errText, ExitRefused, want)
	}
}

// TestMainReadsARefPathAWorkingTreeEntryShadows (ILG-ambiguous): with a working-tree entry literally named
// `<ref>:<group>/build.gradle.kts`, git refuses `show <ref>:<path>` as "both revision and filename" unless
// the path list is closed with `--`; the group then read as "no version at the installed ref" and was
// skipped, so its downgrade passed. The entry is untracked and never part of the comparison.
func TestMainReadsARefPathAWorkingTreeEntryShadows(t *testing.T) {
	gitEnv(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	versions(t, repo, map[string]string{"GroupA": "3.2.0", "GroupB": "1.0.0"})
	commit(t, repo, "installed")
	git(t, repo, "tag", "installed")
	versions(t, repo, map[string]string{"GroupA": "3.1.0"})
	commit(t, repo, "downgrade")
	versions(t, repo, map[string]string{"installed:GroupA": "9.9.9", "main:GroupB": "0.0.1"}) // shadows, not committed
	want := "DOWNGRADE GroupA/build.gradle.kts 3.2.0 -> 3.1.0\nok GroupB/build.gradle.kts 1.0.0 -> 1.0.0\n"
	if code, out, errText := run(repo, "main", "installed"); code != ExitRefused || out != want || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q, want exit %d stdout %q", code, out, errText, ExitRefused, want)
	}
}

// TestMainPrintsAControlCharacterPathOnOneLine: a group path holding a newline is still one row; the path
// is printed as git quotes one (double quotes, C escapes), so a reader that splits on lines sees every group.
func TestMainPrintsAControlCharacterPathOnOneLine(t *testing.T) {
	gitEnv(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	versions(t, repo, map[string]string{"Gro\nup": "2.0.0", "Tab\tbed": "1.1.0", `Quo"te\d`: "1.0.0", "Plain": "4.0.0"})
	commit(t, repo, "installed")
	git(t, repo, "tag", "installed")
	versions(t, repo, map[string]string{"Gro\nup": "1.9.0", "Tab\tbed": ""})
	commit(t, repo, "downgrade")
	want := `DOWNGRADE "Gro\nup/build.gradle.kts" 2.0.0 -> 1.9.0` + "\n" +
		"ok Plain/build.gradle.kts 4.0.0 -> 4.0.0\n" +
		`ok "Quo\"te\\d/build.gradle.kts" 1.0.0 -> 1.0.0` + "\n" +
		`MISSING "Tab\tbed/build.gradle.kts" 1.1.0` + "\n"
	if code, out, errText := run(repo, "main", "installed"); code != ExitRefused || out != want || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q, want exit %d stdout %q", code, out, errText, ExitRefused, want)
	}
}

// TestMainQuotesEachByteThatBreaksARow: each byte that makes a path unprintable as it is quotes the row on
// its own: DEL, a backslash, a double quote, and a control byte with no letter escape (three octal digits).
func TestMainQuotesEachByteThatBreaksARow(t *testing.T) {
	gitEnv(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	versions(t, repo, map[string]string{"De\x7fl": "1.0.0", `Back\slash`: "1.0.0", `Quo"te`: "1.0.0", "Ct\x01rl": "1.0.0"})
	commit(t, repo, "installed")
	want := `ok "Back\\slash/build.gradle.kts" 1.0.0 -> 1.0.0` + "\n" +
		`ok "Ct\001rl/build.gradle.kts" 1.0.0 -> 1.0.0` + "\n" +
		`ok "De\177l/build.gradle.kts" 1.0.0 -> 1.0.0` + "\n" +
		`ok "Quo\"te/build.gradle.kts" 1.0.0 -> 1.0.0` + "\n"
	if code, out, errText := run(repo, "main", "main"); code != ExitOK || out != want || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q, want exit %d stdout %q", code, out, errText, ExitOK, want)
	}
}
