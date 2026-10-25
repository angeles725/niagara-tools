package batserrexit

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// run lints one t.bats holding src and returns the exit code and stdout, with the file path as "@F@".
func run(t *testing.T, src string) (int, string) {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "t.bats")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	if e.Len() != 0 {
		t.Errorf("stderr %q", e.String())
	}
	return code, strings.ReplaceAll(o.String(), f, "@F@")
}

func and(line int) string {
	return "FAIL  " + CheckAndList + "  @F@:" + strconv.Itoa(line) + "  " + msgAndList + "\n"
}

func neg(line int) string {
	return "FAIL  " + CheckNegation + "  @F@:" + strconv.Itoa(line) + "  " + msgNegation + "\n"
}

// TestCases: each case is one @test body after `@test "x" {` (line 1) and the rows it gives, line numbered from 2.
func TestCases(t *testing.T) {
	for _, c := range []struct {
		name, body, want string
	}{
		{"and-list mid-test", "  [ \"$status\" -eq 1 ] && [[ \"$output\" == *x* ]]\n  true\n", and(2)},
		{"three assertions and grep", "  [ -f a ] && grep -q x a && cmp a b\n  true\n", and(2)},
		{"assert helpers and a negated element", "  assert_success && ! grep -q x f\n  true\n", and(2)},
		{"bare negation mid-test", "  ! grep -q x f\n  true\n", neg(2)},
		{"negated pipeline mid-test", "  ! printf x | grep -q y\n  true\n", neg(2)},
		{"last line and-list is seen", "  true\n  [ a = a ] && [ b = b ]\n", ""},
		{"last line negation is seen", "  true\n  ! false\n", ""},
		{"or with an assertion last", "  [ a = a ] || [ b = b ]\n  [ a = a ] || fail x\n  true\n", ""},
		{"and-list with an or tail", "  [ -n \"$X\" ] && [ -f \"$X\" ] || skip \"no X\"\n  true\n", ""},
		{"deliberate or-true", "  [ -f a ] || true\n  true\n", ""},
		{"guard ending in an action", "  [ -n \"$p\" ] && ln -s \"$p\" d\n  true\n", ""},
		{"split form", "  [ \"$status\" -eq 1 ]\n  [[ \"$output\" == *x* ]]\n  ! grep -q x f || return 1\n  run ! false\n  true\n", ""},
		{"conditions are not assertions", "  if [ a = a ] && [ b = b ]; then :; fi\n  while ! true; do :; done\n  true\n", ""},
		{"[[ && ]] is one command", "  [[ a == a && b == b ]]\n  true\n", ""},
		{"a run bash -c string is not parsed", "  run bash -c '[ a = b ] && [ c = d ]\n! true'\n  true\n", ""},
		{"a heredoc body is not parsed", "  cat > f <<'EOF'\n[ a = b ] && [ c = d ]\n! true\nEOF\n  true\n", ""},
		{"inside an if body mid-test", "  if true; then\n    [ a = b ] && [ c = d ]\n    ! true\n  fi\n  true\n", and(3) + neg(4)},
		{"last of an if body that ends the test", "  if true; then\n    [ a = b ] && [ c = d ]\n  fi\n", ""},
		{"last of a for body mid-test", "  for i in 1 2; do [ a = b ] && [ c = d ]; done\n  true\n", and(2)},
		{"last of a { } group mid-test", "  { [ a = b ] && [ c = d ]; }\n  true\n", and(2)},
		{"last of a standalone subshell", "  ( cd /; [ a = b ] && [ c = d ] )\n  true\n", ""},
		{"mid subshell", "  ( [ a = b ] && [ c = d ]; true )\n  true\n", and(2)},
		{"a continued and-list", "  [ a = b ] \\\n    && [ c = d ]\n  true\n", and(2)},
		{"a non-final list element is ignored", "  [ -d x ] && { [ a = b ] && [ c = d ]; } || true\n  true\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, out := run(t, "@test \"x\" {\n"+c.body+"}\n")
			wantCode := 0
			if c.want != "" {
				wantCode = 1
			}
			if code != wantCode || out != c.want {
				t.Errorf("exit %d out\n%s\nwant exit %d\n%s", code, out, wantCode, c.want)
			}
		})
	}
}

// TestOnlyTestBodies: a helper function, setup and file-level code are not read; a second @test is.
func TestOnlyTestBodies(t *testing.T) {
	src := "setup() {\n  [ a = b ] && [ c = d ]\n  true\n}\nhelper() { ! true; true; }\n[ a = b ] && [ c = d ]\n" +
		"@test 'one' {\n  true\n}\n@test \"two\" {\n  [ a = b ] && [ c = d ]\n  true\n}\n"
	if code, out := run(t, src); code != 1 || out != and(11) {
		t.Errorf("exit %d out %q, want 1 and %q", code, out, and(11))
	}
}

// TestEmptyRoot: a root with no *.bats is the ERROR row (exit 3); testdata/ and fixtures/ are pruned.
func TestEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"testdata", "fixtures"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "x.bats"), []byte("@test x {\n ! true\n true\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	want := "ERROR  " + ID + "  " + dir + "  no *.bats under the root (wrong path: pass a bats test directory)\n"
	if code != 3 || o.String() != want {
		t.Errorf("exit %d out %q err %q, want 3 and %q", code, o.String(), e.String(), want)
	}
}
