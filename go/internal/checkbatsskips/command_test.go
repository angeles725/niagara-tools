package checkbatsskips

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const baseAllow = "# comment lines and blank lines are ignored\n\n^tool [a-z]+ not installed$\n"

// fixture writes a TAP and an allowlist under a fresh directory and returns their paths.
func fixture(t *testing.T, tap, allow string) (tapPath, allowPath string) {
	t.Helper()
	dir := t.TempDir()
	tapPath, allowPath = filepath.Join(dir, "run.tap"), filepath.Join(dir, "allow.txt")
	for p, text := range map[string]string{tapPath: tap, allowPath: allow} {
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return tapPath, allowPath
}

func run(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// TestMainRules pins every rule of the check through the command: stdout, stderr and the exit code.
// @TAP@ and @ALLOW@ in the expected text stand for the fixture paths.
func TestMainRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, tap, allow string
		code             int
		out, err         string
	}{
		{"SK1 every skip allowlisted", "1..3\nok 1 a\nok 2 b # skip tool eslint not installed\nnot ok 3 c\n", baseAllow,
			0, "check-bats-skips: 1 skip(s), all allowlisted\n", ""},
		{"no skip at all", "1..1\nok 1 a\n", baseAllow, 0, "check-bats-skips: 0 skip(s), all allowlisted\n", ""},
		{"SK2 unlisted reason names the test", "1..2\nok 1 a # skip tool eslint not installed\nok 2 b-name # skip junit jar missing\n", baseAllow,
			1, "UNLISTED skip: ok 2 b-name — junit jar missing\n", "check-bats-skips: 1 of 2 skip(s) match no line of @ALLOW@\n"},
		{"SK3 a line matches the whole reason", "1..1\nok 1 a # skip tool eslint not installed, and the fixture is gone\n", "tool [a-z]+ not installed\n",
			1, "UNLISTED skip: ok 1 a — tool eslint not installed, and the fixture is gone\n", "check-bats-skips: 1 of 1 skip(s) match no line of @ALLOW@\n"},
		{"SK4 a skip with no reason is unlisted", "1..1\nok 1 a # skip\n", baseAllow,
			1, "UNLISTED skip: ok 1 a — (no reason)\n", "check-bats-skips: 1 of 1 skip(s) match no line of @ALLOW@\n"},
		{"SK4 a line may allow the empty reason", "1..1\nok 1 a # skip\n", "^$\n", 0, "check-bats-skips: 1 skip(s), all allowlisted\n", ""},
		{"SK5 a TAP with no plan", "", baseAllow, 3, "", "check-bats-skips: no 1..N plan in @TAP@ (bats did not run)\n"},
		{"SK5 a plan must be a whole line", "x 1..3\nok 1 a\n", baseAllow, 3, "", "check-bats-skips: no 1..N plan in @TAP@ (bats did not run)\n"},
		{"SK6 a bad ERE is an env error", "1..1\nok 1 a # skip tool x not installed\n", "^tool (x not installed$\n",
			3, "", "check-bats-skips: allowlist line is not a valid ERE: ^tool (x not installed$\n"},
		{"SK8 the last valid marker splits", "1..1\nok 1 a # skip junit jar missing # skipped twice\n", baseAllow,
			1, "UNLISTED skip: ok 1 a — junit jar missing # skipped twice\n", "check-bats-skips: 1 of 1 skip(s) match no line of @ALLOW@\n"},
		{"SK8 a name holding a marker keeps it", "1..1\nok 1 a # skip b # skip tool x not installed\n", baseAllow,
			0, "check-bats-skips: 1 skip(s), all allowlisted\n", ""},
		{"SK9 each line is its own pattern", "1..3\nok 1 a # skip alpha one\nok 2 b # skip beta two\nok 3 c # skip one beta\n", "^alpha one$\n^beta two$\n",
			1, "UNLISTED skip: ok 3 c — one beta\n", "check-bats-skips: 1 of 3 skip(s) match no line of @ALLOW@\n"},
		{"SK10 a last line with no newline counts", "1..1\nok 1 a # skip junit jar missing\n", "^junit jar missing$",
			0, "check-bats-skips: 1 skip(s), all allowlisted\n", ""},
		{"an allowlist of comments allows nothing", "1..1\nok 1 a # skip tool x not installed\n", "# nothing\n\n",
			1, "UNLISTED skip: ok 1 a — tool x not installed\n", "check-bats-skips: 1 of 1 skip(s) match no line of @ALLOW@\n"},
		{"a failed test is never a skip", "1..1\nnot ok 1 a # skip junit jar missing\n", baseAllow,
			0, "check-bats-skips: 0 skip(s), all allowlisted\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tap, allow := fixture(t, c.tap, c.allow)
			code, out, errText := run(tap, allow)
			sub := strings.NewReplacer("@TAP@", tap, "@ALLOW@", allow)
			if code != c.code || out != sub.Replace(c.out) || errText != sub.Replace(c.err) {
				t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit %d\nstdout %q\nstderr %q", code, out, errText, c.code, sub.Replace(c.out), sub.Replace(c.err))
			}
		})
	}
}

// TestMainUnreadable: usage, a missing TAP and a missing allowlist are exit 3, never a pass (SK5).
func TestMainUnreadable(t *testing.T) {
	t.Parallel()
	tap, allow := fixture(t, "1..0\n", baseAllow)
	missing := filepath.Join(t.TempDir(), "missing")
	cases := []struct {
		name string
		args []string
		err  string
	}{
		{"one argument", []string{tap}, "check-bats-skips: usage: check-bats-skips.sh <bats.tap> <allowlist>\n"},
		{"three arguments", []string{tap, allow, allow}, "check-bats-skips: usage: check-bats-skips.sh <bats.tap> <allowlist>\n"},
		{"missing TAP", []string{missing, allow}, "check-bats-skips: cannot read " + missing + "\n"},
		{"missing allowlist", []string{tap, missing}, "check-bats-skips: cannot read " + missing + "\n"},
		{"a directory is not a TAP", []string{filepath.Dir(tap), allow}, "check-bats-skips: cannot read " + filepath.Dir(tap) + "\n"},
	}
	for _, c := range cases {
		if code, out, errText := run(c.args...); code != ExitEnv || out != "" || errText != c.err {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 and stderr %q", c.name, code, out, errText, c.err)
		}
	}
}

// TestCommittedAllowlist pins the dialect: every line of tests/bats-skip-allowlist.txt is an ERE this
// port compiles, and it accepts the skips the CI bats job is known to take (bats SK7).
func TestCommittedAllowlist(t *testing.T) {
	t.Parallel()
	committed, err := filepath.Abs("../../../tests/bats-skip-allowlist.txt")
	if err != nil {
		t.Fatal(err)
	}
	tap, _ := fixture(t, strings.Join([]string{"1..9",
		"ok 1 a # skip campaign-9 close gate — run with C9_CLOSE=1 at close",
		"ok 2 b # skip C11 close gate — set C11_CLOSE=1 to run",
		"ok 3 c # skip client read tree not on this machine (set C9_CLIENT_ROOT)",
		"ok 4 d # skip no niagara_home/JDK 8 for the real round-trip (local-only bless evidence, never a fake PASS)",
		"ok 5 e # skip no Chrome (set KIT_CHROME or PUPPETEER_EXECUTABLE_PATH)",
		"ok 6 f # skip SiteA bog not on this machine (set C8_SITEA_BOG)",
		"ok 7 g # skip the pinned eslint is installed under toolbelt/eslint; the PATH fallback is not reached",
		"ok 8 h # skip launcher not installed on this machine",
		"ok 9 i # skip ModA-ux rc not on this machine (local-only real smoke)", ""}, "\n"), "")
	if code, out, errText := run(tap, committed); code != ExitOK || out != "check-bats-skips: 9 skip(s), all allowlisted\n" || errText != "" {
		t.Errorf("committed allowlist: exit %d stdout %q stderr %q", code, out, errText)
	}
	tap, _ = fixture(t, "1..1\nok 1 a # skip acorn not installed\n", "")
	if code, _, _ := run(tap, committed); code != ExitUnlisted {
		t.Errorf("a lost tool install is exit %d, want %d", code, ExitUnlisted)
	}
}
